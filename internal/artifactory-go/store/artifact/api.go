package artifact

import (
	"context"
	"encoding/json"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
)

type API interface {
	Get(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
	) (artifactModel.Artifact, error)

	// GetDefinition resolves the immutable admitted Definition currently
	// linked by ref.
	//
	// This is intentionally Artifact-owned rather than a direct Definition
	// repository lookup. The implementation verifies the Artifact's current
	// resolved-definition link and preserves the existing incompatible-state
	// behavior used for source-kind transition diagnostics.
	//
	// Call definition.API.GetDefinition when the caller already owns a
	// RootID and Definition digest and does not need Artifact relationship
	// validation.
	GetDefinition(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
	) (definitionModel.Definition, error)

	// GetMany is for callers that need complete Artifact entities.
	// Listing should normally go through artifact/catalog.API.
	GetMany(
		ctx context.Context,
		refs []artifactModel.ArtifactRef,
	) ([]artifactModel.Artifact, error)

	FindByOrigin(
		ctx context.Context,
		rootID rootModel.RootID,
		binding artifactModel.SourceBinding,
		kind artifactModel.ArtifactKind,
	) (artifactModel.Artifact, error)

	SetEnabled(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
		expectedRevision uint64,
		enabled bool,
	) (artifactModel.Artifact, error)

	SetDisplayName(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
		expectedRevision uint64,
		displayName string,
	) (artifactModel.Artifact, error)

	UpdateData(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
		expectedRevision uint64,
		data json.RawMessage,
	) (artifactModel.Artifact, error)

	Purge(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
		expectedRevision uint64,
	) error
}
