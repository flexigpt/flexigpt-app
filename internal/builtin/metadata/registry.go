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

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

const (
	SchemaVersion        = "v2"
	collectionEntrypoint = "collection.json"
	skillEntrypoint      = "SKILL.md"
)

//go:embed builtin-registry.json
var registryJSON []byte

// Registry is application metadata. It is never a portable Skill package
// manifest and must never be serialized into an export payload.
type Registry struct {
	SchemaVersion string       `json:"schemaVersion"`
	Root          Root         `json:"root"`
	Source        Source       `json:"source"`
	Collections   []Collection `json:"collections"`
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

type Collection struct {
	ID        basespec.CollectionID `json:"id"`
	Payload   basespec.Locator      `json:"payload"`
	Enabled   bool                  `json:"enabled"`
	Artifacts []Artifact            `json:"artifacts"`
}

type Artifact struct {
	ID      basespec.ArtifactID `json:"id"`
	Member  basespec.Locator    `json:"member"`
	Enabled bool                `json:"enabled"`
}

// SkillReference is a portable semantic built-in reference. It deliberately
// contains no Artifact Store IDs, package location, enablement, revision, or
// installation-specific metadata.
type SkillReference struct {
	Collection basespec.LogicalName `json:"collection"`
	Skill      basespec.LogicalName `json:"skill"`
}

func (r SkillReference) Validate() error {
	if err := basespec.ValidateLogicalName(r.Collection); err != nil {
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
	if len(r.Collections) == 0 {
		return fmt.Errorf(
			"%w: built-in registry has no Collection registrations",
			basespec.ErrInvalid,
		)
	}
	if r.Root.ID == basespec.RootID(r.Source.ID) {
		return fmt.Errorf(
			"%w: built-in Root and Source IDs must differ",
			basespec.ErrConflict,
		)
	}

	collectionIDs := make(map[basespec.CollectionID]struct{}, len(r.Collections))
	artifactIDs := make(map[basespec.ArtifactID]struct{})
	allIDs := map[string]struct{}{
		string(r.Root.ID):   {},
		string(r.Source.ID): {},
	}

	for collectionIndex, collection := range r.Collections {
		if err := basespec.ValidateCollectionID(collection.ID); err != nil {
			return fmt.Errorf("collections[%d]: %w", collectionIndex, err)
		}
		if err := basespec.ValidatePortableLocator(collection.Payload, false); err != nil {
			return fmt.Errorf("collections[%d]: %w", collectionIndex, err)
		}
		if path.Base(string(collection.Payload)) != collectionEntrypoint ||
			path.Dir(string(collection.Payload)) == "." {
			return fmt.Errorf(
				"%w: collections[%d] payload must be a nested %q",
				basespec.ErrInvalid,
				collectionIndex,
				collectionEntrypoint,
			)
		}
		if len(collection.Artifacts) == 0 {
			return fmt.Errorf(
				"%w: collections[%d] has no Artifact registrations",
				basespec.ErrInvalid,
				collectionIndex,
			)
		}
		if _, duplicate := collectionIDs[collection.ID]; duplicate {
			return fmt.Errorf(
				"%w: duplicate built-in Collection ID %q",
				basespec.ErrConflict,
				collection.ID,
			)
		}
		if _, duplicate := allIDs[string(collection.ID)]; duplicate {
			return fmt.Errorf(
				"%w: duplicate built-in ID %q",
				basespec.ErrConflict,
				collection.ID,
			)
		}
		collectionIDs[collection.ID] = struct{}{}
		allIDs[string(collection.ID)] = struct{}{}

		members := make(map[basespec.Locator]struct{}, len(collection.Artifacts))
		for artifactIndex, value := range collection.Artifacts {
			if err := basespec.ValidateArtifactID(value.ID); err != nil {
				return fmt.Errorf(
					"collections[%d].artifacts[%d]: %w",
					collectionIndex,
					artifactIndex,
					err,
				)
			}
			if err := basespec.ValidatePortableLocator(value.Member, false); err != nil {
				return fmt.Errorf(
					"collections[%d].artifacts[%d]: %w",
					collectionIndex,
					artifactIndex,
					err,
				)
			}
			if path.Base(string(value.Member)) != skillEntrypoint ||
				path.Dir(string(value.Member)) == "." {
				return fmt.Errorf(
					"%w: collections[%d].artifacts[%d] must reference a packaged %s",
					basespec.ErrInvalid,
					collectionIndex,
					artifactIndex,
					skillEntrypoint,
				)
			}
			if _, duplicate := members[value.Member]; duplicate {
				return fmt.Errorf(
					"%w: duplicate built-in Collection member %q",
					basespec.ErrConflict,
					value.Member,
				)
			}
			if _, duplicate := artifactIDs[value.ID]; duplicate {
				return fmt.Errorf(
					"%w: duplicate built-in Artifact ID %q",
					basespec.ErrConflict,
					value.ID,
				)
			}
			if _, duplicate := allIDs[string(value.ID)]; duplicate {
				return fmt.Errorf(
					"%w: duplicate built-in ID %q",
					basespec.ErrConflict,
					value.ID,
				)
			}
			members[value.Member] = struct{}{}
			artifactIDs[value.ID] = struct{}{}
			allIDs[string(value.ID)] = struct{}{}
		}
	}
	return nil
}

func (r Registry) ValidatePackageLocations(packages fs.FS) error {
	_, err := r.Hydrate(packages)
	return err
}

func (r Registry) OrderedCollections() []Collection {
	output := append([]Collection(nil), r.Collections...)
	sort.Slice(output, func(left, right int) bool {
		return output[left].Payload < output[right].Payload
	})
	return output
}
