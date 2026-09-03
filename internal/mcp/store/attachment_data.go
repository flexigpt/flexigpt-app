package store

import (
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type AttachmentData struct {
	SchemaVersion  string                       `json:"schemaVersion"`
	PackageAddress source.ManagedPackageAddress `json:"packageAddress"`
}

func DecodeAttachmentData(
	raw json.RawMessage,
) (AttachmentData, error) {
	value, err := jsonutil.DecodeCanonicalObject[AttachmentData](
		raw,
		basespec.MaxDefinitionBytes,
	)
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
	if value.SchemaVersion != artifactbuiltin.MCPSchemaVersion {
		return nil, fmt.Errorf(
			"%w: invalid MCP Bundle attachment data schema",
			basespec.ErrInvalid,
		)
	}
	if err := validateBundlePackageAddress(value.PackageAddress); err != nil {
		return nil, err
	}
	if _, err := DocumentLocatorForPackage(value.PackageAddress); err != nil {
		return nil, err
	}
	return jsonutil.MarshalCanonicalObject(
		value,
		basespec.MaxDefinitionBytes,
	)
}

func (d AttachmentData) DocumentLocator() (basespec.Locator, error) {
	if _, err := EncodeAttachmentData(d); err != nil {
		return "", err
	}
	return DocumentLocatorForPackage(d.PackageAddress)
}
