package aggregate

import (
	"context"
	"errors"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/server"
	mcpDomainServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/server"
)

type RuntimeInvalidator interface {
	Invalidate(
		ctx context.Context,
		server mcpServer.ServerID,
	) error
}

type Lifecycle struct {
	store interface {
		UpdateServerInstallation(
			ctx context.Context,
			ref artifact.ArtifactRef,
			expectedArtifactRevision uint64,
			data mcpDomainServer.ServerData,
		) (artifact.Artifact, error)

		UpdateProtectedServerInstallation(
			ctx context.Context,
			ref artifact.ArtifactRef,
			expectedOverlayRevision uint64,
			data mcpDomainServer.ServerData,
		) error
	}
	runtime RuntimeInvalidator
}

func NewLifecycle(
	store interface {
		UpdateServerInstallation(
			ctx context.Context,
			ref artifact.ArtifactRef,
			expectedArtifactRevision uint64,
			data mcpDomainServer.ServerData,
		) (artifact.Artifact, error)

		UpdateProtectedServerInstallation(
			ctx context.Context,
			ref artifact.ArtifactRef,
			expectedOverlayRevision uint64,
			data mcpDomainServer.ServerData,
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
	ref artifact.ArtifactRef,
) error {
	if l == nil || l.runtime == nil {
		return mcpServer.ErrClosed
	}
	serverID, err := RuntimeServerIDForArtifact(ref)
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
	refs []artifact.ArtifactRef,
) error {
	if l == nil || l.runtime == nil {
		return mcpServer.ErrClosed
	}

	unique := make(map[artifact.ArtifactRef]struct{}, len(refs))
	for _, ref := range refs {
		unique[ref] = struct{}{}
	}

	ordered := make([]artifact.ArtifactRef, 0, len(unique))
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

func (l *Lifecycle) UpdateServerInstallation(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedArtifactRevision uint64,
	data mcpDomainServer.ServerData,
) (artifact.Artifact, error) {
	if err := l.InvalidateServer(ctx, ref); err != nil {
		return artifact.Artifact{}, err
	}
	return l.store.UpdateServerInstallation(
		ctx,
		ref,
		expectedArtifactRevision,
		data,
	)
}

func (l *Lifecycle) UpdateProtectedServerInstallation(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedOverlayRevision uint64,
	data mcpDomainServer.ServerData,
) error {
	if err := l.InvalidateServer(ctx, ref); err != nil {
		return err
	}
	return l.store.UpdateProtectedServerInstallation(
		ctx,
		ref,
		expectedOverlayRevision,
		data,
	)
}
