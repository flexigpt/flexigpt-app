package inferencewrapper

import (
	"context"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/attachment"
	inferenceSpec "github.com/flexigpt/inference-go/spec"
)

func buildContentItemsFromAttachments(
	ctx context.Context,
	atts []attachment.Attachment,
) ([]inferenceSpec.InputOutputContentItemUnion, error) {
	items := make([]inferenceSpec.InputOutputContentItemUnion, 0)
	if len(atts) == 0 {
		return items, nil
	}

	blocks, err := attachment.BuildContentBlocks(
		ctx,
		atts,
		attachment.WithOverrideOriginalContentBlock(true),
		attachment.WithOnlyTextKindContentBlock(false),
	)
	if err != nil {
		return nil, err
	}

	for _, b := range blocks {
		switch b.Kind {
		case attachment.ContentBlockText:
			if b.Text == nil {
				continue
			}
			txt := strings.TrimSpace(*b.Text)
			if txt == "" {
				continue
			}
			formattedTxt, err := attachment.FormatTextBlockForLLM(b)
			if err != nil {
				return nil, err
			}
			if formattedTxt == "" {
				continue
			}
			items = append(items, inferenceSpec.InputOutputContentItemUnion{
				Kind: inferenceSpec.ContentItemKindText,
				TextItem: &inferenceSpec.ContentItemText{
					Text: formattedTxt,
				},
			})

		case attachment.ContentBlockImage:
			var data, urlStr string
			if b.Base64Data != nil {
				data = strings.TrimSpace(*b.Base64Data)
			}
			if b.URL != nil {
				urlStr = strings.TrimSpace(*b.URL)
			}
			// Require at least one of base64 or URL to be present.
			if data == "" && urlStr == "" {
				continue
			}
			mime := inferenceSpec.DefaultImageDataMIME
			if b.MIMEType != nil && strings.TrimSpace(*b.MIMEType) != "" {
				mime = strings.TrimSpace(*b.MIMEType)
			}
			name := ""
			if b.FileName != nil {
				name = strings.TrimSpace(*b.FileName)
			}
			img := &inferenceSpec.ContentItemImage{
				ImageName: name,
				ImageMIME: mime,
			}
			if data != "" {
				img.ImageData = data
			}
			if urlStr != "" {
				img.ImageURL = urlStr
			}
			items = append(items, inferenceSpec.InputOutputContentItemUnion{
				Kind:      inferenceSpec.ContentItemKindImage,
				ImageItem: img,
			})

		case attachment.ContentBlockFile:
			var data, urlStr string
			if b.Base64Data != nil {
				data = strings.TrimSpace(*b.Base64Data)
			}
			if b.URL != nil {
				urlStr = strings.TrimSpace(*b.URL)
			}
			// Require at least one of base64 or URL to be present.
			if data == "" && urlStr == "" {
				continue
			}
			mime := inferenceSpec.DefaultFileDataMIME
			if b.MIMEType != nil && strings.TrimSpace(*b.MIMEType) != "" {
				mime = strings.TrimSpace(*b.MIMEType)
			}
			name := ""
			if b.FileName != nil {
				name = strings.TrimSpace(*b.FileName)
			}

			file := &inferenceSpec.ContentItemFile{
				FileName: name,
				FileMIME: mime,
			}
			if data != "" {
				file.FileData = data
			}
			if urlStr != "" {
				file.FileURL = urlStr
			}

			items = append(items, inferenceSpec.InputOutputContentItemUnion{
				Kind:     inferenceSpec.ContentItemKindFile,
				FileItem: file,
			})
		}
	}

	return items, nil
}
