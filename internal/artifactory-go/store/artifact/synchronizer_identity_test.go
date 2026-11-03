package artifact

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition"
	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

func TestSynchronizePreservesIDForDefinitionChangeAtSameTypedOrigin(
	t *testing.T,
) {
	rootID := rootModel.RootID(
		"0192c4c0-0000-7000-8000-000000000001",
	)
	sourceID := sourceModel.SourceID(
		"0192c4c0-0001-7000-8000-000000000001",
	)
	artifactID := artifactModel.ArtifactID(
		"0192c4c0-0002-7000-8000-000000000001",
	)
	binding := artifactModel.SourceBinding{
		SourceID: sourceID,
		Locator:  storeSpec.Locator("agents/stable-agent.yaml"),
	}
	createdAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	previousDefinition := admittedDefinitionForIdentityTest(
		t,
		`{"field":"before"}`,
	)
	updatedDefinition := admittedDefinitionForIdentityTest(
		t,
		`{"field":"after","newOptionalField":true}`,
	)

	previousContent := cryptoutil.DigestBytes([]byte("before"))
	updatedContent := cryptoutil.DigestBytes([]byte("after"))
	resolvedPrevious := previousDefinition.Digest

	existing := artifactModel.Artifact{
		ID:                  artifactID,
		RootID:              rootID,
		Binding:             binding,
		Kind:                artifactModel.ArtifactKind("agent"),
		LogicalName:         previousDefinition.LogicalName,
		LogicalVersion:      previousDefinition.LogicalVersion,
		ResolvedDefinition:  &resolvedPrevious,
		SourceContentDigest: &previousContent,
		State:               artifactModel.StateAvailable,
		DisplayName:         "Stable Agent",
		Enabled:             true,
		Data:                json.RawMessage([]byte(`{}`)),
		Revision:            7,
		CreatedAt:           createdAt,
		ModifiedAt:          createdAt,
	}

	sourceValue := sourceModel.Source{
		ID:             sourceID,
		RootID:         rootID,
		RootStorageKey: storeSpec.StorageKey("root"),
		StorageKey:     storeSpec.StorageKey("source"),
		Kind:           sourceModel.SourceKind("managed-directory"),
		DisplayName:    "Test Source",
		Enabled:        true,
		Config:         json.RawMessage([]byte(`{}`)),
		Discovery: sourceModel.DiscoverySpec{
			ExplicitLocators: []storeSpec.Locator{binding.Locator},
		}.Normalized(),
		Revision:   1,
		CreatedAt:  createdAt,
		ModifiedAt: createdAt,
	}

	allocations := 0
	synchronizer, err := NewSynchronizer(
		clockutil.System{},
		IDProviderFunc(func(context.Context) (artifactModel.ArtifactID, error) {
			allocations++
			return artifactModel.ArtifactID(
				"0192c4c0-0003-7000-8000-000000000001",
			), nil
		}),
	)
	if err != nil {
		t.Fatalf("create synchronizer: %v", err)
	}

	observationDefinition := updatedDefinition.Clone()
	result, err := synchronizer.Synchronize(
		t.Context(),
		rootID,
		sourceValue,
		[]ingest.Observation{{
			RootID:              rootID,
			Binding:             binding,
			Kind:                artifactModel.ArtifactKind("agent"),
			LogicalName:         updatedDefinition.LogicalName,
			LogicalVersion:      updatedDefinition.LogicalVersion,
			Definition:          &observationDefinition,
			SourceContentDigest: &updatedContent,
			DecoderID:           storeSpec.DecoderID("testDecoder"),
			State:               ingest.ObservationValid,
		}},
		[]storeSpec.Locator{binding.Locator},
		[]artifactModel.Artifact{existing},
	)
	if err != nil {
		t.Fatalf("synchronize Artifact: %v", err)
	}

	if allocations != 0 {
		t.Fatalf(
			"synchronizer allocated %d Artifact IDs for an existing typed origin",
			allocations,
		)
	}
	if len(result.Creates) != 0 {
		t.Fatalf(
			"synchronizer created %d Artifacts for an existing typed origin",
			len(result.Creates),
		)
	}
	if len(result.Updates) != 1 {
		t.Fatalf(
			"synchronizer produced %d updates, want one",
			len(result.Updates),
		)
	}

	update := result.Updates[0]
	if update.ArtifactID != artifactID {
		t.Fatalf(
			"updated Artifact ID = %q, want preserved ID %q",
			update.ArtifactID,
			artifactID,
		)
	}
	if update.ExpectedRevision != existing.Revision ||
		update.Revision != existing.Revision+1 {
		t.Fatalf(
			"updated Artifact revision = %d from %d, want %d from %d",
			update.Revision,
			update.ExpectedRevision,
			existing.Revision+1,
			existing.Revision,
		)
	}
	if update.ResolvedDefinition == nil ||
		*update.ResolvedDefinition != updatedDefinition.Digest {
		t.Fatalf(
			"updated Artifact Definition = %v, want %q",
			update.ResolvedDefinition,
			updatedDefinition.Digest,
		)
	}
	if update.SourceContentDigest == nil ||
		*update.SourceContentDigest != updatedContent {
		t.Fatalf(
			"updated Artifact source digest = %v, want %q",
			update.SourceContentDigest,
			updatedContent,
		)
	}
}

func admittedDefinitionForIdentityTest(
	t *testing.T,
	body string,
) definitionModel.Definition {
	t.Helper()

	value, err := definition.Admit(definitionModel.Definition{
		Kind:          artifactModel.ArtifactKind("agent"),
		SchemaID:      "agent",
		SchemaVersion: "v1",
		LogicalName:   storeSpec.LogicalName("stable-agent"),
		DisplayName:   "Stable Agent",
		Body:          json.RawMessage([]byte(body)),
	})
	if err != nil {
		t.Fatalf("admit Definition: %v", err)
	}
	return value
}
