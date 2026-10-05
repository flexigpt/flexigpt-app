package artifactcleanup

import (
	"context"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
)

type API interface {
	// PurgeArtifactLocalState removes every generic overlay and binding
	// attached to ref. It intentionally leaves Artifact.Data untouched because
	// generic Store does not own arbitrary family data namespaces.
	PurgeArtifactLocalState(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
	) error

	// CleanupArtifactLocalState atomically removes selected local-data fields,
	// overlays, bindings, and queues detached physical secret values.
	CleanupArtifactLocalState(
		ctx context.Context,
		request PurgeRequest,
	) (PurgeResult, error)
}
