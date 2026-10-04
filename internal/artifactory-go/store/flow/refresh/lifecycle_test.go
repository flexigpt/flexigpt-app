package refresh

import (
	"testing"
	"time"

	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
)

func TestLifecyclePublicationAllowsEmptyRefreshStateExpectation(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	publication := LifecyclePublication{Transition: source.LifecycleTransition{
		Source: sourceModel.Source{
			ID:             "019d3150-6a51-7a6b-a34e-d9032342bc31",
			RootID:         rootModel.RootID("019d3150-6a52-7a6b-a34e-d9032342bc31"),
			RootStorageKey: "test-root",
			StorageKey:     "test-source",
			Kind:           "managed-source",
			DisplayName:    "Test source",
			Enabled:        false,
			Config:         []byte(`{}`),
			Revision:       2,
			CreatedAt:      now,
			ModifiedAt:     now.Add(time.Second),
		},
		ExpectedSourceRevision: 1,
		Invalidation:           source.LifecycleInvalidationDisabled,
	}}
	if err := publication.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}
