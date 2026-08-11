package bundle

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/mcp/schema"
)

// canonicalizeBundleBytes is the only public-lifecycle ingress for portable
// MCP Bundle bytes. The input remains raw until Artifact Store dispatches the
// expected registered codec, executes JSON Schema validation, verifies
// canonical JSON, and invokes MCP semantic canonicalization.
func (a *API) canonicalizeBundleBytes(
	ctx context.Context,
	raw []byte,
) (schema.BundleDocument, json.RawMessage, error) {
	if a == nil || a.dependencies.ShareableDocuments == nil {
		return schema.BundleDocument{}, nil, fmt.Errorf(
			"%w: Artifact Store shareable document canonicalizer is unavailable",
			basespec.ErrClosed,
		)
	}
	if len(raw) == 0 {
		return schema.BundleDocument{}, nil, fmt.Errorf(
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
		return schema.BundleDocument{}, nil, fmt.Errorf(
			"canonicalize MCP Bundle through Artifact Store schema registry: %w",
			err,
		)
	}

	document, err := schema.BundleFromParsedDocument(parsed)
	if err != nil {
		return schema.BundleDocument{}, nil, err
	}
	return document, append(json.RawMessage(nil), parsed.Raw...), nil
}

// canonicalizeTrustedBundleDocument is reserved for already typed,
// application-owned installer input. It still enters the same Artifact Store
// expected-schema boundary after encoding.
//
// It must not be used by transport handlers or source readers.
func (a *API) canonicalizeTrustedBundleDocument(
	ctx context.Context,
	input schema.BundleDocument,
) (schema.BundleDocument, json.RawMessage, error) {
	raw, err := json.Marshal(input)
	if err != nil {
		return schema.BundleDocument{}, nil, fmt.Errorf(
			"encode trusted MCP Bundle document: %w",
			err,
		)
	}
	return a.canonicalizeBundleBytes(ctx, raw)
}
