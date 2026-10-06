package decoder

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema"
	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	ingestModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
)

const JSONDecoderID spec.DecoderID = "artifact-declaration-json"

type JSONDecoder struct {
	core *canonicalDecoder
}

func NewJSONDecoder(registry *coreinterpretation.Registry) *JSONDecoder {
	return &JSONDecoder{
		core: newCanonicalDecoder(registry, nil),
	}
}

// NewJSONDecoderForSchemaKeys constructs a canonical JSON decoder that
// dispatches only the supplied complete schema keys while retaining the full
// registry for family semantic reconstruction.
func NewJSONDecoderForSchemaKeys(
	registry *coreinterpretation.Registry,
	keys []schemaModel.Key,
) (*JSONDecoder, error) {
	core := newCanonicalDecoder(registry, keys)
	if core.selectionErr != nil {
		return nil, core.selectionErr
	}
	return &JSONDecoder{core: core}, nil
}

func (*JSONDecoder) ID() spec.DecoderID {
	return JSONDecoderID
}

func (*JSONDecoder) Revision() string {
	return "artifact-declaration-json/v1"
}

func (d *JSONDecoder) RequiredSchemaKeys() []schemaModel.Key {
	if d == nil || d.core == nil {
		return nil
	}
	return d.core.RequiredSchemaKeys()
}

func (d *JSONDecoder) BindExpectedCanonicalizer(
	catalog schema.Catalog,
) error {
	if d == nil || d.core == nil {
		return fmt.Errorf(
			"%w: canonical JSON declaration decoder is unavailable",
			spec.ErrInvalid,
		)
	}
	return d.core.BindExpectedCanonicalizer(catalog)
}

func (d *JSONDecoder) Recognize(
	_ context.Context,
	candidate ingestModel.Candidate,
) ingestModel.Recognition {
	requested := candidate.RequestsDecoder(JSONDecoderID)
	extension := strings.ToLower(
		path.Ext(string(candidate.Locator)),
	)
	if extension != ".json" && !requested {
		return ingestModel.RecognitionNone
	}

	var header struct {
		Type declaration.Type `json:"type"`
	}
	if err := json.Unmarshal(candidate.Content, &header); err != nil {
		if requested {
			return ingestModel.RecognitionPossible
		}
		return ingestModel.RecognitionNone
	}
	if d == nil || d.core == nil || !d.core.SupportsType(header.Type) {
		if requested {
			return ingestModel.RecognitionPossible
		}
		return ingestModel.RecognitionNone
	}
	return ingestModel.RecognitionPreferred
}

func (d *JSONDecoder) Decode(
	ctx context.Context,
	candidate ingestModel.Candidate,
) ([]ingestModel.Decoded, []diagnostic.Diagnostic) {
	if d == nil || d.core == nil {
		return nil, []diagnostic.Diagnostic{{
			Severity: diagnostic.SeverityError,
			Code:     "artifact.declaration-schema-unavailable",
			Message:  "canonical JSON declaration decoder is unavailable",
			Location: &diagnostic.Location{Locator: candidate.Locator},
		}}
	}
	return d.core.Decode(
		ctx,
		candidate,
		candidate.Content,
	)
}
