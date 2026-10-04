package model

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
)

// ValidateRead verifies the structural invariants required when an Artifact
// has already crossed its write/admission boundary. It intentionally avoids
// canonicalizing or reparsing local Data again on every persisted read.
func (a Artifact) ValidateRead() error {
	if err := a.ID.Validate(); err != nil {
		return err
	}
	if err := a.RootID.Validate(); err != nil {
		return err
	}
	if err := a.Binding.Validate(); err != nil {
		return err
	}
	if err := a.Kind.Validate(); err != nil {
		return err
	}
	if err := a.LogicalName.Validate(); err != nil {
		return err
	}
	if err := a.LogicalVersion.Validate(true); err != nil {
		return err
	}
	if err := a.State.Validate(a.ResolvedDefinition, a.SourceContentDigest); err != nil {
		return err
	}
	if err := diagnostic.Validate(a.Diagnostics); err != nil {
		return err
	}
	if err := spec.ValidateRequiredText("Artifact display name", a.DisplayName, spec.MaxDisplayNameBytes); err != nil {
		return err
	}
	if err := validateReadData(a.Data); err != nil {
		return err
	}
	if a.Revision == 0 {
		return fmt.Errorf("%w: Artifact revision must be positive", spec.ErrInvalid)
	}
	if a.CreatedAt.IsZero() || a.ModifiedAt.IsZero() {
		return fmt.Errorf("%w: Artifact timestamps are required", spec.ErrInvalid)
	}
	if a.ModifiedAt.Before(a.CreatedAt) {
		return fmt.Errorf("%w: Artifact modified time precedes creation", spec.ErrInvalid)
	}
	return nil
}

func validateReadData(raw json.RawMessage) error {
	if len(raw) == 0 || len(raw) > spec.MaxLocalDataBytes {
		return fmt.Errorf("%w: Artifact local data is empty or exceeds its size limit", spec.ErrInvalid)
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) < 2 || trimmed[0] != '{' || trimmed[len(trimmed)-1] != '}' {
		return fmt.Errorf("%w: Artifact local data must be a JSON object", spec.ErrInvalid)
	}
	return nil
}
