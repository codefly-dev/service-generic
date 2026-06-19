package main

import (
	"context"
	"embed"

	"github.com/codefly-dev/core/agents"
	"github.com/codefly-dev/core/agents/services"
	agentv0 "github.com/codefly-dev/core/generated/go/codefly/services/agent/v0"
	configurations "github.com/codefly-dev/core/resources"
	"github.com/codefly-dev/core/shared"
	"github.com/codefly-dev/core/templates"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var agent = shared.Must(configurations.LoadFromFs[configurations.Agent](shared.Embed(infoFS)))

// Service is the generic codefly agent. Language-agnostic: provides baseline
// filesystem, git, and search operations for ANY repository.
type Service struct {
	*services.Base
	sourceLocation string
}

func (s *Service) GetAgentInformation(ctx context.Context, _ *agentv0.AgentInformationRequest) (*agentv0.AgentInformation, error) {
	defer s.Wool.Catch()

	readme, err := templates.ApplyTemplateFrom(ctx, shared.Embed(readmeFS), "templates/agent/README.md", s.Information)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &agentv0.AgentInformation{
		RuntimeRequirements: []*agentv0.Runtime{},
		Capabilities: []*agentv0.Capability{
			{Type: agentv0.Capability_RUNTIME},
		},
		Languages:  []*agentv0.Language{},
		Protocols:  []*agentv0.Protocol{},
		ReadMe:     readme,
	}, nil
}

func NewService() *Service {
	return &Service{
		Base: services.NewServiceBase(context.Background(), agent),
	}
}

func main() {
	svc := NewService()
	code := NewCode(svc)
	runtime := NewRuntime(svc)
	agents.Serve(agents.PluginRegistration{
		Agent:   svc,
		Runtime: runtime,
		Code:    code,
		Tooling: NewTooling(code, runtime),
	})
}

//go:embed agent.codefly.yaml
var infoFS embed.FS

//go:embed templates/agent
var readmeFS embed.FS
