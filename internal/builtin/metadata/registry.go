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
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	skillArtifact "github.com/flexigpt/flexigpt-app/internal/skill/artifact"
	skillBundle "github.com/flexigpt/flexigpt-app/internal/skill/bundle"
)

const (
	SchemaVersion        = "v1"
	collectionEntrypoint = "collection.json"
	skillEntrypoint      = "SKILL.md"
)

//go:embed builtin-registry.json
var registryJSON []byte

type HydratedRegistry struct {
	Registry    Registry
	Collections []HydratedCollection
}

type HydratedCollection struct {
	Registration          Collection
	Definition            definition.CollectionDefinition
	SourceScope           basespec.Locator
	ExpectedMemberDigests map[basespec.Locator]cryptoutil.Digest
	Artifacts             []HydratedArtifact
}

type HydratedArtifact struct {
	Registration      Artifact
	Member            definition.ContentRef
	SkillDefinition   definition.Definition
	EmbeddedDirectory basespec.Locator
	SourceDirectory   basespec.Locator
}

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

func (r Registry) OrderedCollections() []Collection {
	output := append([]Collection(nil), r.Collections...)
	sort.Slice(output, func(left, right int) bool {
		return output[left].Payload < output[right].Payload
	})
	return output
}

// Hydrate converts embedded portable Collection source descriptors into
// canonical, integrity-pinned Collection Definitions. The checked-in source
// descriptor may omit digest fields; this function computes them exclusively
// from the embedded package bytes before any local installation occurs.
func (r Registry) Hydrate(packages fs.FS) (HydratedRegistry, error) {
	if packages == nil {
		return HydratedRegistry{}, fmt.Errorf(
			"%w: built-in package filesystem is nil",
			basespec.ErrInvalid,
		)
	}
	if err := r.Validate(); err != nil {
		return HydratedRegistry{}, err
	}

	output := HydratedRegistry{
		Registry:    r,
		Collections: make([]HydratedCollection, 0, len(r.Collections)),
	}
	for index, registration := range r.OrderedCollections() {
		value, err := hydrateCollection(packages, registration)
		if err != nil {
			return HydratedRegistry{}, fmt.Errorf(
				"collections[%d]: %w",
				index,
				err,
			)
		}
		output.Collections = append(output.Collections, value)
	}
	return output, nil
}

func (r HydratedRegistry) OrderedCollections() []HydratedCollection {
	output := append([]HydratedCollection(nil), r.Collections...)
	sort.Slice(output, func(left, right int) bool {
		return output[left].Definition.LogicalName <
			output[right].Definition.LogicalName
	})
	return output
}

func hydrateCollection(
	packages fs.FS,
	registration Collection,
) (HydratedCollection, error) {
	raw, err := fs.ReadFile(packages, string(registration.Payload))
	if err != nil {
		return HydratedCollection{}, fmt.Errorf(
			"read portable Collection payload %q: %w",
			registration.Payload,
			err,
		)
	}
	payload, err := decodeCollectionPayload(raw)
	if err != nil {
		return HydratedCollection{}, err
	}

	scope := basespec.Locator(path.Dir(string(registration.Payload)))
	if err := basespec.ValidatePortableLocator(scope, false); err != nil {
		return HydratedCollection{}, err
	}

	memberContents := make(
		map[basespec.Locator][]byte,
		len(payload.Members),
	)
	for index := range payload.Members {
		member := payload.Members[index]
		if err := member.Validate(); err != nil {
			return HydratedCollection{}, fmt.Errorf(
				"payload member %d: %w",
				index,
				err,
			)
		}
		if member.Locator == "" ||
			member.URI != "" ||
			member.SubresourceLocator != "" {
			return HydratedCollection{}, fmt.Errorf(
				"%w: built-in Collection member %d must use a local relative locator",
				basespec.ErrInvalid,
				index,
			)
		}

		sourceLocator, err := scopedLocator(scope, member.Locator)
		if err != nil {
			return HydratedCollection{}, err
		}
		content, err := fs.ReadFile(packages, string(sourceLocator))
		if err != nil {
			return HydratedCollection{}, fmt.Errorf(
				"read Collection member %q: %w",
				sourceLocator,
				err,
			)
		}
		if len(content) > basespec.MaxCandidateBytes {
			return HydratedCollection{}, fmt.Errorf(
				"%w: Collection member %q exceeds the candidate byte limit",
				basespec.ErrInvalid,
				member.Locator,
			)
		}

		actualDigest := cryptoutil.DigestBytes(content)
		if member.Digest != nil && *member.Digest != actualDigest {
			return HydratedCollection{}, fmt.Errorf(
				"%w: member %q supplied %q, calculated %q",
				basespec.ErrDigestMismatch,
				member.Locator,
				*member.Digest,
				actualDigest,
			)
		}
		member.Digest = &actualDigest
		payload.Members[index] = member
		memberContents[member.Locator] = append([]byte(nil), content...)
	}

	canonical, err := skillBundle.CanonicalizePortableBundleDefinition(payload)
	if err != nil {
		return HydratedCollection{}, err
	}

	registered := make(
		map[basespec.Locator]Artifact,
		len(registration.Artifacts),
	)
	for _, value := range registration.Artifacts {
		registered[value.Member] = value
	}
	if len(registered) != len(canonical.Members) {
		return HydratedCollection{}, fmt.Errorf(
			"%w: payload has %d members but registry has %d Artifact registrations",
			basespec.ErrConflict,
			len(canonical.Members),
			len(registered),
		)
	}

	output := HydratedCollection{
		Registration:          registration,
		Definition:            canonical,
		SourceScope:           scope,
		ExpectedMemberDigests: make(map[basespec.Locator]cryptoutil.Digest),
		Artifacts:             make([]HydratedArtifact, 0, len(canonical.Members)),
	}
	for _, member := range canonical.Members {
		registrationValue, found := registered[member.Locator]
		if !found {
			return HydratedCollection{}, fmt.Errorf(
				"%w: payload member %q has no static Artifact registration",
				basespec.ErrReferenceUnresolved,
				member.Locator,
			)
		}
		if member.Digest == nil {
			return HydratedCollection{}, fmt.Errorf(
				"%w: hydrated member %q has no digest",
				basespec.ErrInvalid,
				member.Locator,
			)
		}

		expectedName := path.Base(path.Dir(string(member.Locator)))
		skillDefinition, _, err := skillArtifact.DecodeSkillDocument(
			memberContents[member.Locator],
			expectedName,
		)
		if err != nil {
			return HydratedCollection{}, fmt.Errorf(
				"decode member %q: %w",
				member.Locator,
				err,
			)
		}

		embeddedDirectory, err := scopedLocator(
			scope,
			basespec.Locator(path.Dir(string(member.Locator))),
		)
		if err != nil {
			return HydratedCollection{}, err
		}
		info, err := fs.Stat(packages, string(embeddedDirectory))
		if err != nil {
			return HydratedCollection{}, err
		}
		if !info.IsDir() {
			return HydratedCollection{}, fmt.Errorf(
				"%w: member package %q is not a directory",
				basespec.ErrInvalid,
				embeddedDirectory,
			)
		}

		output.ExpectedMemberDigests[member.Locator] = *member.Digest
		output.Artifacts = append(output.Artifacts, HydratedArtifact{
			Registration:      registrationValue,
			Member:            member,
			SkillDefinition:   skillDefinition,
			EmbeddedDirectory: embeddedDirectory,
			SourceDirectory:   embeddedDirectory,
		})
	}
	return output, nil
}

func decodeCollectionPayload(
	raw []byte,
) (definition.CollectionDefinition, error) {
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxDefinitionBytes,
	)
	if err != nil {
		return definition.CollectionDefinition{}, err
	}

	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()

	var value definition.CollectionDefinition
	if err := decoder.Decode(&value); err != nil {
		return definition.CollectionDefinition{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("portable Collection payload contains trailing JSON values")
		}
		return definition.CollectionDefinition{}, err
	}
	return value, nil
}

func scopedLocator(
	scope basespec.Locator,
	member basespec.Locator,
) (basespec.Locator, error) {
	value := basespec.Locator(path.Join(string(scope), string(member)))
	if err := basespec.ValidatePortableLocator(value, false); err != nil {
		return "", err
	}
	return value, nil
}
