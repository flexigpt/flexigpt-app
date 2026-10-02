package resolve

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec/artifact"
)

// ResolvePluginMembers resolves only direct Plugin membership.
//
// It retains ordinary alias traversal, named lookup, scope handling, local
// locator resolution, selector expansion, ambiguity handling, and contained
// member matching.
//
// It deliberately does not recursively expand each selected member's own
// capabilities. Collection listing and membership inspection need direct
// selection, not a complete Agent, Skill, MCP, Tool, or Workflow graph.
func (r *Resolver) ResolvePluginMembers(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (*ResolvedEntry, error) {
	if r == nil || r.artifacts == nil {
		return nil, basespec.ErrClosed
	}
	if err := validateResolutionContext(ctx); err != nil {
		return nil, err
	}
	if err := ref.Validate(); err != nil {
		return nil, err
	}

	terminal, err := r.resolveTerminalArtifact(
		ctx,
		ref,
		declaration.TypePlugin,
		"",
	)
	if err != nil {
		return nil, err
	}

	state := newResolutionState()
	state.directRoot = &terminal
	return r.resolveArtifact(
		ctx,
		&state,
		terminal,
		declaration.TypePlugin,
		"",
		0,
	)
}
