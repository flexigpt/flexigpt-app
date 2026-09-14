package decoder

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
)

const JSONDecoderID basespec.DecoderID = "artifact-declaration-json"

type JSONDecoder struct {
	core *canonicalDecoder
}

func NewJSONDecoder() *JSONDecoder {
	return &JSONDecoder{
		core: newCanonicalDecoder(),
	}
}

func (*JSONDecoder) ID() basespec.DecoderID {
	return JSONDecoderID
}

func (*JSONDecoder) Revision() string {
	return "artifact-declaration-json/v2"
}

func (*JSONDecoder) RequiredSchemaKeys() []schema.Key {
	return newCanonicalDecoder().RequiredSchemaKeys()
}

func (d *JSONDecoder) BindExpectedCanonicalizer(
	catalog providerapi.SchemaCatalog,
) error {
	if d == nil || d.core == nil {
		return fmt.Errorf(
			"%w: canonical JSON declaration decoder is unavailable",
			basespec.ErrInvalid,
		)
	}
	return d.core.BindExpectedCanonicalizer(catalog)
}

func (*JSONDecoder) Recognize(
	_ context.Context,
	candidate providerapi.Candidate,
) providerapi.Recognition {
	requested := candidate.RequestsDecoder(JSONDecoderID)
	extension := strings.ToLower(
		path.Ext(string(candidate.Locator)),
	)
	if (extension == ".yaml" || extension == ".yml") &&
		!requested {
		return providerapi.RecognitionNone
	}

	var header struct {
		Type declaration.Type `json:"type"`
	}
	if err := json.Unmarshal(candidate.Content, &header); err != nil {
		if requested {
			return providerapi.RecognitionPreferred
		}
		return providerapi.RecognitionNone
	}
	if !supportsType(header.Type) {
		if requested {
			return providerapi.RecognitionPreferred
		}
		return providerapi.RecognitionNone
	}
	return providerapi.RecognitionPreferred
}

func (d *JSONDecoder) Decode(
	ctx context.Context,
	candidate providerapi.Candidate,
) ([]providerapi.Decoded, []diagnostic.Diagnostic) {
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
