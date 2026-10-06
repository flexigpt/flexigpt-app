package agent_test

import (
	"testing"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	agentAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	pluginAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
)

func TestWorkflow_AgentPluginMembershipRestoresAfterReimport(
	t *testing.T,
) {
	harness, primary := newManagedImportFixture(
		t,
		"restoration-primary",
	)

	secondary, err := harness.api.CreateAgentPlugin(
		t.Context(),
		pluginAPI.CreateRequest{
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
		agentAPI.AgentImportCommitRequest{
			Prepared:                  firstPreview.Prepared,
			PreparedFingerprint:       firstPreview.PreparedFingerprint,
			AcceptedConfirmationCodes: firstPreview.RequiredConfirmationCodes,
		},
	)
	requireNoError(t, err)

	primaryRef := first.Plugin.Artifact.Ref()
	secondaryWithMember, err := harness.api.AddAgentPluginArtifactMember(
		t.Context(),
		pluginAPI.AddArtifactMemberRequest{
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

	_, err = harness.api.AddAgentPluginArtifactMember(
		t.Context(),
		pluginAPI.AddArtifactMemberRequest{
			Plugin:           secondaryWithMember.Artifact.Ref(),
			ExpectedRevision: secondaryWithMember.Artifact.Revision,
			Artifact:         first.Agent.Ref,
		},
	)
	requireErrorIs(t, err, spec.ErrIdentityConflict)

	secondaryRef := secondaryWithMember.Artifact.Ref()
	requirePluginContainsAgent(
		t,
		harness,
		primaryRef,
		first.Agent.Ref,
	)
	requirePluginContainsAgent(
		t,
		harness,
		secondaryRef,
		first.Agent.Ref,
	)

	requireNoError(
		t,
		harness.api.DeleteManagedAgent(
			t.Context(),
			agentAPI.ManagedAgentDeleteRequest{
				Agent:            first.Agent.Ref,
				ExpectedRevision: first.Agent.Revision,
			},
		),
	)

	primaryAfterDelete, err := harness.api.GetAgentPlugin(
		t.Context(),
		primaryRef,
	)
	requireNoError(t, err)
	secondaryAfterDelete, err := harness.api.GetAgentPlugin(
		t.Context(),
		secondaryRef,
	)
	requireNoError(t, err)

	if len(primaryAfterDelete.Members) == 0 ||
		len(secondaryAfterDelete.Members) == 0 {
		t.Fatal("Agent deletion unexpectedly detached Plugin memberships")
	}

	assertPluginDoesNotContainAgent(
		t,
		harness,
		primaryRef,
		first.Agent.Ref,
	)
	assertPluginDoesNotContainAgent(
		t,
		harness,
		secondaryRef,
		first.Agent.Ref,
	)
	requirePluginCapabilityPlanComplete(t, harness, primaryRef, false)
	requirePluginCapabilityPlanComplete(t, harness, secondaryRef, false)

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
		agentAPI.AgentImportCommitRequest{
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

	requirePluginContainsAgent(
		t,
		harness,
		primaryRef,
		restored.Agent.Ref,
	)
	requirePluginContainsAgent(
		t,
		harness,
		secondaryRef,
		restored.Agent.Ref,
	)
	requirePluginCapabilityPlanComplete(t, harness, primaryRef, true)
	requirePluginCapabilityPlanComplete(t, harness, secondaryRef, true)
}

func TestWorkflow_AgentPluginCanAddAndRemoveBuiltinReference(
	t *testing.T,
) {
	harness, pluginValue := newManagedImportFixture(
		t,
		"builtin-member-plugin",
	)

	builtinAgents, err := harness.api.ListAgents(
		t.Context(),
		agentAPI.ListAgentsRequest{
			RootID: topology.BuiltinRootID(),
		},
	)
	requireNoError(t, err)

	builtinAgent := requireNamedAgent(
		t,
		builtinAgents,
		"local-dev-workspace",
	)

	added, err := harness.api.AddAgentPluginMember(
		t.Context(),
		pluginAPI.AddMemberRequest{
			Plugin:           pluginValue.Artifact.Ref(),
			ExpectedRevision: pluginValue.Artifact.Revision,
			Member: pluginAPI.MemberReference{
				Type:  declaration.TypeAgent,
				Name:  builtinAgent.Name,
				Scope: declaration.LookupScopeBuiltin,
			},
		},
	)
	requireNoError(t, err)

	// Plugin-filtered ListAgents is root-local. The member refers to a
	// protected built-in in another Root, so assert it through resolution.
	requirePluginCapabilityContainsAgent(
		t,
		harness,
		added.Artifact.Ref(),
		builtinAgent.Ref,
	)

	memberIndex := requireAgentPluginMemberIndex(
		t,
		added.Members,
		builtinAgent.Name,
	)
	removed, err := harness.api.RemoveAgentPluginMember(
		t.Context(),
		pluginAPI.RemoveMemberRequest{
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
		harness.api.DeleteAgentPlugin(
			t.Context(),
			removed.Artifact.Ref(),
			removed.Artifact.Revision,
		),
	)
}

func requirePluginContainsAgent(
	t *testing.T,
	harness *workflowHarness,
	pluginRef artifactModel.ArtifactRef,
	agentRef artifactModel.ArtifactRef,
) {
	t.Helper()

	agents, err := harness.api.ListAgents(
		t.Context(),
		agentAPI.ListAgentsRequest{
			RootID: topology.UserRootID(),
			Plugin: &pluginRef,
		},
	)
	requireNoError(t, err)
	if !containsAgent(agents, agentRef) {
		t.Fatalf(
			"Plugin %q does not contain Agent %q",
			pluginRef,
			agentRef,
		)
	}
}

func requirePluginCapabilityContainsAgent(
	t *testing.T,
	harness *workflowHarness,
	pluginRef artifactModel.ArtifactRef,
	agentRef artifactModel.ArtifactRef,
) {
	t.Helper()

	plan, err := harness.api.ResolveAgentPluginCapabilities(
		t.Context(),
		pluginRef,
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
		pluginRef,
		agentRef,
		plan.Occurrences,
	)
}

func assertPluginDoesNotContainAgent(
	t *testing.T,
	harness *workflowHarness,
	pluginRef artifactModel.ArtifactRef,
	agentRef artifactModel.ArtifactRef,
) {
	t.Helper()

	agents, err := harness.api.ListAgents(
		t.Context(),
		agentAPI.ListAgentsRequest{
			RootID: topology.UserRootID(),
			Plugin: &pluginRef,
		},
	)
	requireNoError(t, err)
	if containsAgent(agents, agentRef) {
		t.Fatalf(
			"Plugin %q unexpectedly contains Agent %q",
			pluginRef,
			agentRef,
		)
	}
}

func requirePluginCapabilityPlanComplete(
	t *testing.T,
	harness *workflowHarness,
	pluginRef artifactModel.ArtifactRef,
	expected bool,
) {
	t.Helper()

	plan, err := harness.api.ResolveAgentPluginCapabilities(
		t.Context(),
		pluginRef,
	)
	requireNoError(t, err)
	if plan.Complete != expected {
		t.Fatalf(
			"Plugin %q capability completeness = %t, want %t: %#v",
			pluginRef,
			plan.Complete,
			expected,
			plan.Occurrences,
		)
	}
}

func requireRestoredMembership(
	t *testing.T,
	values []agentAPI.AgentRestoredMembership,
	pluginRef artifactModel.ArtifactRef,
) {
	t.Helper()

	for _, value := range values {
		if value.Plugin == pluginRef {
			return
		}
	}
	t.Fatalf(
		"restored memberships %#v do not contain Plugin %q",
		values,
		pluginRef,
	)
}
