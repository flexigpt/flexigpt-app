package bundle

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/mcp/schema"
)

const (
	CollectionDataSchemaVersion = "v1"
	AttachmentDataSchemaVersion = "v1"

	DiscoveryPolicyRevision = "mcp.bundle.discovery.v1"

	RoleManaged basespec.AttachmentRole = "managed"
	RoleBuiltIn basespec.AttachmentRole = "builtin"

	PackageDirectory       basespec.Locator = "package"
	DefaultDocumentLocator basespec.Locator = "package/mcps.json"
)

type CollectionData struct {
	SchemaVersion           string                  `json:"schemaVersion"`
	DiscoveryPolicyRevision string                  `json:"discoveryPolicyRevision"`
	LogicalName             basespec.LogicalName    `json:"logicalName"`
	LogicalVersion          basespec.LogicalVersion `json:"logicalVersion,omitempty"`
	Labels                  map[string]string       `json:"labels,omitempty"`
	ManagedSourceID         basespec.SourceID       `json:"managedSourceID,omitempty"`
}

type AttachmentData struct {
	SchemaVersion   string           `json:"schemaVersion"`
	DocumentLocator basespec.Locator `json:"documentLocator"`
}

func DecodeCollectionData(
	raw json.RawMessage,
) (CollectionData, error) {
	value, err := decodeStrict[CollectionData](raw)
	if err != nil {
		return CollectionData{}, err
	}
	if _, err := EncodeCollectionData(value); err != nil {
		return CollectionData{}, err
	}
	return value, nil
}

func EncodeCollectionData(
	value CollectionData,
) (json.RawMessage, error) {
	if value.SchemaVersion != CollectionDataSchemaVersion ||
		value.DiscoveryPolicyRevision != DiscoveryPolicyRevision {
		return nil, fmt.Errorf(
			"%w: invalid MCP Bundle Collection data",
			basespec.ErrInvalid,
		)
	}
	if err := basespec.ValidateLogicalName(value.LogicalName); err != nil {
		return nil, err
	}
	if value.ManagedSourceID != "" {
		if err := basespec.ValidateSourceID(value.ManagedSourceID); err != nil {
			return nil, err
		}
	}
	return encodeStrict(value)
}

func DecodeAttachmentData(
	raw json.RawMessage,
) (AttachmentData, error) {
	value, err := decodeStrict[AttachmentData](raw)
	if err != nil {
		return AttachmentData{}, err
	}
	if _, err := EncodeAttachmentData(value); err != nil {
		return AttachmentData{}, err
	}
	return value, nil
}

func EncodeAttachmentData(
	value AttachmentData,
) (json.RawMessage, error) {
	if value.SchemaVersion != AttachmentDataSchemaVersion {
		return nil, fmt.Errorf(
			"%w: invalid MCP Bundle Attachment data",
			basespec.ErrInvalid,
		)
	}
	if err := ValidateDocumentLocator(value.DocumentLocator); err != nil {
		return nil, err
	}
	return encodeStrict(value)
}

func packageDirectoryForDocument(
	value basespec.Locator,
) (basespec.Locator, error) {
	if err := ValidateDocumentLocator(value); err != nil {
		return "", err
	}
	return basespec.Locator(path.Dir(string(value))), nil
}

// ValidateDocumentLocator accepts one portable nested MCP Bundle document
// locator. The filename must be one of schema.BundleDocumentFileNames.
//
// A nested path is required because managed package publication owns a
// package directory, never the complete managed Source root.
func ValidateDocumentLocator(value basespec.Locator) error {
	if err := basespec.ValidatePortableLocator(value, false); err != nil {
		return err
	}
	if path.Dir(string(value)) == "." ||
		!schema.IsBundleDocumentLocator(value) {
		return fmt.Errorf(
			"%w: MCP document locator must be nested and use a supported MCP Bundle filename",
			basespec.ErrInvalid,
		)
	}
	return nil
}

func encodeStrict(value any) (json.RawMessage, error) {
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

func decodeStrict[T any](raw json.RawMessage) (T, error) {
	var output T
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return output, err
	}
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil {
		return output, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("MCP local data has trailing JSON")
		}
		return output, err
	}
	return output, nil
}
