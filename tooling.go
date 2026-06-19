package main

import (
	"context"
	"fmt"

	codev0 "github.com/codefly-dev/core/generated/go/codefly/services/code/v0"
	runtimev0 "github.com/codefly-dev/core/generated/go/codefly/services/runtime/v0"
	toolingv0 "github.com/codefly-dev/core/generated/go/codefly/services/tooling/v0"
)

// Tooling implements the Tooling gRPC service for the generic agent.
// Delegates file/search/edit to Code. LSP and call graph return empty
// (no language knowledge). Build/test/lint return "not available".
type Tooling struct {
	toolingv0.UnimplementedToolingServer
	code    *Code
	runtime *Runtime
}

func NewTooling(code *Code, runtime *Runtime) *Tooling {
	return &Tooling{code: code, runtime: runtime}
}

// ── File/Search operations (delegate to Code) ──────────

func (t *Tooling) GetProjectInfo(ctx context.Context, _ *toolingv0.GetProjectInfoRequest) (*toolingv0.GetProjectInfoResponse, error) {
	resp, err := t.code.Execute(ctx, &codev0.CodeRequest{
		Operation: &codev0.CodeRequest_GetProjectInfo{GetProjectInfo: &codev0.GetProjectInfoRequest{}},
	})
	if err != nil {
		return nil, fmt.Errorf("tooling get_project_info: %w", err)
	}
	pi := resp.GetGetProjectInfo()
	if pi == nil {
		return &toolingv0.GetProjectInfoResponse{}, nil
	}
	return &toolingv0.GetProjectInfoResponse{
		Language: "generic", FileHashes: pi.FileHashes, Error: pi.Error,
	}, nil
}

func (t *Tooling) Fix(ctx context.Context, req *toolingv0.FixRequest) (*toolingv0.FixResponse, error) {
	resp, err := t.code.Execute(ctx, &codev0.CodeRequest{
		Operation: &codev0.CodeRequest_Fix{Fix: &codev0.FixRequest{File: req.File}},
	})
	if err != nil {
		return nil, fmt.Errorf("tooling fix: %w", err)
	}
	fix := resp.GetFix()
	if fix == nil {
		return &toolingv0.FixResponse{Success: false, Error: "no response"}, nil
	}
	return &toolingv0.FixResponse{
		Success: fix.Success, Content: fix.Content,
		Error: fix.Error, Actions: fix.Actions,
	}, nil
}

func (t *Tooling) ApplyEdit(ctx context.Context, req *toolingv0.ApplyEditRequest) (*toolingv0.ApplyEditResponse, error) {
	resp, err := t.code.Execute(ctx, &codev0.CodeRequest{
		Operation: &codev0.CodeRequest_ApplyEdit{ApplyEdit: &codev0.ApplyEditRequest{
			File: req.File, Find: req.Find, Replace: req.Replace, AutoFix: req.AutoFix,
		}},
	})
	if err != nil {
		return nil, fmt.Errorf("tooling apply_edit: %w", err)
	}
	ae := resp.GetApplyEdit()
	if ae == nil {
		return &toolingv0.ApplyEditResponse{Success: false, Error: "no response"}, nil
	}
	return &toolingv0.ApplyEditResponse{
		Success: ae.Success, Content: ae.Content,
		Error: ae.Error, Strategy: ae.Strategy, FixActions: ae.FixActions,
	}, nil
}

// ── LSP operations (not available — no language) ───────

func (t *Tooling) ListSymbols(ctx context.Context, req *toolingv0.ListSymbolsRequest) (*toolingv0.ListSymbolsResponse, error) {
	return &toolingv0.ListSymbolsResponse{}, nil
}

func (t *Tooling) GetDiagnostics(ctx context.Context, req *toolingv0.GetDiagnosticsRequest) (*toolingv0.GetDiagnosticsResponse, error) {
	return &toolingv0.GetDiagnosticsResponse{}, nil
}

func (t *Tooling) GoToDefinition(ctx context.Context, req *toolingv0.GoToDefinitionRequest) (*toolingv0.GoToDefinitionResponse, error) {
	return &toolingv0.GoToDefinitionResponse{}, nil
}

func (t *Tooling) FindReferences(ctx context.Context, req *toolingv0.FindReferencesRequest) (*toolingv0.FindReferencesResponse, error) {
	return &toolingv0.FindReferencesResponse{}, nil
}

func (t *Tooling) RenameSymbol(ctx context.Context, req *toolingv0.RenameSymbolRequest) (*toolingv0.RenameSymbolResponse, error) {
	return &toolingv0.RenameSymbolResponse{Success: false, Error: "rename not available: generic agent has no language knowledge"}, nil
}

func (t *Tooling) GetHoverInfo(ctx context.Context, req *toolingv0.GetHoverInfoRequest) (*toolingv0.GetHoverInfoResponse, error) {
	return &toolingv0.GetHoverInfoResponse{}, nil
}

func (t *Tooling) GetCompletions(ctx context.Context, req *toolingv0.GetCompletionsRequest) (*toolingv0.GetCompletionsResponse, error) {
	return &toolingv0.GetCompletionsResponse{}, nil
}

func (t *Tooling) GetCallGraph(ctx context.Context, req *toolingv0.GetCallGraphRequest) (*toolingv0.GetCallGraphResponse, error) {
	return &toolingv0.GetCallGraphResponse{Error: "call graph not available: generic agent has no language knowledge"}, nil
}

// ── Dependencies (not available — no language) ─────────

func (t *Tooling) ListDependencies(ctx context.Context, _ *toolingv0.ListDependenciesRequest) (*toolingv0.ListDependenciesResponse, error) {
	return &toolingv0.ListDependenciesResponse{Error: "dependency listing not available: generic agent"}, nil
}

func (t *Tooling) AddDependency(ctx context.Context, req *toolingv0.AddDependencyRequest) (*toolingv0.AddDependencyResponse, error) {
	return &toolingv0.AddDependencyResponse{Success: false, Error: "add dependency not available: generic agent"}, nil
}

func (t *Tooling) RemoveDependency(ctx context.Context, req *toolingv0.RemoveDependencyRequest) (*toolingv0.RemoveDependencyResponse, error) {
	return &toolingv0.RemoveDependencyResponse{Success: false, Error: "remove dependency not available: generic agent"}, nil
}

// ── Dev Validation (not available — no language) ───────

func (t *Tooling) Build(ctx context.Context, _ *toolingv0.BuildRequest) (*toolingv0.BuildResponse, error) {
	return &toolingv0.BuildResponse{Success: false, Output: "build not available: generic agent has no language knowledge"}, nil
}

func (t *Tooling) Test(ctx context.Context, _ *toolingv0.TestRequest) (*toolingv0.TestResponse, error) {
	return &toolingv0.TestResponse{Success: false, Output: "test not available: generic agent has no language knowledge"}, nil
}

func (t *Tooling) Lint(ctx context.Context, _ *toolingv0.LintRequest) (*toolingv0.LintResponse, error) {
	return &toolingv0.LintResponse{Success: false, Output: "lint not available: generic agent has no language knowledge"}, nil
}

// ── Unused parameter suppression ───────────────────────
var (
	_ = (*runtimev0.RuntimeServer)(nil)
)
