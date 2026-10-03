package model

import (
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

// ParsedDocument is a canonical document accepted by a registered schema
// codec. It is a Store domain value, not a provider-private type.
type ParsedDocument struct {
	Key    Key
	Digest cryptoutil.Digest
	Raw    json.RawMessage
}

func (d ParsedDocument) Validate() error {
	if err := d.Key.Validate(); err != nil {
		return err
	}
	if err := cryptoutil.ValidateDigest(d.Digest); err != nil {
		return err
	}
	if len(d.Raw) == 0 {
		return fmt.Errorf(
			"%w: parsed document is empty",
			spec.ErrInvalid,
		)
	}
	return nil
}

func (d ParsedDocument) Clone() ParsedDocument {
	output := d
	output.Raw = append(json.RawMessage(nil), d.Raw...)
	return output
}
