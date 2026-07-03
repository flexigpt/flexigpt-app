package attachment

import (
	"context"
	"errors"
	"mime"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/llmtoolsutil"
	llmtoolsSpec "github.com/flexigpt/llmtools-go/spec"
	"github.com/flexigpt/llmtools-go/webtool"
)

const attachmentFetchTextMaxLength = 100000

func fetchURLContentBlock(
	ctx context.Context,
	rawURL string,
	encoding webtool.FetchURLEncoding,
) (*ContentBlock, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, errors.New("empty url")
	}

	toolOut, err := llmtoolsutil.FetchURL(ctx, webtool.FetchURLArgs{
		URL:       rawURL,
		Encoding:  string(encoding),
		MaxLength: attachmentFetchTextMaxLength,
	})
	if err != nil {
		return nil, err
	}

	return contentBlockFromFetchOutputs(rawURL, toolOut)
}

func contentBlockFromFetchOutputs(
	rawURL string,
	toolOut []llmtoolsSpec.ToolOutputUnion,
) (*ContentBlock, error) {
	for i := range toolOut {
		out := toolOut[i]

		switch out.Kind {
		case llmtoolsSpec.ToolOutputKindText:
			if out.TextItem == nil || strings.TrimSpace(out.TextItem.Text) == "" {
				continue
			}

			text := out.TextItem.Text
			mimeType := "text/plain"
			return &ContentBlock{
				Kind:     ContentBlockText,
				Text:     &text,
				URL:      &rawURL,
				MIMEType: &mimeType,
			}, nil

		case llmtoolsSpec.ToolOutputKindImage:
			if out.ImageItem == nil || strings.TrimSpace(out.ImageItem.ImageData) == "" {
				continue
			}

			data := out.ImageItem.ImageData
			cb := &ContentBlock{
				Kind:       ContentBlockImage,
				Base64Data: &data,
				URL:        &rawURL,
			}

			if mt := normalizeMIMEType(out.ImageItem.ImageMIME); mt != "" {
				cb.MIMEType = &mt
			}
			if name := strings.TrimSpace(out.ImageItem.ImageName); name != "" {
				cb.FileName = &name
			}

			return cb, nil

		case llmtoolsSpec.ToolOutputKindFile:
			if out.FileItem == nil || strings.TrimSpace(out.FileItem.FileData) == "" {
				continue
			}

			data := out.FileItem.FileData
			cb := &ContentBlock{
				Kind:       ContentBlockFile,
				Base64Data: &data,
				URL:        &rawURL,
			}

			if mt := normalizeMIMEType(out.FileItem.FileMIME); mt != "" {
				cb.MIMEType = &mt
			}
			if name := strings.TrimSpace(out.FileItem.FileName); name != "" {
				cb.FileName = &name
			}

			return cb, nil
		default:
		}
	}

	return nil, errors.New("fetchurl returned no usable content")
}

func normalizeMIMEType(ct string) string {
	ct = strings.TrimSpace(ct)
	if ct == "" {
		return ""
	}

	mt, _, err := mime.ParseMediaType(ct)
	if err == nil && strings.TrimSpace(mt) != "" {
		return strings.ToLower(strings.TrimSpace(mt))
	}

	if i := strings.Index(ct, ";"); i >= 0 {
		ct = ct[:i]
	}

	return strings.ToLower(strings.TrimSpace(ct))
}

func isHTMLMIMEType(ct string) bool {
	ct = normalizeMIMEType(ct)
	return ct == "text/html" ||
		ct == "application/xhtml+xml" ||
		strings.Contains(ct, "html")
}

func isPlainTextLikeMIME(ct string) bool {
	ct = normalizeMIMEType(ct)
	if ct == "" || isHTMLMIMEType(ct) {
		return false
	}

	if strings.HasPrefix(ct, "text/") {
		return true
	}
	if strings.HasSuffix(ct, "+json") || strings.HasSuffix(ct, "+xml") {
		return true
	}

	switch ct {
	case "application/json",
		"application/xml",
		"application/javascript",
		"application/ecmascript",
		"application/x-javascript",
		"application/x-ndjson",
		"application/yaml",
		"application/x-yaml",
		"application/toml",
		"application/x-www-form-urlencoded":
		return true
	default:
		return false
	}
}

func mimeTypeFromURLPath(rawURL string) string {
	pathForExt := strings.TrimSpace(rawURL)
	if u, err := url.Parse(rawURL); err == nil {
		pathForExt = u.Path
	}

	if ext := filepath.Ext(pathForExt); ext != "" {
		return normalizeMIMEType(mime.TypeByExtension(ext))
	}

	return ""
}

func filenameFromURLPath(rawURL string) string {
	if u, err := url.Parse(rawURL); err == nil {
		name := filepath.Base(u.Path)
		if decoded, err := url.PathUnescape(name); err == nil {
			name = decoded
		}
		name = strings.Trim(name, ". \t\r\n")
		if name != "" && name != "/" {
			return name
		}
	}

	return "download"
}
