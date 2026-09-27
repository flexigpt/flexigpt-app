package consumerapi_test

import (
	"testing"

	agentConsumerAPI "github.com/flexigpt/flexigpt-app/internal/agent/store/consumerapi"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
)

func TestAgentListDefaultsToMetadataAndCanIncludeDocument(
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
		if item.Document != nil {
			t.Fatalf(
				"metadata Agent list unexpectedly includes document for %q",
				item.Name,
			)
		}
	}

	withDocuments, err := harness.api.ListAgents(
		t.Context(),
		agentConsumerAPI.ListAgentsRequest{
			RootID:          documentTopology.BuiltinRootID(),
			IncludeDocument: true,
		},
	)
	requireNoError(t, err)
	if len(withDocuments) != len(metadata) {
		t.Fatalf(
			"Agent list with documents=%d, metadata list=%d",
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
				"Agent document name=%q, list name=%q",
				item.Document.Name,
				item.Name,
			)
		}
	}
	if !foundDocument {
		t.Fatal("Agent list with IncludeDocument has no documents")
	}
}
