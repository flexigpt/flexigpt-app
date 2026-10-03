package providerapi

import (
	"context"
	"fmt"

	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	ingestModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain"
	"github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/sourceformat"
)

type Decoder struct{}

func NewDecoder() *Decoder {
	return &Decoder{}
}

func (*Decoder) ID() spec.DecoderID {
	return mcpDomain.SourceDecoderID
}

func (*Decoder) Revision() string {
	return "mcp-source-decoder-v1"
}

func (*Decoder) Recognize(
	_ context.Context,
	candidate ingestModel.Candidate,
) ingestModel.Recognition {
	switch {
	case sourceformat.IsRetiredMCPCollection(candidate.Content):
		return ingestModel.RecognitionPreferred
	case sourceformat.IsMCPConfig(candidate.Content):
		return ingestModel.RecognitionPreferred
	case isMCPConfigCandidate(candidate):
		return ingestModel.RecognitionPossible
	default:
		return ingestModel.RecognitionNone
	}
}

func isMCPConfigCandidate(
	candidate ingestModel.Candidate,
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
	candidate ingestModel.Candidate,
) ([]ingestModel.Decoded, []diagnostic.Diagnostic) {
	switch {
	case sourceformat.IsRetiredMCPCollection(candidate.Content):
		return nil, decoderError(
			candidate.Locator,
			"",
			fmt.Errorf(
				"%w: proprietary MCP collection manifests are retired; use a canonical type: plugin declaration",
				spec.ErrUnsupported,
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
				spec.ErrInvalid,
			),
		)

	}
	return nil, nil
}

func decodedValues(
	values []sourceformat.Decoded,
) []ingestModel.Decoded {
	output := make([]ingestModel.Decoded, 0, len(values))
	for _, value := range values {
		output = append(output, ingestModel.Decoded{
			SubresourceLocator: value.SubresourceLocator,
			Definition:         value.Definition,
		})
	}
	return output
}

func decoderError(
	locator spec.Locator,
	subresource spec.SubresourceLocator,
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
