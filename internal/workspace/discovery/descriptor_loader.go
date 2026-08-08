package discovery

import (
	"context"
	"errors"
	"fmt"
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/shareable"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	builtinSchema "github.com/flexigpt/flexigpt-app/internal/builtin/schema"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

type DescriptorLoader struct {
	runtime       source.Runtime
	canonicalizer shareable.Canonicalizer
}

const descriptorLocator basespec.Locator = spec.WorkspaceMetadataDirectory + "/" + spec.WorkspaceDescriptorFileName

// Descriptor is the portable workspace.collection document stored at
// .flexigpt/workspace.json.
type Descriptor = builtinSchema.WorkspaceCollectionV1

type DescriptorObservation struct {
	Preferences            spec.DiscoveryPreferences
	SourceID               basespec.SourceID
	Generation             string
	ExpectedContentDigests map[basespec.Locator]cryptoutil.Digest
}

func NewDescriptorLoader(
	runtime source.Runtime,
	canonicalizer shareable.Canonicalizer,
) (*DescriptorLoader, error) {
	if runtime == nil || canonicalizer == nil {
		return nil, fmt.Errorf(
			"%w: Workspace descriptor loader dependencies are incomplete",
			spec.ErrInvalidWorkspace,
		)
	}
	return &DescriptorLoader{
		runtime:       runtime,
		canonicalizer: canonicalizer,
	}, nil
}

func (l *DescriptorLoader) Load(
	ctx context.Context,
	value spec.Workspace,
) (observation DescriptorObservation, returnErr error) {
	if ctx == nil {
		return DescriptorObservation{}, fmt.Errorf(
			"%w: Workspace descriptor context is nil",
			spec.ErrInvalidWorkspace,
		)
	}
	if err := ctx.Err(); err != nil {
		return DescriptorObservation{}, err
	}
	if value.PrimarySourceID == "" {
		return DescriptorObservation{}, nil
	}
	sourceValue, err := l.runtime.Get(
		ctx,
		value.Collection.RootID,
		value.PrimarySourceID,
	)
	if err != nil {
		return DescriptorObservation{}, err
	}
	snapshot, err := l.runtime.Open(ctx, sourceValue)
	if err != nil {
		return DescriptorObservation{}, err
	}
	defer func() {
		returnErr = errors.Join(returnErr, snapshot.Close())
	}()

	observation = DescriptorObservation{
		SourceID:   sourceValue.ID,
		Generation: snapshot.Generation(),
	}
	entry, err := snapshot.Stat(ctx, descriptorLocator)

	// A missing descriptor is valid. Its source generation remains a refresh
	// precondition so a descriptor cannot appear or disappear between bootstrap
	// observation and final catalog publication.
	if errors.Is(err, basespec.ErrNotFound) {
		if err := snapshot.Confirm(ctx); err != nil {
			return DescriptorObservation{}, err
		}
		return observation, nil
	}
	if err != nil {
		return DescriptorObservation{}, err
	}
	if err := entry.Validate(); err != nil {
		return DescriptorObservation{}, fmt.Errorf(
			"%w: Source returned an invalid Workspace descriptor entry: %w",
			spec.ErrWorkspaceDefinitionInvalid,
			err,
		)
	}
	if entry.Locator != descriptorLocator {
		return DescriptorObservation{}, fmt.Errorf(
			"%w: Source returned %q for Workspace descriptor %q",
			spec.ErrWorkspaceDefinitionInvalid,
			entry.Locator,
			spec.DescriptorLocator,
		)
	}
	if !entry.IsRegular ||
		entry.SizeBytes > basespec.MaxDefinitionBytes {
		return DescriptorObservation{}, spec.ErrWorkspaceDefinitionInvalid
	}
	content, err := source.ReadSnapshotEntry(
		ctx,
		snapshot,
		entry,
		int64(basespec.MaxDefinitionBytes),
	)
	if err != nil {
		return DescriptorObservation{}, err
	}
	if err := snapshot.Confirm(ctx); err != nil {
		return DescriptorObservation{}, err
	}

	document, err := l.canonicalizer.Canonicalize(ctx, content)
	if err != nil {
		return DescriptorObservation{}, fmt.Errorf(
			"%w: canonicalize Workspace descriptor: %w",
			spec.ErrWorkspaceDefinitionInvalid,
			err,
		)
	}
	expectedKey := shareable.SchemaKey{
		Entity:        shareable.EntityCollection,
		Kind:          spec.CollectionKind,
		SchemaID:      spec.WorkspaceDescriptorSchemaID,
		SchemaVersion: spec.WorkspaceDescriptorSchemaVersion,
	}
	if document.Key != expectedKey {
		return DescriptorObservation{}, fmt.Errorf(
			"%w: descriptor schema does not identify %q",
			spec.ErrWorkspaceDefinitionInvalid,
			spec.CollectionKind,
		)
	}

	descriptor, err := builtinSchema.ParseWorkspaceCollectionV1(document.Raw)
	if err != nil {
		return DescriptorObservation{}, fmt.Errorf(
			"%w: decode canonical Workspace descriptor: %w",
			spec.ErrWorkspaceDefinitionInvalid,
			err,
		)
	}

	body, err := builtinSchema.DecodeWorkspaceCollectionV1Body(
		descriptor.Body,
	)
	if err != nil {
		return DescriptorObservation{}, fmt.Errorf(
			"%w: %w",
			spec.ErrWorkspaceDefinitionInvalid,
			err,
		)
	}

	descriptorDirectory, err := documentBaseLocator(
		spec.DescriptorLocator,
	)
	if err != nil {
		return DescriptorObservation{}, fmt.Errorf(
			"%w: resolve Workspace descriptor base: %w",
			spec.ErrWorkspaceDefinitionInvalid,
			err,
		)
	}
	preferences, err := resolveDescriptorDiscoveryPreferences(
		body.Discovery,
		descriptorDirectory,
	)
	if err != nil {
		return DescriptorObservation{}, fmt.Errorf(
			"%w: resolve Workspace descriptor discovery preferences: %w",
			spec.ErrWorkspaceDefinitionInvalid,
			err,
		)
	}
	expectedContentDigests := make(
		map[basespec.Locator]cryptoutil.Digest,
		len(descriptor.Members),
	)
	for index, member := range descriptor.Members {
		switch {
		case member.Locator != "":
			memberLocator := basespec.Locator(member.Locator)
			resolvedLocator, err := resolveRelativeLocator(
				descriptorDirectory,
				memberLocator,
				false,
			)
			if err != nil {
				return DescriptorObservation{}, fmt.Errorf(
					"%w: descriptor member %d locator: %w",
					spec.ErrWorkspaceDefinitionInvalid,
					index,
					err,
				)
			}
			if member.SubresourceLocator != "" {
				return DescriptorObservation{}, fmt.Errorf(
					"%w: descriptor member %d subresources are not supported by source discovery",
					spec.ErrWorkspaceDefinitionInvalid,
					index,
				)
			}
			preferences.AdditionalLocators = appendUniqueLocators(
				preferences.AdditionalLocators,
				resolvedLocator,
			)
			if member.Digest != nil {
				memberDigest := cryptoutil.Digest(*member.Digest)
				if existing, exists := expectedContentDigests[resolvedLocator]; exists &&
					existing != memberDigest {
					return DescriptorObservation{}, fmt.Errorf(
						"%w: descriptor members declare conflicting digests for %q",
						spec.ErrWorkspaceDefinitionInvalid,
						resolvedLocator,
					)
				}
				expectedContentDigests[resolvedLocator] = memberDigest
			}

		case member.URI != "":
			return DescriptorObservation{}, fmt.Errorf(
				"%w: descriptor member %d requires an external URI resolver: %w",
				spec.ErrWorkspaceDefinitionInvalid,
				index,
				basespec.ErrUnsupported,
			)

		default:
			return DescriptorObservation{}, fmt.Errorf(
				"%w: descriptor member %d requires unsupported embedded content handling: %w",
				spec.ErrWorkspaceDefinitionInvalid,
				index,
				basespec.ErrUnsupported,
			)
		}
	}
	if err := spec.ValidateDiscoveryPreferences(preferences); err != nil {
		return DescriptorObservation{}, fmt.Errorf(
			"%w: %w",
			spec.ErrWorkspaceDefinitionInvalid,
			err,
		)
	}
	observation.Preferences = preferences
	if len(expectedContentDigests) != 0 {
		observation.ExpectedContentDigests = expectedContentDigests
	}
	return observation, nil
}

func resolveDescriptorDiscoveryPreferences(
	input builtinSchema.WorkspaceDiscoveryV1,
	base basespec.Locator,
) (spec.DiscoveryPreferences, error) {
	output := spec.DiscoveryPreferences{
		IncludeReadme: input.IncludeReadme,
	}
	for index, locator := range input.AdditionalLocators {
		resolved, err := resolveRelativeLocator(
			base,
			basespec.Locator(locator),
			false,
		)
		if err != nil {
			return spec.DiscoveryPreferences{}, fmt.Errorf(
				"additionalLocators[%d]: %w",
				index,
				err,
			)
		}
		output.AdditionalLocators = append(output.AdditionalLocators, resolved)
	}
	for index, root := range input.AdditionalRoots {
		resolved, err := resolveRelativeLocator(
			base,
			basespec.Locator(root.Root),
			true,
		)
		if err != nil {
			return spec.DiscoveryPreferences{}, fmt.Errorf(
				"additionalRoots[%d]: %w",
				index,
				err,
			)
		}
		output.AdditionalRoots = append(output.AdditionalRoots, spec.DiscoveryRoot{
			Root:            resolved,
			Recursive:       root.Recursive,
			IncludePatterns: append([]string(nil), root.IncludePatterns...),
		})
	}
	return output, nil
}

// documentBaseLocator returns the source-relative directory containing a
// portable document. A document in the source root has "." as its base.
func documentBaseLocator(
	document basespec.Locator,
) (basespec.Locator, error) {
	if err := basespec.ValidateLocator(document, false); err != nil {
		return "", fmt.Errorf("portable document locator: %w", err)
	}

	base := basespec.Locator(path.Dir(string(document)))
	if err := basespec.ValidateLocator(base, true); err != nil {
		return "", fmt.Errorf("portable document base locator: %w", err)
	}
	return base, nil
}

func resolveRelativeLocator(
	base basespec.Locator,
	relative basespec.Locator,
	allowRelativeRoot bool,
) (basespec.Locator, error) {
	if err := basespec.ValidatePortableLocator(base, true); err != nil {
		return "", fmt.Errorf("portable base locator: %w", err)
	}
	if err := basespec.ValidatePortableLocator(relative, allowRelativeRoot); err != nil {
		return "", fmt.Errorf("portable relative locator: %w", err)
	}

	var resolved basespec.Locator
	switch {
	case base == ".":
		resolved = relative
	case relative == ".":
		resolved = base
	default:
		resolved = basespec.Locator(
			path.Join(string(base), string(relative)),
		)
	}

	if err := basespec.ValidatePortableLocator(resolved, allowRelativeRoot); err != nil {
		return "", fmt.Errorf("resolved portable locator: %w", err)
	}
	return resolved, nil
}
