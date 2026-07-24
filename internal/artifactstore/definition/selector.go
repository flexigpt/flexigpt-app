package definition

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
)

type Selector struct {
	Kind              artifactstore.ArtifactKind `json:"kind"`
	LogicalName       artifactstore.LogicalName  `json:"logicalName,omitempty"`
	VersionConstraint string                     `json:"versionConstraint,omitempty"`
	Labels            map[string]string          `json:"labels,omitempty"`
}

func (s Selector) Validate() error {
	if err := artifactstore.ValidateArtifactKind(s.Kind); err != nil {
		return fmt.Errorf("selector: %w", err)
	}
	if s.LogicalName != "" {
		if err := artifactstore.ValidateLogicalName(s.LogicalName); err != nil {
			return fmt.Errorf("selector: %w", err)
		}
	}
	if err := artifactstore.ValidateOptionalText(
		"selector version constraint",
		s.VersionConstraint,
		artifactstore.MaxVersionBytes,
	); err != nil {
		return err
	}
	if len(s.Labels) > artifactstore.MaxLabels {
		return fmt.Errorf(
			"%w: selector labels exceed %d entries",
			artifactstore.ErrInvalid,
			artifactstore.MaxLabels,
		)
	}
	for key, value := range s.Labels {
		if err := artifactstore.ValidateIdentifier(
			"selector label key",
			key,
			artifactstore.MaxKindBytes,
		); err != nil {
			return err
		}
		if err := artifactstore.ValidateRequiredText(
			"selector label value",
			value,
			artifactstore.MaxLabelValueBytes,
		); err != nil {
			return err
		}
	}
	return nil
}
