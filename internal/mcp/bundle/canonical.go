package bundle

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/mcp/schema"
)

// canonicalizeBundleDocument is the only Bundle lifecycle ingress for
// portable MCP Bundle content. Artifact Store owns registry dispatch, JSON
// Schema validation, canonical JSON enforcement, and codec invocation.
//
// MCP receives only the already canonical typed projection and uses it for
// feature-specific Definition projection, installation validation, policy
// composition, and managed package lifecycle.
func (a *API) canonicalizeBundleDocument(
	ctx context.Context,
	input schema.BundleDocument,
) (schema.BundleDocument, json.RawMessage, error) {
	if a == nil || a.dependencies.ShareableDocuments == nil {
		return schema.BundleDocument{}, nil, fmt.Errorf(
			"%w: Artifact Store shareable document canonicalizer is unavailable",
			basespec.ErrClosed,
		)
	}

	raw, err := json.Marshal(input)
	if err != nil {
		return schema.BundleDocument{}, nil, fmt.Errorf(
			"encode MCP Bundle document for Artifact Store validation: %w",
			err,
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
