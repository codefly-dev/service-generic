package main

import (
	"context"
	"path/filepath"
	"sync"

	corecode "github.com/codefly-dev/core/code"
	codev0 "github.com/codefly-dev/core/generated/go/codefly/services/code/v0"
	"github.com/codefly-dev/core/wool"
)

// Code implements the codefly Code gRPC service for the generic agent.
// It embeds DefaultCodeServer directly — NO runtime or mutation overrides.
// No semantic analyzer is installed, so the agent stays free of the
// tree-sitter CGO stack and builds with CGO_ENABLED=0.
//
// Provides: ReadFile, WriteFile, CreateFile, DeleteFile, MoveFile, ListFiles,
// Search, GitLog, GitDiff, GitShow, GitBlame, ApplyEdit, GetProjectInfo
// (declarative language classification and file hashes only).
//
// Source-semantics inspection — dependency, import, and symbol extraction, plus
// the semantic index — reports a typed unsupported operation because it would
// require the CGO analyzer. Runtime build/test/lint remain unsupported too.
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
	c.DefaultCodeServer = corecode.NewDefaultCodeServer(source, corecode.WithCachedFS())
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
