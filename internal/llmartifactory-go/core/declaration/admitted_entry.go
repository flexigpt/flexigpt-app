package declaration

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// DecodeAdmittedEntryInto decodes one Entry whose containing Definition has
// already crossed generic Definition admission.
//
// It intentionally does not execute a family JSON Schema. Callers must use it
// only after Definition admission or after an enclosing declaration schema has
// already validated the entry. Family semantic validation remains mandatory.
func DecodeAdmittedEntryInto(
	entry Entry,
	target any,
) error {
	if target == nil {
		return fmt.Errorf(
			"%w: admitted declaration decode target is nil",
			spec.ErrInvalid,
		)
	}
	return entry.DecodeInto(target)
}
