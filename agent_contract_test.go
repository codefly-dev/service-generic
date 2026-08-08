package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	basev0 "github.com/codefly-dev/core/generated/go/codefly/base/v0"
	agentv0 "github.com/codefly-dev/core/generated/go/codefly/services/agent/v0"
	codev0 "github.com/codefly-dev/core/generated/go/codefly/services/code/v0"
	runtimev0 "github.com/codefly-dev/core/generated/go/codefly/services/runtime/v0"
	toolingv0 "github.com/codefly-dev/core/generated/go/codefly/services/tooling/v0"
	"github.com/codefly-dev/core/resources"
)

func TestGenericAgentInformationLoadsEmbeddedCapabilityGuide(t *testing.T) {
	info, err := NewService().GetAgentInformation(t.Context(), &agentv0.AgentInformationRequest{})
	if err != nil {
		t.Fatalf("get agent information: %v", err)
	}
	if !strings.Contains(info.GetReadMe(), "Language-agnostic codefly agent") || !strings.Contains(info.GetReadMe(), "No language-specific build/test/lint") {
		t.Fatalf("agent README does not describe the generic capability boundary: %q", info.GetReadMe())
	}
}

func TestGenericAgentProvidesRealBaselineCodeAndProjectInfo(t *testing.T) {
	root := t.TempDir()
	const content = "generic source evidence\n"
	if err := os.WriteFile(filepath.Join(root, "README.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	service := NewService()
	service.sourceLocation = root
	code := NewCode(service)
	read, err := code.Execute(t.Context(), &codev0.CodeRequest{
		Operation: &codev0.CodeRequest_ReadFile{ReadFile: &codev0.ReadFileRequest{Path: "README.txt"}},
	})
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if got := read.GetReadFile(); got == nil || got.GetContent() != content {
		t.Fatalf("read response = %+v, want exact real-file content", read)
	}

	tooling := NewTooling(code, NewRuntime(service))
	info, err := tooling.GetProjectInfo(t.Context(), &toolingv0.GetProjectInfoRequest{})
	if err != nil {
		t.Fatalf("get project info: %v", err)
	}
	if info.GetFailure() != nil || info.GetLanguage() != "generic" || info.GetFileHashes()["README.txt"] == "" {
		t.Fatalf("project info = %+v, want generic language and README hash", info)
	}
}

func TestGenericRuntimeReportsUnsupportedDevCapabilities(t *testing.T) {
	runtime := NewRuntime(NewService())
	tests := []struct {
		name string
		call func(context.Context) *basev0.Failure
	}{
		{name: "build", call: func(ctx context.Context) *basev0.Failure {
			response, err := runtime.Build(ctx, &runtimev0.BuildRequest{})
			if err != nil {
				t.Fatalf("build transport error: %v", err)
			}
			return response.GetStatus().GetFailure()
		}},
		{name: "test", call: func(ctx context.Context) *basev0.Failure {
			response, err := runtime.Test(ctx, &runtimev0.TestRequest{})
			if err != nil {
				t.Fatalf("test transport error: %v", err)
			}
			return response.GetStatus().GetFailure()
		}},
		{name: "lint", call: func(ctx context.Context) *basev0.Failure {
			response, err := runtime.Lint(ctx, &runtimev0.LintRequest{})
			if err != nil {
				t.Fatalf("lint transport error: %v", err)
			}
			return response.GetStatus().GetFailure()
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			failure := test.call(t.Context())
			if failure == nil || failure.GetCode() != basev0.FailureCode_FAILURE_CODE_UNSUPPORTED_OPERATION {
				t.Fatalf("failure = %+v, want typed unsupported operation", failure)
			}
		})
	}
}

func TestGenericRuntimeLoadsAttachedSourceThroughRealServiceLifecycle(t *testing.T) {
	workspace := t.TempDir()
	serviceDir := filepath.Join(workspace, "services", "source")
	sourceDir := filepath.Join(serviceDir, "code")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	const content = "attached generic source\n"
	if err := os.WriteFile(filepath.Join(sourceDir, "README.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	service := &resources.Service{
		Name: "source", Version: "0.0.0", Agent: agent,
		Spec: map[string]any{"source-dir": "code"},
	}
	service.WithDir(serviceDir)
	if err := service.Save(t.Context()); err != nil {
		t.Fatalf("save real service declaration: %v", err)
	}
	environment, err := resources.LocalEnvironment().Proto()
	if err != nil {
		t.Fatalf("local environment: %v", err)
	}
	t.Setenv("CODEFLY_AGENT_WORKDIR", serviceDir)

	svc := NewService()
	runtime := NewRuntime(svc)
	response, err := runtime.Load(t.Context(), &runtimev0.LoadRequest{
		Identity: &basev0.ServiceIdentity{
			Name: "source", Module: "source-workspace", Workspace: "source-workspace", WorkspacePath: workspace,
			RelativeToWorkspace: "services/source",
		},
		Environment: environment,
	})
	if err != nil {
		t.Fatalf("runtime load transport error: %v", err)
	}
	if response.GetStatus().GetState() != runtimev0.LoadStatus_READY {
		t.Fatalf("runtime load status = %+v, want READY", response.GetStatus())
	}
	if svc.sourceLocation != sourceDir {
		t.Fatalf("source location = %q, want %q", svc.sourceLocation, sourceDir)
	}
	read, err := NewCode(svc).Execute(t.Context(), &codev0.CodeRequest{
		Operation: &codev0.CodeRequest_ReadFile{ReadFile: &codev0.ReadFileRequest{Path: "README.txt"}},
	})
	if err != nil {
		t.Fatalf("read attached source: %v", err)
	}
	if got := read.GetReadFile(); got == nil || got.GetContent() != content {
		t.Fatalf("read response = %+v, want attached source content", read)
	}
}
