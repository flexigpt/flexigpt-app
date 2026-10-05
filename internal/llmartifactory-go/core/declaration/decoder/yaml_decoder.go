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
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

const YAMLDecoderID spec.DecoderID = "artifact-declaration-yaml"

type YAMLDecoder struct {
	core *canonicalDecoder
}

func NewYAMLDecoder(registry *interpretation.Registry) *YAMLDecoder {
	return &YAMLDecoder{
		core: newCanonicalDecoder(registry),
	}
}

func (*YAMLDecoder) ID() spec.DecoderID {
	return YAMLDecoderID
}

func (*YAMLDecoder) Revision() string {
	return "artifact-declaration-yaml/v1"
}

func (d *YAMLDecoder) RequiredSchemaKeys() []schemaModel.Key {
	if d == nil || d.core == nil {
		return nil
	}
	return d.core.RequiredSchemaKeys()
}

func (d *YAMLDecoder) BindExpectedCanonicalizer(
	catalog schema.Catalog,
) error {
	if d == nil || d.core == nil {
		return fmt.Errorf(
			"%w: canonical YAML declaration decoder is unavailable",
			spec.ErrInvalid,
		)
	}
	return d.core.BindExpectedCanonicalizer(catalog)
}

func (d *YAMLDecoder) Recognize(
	_ context.Context,
	candidate ingestModel.Candidate,
) ingestModel.Recognition {
	requested := candidate.RequestsDecoder(YAMLDecoderID)
	extension := strings.ToLower(path.Ext(string(candidate.Locator)))
	if extension != ".yaml" &&
		extension != ".yml" &&
		!requested {
		return ingestModel.RecognitionNone
	}
	raw, err := yamlutil.CanonicalObjectJSON(
		candidate.Content,
		spec.MaxDefinitionBytes,
	)
	if err != nil {
		if requested {
			return ingestModel.RecognitionPossible
		}
		return ingestModel.RecognitionNone
	}
	var header struct {
		Type declaration.Type `json:"type"`
	}
	if err := json.Unmarshal(raw, &header); err != nil {
		if requested {
			return ingestModel.RecognitionPossible
		}
		return ingestModel.RecognitionNone
	}
	if d == nil || d.core == nil || d.core.interpretations == nil ||
		!d.core.interpretations.SupportsType(header.Type) {
		if requested {
			return ingestModel.RecognitionPossible
		}
		return ingestModel.RecognitionNone
	}
	return ingestModel.RecognitionPreferred
}

func (d *YAMLDecoder) Decode(
	ctx context.Context,
	candidate ingestModel.Candidate,
) ([]ingestModel.Decoded, []diagnostic.Diagnostic) {
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
		spec.MaxDefinitionBytes,
	)
	if err != nil {
		return nil, yamlDiagnostic(candidate.Locator, "", err)
	}
	return d.core.Decode(ctx, candidate, raw)
}

func yamlDiagnostic(
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
		Code:     "artifact.declaration-yaml-invalid",
		Message:  diagnostic.BoundedMessage(err.Error()),
		Location: location,
	}}
}
