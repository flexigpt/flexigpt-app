package topology

import (
	"context"
	"fmt"

	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// Declaration is application-supplied protected topology metadata.
//
// It intentionally describes only generic Artifact Store entities. It does
// not describe package bytes, collection kinds, artifact kinds, feature roles,
// or any built-in product semantics.
type Declaration struct {
	Root    rootModel.RootDraft `json:"root"`
	Sources []sourceModel.Draft `json:"sources"`
}

// Installed is the verified local protected topology created from a
// Declaration. It is application metadata, never portable package data.
type Installed struct {
	Root    rootModel.Root
	Sources []sourceModel.Summary
}

// Ensurer is implemented by Artifact Store composition. Feature installers
// depend on this narrow port rather than on assembly.Components.
type Ensurer interface {
	EnsureProtectedTopology(
		ctx context.Context,
		declaration Declaration,
	) (Installed, error)
}

func (d Declaration) Validate() error {
	if err := d.Root.ID.Validate(); err != nil {
		return err
	}
	if err := d.Root.StorageKey.Validate(); err != nil {
		return err
	}
	if err := spec.ValidateRequiredText(
		"protected Root display name",
		d.Root.DisplayName,
		spec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := spec.ValidateOptionalText(
		"protected Root description",
		d.Root.Description,
		spec.MaxDescriptionBytes,
	); err != nil {
		return err
	}
	if len(d.Sources) == 0 {
		return fmt.Errorf(
			"%w: protected topology requires at least one Source declaration",
			spec.ErrInvalid,
		)
	}

	seen := make(map[sourceModel.SourceID]struct{}, len(d.Sources))
	for index, draft := range d.Sources {
		if err := draft.ID.Validate(); err != nil {
			return fmt.Errorf("protected Sources[%d]: %w", index, err)
		}
		if err := draft.StorageKey.Validate(); err != nil {
			return fmt.Errorf("protected Sources[%d]: %w", index, err)
		}
		if err := draft.Kind.Validate(); err != nil {
			return fmt.Errorf("protected Sources[%d]: %w", index, err)
		}
		if err := spec.ValidateRequiredText(
			"protected Source display name",
			draft.DisplayName,
			spec.MaxDisplayNameBytes,
		); err != nil {
			return fmt.Errorf("protected Sources[%d]: %w", index, err)
		}
		if err := draft.Discovery.Validate(); err != nil {
			return fmt.Errorf(
				"protected Sources[%d] discovery: %w",
				index,
				err,
			)
		}
		if _, duplicate := seen[draft.ID]; duplicate {
			return fmt.Errorf(
				"%w: duplicate protected Source %q",
				spec.ErrConflict,
				draft.ID,
			)
		}
		seen[draft.ID] = struct{}{}
	}
	return nil
}
