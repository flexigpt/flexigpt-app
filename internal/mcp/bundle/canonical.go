package bundle

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/shareable"
	"github.com/flexigpt/flexigpt-app/internal/mcp/schema"
)

// canonicalizeBundleBytes is the only public-lifecycle ingress for portable
// MCP Bundle bytes. The input remains raw until Artifact Store dispatches the
// expected registered codec, executes JSON Schema validation, verifies
// canonical JSON, and invokes MCP semantic canonicalization.
func (a *API) canonicalizeBundleBytes(
	ctx context.Context,
	raw []byte,
) (schema.BundleDocument, shareable.ParsedDocument, error) {
	if a == nil || a.dependencies.ShareableDocuments == nil {
		return schema.BundleDocument{}, shareable.ParsedDocument{}, fmt.Errorf(
			"%w: Artifact Store shareable document canonicalizer is unavailable",
			basespec.ErrClosed,
		)
	}
	if len(raw) == 0 {
		return schema.BundleDocument{}, shareable.ParsedDocument{}, fmt.Errorf(
			"%w: MCP Bundle document is required",
			basespec.ErrInvalid,
		)
	}

	parsed, err := a.dependencies.ShareableDocuments.CanonicalizeExpected(
		ctx,
		schema.BundleCodec{}.Key(),
		raw,
	)
	if err != nil {
		return schema.BundleDocument{}, shareable.ParsedDocument{}, fmt.Errorf(
			"canonicalize MCP Bundle through Artifact Store schema registry: %w",
			err,
		)
	}

	document, err := schema.BundleFromParsedDocument(parsed)
	if err != nil {
		return schema.BundleDocument{}, shareable.ParsedDocument{}, err
	}
	return document, parsed.Clone(), nil
}
