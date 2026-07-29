package artifactadapter

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/workspace/attachmentdata"
	"github.com/flexigpt/flexigpt-app/internal/workspace/collectiondata"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

func EncodeArtifactData(
	value spec.ArtifactData,
) (json.RawMessage, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(canonical), nil
}

func DecodeArtifactData(
	raw json.RawMessage,
) (spec.ArtifactData, error) {
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return spec.ArtifactData{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()
	var value spec.ArtifactData
	if err := decoder.Decode(&value); err != nil {
		return spec.ArtifactData{}, fmt.Errorf(
			"%w: decode Workspace artifact data: %w",
			spec.ErrInvalidWorkspace,
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

func validateWorkspaceState(
	value collection.Collection,
	data spec.CollectionData,
	attachments []collection.Attachment,
	sources []source.Summary,
) (spec.Mode, basespec.SourceID, error) {
	if err := value.Validate(); err != nil {
		return "", "", fmt.Errorf("%w: invalid Workspace collection: %w", spec.ErrInvalidWorkspace, err)
	}
	if value.Kind != spec.CollectionKind {
		return "", "", fmt.Errorf(
			"%w: collection %q has kind %q",
			spec.ErrNotWorkspace,
			value.ID,
			value.Kind,
		)
	}
	if err := collectiondata.ValidateCollectionData(data); err != nil {
		return "", "", err
	}
	sourcesByID := make(map[basespec.SourceID]source.Summary, len(sources))

	for _, sourceValue := range sources {
		if err := sourceValue.Validate(); err != nil {
			return "", "", fmt.Errorf(
				"%w: invalid Workspace source summary: %w",
				spec.ErrInvalidWorkspace,
				err,
			)
		}
		if _, duplicate := sourcesByID[sourceValue.ID]; duplicate {
			return "", "", fmt.Errorf(
				"%w: duplicate Workspace source summary %q",
				spec.ErrInvalidWorkspace,
				sourceValue.ID,
			)
		}
		if sourceValue.RootID != value.RootID {
			return "", "", fmt.Errorf(
				"%w: Workspace source %q belongs to another Root",
				spec.ErrInvalidWorkspace,
				sourceValue.ID,
			)
		}
		sourcesByID[sourceValue.ID] = sourceValue
	}

	primaryCount := 0
	var primarySourceID basespec.SourceID
	seenAttachments := make(map[basespec.SourceID]struct{}, len(attachments))
	for _, attachment := range attachments {
		if err := attachment.Validate(); err != nil {
			return "", "", fmt.Errorf(
				"%w: invalid Workspace attachment: %w",
				spec.ErrInvalidWorkspace,
				err,
			)
		}

		if _, duplicate := seenAttachments[attachment.SourceID]; duplicate {
			return "", "", fmt.Errorf(
				"%w: duplicate Workspace attachment source %q",
				spec.ErrInvalidWorkspace,
				attachment.SourceID,
			)
		}
		seenAttachments[attachment.SourceID] = struct{}{}
		if attachment.RootID != value.RootID || attachment.CollectionID != value.ID {
			return "", "", fmt.Errorf("%w: attachment belongs to another collection", spec.ErrInvalidWorkspace)
		}
		operation, supported := attachmentdata.AttachmentOperationFor(attachment.Role)
		if !supported {
			return "", "", fmt.Errorf(

				"%w: unsupported attachment role %q",
				spec.ErrInvalidWorkspace,
				attachment.Role,
			)
		}

		att, err := attachmentdata.DecodeAttachmentData(attachment.Data)
		if err != nil {
			return "", "", fmt.Errorf(
				"%w: invalid attachment data for source %q: %w",
				spec.ErrInvalidWorkspace,
				attachment.SourceID,
				err,
			)
		}
		if err := attachmentdata.ValidateAttachmentDataForRole(attachment.Role, att); err != nil {
			return "", "", err
		}

		sourceValue, exists := sourcesByID[attachment.SourceID]
		if !exists {
			return "", "", fmt.Errorf(

				"%w: attachment source %q is unavailable",
				spec.ErrInvalidWorkspace,
				attachment.SourceID,
			)
		}
		if attachment.Enabled && !sourceValue.Enabled {
			return "", "", fmt.Errorf(
				"%w: enabled Workspace attachment %q uses a disabled Source",
				spec.ErrInvalidWorkspace,
				attachment.SourceID,
			)
		}

		if operation.IsPrimary {
			primaryCount++
			primarySourceID = attachment.SourceID

			if !attachment.Enabled || !sourceValue.Enabled {
				return "", "", fmt.Errorf(
					"%w: primary source and attachment must be enabled",
					spec.ErrInvalidWorkspace,
				)
			}
			if sourceValue.Kind != operation.RequiredSourceKind {
				return "", "", fmt.Errorf(
					"%w: primary source must be a filesystem source",
					spec.ErrInvalidWorkspace,
				)
			}
		}
	}
	switch primaryCount {
	case 0:
		return spec.ModeEmpty, "", nil
	case 1:
		return spec.ModeFilesystem, primarySourceID, nil
	default:
		return "", "", fmt.Errorf(
			"%w: Workspace cannot have multiple primary attachments",
			spec.ErrInvalidWorkspace,
		)
	}
}

func validateRole(role basespec.AttachmentRole) error {
	if _, supported := attachmentdata.AttachmentOperationFor(role); supported {
		return nil
	}
	return fmt.Errorf(
		"%w: unsupported attachment role %q",
		spec.ErrInvalidWorkspace,
		role,
	)
}
