package schemaadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	skillArtifact "github.com/flexigpt/flexigpt-app/internal/skill/store/artifact"
)

type Artifact struct {
	ID      basespec.ArtifactID `json:"id"`
	Member  basespec.Locator    `json:"member"`
	Enabled bool                `json:"enabled"`
}

type Collection struct {
	ID                        basespec.CollectionID `json:"id"`
	EmbeddedCollectionLocator basespec.Locator      `json:"embeddedCollectionLocator"`
	Enabled                   bool                  `json:"enabled"`
	Artifacts                 []Artifact            `json:"artifacts"`
}

// Registry is the non-portable registration manifest for embedded Agent
// Skills. It is not a portable Skill Bundle definition and is never exported.
type Registry struct {
	SchemaVersion string       `json:"schemaVersion"`
	Collections   []Collection `json:"collections"`
}

type HydratedArtifact struct {
	Registration    Artifact
	Member          artifactbuiltin.ContentRef
	SkillDefinition definition.Definition
}

type HydratedCollection struct {
	Registration          Collection
	Definition            artifactbuiltin.SkillCollectionV1
	EmbeddedPackageRoot   basespec.Locator
	ExpectedMemberDigests map[basespec.Locator]cryptoutil.Digest
	Artifacts             []HydratedArtifact
}
type HydratedRegistry struct {
	Registry    Registry
	Collections []HydratedCollection
}

func LoadRegistry() (Registry, error) {
	raw, err := artifactbuiltin.ReadEmbeddedSkillRegistry()
	if err != nil {
		return Registry{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
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

// Hydrate converts embedded portable collection descriptors into canonical,
// integrity-pinned Skill collection documents. The checked-in descriptor may
// omit digest fields. Member digests are enriched from embedded package bytes
// and the final document is canonicalized through Artifact Store's registered
// shareable codec before local installation begins.
func (r Registry) Hydrate(
	ctx context.Context,
	canonicalizer providerapi.ExpectedCanonicalizer,
	packages fs.FS,
) (HydratedRegistry, error) {
	if ctx == nil {
		return HydratedRegistry{}, fmt.Errorf(
			"%w: built-in skill hydration context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return HydratedRegistry{}, err
	}
	if canonicalizer == nil {
		return HydratedRegistry{}, fmt.Errorf(
			"%w: built-in skill shareable canonicalizer is nil",
			basespec.ErrInvalid,
		)
	}
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
		value, err := hydrateCollection(
			ctx,
			canonicalizer,
			packages,
			registration,
		)
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

func (r Registry) Validate() error {
	if r.SchemaVersion != artifactbuiltin.SkillCollectionV1SchemaVersion {
		return fmt.Errorf(
			"%w: unsupported built-in Skill registry schema %q",
			basespec.ErrInvalid,
			r.SchemaVersion,
		)
	}
	if len(r.Collections) == 0 {
		return fmt.Errorf(
			"%w: built-in Skill registry has no Collection registrations",
			basespec.ErrInvalid,
		)
	}

	collectionIDs := make(map[basespec.CollectionID]struct{}, len(r.Collections))
	artifactIDs := make(map[basespec.ArtifactID]struct{})
	allIDs := make(map[string]struct{}, len(r.Collections))

	for collectionIndex, collection := range r.Collections {
		if err := basespec.ValidateCollectionID(collection.ID); err != nil {
			return fmt.Errorf("collections[%d]: %w", collectionIndex, err)
		}
		if err := basespec.ValidatePortableLocator(collection.EmbeddedCollectionLocator, false); err != nil {
			return fmt.Errorf("collections[%d]: %w", collectionIndex, err)
		}
		if path.Base(string(collection.EmbeddedCollectionLocator)) !=
			string(artifactbuiltin.SkillCollectionFileName) ||
			path.Dir(string(collection.EmbeddedCollectionLocator)) == "." {
			return fmt.Errorf(
				"%w: collections[%d] embedded collection locator must be a nested %q",
				basespec.ErrInvalid,
				collectionIndex,
				artifactbuiltin.SkillCollectionFileName,
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
			if path.Base(string(value.Member)) !=
				string(artifactbuiltin.AgentSkillDefinitionFileName) ||
				path.Dir(string(value.Member)) == "." {
				return fmt.Errorf(
					"%w: collections[%d].artifacts[%d] must reference a packaged %s",
					basespec.ErrInvalid,
					collectionIndex,
					artifactIndex,
					artifactbuiltin.AgentSkillDefinitionFileName,
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
		return output[left].EmbeddedCollectionLocator <
			output[right].EmbeddedCollectionLocator
	})
	return output
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
	ctx context.Context,
	canonicalizer providerapi.ExpectedCanonicalizer,
	packages fs.FS,
	registration Collection,
) (HydratedCollection, error) {
	raw, err := fs.ReadFile(packages, string(registration.EmbeddedCollectionLocator))
	if err != nil {
		return HydratedCollection{}, fmt.Errorf(
			"read embedded Collection document %q: %w",
			registration.EmbeddedCollectionLocator,
			err,
		)
	}
	payload, err := decodeCollectionPayload(raw)
	if err != nil {
		return HydratedCollection{}, err
	}

	scope := basespec.Locator(path.Dir(string(registration.EmbeddedCollectionLocator)))
	if err := basespec.ValidatePortableLocator(scope, false); err != nil {
		return HydratedCollection{}, err
	}

	memberContents := make(
		map[basespec.Locator][]byte,
		len(payload.Members),
	)
	for index := range payload.Members {
		if err := ctx.Err(); err != nil {
			return HydratedCollection{}, err
		}
		member := payload.Members[index]
		if member.Locator == "" ||
			member.URI != "" ||
			member.SubresourceLocator != "" {
			return HydratedCollection{}, fmt.Errorf(
				"%w: built-in Collection member %d must use a local relative locator",
				basespec.ErrInvalid,
				index,
			)
		}

		memberLocator := basespec.Locator(member.Locator)
		sourceLocator, err := scopedLocator(scope, memberLocator)
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
		if member.Digest != nil &&
			*member.Digest != string(actualDigest) {
			return HydratedCollection{}, fmt.Errorf(
				"%w: member %q supplied %q, calculated %q",
				basespec.ErrDigestMismatch,
				member.Locator,
				*member.Digest,
				actualDigest,
			)
		}
		digest := string(actualDigest)
		member.Digest = &digest
		payload.Members[index] = member
		memberContents[memberLocator] = append([]byte(nil), content...)
	}

	hydratedRaw, err := json.Marshal(payload)
	if err != nil {
		return HydratedCollection{}, err
	}
	parsed, err := canonicalizer.CanonicalizeExpected(
		ctx,
		artifactbuiltin.SkillCollectionV1SchemaKey,
		hydratedRaw,
	)
	if err != nil {
		return HydratedCollection{}, err
	}
	canonical, err := artifactbuiltin.ParseSkillCollectionV1(parsed.Raw)
	if err != nil {
		return HydratedCollection{}, err
	}
	if canonical.Digest == nil ||
		cryptoutil.Digest(*canonical.Digest) != parsed.Digest {
		return HydratedCollection{}, fmt.Errorf(
			"%w: canonical built-in collection digest does not match codec output",
			basespec.ErrDigestMismatch,
		)
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
		EmbeddedPackageRoot:   scope,
		ExpectedMemberDigests: make(map[basespec.Locator]cryptoutil.Digest),
		Artifacts:             make([]HydratedArtifact, 0, len(canonical.Members)),
	}
	for _, member := range canonical.Members {
		memberLocator := basespec.Locator(member.Locator)
		registrationValue, found := registered[memberLocator]
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
		memberDigest := cryptoutil.Digest(*member.Digest)

		expectedName := path.Base(path.Dir(member.Locator))
		skillDefinition, _, err := skillArtifact.DecodeSkillDocument(
			memberContents[memberLocator],
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
			basespec.Locator(path.Dir(member.Locator)),
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

		output.ExpectedMemberDigests[memberLocator] = memberDigest
		output.Artifacts = append(output.Artifacts, HydratedArtifact{
			Registration:    registrationValue,
			Member:          member,
			SkillDefinition: skillDefinition,
		})
	}
	return output, nil
}

func decodeCollectionPayload(
	raw []byte,
) (artifactbuiltin.SkillCollectionV1, error) {
	return artifactbuiltin.ParseSkillCollectionV1(raw)
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
