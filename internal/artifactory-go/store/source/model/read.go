package model

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// ValidateRead verifies Source metadata already admitted through Source
// configuration normalization. It intentionally avoids canonicalizing or
// reparsing Config on repeated repository and runtime reads.
func (s Source) ValidateRead() error {
	if err := s.Summary().Validate(); err != nil {
		return err
	}
	return validateReadConfig(s.Config)
}

func validateReadConfig(raw json.RawMessage) error {
	if len(raw) == 0 || len(raw) > spec.MaxConfigBytes {
		return fmt.Errorf("%w: source config is empty or exceeds its size limit", spec.ErrInvalid)
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) < 2 || trimmed[0] != '{' || trimmed[len(trimmed)-1] != '}' {
		return fmt.Errorf("%w: source config must be a JSON object", spec.ErrInvalid)
	}
	return nil
}
