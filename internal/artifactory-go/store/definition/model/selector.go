package model

import (
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

type Selector struct {
	Kind              artifactModel.ArtifactKind `json:"kind"`
	LogicalName       spec.LogicalName           `json:"logicalName,omitempty"`
	VersionConstraint string                     `json:"versionConstraint,omitempty"`
	Labels            map[string]string          `json:"labels,omitempty"`
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
	if err := spec.ValidateOptionalText(
		"selector version constraint",
		s.VersionConstraint,
		spec.MaxVersionBytes,
	); err != nil {
		return err
	}
	if err := spec.ValidateLabels("selector", s.Labels); err != nil {
		return err
	}
	return nil
}
