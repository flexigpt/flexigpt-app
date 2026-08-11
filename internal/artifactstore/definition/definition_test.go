package definition

import (
	"context"
	"errors"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

const definitionTestRootID basespec.RootID = "019d3150-6a15-7a6b-a34e-d9032342bc31"

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
	if canonical.Labels["a-label"] != "first" || canonical.Dependencies[0].Labels["scope"] != "test" ||
		string(canonical.Body) != `{"a":1,"z":2}` {
		t.Fatalf("canonical definition retained caller-owned data: %#v", canonical)
	}

	reordered := definitionTestValue()
	reordered.Labels = map[string]string{"a-label": "first", "z-label": "last"}
	reordered.Body = []byte(` { "a" : 1.0, "z" : 2e0 } `)
	second, err := Canonicalize(reordered)
	if err != nil {
		t.Fatalf("Canonicalize reordered: %v", err)
	}
	if second.Digest != canonical.Digest {
		t.Fatalf("digest=%q, want deterministic %q", second.Digest, canonical.Digest)
	}

	wrong := definitionTestValue()
	wrong.Digest = cryptoutil.DigestBytes([]byte("not this definition"))
	if _, err := Canonicalize(wrong); !errors.Is(err, basespec.ErrDigestMismatch) {
		t.Fatalf("mismatched supplied digest error=%v, want ErrDigestMismatch", err)
	}
}

type definitionTestReader struct {
	value Definition
	err   error
	calls int
}

func (r *definitionTestReader) Get(
	context.Context,
	basespec.RootID,
	cryptoutil.Digest,
) (Definition, error) {
	r.calls++
	return r.value, r.err
}

type definitionTestRepository struct {
	definitionTestReader

	putValue Definition
	putErr   error
	putCalls int
}

func (r *definitionTestRepository) Put(
	ctx context.Context,
	rootID basespec.RootID,
	value Definition,
) (Definition, error) {
	r.putCalls++
	if r.putErr != nil {
		return Definition{}, r.putErr
	}
	if r.putValue.Digest == "" {
		return value, nil
	}
	return r.putValue, nil
}

func TestReadCanonicalAndRootScopedRepositoryEnforceIntegrity(t *testing.T) {
	t.Parallel()

	stored, err := Canonicalize(definitionTestValue())
	if err != nil {
		t.Fatalf("canonical fixture: %v", err)
	}
	reader := &definitionTestReader{value: stored}
	got, err := ReadCanonical(t.Context(), reader, definitionTestRootID, stored.Digest)
	if err != nil {
		t.Fatalf("ReadCanonical: %v", err)
	}
	if got.Digest != stored.Digest || reader.calls != 1 {
		t.Fatalf("ReadCanonical result=%#v calls=%d", got, reader.calls)
	}

	wrong := stored
	wrong.Body = []byte(`{"different":true}`)
	reader.value = wrong
	if _, err := ReadCanonical(
		t.Context(),
		reader,
		definitionTestRootID,
		stored.Digest,
	); !errors.Is(
		err,
		basespec.ErrDigestMismatch,
	) {
		t.Fatalf("wrong reader result error=%v, want ErrDigestMismatch", err)
	}

	repository := &definitionTestRepository{definitionTestReader: definitionTestReader{value: stored}}
	validated := 0
	scoped, err := NewRootScopedRepository(repository, func(_ context.Context, rootID basespec.RootID) error {
		validated++
		if rootID != definitionTestRootID {
			t.Fatalf("validator root=%q", rootID)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("NewRootScopedRepository: %v", err)
	}
	if _, err := scoped.Get(t.Context(), definitionTestRootID, stored.Digest); err != nil {
		t.Fatalf("scoped Get: %v", err)
	}
	written, err := scoped.Put(t.Context(), definitionTestRootID, definitionTestValue())
	if err != nil {
		t.Fatalf("scoped Put: %v", err)
	}
	if written.Digest != stored.Digest || repository.putCalls != 1 || validated != 2 {
		t.Fatalf(
			"scoped repository state: written=%q putCalls=%d validations=%d",
			written.Digest,
			repository.putCalls,
			validated,
		)
	}

	deniedRepository := &definitionTestRepository{}
	denied, err := NewRootScopedRepository(deniedRepository, func(context.Context, basespec.RootID) error {
		return basespec.ErrRootNotFound
	})
	if err != nil {
		t.Fatalf("NewRootScopedRepository denied: %v", err)
	}
	if _, err := denied.Get(
		t.Context(),
		definitionTestRootID,
		stored.Digest,
	); !errors.Is(
		err,
		basespec.ErrRootNotFound,
	) {
		t.Fatalf("denied Get error=%v", err)
	}
	if deniedRepository.calls != 0 {
		t.Fatalf("repository called despite root denial: %d", deniedRepository.calls)
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
