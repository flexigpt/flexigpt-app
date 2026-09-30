package compositionapi

import (
	"context"
	"encoding/json"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/overlay"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/resource"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/secret"
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

	// GetMany loads complete Artifact entities for callers that genuinely need
	// source state, diagnostics, local data, or resource materialization.
	//
	// Ordinary listings must use ListByRoot, ListBySource, or
	// FindByIdentity instead.
	GetMany(
		ctx context.Context,
		refs []artifact.ArtifactRef,
	) ([]artifact.Artifact, error)

	ListByRoot(
		ctx context.Context,
		rootID root.RootID,
		options catalog.ListOptions,
	) ([]catalog.Entry, error)

	ListBySource(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
		options catalog.ListOptions,
	) ([]catalog.Entry, error)

	FindByIdentity(
		ctx context.Context,
		rootID root.RootID,
		kind artifact.ArtifactKind,
		logicalName basespec.LogicalName,
		options catalog.ListOptions,
	) ([]catalog.Entry, error)

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

	// GetDefinitions loads immutable admitted Definitions in bulk. It is used
	// by ListOptions.IncludeDocument and by consumers that already selected
	// immutable Definition keys from catalog metadata.
	GetDefinitions(
		ctx context.Context,
		keys []definition.Key,
	) ([]definition.Definition, error)

	// SetEnabled changes universal local Artifact metadata. Unlike other local
	// Artifact mutations, it is valid for Artifacts in protected Roots and does
	// not require installer privilege.
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

// ProtectedOverlayAPI owns non-secret local state for Artifacts in protected
// Roots. Mutable Artifact local state continues to use Artifact.Data.
type ProtectedOverlayAPI interface {
	Get(
		ctx context.Context,
		ref artifact.ArtifactRef,
		namespace overlay.Namespace,
	) (overlay.Record, bool, error)

	Put(
		ctx context.Context,
		request overlay.PutRequest,
	) (overlay.Record, error)

	// Delete removes the complete protected overlay and every secret binding
	// below the same Artifact/namespace pair.
	Delete(
		ctx context.Context,
		ref artifact.ArtifactRef,
		namespace overlay.Namespace,
		expectedArtifactRevision uint64,
		expectedOverlayRevision uint64,
	) error
}

// StoreOverlayAPI owns non-secret store-scoped local state. It is used for
// application-local feature preferences that do not belong to one source-backed
// Artifact, Root, Source, or Definition.
type StoreOverlayAPI interface {
	GetStoreOverlay(
		ctx context.Context,
		namespace overlay.Namespace,
	) (overlay.StoreRecord, bool, error)

	PutStoreOverlay(
		ctx context.Context,
		request overlay.StorePutRequest,
	) (overlay.StoreRecord, error)

	DeleteStoreOverlay(
		ctx context.Context,
		namespace overlay.Namespace,
		expectedRevision uint64,
	) error
}

// SecretBindingAPI manages Artifact-local secret references and public-safe
// metadata. It never returns a plaintext secret value.
type SecretBindingAPI interface {
	GetBinding(
		ctx context.Context,
		key secret.BindingKey,
	) (secret.Binding, bool, error)

	ReplaceBinding(
		ctx context.Context,
		request secret.ReplaceBindingRequest,
	) (secret.Binding, error)

	ClearBinding(
		ctx context.Context,
		request secret.ClearBindingRequest,
	) error
}

// SecretRuntimeAPI is the narrow trusted runtime capability used by Model,
// MCP, and future runtime integrations. It must never be exposed through
// transport-facing wrappers.
type SecretRuntimeAPI interface {
	ReadBinding(
		ctx context.Context,
		key secret.BindingKey,
		expectedRef secret.Ref,
	) (string, secret.Binding, error)
}

// LocalStateMaintenanceAPI is reserved for trusted lifecycle code such as
// managed Artifact deletion and protected compiled-package reconciliation.
type LocalStateMaintenanceAPI interface {
	PurgeArtifactLocalState(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) error

	DrainSecretGarbage(
		ctx context.Context,
	) error
}

type ProtectionAPI interface {
	IsProtectedRoot(rootID root.RootID) bool
	RequirePrivilegedInstaller(ctx context.Context) error
}
