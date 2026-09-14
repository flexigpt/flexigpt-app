package providerapi

import (
	"bytes"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

const markdownMediaType = "text/markdown"

func normalizeMarkdown(
	content []byte,
) (string, error) {
	if !utf8.Valid(content) {
		return "", fmt.Errorf(
			"%w: Markdown source must contain valid UTF-8",
			basespec.ErrInvalid,
		)
	}
	if bytes.ContainsRune(content, 0) {
		return "", fmt.Errorf(
			"%w: Markdown source contains a NUL byte",
			basespec.ErrInvalid,
		)
	}

	value := strings.ReplaceAll(
		strings.ReplaceAll(string(content), "\r\n", "\n"),
		"\r",
		"\n",
	)
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf(
			"%w: Markdown source is empty",
			basespec.ErrInvalid,
		)
	}
	return value, nil
}

func stringPointer(value string) *string {
	output := value
	return &output
}
