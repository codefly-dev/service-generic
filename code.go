package main

import (
	"context"
	"os"

	corecode "github.com/codefly-dev/core/code"
	codev0 "github.com/codefly-dev/core/generated/go/codefly/services/code/v0"
)

// Code implements the codefly Code gRPC service for the generic agent.
// It embeds DefaultCodeServer directly — NO language-specific overrides.
//
// Provides: ReadFile, WriteFile, CreateFile, DeleteFile, MoveFile, ListFiles,
// Search, GitLog, GitDiff, GitShow, GitBlame, ApplyEdit, GetProjectInfo.
//
// LSP operations return "not available" (DefaultCodeServer stubs).
// Call graph, symbols, dependencies return empty/error (no language knowledge).
type Code struct {
	*corecode.DefaultCodeServer
	*Service
	initialized bool
}

func NewCode(svc *Service) *Code {
	return &Code{
		Service:            svc,
		DefaultCodeServer:  corecode.NewDefaultCodeServer("."),
	}
}

// InitServer creates the DefaultCodeServer once sourceDir is resolved.
// Uses CachedVFS for in-memory file tree caching with fsnotify updates.
func (c *Code) InitServer() {
	c.DefaultCodeServer = corecode.NewDefaultCodeServer(c.sourceDir(), corecode.WithCachedFS())
	c.initialized = true
}

func (c *Code) ensureInit() {
	if !c.initialized {
		c.InitServer()
	}
}

func (c *Code) sourceDir() string {
	if c.sourceLocation != "" {
		return c.sourceLocation
	}
	if wd := os.Getenv("CODEFLY_AGENT_WORKDIR"); wd != "" {
		return wd
	}
	return c.Location
}

func (c *Code) Execute(ctx context.Context, req *codev0.CodeRequest) (*codev0.CodeResponse, error) {
	c.ensureInit()
	return c.DefaultCodeServer.Execute(ctx, req)
}
