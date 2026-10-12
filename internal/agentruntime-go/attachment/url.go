package attachment

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/flexigpt/llmtools-go/webtool"
)

// URLRef carries metadata for URL-based attachments.
//
// URL:          the raw user-provided URL (after trimming whitespace).
// Normalized:   a canonicalized string representation used internally.
// OrigNormalized: snapshot of the original Normalized value so we can
//
//	detect whether a URL has been modified in-place.
type URLRef struct {
	URL            string `json:"url"`
	Normalized     string `json:"normalized,omitempty"`
	OrigNormalized string `json:"origNormalized"`
}

// PopulateRef validates and normalizes the URL stored in the URLRef.
// It must be called before the URLRef is used.
//
// It ensures:
//   - The URL is non-empty.
//   - The URL parses successfully.
//   - The URL is absolute (has a scheme/host).
//   - Normalized and OrigNormalized are populated.
func (ref *URLRef) PopulateRef(ctx context.Context, replaceOrig bool) error {
	if ref == nil {
		return errors.New("url attachment missing ref")
	}
	raw := strings.TrimSpace(ref.URL)
	if raw == "" {
		return errors.New("url attachment missing url")
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid url %q: %w", raw, err)
	}
	if !parsed.IsAbs() {
		return fmt.Errorf("url %q must be absolute", raw)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("unsupported url scheme %q; only http and https are allowed", parsed.Scheme)
	}

	nURL := parsed.String()
	ref.URL = raw
	ref.Normalized = nURL
	if strings.TrimSpace(ref.OrigNormalized) == "" || replaceOrig {
		ref.OrigNormalized = nURL
	}
	return nil
}

// IsModified reports whether the URL has been modified from its original
// normalized form. This is useful for detecting in-place edits.
func (ref *URLRef) IsModified() bool {
	if ref == nil {
		return false
	}
	if strings.TrimSpace(ref.OrigNormalized) == "" {
		return false
	}
	return ref.Normalized != ref.OrigNormalized
}

// BuildContentBlock builds a ContentBlock representation for a URL-based
// attachment, depending on the desired AttachmentContentBlockMode.
//
// The behaviour is:
//   - AttachmentContentBlockModeTextLink:      always returns a simple text link.
//   - AttachmentContentBlockModeImage:         uses fetchurl binary mode and returns an image block
//     when the fetched content is actually an image, else falls back to a simple link.
//   - AttachmentContentBlockModePageContent / AttachmentContentBlockModeText:
//     uses fetchurl auto mode, which handles HTML/text/PDF/image/file output.
//   - AttachmentContentBlockModeFile:          uses fetchurl binary mode and returns a file block,
//     else falls back to a link.
//   - Any other modes (PR diff/page, commit diff/page, not readable, etc.):
//     safest fallback is a link-only block.
func (ref *URLRef) BuildContentBlock(
	ctx context.Context,
	attachmentContentBlockMode AttachmentContentBlockMode,
	onlyIfTextKind bool,
) (*ContentBlock, error) {
	rawURL := strings.TrimSpace(ref.URL)
	if rawURL == "" {
		return nil, errors.New("got invalid url")
	}

	switch attachmentContentBlockMode {

	case AttachmentContentBlockModeImageURL:
		if onlyIfTextKind {
			return nil, ErrNonTextContentBlock
		}
		return ref.buildImageURLContentBlock(ctx)

	case AttachmentContentBlockModeFileURL:
		if onlyIfTextKind {
			return nil, ErrNonTextContentBlock
		}
		return ref.buildFileURLContentBlock(ctx)

	case AttachmentContentBlockModeImage:
		if onlyIfTextKind {
			return nil, ErrNonTextContentBlock
		}
		return ref.buildImageBlockFromURL(ctx)

	case AttachmentContentBlockModeFile:
		if onlyIfTextKind {
			return nil, ErrNonTextContentBlock
		}
		return ref.buildFileBlockFromURL(ctx)

	case AttachmentContentBlockModeTextLink:
		// Minimal representation: just the URL (optionally with a label).
		return ref.buildTextLinkContentBlock(), nil

	case AttachmentContentBlockModePageContent, AttachmentContentBlockModeText:
		// New pipeline: image/pdf → html/text → link.
		return ref.buildBlocksForURLPage(ctx, onlyIfTextKind)

	case AttachmentContentBlockModeNotReadable,
		AttachmentContentBlockModePRDiff,
		AttachmentContentBlockModePRPage,
		AttachmentContentBlockModeCommitDiff,
		AttachmentContentBlockModeCommitPage:
		// Unknown or special mode: safest fallback is link-only.
		return ref.buildTextLinkContentBlock(), nil

	default:
		return nil, errors.New("unknown attachment mode")
	}
}

// buildImageURLContentBlock builds a ContentBlockImage that carries only the
// remote URL (no base64). This is used with AttachmentContentBlockModeImageURL
// so that the LLM provider can fetch the image directly (e.g. Anthropic,
// OpenAI Chat/Responses).
//
// This mode intentionally does not download bytes.
func (ref *URLRef) buildImageURLContentBlock(_ context.Context) (*ContentBlock, error) {
	rawURL := strings.TrimSpace(ref.URL)
	if rawURL == "" {
		return nil, errors.New("got invalid url")
	}

	cb := &ContentBlock{
		Kind: ContentBlockImage,
		URL:  &rawURL,
	}

	if mt := mimeTypeFromURLPath(rawURL); strings.HasPrefix(mt, "image/") {
		cb.MIMEType = &mt
	}

	return cb, nil
}

// buildFileURLContentBlock builds a ContentBlockFile that carries only the
// remote URL (no base64). This is used with AttachmentContentBlockModeFileURL
// so that the LLM provider can fetch the file directly (e.g. PDFs via URL for
// Anthropic/OpenAI Responses).
//
// This mode intentionally does not download bytes.
func (ref *URLRef) buildFileURLContentBlock(_ context.Context) (*ContentBlock, error) {
	rawURL := strings.TrimSpace(ref.URL)
	if rawURL == "" {
		return nil, errors.New("got invalid url")
	}

	fname := filenameFromURLPath(rawURL)
	cb := &ContentBlock{
		Kind:     ContentBlockFile,
		URL:      &rawURL,
		FileName: &fname,
	}

	if mt := mimeTypeFromURLPath(rawURL); mt != "" {
		cb.MIMEType = &mt
	}

	return cb, nil
}

// buildBlocksForURLPage delegates URL fetching and extraction to the shared
// llmtools-go webtool/fetchurl implementation.
func (ref *URLRef) buildBlocksForURLPage(ctx context.Context, onlyIfTextKind bool) (*ContentBlock, error) {
	cb, err := ref.fetchURLContentBlock(ctx, webtool.FetchEncodingAuto)
	if err != nil {
		//nolint:nilerr // Text link content is a fallback.
		return ref.buildTextLinkContentBlock(), nil
	}
	if onlyIfTextKind && cb.Kind != ContentBlockText {
		return nil, ErrNonTextContentBlock
	}
	return cb, nil
}

func (ref *URLRef) buildImageBlockFromURL(ctx context.Context) (*ContentBlock, error) {
	cb, err := ref.fetchURLContentBlock(ctx, webtool.FetchEncodingBinary)
	if err != nil {
		//nolint:nilerr // Text link content is a fallback.
		return ref.buildTextLinkContentBlock(), nil
	}
	if cb.Kind != ContentBlockImage {
		return ref.buildTextLinkContentBlock(), nil
	}
	return cb, nil
}

// buildFileBlockFromURL fetches bytes through fetchurl binary mode and exposes
// them as a downloadable file. If fetchurl returns an image output, it is
// intentionally re-wrapped as a file because this is file mode.
func (ref *URLRef) buildFileBlockFromURL(ctx context.Context) (*ContentBlock, error) {
	cb, err := ref.fetchURLContentBlock(ctx, webtool.FetchEncodingBinary)
	if err != nil {
		//nolint:nilerr // Text link content is a fallback.
		return ref.buildTextLinkContentBlock(), nil
	}
	if cb.Kind == ContentBlockText {
		return ref.buildTextLinkContentBlock(), nil
	}
	if cb.Kind == ContentBlockImage {
		cb.Kind = ContentBlockFile
	}
	if cb.FileName == nil || strings.TrimSpace(*cb.FileName) == "" {
		rawURL := strings.TrimSpace(ref.URL)
		fname := filenameFromURLPath(rawURL)
		cb.FileName = &fname
	}
	return cb, nil
}

func (ref *URLRef) fetchURLContentBlock(
	ctx context.Context,
	encoding webtool.FetchURLEncoding,
) (*ContentBlock, error) {
	rawURL := strings.TrimSpace(ref.URL)
	if rawURL == "" {
		return nil, errors.New("got invalid url")
	}

	return fetchURLContentBlock(ctx, rawURL, encoding)
}

// buildTextLinkContentBlock returns a simple text block that represents the
// URL as a human-readable link. If the attachment has a label distinct from
// the URL, it is included as "label (url)".
func (ref *URLRef) buildTextLinkContentBlock() *ContentBlock {
	rawURL := strings.TrimSpace(ref.URL)
	return &ContentBlock{
		Kind: ContentBlockText,
		Text: &rawURL,
		URL:  &rawURL,
	}
}
