package decoder

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema"
	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	ingestModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
)

type canonicalDecoder struct {
	schemas         schema.ExpectedCanonicalizer
	dispatch        *declaration.Dispatcher
	interpretations *interpretation.Registry
}

func newCanonicalDecoder(
	interpretations *interpretation.Registry,
) *canonicalDecoder {
	return &canonicalDecoder{
		interpretations: interpretations,
	}
}

func (d *canonicalDecoder) RequiredSchemaKeys() []schemaModel.Key {
	if d == nil || d.interpretations == nil {
		return nil
	}
	return d.interpretations.SchemaKeys()
}

func (d *canonicalDecoder) BindExpectedCanonicalizer(catalog schema.Catalog) error {
	if d == nil || d.interpretations == nil || catalog == nil {
		return fmt.Errorf(
			"%w: canonical declaration schema catalog is nil",
			spec.ErrInvalid,
		)
	}
	dispatch, err := declaration.NewDispatcher(
		d.interpretations.SchemaKeys(),
	)
	if err != nil {
		return err
	}
	d.schemas = catalog
	d.dispatch = dispatch
	return nil
}

func (d *canonicalDecoder) Decode(
	ctx context.Context,
	candidate ingestModel.Candidate,
	raw []byte,
) ([]ingestModel.Decoded, []diagnostic.Diagnostic) {
	if d == nil || d.schemas == nil || d.dispatch == nil {
		return nil, decodeDiagnostic(
			candidate,
			"artifact.declaration-schema-unavailable",
			fmt.Errorf("%w: canonical declaration decoder is not bound", spec.ErrClosed),
		)
	}

	key, err := d.dispatch.Resolve(raw)
	if err != nil {
		return nil, decodeDiagnostic(candidate, "artifact.declaration-invalid", err)
	}

	document, err := declaration.CanonicalDocumentForSchema(raw)
	if err != nil {
		return nil, decodeDiagnostic(
			candidate,
			"artifact.declaration-invalid",
			err,
		)
	}

	parsed, err := d.schemas.CanonicalizeExpected(
		ctx,
		key,
		document,
	)
	if err != nil {
		return nil, decodeDiagnostic(candidate, "artifact.declaration-invalid", err)
	}

	entries, err := d.interpretations.DefinitionsForSchemaValidatedDocument(
		parsed.Key,
		parsed.Raw,
	)
	if err != nil {
		return nil, decodeDiagnostic(candidate, "artifact.declaration-nested-invalid", err)
	}

	output := make([]ingestModel.Decoded, 0, len(entries))
	for _, named := range entries {
		output = append(output, ingestModel.Decoded{
			SubresourceLocator: named.SubresourceLocator,
			Definition:         named.Definition,
		})
	}
	return output, nil
}

func decodeDiagnostic(
	candidate ingestModel.Candidate,
	code string,
	err error,
) []diagnostic.Diagnostic {
	return decodeDiagnosticAt(candidate, "", code, err)
}

func decodeDiagnosticAt(
	candidate ingestModel.Candidate,
	subresource spec.SubresourceLocator,
	code string,
	err error,
) []diagnostic.Diagnostic {
	return []diagnostic.Diagnostic{{
		Severity: diagnostic.SeverityError,
		Code:     code,
		Message:  diagnostic.BoundedMessage(err.Error()),
		Location: &diagnostic.Location{
			Locator:            candidate.Locator,
			SubresourceLocator: subresource,
		},
	}}
}
