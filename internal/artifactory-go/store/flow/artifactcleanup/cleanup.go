package artifactcleanup

import (
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	overlayModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// PurgeRequest describes one atomic Artifact-local cleanup operation.
//
// AllNamespaces removes every protected overlay and secret binding attached to
// the Artifact. Namespaces selects only listed overlay/binding namespaces.
// DataNamespaces always selects only the listed Artifact.Data fields.
type PurgeRequest struct {
	Artifact artifactModel.ArtifactRef `json:"artifact"`

	ExpectedArtifactRevision uint64 `json:"expectedArtifactRevision"`

	AllNamespaces  bool                          `json:"allNamespaces,omitempty"`
	Namespaces     []overlayModel.Namespace      `json:"namespaces,omitempty"`
	DataNamespaces []artifactModel.DataNamespace `json:"dataNamespaces,omitempty"`
}

func (r PurgeRequest) Validate() error {
	if err := r.Artifact.Validate(); err != nil {
		return err
	}
	if r.ExpectedArtifactRevision == 0 {
		return fmt.Errorf(
			"%w: expected Artifact revision is required",
			spec.ErrInvalid,
		)
	}
	if r.AllNamespaces && len(r.Namespaces) != 0 {
		return fmt.Errorf(
			"%w: all-namespaces cleanup cannot also select namespaces",
			spec.ErrInvalid,
		)
	}
	if !r.AllNamespaces &&
		len(r.Namespaces) == 0 &&
		len(r.DataNamespaces) == 0 {
		return fmt.Errorf(
			"%w: Artifact cleanup selects no local state",
			spec.ErrInvalid,
		)
	}

	seenOverlay := make(map[overlayModel.Namespace]struct{}, len(r.Namespaces))
	for _, namespace := range r.Namespaces {
		if err := namespace.Validate(); err != nil {
			return err
		}
		if _, duplicate := seenOverlay[namespace]; duplicate {
			return fmt.Errorf(
				"%w: Artifact cleanup repeats overlay namespace %q",
				spec.ErrInvalid,
				namespace,
			)
		}
		seenOverlay[namespace] = struct{}{}
	}

	seenData := make(map[artifactModel.DataNamespace]struct{}, len(r.DataNamespaces))
	for _, namespace := range r.DataNamespaces {
		if err := namespace.Validate(); err != nil {
			return err
		}
		if _, duplicate := seenData[namespace]; duplicate {
			return fmt.Errorf(
				"%w: Artifact cleanup repeats data namespace %q",
				spec.ErrInvalid,
				namespace,
			)
		}
		seenData[namespace] = struct{}{}
	}
	return nil
}

// PurgeResult returns the Artifact revision visible after cleanup. It changes
// only when selected Artifact.Data fields were removed.
type PurgeResult struct {
	Artifact artifactModel.Artifact `json:"artifact"`
}

func (r PurgeResult) Validate() error {
	return r.Artifact.Validate()
}
