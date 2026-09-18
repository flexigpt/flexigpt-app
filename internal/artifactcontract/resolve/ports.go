package resolve

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
)

// ArtifactReader is the Root-scoped read boundary for normal resolver work.
type ArtifactReader interface {
	Get(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) (artifact.Artifact, error)

	FindByIdentity(
		ctx context.Context,
		rootID root.RootID,
		kind artifact.ArtifactKind,
		logicalName basespec.LogicalName,
	) ([]artifact.Artifact, error)

	GetDefinition(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) (definition.Definition, error)
}

// SourceArtifactReader is required for member selector expansion only.
//
// The returned records are source-backed Artifact occurrences. Selector
// matching is done by this package, using source locator, subresource locator,
// Artifact type, and logical name.
type SourceArtifactReader interface {
	ListBySource(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
	) ([]artifact.Artifact, error)
}

// SourceEntryInspector confirms physical Source entry metadata without
// exposing Source configuration or native filesystem paths. Selector
// resolution uses it only to verify that a local selector base is a
// directory.
type SourceEntryInspector interface {
	StatSourceEntry(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
		locator basespec.Locator,
	) (source.Entry, error)
}

type RefreshTarget struct {
	RootID   root.RootID
	SourceID source.SourceID
}

func (t RefreshTarget) Validate() error {
	if err := t.RootID.Validate(); err != nil {
		return err
	}
	return t.SourceID.Validate()
}

type RefreshDirective struct {
	Target  RefreshTarget
	Changed bool
}

type SelectorRefreshRequest struct {
	Parent   artifact.Artifact
	Selector declaration.Selector
}

type LocatedMemberRefreshRequest struct {
	Parent artifact.Artifact
	Member declaration.Entry
}

// RefreshCoordinator belongs at application composition boundaries.
//
// It owns Source discovery configuration and Source refresh. The resolver
// owns only the reachable selector and local-locator closure traversal.
type RefreshCoordinator interface {
	PrepareSelectorDiscovery(
		ctx context.Context,
		request SelectorRefreshRequest,
	) ([]RefreshDirective, error)

	PrepareLocatedMemberDiscovery(
		ctx context.Context,
		request LocatedMemberRefreshRequest,
	) ([]RefreshDirective, error)

	RefreshSource(
		ctx context.Context,
		target RefreshTarget,
	) error
}
