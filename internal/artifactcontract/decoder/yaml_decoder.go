package decoder

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/api/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/schema"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

const YAMLDecoderID model.DecoderID = "artifact-declaration-yaml"

type YAMLDecoder struct {
	core *canonicalDecoder
}

func NewYAMLDecoder() *YAMLDecoder {
	return &YAMLDecoder{
		core: newCanonicalDecoder(),
	}
}

func (*YAMLDecoder) ID() model.DecoderID {
	return YAMLDecoderID
}

func (*YAMLDecoder) Revision() string {
	return "artifact-declaration-yaml/v1"
}

func (*YAMLDecoder) RequiredSchemaKeys() []schema.Key {
	return newCanonicalDecoder().RequiredSchemaKeys()
}

func (d *YAMLDecoder) BindExpectedCanonicalizer(
	catalog providerapi.SchemaCatalog,
) error {
	if d == nil || d.core == nil {
		return fmt.Errorf(
			"%w: canonical YAML declaration decoder is unavailable",
			model.ErrInvalid,
		)
	}
	return d.core.BindExpectedCanonicalizer(catalog)
}

func (*YAMLDecoder) Recognize(
	_ context.Context,
	candidate providerapi.Candidate,
) providerapi.Recognition {
	requested := candidate.RequestsDecoder(YAMLDecoderID)
	declared := documentTopology.IsCanonicalYAMLDocument(
		candidate.Locator,
	)
	extension := strings.ToLower(path.Ext(string(candidate.Locator)))
	if extension != ".yaml" &&
		extension != ".yml" &&
		!requested &&
		!declared {
		return providerapi.RecognitionNone
	}
	raw, err := yamlutil.CanonicalObjectJSON(
		candidate.Content,
		model.MaxDefinitionBytes,
	)
	if err != nil {
		if requested || declared {
			return providerapi.RecognitionPossible
		}
		return providerapi.RecognitionNone
	}
	var header struct {
		Type declaration.Type `json:"type"`
	}
	if err := json.Unmarshal(raw, &header); err != nil {
		if requested || declared {
			return providerapi.RecognitionPossible
		}
		return providerapi.RecognitionNone
	}
	if !supportsType(header.Type) {
		if requested || declared {
			return providerapi.RecognitionPossible
		}
		return providerapi.RecognitionNone
	}
	return providerapi.RecognitionPreferred
}

func (d *YAMLDecoder) Decode(
	ctx context.Context,
	candidate providerapi.Candidate,
) ([]providerapi.Decoded, []diagnostic.Diagnostic) {
	if d == nil || d.core == nil {
		return nil, []diagnostic.Diagnostic{{
			Severity: diagnostic.SeverityError,
			Code:     "artifact.declaration-schema-unavailable",
			Message:  "canonical YAML declaration decoder is unavailable",
			Location: &diagnostic.Location{Locator: candidate.Locator},
		}}
	}
	raw, err := yamlutil.CanonicalObjectJSON(
		candidate.Content,
		model.MaxDefinitionBytes,
	)
	if err != nil {
		return nil, yamlDiagnostic(candidate.Locator, "", err)
	}
	return d.core.Decode(ctx, candidate, raw)
}

func yamlDiagnostic(
	locator model.Locator,
	subresource model.SubresourceLocator,
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
		Code:     "artifact.declaration-yaml-invalid",
		Message:  diagnostic.BoundedMessage(err.Error()),
		Location: location,
	}}
}
