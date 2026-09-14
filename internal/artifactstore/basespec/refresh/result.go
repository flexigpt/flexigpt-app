package refresh

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
)

// RefreshSourceResult is the Store-owned result of one Source refresh.
//
// It belongs outside basespec/source because Artifact imports SourceBinding
// from basespec/source. Keeping result values here avoids an import cycle.
type RefreshSourceResult struct {
	State                 source.RefreshState     `json:"state"`
	CreatedArtifacts      []artifact.ArtifactID   `json:"createdArtifacts,omitempty"`
	UpdatedArtifacts      []artifact.ArtifactID   `json:"updatedArtifacts,omitempty"`
	MissingArtifacts      []artifact.ArtifactID   `json:"missingArtifacts,omitempty"`
	InvalidArtifacts      []artifact.ArtifactID   `json:"invalidArtifacts,omitempty"`
	IncompatibleArtifacts []artifact.ArtifactID   `json:"incompatibleArtifacts,omitempty"`
	Diagnostics           []diagnostic.Diagnostic `json:"diagnostics,omitempty"`
	Candidates            int                     `json:"candidates"`
}

func (r RefreshSourceResult) Validate() error {
	if err := r.State.Validate(); err != nil {
		return err
	}
	if r.Candidates < 0 {
		return fmt.Errorf(
			"%w: Source refresh candidate count cannot be negative",
			basespec.ErrInvalid,
		)
	}
	if err := diagnostic.Validate(r.Diagnostics); err != nil {
		return err
	}

	created := make(map[artifact.ArtifactID]struct{})
	for _, artifactID := range r.CreatedArtifacts {
		if err := artifactID.Validate(); err != nil {
			return err
		}
		if _, duplicate := created[artifactID]; duplicate {
			return fmt.Errorf(
				"%w: Source refresh repeats created Artifact %q",
				basespec.ErrInvalid,
				artifactID,
			)
		}
		created[artifactID] = struct{}{}
	}

	updated := make(map[artifact.ArtifactID]struct{})
	for _, artifactID := range r.UpdatedArtifacts {
		if err := artifactID.Validate(); err != nil {
			return err
		}
		if _, duplicate := updated[artifactID]; duplicate {
			return fmt.Errorf(
				"%w: Source refresh repeats updated Artifact %q",
				basespec.ErrInvalid,
				artifactID,
			)
		}
		if _, createdNow := created[artifactID]; createdNow {
			return fmt.Errorf(
				"%w: Source refresh Artifact %q is both created and updated",
				basespec.ErrInvalid,
				artifactID,
			)
		}
		updated[artifactID] = struct{}{}
	}

	stateChanged := make(map[artifact.ArtifactID]struct{})
	for _, group := range [][]artifact.ArtifactID{
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
					basespec.ErrInvalid,
					artifactID,
				)
			}
			if _, duplicate := stateChanged[artifactID]; duplicate {
				return fmt.Errorf(
					"%w: Source refresh repeats Artifact state change %q",
					basespec.ErrInvalid,
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
		[]artifact.ArtifactID(nil),
		r.CreatedArtifacts...,
	)
	output.UpdatedArtifacts = append(
		[]artifact.ArtifactID(nil),
		r.UpdatedArtifacts...,
	)
	output.MissingArtifacts = append(
		[]artifact.ArtifactID(nil),
		r.MissingArtifacts...,
	)
	output.InvalidArtifacts = append(
		[]artifact.ArtifactID(nil),
		r.InvalidArtifacts...,
	)
	output.IncompatibleArtifacts = append(
		[]artifact.ArtifactID(nil),
		r.IncompatibleArtifacts...,
	)
	output.Diagnostics = diagnostic.Clone(r.Diagnostics)
	return output
}

type RefreshRootResult struct {
	RootID      root.RootID             `json:"rootID"`
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

	seen := make(map[source.SourceID]struct{}, len(r.Sources))
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
				basespec.ErrInvalid,
			)
		}
		if _, duplicate := seen[result.State.SourceID]; duplicate {
			return fmt.Errorf(
				"%w: Root refresh repeats Source %q",
				basespec.ErrInvalid,
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
