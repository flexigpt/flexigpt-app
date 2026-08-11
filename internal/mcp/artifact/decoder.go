package artifact

import (
	"context"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/mcp/schema"
)

type Decoder struct{}

func NewDecoder() *Decoder {
	return &Decoder{}
}

func (*Decoder) ID() basespec.DecoderID {
	return DecoderID
}

func (*Decoder) Revision() string {
	return DecoderRevision
}

func (*Decoder) Recognize(
	_ context.Context,
	candidate discovery.Candidate,
) discovery.Recognition {
	if candidate.RequestsDecoder(DecoderID) &&
		schema.IsBundleDocumentLocator(candidate.Locator) {
		return discovery.RecognitionPreferred
	}
	return discovery.RecognitionNone
}

func (*Decoder) Decode(
	_ context.Context,
	candidate discovery.Candidate,
) ([]discovery.Decoded, []diagnostic.Diagnostic) {
	if !candidate.RequestsDecoder(DecoderID) ||
		!schema.IsBundleDocumentLocator(candidate.Locator) {
		return nil, nil
	}

	bundle, _, err := schema.ParseBundle(candidate.Content)
	if err != nil {
		return nil, []diagnostic.Diagnostic{{
			Severity: diagnostic.DiagnosticError,
			Code:     "mcp.bundle.invalid",
			Message:  diagnostic.BoundedDiagnosticMessage(err.Error()),
			Location: &diagnostic.DiagnosticLocation{
				Locator: candidate.Locator,
			},
		}}
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
