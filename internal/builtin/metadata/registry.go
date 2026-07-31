package metadata

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

const (
	SchemaVersion   = "v1"
	skillEntrypoint = "SKILL.md"
)

//go:embed builtin-registry.json
var registryJSON []byte

// Registry is application metadata. It is never a portable Skill package
// manifest and must never be serialized into an export payload.
type Registry struct {
	SchemaVersion string   `json:"schemaVersion"`
	Root          Root     `json:"root"`
	Source        Source   `json:"source"`
	Bundles       []Bundle `json:"bundles"`
}

type Root struct {
	ID          basespec.RootID `json:"id"`
	DisplayName string          `json:"displayName"`
	Description string          `json:"description,omitempty"`
}

type Source struct {
	ID          basespec.SourceID   `json:"id"`
	Kind        basespec.SourceKind `json:"kind"`
	DisplayName string              `json:"displayName"`
	Enabled     bool                `json:"enabled"`
}

type Bundle struct {
	ID             basespec.CollectionID   `json:"id"`
	LogicalName    basespec.LogicalName    `json:"logicalName"`
	LogicalVersion basespec.LogicalVersion `json:"logicalVersion,omitempty"`
	DisplayName    string                  `json:"displayName"`
	Description    string                  `json:"description,omitempty"`
	Enabled        bool                    `json:"enabled"`
	Skills         []Skill                 `json:"skills"`
}

type Skill struct {
	ID          basespec.ArtifactID  `json:"id"`
	LogicalName basespec.LogicalName `json:"logicalName"`
	Package     basespec.Locator     `json:"package"`
	Enabled     bool                 `json:"enabled"`
}

// SkillReference is the portable semantic built-in reference. It deliberately
// contains no Artifact Store IDs, package location, enablement, revision, or
// installation-specific metadata.
type SkillReference struct {
	Bundle basespec.LogicalName `json:"bundle"`
	Skill  basespec.LogicalName `json:"skill"`
}

func (r SkillReference) Validate() error {
	if err := basespec.ValidateLogicalName(r.Bundle); err != nil {
		return err
	}
	return basespec.ValidateLogicalName(r.Skill)
}

func LoadRegistry() (Registry, error) {
	decoder := json.NewDecoder(bytes.NewReader(registryJSON))
	decoder.DisallowUnknownFields()

	var value Registry
	if err := decoder.Decode(&value); err != nil {
		return Registry{}, fmt.Errorf("decode built-in registry: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("built-in registry contains trailing JSON values")
		}
		return Registry{}, fmt.Errorf("%w: %w", basespec.ErrInvalid, err)
	}
	if err := value.Validate(); err != nil {
		return Registry{}, err
	}
	return value, nil
}

func (r Registry) Validate() error {
	if r.SchemaVersion != SchemaVersion {
		return fmt.Errorf(
			"%w: unsupported built-in registry schema %q",
			basespec.ErrInvalid,
			r.SchemaVersion,
		)
	}
	if err := basespec.ValidateRootID(r.Root.ID); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"built-in root display name",
		r.Root.DisplayName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateOptionalText(
		"built-in root description",
		r.Root.Description,
		basespec.MaxDescriptionBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateSourceID(r.Source.ID); err != nil {
		return err
	}
	if err := basespec.ValidateSourceKind(r.Source.Kind); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"built-in Source display name",
		r.Source.DisplayName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if len(r.Bundles) == 0 {
		return fmt.Errorf(
			"%w: built-in registry has no logical bundle groups",
			basespec.ErrInvalid,
		)
	}
	if r.Root.ID == basespec.RootID(r.Source.ID) {
		return fmt.Errorf(
			"%w: built-in Root and Source IDs must differ",
			basespec.ErrConflict,
		)
	}

	bundleNames := make(map[basespec.LogicalName]struct{}, len(r.Bundles))
	bundleIDs := make(map[basespec.CollectionID]struct{}, len(r.Bundles))
	artifactIDs := make(map[basespec.ArtifactID]struct{})
	allIDs := map[string]struct{}{
		string(r.Root.ID):   {},
		string(r.Source.ID): {},
	}

	for bundleIndex, bundle := range r.Bundles {
		if err := basespec.ValidateCollectionID(bundle.ID); err != nil {
			return fmt.Errorf("bundles[%d]: %w", bundleIndex, err)
		}
		if err := basespec.ValidateLogicalName(bundle.LogicalName); err != nil {
			return fmt.Errorf("bundles[%d]: %w", bundleIndex, err)
		}
		if err := basespec.ValidateLogicalVersion(bundle.LogicalVersion, true); err != nil {
			return fmt.Errorf("bundles[%d]: %w", bundleIndex, err)
		}
		if err := basespec.ValidateRequiredText(
			"built-in bundle display name",
			bundle.DisplayName,
			basespec.MaxDisplayNameBytes,
		); err != nil {
			return fmt.Errorf("bundles[%d]: %w", bundleIndex, err)
		}
		if err := basespec.ValidateOptionalText(
			"built-in bundle description",
			bundle.Description,
			basespec.MaxDescriptionBytes,
		); err != nil {
			return fmt.Errorf("bundles[%d]: %w", bundleIndex, err)
		}
		if _, duplicate := bundleNames[bundle.LogicalName]; duplicate {
			return fmt.Errorf(
				"%w: duplicate built-in bundle %q",
				basespec.ErrConflict,
				bundle.LogicalName,
			)
		}
		if _, duplicate := bundleIDs[bundle.ID]; duplicate {
			return fmt.Errorf(
				"%w: duplicate built-in Collection ID %q",
				basespec.ErrConflict,
				bundle.ID,
			)
		}
		if _, duplicate := allIDs[string(bundle.ID)]; duplicate {
			return fmt.Errorf(
				"%w: duplicate built-in ID %q",
				basespec.ErrConflict,
				bundle.ID,
			)
		}
		bundleNames[bundle.LogicalName] = struct{}{}
		bundleIDs[bundle.ID] = struct{}{}
		allIDs[string(bundle.ID)] = struct{}{}

		skillNames := make(map[basespec.LogicalName]struct{}, len(bundle.Skills))
		for skillIndex, skill := range bundle.Skills {
			if err := basespec.ValidateArtifactID(skill.ID); err != nil {
				return fmt.Errorf(
					"bundles[%d].skills[%d]: %w",
					bundleIndex,
					skillIndex,
					err,
				)
			}
			if err := basespec.ValidateLogicalName(skill.LogicalName); err != nil {
				return fmt.Errorf(
					"bundles[%d].skills[%d]: %w",
					bundleIndex,
					skillIndex,
					err,
				)
			}
			if err := basespec.ValidatePortableLocator(skill.Package, false); err != nil {
				return fmt.Errorf(
					"bundles[%d].skills[%d]: %w",
					bundleIndex,
					skillIndex,
					err,
				)
			}
			if _, duplicate := skillNames[skill.LogicalName]; duplicate {
				return fmt.Errorf(
					"%w: duplicate built-in Skill %q",
					basespec.ErrConflict,
					skill.LogicalName,
				)
			}
			if _, duplicate := artifactIDs[skill.ID]; duplicate {
				return fmt.Errorf(
					"%w: duplicate built-in Artifact ID %q",
					basespec.ErrConflict,
					skill.ID,
				)
			}
			if _, duplicate := allIDs[string(skill.ID)]; duplicate {
				return fmt.Errorf(
					"%w: duplicate built-in ID %q",
					basespec.ErrConflict,
					skill.ID,
				)
			}
			skillNames[skill.LogicalName] = struct{}{}
			artifactIDs[skill.ID] = struct{}{}
			allIDs[string(skill.ID)] = struct{}{}
		}
	}
	return nil
}

func (r Registry) ResolveSkill(
	bundleLogicalName basespec.LogicalName,
	skillLogicalName basespec.LogicalName,
) (artifact.ArtifactRef, error) {
	for _, bundle := range r.Bundles {
		if bundle.LogicalName != bundleLogicalName {
			continue
		}
		for _, skill := range bundle.Skills {
			if skill.LogicalName == skillLogicalName {
				return artifact.ArtifactRef{
					RootID:     r.Root.ID,
					ArtifactID: skill.ID,
				}, nil
			}
		}
	}
	return artifact.ArtifactRef{}, fmt.Errorf(
		"%w: built-in Skill %q/%q",
		basespec.ErrNotFound,
		bundleLogicalName,
		skillLogicalName,
	)
}

func (r Registry) ValidatePackageLocations(packages fs.FS) error {
	for _, bundle := range r.Bundles {
		for _, skill := range bundle.Skills {
			packageInfo, err := fs.Stat(packages, string(skill.Package))
			if err != nil {
				return fmt.Errorf(
					"built-in package directory %q: %w",
					skill.Package,
					err,
				)
			}
			if !packageInfo.IsDir() {
				return fmt.Errorf(
					"%w: built-in package %q is not a directory",
					basespec.ErrInvalid,
					skill.Package,
				)
			}
			entrypoint := path.Join(string(skill.Package), skillEntrypoint)
			info, err := fs.Stat(packages, entrypoint)
			if err != nil {
				return fmt.Errorf(
					"built-in package %q: %w",
					skill.Package,
					err,
				)
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf(
					"%w: built-in package %q has no regular %s",
					basespec.ErrInvalid,
					skill.Package,
					skillEntrypoint,
				)
			}
		}
	}
	return nil
}

func (r Registry) OrderedBundles() []Bundle {
	output := append([]Bundle(nil), r.Bundles...)
	sort.Slice(output, func(left, right int) bool {
		return output[left].LogicalName < output[right].LogicalName
	})
	return output
}
