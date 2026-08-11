package discovery

import (
	"bytes"
	"context"
	"errors"
	"io"
	"path"
	"sync"
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
	discoveryTestRootID       basespec.RootID       = "019d3150-6a22-7a6b-a34e-d9032342bc31"
	discoveryTestCollectionID basespec.CollectionID = "019d3150-6a23-7a6b-a34e-d9032342bc31"
	discoveryTestSourceID     basespec.SourceID     = "019d3150-6a24-7a6b-a34e-d9032342bc31"
)

type discoveryTestClock struct{ now time.Time }

func (c discoveryTestClock) Now() time.Time { return c.now }

type discoveryTestDecoder struct {
	id          basespec.DecoderID
	revision    string
	recognition Recognition
	decoded     []Decoded
	diagnostics []diagnostic.Diagnostic
}

func (d discoveryTestDecoder) ID() basespec.DecoderID { return d.id }
func (d discoveryTestDecoder) Revision() string       { return d.revision }
func (d discoveryTestDecoder) Recognize(context.Context, Candidate) Recognition {
	return d.recognition
}

func (d discoveryTestDecoder) Decode(context.Context, Candidate) ([]Decoded, []diagnostic.Diagnostic) {
	return d.decoded, d.diagnostics
}

type discoveryTestSnapshot struct {
	generation string
	content    map[basespec.Locator][]byte
}

func (s discoveryTestSnapshot) Generation() string { return s.generation }

func (s discoveryTestSnapshot) Stat(ctx context.Context, locator basespec.Locator) (source.Entry, error) {
	if err := ctx.Err(); err != nil {
		return source.Entry{}, err
	}
	content, found := s.content[locator]
	if !found {
		return source.Entry{}, basespec.ErrNotFound
	}
	return source.Entry{
		Locator:   locator,
		Name:      path.Base(string(locator)),
		SizeBytes: int64(len(content)),
		IsRegular: true,
	}, nil
}

func (s discoveryTestSnapshot) ReadDir(ctx context.Context, _ basespec.Locator) ([]source.Entry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, basespec.ErrNotFound
}

func (s discoveryTestSnapshot) Open(ctx context.Context, locator basespec.Locator) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	content, found := s.content[locator]
	if !found {
		return nil, basespec.ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(content)), nil
}

func (discoveryTestSnapshot) Confirm(context.Context) error { return nil }
func (discoveryTestSnapshot) Close() error                  { return nil }

func TestDiscoverCreatesCanonicalOccurrenceAndDefinition(t *testing.T) {
	t.Parallel()

	decoder := discoveryTestDecoder{
		id:          "test.decoder",
		revision:    "v1",
		recognition: RecognitionPreferred,
		decoded:     []Decoded{{Definition: discoveryTestDefinition()}},
	}
	registry, err := NewDecoderRegistry(decoder)
	if err != nil {
		t.Fatalf("NewDecoderRegistry: %v", err)
	}
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	engine, err := NewEngine(registry, discoveryTestClock{now: now})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	plan := discoveryTestPlan()
	plan.DecoderHints = []DecoderHint{{Locator: "example.json", DecoderIDs: []basespec.DecoderID{"test.decoder"}}}
	result, err := engine.Discover(
		t.Context(),
		discoveryTestRootID,
		discoveryTestCollectionID,
		discoveryTestSourceID,
		"test.source",
		discoveryTestSnapshot{
			generation: "generation-1",
			content:    map[basespec.Locator][]byte{"example.json": []byte(`{"payload":1}`)},
		},
		plan,
		nil,
	)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if result.Candidates != 1 || len(result.Occurrences) != 1 || len(result.Definitions) != 1 {
		t.Fatalf("result=%#v", result)
	}
	occurrence := result.Occurrences[0]
	if occurrence.State != catalog.OccurrenceValid || occurrence.DecoderID != "test.decoder" ||
		occurrence.ObservedAt != now {
		t.Fatalf("occurrence=%#v", occurrence)
	}
	if occurrence.DefinitionDigest == nil {
		t.Fatal("valid occurrence has no definition digest")
	}
	if _, found := result.Definitions[*occurrence.DefinitionDigest]; !found {
		t.Fatalf("definition %q was not returned", *occurrence.DefinitionDigest)
	}
}

func TestDiscoverReportsAmbiguityDigestMismatchAndInvalidPlans(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	one := discoveryTestDecoder{id: "test.one", revision: "v1", recognition: RecognitionPossible}
	two := discoveryTestDecoder{id: "test.two", revision: "v1", recognition: RecognitionPossible}
	registry, err := NewDecoderRegistry(one, two)
	if err != nil {
		t.Fatalf("NewDecoderRegistry: %v", err)
	}
	engine, err := NewEngine(registry, discoveryTestClock{now: now})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	snapshot := discoveryTestSnapshot{
		generation: "generation-1",
		content:    map[basespec.Locator][]byte{"example.json": []byte("content")},
	}
	result, err := engine.Discover(
		t.Context(),
		discoveryTestRootID,
		discoveryTestCollectionID,
		discoveryTestSourceID,
		"test.source",
		snapshot,
		discoveryTestPlan(),
		nil,
	)
	if err != nil {
		t.Fatalf("ambiguous Discover: %v", err)
	}
	if len(result.Occurrences) != 1 || result.Occurrences[0].State != catalog.OccurrenceInvalid ||
		len(result.Diagnostics) != 1 ||
		result.Diagnostics[0].Code != DiagnosticCodeDecoderAmbiguous {
		t.Fatalf("ambiguous result=%#v", result)
	}

	plan := discoveryTestPlan()
	plan.ExpectedContentDigests = map[basespec.Locator]cryptoutil.Digest{
		"example.json": cryptoutil.DigestBytes([]byte("other")),
	}
	mismatch, err := engine.Discover(
		t.Context(),
		discoveryTestRootID,
		discoveryTestCollectionID,
		discoveryTestSourceID,
		"test.source",
		snapshot,
		plan,
		nil,
	)
	if err != nil {
		t.Fatalf("digest mismatch Discover: %v", err)
	}
	if len(mismatch.Diagnostics) != 1 || mismatch.Diagnostics[0].Code != DiagnosticCodeContentDigestMismatch {
		t.Fatalf("digest mismatch result=%#v", mismatch)
	}

	unavailable := discoveryTestPlan()
	unavailable.AllowedDecoderIDs = []basespec.DecoderID{"missing.decoder"}
	if _, err := engine.Discover(
		t.Context(),
		discoveryTestRootID,
		discoveryTestCollectionID,
		discoveryTestSourceID,
		"test.source",
		snapshot,
		unavailable,
		nil,
	); !errors.Is(
		err,
		basespec.ErrDecoderUnavailable,
	) {
		t.Fatalf("unavailable decoder error=%v", err)
	}
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := engine.Discover(
		cancelled,
		discoveryTestRootID,
		discoveryTestCollectionID,
		discoveryTestSourceID,
		"test.source",
		snapshot,
		discoveryTestPlan(),
		nil,
	); !errors.Is(
		err,
		context.Canceled,
	) {
		t.Fatalf("cancelled discover error=%v", err)
	}
}

func TestDiscoverIsSafeForConcurrentReadOnlySnapshots(t *testing.T) {
	t.Parallel()

	decoder := discoveryTestDecoder{
		id:          "test.decoder",
		revision:    "v1",
		recognition: RecognitionPreferred,
		decoded:     []Decoded{{Definition: discoveryTestDefinition()}},
	}
	registry, err := NewDecoderRegistry(decoder)
	if err != nil {
		t.Fatalf("NewDecoderRegistry: %v", err)
	}
	engine, err := NewEngine(registry, discoveryTestClock{now: time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	snapshot := discoveryTestSnapshot{
		generation: "generation-1",
		content:    map[basespec.Locator][]byte{"example.json": []byte(`{"payload":1}`)},
	}
	const workers = 24
	var group sync.WaitGroup
	errorsSeen := make(chan error, workers)
	for range workers {
		group.Go(func() {
			result, err := engine.Discover(
				t.Context(),
				discoveryTestRootID,
				discoveryTestCollectionID,
				discoveryTestSourceID,
				"test.source",
				snapshot,
				discoveryTestPlan(),
				nil,
			)
			if err != nil {
				errorsSeen <- err
				return
			}
			if len(result.Occurrences) != 1 || result.Occurrences[0].State != catalog.OccurrenceValid {
				errorsSeen <- errors.New("unexpected concurrent discovery result")
			}
		})
	}
	group.Wait()
	close(errorsSeen)
	for err := range errorsSeen {
		t.Fatalf("concurrent discovery: %v", err)
	}
}

func discoveryTestPlan() SourcePlan {
	return SourcePlan{
		SourceID:         discoveryTestSourceID,
		ExplicitLocators: []basespec.Locator{"example.json"},
	}
}

func discoveryTestDefinition() definition.Definition {
	return definition.Definition{
		Kind:          "test.artifact",
		SchemaID:      "test.schema",
		SchemaVersion: "v1",
		LogicalName:   "example",
		Body:          []byte(`{"value":true}`),
	}
}
