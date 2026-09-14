package providerapi

import (
	"context"
	"encoding/json"

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
	return mcpDomain.CanonicalDecoderID
}

func (*Decoder) Revision() string {
	return "mcp-source-decoder-v3"
}

func (*Decoder) Recognize(
	_ context.Context,
	candidate providerapi.Candidate,
) providerapi.Recognition {
	var header struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(candidate.Content, &header); err != nil {
		return providerapi.RecognitionNone
	}

	switch {

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
	var header struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(candidate.Content, &header); err != nil {
		return nil, nil
	}

	switch {

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
