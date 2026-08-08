package main

import (
	"context"

	"github.com/codefly-dev/core/agents/services"
	"github.com/codefly-dev/core/failures"
	basev0 "github.com/codefly-dev/core/generated/go/codefly/base/v0"
	runtimev0 "github.com/codefly-dev/core/generated/go/codefly/services/runtime/v0"
)

// Runtime implements the codefly Runtime gRPC service for the generic agent.
// Minimal: Load resolves the source directory. No build/test/lint (no language).
type Runtime struct {
	services.RuntimeServer
	*Service
}

func NewRuntime(svc *Service) *Runtime {
	return &Runtime{Service: svc}
}

func (s *Runtime) Load(ctx context.Context, req *runtimev0.LoadRequest) (*runtimev0.LoadResponse, error) {
	defer s.Wool.Catch()
	response, err := s.Runtime.LoadService(ctx, req, services.RuntimeLoad{Settings: s.Settings})
	if err == nil && response.GetStatus().GetState() == runtimev0.LoadStatus_READY {
		s.sourceLocation = s.ResolveSourceLocation()
	}
	return response, err
}

func (s *Runtime) Init(ctx context.Context, req *runtimev0.InitRequest) (*runtimev0.InitResponse, error) {
	defer s.Wool.Catch()

	s.Runtime.LogInitRequest(req)
	return s.Runtime.InitResponse()
}

func (s *Runtime) Start(ctx context.Context, req *runtimev0.StartRequest) (*runtimev0.StartResponse, error) {
	defer s.Wool.Catch()

	// Generic agent has no process to start — it's a passive service.
	return s.Runtime.StartResponse()
}

func (s *Runtime) Stop(ctx context.Context, req *runtimev0.StopRequest) (*runtimev0.StopResponse, error) {
	defer s.Wool.Catch()
	return s.Runtime.StopResponse()
}

func (s *Runtime) Destroy(ctx context.Context, req *runtimev0.DestroyRequest) (*runtimev0.DestroyResponse, error) {
	defer s.Wool.Catch()
	return s.Runtime.DestroyResponse()
}

func (s *Runtime) Information(ctx context.Context, req *runtimev0.InformationRequest) (*runtimev0.InformationResponse, error) {
	return s.Runtime.InformationResponse(ctx, req)
}

// Build reports the generic agent's intentional lack of a language build
// capability through the typed Runtime contract. Returning gRPC Unimplemented
// would misclassify an expected capability boundary as transport failure.
func (s *Runtime) Build(context.Context, *runtimev0.BuildRequest) (*runtimev0.BuildResponse, error) {
	message := "build not available: generic agent has no language knowledge"
	return &runtimev0.BuildResponse{
		Status: &runtimev0.BuildStatus{
			State:   runtimev0.BuildStatus_ERROR,
			Message: message,
			Failure: unsupportedRuntimeFailure("runtime.build", message),
		},
		Output: message,
	}, nil
}

// Test reports an unsupported capability instead of inventing a test command
// for an unknown language or framework.
func (s *Runtime) Test(context.Context, *runtimev0.TestRequest) (*runtimev0.TestResponse, error) {
	message := "test not available: generic agent has no language knowledge"
	failure := unsupportedRuntimeFailure("runtime.test", message)
	return &runtimev0.TestResponse{Status: &runtimev0.TestStatus{
		State:   runtimev0.TestStatus_ERROR,
		Message: message,
		Failure: failure,
	}, Result: &runtimev0.TestRunResult{
		State:   runtimev0.TestRunResult_ERRORED,
		Message: message,
		Failure: failure,
	}}, nil
}

// Lint reports an unsupported capability instead of inventing a linter for an
// unknown language or framework.
func (s *Runtime) Lint(context.Context, *runtimev0.LintRequest) (*runtimev0.LintResponse, error) {
	message := "lint not available: generic agent has no language knowledge"
	return &runtimev0.LintResponse{
		Status: &runtimev0.LintStatus{
			State:   runtimev0.LintStatus_ERROR,
			Message: message,
			Failure: unsupportedRuntimeFailure("runtime.lint", message),
		},
		Output: message,
	}, nil
}

func unsupportedRuntimeFailure(operation, message string) *basev0.Failure {
	return failures.New(basev0.FailureCode_FAILURE_CODE_UNSUPPORTED_OPERATION, operation, message)
}
