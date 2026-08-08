package main

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	corecode "github.com/codefly-dev/core/code"
	codev0 "github.com/codefly-dev/core/generated/go/codefly/services/code/v0"
	"github.com/codefly-dev/core/wool"
)

// Code implements the codefly Code gRPC service for the generic agent.
// It embeds DefaultCodeServer directly — NO runtime or mutation overrides.
// Core's declarative project-info fallback may inspect JVM/.NET manifests and
// source imports, but it never executes a native build tool.
//
// Provides: ReadFile, WriteFile, CreateFile, DeleteFile, MoveFile, ListFiles,
// Search, GitLog, GitDiff, GitShow, GitBlame, ApplyEdit, GetProjectInfo,
// and read-only declarative dependency inspection.
//
// LSP operations return "not available" (DefaultCodeServer stubs).
// Call graph and symbols return typed unsupported results. Runtime build/test/
// lint remain unsupported even when declarative inspection succeeds.
type Code struct {
	*corecode.DefaultCodeServer
	*Service
	serverMu          sync.Mutex
	initializedSource string
}

func NewCode(svc *Service) *Code {
	return &Code{
		Service:           svc,
		DefaultCodeServer: corecode.NewDefaultCodeServer("."),
	}
}

// InitServer creates the DefaultCodeServer once sourceDir is resolved.
// Uses CachedVFS for in-memory file tree caching with fsnotify updates.
func (c *Code) InitServer() {
	source := filepath.Clean(c.sourceDir())
	if c.DefaultCodeServer != nil && c.initializedSource == source {
		return
	}
	previous := c.DefaultCodeServer
	c.Wool.Debug("binding generic Code source", wool.DirField(source))
	c.DefaultCodeServer = corecode.NewDefaultCodeServer(source, corecode.WithCachedFS())
	c.initializedSource = source
	if previous != nil {
		_ = previous.Close()
	}
}

func (c *Code) ensureInit() {
	c.InitServer()
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
	c.serverMu.Lock()
	defer c.serverMu.Unlock()
	c.ensureInit()
	return c.DefaultCodeServer.Execute(ctx, req)
}
