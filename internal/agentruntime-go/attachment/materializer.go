package attachment

import (
	"context"
	"encoding/json"
	"fmt"

	sessionRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/session"
)

const (
	attachmentWirePathKey         = "path"
	attachmentWireOriginalPathKey = "origPath"
	attachmentWireNameKey         = "name"
	attachmentWireExistsKey       = "exists"
	attachmentWireDirectoryKey    = "isDir"
	attachmentWireSizeKey         = "size"
	attachmentWireOriginalSizeKey = "origSize"
	attachmentWireOriginalModKey  = "origModTime"
)

type Materializer interface {
	Materialize(
		ctx context.Context,
		references []sessionRuntime.AttachmentReference,
	) ([]Attachment, error)
}

// JSONReferenceMaterializer converts Session durable references into the
// existing attachment domain shape without copying hydrated Base64 payloads
// into Session state.
type JSONReferenceMaterializer struct{}

func NewJSONReferenceMaterializer() *JSONReferenceMaterializer {
	return &JSONReferenceMaterializer{}
}

func (m *JSONReferenceMaterializer) Materialize(
	_ context.Context,
	references []sessionRuntime.AttachmentReference,
) ([]Attachment, error) {
	if len(references) == 0 {
		return nil, nil
	}

	wire := make([]map[string]any, 0, len(references))
	for _, reference := range references {
		value, err := attachmentWire(reference)
		if err != nil {
			return nil, err
		}
		wire = append(wire, value)
	}

	raw, err := json.Marshal(wire)
	if err != nil {
		return nil, fmt.Errorf("marshal attachment references: %w", err)
	}

	var output []Attachment
	if err := json.Unmarshal(raw, &output); err != nil {
		return nil, fmt.Errorf("decode attachment references: %w", err)
	}

	return output, nil
}

func attachmentWire(
	reference sessionRuntime.AttachmentReference,
) (map[string]any, error) {
	value := map[string]any{
		"label": reference.Label,
	}
	if reference.Mode != "" {
		value["mode"] = reference.Mode
	}
	if len(reference.AvailableModes) != 0 {
		value["availableContentBlockModes"] = reference.AvailableModes
	}

	switch reference.Kind {
	case sessionRuntime.AttachmentReferenceFile:
		if reference.File == nil {
			return nil, errorsf("file attachment reference has no file reference")
		}
		value["kind"] = "file"
		value["fileRef"] = localFileWire(reference.File)

	case sessionRuntime.AttachmentReferenceImage:
		if reference.Image == nil {
			return nil, errorsf("image attachment reference has no image reference")
		}
		value["kind"] = "image"
		value["imageRef"] = localImageWire(reference.Image)

	case sessionRuntime.AttachmentReferenceURL:
		if reference.URL == nil {
			return nil, errorsf("URL attachment reference has no URL reference")
		}
		value["kind"] = "url"
		value["urlRef"] = map[string]any{
			"url":            reference.URL.URL,
			"normalized":     reference.URL.Normalized,
			"origNormalized": reference.URL.Original,
		}

	case sessionRuntime.AttachmentReferenceGeneric:
		if reference.Generic == nil {
			return nil, errorsf("generic attachment reference has no handle")
		}
		value["kind"] = "docIndex"
		value["genericRef"] = map[string]any{
			"handle":     reference.Generic.Handle,
			"origHandle": reference.Generic.OriginalHandle,
		}

	case sessionRuntime.AttachmentReferenceDirectory:
		if reference.Directory == nil {
			return nil, errorsf("directory attachment reference has no directory reference")
		}
		value["kind"] = "file"
		value["fileRef"] = map[string]any{
			attachmentWirePathKey:         reference.Directory.Path,
			attachmentWireOriginalPathKey: reference.Directory.OriginalPath,
			attachmentWireNameKey:         reference.Directory.Name,
			attachmentWireExistsKey:       true,
			attachmentWireDirectoryKey:    true,
			attachmentWireSizeKey:         0,
			attachmentWireOriginalSizeKey: 0,
			attachmentWireOriginalModKey:  "",
		}

	default:
		return nil, errorsf("unsupported attachment reference kind %q", reference.Kind)
	}

	return value, nil
}

func localFileWire(
	reference *sessionRuntime.LocalAttachmentReference,
) map[string]any {
	return map[string]any{
		attachmentWirePathKey:         reference.Path,
		attachmentWireOriginalPathKey: reference.OriginalPath,
		attachmentWireNameKey:         reference.Name,
		attachmentWireExistsKey:       reference.Exists,
		attachmentWireDirectoryKey:    reference.IsDirectory,
		attachmentWireSizeKey:         reference.Size,
		attachmentWireOriginalSizeKey: reference.OriginalSize,
		"modTime":                     reference.ModifiedAt,
		attachmentWireOriginalModKey:  reference.OriginalMod,
	}
}

func localImageWire(
	reference *sessionRuntime.LocalAttachmentReference,
) map[string]any {
	return map[string]any{
		attachmentWirePathKey:         reference.Path,
		attachmentWireOriginalPathKey: reference.OriginalPath,
		attachmentWireNameKey:         reference.Name,
		attachmentWireExistsKey:       reference.Exists,
		attachmentWireDirectoryKey:    reference.IsDirectory,
		attachmentWireSizeKey:         reference.Size,
		attachmentWireOriginalSizeKey: reference.OriginalSize,
		"modTime":                     reference.ModifiedAt,
		attachmentWireOriginalModKey:  reference.OriginalMod,
		"mimeType":                    reference.MIMEType,
		"width":                       reference.Width,
		"height":                      reference.Height,
		"format":                      reference.Format,
	}
}

func errorsf(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}
