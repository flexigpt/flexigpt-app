package model

import (
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
)

// RefreshSourceResult is the Store-owned result of one Source refresh.
//
// It belongs outside basespec/source because Artifact imports SourceBinding
// from basespec/source. Keeping result values here avoids an import cycle.
type RefreshSourceResult struct {
	State                 State                      `json:"state"`
	CreatedArtifacts      []artifactModel.ArtifactID `json:"createdArtifacts,omitempty"`
	UpdatedArtifacts      []artifactModel.ArtifactID `json:"updatedArtifacts,omitempty"`
	MissingArtifacts      []artifactModel.ArtifactID `json:"missingArtifacts,omitempty"`
	InvalidArtifacts      []artifactModel.ArtifactID `json:"invalidArtifacts,omitempty"`
	IncompatibleArtifacts []artifactModel.ArtifactID `json:"incompatibleArtifacts,omitempty"`
	Diagnostics           []diagnostic.Diagnostic    `json:"diagnostics,omitempty"`
	Candidates            int                        `json:"candidates"`
}

func (r RefreshSourceResult) Validate() error {
	if err := r.State.Validate(); err != nil {
		return err
	}
	if r.Candidates < 0 {
		return fmt.Errorf(
			"%w: Source refresh candidate count cannot be negative",
			spec.ErrInvalid,
		)
	}
	if err := diagnostic.Validate(r.Diagnostics); err != nil {
		return err
	}

	created := make(map[artifactModel.ArtifactID]struct{})
	for _, artifactID := range r.CreatedArtifacts {
		if err := artifactID.Validate(); err != nil {
			return err
		}
		if _, duplicate := created[artifactID]; duplicate {
			return fmt.Errorf(
				"%w: Source refresh repeats created Artifact %q",
				spec.ErrInvalid,
				artifactID,
			)
		}
		created[artifactID] = struct{}{}
	}

	updated := make(map[artifactModel.ArtifactID]struct{})
	for _, artifactID := range r.UpdatedArtifacts {
		if err := artifactID.Validate(); err != nil {
			return err
		}
		if _, duplicate := updated[artifactID]; duplicate {
			return fmt.Errorf(
				"%w: Source refresh repeats updated Artifact %q",
				spec.ErrInvalid,
				artifactID,
			)
		}
		if _, createdNow := created[artifactID]; createdNow {
			return fmt.Errorf(
				"%w: Source refresh Artifact %q is both created and updated",
				spec.ErrInvalid,
				artifactID,
			)
		}
		updated[artifactID] = struct{}{}
	}

	stateChanged := make(map[artifactModel.ArtifactID]struct{})
	for _, group := range [][]artifactModel.ArtifactID{
		r.MissingArtifacts,
		r.InvalidArtifacts,
		r.IncompatibleArtifacts,
	} {
		for _, artifactID := range group {
			if err := artifactID.Validate(); err != nil {
				return err
			}
			if _, changed := updated[artifactID]; !changed {
				return fmt.Errorf(
					"%w: Source refresh state change %q is not an Artifact update",
					spec.ErrInvalid,
					artifactID,
				)
			}
			if _, duplicate := stateChanged[artifactID]; duplicate {
				return fmt.Errorf(
					"%w: Source refresh repeats Artifact state change %q",
					spec.ErrInvalid,
					artifactID,
				)
			}
			stateChanged[artifactID] = struct{}{}
		}
	}
	return nil
}

func (r RefreshSourceResult) Clone() RefreshSourceResult {
	output := r
	output.State = r.State.Clone()
	output.CreatedArtifacts = append(
		[]artifactModel.ArtifactID(nil),
		r.CreatedArtifacts...,
	)
	output.UpdatedArtifacts = append(
		[]artifactModel.ArtifactID(nil),
		r.UpdatedArtifacts...,
	)
	output.MissingArtifacts = append(
		[]artifactModel.ArtifactID(nil),
		r.MissingArtifacts...,
	)
	output.InvalidArtifacts = append(
		[]artifactModel.ArtifactID(nil),
		r.InvalidArtifacts...,
	)
	output.IncompatibleArtifacts = append(
		[]artifactModel.ArtifactID(nil),
		r.IncompatibleArtifacts...,
	)
	output.Diagnostics = diagnostic.Clone(r.Diagnostics)
	return output
}

type RefreshRootResult struct {
	RootID      rootModel.RootID        `json:"rootID"`
	Sources     []RefreshSourceResult   `json:"sources"`
	Diagnostics []diagnostic.Diagnostic `json:"diagnostics,omitempty"`
}

func (r RefreshRootResult) Validate() error {
	if err := r.RootID.Validate(); err != nil {
		return err
	}
	if err := diagnostic.Validate(r.Diagnostics); err != nil {
		return err
	}

	seen := make(map[sourceModel.SourceID]struct{}, len(r.Sources))
	for index, result := range r.Sources {
		if err := result.Validate(); err != nil {
			return fmt.Errorf(
				"root refresh Sources[%d]: %w",
				index,
				err,
			)
		}
		if result.State.RootID != r.RootID {
			return fmt.Errorf(
				"%w: Source refresh belongs to another Root",
				spec.ErrInvalid,
			)
		}
		if _, duplicate := seen[result.State.SourceID]; duplicate {
			return fmt.Errorf(
				"%w: Root refresh repeats Source %q",
				spec.ErrInvalid,
				result.State.SourceID,
			)
		}
		seen[result.State.SourceID] = struct{}{}
	}
	return nil
}

func (r RefreshRootResult) Clone() RefreshRootResult {
	output := r
	output.Sources = make(
		[]RefreshSourceResult,
		len(r.Sources),
	)
	for index, value := range r.Sources {
		output.Sources[index] = value.Clone()
	}
	output.Diagnostics = diagnostic.Clone(r.Diagnostics)
	return output
}
