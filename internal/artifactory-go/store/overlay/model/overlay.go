package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

// Namespace identifies one application-owned protected Artifact overlay slot.
//
// Artifact Store owns persistence, revision checks, root protection, and
// lifecycle cleanup. The owning feature owns the payload schema and semantic
// interpretation.
type Namespace string

func (v Namespace) Validate() error {
	return spec.ValidateIdentifier(
		"protected overlay namespace",
		string(v),
		spec.MaxKindBytes,
	)
}

// Record is one non-secret local overlay for one protected Artifact.
//
// Payload must never contain a secret value. Secret references and hashes are
// persisted through the separate Artifact Store secret-binding extension.
type Record struct {
	Artifact      artifactModel.ArtifactRef `json:"artifact"`
	Namespace     Namespace                 `json:"namespace"`
	SchemaVersion string                    `json:"schemaVersion"`
	Payload       json.RawMessage           `json:"payload"`
	Revision      uint64                    `json:"revision"`
	CreatedAt     time.Time                 `json:"createdAt"`
	ModifiedAt    time.Time                 `json:"modifiedAt"`
}

func (r Record) Validate() error {
	if err := r.Artifact.Validate(); err != nil {
		return err
	}
	if err := r.Namespace.Validate(); err != nil {
		return err
	}
	if err := spec.ValidateRequiredText(
		"protected overlay schema version",
		r.SchemaVersion,
		spec.MaxVersionBytes,
	); err != nil {
		return err
	}

	canonical, err := CanonicalPayload(r.Payload)
	if err != nil {
		return err
	}
	if !bytes.Equal(canonical, r.Payload) {
		return fmt.Errorf(
			"%w: protected overlay payload is not canonical",
			spec.ErrInvalid,
		)
	}

	if r.Revision == 0 {
		return fmt.Errorf(
			"%w: protected overlay revision is required",
			spec.ErrInvalid,
		)
	}
	if r.CreatedAt.IsZero() || r.ModifiedAt.IsZero() {
		return fmt.Errorf(
			"%w: protected overlay timestamps are required",
			spec.ErrInvalid,
		)
	}
	if r.ModifiedAt.Before(r.CreatedAt) {
		return fmt.Errorf(
			"%w: protected overlay modified time precedes creation",
			spec.ErrInvalid,
		)
	}
	return nil
}

func (r Record) Clone() Record {
	output := r
	output.Payload = append(json.RawMessage(nil), r.Payload...)
	return output
}

// PutRequest replaces one complete non-secret protected Artifact overlay.
//
// ExpectedArtifactRevision protects against writing an overlay for an Artifact
// whose source-derived identity or local state changed after the caller read
// it. ExpectedOverlayRevision is zero only when creating a new overlay.
type PutRequest struct {
	Artifact artifactModel.ArtifactRef `json:"artifact"`

	Namespace     Namespace       `json:"namespace"`
	SchemaVersion string          `json:"schemaVersion"`
	Payload       json.RawMessage `json:"payload"`

	ExpectedArtifactRevision uint64 `json:"expectedArtifactRevision"`
	ExpectedOverlayRevision  uint64 `json:"expectedOverlayRevision"`
}

func (r PutRequest) Validate() error {
	if err := r.Artifact.Validate(); err != nil {
		return err
	}
	if err := r.Namespace.Validate(); err != nil {
		return err
	}
	if err := spec.ValidateRequiredText(
		"protected overlay schema version",
		r.SchemaVersion,
		spec.MaxVersionBytes,
	); err != nil {
		return err
	}
	if r.ExpectedArtifactRevision == 0 {
		return fmt.Errorf(
			"%w: expected Artifact revision is required",
			spec.ErrInvalid,
		)
	}
	return nil
}

// CanonicalPayload validates one non-secret overlay payload as a canonical
// bounded JSON object.
func CanonicalPayload(
	raw json.RawMessage,
) (json.RawMessage, error) {
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		spec.MaxLocalDataBytes,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: protected overlay payload: %w",
			spec.ErrInvalid,
			err,
		)
	}
	return json.RawMessage(canonical), nil
}
