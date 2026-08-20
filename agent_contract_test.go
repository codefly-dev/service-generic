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
	if info.GetFailure().GetCode() != basev0.FailureCode_FAILURE_CODE_UNSUPPORTED_OPERATION || info.GetLanguage() != "unknown" || info.GetFileHashes()["README.txt"] == "" {
		t.Fatalf("project info = %+v, want typed unsupported semantics plus preserved README hash", info)
	}
}

// The generic agent ships without the tree-sitter CGO stack, so source-semantics
// inspection (dependency, import, and symbol extraction) is reported as a typed
// unsupported operation. Language classification and file hashes stay available
// because they are declarative and need no analyzer.
func TestGenericAgentReportsSourceSemanticsUnsupportedWithoutAnalyzer(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		manifest string
		language string
	}{
		{
			name: "jvm",
			files: map[string]string{
				"build.gradle": `def grpcVersion = "1.82.1"
dependencies { implementation "io.grpc:grpc-stub:${grpcVersion}" }
`,
				"src/main/java/example/App.java": "package example;\nimport io.grpc.Server;\nclass App {}\n",
			},
			manifest: "build.gradle", language: "jvm",
		},
		{
			name: "dotnet",
			files: map[string]string{
				"cart.sln":                "Microsoft Visual Studio Solution File, Format Version 12.00\n",
				"src/cart.csproj":         `<Project Sdk="Microsoft.NET.Sdk.Web"><PropertyGroup><TargetFramework>net10.0</TargetFramework></PropertyGroup><ItemGroup><PackageReference Include="Grpc.AspNetCore" Version="2.80.0" /></ItemGroup></Project>`,
				"src/services/CartSvc.cs": "using Grpc.Core;\nclass CartSvc {}\n",
			},
			manifest: "cart.sln", language: "dotnet",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			for relative, content := range test.files {
				filename := filepath.Join(root, filepath.FromSlash(relative))
				if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filename, []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			service := NewService()
			service.sourceLocation = root
			tooling := NewTooling(NewCode(service), NewRuntime(service))

			info, err := tooling.GetProjectInfo(t.Context(), &toolingv0.GetProjectInfoRequest{})
			if err != nil {
				t.Fatal(err)
			}
			if info.GetFailure().GetCode() != basev0.FailureCode_FAILURE_CODE_UNSUPPORTED_OPERATION {
				t.Fatalf("project info failure = %+v, want typed unsupported operation", info.GetFailure())
			}
			if info.GetLanguage() != test.language {
				t.Fatalf("project info language = %q, want %q", info.GetLanguage(), test.language)
			}
			if info.GetFileHashes()[test.manifest] == "" {
				t.Fatalf("project info file hashes = %+v, want preserved hash for %q", info.GetFileHashes(), test.manifest)
			}
			if len(info.GetDependencies()) != 0 || len(info.GetSourceFiles()) != 0 {
				t.Fatalf("project info = %+v, want no analyzer-derived dependencies or imports", info)
			}

			semantic, err := tooling.GetSemanticIndex(t.Context(), &toolingv0.GetSemanticIndexRequest{})
			if err != nil {
				t.Fatal(err)
			}
			if semantic.GetFailure().GetCode() != basev0.FailureCode_FAILURE_CODE_UNSUPPORTED_OPERATION {
				t.Fatalf("semantic index failure = %+v, want typed unsupported operation", semantic.GetFailure())
			}
			if semantic.GetIndex().GetState() != basev0.SemanticIndexState_SEMANTIC_INDEX_STATE_NOT_ATTEMPTED {
				t.Fatalf("semantic index state = %v, want NOT_ATTEMPTED", semantic.GetIndex().GetState())
			}
		})
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

func TestGenericRuntimeTestReportsCompleteUnsupportedResult(t *testing.T) {
	runtime := NewRuntime(NewService())
	response, err := runtime.Test(t.Context(), &runtimev0.TestRequest{})
	if err != nil {
		t.Fatalf("test transport error: %v", err)
	}
	status := response.GetStatus()
	result := response.GetResult()
	if status.GetState() != runtimev0.TestStatus_ERROR {
		t.Fatalf("status state = %v, want ERROR", status.GetState())
	}
	if result.GetState() != runtimev0.TestRunResult_ERRORED {
		t.Fatalf("result state = %v, want ERRORED", result.GetState())
	}
	if result.GetMessage() == "" || result.GetMessage() != status.GetMessage() {
		t.Fatalf("result message = %q, want status message %q", result.GetMessage(), status.GetMessage())
	}
	if result.GetFailure().GetCode() != basev0.FailureCode_FAILURE_CODE_UNSUPPORTED_OPERATION {
		t.Fatalf("result failure = %+v, want typed unsupported operation", result.GetFailure())
	}
	if result.GetFailure().GetOperation() != status.GetFailure().GetOperation() {
		t.Fatalf("result failure operation = %q, want status operation %q", result.GetFailure().GetOperation(), status.GetFailure().GetOperation())
	}
}

func TestGenericRuntimeLoadsAttachedSourceThroughRealServiceLifecycle(t *testing.T) {
	workspace := t.TempDir()
	serviceDir := filepath.Join(workspace, "services", "source")
	sourceDir := t.TempDir()
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
	if err := os.Symlink(sourceDir, filepath.Join(serviceDir, "code")); err != nil {
		t.Fatalf("link attached source: %v", err)
	}
	environment, err := resources.LocalEnvironment().Proto()
	if err != nil {
		t.Fatalf("local environment: %v", err)
	}
	t.Setenv("CODEFLY_AGENT_WORKDIR", serviceDir)

	svc := NewService()
	code := NewCode(svc)
	// Read-only Code runs before Runtime.Load in the production lazy-runtime
	// path. It must resolve the real service declaration and attached source on
	// that first request, without pinning the generated service directory.
	read, err := code.Execute(t.Context(), &codev0.CodeRequest{
		Operation: &codev0.CodeRequest_ReadFile{ReadFile: &codev0.ReadFileRequest{Path: "README.txt"}},
	})
	if err != nil {
		t.Fatalf("pre-load attached source read: %v", err)
	} else if got := read.GetReadFile(); got == nil || got.GetContent() != content {
		t.Fatalf("pre-load read response = %+v, want attached source content", read)
	}
	physicalSource, err := filepath.EvalSymlinks(sourceDir)
	if err != nil {
		t.Fatalf("resolve physical source: %v", err)
	}
	if svc.sourceLocation != physicalSource {
		t.Fatalf("pre-load source location = %q, want %q", svc.sourceLocation, physicalSource)
	}
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
	if svc.sourceLocation != physicalSource {
		t.Fatalf("source location = %q, want %q", svc.sourceLocation, physicalSource)
	}
	read, err = code.Execute(t.Context(), &codev0.CodeRequest{
		Operation: &codev0.CodeRequest_ReadFile{ReadFile: &codev0.ReadFileRequest{Path: "README.txt"}},
	})
	if err != nil {
		t.Fatalf("read attached source: %v", err)
	}
	if got := read.GetReadFile(); got == nil || got.GetContent() != content {
		t.Fatalf("read response = %+v, want attached source content", read)
	}
}
