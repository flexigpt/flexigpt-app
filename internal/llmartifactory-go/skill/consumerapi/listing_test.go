package consumerapi_test

import (
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	skillConsumerAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/consumerapi"
)

func TestSkillListReturnsConsumerMetadataOnly(
	t *testing.T,
) {
	fixture := newSkillWorkflowFixture(t)
	fixture.bootstrapBuiltins(t)

	metadata, err := fixture.api.ListSkills(
		t.Context(),
		skillConsumerAPI.ListSkillsRequest{
			RootID: topology.BuiltinRootID(),
		},
	)
	requireNoError(t, err)
	if len(metadata) == 0 {
		t.Fatal("built-in Skill list is empty")
	}

	for _, item := range metadata {
		if item.DefinitionDigest == "" {
			t.Fatalf(
				"Skill list item %q has no Definition digest",
				item.Name,
			)
		}
		if !item.BuiltIn {
			t.Fatalf("built-in Skill %q is not marked BuiltIn", item.Name)
		}
	}
}
