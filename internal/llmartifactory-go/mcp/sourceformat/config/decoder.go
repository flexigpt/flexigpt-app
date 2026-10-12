package sourceformat

import (
	"context"
	"fmt"

	ingestModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/support"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/domain"
)

type Decoder struct {
	candidates support.Candidates
}

func NewDecoder(
	candidates support.Candidates,
) *Decoder {
	return &Decoder{
		candidates: candidates.Clone(),
	}
}

func (*Decoder) ID() storeSpec.DecoderID {
	return mcpDomain.SourceDecoderID
}

func (*Decoder) Revision() string {
	return "mcp-source-decoder-v1"
}

func (d *Decoder) Recognize(
	_ context.Context,
	candidate ingestModel.Candidate,
) ingestModel.Recognition {
	switch {
	case IsMCPConfig(candidate.Content):
		return ingestModel.RecognitionPreferred
	case d.isMCPConfigCandidate(candidate):
		return ingestModel.RecognitionPossible
	default:
		return ingestModel.RecognitionNone
	}
}

func (d *Decoder) Decode(
	ctx context.Context,
	candidate ingestModel.Candidate,
) ([]ingestModel.Decoded, []diagnostic.Diagnostic) {
	switch {
	case IsMCPConfig(candidate.Content):
		values, err := DecodeMCPConfig(
			candidate.Content,
		)
		if err != nil {
			return nil, decoderError(candidate.Locator, "", err)
		}
		return decodedValues(values), nil

	case d.isMCPConfigCandidate(candidate):
		return nil, decoderError(
			candidate.Locator,
			"",
			fmt.Errorf(
				"%w: MCP configuration requires mcpServers",
				storeSpec.ErrInvalid,
			),
		)

	}
	return nil, nil
}

func (d *Decoder) isMCPConfigCandidate(
	candidate ingestModel.Candidate,
) bool {
	if candidate.RequestsDecoder(mcpDomain.SourceDecoderID) {
		return true
	}
	return d.candidates.Matches(candidate.Locator)
}

func decodedValues(
	values []Decoded,
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
	locator storeSpec.Locator,
	subresource storeSpec.SubresourceLocator,
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
