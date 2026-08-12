package bundle

import (
	"encoding/json"
	"fmt"
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/builtin/schema"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

func DecodeAttachmentData(
	raw json.RawMessage,
) (AttachmentData, error) {
	value, err := jsonutil.DecodeCanonicalObject[AttachmentData](raw, basespec.MaxDefinitionBytes)
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
	return jsonutil.MarshalCanonicalObject(value, basespec.MaxDefinitionBytes)
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
