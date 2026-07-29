package definition

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

type Selector struct {
	Kind              basespec.ArtifactKind `json:"kind"`
	LogicalName       basespec.LogicalName  `json:"logicalName,omitempty"`
	VersionConstraint string                `json:"versionConstraint,omitempty"`
	Labels            map[string]string     `json:"labels,omitempty"`
}

func (s Selector) Validate() error {
	if err := basespec.ValidateArtifactKind(s.Kind); err != nil {
		return fmt.Errorf("selector: %w", err)
	}
	if s.LogicalName != "" {
		if err := basespec.ValidateLogicalName(s.LogicalName); err != nil {
			return fmt.Errorf("selector: %w", err)
		}
	}
	if err := basespec.ValidateOptionalText(
		"selector version constraint",
		s.VersionConstraint,
		basespec.MaxVersionBytes,
	); err != nil {
		return err
	}
	if len(s.Labels) > basespec.MaxLabels {
		return fmt.Errorf(
			"%w: selector labels exceed %d entries",
			basespec.ErrInvalid,
			basespec.MaxLabels,
		)
	}
	for key, value := range s.Labels {
		if err := basespec.ValidateIdentifier(
			"selector label key",
			key,
			basespec.MaxKindBytes,
		); err != nil {
			return err
		}
		if err := basespec.ValidateRequiredText(
			"selector label value",
			value,
			basespec.MaxLabelValueBytes,
		); err != nil {
			return err
		}
	}
	return nil
}
