package compositionapi

import (
	"context"
	"encoding/json"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/resource"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
)

type RootAPI interface {
	Create(
		ctx context.Context,
		draft root.RootDraft,
	) (root.Root, error)

	Get(
		ctx context.Context,
		id root.RootID,
	) (root.Root, error)

	List(ctx context.Context) ([]root.Root, error)

	Update(
		ctx context.Context,
		id root.RootID,
		update root.RootUpdate,
	) (root.Root, error)

	Retire(
		ctx context.Context,
		id root.RootID,
		expectedRevision uint64,
	) (root.Root, error)

	Purge(
		ctx context.Context,
		id root.RootID,
		expectedRevision uint64,
	) error
}

type SourceAPI interface {
	Create(
		ctx context.Context,
		rootID root.RootID,
		draft source.Draft,
	) (source.Summary, error)

	Ensure(
		ctx context.Context,
		rootID root.RootID,
		draft source.Draft,
	) (source.Summary, bool, error)

	Discard(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
		expectedRevision uint64,
	) error

	Get(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
	) (source.Summary, error)

	List(
		ctx context.Context,
		rootID root.RootID,
	) ([]source.Summary, error)

	Update(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
		update source.Update,
	) (source.Summary, error)

	Retire(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
		expectedRevision uint64,
	) (source.Summary, error)

	Purge(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
		expectedRevision uint64,
	) error

	Kinds() []source.SourceKind
}

type DiscoveryAPI interface {
	RefreshRoot(
		ctx context.Context,
		rootID root.RootID,
	) (refresh.RefreshRootResult, error)

	RefreshSource(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
	) (refresh.RefreshSourceResult, error)

	InspectSource(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
	) (source.RefreshInspection, error)
}

type ArtifactAPI interface {
	Get(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) (artifact.Artifact, error)

	ListByRoot(
		ctx context.Context,
		rootID root.RootID,
	) ([]artifact.Artifact, error)

	ListBySource(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
	) ([]artifact.Artifact, error)

	FindByIdentity(
		ctx context.Context,
		rootID root.RootID,
		kind artifact.ArtifactKind,
		logicalName basespec.LogicalName,
	) ([]artifact.Artifact, error)

	FindByOrigin(
		ctx context.Context,
		rootID root.RootID,
		binding artifact.SourceBinding,
		kind artifact.ArtifactKind,
	) (artifact.Artifact, error)

	GetDefinition(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) (definition.Definition, error)

	SetEnabled(
		ctx context.Context,
		ref artifact.ArtifactRef,
		expectedRevision uint64,
		enabled bool,
	) (artifact.Artifact, error)

	SetDisplayName(
		ctx context.Context,
		ref artifact.ArtifactRef,
		expectedRevision uint64,
		displayName string,
	) (artifact.Artifact, error)

	UpdateData(
		ctx context.Context,
		ref artifact.ArtifactRef,
		expectedRevision uint64,
		data json.RawMessage,
	) (artifact.Artifact, error)

	Purge(
		ctx context.Context,
		ref artifact.ArtifactRef,
		expectedRevision uint64,
	) error
}

type ResourceAPI interface {
	ResolveArtifact(
		ctx context.Context,
		ref artifact.ArtifactRef,
		options resource.ResolveOptions,
	) (resource.ResolvedArtifact, error)

	ResolveVerifiedLocalPath(
		ctx context.Context,
		resolved resource.ResolvedArtifact,
		localLocator basespec.Locator,
	) (string, error)

	ReadSourceEntry(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
		locator basespec.Locator,
		maximumBytes int64,
	) (resource.VerifiedEntry, error)

	StatSourceEntry(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
		locator basespec.Locator,
	) (source.Entry, error)

	ReadSourceTree(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
		base basespec.Locator,
		include []string,
		exclude []string,
		maximumEntries int,
		maximumBytes int64,
	) ([]resource.VerifiedEntry, error)

	SupportsLocalPath(kind source.SourceKind) bool
}

type SchemaAPI interface {
	CanonicalizeExpected(
		ctx context.Context,
		expected schema.Key,
		raw []byte,
	) (schema.ParsedDocument, error)
}

type ManagedArtifactAPI interface {
	Publish(
		ctx context.Context,
		request artifact.PublishArtifactRequest,
	) (artifact.PublishArtifactResult, error)

	Remove(
		ctx context.Context,
		request artifact.RemoveArtifactRequest,
	) error
}

type ProtectionAPI interface {
	IsProtectedRoot(rootID root.RootID) bool
	RequirePrivilegedInstaller(ctx context.Context) error
}
