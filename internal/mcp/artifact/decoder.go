package artifact

import (
	"context"
	"fmt"
	"slices"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/shareable"
	"github.com/flexigpt/flexigpt-app/internal/mcp/schema"
)

type Decoder struct {
	schemas *shareable.Registry
}

func NewDecoder() *Decoder {
	return &Decoder{}
}

func (*Decoder) ID() basespec.DecoderID {
	return DecoderID
}

func (*Decoder) Revision() string {
	return DecoderRevision
}

func (d *Decoder) BindShareableSchemas(
	schemas *shareable.Registry,
) error {
	if d == nil || schemas == nil {
		return fmt.Errorf(
			"%w: MCP decoder requires a shareable schema registry",
			basespec.ErrInvalid,
		)
	}

	expected := schema.NewBundleCodec().Key()
	if slices.Contains(schemas.Keys(), expected) {
		d.schemas = schemas
		return nil
	}
	return fmt.Errorf(
		"%w: MCP Bundle shareable schema is not registered",
		basespec.ErrInvalid,
	)
}

func (d *Decoder) Recognize(
	_ context.Context,
	candidate discovery.Candidate,
) discovery.Recognition {
	if candidate.RequestsDecoder(DecoderID) &&
		schema.IsBundleDocumentLocator(candidate.Locator) {
		return discovery.RecognitionPreferred
	}
	return discovery.RecognitionNone
}

func (d *Decoder) Decode(
	ctx context.Context,
	candidate discovery.Candidate,
) ([]discovery.Decoded, []diagnostic.Diagnostic) {
	if !candidate.RequestsDecoder(DecoderID) ||
		!schema.IsBundleDocumentLocator(candidate.Locator) {
		return nil, nil
	}
	if d == nil || d.schemas == nil {
		return nil, decoderError(
			candidate.Locator,
			"bundle",
			fmt.Errorf("%w: MCP decoder has no bound schema registry", basespec.ErrClosed),
		)
	}

	parsed, err := d.schemas.CanonicalizeEntity(
		ctx,
		shareable.EntityCollection,
		candidate.Content,
	)
	if err != nil {
		return nil, decoderError(candidate.Locator, "bundle", err)
	}
	if parsed.Key != schema.NewBundleCodec().Key() {
		return nil, decoderError(
			candidate.Locator,
			"bundle",
			fmt.Errorf("%w: MCP decoder received another shareable schema", basespec.ErrInvalid),
		)
	}

	bundle, _, err := schema.ParseBundle(parsed.Raw)
	if err != nil || bundle.Digest != parsed.Digest {
		if err == nil {
			err = fmt.Errorf("%w: MCP Bundle digest differs from registry output", basespec.ErrDigestMismatch)
		}
		return nil, decoderError(candidate.Locator, "bundle", err)
	}

	serverNames := make([]string, 0, len(bundle.MCPServers))
	for name := range bundle.MCPServers {
		serverNames = append(serverNames, name)
	}
	sort.Strings(serverNames)

	policyNames := make(
		[]string,
		0,
		len(bundle.BundleExtension.Policies),
	)
	for name := range bundle.BundleExtension.Policies {
		policyNames = append(policyNames, name)
	}
	sort.Strings(policyNames)

	output := make(
		[]discovery.Decoded,
		0,
		len(serverNames)+len(policyNames),
	)

	for _, name := range serverNames {
		serverDocument, err := schema.ServerFromBundle(bundle, name)
		if err != nil {
			return nil, decoderError(candidate.Locator, name, err)
		}
		definition, err := DefinitionForServer(serverDocument)
		if err != nil {
			return nil, decoderError(candidate.Locator, name, err)
		}
		output = append(output, discovery.Decoded{
			SubresourceLocator: ServerSubresource(
				basespec.LogicalName(name),
			),
			Definition: definition,
		})
	}

	for _, name := range policyNames {
		definition, err := DefinitionForPolicy(
			bundle.BundleExtension.Policies[name],
		)
		if err != nil {
			return nil, decoderError(candidate.Locator, name, err)
		}
		output = append(output, discovery.Decoded{
			SubresourceLocator: PolicySubresource(
				basespec.LogicalName(name),
			),
			Definition: definition,
		})
	}

	return output, nil
}

func decoderError(
	locator basespec.Locator,
	subresource string,
	err error,
) []diagnostic.Diagnostic {
	return []diagnostic.Diagnostic{{
		Severity: diagnostic.DiagnosticError,
		Code:     "mcp.bundle.subresource-invalid",
		Message: diagnostic.BoundedDiagnosticMessage(
			fmt.Sprintf("%s: %v", subresource, err),
		),
		Location: &diagnostic.DiagnosticLocation{
			Locator: locator,
		},
	}}
}
