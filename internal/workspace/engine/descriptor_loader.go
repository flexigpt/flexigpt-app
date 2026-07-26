package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/jsoncanon"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/portable"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

type DescriptorLoader struct {
	runtime source.Runtime
}

func NewDescriptorLoader(runtime source.Runtime) (*DescriptorLoader, error) {
	if runtime == nil {
		return nil, fmt.Errorf(
			"%w: Workspace descriptor loader runtime is nil",
			ErrInvalidWorkspace,
		)
	}
	return &DescriptorLoader{runtime: runtime}, nil
}

func (l *DescriptorLoader) Load(
	ctx context.Context,
	value Workspace,
) (observation DescriptorObservation, returnErr error) {
	if ctx == nil {
		return DescriptorObservation{}, fmt.Errorf(
			"%w: Workspace descriptor context is nil",
			ErrInvalidWorkspace,
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
	entry, err := snapshot.Stat(ctx, DescriptorLocator)

	// A missing descriptor is valid. Its source generation remains a refresh
	// precondition so a descriptor cannot appear or disappear between bootstrap
	// observation and final catalog publication.
	if errors.Is(err, artifactstore.ErrNotFound) {
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
			ErrWorkspaceDefinitionInvalid,
			err,
		)
	}
	if entry.Locator != DescriptorLocator {
		return DescriptorObservation{}, fmt.Errorf(
			"%w: Source returned %q for Workspace descriptor %q",
			ErrWorkspaceDefinitionInvalid,
			entry.Locator,
			DescriptorLocator,
		)
	}
	if !entry.IsRegular ||
		entry.SizeBytes > artifactstore.MaxDefinitionBodyBytes {
		return DescriptorObservation{}, ErrWorkspaceDefinitionInvalid
	}
	reader, err := snapshot.Open(ctx, DescriptorLocator)
	if err != nil {
		return DescriptorObservation{}, err
	}
	content, readErr := io.ReadAll(io.LimitReader(
		reader,
		artifactstore.MaxDefinitionBodyBytes+1,
	))
	closeErr := reader.Close()
	if readErr != nil {
		return DescriptorObservation{}, readErr
	}
	if closeErr != nil {
		return DescriptorObservation{}, closeErr
	}
	if len(content) > artifactstore.MaxDefinitionBodyBytes ||
		int64(len(content)) != entry.SizeBytes {
		return DescriptorObservation{}, fmt.Errorf(
			"%w: Workspace descriptor changed or exceeds its limit",
			artifactstore.ErrConflict,
		)
	}
	if err := snapshot.Confirm(ctx); err != nil {
		return DescriptorObservation{}, err
	}

	canonical, err := jsoncanon.CanonicalizeObject(
		content,
		artifactstore.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return DescriptorObservation{}, fmt.Errorf(
			"%w: canonicalize Workspace descriptor: %w",
			ErrWorkspaceDefinitionInvalid,
			err,
		)
	}

	rawDescriptor, err := decodeDescriptor(canonical)
	if err != nil {
		return DescriptorObservation{}, fmt.Errorf(
			"%w: decode Workspace descriptor: %w",
			ErrWorkspaceDefinitionInvalid,
			err,
		)
	}

	descriptor, err := portable.CanonicalizeCollectionDefinition(rawDescriptor)
	if err != nil {
		return DescriptorObservation{}, fmt.Errorf(
			"%w: %w",
			ErrWorkspaceDefinitionInvalid,
			err,
		)
	}
	if descriptor.Kind != CollectionKind {
		return DescriptorObservation{}, fmt.Errorf(
			"%w: descriptor kind must be %q",
			ErrWorkspaceDefinitionInvalid,
			CollectionKind,
		)
	}
	if descriptor.SchemaID != WorkspaceDescriptorSchemaID {
		return DescriptorObservation{}, fmt.Errorf(
			"%w: descriptor schema ID must be %q",
			ErrWorkspaceDefinitionInvalid,
			WorkspaceDescriptorSchemaID,
		)
	}
	if descriptor.SchemaVersion != WorkspaceDescriptorSchemaVersion {
		return DescriptorObservation{}, fmt.Errorf(
			"%w: descriptor schema version must be %q",
			ErrWorkspaceDefinitionInvalid,
			WorkspaceDescriptorSchemaVersion,
		)
	}

	body, err := DecodeDefinitionBody[descriptorBody](descriptor.Body)
	if err != nil {
		return DescriptorObservation{}, fmt.Errorf(
			"%w: %w",
			ErrWorkspaceDefinitionInvalid,
			err,
		)
	}

	descriptorDirectory, err := portable.DocumentBaseLocator(
		DescriptorLocator,
	)
	if err != nil {
		return DescriptorObservation{}, fmt.Errorf(
			"%w: resolve Workspace descriptor base: %w",
			ErrWorkspaceDefinitionInvalid,
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
			ErrWorkspaceDefinitionInvalid,
			err,
		)
	}
	expectedContentDigests := make(
		map[artifactstore.Locator]artifactstore.Digest,
		len(descriptor.Members),
	)
	for index, member := range descriptor.Members {
		switch {
		case member.Locator != "":
			resolvedLocator, err := portable.ResolveRelativeLocator(
				descriptorDirectory,
				member.Locator,
			)
			if err != nil {
				return DescriptorObservation{}, fmt.Errorf(
					"%w: descriptor member %d locator: %w",
					ErrWorkspaceDefinitionInvalid,
					index,
					err,
				)
			}
			if member.SubresourceLocator != "" {
				return DescriptorObservation{}, fmt.Errorf(
					"%w: descriptor member %d subresources are not supported by source discovery",
					ErrWorkspaceDefinitionInvalid,
					index,
				)
			}
			preferences.AdditionalLocators = appendUniqueLocators(
				preferences.AdditionalLocators,
				resolvedLocator,
			)
			if member.Digest != nil {
				if expected, exists := expectedContentDigests[resolvedLocator]; exists &&
					expected != *member.Digest {
					return DescriptorObservation{}, fmt.Errorf(
						"%w: descriptor members declare conflicting digests for %q",
						ErrWorkspaceDefinitionInvalid,
						resolvedLocator,
					)
				}
				expectedContentDigests[resolvedLocator] = *member.Digest
			}

		case member.URI != "":
			return DescriptorObservation{}, fmt.Errorf(
				"%w: descriptor member %d requires an external URI resolver: %w",
				ErrWorkspaceDefinitionInvalid,
				index,
				artifactstore.ErrUnsupported,
			)

		default:
			return DescriptorObservation{}, fmt.Errorf(
				"%w: descriptor member %d requires unsupported embedded content handling: %w",
				ErrWorkspaceDefinitionInvalid,
				index,
				artifactstore.ErrUnsupported,
			)
		}
	}
	if err := validateDiscoveryPreferences(preferences); err != nil {
		return DescriptorObservation{}, fmt.Errorf(
			"%w: %w",
			ErrWorkspaceDefinitionInvalid,
			err,
		)
	}
	observation.Preferences = preferences
	if len(expectedContentDigests) != 0 {
		observation.ExpectedContentDigests = expectedContentDigests
	}
	return observation, nil
}

type descriptorBody struct {
	Discovery DiscoveryPreferences `json:"discovery"`
}

func decodeDescriptor(raw []byte) (Descriptor, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()

	var descriptor Descriptor
	if err := decoder.Decode(&descriptor); err != nil {
		return Descriptor{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("descriptor contains trailing JSON values")
		}
		return Descriptor{}, err
	}
	return descriptor, nil
}

func resolveDescriptorDiscoveryPreferences(
	input DiscoveryPreferences,
	base artifactstore.Locator,
) (DiscoveryPreferences, error) {
	output := DiscoveryPreferences{
		IncludeReadme: input.IncludeReadme,
	}
	for index, locator := range input.AdditionalLocators {
		resolved, err := portable.ResolveRelativeLocator(base, locator)
		if err != nil {
			return DiscoveryPreferences{}, fmt.Errorf(
				"additionalLocators[%d]: %w",
				index,
				err,
			)
		}
		output.AdditionalLocators = append(output.AdditionalLocators, resolved)
	}
	for index, root := range input.AdditionalRoots {
		resolved, err := portable.ResolveRelativeDirectoryLocator(
			base,
			root.Root,
		)
		if err != nil {
			return DiscoveryPreferences{}, fmt.Errorf(
				"additionalRoots[%d]: %w",
				index,
				err,
			)
		}
		output.AdditionalRoots = append(output.AdditionalRoots, DiscoveryRoot{
			Root:            resolved,
			Recursive:       root.Recursive,
			IncludePatterns: append([]string(nil), root.IncludePatterns...),
		})
	}
	return output, nil
}
