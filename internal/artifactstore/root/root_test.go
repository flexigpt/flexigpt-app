package root

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

const rootTestID basespec.RootID = "019d3150-6a40-7a6b-a34e-d9032342bc31"

type rootTestClock struct{ now time.Time }

func (c rootTestClock) Now() time.Time { return c.now }

type rootTestIDs struct{ value string }

func (g rootTestIDs) NewID(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return g.value, nil
}

type rootTestRepository struct {
	mu      sync.Mutex
	values  map[basespec.RootID]Root
	updates int
}

func cloneRoot(value Root) Root {
	output := value
	if value.RetiredAt != nil {
		copyValue := *value.RetiredAt
		output.RetiredAt = &copyValue
	}
	return output
}

func (r *rootTestRepository) Create(_ context.Context, value Root) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, found := r.values[value.ID]; found {
		return basespec.ErrConflict
	}
	r.values[value.ID] = cloneRoot(value)
	return nil
}

func (r *rootTestRepository) Get(_ context.Context, id basespec.RootID) (Root, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, found := r.values[id]
	if !found {
		return Root{}, basespec.ErrRootNotFound
	}
	return cloneRoot(value), nil
}

func (r *rootTestRepository) List(context.Context) ([]Root, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	values := make([]Root, 0, len(r.values))
	for _, value := range r.values {
		values = append(values, cloneRoot(value))
	}
	return values, nil
}

func (r *rootTestRepository) Update(_ context.Context, value Root, expected uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, found := r.values[value.ID]
	if !found {
		return basespec.ErrRootNotFound
	}
	if current.Revision != expected {
		return basespec.ErrConflict
	}
	r.values[value.ID] = cloneRoot(value)
	r.updates++
	return nil
}

func (r *rootTestRepository) Retire(_ context.Context, value Root, expected uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, found := r.values[value.ID]
	if !found {
		return basespec.ErrRootNotFound
	}
	if current.Revision != expected {
		return basespec.ErrConflict
	}
	r.values[value.ID] = cloneRoot(value)
	return nil
}

func (r *rootTestRepository) Purge(_ context.Context, id basespec.RootID, expected uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, found := r.values[id]
	if !found {
		return basespec.ErrRootNotFound
	}
	if current.Revision != expected {
		return basespec.ErrConflict
	}
	delete(r.values, id)
	return nil
}

func TestServiceLifecycleNoOpsAndConcurrentRevisionConflicts(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	repository := &rootTestRepository{values: make(map[basespec.RootID]Root)}
	service, err := NewService(repository, rootTestIDs{value: string(rootTestID)}, rootTestClock{now: now})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	created, err := service.Create(t.Context(), RootDraft{DisplayName: "Root", Description: "initial"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.Revision != 1 || !created.CreatedAt.Equal(now) {
		t.Fatalf("created=%#v", created)
	}

	unchanged, err := service.Update(
		t.Context(),
		created.ID,
		RootUpdate{ExpectedRevision: 1, DisplayName: "Root", Description: "initial"},
	)
	if err != nil {
		t.Fatalf("unchanged Update: %v", err)
	}
	if unchanged.Revision != 1 || repository.updates != 0 {
		t.Fatalf("no-op update=%#v repository updates=%d", unchanged, repository.updates)
	}
	updated, err := service.Update(
		t.Context(),
		created.ID,
		RootUpdate{ExpectedRevision: 1, DisplayName: "Renamed", Description: "initial"},
	)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Revision != 2 || !updated.ModifiedAt.After(created.ModifiedAt) {
		t.Fatalf("updated=%#v", updated)
	}
	if _, err := service.Update(
		t.Context(),
		created.ID,
		RootUpdate{ExpectedRevision: 1, DisplayName: "stale"},
	); !errors.Is(
		err,
		basespec.ErrConflict,
	) {
		t.Fatalf("stale Update error=%v, want ErrConflict", err)
	}

	start := make(chan struct{})
	results := make(chan error, 2)
	var group sync.WaitGroup
	for range 2 {
		group.Go(func() {
			<-start
			_, err := service.Update(
				t.Context(),
				created.ID,
				RootUpdate{ExpectedRevision: updated.Revision, DisplayName: "Concurrent", Description: "initial"},
			)
			results <- err
		})
	}
	close(start)
	group.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
			continue
		}
		if !errors.Is(err, basespec.ErrConflict) {
			t.Fatalf("concurrent Update error=%v", err)
		}
	}
	if successes != 1 {
		t.Fatalf("concurrent Update successes=%d, want 1", successes)
	}

	current, err := service.Get(t.Context(), created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	retired, err := service.Retire(t.Context(), created.ID, current.Revision)
	if err != nil {
		t.Fatalf("Retire: %v", err)
	}
	if retired.RetiredAt == nil || retired.Revision != current.Revision+1 {
		t.Fatalf("retired=%#v", retired)
	}
	if err := service.Purge(t.Context(), created.ID, retired.Revision); err != nil {
		t.Fatalf("Purge: %v", err)
	}
	if _, err := service.Get(t.Context(), created.ID); !errors.Is(err, basespec.ErrRootNotFound) {
		t.Fatalf("Get purged root error=%v", err)
	}
}

func TestServiceRejectsMissingExpectedRevisions(t *testing.T) {
	t.Parallel()

	repository := &rootTestRepository{values: make(map[basespec.RootID]Root)}
	service, err := NewService(repository, rootTestIDs{value: string(rootTestID)}, rootTestClock{now: time.Now().UTC()})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	if err := service.Purge(t.Context(), rootTestID, 0); !errors.Is(err, basespec.ErrInvalid) {
		t.Fatalf("Purge missing revision error=%v", err)
	}
	if _, err := service.Update(
		t.Context(),
		rootTestID,
		RootUpdate{},
	); !errors.Is(
		err,
		basespec.ErrInvalid,
	) {
		t.Fatalf("Update missing revision error=%v", err)
	}
	if _, err := NewService(nil, rootTestIDs{}, rootTestClock{}); !errors.Is(err, basespec.ErrInvalid) {
		t.Fatalf("NewService missing dependency error=%v", err)
	}
}
