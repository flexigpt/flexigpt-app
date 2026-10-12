package model

import (
	"bytes"
	"fmt"

	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// ValidateAdmitted checks inexpensive structural guarantees of a Definition
// that has already passed Definition.Admit. It deliberately does not
// re-canonicalize or rehash Body on ordinary immutable reads.
func ValidateAdmitted(value Definition) error {
	if err := validateDefinitionFields(value); err != nil {
		return err
	}
	body := bytes.TrimSpace(value.Body)
	if len(body) < 2 || len(body) > storeSpec.MaxDefinitionBodyBytes || body[0] != '{' || body[len(body)-1] != '}' {
		return fmt.Errorf("%w: admitted Definition body must be a bounded JSON object", storeSpec.ErrInvalid)
	}
	return nil
}
