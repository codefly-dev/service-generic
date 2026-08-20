package main

import (
	"context"
	"path/filepath"
	"sync"

	corecode "github.com/codefly-dev/core/code"
	"github.com/codefly-dev/core/code/semantic"
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
// Semantic projection is supplied by Core and exposed through Tooling; project
// bytes never leave the agent. Runtime build/test/lint remain unsupported even
// when declarative and semantic inspection succeed.
//
// Core omits the tree-sitter analyzer by default so Go agents stay CGO-free, so
// the generic agent installs it explicitly — JVM/.NET inspection and semantic
// projection are part of its contract, and it already builds with CGO_ENABLED=1.
type Code struct {
	*corecode.DefaultCodeServer
	*Service
	serverMu          sync.Mutex
	initializedSource string
}

func NewCode(svc *Service) *Code {
	return &Code{
		Service:           svc,
		DefaultCodeServer: corecode.NewDefaultCodeServer(".", corecode.WithSemanticAnalyzer(semantic.New())),
	}
}

// InitServer creates the DefaultCodeServer once sourceDir is resolved.
// Uses CachedVFS for in-memory file tree caching with fsnotify updates.
func (c *Code) InitServer(ctx context.Context) error {
	source, err := c.sourceLocationForCode(ctx)
	if err != nil {
		return err
	}
	source = filepath.Clean(source)
	if c.DefaultCodeServer != nil && c.initializedSource == source {
		return nil
	}
	previous := c.DefaultCodeServer
	c.Wool.Debug("binding generic Code source", wool.DirField(source))
	c.DefaultCodeServer = corecode.NewDefaultCodeServer(source, corecode.WithCachedFS(), corecode.WithSemanticAnalyzer(semantic.New()))
	c.initializedSource = source
	if previous != nil {
		_ = previous.Close()
	}
	return nil
}

func (c *Code) Execute(ctx context.Context, req *codev0.CodeRequest) (*codev0.CodeResponse, error) {
	c.serverMu.Lock()
	defer c.serverMu.Unlock()
	if err := c.InitServer(ctx); err != nil {
		return nil, err
	}
	return c.DefaultCodeServer.Execute(ctx, req)
}
