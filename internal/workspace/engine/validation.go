package engine

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/jsoncanon"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

func (s ArtifactSupport) Validate() error {
	if err := artifactstore.ValidateArtifactKind(s.Kind); err != nil {
		return err
	}
	if err := artifactstore.ValidateSchemaID(s.SchemaID); err != nil {
		return err
	}
	if err := artifactstore.ValidateDecoderID(s.DecoderID); err != nil {
		return err
	}
	if s.Validator == nil {
		return fmt.Errorf(
			"%w: Workspace artifact support %q has no semantic validator",
			ErrInvalidWorkspace,
			s.Kind,
		)
	}
	return nil
}

func validateDiscoveryProfiles(value DiscoveryProfiles) error {
	if err := validateDiscoveryProfile(value.Primary); err != nil {
		return err
	}
	return validateDiscoveryProfile(value.Attached)
}

func validateDiscoveryProfile(value DiscoveryProfile) error {
	roots := make([]DiscoveryRoot, 0, len(value.DirectoryRoots))
	for _, root := range value.DirectoryRoots {
		roots = append(roots, DiscoveryRoot{
			Root:            root.Root,
			Recursive:       root.Recursive,
			IncludePatterns: append([]string(nil), root.IncludePatterns...),
		})
	}
	if err := validateDiscoveryPreferences(DiscoveryPreferences{
		AdditionalLocators: append(
			[]artifactstore.Locator(nil),
			value.ExplicitLocators...,
		),
		AdditionalRoots: roots,
	}); err != nil {
		return err
	}
	if value.ReadmeLocator == "" {
		return nil
	}
	return artifactstore.ValidateLocator(value.ReadmeLocator, false)
}

func encodeCollectionData(value CollectionData) (json.RawMessage, error) {
	if err := validateCollectionData(value); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	canonical, err := jsoncanon.CanonicalizeObject(
		raw,
		artifactstore.MaxLocalDataBytes,
	)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(canonical), nil
}

func decodeCollectionData(raw json.RawMessage) (CollectionData, error) {
	canonical, err := jsoncanon.CanonicalizeObject(
		raw,
		artifactstore.MaxLocalDataBytes,
	)
	if err != nil {
		return CollectionData{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()
	var value CollectionData
	if err := decoder.Decode(&value); err != nil {
		return CollectionData{}, err
	}
	if err := validateCollectionData(value); err != nil {
		return CollectionData{}, err
	}
	return value, nil
}

func encodeAttachmentData(
	value AttachmentData,
) (json.RawMessage, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	canonical, err := jsoncanon.CanonicalizeObject(
		raw,
		artifactstore.MaxLocalDataBytes,
	)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(canonical), nil
}

func decodeAttachmentData(
	raw json.RawMessage,
) (AttachmentData, error) {
	canonical, err := jsoncanon.CanonicalizeObject(
		raw,
		artifactstore.MaxLocalDataBytes,
	)
	if err != nil {
		return AttachmentData{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()
	var value AttachmentData
	if err := decoder.Decode(&value); err != nil {
		return AttachmentData{}, err
	}
	return value, nil
}

func EncodeArtifactData(
	value ArtifactData,
) (json.RawMessage, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	canonical, err := jsoncanon.CanonicalizeObject(
		raw,
		artifactstore.MaxLocalDataBytes,
	)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(canonical), nil
}

func DecodeArtifactData(
	raw json.RawMessage,
) (ArtifactData, error) {
	canonical, err := jsoncanon.CanonicalizeObject(
		raw,
		artifactstore.MaxLocalDataBytes,
	)
	if err != nil {
		return ArtifactData{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()
	var value ArtifactData
	if err := decoder.Decode(&value); err != nil {
		return ArtifactData{}, fmt.Errorf(
			"%w: decode Workspace artifact data: %w",
			ErrInvalidWorkspace,
			err,
		)
	}
	return value, nil
}

func ArtifactRuntimeDisabled(value artifact.Artifact) (bool, error) {
	data, err := DecodeArtifactData(value.Data)
	if err != nil {
		return false, err
	}
	return data.RuntimeDisabled, nil
}

func validateAttachmentDataForRole(
	role artifactstore.AttachmentRole,
	value AttachmentData,
) error {
	operation, supported := attachmentOperationFor(role)
	if !supported {
		return fmt.Errorf(
			"%w: unsupported attachment role %q",
			ErrInvalidWorkspace,
			role,
		)
	}
	if !operation.allowsAttachmentDiscoveryOverrides &&
		(value.Recursive != nil || value.Authoritative != nil) {
		return fmt.Errorf(
			"%w: attachment role %q does not allow discovery overrides",
			ErrInvalidWorkspace,
			role,
		)
	}
	return nil
}

func validateWorkspaceState(
	value collection.Collection,
	data CollectionData,
	attachments []collection.Attachment,
	sources []source.Summary,
) (Mode, artifactstore.SourceID, error) {
	if err := value.Validate(); err != nil {
		return "", "", fmt.Errorf("%w: invalid Workspace collection: %w", ErrInvalidWorkspace, err)
	}
	if value.Kind != CollectionKind {
		return "", "", fmt.Errorf(
			"%w: collection %q has kind %q",
			ErrNotWorkspace,
			value.ID,
			value.Kind,
		)
	}
	if err := validateCollectionData(data); err != nil {
		return "", "", err
	}
	sourcesByID := make(map[artifactstore.SourceID]source.Summary, len(sources))

	for _, sourceValue := range sources {
		if err := sourceValue.Validate(); err != nil {
			return "", "", fmt.Errorf(
				"%w: invalid Workspace source summary: %w",
				ErrInvalidWorkspace,
				err,
			)
		}
		if _, duplicate := sourcesByID[sourceValue.ID]; duplicate {
			return "", "", fmt.Errorf(
				"%w: duplicate Workspace source summary %q",
				ErrInvalidWorkspace,
				sourceValue.ID,
			)
		}
		if sourceValue.RootID != value.RootID {
			return "", "", fmt.Errorf(
				"%w: Workspace source %q belongs to another Root",
				ErrInvalidWorkspace,
				sourceValue.ID,
			)
		}
		sourcesByID[sourceValue.ID] = sourceValue
	}

	primaryCount := 0
	var primarySourceID artifactstore.SourceID
	seenAttachments := make(map[artifactstore.SourceID]struct{}, len(attachments))
	for _, attachment := range attachments {
		if err := attachment.Validate(); err != nil {
			return "", "", fmt.Errorf(
				"%w: invalid Workspace attachment: %w",
				ErrInvalidWorkspace,
				err,
			)
		}

		if _, duplicate := seenAttachments[attachment.SourceID]; duplicate {
			return "", "", fmt.Errorf(
				"%w: duplicate Workspace attachment source %q",
				ErrInvalidWorkspace,
				attachment.SourceID,
			)
		}
		seenAttachments[attachment.SourceID] = struct{}{}
		if attachment.RootID != value.RootID || attachment.CollectionID != value.ID {
			return "", "", fmt.Errorf("%w: attachment belongs to another collection", ErrInvalidWorkspace)
		}
		operation, supported := attachmentOperationFor(attachment.Role)
		if !supported {
			return "", "", fmt.Errorf(

				"%w: unsupported attachment role %q",
				ErrInvalidWorkspace,
				attachment.Role,
			)
		}

		attachmentData, err := decodeAttachmentData(attachment.Data)
		if err != nil {
			return "", "", fmt.Errorf(
				"%w: invalid attachment data for source %q: %w",
				ErrInvalidWorkspace,
				attachment.SourceID,
				err,
			)
		}
		if err := validateAttachmentDataForRole(attachment.Role, attachmentData); err != nil {
			return "", "", err
		}

		sourceValue, exists := sourcesByID[attachment.SourceID]
		if !exists {
			return "", "", fmt.Errorf(

				"%w: attachment source %q is unavailable",
				ErrInvalidWorkspace,
				attachment.SourceID,
			)
		}
		if attachment.Enabled && !sourceValue.Enabled {
			return "", "", fmt.Errorf(
				"%w: enabled Workspace attachment %q uses a disabled Source",
				ErrInvalidWorkspace,
				attachment.SourceID,
			)
		}

		if operation.isPrimary {
			primaryCount++
			primarySourceID = attachment.SourceID

			if !attachment.Enabled || !sourceValue.Enabled {
				return "", "", fmt.Errorf(
					"%w: primary source and attachment must be enabled",
					ErrInvalidWorkspace,
				)
			}
			if sourceValue.Kind != operation.requiredSourceKind {
				return "", "", fmt.Errorf(
					"%w: primary source must be a filesystem source",
					ErrInvalidWorkspace,
				)
			}
		}
	}
	switch primaryCount {
	case 0:
		return ModeEmpty, "", nil
	case 1:
		return ModeFilesystem, primarySourceID, nil
	default:
		return "", "", fmt.Errorf(
			"%w: Workspace cannot have multiple primary attachments",
			ErrInvalidWorkspace,
		)
	}
}

func validateCollectionData(value CollectionData) error {
	if err := validateDiscoveryPreferences(value.Discovery); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidWorkspace, err)
	}
	if err := artifactstore.ValidateRequiredText(
		"workspace discovery policy revision",
		value.DiscoveryPolicyRevision,
		artifactstore.MaxVersionBytes,
	); err != nil {
		return fmt.Errorf(
			"%w: %w",
			ErrInvalidWorkspace,
			err,
		)
	}
	return nil
}

func validateDiscoveryPreferences(
	value DiscoveryPreferences,
) error {
	seenLocators := make(map[artifactstore.Locator]struct{})
	for _, locator := range value.AdditionalLocators {
		if err := artifactstore.ValidateLocator(locator, false); err != nil {
			return err
		}
		if _, duplicate := seenLocators[locator]; duplicate {
			return fmt.Errorf(
				"%w: duplicate discovery locator %q",
				artifactstore.ErrInvalid,
				locator,
			)
		}
		seenLocators[locator] = struct{}{}
	}

	seenRoots := make(map[artifactstore.Locator]struct{})
	for _, root := range value.AdditionalRoots {
		if err := artifactstore.ValidateLocator(root.Root, true); err != nil {
			return err
		}
		if _, duplicate := seenRoots[root.Root]; duplicate {
			return fmt.Errorf(
				"%w: duplicate discovery root %q",
				artifactstore.ErrInvalid,
				root.Root,
			)
		}
		seenRoots[root.Root] = struct{}{}
		seenPatterns := make(map[string]struct{}, len(root.IncludePatterns))
		for _, pattern := range root.IncludePatterns {
			if err := discovery.ValidateIncludePattern(pattern); err != nil {
				return err
			}
			if _, duplicate := seenPatterns[pattern]; duplicate {
				return fmt.Errorf("%w: duplicate include pattern %q", artifactstore.ErrInvalid, pattern)
			}
			seenPatterns[pattern] = struct{}{}
		}
	}
	return nil
}

func validateRole(role artifactstore.AttachmentRole) error {
	if _, supported := attachmentOperationFor(role); supported {
		return nil
	}
	return fmt.Errorf(
		"%w: unsupported attachment role %q",
		ErrInvalidWorkspace,
		role,
	)
}
