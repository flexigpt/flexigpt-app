package ingest

import (
	"fmt"

	schema "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema"
	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// BindSchemaCatalog validates decoder schema dependencies and supplies the
// narrow expected-canonicalization capability required by bound decoders.
func BindSchemaCatalog(
	decoders []Decoder,
	schemas schema.Catalog,
) error {
	if schemas == nil {
		return fmt.Errorf(
			"%w: schema catalog is nil",
			spec.ErrInvalid,
		)
	}

	available := make(map[schemaModel.Key]struct{})
	for _, key := range schemas.Keys() {
		available[key] = struct{}{}
	}

	for _, decoder := range decoders {
		binder, supported := decoder.(SchemaCanonicalizerBinder)
		if !supported {
			continue
		}

		required := binder.RequiredSchemaKeys()
		seen := make(map[schemaModel.Key]struct{}, len(required))

		for index, key := range required {
			if err := key.Validate(); err != nil {
				return fmt.Errorf(
					"decoder %q required schema %d: %w",
					decoder.ID(),
					index,
					err,
				)
			}
			if _, duplicate := seen[key]; duplicate {
				return fmt.Errorf(
					"%w: decoder %q repeats required schema %q/%q/%q",
					spec.ErrInvalid,
					decoder.ID(),
					key.Kind,
					key.SchemaID,
					key.SchemaVersion,
				)
			}
			seen[key] = struct{}{}

			if _, found := available[key]; !found {
				return fmt.Errorf(
					"%w: decoder %q requires unregistered schema %q/%q/%q",
					spec.ErrInvalid,
					decoder.ID(),
					key.Kind,
					key.SchemaID,
					key.SchemaVersion,
				)
			}
		}

		if err := binder.BindExpectedCanonicalizer(schemas); err != nil {
			return fmt.Errorf(
				"bind schema catalog to decoder %q: %w",
				decoder.ID(),
				err,
			)
		}
	}

	return nil
}
