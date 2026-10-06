package llmartifactory

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/compose"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema"
	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
)

func validateRegistrationSelection(
	store *compose.Store,
	codecs []schema.Codec,
	decoders []ingest.Decoder,
	interpretations *coreinterpretation.Registry,
) error {
	if store == nil || store.Schemas == nil {
		return fmt.Errorf(
			"%w: generic Store schema catalog is unavailable",
			spec.ErrInvalid,
		)
	}
	if interpretations == nil {
		return fmt.Errorf(
			"%w: LLM declaration interpretation registry is nil",
			spec.ErrInvalid,
		)
	}

	keyReader, supported := store.Schemas.(schema.KeyReader)
	if !supported {
		return fmt.Errorf(
			"%w: generic Store schema catalog does not expose registered keys",
			spec.ErrInvalid,
		)
	}

	registered := make(map[schemaModel.Key]struct{})
	for _, key := range keyReader.Keys() {
		if err := key.Validate(); err != nil {
			return fmt.Errorf(
				"%w: generic Store returned an invalid schema key: %w",
				spec.ErrInvalid,
				err,
			)
		}
		registered[key] = struct{}{}
	}

	expected := make(map[schemaModel.Key]struct{})
	for _, key := range interpretations.SchemaKeys() {
		expected[key] = struct{}{}
	}

	selected := make(map[schemaModel.Key]struct{}, len(codecs))
	for _, codec := range codecs {
		key := codec.Key()
		if _, found := expected[key]; !found {
			return fmt.Errorf(
				"%w: selected schema %q/%q/%q has no family interpretation",
				spec.ErrInvalid,
				key.Kind,
				key.SchemaID,
				key.SchemaVersion,
			)
		}
		if _, found := registered[key]; !found {
			return fmt.Errorf(
				"%w: selected schema %q/%q/%q is absent from the generic Store",
				spec.ErrInvalid,
				key.Kind,
				key.SchemaID,
				key.SchemaVersion,
			)
		}
		selected[key] = struct{}{}
	}

	for key := range expected {
		if _, found := selected[key]; found {
			continue
		}
		return fmt.Errorf(
			"%w: family interpretation schema %q/%q/%q is not selected",
			spec.ErrInvalid,
			key.Kind,
			key.SchemaID,
			key.SchemaVersion,
		)
	}

	for _, decoder := range decoders {
		binder, supported := decoder.(ingest.SchemaCanonicalizerBinder)
		if !supported {
			continue
		}
		for _, key := range binder.RequiredSchemaKeys() {
			if _, found := selected[key]; found {
				continue
			}
			return fmt.Errorf(
				"%w: decoder %q requires an unselected schema %q/%q/%q",
				spec.ErrInvalid,
				decoder.ID(),
				key.Kind,
				key.SchemaID,
				key.SchemaVersion,
			)
		}
	}

	return nil
}
