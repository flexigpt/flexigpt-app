package topology

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
)

// Declaration is application-supplied protected topology metadata.
//
// It intentionally describes only generic Artifact Store entities. It does
// not describe package bytes, collection kinds, artifact kinds, feature roles,
// or any built-in product semantics.
type Declaration struct {
	Root    root.RootDraft `json:"root"`
	Sources []source.Draft `json:"sources"`
}

// Installed is the verified local protected topology created from a
// Declaration. It is application metadata, never portable package data.
type Installed struct {
	Root    root.Root
	Sources []source.Summary
}

// Ensurer is implemented by Artifact Store composition. Feature installers
// depend on this narrow port rather than on system.Components.
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
	if err := basespec.ValidateRequiredText(
		"protected Root display name",
		d.Root.DisplayName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateOptionalText(
		"protected Root description",
		d.Root.Description,
		basespec.MaxDescriptionBytes,
	); err != nil {
		return err
	}
	if len(d.Sources) == 0 {
		return fmt.Errorf(
			"%w: protected topology requires at least one Source declaration",
			basespec.ErrInvalid,
		)
	}

	seen := make(map[source.SourceID]struct{}, len(d.Sources))
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
		if err := basespec.ValidateRequiredText(
			"protected Source display name",
			draft.DisplayName,
			basespec.MaxDisplayNameBytes,
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
				basespec.ErrConflict,
				draft.ID,
			)
		}
		seen[draft.ID] = struct{}{}
	}
	return nil
}
