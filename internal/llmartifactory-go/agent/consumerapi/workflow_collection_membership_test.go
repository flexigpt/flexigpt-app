package consumerapi_test

import (
	"testing"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	agentConsumerAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/consumerapi"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
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
		plugin.CreateRequest{
			RootID:      topology.UserRootID(),
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

	primaryRef := first.Plugin.Artifact.Ref()
	secondaryWithMember, err := harness.api.AddAgentCollectionArtifactMember(
		t.Context(),
		plugin.AddArtifactMemberRequest{
			Plugin:           secondary.Artifact.Ref(),
			ExpectedRevision: secondary.Artifact.Revision,
			Artifact:         first.Agent.Ref,
		},
	)
	requireNoError(t, err)
	if len(secondaryWithMember.Members) != 1 {
		t.Fatalf(
			"secondary Plugin members = %#v, want one member",
			secondaryWithMember.Members,
		)
	}

	_, err = harness.api.AddAgentCollectionArtifactMember(
		t.Context(),
		plugin.AddArtifactMemberRequest{
			Plugin:           secondaryWithMember.Artifact.Ref(),
			ExpectedRevision: secondaryWithMember.Artifact.Revision,
			Artifact:         first.Agent.Ref,
		},
	)
	requireErrorIs(t, err, spec.ErrIdentityConflict)

	secondaryRef := secondaryWithMember.Artifact.Ref()
	requireCollectionContainsAgent(
		t,
		harness,
		primaryRef,
		first.Agent.Ref,
	)
	requireCollectionContainsAgent(
		t,
		harness,
		secondaryRef,
		first.Agent.Ref,
	)

	requireNoError(
		t,
		harness.api.DeleteManagedAgent(
			t.Context(),
			agentConsumerAPI.ManagedAgentDeleteRequest{
				Agent:            first.Agent.Ref,
				ExpectedRevision: first.Agent.Revision,
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
		t.Fatal("Agent deletion unexpectedly detached Plugin memberships")
	}

	assertCollectionDoesNotContainAgent(
		t,
		harness,
		primaryRef,
		first.Agent.Ref,
	)
	assertCollectionDoesNotContainAgent(
		t,
		harness,
		secondaryRef,
		first.Agent.Ref,
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
		restored.Agent.Ref,
	)
	requireCollectionContainsAgent(
		t,
		harness,
		secondaryRef,
		restored.Agent.Ref,
	)
	requireCollectionPlanComplete(t, harness, primaryRef, true)
	requireCollectionPlanComplete(t, harness, secondaryRef, true)
}

func TestWorkflow_AgentCollectionCanAddAndRemoveBuiltinReference(
	t *testing.T,
) {
	harness, collectionValue := newManagedImportFixture(
		t,
		"builtin-member-plugin",
	)

	builtinAgents, err := harness.api.ListAgents(
		t.Context(),
		agentConsumerAPI.ListAgentsRequest{
			RootID: topology.BuiltinRootID(),
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
		plugin.AddMemberRequest{
			Plugin:           collectionValue.Artifact.Ref(),
			ExpectedRevision: collectionValue.Artifact.Revision,
			Member: plugin.MemberReference{
				Type:  declaration.TypeAgent,
				Name:  builtinAgent.Name,
				Scope: declaration.LookupScopeBuiltin,
			},
		},
	)
	requireNoError(t, err)

	// Plugin-filtered ListAgents is root-local. The member refers to a
	// protected built-in in another Root, so assert it through resolution.
	requireCollectionCapabilityContainsAgent(
		t,
		harness,
		added.Artifact.Ref(),
		builtinAgent.Ref,
	)

	memberIndex := requireAgentCollectionMemberIndex(
		t,
		added.Members,
		builtinAgent.Name,
	)
	removed, err := harness.api.RemoveAgentCollectionMember(
		t.Context(),
		plugin.RemoveMemberRequest{
			Plugin:           added.Artifact.Ref(),
			ExpectedRevision: added.Artifact.Revision,
			Index:            memberIndex,
		},
	)
	requireNoError(t, err)

	if len(removed.Members) != 0 {
		t.Fatalf(
			"Plugin members after built-in removal = %#v, want empty",
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
	collectionRef artifactModel.ArtifactRef,
	agentRef artifactModel.ArtifactRef,
) {
	t.Helper()

	agents, err := harness.api.ListAgents(
		t.Context(),
		agentConsumerAPI.ListAgentsRequest{
			RootID: topology.UserRootID(),
			Plugin: &collectionRef,
		},
	)
	requireNoError(t, err)
	if !containsAgent(agents, agentRef) {
		t.Fatalf(
			"Plugin %q does not contain Agent %q",
			collectionRef,
			agentRef,
		)
	}
}

func requireCollectionCapabilityContainsAgent(
	t *testing.T,
	harness *workflowHarness,
	collectionRef artifactModel.ArtifactRef,
	agentRef artifactModel.ArtifactRef,
) {
	t.Helper()

	plan, err := harness.api.ListAgentCollectionMembers(
		t.Context(),
		collectionRef,
	)
	requireNoError(t, err)

	for _, occurrence := range plan.Occurrences {
		if occurrence.Type != declaration.TypeAgent ||
			occurrence.Target == nil ||
			occurrence.Target.Form != composition.TargetFormArtifact ||
			occurrence.Target.Artifact == nil ||
			*occurrence.Target.Artifact != agentRef {
			continue
		}
		return
	}

	t.Fatalf(
		"Plugin %q capability plan does not resolve Agent %q: %#v",
		collectionRef,
		agentRef,
		plan.Occurrences,
	)
}

func assertCollectionDoesNotContainAgent(
	t *testing.T,
	harness *workflowHarness,
	collectionRef artifactModel.ArtifactRef,
	agentRef artifactModel.ArtifactRef,
) {
	t.Helper()

	agents, err := harness.api.ListAgents(
		t.Context(),
		agentConsumerAPI.ListAgentsRequest{
			RootID: topology.UserRootID(),
			Plugin: &collectionRef,
		},
	)
	requireNoError(t, err)
	if containsAgent(agents, agentRef) {
		t.Fatalf(
			"Plugin %q unexpectedly contains Agent %q",
			collectionRef,
			agentRef,
		)
	}
}

func requireCollectionPlanComplete(
	t *testing.T,
	harness *workflowHarness,
	collectionRef artifactModel.ArtifactRef,
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
			"Plugin %q capability completeness = %t, want %t: %#v",
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
	collectionRef artifactModel.ArtifactRef,
) {
	t.Helper()

	for _, value := range values {
		if value.Plugin == collectionRef {
			return
		}
	}
	t.Fatalf(
		"restored memberships %#v do not contain Plugin %q",
		values,
		collectionRef,
	)
}
