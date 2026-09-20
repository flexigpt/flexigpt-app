package providerapi

import (
	"context"
	"fmt"

	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain"
	"github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/sourceformat"
)

type Decoder struct{}

func NewDecoder() *Decoder {
	return &Decoder{}
}

func (*Decoder) ID() basespec.DecoderID {
	return mcpDomain.SourceDecoderID
}

func (*Decoder) Revision() string {
	return "mcp-source-decoder-v1"
}

func (*Decoder) Recognize(
	_ context.Context,
	candidate providerapi.Candidate,
) providerapi.Recognition {
	switch {
	case sourceformat.IsRetiredMCPCollection(candidate.Content):
		return providerapi.RecognitionPreferred
	case sourceformat.IsMCPConfig(candidate.Content):
		return providerapi.RecognitionPreferred
	case isMCPConfigCandidate(candidate):
		return providerapi.RecognitionPossible
	default:
		return providerapi.RecognitionNone
	}
}

func isMCPConfigCandidate(
	candidate providerapi.Candidate,
) bool {
	if candidate.RequestsDecoder(mcpDomain.SourceDecoderID) {
		return true
	}
	return documentTopology.IsMCPConfigDocument(
		candidate.Locator,
	)
}

func (d *Decoder) Decode(
	ctx context.Context,
	candidate providerapi.Candidate,
) ([]providerapi.Decoded, []diagnostic.Diagnostic) {
	switch {
	case sourceformat.IsRetiredMCPCollection(candidate.Content):
		return nil, decoderError(
			candidate.Locator,
			"",
			fmt.Errorf(
				"%w: proprietary MCP collection manifests are retired; use a canonical type: plugin declaration",
				basespec.ErrUnsupported,
			),
		)

	case sourceformat.IsMCPConfig(candidate.Content):
		values, err := sourceformat.DecodeMCPConfig(
			candidate.Content,
		)
		if err != nil {
			return nil, decoderError(candidate.Locator, "", err)
		}
		return decodedValues(values), nil

	case isMCPConfigCandidate(candidate):
		return nil, decoderError(
			candidate.Locator,
			"",
			fmt.Errorf(
				"%w: MCP configuration requires mcpServers",
				basespec.ErrInvalid,
			),
		)

	}
	return nil, nil
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
