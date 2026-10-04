package artifact

import (
	"testing"
	"time"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

func TestDeriveLifecycleInvalidationPreservesLocalArtifactFields(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	definitionDigest := cryptoutil.DigestBytes([]byte("definition"))
	contentDigest := cryptoutil.DigestBytes([]byte("content"))
	rootID := rootModel.RootID("019d3150-6a41-7a6b-a34e-d9032342bc31")
	sourceID := sourceModel.SourceID("019d3150-6a42-7a6b-a34e-d9032342bc31")
	current := artifactModel.Artifact{
		ID:                  "019d3150-6a43-7a6b-a34e-d9032342bc31",
		RootID:              rootID,
		Binding:             artifactModel.SourceBinding{SourceID: sourceID, Locator: "declaration.json"},
		Kind:                "agent",
		LogicalName:         "example",
		ResolvedDefinition:  &definitionDigest,
		SourceContentDigest: &contentDigest,
		State:               artifactModel.StateAvailable,
		DisplayName:         "Local display name",
		Enabled:             false,
		Data:                []byte(`{"local":true}`),
		Revision:            5,
		CreatedAt:           now,
		ModifiedAt:          now.Add(time.Second),
	}
	transitionSource := sourceModel.Source{
		ID:             sourceID,
		RootID:         rootID,
		RootStorageKey: "test-root",
		StorageKey:     "test-source",
		Kind:           "managed-source",
		DisplayName:    "Test source",
		Enabled:        false,
		Config:         []byte(`{}`),
		Revision:       2,
		CreatedAt:      now,
		ModifiedAt:     now.Add(2 * time.Second),
	}
	synchronizer, err := NewSynchronizer(clockutil.System{}, NewUUIDIDProvider())
	if err != nil {
		t.Fatalf("NewSynchronizer: %v", err)
	}
	updates, err := synchronizer.DeriveLifecycleInvalidation(
		t.Context(),
		source.LifecycleTransition{
			Source:                 transitionSource,
			ExpectedSourceRevision: 1,
			Invalidation:           source.LifecycleInvalidationDisabled,
		},
		[]artifactModel.Artifact{current},
	)
	if err != nil {
		t.Fatalf("DeriveLifecycleInvalidation: %v", err)
	}
	if len(updates) != 1 {
		t.Fatalf("updates=%d, want 1", len(updates))
	}
	update := updates[0]
	if update.State != artifactModel.StateMissing || update.ResolvedDefinition != nil ||
		update.SourceContentDigest != nil {
		t.Fatalf("update=%#v", update)
	}
	if update.ExpectedRevision != current.Revision || update.Revision != current.Revision+1 {
		t.Fatalf("revision transition=%d/%d", update.ExpectedRevision, update.Revision)
	}
}
