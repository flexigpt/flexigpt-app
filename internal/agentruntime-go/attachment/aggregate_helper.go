package attachment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/llmtoolsutil"
	"github.com/flexigpt/llmtools-go/fstool"
)

// BuildAttachmentForFile builds an Attachment for a local filesystem path.
// It inspects the MIME type / extension and chooses an appropriate
// AttachmentKind, default Mode, and AvailableContentBlockModes.
// The returned attachment is fully populated via PopulateRef().
// Note that this builds a fresh attachment, i.e both original ref and current are populated here.
func BuildAttachmentForFile(ctx context.Context, pathInfo *PathInfo) (*Attachment, error) {
	if pathInfo == nil {
		return nil, errors.New("invalid input pathinfo")
	}

	if !pathInfo.Exists {
		return nil, fmt.Errorf("file does not exist: %s", pathInfo.Path)
	}
	if pathInfo.IsDir {
		return nil, fmt.Errorf("path %q is a directory; expected file", pathInfo.Path)
	}

	toolOut, err := llmtoolsutil.MIMEForPath(ctx, fstool.MIMEForPathArgs{
		Path: pathInfo.Path,
	})
	if err != nil || toolOut == nil {
		return nil, errors.Join(ErrUnreadableFile, err)
	}

	baseMIMEType := toolOut.BaseMIMEType
	extMode := toolOut.Mode
	baseName := filepath.Base(pathInfo.Path)

	switch extMode {
	case fstool.MIMEModeImage:
		// Treat images as dedicated image attachments.
		att := &Attachment{
			Kind:  AttachmentImage,
			Label: baseName,
			Mode:  AttachmentContentBlockModeImage,
			AvailableContentBlockModes: []AttachmentContentBlockMode{
				AttachmentContentBlockModeImage,
			},
			ImageRef: &ImageRef{
				Path:    pathInfo.Path,
				Name:    pathInfo.Name,
				Exists:  pathInfo.Exists,
				IsDir:   pathInfo.IsDir,
				Size:    pathInfo.Size,
				ModTime: pathInfo.ModTime,
			},
		}
		if err := att.PopulateRef(ctx, false); err != nil {
			return nil, err
		}
		return att, nil

	case fstool.MIMEModeText:
		// Source code / markdown / text files: send as text by default.
		att := &Attachment{
			Kind:  AttachmentFile,
			Label: baseName,
			Mode:  AttachmentContentBlockModeText,
			AvailableContentBlockModes: []AttachmentContentBlockMode{
				AttachmentContentBlockModeText,
			},
			FileRef: &FileRef{
				PathInfo: *pathInfo,
			},
		}
		if err := att.PopulateRef(ctx, false); err != nil {
			return nil, err
		}
		return att, nil

	case fstool.MIMEModeDocument:
		// Documents (PDF, Office, etc.).
		// As of now APIs and we internally only support PDF docs.
		// PDFs can be treated as text (with extraction) or as original file.
		if MIMEType(baseMIMEType) != MIMEApplicationPDF {
			return buildUnreadableFileAttachment(*pathInfo), nil
		}

		att := &Attachment{
			Kind:  AttachmentFile,
			Label: baseName,
			Mode:  AttachmentContentBlockModeText,
			AvailableContentBlockModes: []AttachmentContentBlockMode{
				AttachmentContentBlockModeText,
				AttachmentContentBlockModeFile,
			},
			FileRef: &FileRef{
				PathInfo: *pathInfo,
			},
		}
		if err := att.PopulateRef(ctx, false); err != nil {
			return nil, err
		}
		return att, nil

	case fstool.MIMEModeDefault:
		return buildUnreadableFileAttachment(*pathInfo), nil

	default:
		// Unknown / binary. We still keep it as a file attachment but mark it not-readable so BuildContentBlock
		// produces a short placeholder instead of trying to read it.
		return buildUnreadableFileAttachment(*pathInfo), nil

	}
}

func buildUnreadableFileAttachment(pathInfo PathInfo) *Attachment {
	return &Attachment{
		Kind:  AttachmentFile,
		Label: filepath.Base(pathInfo.Path),
		Mode:  AttachmentContentBlockModeNotReadable,
		AvailableContentBlockModes: []AttachmentContentBlockMode{
			AttachmentContentBlockModeNotReadable,
		},
		FileRef: &FileRef{
			PathInfo: pathInfo,
		},
	}
}

// BuildAttachmentForURL builds an Attachment for a remote URL.
// It uses a timeout context to peek and infer type and then give proper options.
func BuildAttachmentForURL(rawURL string) (*Attachment, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	a, err := BuildAttachmentForURLWithContext(ctx, rawURL)
	return a, err
}

// BuildAttachmentForURLWithContext builds an Attachment for a remote URL.
//
// Mode detection strategy:
//  1. Extension-based detection when the URL path has a recognizable suffix.
//  2. Fallback to PageContent + TextLink. The actual fetch/extract happens later
//     through llmtools-go webtool/fetchurl.
//
// Note: It can still return an error for invalid/empty/non-absolute URLs because
// PopulateRef enforces validity.
func BuildAttachmentForURLWithContext(ctx context.Context, rawURL string) (*Attachment, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return nil, errors.New("empty url")
	}

	// Parse early mainly to infer extension; PopulateRef will validate absolute URL later.
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, errors.New("invalid url")
	}

	// Extension fallback inference.
	ext := strings.ToLower(filepath.Ext(parsed.Path)) // includes leading "."
	if len(ext) <= 1 {
		ext = ""
	}

	// Canonical mode sets we reuse.
	imageModes := []AttachmentContentBlockMode{
		AttachmentContentBlockModeImage,
		AttachmentContentBlockModeImageURL,
		AttachmentContentBlockModeTextLink,
	}
	pdfModes := []AttachmentContentBlockMode{
		AttachmentContentBlockModeText, // allow PDF -> extracted text view
		AttachmentContentBlockModeFile, // raw file (download+inline)
		AttachmentContentBlockModeFileURL,
		AttachmentContentBlockModeTextLink,
	}
	fileModes := []AttachmentContentBlockMode{
		AttachmentContentBlockModeFile,
		AttachmentContentBlockModeTextLink,
	}
	textModes := []AttachmentContentBlockMode{
		AttachmentContentBlockModeText,
		AttachmentContentBlockModeTextLink,
	}
	pageModes := []AttachmentContentBlockMode{
		AttachmentContentBlockModePageContent,
		AttachmentContentBlockModeTextLink,
	}

	// Default to web page fetching. The fetchurl tool will later return text,
	// image, or file output based on the actual response.
	mode := AttachmentContentBlockModePageContent
	available := pageModes

	// Use extension-based classification when a path suffix is available. This
	// avoids doing network I/O while the attachment is being created.
	if ext != "" {
		toolOut, err := llmtoolsutil.MIMEForExtension(ctx, fstool.MIMEForExtensionArgs{
			Extension: ext,
		})
		if err == nil && toolOut != nil {
			baseMIMEType := normalizeMIMEType(toolOut.BaseMIMEType)

			switch toolOut.Mode {
			case fstool.MIMEModeImage:
				mode = AttachmentContentBlockModeImage
				available = imageModes

			case fstool.MIMEModeDocument:
				if MIMEType(baseMIMEType) == MIMEApplicationPDF || ext == string(ExtPDF) {
					mode = AttachmentContentBlockModeFile
					available = pdfModes
				} else {
					mode = AttachmentContentBlockModeFile
					available = fileModes
				}

			case fstool.MIMEModeText:
				if isHTMLMIMEType(baseMIMEType) || ext == string(ExtHTML) || ext == string(ExtHTM) {
					mode = AttachmentContentBlockModePageContent
					available = pageModes
				} else {
					mode = AttachmentContentBlockModeText
					available = textModes
				}

			default:
				// Unknown extensions are more likely downloadable files than web
				// pages, so expose file mode plus link fallback.
				mode = AttachmentContentBlockModeFile
				available = fileModes
			}
		} else {
			if isPlainTextLikeMIME(mimeTypeFromURLPath(trimmed)) {
				mode = AttachmentContentBlockModeText
				available = textModes
			} else {
				mode = AttachmentContentBlockModeFile
				available = fileModes
			}
		}
	}

	att := &Attachment{
		Kind:                       AttachmentURL,
		Label:                      trimmed,
		Mode:                       mode,
		AvailableContentBlockModes: available,
		URLRef: &URLRef{
			URL: trimmed,
		},
	}

	// Ensure ref is populated/validated (absolute URL requirement, normalized fields, etc.)
	if err := att.PopulateRef(ctx, false); err != nil {
		return nil, err
	}

	return att, nil
}

// BuildContentBlocks converts high-level attachments (file paths, URLs, etc.)
// into provider-agnostic content blocks that can then be adapted for each LLM.
func BuildContentBlocks(ctx context.Context, atts []Attachment, opts ...ContentBlockOption) ([]ContentBlock, error) {
	if len(atts) == 0 {
		return nil, nil
	}
	blocks := make([]ContentBlock, 0, len(atts))
	buildContentOptions := getBuildContentBlockOptions(opts...)
	for i := range atts {
		att := &atts[i]
		b, err := att.BuildContentBlock(ctx, opts...)
		if err != nil {
			switch {
			case errors.Is(err, ErrExistingContentBlock):
				// If content block already existed we should just reattach it.
				b = att.ContentBlock
				slog.Warn("got existing att", "a", att)

			case errors.Is(err, ErrAttachmentModifiedSinceSnapshot) && !buildContentOptions.OverrideOriginal:
				displayBlock, err := att.GetTextBlockWithDisplayNameOnly(
					"attachment modified since this message was sent",
				)
				if err != nil {
					continue
				}
				b = displayBlock
			default:
				slog.Warn("failed to build content block for attachment", "err", err, "attachment", att)
				// Skip this content block. It is ok if the build block skipped this because OnlyIfTextKind was set or
				// any other error.
				continue
			}
		}

		blocks = append(blocks, *b)
	}

	return blocks, nil
}
