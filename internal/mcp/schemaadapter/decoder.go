package schemaadapter

import (
	"context"
	"fmt"
	"slices"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/shareable"

	"github.com/flexigpt/flexigpt-app/internal/mcp/bundle"
	"github.com/flexigpt/flexigpt-app/internal/mcp/policy"
	"github.com/flexigpt/flexigpt-app/internal/mcp/server"
)

type Decoder struct {
	documents shareable.ExpectedCanonicalizer
}

func NewDecoder() *Decoder {
	return &Decoder{}
}

func (*Decoder) ID() basespec.DecoderID {
	return artifactbuiltin.DecoderID
}

func (*Decoder) Revision() string {
	return artifactbuiltin.DecoderRevision
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

	expected := artifactbuiltin.MCPSchemaKey
	if slices.Contains(schemas.Keys(), expected) {
		d.documents = schemas
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
	if candidate.RequestsDecoder(artifactbuiltin.DecoderID) &&
		bundle.IsBundleDocumentLocator(candidate.Locator) {
		return discovery.RecognitionPreferred
	}
	return discovery.RecognitionNone
}

func (d *Decoder) Decode(
	ctx context.Context,
	candidate discovery.Candidate,
) ([]discovery.Decoded, []diagnostic.Diagnostic) {
	if !candidate.RequestsDecoder(artifactbuiltin.DecoderID) ||
		!bundle.IsBundleDocumentLocator(candidate.Locator) {
		return nil, nil
	}
	if d == nil || d.documents == nil {
		return nil, decoderError(
			candidate.Locator,
			"bundle",
			fmt.Errorf("%w: MCP decoder has no bound schema registry", basespec.ErrClosed),
		)
	}

	parsed, err := d.documents.CanonicalizeExpected(
		ctx,
		artifactbuiltin.MCPSchemaKey,
		candidate.Content,
	)
	if err != nil {
		return nil, decoderError(candidate.Locator, "bundle", err)
	}
	b, err := bundle.BundleFromParsedDocument(parsed)
	if err != nil {
		return nil, decoderError(candidate.Locator, "bundle", err)
	}

	serverNames := make([]string, 0, len(b.MCPServers))
	for name := range b.MCPServers {
		serverNames = append(serverNames, name)
	}
	sort.Strings(serverNames)

	policyNames := make(
		[]string,
		0,
		len(b.BundleExtension.Policies),
	)
	for name := range b.BundleExtension.Policies {
		policyNames = append(policyNames, name)
	}
	sort.Strings(policyNames)

	output := make(
		[]discovery.Decoded,
		0,
		len(serverNames)+len(policyNames),
	)

	for _, name := range serverNames {
		serverDocument, err := bundle.ServerFromCanonicalBundle(b, name)
		if err != nil {
			return nil, decoderError(candidate.Locator, name, err)
		}
		definition, err := server.DefinitionForCanonicalServer(serverDocument)
		if err != nil {
			return nil, decoderError(candidate.Locator, name, err)
		}
		output = append(output, discovery.Decoded{
			SubresourceLocator: server.ServerSubresource(
				basespec.LogicalName(name),
			),
			Definition: definition,
		})
	}

	for _, name := range policyNames {
		definition, err := policy.DefinitionForCanonicalPolicy(
			b.BundleExtension.Policies[name],
		)
		if err != nil {
			return nil, decoderError(candidate.Locator, name, err)
		}
		output = append(output, discovery.Decoded{
			SubresourceLocator: policy.PolicySubresource(
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
