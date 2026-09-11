package artifactadapter

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/workspacecollectionv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/workspace/store/domain"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/domain/attachmentdata"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/domain/collectiondata"
)

func ArtifactRuntimeDisabled(value artifact.Artifact) (bool, error) {
	data, err := DecodeArtifactData(value.Data)
	if err != nil {
		return false, err
	}
	return data.RuntimeDisabled, nil
}

func EncodeArtifactData(
	value workspaceDomain.ArtifactData,
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
) (workspaceDomain.ArtifactData, error) {
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return workspaceDomain.ArtifactData{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()
	var value workspaceDomain.ArtifactData
	if err := decoder.Decode(&value); err != nil {
		return workspaceDomain.ArtifactData{}, fmt.Errorf(
			"%w: decode Workspace artifact data: %w",
			workspaceDomain.ErrInvalidWorkspace,
			err,
		)
	}
	return value, nil
}

// DeriveWorkspaceTopology derives Workspace mode and primary Source from
// Artifact Store values and already-decoded Workspace CollectionData.
//
// It retains Workspace-specific topology and attachment-data validation while
// skipping generic Artifact Store model validation.
func DeriveWorkspaceTopology(
	value collection.Collection,
	data workspaceDomain.CollectionData,
	attachments []collection.Attachment,
	sources []source.Summary,
) (workspaceDomain.Mode, source.SourceID, error) {
	return validateWorkspaceState(
		value,
		data,
		attachments,
		sources,
		false,
	)
}

func validateWorkspaceState(
	value collection.Collection,
	data workspaceDomain.CollectionData,
	attachments []collection.Attachment,
	sources []source.Summary,
	validateInputs bool,
) (workspaceDomain.Mode, source.SourceID, error) {
	if validateInputs {
		if err := value.Validate(); err != nil {
			return "", "", fmt.Errorf("%w: invalid Workspace collection: %w", workspaceDomain.ErrInvalidWorkspace, err)
		}
	}
	if value.Kind != workspacecollectionv1.WorkspaceCollectionKind {
		return "", "", fmt.Errorf(
			"%w: collection %q has kind %q",
			workspaceDomain.ErrNotWorkspace,
			value.ID,
			value.Kind,
		)
	}
	if validateInputs {
		if err := collectiondata.ValidateCollectionData(data); err != nil {
			return "", "", err
		}
	}
	sourcesByID := make(map[source.SourceID]source.Summary, len(sources))

	for _, sourceValue := range sources {
		if validateInputs {
			if err := sourceValue.Validate(); err != nil {
				return "", "", fmt.Errorf(
					"%w: invalid Workspace source summary: %w",
					workspaceDomain.ErrInvalidWorkspace,
					err,
				)
			}
		}
		if _, duplicate := sourcesByID[sourceValue.ID]; duplicate {
			return "", "", fmt.Errorf(
				"%w: duplicate Workspace source summary %q",
				workspaceDomain.ErrInvalidWorkspace,
				sourceValue.ID,
			)
		}
		if sourceValue.RootID != value.RootID {
			return "", "", fmt.Errorf(
				"%w: Workspace source %q belongs to another Root",
				workspaceDomain.ErrInvalidWorkspace,
				sourceValue.ID,
			)
		}
		sourcesByID[sourceValue.ID] = sourceValue
	}

	primaryCount := 0
	var primarySourceID source.SourceID
	seenAttachments := make(map[source.SourceID]struct{}, len(attachments))
	for _, attachment := range attachments {
		if validateInputs {
			if err := attachment.Validate(); err != nil {
				return "", "", fmt.Errorf(
					"%w: invalid Workspace attachment: %w",
					workspaceDomain.ErrInvalidWorkspace,
					err,
				)
			}
		}

		if _, duplicate := seenAttachments[attachment.SourceID]; duplicate {
			return "", "", fmt.Errorf(
				"%w: duplicate Workspace attachment source %q",
				workspaceDomain.ErrInvalidWorkspace,
				attachment.SourceID,
			)
		}
		seenAttachments[attachment.SourceID] = struct{}{}
		if attachment.RootID != value.RootID || attachment.CollectionID != value.ID {
			return "", "", fmt.Errorf(
				"%w: attachment belongs to another collection",
				workspaceDomain.ErrInvalidWorkspace,
			)
		}
		operation, supported := attachmentdata.AttachmentOperationFor(attachment.Role)
		if !supported {
			return "", "", fmt.Errorf(

				"%w: unsupported attachment role %q",
				workspaceDomain.ErrInvalidWorkspace,
				attachment.Role,
			)
		}

		att, err := attachmentdata.DecodeAttachmentData(attachment.Data)
		if err != nil {
			return "", "", fmt.Errorf(
				"%w: invalid attachment data for source %q: %w",
				workspaceDomain.ErrInvalidWorkspace,
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
				workspaceDomain.ErrInvalidWorkspace,
				attachment.SourceID,
			)
		}
		if attachment.Enabled && !sourceValue.Enabled {
			return "", "", fmt.Errorf(
				"%w: enabled Workspace attachment %q uses a disabled Source",
				workspaceDomain.ErrInvalidWorkspace,
				attachment.SourceID,
			)
		}

		if operation.IsPrimary {
			primaryCount++
			primarySourceID = attachment.SourceID

			if !attachment.Enabled || !sourceValue.Enabled {
				return "", "", fmt.Errorf(
					"%w: primary source and attachment must be enabled",
					workspaceDomain.ErrInvalidWorkspace,
				)
			}
			if sourceValue.Kind != operation.RequiredSourceKind {
				return "", "", fmt.Errorf(
					"%w: primary source must be a filesystem source",
					workspaceDomain.ErrInvalidWorkspace,
				)
			}
		}
	}
	switch primaryCount {
	case 0:
		return workspaceDomain.ModeEmpty, "", nil
	case 1:
		return workspaceDomain.ModeFilesystem, primarySourceID, nil
	default:
		return "", "", fmt.Errorf(
			"%w: Workspace cannot have multiple primary attachments",
			workspaceDomain.ErrInvalidWorkspace,
		)
	}
}
