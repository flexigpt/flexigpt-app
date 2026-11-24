package attachmentruntime

import (
	"fmt"
	"strings"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	inferenceSpec "github.com/flexigpt/inference-go/spec"
)

const (
	defaultProviderImageMIME = "image/png"
	defaultProviderFileMIME  = "application/octet-stream"
	fileTextEnvelopeFormat   = "<<<FILE name=%q>>>\n%s\n<<<END FILE>>>"
)

// ProviderContentItem is the Attachment-domain conversion from a durable V3
// semantic Attachment snapshot to provider-neutral Model content.
func ProviderContentItem(
	value sessionv3Runtime.Attachment,
) (inferenceSpec.InputOutputContentItemUnion, bool) {
	switch value.Kind {
	case sessionv3Runtime.AttachmentKindText:
		if strings.TrimSpace(value.Text) == "" {
			return inferenceSpec.InputOutputContentItemUnion{}, false
		}
		return inferenceSpec.InputOutputContentItemUnion{
			Kind: inferenceSpec.ContentItemKindText,
			TextItem: &inferenceSpec.ContentItemText{
				Text: providerAttachmentText(value),
			},
		}, true
	case sessionv3Runtime.AttachmentKindImage:
		image := &inferenceSpec.ContentItemImage{ImageMIME: defaultProviderImageMIME}
		image.ImageName = value.Label
		if value.MIMEType != "" {
			image.ImageMIME = value.MIMEType
		}
		image.ImageData = value.Base64Data
		image.ImageURL = value.URL
		if image.ImageData == "" && image.ImageURL == "" {
			return inferenceSpec.InputOutputContentItemUnion{}, false
		}
		return inferenceSpec.InputOutputContentItemUnion{
			Kind: inferenceSpec.ContentItemKindImage, ImageItem: image,
		}, true
	case sessionv3Runtime.AttachmentKindFile:
		file := &inferenceSpec.ContentItemFile{FileMIME: defaultProviderFileMIME}
		file.FileName = value.Label
		if value.MIMEType != "" {
			file.FileMIME = value.MIMEType
		}
		file.FileData = value.Base64Data
		file.FileURL = value.URL
		if file.FileData == "" && file.FileURL == "" {
			if strings.TrimSpace(value.Text) == "" {
				return inferenceSpec.InputOutputContentItemUnion{}, false
			}
			return inferenceSpec.InputOutputContentItemUnion{
				Kind:     inferenceSpec.ContentItemKindText,
				TextItem: &inferenceSpec.ContentItemText{Text: value.Text},
			}, true
		}
		return inferenceSpec.InputOutputContentItemUnion{
			Kind: inferenceSpec.ContentItemKindFile, FileItem: file,
		}, true
	case sessionv3Runtime.AttachmentKindURL:
		text := strings.TrimSpace(value.URL)
		if text == "" {
			text = value.Text
		}
		if strings.TrimSpace(text) == "" {
			return inferenceSpec.InputOutputContentItemUnion{}, false
		}
		return inferenceSpec.InputOutputContentItemUnion{
			Kind:     inferenceSpec.ContentItemKindText,
			TextItem: &inferenceSpec.ContentItemText{Text: text},
		}, true
	default:
		return inferenceSpec.InputOutputContentItemUnion{}, false
	}
}

func providerAttachmentText(value sessionv3Runtime.Attachment) string {
	if strings.TrimSpace(value.Label) == "" {
		return value.Text
	}
	return fmt.Sprintf(fileTextEnvelopeFormat, value.Label, value.Text)
}
