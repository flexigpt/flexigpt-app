package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// StoreRecord is one non-secret Artifact Store-scoped overlay.
//
// It is intentionally not attached to an Artifact, Root, Source, Definition,
// or package. It is used for application-local non-secret feature state that
// belongs in Artifact Store but has no source-backed Artifact owner.
type StoreRecord struct {
	Namespace     Namespace       `json:"namespace"`
	SchemaVersion string          `json:"schemaVersion"`
	Payload       json.RawMessage `json:"payload"`
	Revision      uint64          `json:"revision"`
	CreatedAt     time.Time       `json:"createdAt"`
	ModifiedAt    time.Time       `json:"modifiedAt"`
}

func (r StoreRecord) Validate() error {
	if err := r.Namespace.Validate(); err != nil {
		return err
	}
	if err := storeSpec.ValidateRequiredText(
		"store overlay schema version",
		r.SchemaVersion,
		storeSpec.MaxVersionBytes,
	); err != nil {
		return err
	}

	canonical, err := CanonicalPayload(r.Payload)
	if err != nil {
		return err
	}
	if !bytes.Equal(canonical, r.Payload) {
		return fmt.Errorf(
			"%w: store overlay payload is not canonical",
			storeSpec.ErrInvalid,
		)
	}

	if r.Revision == 0 {
		return fmt.Errorf(
			"%w: store overlay revision is required",
			storeSpec.ErrInvalid,
		)
	}
	if r.CreatedAt.IsZero() || r.ModifiedAt.IsZero() {
		return fmt.Errorf(
			"%w: store overlay timestamps are required",
			storeSpec.ErrInvalid,
		)
	}
	if r.ModifiedAt.Before(r.CreatedAt) {
		return fmt.Errorf(
			"%w: store overlay modified time precedes creation",
			storeSpec.ErrInvalid,
		)
	}
	return nil
}

func (r StoreRecord) Clone() StoreRecord {
	output := r
	output.Payload = append(json.RawMessage(nil), r.Payload...)
	return output
}

// StorePutRequest replaces one complete store-scoped overlay.
//
// ExpectedRevision is zero only when creating a new overlay.
type StorePutRequest struct {
	Namespace     Namespace       `json:"namespace"`
	SchemaVersion string          `json:"schemaVersion"`
	Payload       json.RawMessage `json:"payload"`

	ExpectedRevision uint64 `json:"expectedRevision"`
}

func (r StorePutRequest) Validate() error {
	if err := r.Namespace.Validate(); err != nil {
		return err
	}
	if err := storeSpec.ValidateRequiredText(
		"store overlay schema version",
		r.SchemaVersion,
		storeSpec.MaxVersionBytes,
	); err != nil {
		return err
	}

	return nil
}
