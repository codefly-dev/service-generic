package main

import (
	"context"
	"os"

	"github.com/codefly-dev/core/agents/services"
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
	err := s.Base.Load(ctx, req.Identity, nil)
	if err != nil {
		return s.Runtime.LoadErrorf(err, "loading base")
	}

	defer s.Wool.Catch()

	if req.DisableCatch {
		s.Wool.DisableCatch()
	}

	s.Runtime.SetEnvironment(req.Environment)

	// Resolve source location: prefer CODEFLY_AGENT_WORKDIR, fall back to service location.
	if wd := os.Getenv("CODEFLY_AGENT_WORKDIR"); wd != "" {
		s.sourceLocation = wd
	} else {
		s.sourceLocation = s.Location
	}

	return s.Runtime.LoadResponse()
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
