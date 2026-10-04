package source

import (
	"testing"
	"time"

	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
)

func TestLifecycleTransitionValidatesSourceOwnedStates(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	base := sourceModel.Source{
		ID:             "019d3150-6a31-7a6b-a34e-d9032342bc31",
		RootID:         rootModel.RootID("019d3150-6a32-7a6b-a34e-d9032342bc31"),
		RootStorageKey: "test-root",
		StorageKey:     "test-source",
		Kind:           "managed-source",
		DisplayName:    "Test source",
		Enabled:        true,
		Config:         []byte(`{}`),
		Revision:       2,
		CreatedAt:      now,
		ModifiedAt:     now.Add(time.Second),
	}

	cases := []LifecycleTransition{
		{Source: base, ExpectedSourceRevision: 1, Invalidation: LifecycleInvalidationDiscoveryRemoved},
		func() LifecycleTransition {
			value := base.Clone()
			value.Enabled = false
			return LifecycleTransition{
				Source:                 value,
				ExpectedSourceRevision: 1,
				Invalidation:           LifecycleInvalidationDisabled,
			}
		}(),
		func() LifecycleTransition {
			value := base.Clone()
			retired := value.ModifiedAt
			value.Enabled, value.RetiredAt = false, &retired
			return LifecycleTransition{
				Source:                 value,
				ExpectedSourceRevision: 1,
				Invalidation:           LifecycleInvalidationRetired,
			}
		}(),
	}
	for index, transition := range cases {
		if err := transition.Validate(); err != nil {
			t.Fatalf("transition %d: %v", index, err)
		}
	}
}
