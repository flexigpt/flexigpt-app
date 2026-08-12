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
	if err := basespec.ValidateLabels("selector", s.Labels); err != nil {
		return err
	}
	return nil
}
