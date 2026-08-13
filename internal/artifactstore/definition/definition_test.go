package definition

import (
	"errors"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

func TestCanonicalizeIsDeterministicAndOwnsMutableFields(t *testing.T) {
	t.Parallel()

	input := definitionTestValue()
	canonical, err := Canonicalize(input)
	if err != nil {
		t.Fatalf("Canonicalize: %v", err)
	}
	if err := canonical.Validate(); err != nil {
		t.Fatalf("canonical validation: %v", err)
	}
	if string(canonical.Body) != `{"a":1,"z":2}` {
		t.Fatalf("canonical body=%s", canonical.Body)
	}

	input.Labels["a-label"] = "changed"
	input.Dependencies[0].Labels["scope"] = "changed"
	input.Body[2] = 'x'

	if canonical.Labels["a-label"] != "first" ||
		canonical.Dependencies[0].Labels["scope"] != "test" ||
		string(canonical.Body) != `{"a":1,"z":2}` {
		t.Fatalf("canonical definition retained caller-owned data: %#v", canonical)
	}

	reordered := definitionTestValue()
	reordered.Labels = map[string]string{
		"a-label": "first",
		"z-label": "last",
	}
	reordered.Body = []byte(` { "a" : 1.0, "z" : 2e0 } `)

	second, err := Canonicalize(reordered)
	if err != nil {
		t.Fatalf("Canonicalize reordered: %v", err)
	}
	if second.Digest != canonical.Digest {
		t.Fatalf(
			"digest=%q, want deterministic %q",
			second.Digest,
			canonical.Digest,
		)
	}

	wrong := definitionTestValue()
	wrong.Digest = cryptoutil.DigestBytes([]byte("not this definition"))
	if _, err := Canonicalize(wrong); !errors.Is(
		err,
		basespec.ErrDigestMismatch,
	) {
		t.Fatalf(
			"mismatched supplied digest error=%v, want ErrDigestMismatch",
			err,
		)
	}
}

func TestDefinitionCloneOwnsMutableFields(t *testing.T) {
	t.Parallel()

	canonical, err := Canonicalize(definitionTestValue())
	if err != nil {
		t.Fatalf("canonical fixture: %v", err)
	}
	cloned := canonical.Clone()

	canonical.Labels["a-label"] = "changed"
	canonical.Dependencies[0].Labels["scope"] = "changed"
	canonical.Body[2] = 'x'

	if cloned.Labels["a-label"] != "first" ||
		cloned.Dependencies[0].Labels["scope"] != "test" ||
		string(cloned.Body) != `{"a":1,"z":2}` {
		t.Fatalf("Definition.Clone retained mutable state: %#v", cloned)
	}
}

func definitionTestValue() Definition {
	return Definition{
		Kind:           "test.artifact",
		SchemaID:       "test.schema",
		SchemaVersion:  "v1",
		LogicalName:    "example",
		LogicalVersion: "1",
		DisplayName:    "Example",
		Description:    "A canonical test definition",
		Labels: map[string]string{
			"z-label": "last",
			"a-label": "first",
		},
		Body: []byte(`{"z":2,"a":1}`),
		Dependencies: []Selector{{
			Kind:        "test.dependency",
			LogicalName: "dependency",
			Labels:      map[string]string{"scope": "test"},
		}},
	}
}
