package integration

import (
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	agentAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent"
)

func TestAgentListReturnsConsumerMetadataOnly(
	t *testing.T,
) {
	harness := newWorkflowHarness(t)
	harness.installBundledAgents(t)

	metadata, err := harness.api.ListAgents(
		t.Context(),
		agentAPI.ListAgentsRequest{
			RootID: topology.BuiltinRootID(),
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
