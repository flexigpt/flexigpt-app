package definition

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

func validateLabels(
	subject string,
	values map[string]string,
) error {
	if len(values) > basespec.MaxLabels {
		return fmt.Errorf(
			"%w: %s labels exceed %d entries",
			basespec.ErrInvalid,
			subject,
			basespec.MaxLabels,
		)
	}
	for key, value := range values {
		if err := basespec.ValidateIdentifier(
			subject+" label key",
			key,
			basespec.MaxKindBytes,
		); err != nil {
			return err
		}
		if err := basespec.ValidateRequiredText(
			subject+" label value",
			value,
			basespec.MaxLabelValueBytes,
		); err != nil {
			return err
		}
	}
	return nil
}
