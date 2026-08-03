package metadata

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/skillartifact"
	"github.com/flexigpt/flexigpt-app/internal/skillbundle"
)

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

func (r HydratedRegistry) ResolveSkill(
	collectionName basespec.LogicalName,
	skillName basespec.LogicalName,
) (artifact.ArtifactRef, error) {
	for _, collectionValue := range r.Collections {
		if collectionValue.Definition.LogicalName != collectionName {
			continue
		}
		for _, value := range collectionValue.Artifacts {
			if value.SkillDefinition.LogicalName != skillName {
				continue
			}
			return artifact.ArtifactRef{
				RootID:     r.Registry.Root.ID,
				ArtifactID: value.Registration.ID,
			}, nil
		}
	}
	return artifact.ArtifactRef{}, fmt.Errorf(
		"%w: built-in Skill %q/%q",
		basespec.ErrNotFound,
		collectionName,
		skillName,
	)
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

	canonical, err := skillbundle.CanonicalizePortableBundleDefinition(payload)
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
		skillDefinition, _, err := skillartifact.DecodeSkillDocument(
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
