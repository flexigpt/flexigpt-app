package catalog

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

const (
	catalogTestRootID       basespec.RootID       = "019d3150-6a16-7a6b-a34e-d9032342bc31"
	catalogTestCollectionID basespec.CollectionID = "019d3150-6a17-7a6b-a34e-d9032342bc31"
	catalogTestSourceID     basespec.SourceID     = "019d3150-6a18-7a6b-a34e-d9032342bc31"
)

func TestSnapshotCloneAndEqualityIgnoreOccurrenceOrder(t *testing.T) {
	t.Parallel()

	original := catalogTestSnapshot()
	second := catalogTestOccurrence("two.json")
	original.Occurrences = append(original.Occurrences, second)
	if err := original.Validate(); err != nil {
		t.Fatalf("fixture validation: %v", err)
	}
	cloned := CloneSnapshot(original)
	original.AttachmentRevisions[catalogTestSourceID] = 9
	original.Occurrences[0].Diagnostics = []diagnostic.Diagnostic{{
		Severity: diagnostic.DiagnosticWarning,
		Code:     "test.warning",
		Message:  "changed",
	}}
	if cloned.AttachmentRevisions[catalogTestSourceID] != 1 || len(cloned.Occurrences[0].Diagnostics) != 0 {
		t.Fatalf("CloneSnapshot retained mutable state: %#v", cloned)
	}

	reordered := CloneSnapshot(cloned)
	reordered.Occurrences[0], reordered.Occurrences[1] = reordered.Occurrences[1], reordered.Occurrences[0]
	if !EqualSnapshot(cloned, reordered) {
		t.Fatal("EqualSnapshot treated occurrence ordering as semantic")
	}
	duplicates := []Occurrence{cloned.Occurrences[0], cloned.Occurrences[0]}
	if EqualOccurrences(duplicates, duplicates) {
		t.Fatal("EqualOccurrences accepted duplicate occurrence keys")
	}
}

type catalogTestReader struct {
	value Snapshot
	err   error
}

func (r catalogTestReader) GetCurrent(context.Context, collection.CollectionRef) (Snapshot, error) {
	return r.value, r.err
}

func TestReadCurrentValidatesOwnershipAndPreservesStaleCatalogs(t *testing.T) {
	t.Parallel()

	ref := collection.CollectionRef{RootID: catalogTestRootID, CollectionID: catalogTestCollectionID}
	value := catalogTestSnapshot()
	read, err := ReadCurrent(
		t.Context(),
		catalogTestReader{value: value, err: basespec.ErrCatalogStale},
		ref,
	)
	if !errors.Is(err, basespec.ErrCatalogStale) {
		t.Fatalf("stale read error=%v", err)
	}
	value.AttachmentRevisions[catalogTestSourceID] = 7
	if read.AttachmentRevisions[catalogTestSourceID] != 1 {
		t.Fatalf("ReadCurrent did not return owned copy: %#v", read.AttachmentRevisions)
	}

	wrong := catalogTestSnapshot()
	wrong.RootID = "019d3150-6a19-7a6b-a34e-d9032342bc31"
	if _, err := ReadCurrent(
		t.Context(),
		catalogTestReader{value: wrong},
		ref,
	); !errors.Is(
		err,
		basespec.ErrInvalid,
	) {
		t.Fatalf("wrong ownership error=%v, want ErrInvalid", err)
	}
	if _, err := ReadCurrent(t.Context(), nil, ref); !errors.Is(err, basespec.ErrInvalid) {
		t.Fatalf("nil reader error=%v, want ErrInvalid", err)
	}
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := ReadCurrent(
		cancelled,
		catalogTestReader{value: catalogTestSnapshot()},
		ref,
	); !errors.Is(
		err,
		context.Canceled,
	) {
		t.Fatalf("cancelled read error=%v", err)
	}
}

func catalogTestSnapshot() Snapshot {
	return Snapshot{
		RootID:              catalogTestRootID,
		CollectionID:        catalogTestCollectionID,
		Revision:            1,
		CollectionRevision:  1,
		AttachmentRevisions: map[basespec.SourceID]uint64{catalogTestSourceID: 1},
		SourceRevisions:     map[basespec.SourceID]uint64{catalogTestSourceID: 1},
		SourceGenerations:   map[basespec.SourceID]string{catalogTestSourceID: "generation-1"},
		PlanFingerprint:     cryptoutil.DigestBytes([]byte("plan")),
		DecoderFingerprint:  cryptoutil.DigestBytes([]byte("decoder")),
		PublishedAt:         time.Date(2026, 3, 25, 12, 1, 0, 0, time.UTC),
		Occurrences:         []Occurrence{catalogTestOccurrence("one.json")},
	}
}

func catalogTestOccurrence(locator basespec.Locator) Occurrence {
	definitionValue := catalogTestDefinition()
	definitionDigest := definitionValue.Digest
	contentDigest := cryptoutil.DigestBytes([]byte("content:" + string(locator)))
	return Occurrence{
		RootID:       catalogTestRootID,
		CollectionID: catalogTestCollectionID,
		Key: OccurrenceKey{
			CollectionID: catalogTestCollectionID,
			SourceID:     catalogTestSourceID,
			Locator:      locator,
		},
		Kind:                "test.artifact",
		LogicalName:         "example",
		DefinitionDigest:    &definitionDigest,
		Definition:          &definitionValue,
		SourceContentDigest: &contentDigest,
		DecoderID:           "test.decoder",
		State:               OccurrenceValid,
		ObservedAt:          time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC),
	}
}

func catalogTestDefinition() definition.Definition {
	value, err := definition.Canonicalize(definition.Definition{
		Kind:           "test.artifact",
		SchemaID:       "test.schema",
		SchemaVersion:  "v1",
		LogicalName:    "example",
		LogicalVersion: "v1",
		Body:           []byte(`{"value":true}`),
	})
	if err != nil {
		panic(err)
	}
	return value
}
