package discovery

import (
	"bytes"
	"context"
	"io"
	"path"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

const (
	unrecognizedTestRootID       basespec.RootID       = "019d3150-6a12-7a6b-a34e-d9032342bc31"
	unrecognizedTestCollectionID basespec.CollectionID = "019d3150-6a13-7a6b-a34e-d9032342bc31"
	unrecognizedTestSourceID     basespec.SourceID     = "019d3150-6a14-7a6b-a34e-d9032342bc31"
)

type unrecognizedTestDecoder struct{}

func (unrecognizedTestDecoder) ID() basespec.DecoderID {
	return "test.decoder"
}

func (unrecognizedTestDecoder) Revision() string {
	return "v1"
}

func (unrecognizedTestDecoder) Recognize(
	context.Context,
	Candidate,
) Recognition {
	return RecognitionNone
}

func (unrecognizedTestDecoder) Decode(
	context.Context,
	Candidate,
) ([]Decoded, []diagnostic.Diagnostic) {
	return nil, nil
}

type unrecognizedTestClock struct {
	now time.Time
}

func (c unrecognizedTestClock) Now() time.Time {
	return c.now
}

type unrecognizedTestSnapshot struct {
	content []byte
}

func (s unrecognizedTestSnapshot) Generation() string {
	return "test-generation"
}

func (s unrecognizedTestSnapshot) Stat(
	ctx context.Context,
	locator basespec.Locator,
) (source.Entry, error) {
	if err := ctx.Err(); err != nil {
		return source.Entry{}, err
	}
	if locator != "candidate.md" {
		return source.Entry{}, basespec.ErrNotFound
	}
	return source.Entry{
		Locator:   locator,
		Name:      path.Base(string(locator)),
		SizeBytes: int64(len(s.content)),
		IsRegular: true,
	}, nil
}

func (s unrecognizedTestSnapshot) ReadDir(
	ctx context.Context,
	_ basespec.Locator,
) ([]source.Entry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, basespec.ErrNotFound
}

func (s unrecognizedTestSnapshot) Open(
	ctx context.Context,
	locator basespec.Locator,
) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if locator != "candidate.md" {
		return nil, basespec.ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(s.content)), nil
}

func (unrecognizedTestSnapshot) Confirm(context.Context) error {
	return nil
}

func (unrecognizedTestSnapshot) Close() error {
	return nil
}

func TestDiscoverMarksPreviouslyObservedCandidateMissingWhenUnrecognized(
	t *testing.T,
) {
	t.Parallel()

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	registry, err := NewDecoderRegistry(unrecognizedTestDecoder{})
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}
	engine, err := NewEngine(registry, unrecognizedTestClock{now: now})
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}

	definitionValue := unrecognizedTestDefinition()
	definitionDigest := definitionValue.Digest
	contentDigest := cryptoutil.DigestBytes([]byte("previous content"))
	previous := catalog.Occurrence{
		RootID:       unrecognizedTestRootID,
		CollectionID: unrecognizedTestCollectionID,
		Key: catalog.OccurrenceKey{
			CollectionID: unrecognizedTestCollectionID,
			SourceID:     unrecognizedTestSourceID,
			Locator:      "candidate.md",
		},
		Kind:                "test.artifact",
		LogicalName:         "candidate",
		DefinitionDigest:    &definitionDigest,
		Definition:          &definitionValue,
		SourceContentDigest: &contentDigest,
		DecoderID:           "test.decoder",
		State:               catalog.OccurrenceValid,
		ObservedAt:          now.Add(-time.Minute),
	}

	result, err := engine.Discover(
		t.Context(),
		unrecognizedTestRootID,
		unrecognizedTestCollectionID,
		unrecognizedTestSourceID,
		"test.source",
		unrecognizedTestSnapshot{content: []byte("not a supported artifact")},
		SourcePlan{
			SourceID:         unrecognizedTestSourceID,
			ExplicitLocators: []basespec.Locator{"candidate.md"},
			Authoritative:    false,
		},
		[]catalog.Occurrence{previous},
	)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(result.Occurrences) != 1 {
		t.Fatalf("occurrences=%d, want 1", len(result.Occurrences))
	}
	if result.Occurrences[0].State != catalog.OccurrenceMissing {
		t.Fatalf("state=%q, want %q", result.Occurrences[0].State, catalog.OccurrenceMissing)
	}
	if result.Occurrences[0].Definition != nil ||
		result.Occurrences[0].DefinitionDigest != nil {
		t.Fatalf(
			"missing occurrence retained cached definition: %#v",
			result.Occurrences[0],
		)
	}
	if len(result.Occurrences[0].Diagnostics) != 1 ||
		result.Occurrences[0].Diagnostics[0].Code != DiagnosticCodeDecoderNoLongerRecognizes {
		t.Fatalf("unexpected occurrence diagnostics: %#v", result.Occurrences[0].Diagnostics)
	}
	if len(result.Diagnostics) != 1 ||
		result.Diagnostics[0].Code != DiagnosticCodeDecoderNoLongerRecognizes {
		t.Fatalf("unexpected result diagnostics: %#v", result.Diagnostics)
	}
}

func unrecognizedTestDefinition() definition.Definition {
	value, err := definition.Canonicalize(definition.Definition{
		Kind:           "test.artifact",
		SchemaID:       "test.schema",
		SchemaVersion:  "v1",
		LogicalName:    "candidate",
		LogicalVersion: "v1",
		Body:           []byte(`{"value":true}`),
	})
	if err != nil {
		panic(err)
	}
	return value
}
