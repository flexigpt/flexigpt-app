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
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/codec"
)

type canonicalDecoder struct {
	schemas  schema.ExpectedCanonicalizer
	dispatch *declaration.Dispatcher
}

func newCanonicalDecoder() *canonicalDecoder {
	return &canonicalDecoder{}
}

func (*canonicalDecoder) RequiredSchemaKeys() []schemaModel.Key {
	return codec.SchemaKeys()
}

func (d *canonicalDecoder) BindExpectedCanonicalizer(catalog schema.Catalog) error {
	if d == nil || catalog == nil {
		return fmt.Errorf(
			"%w: canonical declaration schema catalog is nil",
			spec.ErrInvalid,
		)
	}

	// Only declaration types recognized by this decoder participate. Unrelated
	// artifact-family codecs do not become declaration-language registrations.
	keys := make([]schemaModel.Key, 0)
	for _, key := range catalog.Keys() {
		if key.Entity == schemaModel.EntityArtifact &&
			supportsType(declaration.Type(key.Kind)) {
			keys = append(keys, key)
		}
	}
	dispatch, err := declaration.NewDispatcher(keys)
	if err != nil {
		return err
	}
	d.schemas = catalog
	d.dispatch = dispatch
	return nil
}

func supportsType(value declaration.Type) bool {
	_, found := codec.SchemaKeyForType(value)
	return found
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

	// Dispatch may observe additional versions in a catalog. That does not
	// authorize this decoder's v1 projection to interpret another contract.
	supported, found := codec.SchemaKeyForType(declaration.Type(key.Kind))
	if !found || supported != key {
		return nil, decodeDiagnostic(
			candidate,
			"artifact.declaration-unsupported-type",
			fmt.Errorf(
				"%w: declaration projection does not support schema %q/%q/%q",
				spec.ErrUnsupported,
				key.Kind,
				key.SchemaID,
				key.SchemaVersion,
			),
		)
	}

	parsed, err := d.schemas.CanonicalizeExpected(ctx, key, raw)
	if err != nil {
		return nil, decodeDiagnostic(candidate, "artifact.declaration-invalid", err)
	}
	root, err := declaration.DecodeCanonicalEntryJSON(parsed.Raw)
	if err != nil {
		return nil, decodeDiagnostic(candidate, "artifact.declaration-invalid", err)
	}
	if err := ValidateEntryTree(root); err != nil {
		return nil, decodeDiagnostic(candidate, "artifact.declaration-nested-invalid", err)
	}
	entries, err := declaration.WalkNamedEntries(root)
	if err != nil {
		return nil, decodeDiagnostic(candidate, "artifact.declaration-nested-invalid", err)
	}

	output := make([]ingestModel.Decoded, 0, len(entries))
	for _, named := range entries {
		value, err := definitionForNamedEntry(named)
		if err != nil {
			output = append(output, ingestModel.Decoded{
				SubresourceLocator: named.SubresourceLocator,
				Diagnostics: decodeDiagnosticAt(
					candidate,
					named.SubresourceLocator,
					"artifact.declaration-nested-invalid",
					err,
				),
			})
			continue
		}
		output = append(output, ingestModel.Decoded{
			SubresourceLocator: named.SubresourceLocator,
			Definition:         value,
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
