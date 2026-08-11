package refresh

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

const (
	refreshTestRootID       basespec.RootID       = "019d3150-6a44-7a6b-a34e-d9032342bc31"
	refreshTestCollectionID basespec.CollectionID = "019d3150-6a45-7a6b-a34e-d9032342bc31"
	refreshTestSourceID     basespec.SourceID     = "019d3150-6a46-7a6b-a34e-d9032342bc31"
)

func TestPublicationValidationRejectsBrokenConcurrencyAndOwnership(t *testing.T) {
	t.Parallel()

	valid := refreshTestPublication()
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid Publication.Validate: %v", err)
	}
	missingGeneration := refreshTestPublication()
	missingGeneration.SourceGenerations = nil
	if err := missingGeneration.Validate(); !errors.Is(err, basespec.ErrInvalid) {
		t.Fatalf("missing generation error=%v, want ErrInvalid", err)
	}
	unattached := refreshTestPublication()
	unattached.Occurrences[0].Key.SourceID = "019d3150-6a47-7a6b-a34e-d9032342bc31"
	if err := unattached.Validate(); !errors.Is(err, basespec.ErrInvalid) {
		t.Fatalf("unattached occurrence error=%v, want ErrInvalid", err)
	}
	duplicate := refreshTestPublication()
	duplicate.Occurrences = append(duplicate.Occurrences, duplicate.Occurrences[0])
	if err := duplicate.Validate(); !errors.Is(err, basespec.ErrInvalid) {
		t.Fatalf("duplicate occurrence error=%v, want ErrInvalid", err)
	}
}

type refreshTestSnapshot struct {
	closed int
	err    error
}

func (s *refreshTestSnapshot) Generation() string { return "generation-1" }
func (s *refreshTestSnapshot) Stat(context.Context, basespec.Locator) (source.Entry, error) {
	return source.Entry{}, basespec.ErrNotFound
}

func (s *refreshTestSnapshot) ReadDir(context.Context, basespec.Locator) ([]source.Entry, error) {
	return nil, basespec.ErrNotFound
}

func (s *refreshTestSnapshot) Open(context.Context, basespec.Locator) (io.ReadCloser, error) {
	return nil, basespec.ErrNotFound
}
func (s *refreshTestSnapshot) Confirm(context.Context) error { return nil }
func (s *refreshTestSnapshot) Close() error {
	s.closed++
	return s.err
}

func TestCloseRefreshSnapshotsClosesEverySnapshotAndJoinsErrors(t *testing.T) {
	t.Parallel()

	first := &refreshTestSnapshot{}
	second := &refreshTestSnapshot{err: errors.New("close failure")}
	err := closeRefreshSnapshots([]source.Snapshot{first, nil, second})
	if !errors.Is(err, second.err) {
		t.Fatalf("close error=%v, want joined failure", err)
	}
	if first.closed != 1 || second.closed != 1 {
		t.Fatalf("close calls: first=%d second=%d", first.closed, second.closed)
	}
}

func refreshTestPublication() Publication {
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	definitionDigest := cryptoutil.DigestBytes([]byte("definition"))
	contentDigest := cryptoutil.DigestBytes([]byte("content"))
	return Publication{
		Ref: collection.CollectionRef{
			RootID:       refreshTestRootID,
			CollectionID: refreshTestCollectionID,
		},
		ExpectedCollectionRevision: 1,
		ExpectedAttachmentRevisions: map[basespec.SourceID]uint64{
			refreshTestSourceID: 1,
		},
		ExpectedSourceRevisions: map[basespec.SourceID]uint64{
			refreshTestSourceID: 1,
		},
		SourceGenerations: map[basespec.SourceID]string{
			refreshTestSourceID: "generation-1",
		},
		PlanFingerprint:    cryptoutil.DigestBytes([]byte("plan")),
		DecoderFingerprint: cryptoutil.DigestBytes([]byte("decoder")),
		Occurrences: []catalog.Occurrence{{
			RootID:       refreshTestRootID,
			CollectionID: refreshTestCollectionID,
			Key: catalog.OccurrenceKey{
				CollectionID: refreshTestCollectionID,
				SourceID:     refreshTestSourceID,
				Locator:      "artifact.json",
			},
			Kind:                "test.artifact",
			LogicalName:         "artifact",
			DefinitionDigest:    &definitionDigest,
			SourceContentDigest: &contentDigest,
			DecoderID:           "test.decoder",
			State:               catalog.OccurrenceValid,
			ObservedAt:          now,
		}},
		PublishedAt: now,
	}
}
