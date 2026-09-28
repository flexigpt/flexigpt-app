package consumerapi_test

import (
	"testing"

	agentConsumerAPI "github.com/flexigpt/flexigpt-app/internal/agent/store/consumerapi"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
)

func TestAgentListReturnsConsumerMetadataOnly(
	t *testing.T,
) {
	harness := newWorkflowHarness(t)
	harness.installBundledAgents(t)

	metadata, err := harness.api.ListAgents(
		t.Context(),
		agentConsumerAPI.ListAgentsRequest{
			RootID: documentTopology.BuiltinRootID(),
		},
	)
	requireNoError(t, err)
	if len(metadata) == 0 {
		t.Fatal("built-in Agent list is empty")
	}

	for _, item := range metadata {
		if item.DefinitionDigest == "" {
			t.Fatalf(
				"Agent list item %q has no Definition digest",
				item.Name,
			)
		}
		if !item.BuiltIn {
			t.Fatalf("built-in Agent %q is not marked BuiltIn", item.Name)
		}
	}
}
