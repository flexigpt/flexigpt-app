package codec

import (
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
)

func TestAllSchemaCodecsCoverSchemaKeys(t *testing.T) {
	t.Parallel()

	expected := SchemaKeys()
	codecs := AllSchemaCodecs()

	if len(codecs) != len(expected) {
		t.Fatalf(
			"codec count = %d, want %d",
			len(codecs),
			len(expected),
		)
	}

	expectedSet := make(map[schema.Key]struct{}, len(expected))
	for _, key := range expected {
		if err := key.Validate(); err != nil {
			t.Fatalf("invalid expected schema key %+v: %v", key, err)
		}
		if _, duplicate := expectedSet[key]; duplicate {
			t.Fatalf("duplicate expected schema key %+v", key)
		}
		expectedSet[key] = struct{}{}
	}

	seen := make(map[schema.Key]struct{}, len(codecs))
	for index, value := range codecs {
		if value == nil {
			t.Fatalf("codec %d is nil", index)
		}

		key := value.Key()
		if err := key.Validate(); err != nil {
			t.Fatalf("codec %d has invalid key %+v: %v", index, key, err)
		}
		if _, expected := expectedSet[key]; !expected {
			t.Fatalf("codec %d has unexpected key %+v", index, key)
		}
		if _, duplicate := seen[key]; duplicate {
			t.Fatalf("duplicate codec key %+v", key)
		}
		if len(value.JSONSchema()) == 0 {
			t.Fatalf("codec %d has empty JSON Schema", index)
		}
		seen[key] = struct{}{}
	}

	for _, key := range expected {
		if _, found := seen[key]; !found {
			t.Fatalf("missing codec for schema key %+v", key)
		}
	}
}
