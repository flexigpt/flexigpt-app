package schemaadapter

import (
	"context"
	"fmt"
	"slices"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"

	mcpStore "github.com/flexigpt/flexigpt-app/internal/mcp/store"
	mcpStorePolicy "github.com/flexigpt/flexigpt-app/internal/mcp/store/policy"
	mcpStoreServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/server"
)

type Decoder struct {
	documents providerapi.ExpectedCanonicalizer
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

func (*Decoder) RequiredSchemaKeys() []providerapi.SchemaKey {
	return []providerapi.SchemaKey{
		artifactbuiltin.MCPBundleSchemaKey,
	}
}

func (d *Decoder) BindExpectedCanonicalizer(
	schemas providerapi.SchemaCatalog,
) error {
	if d == nil || schemas == nil {
		return fmt.Errorf(
			"%w: MCP decoder requires a shareable schema registry",
			basespec.ErrInvalid,
		)
	}

	registered := schemas.Keys()
	for _, expected := range d.RequiredSchemaKeys() {
		if slices.Contains(registered, expected) {
			continue
		}
		return fmt.Errorf(
			"%w: MCP Bundle shareable schema is not registered",
			basespec.ErrInvalid,
		)
	}
	d.documents = schemas
	return nil
}

func (d *Decoder) Recognize(
	_ context.Context,
	candidate providerapi.Candidate,
) providerapi.Recognition {
	if candidate.RequestsDecoder(artifactbuiltin.DecoderID) &&
		mcpStore.IsBundleDocumentLocator(candidate.Locator) {
		return providerapi.RecognitionPreferred
	}
	return providerapi.RecognitionNone
}

func (d *Decoder) Decode(
	ctx context.Context,
	candidate providerapi.Candidate,
) ([]providerapi.Decoded, []providerapi.Diagnostic) {
	if !candidate.RequestsDecoder(artifactbuiltin.DecoderID) ||
		!mcpStore.IsBundleDocumentLocator(candidate.Locator) {
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
		artifactbuiltin.MCPBundleSchemaKey,
		candidate.Content,
	)
	if err != nil {
		return nil, decoderError(candidate.Locator, "bundle", err)
	}
	b, err := mcpStore.BundleFromParsedDocument(parsed)
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
		[]providerapi.Decoded,
		0,
		len(serverNames)+len(policyNames),
	)

	for _, name := range serverNames {
		serverDocument, err := mcpStore.ServerFromCanonicalBundle(b, name)
		if err != nil {
			return nil, decoderError(candidate.Locator, name, err)
		}
		definition, err := mcpStoreServer.DefinitionForCanonicalServer(serverDocument)
		if err != nil {
			return nil, decoderError(candidate.Locator, name, err)
		}
		output = append(output, providerapi.Decoded{
			SubresourceLocator: mcpStoreServer.ServerSubresource(
				basespec.LogicalName(name),
			),
			Definition: definition,
		})
	}

	for _, name := range policyNames {
		definition, err := mcpStorePolicy.DefinitionForCanonicalPolicy(
			b.BundleExtension.Policies[name],
		)
		if err != nil {
			return nil, decoderError(candidate.Locator, name, err)
		}
		output = append(output, providerapi.Decoded{
			SubresourceLocator: mcpStorePolicy.PolicySubresource(
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
) []providerapi.Diagnostic {
	return []providerapi.Diagnostic{{
		Severity: providerapi.DiagnosticError,
		Code:     "mcp.mcpStore.subresource-invalid",
		Message: providerapi.BoundedDiagnosticMessage(
			fmt.Sprintf("%s: %v", subresource, err),
		),
		Location: &providerapi.DiagnosticLocation{
			Locator: locator,
		},
	}}
}
