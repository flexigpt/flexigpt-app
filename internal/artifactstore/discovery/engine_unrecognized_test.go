package discovery

import (
	"bytes"
	"context"
	"io"
	"path"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

const (
	unrecognizedTestRootID       artifactstore.RootID       = "019d3150-6a12-7a6b-a34e-d9032342bc31"
	unrecognizedTestCollectionID artifactstore.CollectionID = "019d3150-6a13-7a6b-a34e-d9032342bc31"
	unrecognizedTestSourceID     artifactstore.SourceID     = "019d3150-6a14-7a6b-a34e-d9032342bc31"
)

type unrecognizedTestDecoder struct{}

func (unrecognizedTestDecoder) ID() artifactstore.DecoderID {
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
) ([]Decoded, []artifactstore.Diagnostic) {
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
	locator artifactstore.Locator,
) (source.Entry, error) {
	if err := ctx.Err(); err != nil {
		return source.Entry{}, err
	}
	if locator != "candidate.md" {
		return source.Entry{}, artifactstore.ErrNotFound
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
	_ artifactstore.Locator,
) ([]source.Entry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, artifactstore.ErrNotFound
}

func (s unrecognizedTestSnapshot) Open(
	ctx context.Context,
	locator artifactstore.Locator,
) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if locator != "candidate.md" {
		return nil, artifactstore.ErrNotFound
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

	definitionDigest := artifactstore.DigestBytes([]byte("definition"))
	contentDigest := artifactstore.DigestBytes([]byte("previous content"))
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
			ExplicitLocators: []artifactstore.Locator{"candidate.md"},
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
	if len(result.Occurrences[0].Diagnostics) != 1 ||
		result.Occurrences[0].Diagnostics[0].Code != DiagnosticCodeDecoderNoLongerRecognizes {
		t.Fatalf("unexpected occurrence diagnostics: %#v", result.Occurrences[0].Diagnostics)
	}
	if len(result.Diagnostics) != 1 ||
		result.Diagnostics[0].Code != DiagnosticCodeDecoderNoLongerRecognizes {
		t.Fatalf("unexpected result diagnostics: %#v", result.Diagnostics)
	}
}
