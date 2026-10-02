package definition

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/artifact"
)

type Selector struct {
	Kind              artifact.ArtifactKind `json:"kind"`
	LogicalName       model.LogicalName     `json:"logicalName,omitempty"`
	VersionConstraint string                `json:"versionConstraint,omitempty"`
	Labels            map[string]string     `json:"labels,omitempty"`
}

func (s Selector) Validate() error {
	if err := s.Kind.Validate(); err != nil {
		return fmt.Errorf("selector: %w", err)
	}
	if s.LogicalName != "" {
		if err := s.LogicalName.Validate(); err != nil {
			return fmt.Errorf("selector: %w", err)
		}
	}
	if err := model.ValidateOptionalText(
		"selector version constraint",
		s.VersionConstraint,
		model.MaxVersionBytes,
	); err != nil {
		return err
	}
	if err := model.ValidateLabels("selector", s.Labels); err != nil {
		return err
	}
	return nil
}
