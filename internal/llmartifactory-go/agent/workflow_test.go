package agent_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	agentAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	pluginAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
)

func TestWorkflow_EmptyStore_InstallsAndReadsBundledAgents(
	t *testing.T,
) {
	harness := newWorkflowHarness(t)

	initial, err := harness.api.ListAgentsForManagement(t.Context())
	requireNoError(t, err)
	if len(initial) != 0 {
		t.Fatalf("initial management Agent list = %#v, want empty", initial)
	}

	harness.installBundledAgents(t)

	builtinAgents, err := harness.api.ListAgents(
		t.Context(),
		agentAPI.ListAgentsRequest{
			RootID: topology.BuiltinRootID(),
		},
	)
	requireNoError(t, err)

	known := requireNamedAgent(
		t,
		builtinAgents,
		spec.LogicalName("local-dev-workspace"),
	)
	if !known.BuiltIn {
		t.Fatalf("bundled Agent BuiltIn = false")
	}

	resolution, err := harness.api.ResolveAgent(
		t.Context(),
		known.Ref,
	)
	requireNoError(t, err)
	if !resolution.Capabilities.Complete {
		t.Fatalf(
			"bundled Agent capability plan is incomplete: %#v",
			resolution.Capabilities.Occurrences,
		)
	}

	exported, err := harness.api.ExportAgent(
		t.Context(),
		agentAPI.AgentExportRequest{
			Agent: known.Ref,
		},
	)
	requireNoError(t, err)
	if exported.Type != declaration.TypeAgent {
		t.Fatalf("export type = %q, want %q", exported.Type, declaration.TypeAgent)
	}
	if exported.Name != known.Name {
		t.Fatalf("export name = %q, want %q", exported.Name, known.Name)
	}
	if exported.MediaType != "application/yaml" {
		t.Fatalf(
			"export media type = %q, want application/yaml",
			exported.MediaType,
		)
	}
	if exported.ContentDigest == "" || exported.DefinitionDigest == "" {
		t.Fatalf("export is missing content or definition digest")
	}
	if !strings.Contains(exported.Content, "local-dev-workspace") {
		t.Fatalf("export does not contain the expected Agent name")
	}
	if !exported.BuiltIn {
		t.Fatalf("built-in Agent export has incorrect ownership flags")
	}

	userVisible, err := harness.api.ListAgents(
		t.Context(),
		agentAPI.ListAgentsRequest{
			RootID:         topology.UserRootID(),
			IncludeBuiltin: true,
		},
	)
	requireNoError(t, err)
	requireNamedAgent(
		t,
		userVisible,
		spec.LogicalName("local-dev-workspace"),
	)

	before := append([]agentAPI.AgentListItem(nil), builtinAgents...)

	harness.ensureBundledAgents(t)

	after, err := harness.api.ListAgents(
		t.Context(),
		agentAPI.ListAgentsRequest{
			RootID: topology.BuiltinRootID(),
		},
	)
	requireNoError(t, err)
	requireSameAgentRevisions(t, before, after)

	inspection, err := harness.store.Refresh.InspectSource(
		t.Context(),
		topology.BuiltinRootID(),
		topology.BuiltinPackageSourceID(),
	)
	requireNoError(t, err)
	if !inspection.IsCurrent() {
		t.Fatalf("bundled Agent package Source is not current: %#v", inspection)
	}
}

func TestWorkflow_UserPlugin_ManagedAgentCRUD(
	t *testing.T,
) {
	harness := newWorkflowHarness(t)
	harness.installBundledAgents(t)

	baselineEnsurer, err := agentAPI.NewBaselineEnsurer(
		harness.api,
	)
	requireNoError(t, err)

	baseline, err := baselineEnsurer.EnsureAgentBaselinePlugin(
		t.Context(),
		topology.UserRootID(),
	)
	requireNoError(t, err)
	if !baseline.Baseline {
		t.Fatalf("Agent baseline Plugin is not marked as baseline")
	}
	if !harness.api.IsManagedAgentPlugin(baseline) {
		t.Fatalf("Agent baseline Plugin is not recognized as managed")
	}

	created, err := harness.api.CreateAgentPlugin(
		t.Context(),
		pluginAPI.CreateRequest{
			Name:        "workflow-plugin",
			DisplayName: "Workflow Plugin",
			Description: "Initial workflow Plugin description.",
		},
	)
	requireNoError(t, err)
	if created.Artifact.RootID != topology.UserRootID() {
		t.Fatalf(
			"default Plugin Root = %q, want %q",
			created.Artifact.RootID,
			topology.UserRootID(),
		)
	}
	if !created.Editable || !harness.api.IsManagedAgentPlugin(created) {
		t.Fatalf("new Agent Plugin is not editable managed state")
	}

	updated, err := harness.api.UpdateAgentPlugin(
		t.Context(),
		pluginAPI.UpdateRequest{
			Plugin:           created.Artifact.Ref(),
			ExpectedRevision: created.Artifact.Revision,
			DisplayName:      "Workflow Plugin Updated",
			Description:      "Updated workflow Plugin description.",
		},
	)
	requireNoError(t, err)

	_, err = harness.store.Refresh.RefreshSource(
		t.Context(),
		topology.UserRootID(),
		updated.Artifact.Binding.SourceID,
	)
	requireNoError(t, err)

	readPlugin, err := harness.api.GetAgentPlugin(
		t.Context(),
		updated.Artifact.Ref(),
	)
	requireNoError(t, err)
	if readPlugin.DisplayName != "Workflow Plugin Updated" {
		t.Fatalf(
			"Plugin display name = %q",
			readPlugin.DisplayName,
		)
	}
	if readPlugin.Description != "Updated workflow Plugin description." {
		t.Fatalf(
			"Plugin description = %q",
			readPlugin.Description,
		)
	}

	disabledPlugin, err := harness.api.SetAgentPluginEnabled(
		t.Context(),
		readPlugin.Artifact.Ref(),
		readPlugin.Artifact.Revision,
		false,
	)
	requireNoError(t, err)
	if disabledPlugin.Artifact.Enabled {
		t.Fatalf("Plugin remained enabled after disable")
	}

	enabledPlugin, err := harness.api.SetAgentPluginEnabled(
		t.Context(),
		disabledPlugin.Artifact.Ref(),
		disabledPlugin.Artifact.Revision,
		true,
	)
	requireNoError(t, err)
	if !enabledPlugin.Artifact.Enabled {
		t.Fatalf("Plugin remained disabled after enable")
	}

	destinations, err := harness.api.ListAgentImportDestinations(
		t.Context(),
		topology.UserRootID(),
	)
	requireNoError(t, err)
	destination := requireImportDestination(
		t,
		destinations,
		enabledPlugin.Artifact.Ref(),
	)
	if destination.PluginRevision != enabledPlugin.Artifact.Revision {
		t.Fatalf(
			"destination Plugin revision = %d, want %d",
			destination.PluginRevision,
			enabledPlugin.Artifact.Revision,
		)
	}
	if destination.PluginDisplayName != "Workflow Plugin Updated" {
		t.Fatalf(
			"destination Plugin display name = %q",
			destination.PluginDisplayName,
		)
	}

	allDestinations, err := harness.api.ListAgentImportDestinationsForManagement(
		t.Context(),
	)
	requireNoError(t, err)
	managementDestination := requireImportDestination(
		t,
		allDestinations,
		enabledPlugin.Artifact.Ref(),
	)
	if managementDestination.RootDisplayName == "" {
		t.Fatalf("management import destination has no Root display name")
	}
	if managementDestination.PluginDisplayName != "Workflow Plugin Updated" {
		t.Fatalf(
			"management destination Plugin display name = %q",
			managementDestination.PluginDisplayName,
		)
	}

	inputPath := filepath.Join(t.TempDir(), "workflow-agent.yaml")
	requireNoError(
		t,
		os.WriteFile(
			inputPath,
			[]byte(`type: agent
name: workflow-agent
displayName: Workflow Agent
description: A managed Agent used by the consumer API workflow test.
members:
  - type: text
    name: workflow-agent-instructions
    insert: instructions
    parameters:
      mediaType: text/plain
      content: workflow instructions
`),
			0o600,
		),
	)

	preview, err := harness.api.PreviewAgentImport(
		t.Context(),
		agentAPI.AgentImportPreviewRequest{
			Path:                   inputPath,
			Plugin:                 enabledPlugin.Artifact.Ref(),
			ExpectedPluginRevision: enabledPlugin.Artifact.Revision,
		},
	)
	requireNoError(t, err)
	if !preview.CanImport {
		t.Fatalf("managed Agent preview cannot import: %#v", preview.Issues)
	}
	if preview.Prepared == "" || preview.PreparedFingerprint == "" {
		t.Fatalf("managed Agent preview has no prepared envelope")
	}
	if preview.Agent == nil ||
		preview.Agent.Name != spec.LogicalName("workflow-agent") {
		t.Fatalf("managed Agent preview has wrong projected Agent: %#v", preview.Agent)
	}
	if preview.NormalizedYAML == "" {
		t.Fatalf("managed Agent preview has no normalized YAML")
	}
	if len(preview.RequiredConfirmationCodes) != 0 {
		t.Fatalf(
			"simple managed Agent unexpectedly requires confirmation: %#v",
			preview.RequiredConfirmationCodes,
		)
	}

	committed, err := harness.api.CommitAgentImport(
		t.Context(),
		agentAPI.AgentImportCommitRequest{
			Prepared:                  preview.Prepared,
			PreparedFingerprint:       preview.PreparedFingerprint,
			AcceptedConfirmationCodes: preview.RequiredConfirmationCodes,
		},
	)
	requireNoError(t, err)
	if committed.Plugin.Artifact.Ref() !=
		enabledPlugin.Artifact.Ref() {
		t.Fatalf("import committed membership to another Plugin")
	}

	current, err := harness.api.GetAgent(
		t.Context(),
		committed.Agent.Ref,
	)
	requireNoError(t, err)
	if current.Name != spec.LogicalName("workflow-agent") {
		t.Fatalf("read Agent name = %q", current.Name)
	}

	allUserAgents, err := harness.api.ListAgents(
		t.Context(),
		agentAPI.ListAgentsRequest{
			RootID: topology.UserRootID(),
		},
	)
	requireNoError(t, err)
	if !containsAgent(allUserAgents, committed.Agent.Ref) {
		t.Fatalf("managed Agent is missing from the Root Agent list")
	}

	pluginRef := committed.Plugin.Artifact.Ref()
	pluginAgents, err := harness.api.ListAgents(
		t.Context(),
		agentAPI.ListAgentsRequest{
			RootID: topology.UserRootID(),
			Plugin: &pluginRef,
		},
	)
	requireNoError(t, err)
	if !containsAgent(pluginAgents, committed.Agent.Ref) {
		t.Fatalf("managed Agent is missing from its Plugin")
	}

	resolution, err := harness.api.ResolveAgent(
		t.Context(),
		committed.Agent.Ref,
	)
	requireNoError(t, err)
	if !resolution.Capabilities.Complete {
		t.Fatalf(
			"managed Agent capability plan is incomplete: %#v",
			resolution.Capabilities.Occurrences,
		)
	}

	textRef := requireAvailableCapability(
		t,
		resolution.Capabilities,
		declaration.TypeText,
		spec.LogicalName("workflow-agent-instructions"),
	)
	materialized, err := harness.texts.Materialize(t.Context(), textRef)
	requireNoError(t, err)
	if materialized.Content != "workflow instructions" {
		t.Fatalf("materialized Text = %q", materialized.Content)
	}

	exported, err := harness.api.ExportAgent(
		t.Context(),
		agentAPI.AgentExportRequest{
			Agent: committed.Agent.Ref,
		},
	)
	requireNoError(t, err)
	if exported.BuiltIn {
		t.Fatalf("managed Agent export has unexpected management flags")
	}
	if !strings.Contains(exported.Content, "workflow-agent") {
		t.Fatalf("managed Agent export does not contain Agent name")
	}

	disabledAgent, err := harness.api.SetAgentEnabled(
		t.Context(),
		committed.Agent.Ref,
		committed.Agent.Revision,
		false,
	)
	requireNoError(t, err)
	if disabledAgent.Enabled {
		t.Fatalf("Agent remained enabled after disable")
	}

	disabled := false
	disabledAgents, err := harness.api.ListAgents(
		t.Context(),
		agentAPI.ListAgentsRequest{
			RootID:  topology.UserRootID(),
			Enabled: &disabled,
		},
	)
	requireNoError(t, err)
	if !containsAgent(disabledAgents, committed.Agent.Ref) {
		t.Fatalf("disabled Agent is missing from disabled Agent filter")
	}

	_, err = harness.api.SetAgentEnabled(
		t.Context(),
		committed.Agent.Ref,
		committed.Agent.Revision,
		true,
	)
	requireErrorIs(t, err, spec.ErrConflict)

	enabledAgent, err := harness.api.SetAgentEnabled(
		t.Context(),
		disabledAgent.Ref,
		disabledAgent.Revision,
		true,
	)
	requireNoError(t, err)
	if !enabledAgent.Enabled {
		t.Fatalf("Agent remained disabled after enable")
	}

	requireNoError(
		t,
		harness.api.DeleteManagedAgent(
			t.Context(),
			agentAPI.ManagedAgentDeleteRequest{
				Agent:            enabledAgent.Ref,
				ExpectedRevision: enabledAgent.Revision,
			},
		),
	)

	remainingUserAgents, err := harness.api.ListAgents(
		t.Context(),
		agentAPI.ListAgentsRequest{
			RootID: topology.UserRootID(),
		},
	)
	requireNoError(t, err)
	if containsAgent(remainingUserAgents, enabledAgent.Ref) {
		t.Fatalf("deleted Agent remains in Root Agent list")
	}

	remainingPluginAgents, err := harness.api.ListAgents(
		t.Context(),
		agentAPI.ListAgentsRequest{
			RootID: topology.UserRootID(),
			Plugin: &pluginRef,
		},
	)
	requireNoError(t, err)
	if containsAgent(remainingPluginAgents, enabledAgent.Ref) {
		t.Fatalf("deleted Agent remains in Plugin Agent list")
	}

	currentPlugin, err := harness.api.GetAgentPlugin(
		t.Context(),
		pluginRef,
	)
	requireNoError(t, err)
	if len(currentPlugin.Members) == 0 {
		t.Fatal(
			"deleting the managed Agent unexpectedly detached Plugin membership",
		)
	}

	staleMemberships, err := harness.api.ListAgentPluginMembers(
		t.Context(),
		pluginRef,
	)
	requireNoError(t, err)
	if staleMemberships.Complete {
		t.Fatal(
			"Plugin capability plan remained complete after deleting its Agent",
		)
	}

	err = harness.api.DeleteAgentPlugin(
		t.Context(),
		pluginRef,
		currentPlugin.Artifact.Revision,
	)
	requireErrorIs(t, err, spec.ErrConflict)

	memberIndex := requireAgentPluginMemberIndex(
		t,
		currentPlugin.Members,
		spec.LogicalName("workflow-agent"),
	)
	detachedPlugin, err := harness.api.RemoveAgentPluginMember(
		t.Context(),
		pluginAPI.RemoveMemberRequest{
			Plugin:           pluginRef,
			ExpectedRevision: currentPlugin.Artifact.Revision,
			Index:            memberIndex,
		},
	)
	requireNoError(t, err)
	if len(detachedPlugin.Members) != 0 {
		t.Fatalf(
			"Plugin members after detach = %#v, want empty",
			detachedPlugin.Members,
		)
	}

	currentMemberships, err := harness.api.ListAgentPluginMembers(
		t.Context(),
		detachedPlugin.Artifact.Ref(),
	)
	requireNoError(t, err)
	if !currentMemberships.Complete {
		t.Fatalf(
			"empty Plugin capability plan is incomplete: %#v",
			currentMemberships,
		)
	}

	requireNoError(
		t,
		harness.api.DeleteAgentPlugin(
			t.Context(),
			detachedPlugin.Artifact.Ref(),
			detachedPlugin.Artifact.Revision,
		),
	)

	plugins, err := harness.api.ListAgentPlugins(
		t.Context(),
		topology.UserRootID(),
	)
	requireNoError(t, err)
	if containsPlugin(plugins, pluginRef) {
		t.Fatalf("deleted custom Plugin remains in Plugin list")
	}

	remainingDestinations, err := harness.api.ListAgentImportDestinationsForManagement(
		t.Context(),
	)
	requireNoError(t, err)
	if hasImportDestination(remainingDestinations, pluginRef) {
		t.Fatalf("deleted custom Plugin remains an import destination")
	}
}

func requireNamedAgent(
	t *testing.T,
	values []agentAPI.AgentListItem,
	name spec.LogicalName,
) agentAPI.AgentListItem {
	t.Helper()

	for _, value := range values {
		if value.Name == name {
			return value
		}
	}
	t.Fatalf("Agent %q was not found", name)
	return agentAPI.AgentListItem{}
}

func requireImportDestination(
	t *testing.T,
	values []agentAPI.AgentImportDestination,
	ref artifactModel.ArtifactRef,
) agentAPI.AgentImportDestination {
	t.Helper()

	for _, value := range values {
		if value.Plugin == ref {
			return value
		}
	}
	t.Fatalf("Agent import destination %q was not found", ref)
	return agentAPI.AgentImportDestination{}
}

func requireAvailableCapability(
	t *testing.T,
	plan agentAPI.AgentCapabilityPlan,
	declarationType declaration.Type,
	name spec.LogicalName,
) artifactModel.ArtifactRef {
	t.Helper()

	for _, occurrence := range plan.Occurrences {
		if occurrence.Type != declarationType ||
			occurrence.Name != name {
			continue
		}
		if occurrence.Status != composition.ResolutionAvailable ||
			occurrence.Target == nil ||
			occurrence.Target.Form != composition.TargetFormArtifact ||
			occurrence.Target.Artifact == nil {
			t.Fatalf(
				"capability %s/%s is not available: %#v",
				declarationType,
				name,
				occurrence,
			)
		}
		return *occurrence.Target.Artifact
	}

	t.Fatalf(
		"capability %s/%s was not found in %#v",
		declarationType,
		name,
		plan.Occurrences,
	)
	return artifactModel.ArtifactRef{}
}

func requireAgentPluginMemberIndex(
	t *testing.T,
	values []pluginAPI.MemberReference,
	name spec.LogicalName,
) int {
	t.Helper()

	for index, value := range values {
		if value.Type == declaration.TypeAgent &&
			value.Name == name {
			return index
		}
	}

	t.Fatalf(
		"Agent Plugin member %q was not found in %#v",
		name,
		values,
	)
	return 0
}

func containsAgent(
	values []agentAPI.AgentListItem,
	ref artifactModel.ArtifactRef,
) bool {
	for _, value := range values {
		if value.Ref == ref {
			return true
		}
	}
	return false
}

func containsPlugin(
	values []pluginAPI.ListItem,
	ref artifactModel.ArtifactRef,
) bool {
	for _, value := range values {
		if value.Ref == ref {
			return true
		}
	}
	return false
}

func hasImportDestination(
	values []agentAPI.AgentImportDestination,
	ref artifactModel.ArtifactRef,
) bool {
	for _, value := range values {
		if value.Plugin == ref {
			return true
		}
	}
	return false
}
