package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/jsoncanon"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

func (s ArtifactSupport) Validate() error {
	if err := artifactstore.ValidateArtifactKind(s.Kind); err != nil {
		return err
	}
	if err := artifactstore.ValidateSchemaID(s.SchemaID); err != nil {
		return err
	}
	return artifactstore.ValidateDecoderID(s.DecoderID)
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

func encodeRootData(value RootData) (json.RawMessage, error) {
	if err := validateRootData(value); err != nil {
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

func decodeRootData(raw json.RawMessage) (RootData, error) {
	canonical, err := jsoncanon.CanonicalizeObject(
		raw,
		artifactstore.MaxLocalDataBytes,
	)
	if err != nil {
		return RootData{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()
	var value RootData
	if err := decoder.Decode(&value); err != nil {
		return RootData{}, err
	}
	if err := validateRootData(value); err != nil {
		return RootData{}, err
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

func validateWorkspaceState(
	r root.Root,
	data RootData,
	attachments []root.Attachment,
	sources []source.Summary,
) error {
	if r.Kind != RootKind {
		return fmt.Errorf(
			"%w: root %q has kind %q",
			ErrNotWorkspace,
			r.ID,
			r.Kind,
		)
	}
	if err := validateRootData(data); err != nil {
		return err
	}
	sourcesByID := make(map[artifactstore.SourceID]source.Summary, len(sources))
	for _, value := range sources {
		sourcesByID[value.ID] = value
	}

	primaryCount := 0
	for _, attachment := range attachments {
		operation, supported := attachmentOperationFor(attachment.Role)
		if !supported {
			return fmt.Errorf(
				"%w: unsupported attachment role %q",
				ErrInvalidWorkspace,
				attachment.Role,
			)
		}
		if _, err := decodeAttachmentData(attachment.Data); err != nil {
			return fmt.Errorf(
				"%w: invalid attachment data for source %q: %w",
				ErrInvalidWorkspace,
				attachment.SourceID,
				err,
			)
		}
		sourceValue, exists := sourcesByID[attachment.SourceID]
		if !exists {
			return fmt.Errorf(
				"%w: attachment source %q is unavailable",
				ErrInvalidWorkspace,
				attachment.SourceID,
			)
		}
		if operation.isPrimary {
			primaryCount++
			if data.Mode != ModeFilesystem {
				return fmt.Errorf(
					"%w: empty Workspace cannot have a primary attachment",
					ErrInvalidWorkspace,
				)
			}
			if attachment.SourceID != data.PrimarySourceID {
				return fmt.Errorf(
					"%w: primary attachment does not match primary source",
					ErrInvalidWorkspace,
				)
			}
			if !attachment.Enabled || !sourceValue.Enabled {
				return fmt.Errorf(
					"%w: primary source and attachment must be enabled",
					ErrInvalidWorkspace,
				)
			}
			if sourceValue.Kind != operation.requiredSourceKind {
				return fmt.Errorf(
					"%w: primary source must be a filesystem source",
					ErrInvalidWorkspace,
				)
			}
		}
	}
	switch data.Mode {
	case ModeFilesystem:
		if primaryCount != 1 {
			return fmt.Errorf(
				"%w: filesystem Workspace requires exactly one primary attachment",
				ErrInvalidWorkspace,
			)
		}
	case ModeEmpty:
		if primaryCount != 0 {
			return fmt.Errorf(
				"%w: empty Workspace cannot have a primary attachment",
				ErrInvalidWorkspace,
			)
		}
	}
	return nil
}

func validateRootData(value RootData) error {
	switch value.Mode {
	case ModeEmpty:
		if value.PrimarySourceID != "" {
			return fmt.Errorf(
				"%w: empty Workspace cannot declare a primary source",
				ErrInvalidWorkspace,
			)
		}

	case ModeFilesystem:
		if err := artifactstore.ValidateSourceID(value.PrimarySourceID); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidWorkspace, err)
		}

	default:
		return fmt.Errorf(
			"%w: unsupported Workspace mode %q",
			ErrInvalidWorkspace,
			value.Mode,
		)
	}
	if value.CapabilityProfileVersion != CapabilityProfileVersion {
		return fmt.Errorf(
			"%w: capability profile version must be %q",
			ErrInvalidWorkspace,
			CapabilityProfileVersion,
		)
	}
	if err := artifactstore.ValidateOptionalText(
		"Workspace trust reference",
		value.TrustReference,
		512,
	); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidWorkspace, err)
	}
	if err := validateDiscoveryPreferences(value.Discovery); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidWorkspace, err)
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
			if strings.TrimSpace(pattern) != pattern || pattern == "" {
				return fmt.Errorf(
					"%w: include pattern must be non-empty and trimmed",
					artifactstore.ErrInvalid,
				)
			}
			if _, duplicate := seenPatterns[pattern]; duplicate {
				return fmt.Errorf(
					"%w: duplicate include pattern %q",
					artifactstore.ErrInvalid,
					pattern,
				)
			}
			seenPatterns[pattern] = struct{}{}
			if _, err := path.Match(pattern, "candidate"); err != nil {
				return err
			}
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
