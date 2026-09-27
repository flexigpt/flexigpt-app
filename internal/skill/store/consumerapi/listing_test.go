package consumerapi_test

import (
	"testing"

	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	skillConsumerAPI "github.com/flexigpt/flexigpt-app/internal/skill/store/consumerapi"
)

func TestSkillListDefaultsToMetadataAndCanIncludeDocument(
	t *testing.T,
) {
	fixture := newSkillWorkflowFixture(t)
	fixture.bootstrapBuiltins(t)

	metadata, err := fixture.api.ListSkills(
		t.Context(),
		skillConsumerAPI.ListSkillsRequest{
			RootID: documentTopology.BuiltinRootID(),
		},
	)
	requireNoError(t, err)
	if len(metadata) == 0 {
		t.Fatal("built-in Skill list is empty")
	}
	for _, item := range metadata {
		if item.Document != nil {
			t.Fatalf(
				"metadata Skill list unexpectedly includes document for %q",
				item.Name,
			)
		}
	}

	withDocuments, err := fixture.api.ListSkills(
		t.Context(),
		skillConsumerAPI.ListSkillsRequest{
			RootID:          documentTopology.BuiltinRootID(),
			IncludeDocument: true,
		},
	)
	requireNoError(t, err)
	if len(withDocuments) != len(metadata) {
		t.Fatalf(
			"Skill list with documents=%d, metadata list=%d",
			len(withDocuments),
			len(metadata),
		)
	}

	foundDocument := false
	for _, item := range withDocuments {
		if item.Document == nil {
			continue
		}
		foundDocument = true
		if item.Document.Name != string(item.Name) {
			t.Fatalf(
				"Skill document name=%q, list name=%q",
				item.Document.Name,
				item.Name,
			)
		}
	}
	if !foundDocument {
		t.Fatal("Skill list with IncludeDocument has no documents")
	}
}
