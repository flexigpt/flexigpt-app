package aggregate

import (
	"context"
	"errors"
	"sort"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	serverMCPDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/domain/server"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/server"
)

type RuntimeInvalidator interface {
	Invalidate(
		ctx context.Context,
		server mcpServer.ServerID,
	) error
}

type Lifecycle struct {
	store interface {
		SaveServerSettings(
			ctx context.Context,
			ref artifactModel.ArtifactRef,
			expectedSettingsRevision uint64,
			data serverMCPDomain.ServerData,
		) error
	}
	runtime RuntimeInvalidator
}

func NewLifecycle(
	store interface {
		SaveServerSettings(
			ctx context.Context,
			ref artifactModel.ArtifactRef,
			expectedSettingsRevision uint64,
			data serverMCPDomain.ServerData,
		) error
	},
	runtime RuntimeInvalidator,
) (*Lifecycle, error) {
	if store == nil || runtime == nil {
		return nil, errors.New("MCP lifecycle dependencies are incomplete")
	}
	return &Lifecycle{
		store:   store,
		runtime: runtime,
	}, nil
}

func (l *Lifecycle) InvalidateServer(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) error {
	if l.runtime == nil {
		return mcpServer.ErrClosed
	}
	serverID, err := runtimeServerIDForArtifact(ref)
	if err != nil {
		return err
	}
	return l.runtime.Invalidate(ctx, serverID)
}

// InvalidateServers invalidates a deterministic unique server set. Policy
// mutation uses this to invalidate only servers whose effective policy can
// change, rather than disconnecting every server in a Root.
func (l *Lifecycle) InvalidateServers(
	ctx context.Context,
	refs []artifactModel.ArtifactRef,
) error {
	if l.runtime == nil {
		return mcpServer.ErrClosed
	}

	unique := make(map[artifactModel.ArtifactRef]struct{}, len(refs))
	for _, ref := range refs {
		unique[ref] = struct{}{}
	}

	ordered := make([]artifactModel.ArtifactRef, 0, len(unique))
	for ref := range unique {
		ordered = append(ordered, ref)
	}
	sort.Slice(ordered, func(left, right int) bool {
		if ordered[left].RootID != ordered[right].RootID {
			return ordered[left].RootID < ordered[right].RootID
		}
		return ordered[left].ArtifactID < ordered[right].ArtifactID
	})

	var output error
	for _, ref := range ordered {
		output = errors.Join(output, l.InvalidateServer(ctx, ref))
	}
	return output
}

func (l *Lifecycle) SaveServerSettings(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedSettingsRevision uint64,
	data serverMCPDomain.ServerData,
) error {
	if err := l.InvalidateServer(ctx, ref); err != nil {
		return err
	}
	return l.store.SaveServerSettings(
		ctx,
		ref,
		expectedSettingsRevision,
		data,
	)
}
