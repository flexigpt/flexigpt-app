package providerapi

import (
	"bytes"
	"fmt"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

const markdownMediaType = "text/markdown"

func normalizeMarkdownOptional(
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
	return value, nil
}

// sourceEntryDeclarationLocator points back to the physical source entry
// containing a source-format declaration. It keeps source material out of
// Definition.Body while preserving declaration-relative locator semantics.
func sourceEntryDeclarationLocator(
	locator basespec.Locator,
) *declaration.Locator {
	value := declaration.ScalarLocator(
		"./" + path.Base(string(locator)),
	)
	return &value
}

func stringPointer(value string) *string {
	output := value
	return &output
}
