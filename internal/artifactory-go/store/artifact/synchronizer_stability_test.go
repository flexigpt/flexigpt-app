package artifact

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition"
	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

func TestSynchronizeKeepsArtifactIDWhenDefinitionChangesAtSameOrigin(
	t *testing.T,
) {
	rootID := rootModel.RootID("0192c4c0-0000-7000-8000-000000000001")
	sourceID := sourceModel.SourceID("0192c4c0-0001-7000-8000-000000000001")
	artifactID := artifactModel.ArtifactID("0192c4c0-0002-7000-8000-000000000001")
	locator := storeSpec.Locator("model-provider.yaml")
	now := time.Now().UTC()

	previous := testStableDefinition(t, "previous description")
	next := testStableDefinition(t, "changed description")
	oldContentDigest := cryptoutil.DigestBytes([]byte("previous bytes"))
	newContentDigest := cryptoutil.DigestBytes([]byte("changed bytes"))

	sourceValue := sourceModel.Source{
		ID:             sourceID,
		RootID:         rootID,
		RootStorageKey: storeSpec.StorageKey("builtins"),
		StorageKey:     storeSpec.StorageKey("artifacts"),
		Kind:           sourceModel.SourceKind("managed-directory"),
		DisplayName:    "Built-ins",
		Enabled:        true,
		Config:         json.RawMessage(`{}`),
		Discovery: sourceModel.DiscoverySpec{
			ExplicitLocators: []storeSpec.Locator{locator},
		},
		Revision:   1,
		CreatedAt:  now,
		ModifiedAt: now,
	}

	current := artifactModel.Artifact{
		ID:     artifactID,
		RootID: rootID,
		Binding: artifactModel.SourceBinding{
			SourceID: sourceID,
			Locator:  locator,
		},
		Kind:                artifactModel.ArtifactKind("model.provider"),
		LogicalName:         storeSpec.LogicalName("provider"),
		ResolvedDefinition:  cryptoutil.CloneDigest(&previous.Digest),
		SourceContentDigest: cryptoutil.CloneDigest(&oldContentDigest),
		State:               artifactModel.StateAvailable,
		DisplayName:         "Provider",
		Enabled:             true,
		Data:                json.RawMessage(`{}`),
		Revision:            1,
		CreatedAt:           now,
		ModifiedAt:          now,
	}

	synchronizer, err := NewSynchronizer(
		clockutil.System{},
		IDProviderFunc(func(context.Context) (artifactModel.ArtifactID, error) {
			return "", errors.New("unexpected Artifact ID allocation")
		}),
	)
	if err != nil {
		t.Fatalf("create synchronizer: %v", err)
	}

	result, err := synchronizer.Synchronize(
		t.Context(),
		rootID,
		sourceValue,
		[]ingest.Observation{{
			RootID: rootID,
			Binding: artifactModel.SourceBinding{
				SourceID: sourceID,
				Locator:  locator,
			},
			Kind:                artifactModel.ArtifactKind("model.provider"),
			LogicalName:         storeSpec.LogicalName("provider"),
			Definition:          &next,
			SourceContentDigest: &newContentDigest,
			DecoderID:           storeSpec.DecoderID("artifact-declaration-json"),
			State:               ingest.ObservationValid,
		}},
		[]storeSpec.Locator{locator},
		[]artifactModel.Artifact{current},
	)
	if err != nil {
		t.Fatalf("synchronize changed Definition: %v", err)
	}
	if len(result.Creates) != 0 {
		t.Fatalf("created Artifact count = %d, want 0", len(result.Creates))
	}
	if len(result.Updates) != 1 {
		t.Fatalf("updated Artifact count = %d, want 1", len(result.Updates))
	}

	update := result.Updates[0]
	if update.ArtifactID != artifactID {
		t.Fatalf("updated Artifact ID = %q, want %q", update.ArtifactID, artifactID)
	}
	if update.Binding != current.Binding {
		t.Fatalf("updated Artifact binding = %#v, want %#v", update.Binding, current.Binding)
	}
	if update.ResolvedDefinition == nil ||
		*update.ResolvedDefinition != next.Digest {
		t.Fatalf(
			"updated Definition = %v, want %q",
			update.ResolvedDefinition,
			next.Digest,
		)
	}
}

func testStableDefinition(
	t *testing.T,
	description string,
) definitionModel.Definition {
	t.Helper()

	value, err := definition.Admit(definitionModel.Definition{
		Kind:          artifactModel.ArtifactKind("model.provider"),
		SchemaID:      schemaModel.SchemaID("artifact.model-provider.v1"),
		SchemaVersion: "v1",
		LogicalName:   storeSpec.LogicalName("provider"),
		DisplayName:   "Provider",
		Description:   description,
		Body: json.RawMessage(
			`{"adapter":"example.adapter","name":"provider","type":"model.provider"}`,
		),
	})
	if err != nil {
		t.Fatalf("admit test Definition: %v", err)
	}
	return value
}
