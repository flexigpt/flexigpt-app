package main

import (
	"context"
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	mcpRuntime "github.com/flexigpt/flexigpt-app/internal/mcp/runtime"
)

// mcpRuntimeInvalidator breaks the Bundle API to runtime constructor cycle.
// It is fully configured before any public MCP operation is exposed.
type mcpRuntimeInvalidator struct {
	mu      sync.RWMutex
	runtime *mcpRuntime.MCPRuntimeManager
}

func newMCPRuntimeInvalidator() *mcpRuntimeInvalidator {
	return &mcpRuntimeInvalidator{}
}

func (i *mcpRuntimeInvalidator) Set(
	runtime *mcpRuntime.MCPRuntimeManager,
) {
	i.mu.Lock()
	i.runtime = runtime
	i.mu.Unlock()
}

func (i *mcpRuntimeInvalidator) Invalidate(
	ctx context.Context,
	ref artifact.ArtifactRef,
) error {
	i.mu.RLock()
	runtime := i.runtime
	i.mu.RUnlock()

	if runtime == nil {
		return nil
	}
	return runtime.Invalidate(ctx, ref)
}

func (i *mcpRuntimeInvalidator) InvalidateCollection(
	ctx context.Context,
	ref collection.CollectionRef,
) error {
	i.mu.RLock()
	runtime := i.runtime
	i.mu.RUnlock()

	if runtime == nil {
		return nil
	}
	return runtime.InvalidateCollection(ctx, ref)
}
