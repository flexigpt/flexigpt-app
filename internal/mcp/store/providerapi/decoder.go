package providerapi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/mcppolicyv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/mcpv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain"
	mcpDomainPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/policy"
	"github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/sourceformat"
)

type Decoder struct {
	documents providerapi.ExpectedCanonicalizer
}

func NewDecoder() *Decoder {
	return &Decoder{}
}

func (*Decoder) ID() basespec.DecoderID {
	return mcpDomain.CanonicalDecoderID
}

func (*Decoder) Revision() string {
	return "mcp-source-decoder-v2"
}

func (*Decoder) RequiredSchemaKeys() []schema.Key {
	return []schema.Key{
		mcpv1.MCPSchemaKey,
		mcppolicyv1.MCPPolicySchemaKey,
	}
}

func (d *Decoder) BindExpectedCanonicalizer(
	documents providerapi.SchemaCatalog,
) error {
	if d == nil || documents == nil {
		return fmt.Errorf(
			"%w: MCP decoder schema catalog is nil",
			basespec.ErrInvalid,
		)
	}
	d.documents = documents
	return nil
}

func (*Decoder) Recognize(
	_ context.Context,
	candidate providerapi.Candidate,
) providerapi.Recognition {
	var header struct {
		Type artifactcontract.Type `json:"type"`
		Kind string                `json:"kind"`
	}
	if err := json.Unmarshal(candidate.Content, &header); err != nil {
		return providerapi.RecognitionNone
	}

	switch {
	case header.Type == artifactcontract.TypeMCP,
		header.Type == artifactcontract.TypeMCPPolicy:
		return providerapi.RecognitionPreferred
	case header.Kind == "mcp.bundle":
		return providerapi.RecognitionPreferred
	case sourceformat.IsMCPConfig(candidate.Content):
		return providerapi.RecognitionPossible
	default:
		return providerapi.RecognitionNone
	}
}

func (d *Decoder) Decode(
	ctx context.Context,
	candidate providerapi.Candidate,
) ([]providerapi.Decoded, []diagnostic.Diagnostic) {
	if d == nil || d.documents == nil {
		return nil, decoderError(
			candidate.Locator,
			"",
			fmt.Errorf(
				"%w: MCP decoder has no bound schema catalog",
				basespec.ErrClosed,
			),
		)
	}

	var header struct {
		Type artifactcontract.Type `json:"type"`
		Kind string                `json:"kind"`
	}
	if err := json.Unmarshal(candidate.Content, &header); err != nil {
		return nil, nil
	}

	switch {
	case header.Type == artifactcontract.TypeMCP:
		parsed, err := d.documents.CanonicalizeExpected(
			ctx,
			mcpv1.MCPSchemaKey,
			candidate.Content,
		)
		if err != nil {
			return nil, decoderError(candidate.Locator, "", err)
		}
		document, err := mcpv1.DecodeMCPJSON(parsed.Raw)
		if err != nil {
			return nil, decoderError(candidate.Locator, "", err)
		}
		definitionValue, err := sourceformat.MCPDocumentFromCanonical(
			document,
		)
		if err != nil {
			return nil, decoderError(candidate.Locator, "", err)
		}
		return []providerapi.Decoded{{
			Definition: definitionValue,
		}}, nil

	case header.Type == artifactcontract.TypeMCPPolicy:
		parsed, err := d.documents.CanonicalizeExpected(
			ctx,
			mcppolicyv1.MCPPolicySchemaKey,
			candidate.Content,
		)
		if err != nil {
			return nil, decoderError(candidate.Locator, "", err)
		}
		document, err := mcppolicyv1.DecodeMCPPolicyJSON(parsed.Raw)
		if err != nil {
			return nil, decoderError(candidate.Locator, "", err)
		}
		definitionValue, err := mcpDomainPolicy.DefinitionForDocument(
			document,
		)
		if err != nil {
			return nil, decoderError(candidate.Locator, "", err)
		}
		return []providerapi.Decoded{{
			Definition: definitionValue,
		}}, nil

	case header.Kind == "mcp.bundle":
		values, err := sourceformat.DecodeLegacyBundle(
			candidate.Content,
		)
		if err != nil {
			return nil, decoderError(candidate.Locator, "", err)
		}
		return decodedValues(values), nil

	case sourceformat.IsMCPConfig(candidate.Content):
		values, err := sourceformat.DecodeMCPConfig(
			candidate.Content,
		)
		if err != nil {
			return nil, decoderError(candidate.Locator, "", err)
		}
		return decodedValues(values), nil

	default:
		return nil, nil
	}
}

func decodedValues(
	values []sourceformat.Decoded,
) []providerapi.Decoded {
	output := make([]providerapi.Decoded, 0, len(values))
	for _, value := range values {
		output = append(output, providerapi.Decoded{
			SubresourceLocator: value.SubresourceLocator,
			Definition:         value.Definition,
		})
	}
	return output
}

func decoderError(
	locator basespec.Locator,
	subresource basespec.SubresourceLocator,
	err error,
) []diagnostic.Diagnostic {
	location := &diagnostic.Location{
		Locator: locator,
	}
	if subresource != "" {
		location.SubresourceLocator = subresource
	}
	return []diagnostic.Diagnostic{{
		Severity: diagnostic.SeverityError,
		Code:     "mcp.source-invalid",
		Message:  diagnostic.BoundedMessage(err.Error()),
		Location: location,
	}}
}
