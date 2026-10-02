package decoder

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider"
)

const JSONDecoderID model.DecoderID = "artifact-declaration-json"

type JSONDecoder struct {
	core *canonicalDecoder
}

func NewJSONDecoder() *JSONDecoder {
	return &JSONDecoder{
		core: newCanonicalDecoder(),
	}
}

func (*JSONDecoder) ID() model.DecoderID {
	return JSONDecoderID
}

func (*JSONDecoder) Revision() string {
	return "artifact-declaration-json/v1"
}

func (*JSONDecoder) RequiredSchemaKeys() []schema.Key {
	return newCanonicalDecoder().RequiredSchemaKeys()
}

func (d *JSONDecoder) BindExpectedCanonicalizer(
	catalog provider.SchemaCatalog,
) error {
	if d == nil || d.core == nil {
		return fmt.Errorf(
			"%w: canonical JSON declaration decoder is unavailable",
			model.ErrInvalid,
		)
	}
	return d.core.BindExpectedCanonicalizer(catalog)
}

func (*JSONDecoder) Recognize(
	_ context.Context,
	candidate provider.Candidate,
) provider.Recognition {
	requested := candidate.RequestsDecoder(JSONDecoderID)
	declared := documentTopology.IsCanonicalJSONDocument(
		candidate.Locator,
	)
	extension := strings.ToLower(
		path.Ext(string(candidate.Locator)),
	)
	if (extension == ".yaml" || extension == ".yml") && !requested &&
		!declared {
		return provider.RecognitionNone
	}

	var header struct {
		Type declaration.Type `json:"type"`
	}
	if err := json.Unmarshal(candidate.Content, &header); err != nil {
		if requested || declared {
			return provider.RecognitionPossible
		}
		return provider.RecognitionNone
	}
	if !supportsType(header.Type) {
		if requested || declared {
			return provider.RecognitionPossible
		}
		return provider.RecognitionNone
	}
	return provider.RecognitionPreferred
}

func (d *JSONDecoder) Decode(
	ctx context.Context,
	candidate provider.Candidate,
) ([]provider.Decoded, []diagnostic.Diagnostic) {
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
