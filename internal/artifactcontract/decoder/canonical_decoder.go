package decoder

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/codec"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
)

type canonicalDecoder struct {
	schemas providerapi.ExpectedCanonicalizer
}

func newCanonicalDecoder() *canonicalDecoder {
	return &canonicalDecoder{}
}

func (d *canonicalDecoder) RequiredSchemaKeys() []schema.Key {
	return codec.SchemaKeys()
}

func (d *canonicalDecoder) BindExpectedCanonicalizer(
	schemas providerapi.SchemaCatalog,
) error {
	if d == nil || schemas == nil {
		return fmt.Errorf(
			"%w: canonical declaration schema catalog is nil",
			basespec.ErrInvalid,
		)
	}
	d.schemas = schemas
	return nil
}

func supportsType(
	declarationType declaration.Type,
) bool {
	_, found := codec.SchemaKeyForType(declarationType)
	return found
}

func (d *canonicalDecoder) Decode(
	ctx context.Context,
	candidate providerapi.Candidate,
	raw []byte,
) ([]providerapi.Decoded, []diagnostic.Diagnostic) {
	if d == nil || d.schemas == nil {
		return nil, decodeDiagnostic(
			candidate,
			"artifact.declaration-schema-unavailable",
			fmt.Errorf(
				"%w: canonical declaration decoder has no schema catalog",
				basespec.ErrClosed,
			),
		)
	}

	var header struct {
		Type declaration.Type `json:"type"`
	}
	if err := json.Unmarshal(raw, &header); err != nil {
		return nil, decodeDiagnostic(
			candidate,
			"artifact.declaration-invalid",
			err,
		)
	}

	key, found := codec.SchemaKeyForType(header.Type)
	if !found {
		return nil, decodeDiagnostic(
			candidate,
			"artifact.declaration-unsupported-type",
			fmt.Errorf(
				"%w: unsupported declaration type %q",
				basespec.ErrUnsupported,
				header.Type,
			),
		)
	}

	parsed, err := d.schemas.CanonicalizeExpected(
		ctx,
		key,
		raw,
	)
	if err != nil {
		return nil, decodeDiagnostic(
			candidate,
			"artifact.declaration-invalid",
			err,
		)
	}

	root, err := declaration.DecodeCanonicalEntryJSON(parsed.Raw)
	if err != nil {
		return nil, decodeDiagnostic(
			candidate,
			"artifact.declaration-invalid",
			err,
		)
	}
	if err := ValidateEntryTree(root); err != nil {
		return nil, decodeDiagnostic(
			candidate,
			"artifact.declaration-nested-invalid",
			err,
		)
	}
	entries, err := declaration.WalkNamedEntries(root)
	if err != nil {
		return nil, decodeDiagnostic(
			candidate,
			"artifact.declaration-nested-invalid",
			err,
		)
	}

	output := make([]providerapi.Decoded, 0, len(entries))
	for _, named := range entries {

		value, err := definitionForNamedEntry(named)
		if err != nil {
			output = append(output, providerapi.Decoded{
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
		decoded := providerapi.Decoded{
			SubresourceLocator: named.SubresourceLocator,
			Definition:         value,
		}

		output = append(output, decoded)
	}
	return output, nil
}

func decodeDiagnostic(
	candidate providerapi.Candidate,
	code string,
	err error,
) []diagnostic.Diagnostic {
	return decodeDiagnosticAt(candidate, "", code, err)
}

func decodeDiagnosticAt(
	candidate providerapi.Candidate,
	subresource basespec.SubresourceLocator,
	code string,
	err error,
) []diagnostic.Diagnostic {
	location := &diagnostic.Location{
		Locator: candidate.Locator,
	}
	if subresource != "" {
		location.SubresourceLocator = subresource
	}
	return []diagnostic.Diagnostic{{
		Severity: diagnostic.SeverityError,
		Code:     code,
		Message:  diagnostic.BoundedMessage(err.Error()),
		Location: location,
	}}
}
