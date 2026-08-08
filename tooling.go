package main

import (
	"context"
	"fmt"

	"github.com/codefly-dev/core/failures"
	basev0 "github.com/codefly-dev/core/generated/go/codefly/base/v0"
	codev0 "github.com/codefly-dev/core/generated/go/codefly/services/code/v0"
	toolingv0 "github.com/codefly-dev/core/generated/go/codefly/services/tooling/v0"
)

// Tooling implements the Tooling gRPC service for the generic agent.
// Delegates project metadata and edits to Code. Build/test/lint return
// "not available" because the generic agent has no language toolchain.
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
		return &toolingv0.GetProjectInfoResponse{Failure: failures.Ensure(resp.GetFailure(), basev0.FailureCode_FAILURE_CODE_INTERNAL, "tooling.get-project-info", "code service returned no project-info result")}, nil
	}
	packages := make([]*toolingv0.PackageInfo, 0, len(pi.GetPackages()))
	for _, pkg := range pi.GetPackages() {
		packages = append(packages, &toolingv0.PackageInfo{
			Name: pkg.GetName(), RelativePath: pkg.GetRelativePath(), Files: append([]string(nil), pkg.GetFiles()...),
			Imports: append([]string(nil), pkg.GetImports()...), Doc: pkg.GetDoc(),
		})
	}
	dependencies := make([]*toolingv0.Dependency, 0, len(pi.GetDependencies()))
	for _, dependency := range pi.GetDependencies() {
		dependencies = append(dependencies, &toolingv0.Dependency{
			Name: dependency.GetName(), Version: dependency.GetVersion(), Direct: dependency.GetDirect(),
		})
	}
	sourceFiles := make([]*toolingv0.SourceFileInfo, 0, len(pi.GetSourceFiles()))
	for _, file := range pi.GetSourceFiles() {
		sourceFiles = append(sourceFiles, &toolingv0.SourceFileInfo{Path: file.GetPath(), Imports: append([]string(nil), file.GetImports()...)})
	}
	return &toolingv0.GetProjectInfoResponse{
		Module: pi.GetModule(), Language: pi.GetLanguage(), LanguageVersion: pi.GetLanguageVersion(),
		Packages: packages, Dependencies: dependencies, FileHashes: pi.GetFileHashes(), SourceFiles: sourceFiles,
		Failure: failures.Clone(resp.GetFailure()),
	}, nil
}

func (t *Tooling) Fix(ctx context.Context, req *toolingv0.FixRequest) (*toolingv0.FixResponse, error) {
	resp, err := t.code.Execute(ctx, &codev0.CodeRequest{
		Operation: &codev0.CodeRequest_Fix{Fix: &codev0.FixRequest{File: req.GetFile(), Mode: req.GetMode(), DryRun: req.GetDryRun()}},
	})
	if err != nil {
		return nil, fmt.Errorf("tooling fix: %w", err)
	}
	fix := resp.GetFix()
	if fix == nil {
		return &toolingv0.FixResponse{Success: false, Failure: failures.Ensure(resp.GetFailure(), basev0.FailureCode_FAILURE_CODE_INTERNAL, "tooling.fix", "code service returned no fix result")}, nil
	}
	return &toolingv0.FixResponse{
		Success: fix.Success, Content: fix.Content,
		Actions: fix.Actions, Failure: failures.Clone(resp.GetFailure()),
		Changed: fix.Changed, BeforeSha256: fix.BeforeSha256, AfterSha256: fix.AfterSha256,
		Wrote: fix.Wrote, Output: fix.Output,
	}, nil
}

func (t *Tooling) ApplyEdit(ctx context.Context, req *toolingv0.ApplyEditRequest) (*toolingv0.ApplyEditResponse, error) {
	resp, err := t.code.Execute(ctx, &codev0.CodeRequest{
		Operation: &codev0.CodeRequest_ApplyEdit{ApplyEdit: &codev0.ApplyEditRequest{
			File: req.GetFile(), Find: req.GetFind(), Replace: req.GetReplace(), FixMode: req.GetFixMode(), DryRun: req.GetDryRun(),
		}},
	})
	if err != nil {
		return nil, fmt.Errorf("tooling apply_edit: %w", err)
	}
	ae := resp.GetApplyEdit()
	if ae == nil {
		return &toolingv0.ApplyEditResponse{Success: false, Failure: failures.Ensure(resp.GetFailure(), basev0.FailureCode_FAILURE_CODE_INTERNAL, "tooling.apply-edit", "code service returned no apply-edit result")}, nil
	}
	return &toolingv0.ApplyEditResponse{
		Success: ae.Success, Content: ae.Content,
		Strategy: ae.Strategy, FixActions: ae.FixActions, Failure: failures.Clone(resp.GetFailure()),
		Changed: ae.Changed, BeforeSha256: ae.BeforeSha256, AfterSha256: ae.AfterSha256,
		Wrote: ae.Wrote, Output: ae.Output,
	}, nil
}

// ── Dependencies (read-only declarative fallback) ─────────

func (t *Tooling) ListDependencies(ctx context.Context, _ *toolingv0.ListDependenciesRequest) (*toolingv0.ListDependenciesResponse, error) {
	resp, err := t.code.Execute(ctx, &codev0.CodeRequest{
		Operation: &codev0.CodeRequest_ListDependencies{ListDependencies: &codev0.ListDependenciesRequest{}},
	})
	if err != nil {
		return nil, fmt.Errorf("tooling list_dependencies: %w", err)
	}
	listed := resp.GetListDependencies()
	if listed == nil {
		return &toolingv0.ListDependenciesResponse{Failure: failures.Ensure(resp.GetFailure(), basev0.FailureCode_FAILURE_CODE_INTERNAL, "tooling.list-dependencies", "code service returned no dependency result")}, nil
	}
	dependencies := make([]*toolingv0.Dependency, 0, len(listed.GetDependencies()))
	for _, dependency := range listed.GetDependencies() {
		dependencies = append(dependencies, &toolingv0.Dependency{
			Name: dependency.GetName(), Version: dependency.GetVersion(), Direct: dependency.GetDirect(),
		})
	}
	return &toolingv0.ListDependenciesResponse{Dependencies: dependencies, Failure: failures.Clone(resp.GetFailure())}, nil
}

func (t *Tooling) AddDependency(ctx context.Context, req *toolingv0.AddDependencyRequest) (*toolingv0.AddDependencyResponse, error) {
	return &toolingv0.AddDependencyResponse{Success: false, Failure: unsupportedToolingFailure("tooling.add-dependency", "add dependency not available: generic agent")}, nil
}

func (t *Tooling) RemoveDependency(ctx context.Context, req *toolingv0.RemoveDependencyRequest) (*toolingv0.RemoveDependencyResponse, error) {
	return &toolingv0.RemoveDependencyResponse{Success: false, Failure: unsupportedToolingFailure("tooling.remove-dependency", "remove dependency not available: generic agent")}, nil
}

// ── Dev Validation (not available — no language) ───────

func (t *Tooling) Build(ctx context.Context, _ *toolingv0.BuildRequest) (*toolingv0.BuildResponse, error) {
	message := "build not available: generic agent has no language knowledge"
	return &toolingv0.BuildResponse{Success: false, Output: message, Failure: unsupportedToolingFailure("tooling.build", message)}, nil
}

func (t *Tooling) Test(ctx context.Context, _ *toolingv0.TestRequest) (*toolingv0.TestResponse, error) {
	message := "test not available: generic agent has no language knowledge"
	return &toolingv0.TestResponse{Success: false, Output: message, Failure: unsupportedToolingFailure("tooling.test", message)}, nil
}

func (t *Tooling) Lint(ctx context.Context, _ *toolingv0.LintRequest) (*toolingv0.LintResponse, error) {
	message := "lint not available: generic agent has no language knowledge"
	return &toolingv0.LintResponse{Success: false, Output: message, Failure: unsupportedToolingFailure("tooling.lint", message)}, nil
}

func unsupportedToolingFailure(operation, message string) *basev0.Failure {
	return failures.New(basev0.FailureCode_FAILURE_CODE_UNSUPPORTED_OPERATION, operation, message)
}
