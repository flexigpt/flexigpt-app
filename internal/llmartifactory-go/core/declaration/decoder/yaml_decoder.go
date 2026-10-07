package decoder

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema"
	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	ingestModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/support"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

const YAMLDecoderID spec.DecoderID = "artifact-declaration-yaml"

type YAMLDecoder struct {
	core       *canonicalDecoder
	candidates support.Candidates
}

func NewYAMLDecoder(
	registry *coreinterpretation.Registry,
	candidates support.Candidates,
) *YAMLDecoder {
	return &YAMLDecoder{
		core:       newCanonicalDecoder(registry, nil),
		candidates: candidates.Clone(),
	}
}

// NewYAMLDecoderForSchemaKeys constructs a canonical YAML decoder that
// dispatches only the supplied complete schema keys while retaining the full
// registry for family semantic reconstruction.
func NewYAMLDecoderForSchemaKeys(
	registry *coreinterpretation.Registry,
	keys []schemaModel.Key,
	candidates support.Candidates,
) (*YAMLDecoder, error) {
	core := newCanonicalDecoder(registry, keys)
	if core.selectionErr != nil {
		return nil, core.selectionErr
	}
	if err := candidates.Validate(); err != nil {
		return nil, err
	}
	return &YAMLDecoder{
		core:       core,
		candidates: candidates.Clone(),
	}, nil
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
	if !requested && !d.candidates.Matches(candidate.Locator) {
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
	if d == nil || d.core == nil || !d.core.SupportsType(header.Type) {
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
