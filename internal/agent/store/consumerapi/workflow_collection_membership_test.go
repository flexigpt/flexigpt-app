package consumerapi_test

import (
	"testing"

	agentConsumerAPI "github.com/flexigpt/flexigpt-app/internal/agent/store/consumerapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/collection"
)

func TestWorkflow_AgentCollectionMembershipRestoresAfterReimport(
	t *testing.T,
) {
	harness, primary := newManagedImportFixture(
		t,
		"restoration-primary",
	)

	secondary, err := harness.api.CreateAgentCollection(
		t.Context(),
		collection.CreateRequest{
			RootID:      documentTopology.UserRootID(),
			Name:        "restoration-secondary",
			DisplayName: "Restoration Secondary",
		},
	)
	requireNoError(t, err)

	path := writeSimpleManagedAgentImport(
		t,
		"restored-agent",
	)
	firstPreview := previewManagedAgentImport(
		t,
		harness,
		primary,
		path,
	)
	if !firstPreview.CanImport {
		t.Fatalf("first import cannot proceed: %#v", firstPreview.Issues)
	}

	first, err := harness.api.CommitAgentImport(
		t.Context(),
		agentConsumerAPI.AgentImportCommitRequest{
			Prepared:                  firstPreview.Prepared,
			PreparedFingerprint:       firstPreview.PreparedFingerprint,
			AcceptedConfirmationCodes: firstPreview.RequiredConfirmationCodes,
		},
	)
	requireNoError(t, err)

	primaryRef := first.Collection.Artifact.Ref()
	secondaryWithMember, err := harness.api.AddAgentCollectionArtifactMember(
		t.Context(),
		collection.AddArtifactMemberRequest{
			Collection:       secondary.Artifact.Ref(),
			ExpectedRevision: secondary.Artifact.Revision,
			Artifact:         first.Agent.Artifact.Ref(),
		},
	)
	requireNoError(t, err)
	if len(secondaryWithMember.Members) != 1 {
		t.Fatalf(
			"secondary Collection members = %#v, want one member",
			secondaryWithMember.Members,
		)
	}

	_, err = harness.api.AddAgentCollectionArtifactMember(
		t.Context(),
		collection.AddArtifactMemberRequest{
			Collection:       secondaryWithMember.Artifact.Ref(),
			ExpectedRevision: secondaryWithMember.Artifact.Revision,
			Artifact:         first.Agent.Artifact.Ref(),
		},
	)
	requireErrorIs(t, err, basespec.ErrIdentityConflict)

	secondaryRef := secondaryWithMember.Artifact.Ref()
	requireCollectionContainsAgent(
		t,
		harness,
		primaryRef,
		first.Agent.Artifact.Ref(),
	)
	requireCollectionContainsAgent(
		t,
		harness,
		secondaryRef,
		first.Agent.Artifact.Ref(),
	)

	requireNoError(
		t,
		harness.api.DeleteManagedAgent(
			t.Context(),
			agentConsumerAPI.ManagedAgentDeleteRequest{
				Agent:            first.Agent.Artifact.Ref(),
				ExpectedRevision: first.Agent.Artifact.Revision,
			},
		),
	)

	primaryAfterDelete, err := harness.api.GetAgentCollection(
		t.Context(),
		primaryRef,
	)
	requireNoError(t, err)
	secondaryAfterDelete, err := harness.api.GetAgentCollection(
		t.Context(),
		secondaryRef,
	)
	requireNoError(t, err)

	if len(primaryAfterDelete.Members) == 0 ||
		len(secondaryAfterDelete.Members) == 0 {
		t.Fatal("Agent deletion unexpectedly detached Collection memberships")
	}

	assertCollectionDoesNotContainAgent(
		t,
		harness,
		primaryRef,
		first.Agent.Artifact.Ref(),
	)
	assertCollectionDoesNotContainAgent(
		t,
		harness,
		secondaryRef,
		first.Agent.Artifact.Ref(),
	)
	requireCollectionPlanComplete(t, harness, primaryRef, false)
	requireCollectionPlanComplete(t, harness, secondaryRef, false)

	restoredPreview := previewManagedAgentImport(
		t,
		harness,
		primaryAfterDelete,
		path,
	)
	if !restoredPreview.CanImport {
		t.Fatalf(
			"re-import preview cannot proceed: %#v",
			restoredPreview.Issues,
		)
	}
	requireRestoredMembership(
		t,
		restoredPreview.RestoredMemberships,
		primaryRef,
	)
	requireRestoredMembership(
		t,
		restoredPreview.RestoredMemberships,
		secondaryRef,
	)

	restored, err := harness.api.CommitAgentImport(
		t.Context(),
		agentConsumerAPI.AgentImportCommitRequest{
			Prepared:                  restoredPreview.Prepared,
			PreparedFingerprint:       restoredPreview.PreparedFingerprint,
			AcceptedConfirmationCodes: restoredPreview.RequiredConfirmationCodes,
		},
	)
	requireNoError(t, err)

	requireRestoredMembership(
		t,
		restored.RestoredMemberships,
		primaryRef,
	)
	requireRestoredMembership(
		t,
		restored.RestoredMemberships,
		secondaryRef,
	)

	requireCollectionContainsAgent(
		t,
		harness,
		primaryRef,
		restored.Agent.Artifact.Ref(),
	)
	requireCollectionContainsAgent(
		t,
		harness,
		secondaryRef,
		restored.Agent.Artifact.Ref(),
	)
	requireCollectionPlanComplete(t, harness, primaryRef, true)
	requireCollectionPlanComplete(t, harness, secondaryRef, true)
}

func TestWorkflow_AgentCollectionCanAddAndRemoveBuiltinReference(
	t *testing.T,
) {
	harness, collectionValue := newManagedImportFixture(
		t,
		"builtin-member-collection",
	)

	builtinAgents, err := harness.api.ListAgents(
		t.Context(),
		agentConsumerAPI.ListAgentsRequest{
			RootID: documentTopology.BuiltinRootID(),
		},
	)
	requireNoError(t, err)

	builtinAgent := requireNamedAgent(
		t,
		builtinAgents,
		"local-dev-workspace",
	)

	added, err := harness.api.AddAgentCollectionMember(
		t.Context(),
		collection.AddMemberRequest{
			Collection:       collectionValue.Artifact.Ref(),
			ExpectedRevision: collectionValue.Artifact.Revision,
			Member: collection.MemberReference{
				Type:  declaration.TypeAgent,
				Name:  builtinAgent.Name,
				Scope: declaration.LookupScopeBuiltin,
			},
		},
	)
	requireNoError(t, err)

	requireCollectionContainsAgent(
		t,
		harness,
		added.Artifact.Ref(),
		builtinAgent.Artifact.Ref(),
	)

	memberIndex := requireAgentCollectionMemberIndex(
		t,
		added.Members,
		builtinAgent.Name,
	)
	removed, err := harness.api.RemoveAgentCollectionMember(
		t.Context(),
		collection.RemoveMemberRequest{
			Collection:       added.Artifact.Ref(),
			ExpectedRevision: added.Artifact.Revision,
			Index:            memberIndex,
		},
	)
	requireNoError(t, err)

	if len(removed.Members) != 0 {
		t.Fatalf(
			"Collection members after built-in removal = %#v, want empty",
			removed.Members,
		)
	}

	requireNoError(
		t,
		harness.api.DeleteAgentCollection(
			t.Context(),
			removed.Artifact.Ref(),
			removed.Artifact.Revision,
		),
	)
}

func requireCollectionContainsAgent(
	t *testing.T,
	harness *workflowHarness,
	collectionRef artifact.ArtifactRef,
	agentRef artifact.ArtifactRef,
) {
	t.Helper()

	agents, err := harness.api.ListAgents(
		t.Context(),
		agentConsumerAPI.ListAgentsRequest{
			RootID:     documentTopology.UserRootID(),
			Collection: &collectionRef,
		},
	)
	requireNoError(t, err)
	if !containsAgent(agents, agentRef) {
		t.Fatalf(
			"Collection %q does not contain Agent %q",
			collectionRef,
			agentRef,
		)
	}
}

func assertCollectionDoesNotContainAgent(
	t *testing.T,
	harness *workflowHarness,
	collectionRef artifact.ArtifactRef,
	agentRef artifact.ArtifactRef,
) {
	t.Helper()

	agents, err := harness.api.ListAgents(
		t.Context(),
		agentConsumerAPI.ListAgentsRequest{
			RootID:     documentTopology.UserRootID(),
			Collection: &collectionRef,
		},
	)
	requireNoError(t, err)
	if containsAgent(agents, agentRef) {
		t.Fatalf(
			"Collection %q unexpectedly contains Agent %q",
			collectionRef,
			agentRef,
		)
	}
}

func requireCollectionPlanComplete(
	t *testing.T,
	harness *workflowHarness,
	collectionRef artifact.ArtifactRef,
	expected bool,
) {
	t.Helper()

	plan, err := harness.api.ListAgentCollectionMembers(
		t.Context(),
		collectionRef,
	)
	requireNoError(t, err)
	if plan.Complete != expected {
		t.Fatalf(
			"Collection %q capability completeness = %t, want %t: %#v",
			collectionRef,
			plan.Complete,
			expected,
			plan.Occurrences,
		)
	}
}

func requireRestoredMembership(
	t *testing.T,
	values []agentConsumerAPI.AgentRestoredMembership,
	collectionRef artifact.ArtifactRef,
) {
	t.Helper()

	for _, value := range values {
		if value.Collection == collectionRef {
			return
		}
	}
	t.Fatalf(
		"restored memberships %#v do not contain Collection %q",
		values,
		collectionRef,
	)
}
