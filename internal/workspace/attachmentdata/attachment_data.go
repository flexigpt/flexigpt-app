package attachmentdata

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source/fsdir"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

// attachmentOperationMatrix is the workspace attachment lifecycle and
// discovery-operation matrix.
//
// A role must be present here before it can be attached, validated, or planned.
var attachmentOperationMatrix = [...]AttachmentOperation{
	{
		Role:                                 spec.RolePrimary,
		IsPrimary:                            true,
		RequiredSourceKind:                   fsdir.Kind,
		DefaultAuthoritative:                 true,
		IncludeReadmeWhenRequested:           true,
		AppliesWorkspaceDiscoveryPreferences: true,
	},
	{
		Role:                               spec.RoleBuiltIn,
		CanAttach:                          true,
		DefaultAuthoritative:               true,
		AllowsAttachmentDiscoveryOverrides: true,
	},
	{
		Role:                               spec.RoleLibrary,
		CanAttach:                          true,
		DefaultAuthoritative:               true,
		AllowsAttachmentDiscoveryOverrides: true,
	},
	{
		Role:                               spec.RoleAttachedPackage,
		CanAttach:                          true,
		DefaultAuthoritative:               true,
		AllowsAttachmentDiscoveryOverrides: true,
	},
	{
		Role:                               spec.RoleOverlay,
		CanAttach:                          true,
		DefaultAuthoritative:               true,
		AllowsAttachmentDiscoveryOverrides: true,
	},
}

type AttachmentOperation struct {
	Role                                 basespec.AttachmentRole
	CanAttach                            bool
	IsPrimary                            bool
	RequiredSourceKind                   basespec.SourceKind
	DefaultAuthoritative                 bool
	IncludeReadmeWhenRequested           bool
	AppliesWorkspaceDiscoveryPreferences bool
	AllowsAttachmentDiscoveryOverrides   bool
}

func ValidateAttachmentDataForRole(
	role basespec.AttachmentRole,
	value spec.AttachmentData,
) error {
	operation, supported := AttachmentOperationFor(role)
	if !supported {
		return fmt.Errorf(
			"%w: unsupported attachment role %q",
			spec.ErrInvalidWorkspace,
			role,
		)
	}
	if !operation.AllowsAttachmentDiscoveryOverrides &&
		(value.Recursive != nil || value.Authoritative != nil) {
		return fmt.Errorf(
			"%w: attachment role %q does not allow discovery overrides",
			spec.ErrInvalidWorkspace,
			role,
		)
	}
	return nil
}

func EncodeAttachmentData(
	value spec.AttachmentData,
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

func DecodeAttachmentData(
	raw json.RawMessage,
) (spec.AttachmentData, error) {
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return spec.AttachmentData{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()
	var value spec.AttachmentData
	if err := decoder.Decode(&value); err != nil {
		return spec.AttachmentData{}, err
	}
	return value, nil
}

func AttachmentOperationFor(
	role basespec.AttachmentRole,
) (AttachmentOperation, bool) {
	for _, operation := range attachmentOperationMatrix {
		if operation.Role == role {
			return operation, true
		}
	}
	return AttachmentOperation{}, false
}
