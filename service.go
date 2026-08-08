package main

import (
	"context"
	"embed"
	"os"
	"path/filepath"
	"strings"

	"github.com/codefly-dev/core/agents"
	"github.com/codefly-dev/core/agents/services"
	agentv0 "github.com/codefly-dev/core/generated/go/codefly/services/agent/v0"
	configurations "github.com/codefly-dev/core/resources"
	"github.com/codefly-dev/core/shared"
	"github.com/codefly-dev/core/templates"
	"github.com/codefly-dev/core/toolbox/lang"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var agent = shared.Must(configurations.LoadFromFs[configurations.Agent](shared.Embed(infoFS)))

// Settings contains only the language-neutral source attachment contract.
// The generic agent deliberately carries no build, test, or lint settings.
type Settings struct {
	SourceDir string `yaml:"source-dir"`
}

// Service is the generic codefly agent. Language-agnostic: provides baseline
// filesystem, git, and search operations for ANY repository.
type Service struct {
	*services.Base
	Settings       *Settings
	sourceLocation string
}

func (s *Service) GetAgentInformation(ctx context.Context, _ *agentv0.AgentInformationRequest) (*agentv0.AgentInformation, error) {
	defer s.Wool.Catch()

	readme, err := templates.ApplyTemplateFrom(ctx, shared.Embed(readmeFS), "templates/agent/README.md", s.Information)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return services.Advertisement{
		RuntimeOnly: true,
		ReadMe:      readme,
	}.Build(), nil
}

func NewService() *Service {
	return &Service{
		Base:     services.NewServiceBase(context.Background(), agent),
		Settings: &Settings{},
	}
}

// ResolveSourceLocation binds generic Code and Tooling operations to the
// source directory declared by the service adapter. CODEFLY_AGENT_WORKDIR is
// the generated service root for attached arbitrary-source sessions; the
// language-neutral source-dir setting selects its project attachment.
func (s *Service) ResolveSourceLocation() string {
	root := s.Location
	if workDir := os.Getenv("CODEFLY_AGENT_WORKDIR"); workDir != "" {
		root = workDir
	}
	location := root
	if s.Settings == nil || strings.TrimSpace(s.Settings.SourceDir) == "" {
		return resolveAttachedSource(root)
	}
	sourceDir := filepath.FromSlash(strings.TrimSpace(s.Settings.SourceDir))
	if filepath.IsAbs(sourceDir) {
		location = filepath.Clean(sourceDir)
	} else {
		location = filepath.Join(root, sourceDir)
	}
	return resolveAttachedSource(location)
}

// resolveAttachedSource follows the ephemeral source-workspace symlink before
// CachedVFS inventories it. filepath.WalkDir intentionally does not descend
// through a symlink passed as its root, so retaining the generated link would
// make a valid attachment appear empty after a successful Runtime Load.
func resolveAttachedSource(location string) string {
	location = filepath.Clean(location)
	if physical, err := filepath.EvalSymlinks(location); err == nil {
		return physical
	}
	return location
}

// pluginRegistration wires the generic agent's service surface. It advertises
// no Builder: the generic agent is a passive, language-agnostic toolbox that
// emits no Kubernetes manifests, so the manifest-bundle capability is reported
// as unsupported rather than carrying any manifest or transport responsibility.
func pluginRegistration() agents.PluginRegistration {
	svc := NewService()
	code := NewCode(svc)
	runtime := NewRuntime(svc)
	tooling := NewTooling(code, runtime)
	return agents.PluginRegistration{
		Agent:   svc,
		Runtime: runtime,
		Code:    code,
		Tooling: tooling,
		Toolbox: lang.NewEditToolboxFromTooling(agent.Name, agent.Version, tooling),
	}
}

func main() {
	agents.Serve(pluginRegistration())
}

//go:embed agent.codefly.yaml
var infoFS embed.FS

//go:embed templates/agent
var readmeFS embed.FS
