package composition

import (
	"context"

	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

// ArtifactReader is the entity read boundary used by declaration resolution.
//
// It intentionally follows the Artifact's current immutable Definition link.
// Consumers should satisfy this with store/artifact.API.
//
// This is not a generic Definition repository. Resolver work begins with an
// ArtifactRef and must preserve Artifact state and Artifact -> Definition
// linkage validation.
type ArtifactReader interface {
	Get(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
	) (artifactModel.Artifact, error)

	GetDefinition(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
	) (definitionModel.Definition, error)
}

// ArtifactCatalogReader is the committed Artifact read-projection boundary.
//
// Consumers should satisfy this with store/artifact/catalog.API. It is
// intentionally separate from ArtifactReader because catalog queries return
// lightweight committed entries rather than complete mutable Artifact entities.
type ArtifactCatalogReader interface {
	FindByIdentity(
		ctx context.Context,
		rootID rootModel.RootID,
		kind artifactModel.ArtifactKind,
		logicalName spec.LogicalName,
		options catalogModel.ListOptions,
	) ([]catalogModel.Entry, error)

	ListBySource(
		ctx context.Context,
		rootID rootModel.RootID,
		sourceID sourceModel.SourceID,
		options catalogModel.ListOptions,
	) ([]catalogModel.Entry, error)
}

// SourceEntryInspector confirms physical Source entry metadata without
// exposing Source configuration or native filesystem paths.
//
// Consumers should satisfy this with store/flow/resource.API.
type SourceEntryInspector interface {
	StatSourceEntry(
		ctx context.Context,
		rootID rootModel.RootID,
		sourceID sourceModel.SourceID,
		locator spec.Locator,
	) (sourceModel.Entry, error)
}

type RefreshTarget struct {
	RootID   rootModel.RootID
	SourceID sourceModel.SourceID
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
	Parent   artifactModel.Artifact
	Selector declaration.Selector
}

type LocatedMemberRefreshRequest struct {
	Parent artifactModel.Artifact
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
