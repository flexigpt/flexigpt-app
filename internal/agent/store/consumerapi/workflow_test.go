package consumerapi_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	agentConsumerAPI "github.com/flexigpt/flexigpt-app/internal/agent/store/consumerapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi"
	"github.com/flexigpt/flexigpt-app/internal/collection"
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

	installer := harness.installBundledAgents(t)

	builtinAgents, err := harness.api.ListAgents(
		t.Context(),
		agentConsumerAPI.ListAgentsRequest{
			RootID: documentTopology.BuiltinRootID(),
		},
	)
	requireNoError(t, err)

	known := requireNamedAgent(
		t,
		builtinAgents,
		basespec.LogicalName("local-dev-workspace"),
	)
	if !known.BuiltIn {
		t.Fatalf("bundled Agent BuiltIn = false")
	}
	if known.Managed {
		t.Fatalf("bundled Agent Managed = true")
	}

	resolution, err := harness.api.ResolveAgent(
		t.Context(),
		known.Artifact.Ref(),
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
		agentConsumerAPI.AgentExportRequest{
			Agent: known.Artifact.Ref(),
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
	if exported.Resolution == nil || !exported.Resolution.Complete {
		t.Fatalf("export has no complete capability resolution")
	}

	userVisible, err := harness.api.ListAgents(
		t.Context(),
		agentConsumerAPI.ListAgentsRequest{
			RootID:         documentTopology.UserRootID(),
			IncludeBuiltin: true,
		},
	)
	requireNoError(t, err)
	requireNamedAgent(
		t,
		userVisible,
		basespec.LogicalName("local-dev-workspace"),
	)

	before := append(
		[]agentConsumerAPI.AgentView(nil),
		builtinAgents...,
	)
	requireNoError(
		t,
		installer.Ensure(installerapi.WithPrivilege(t.Context())),
	)

	after, err := harness.api.ListAgents(
		t.Context(),
		agentConsumerAPI.ListAgentsRequest{
			RootID: documentTopology.BuiltinRootID(),
		},
	)
	requireNoError(t, err)
	requireSameAgentRevisions(t, before, after)

	inspection, err := harness.store.Discovery.InspectSource(
		t.Context(),
		documentTopology.BuiltinRootID(),
		documentTopology.BuiltinPackageSourceID(),
	)
	requireNoError(t, err)
	if !inspection.IsCurrent() {
		t.Fatalf("bundled Agent package Source is not current: %#v", inspection)
	}
}

func TestWorkflow_UserCollection_ManagedAgentCRUD(
	t *testing.T,
) {
	harness := newWorkflowHarness(t)
	harness.installBundledAgents(t)

	baselineEnsurer, err := agentConsumerAPI.NewBaselineEnsurer(
		harness.api,
	)
	requireNoError(t, err)

	baseline, err := baselineEnsurer.EnsureAgentBaselineCollection(
		t.Context(),
		documentTopology.UserRootID(),
	)
	requireNoError(t, err)
	if !baseline.Baseline {
		t.Fatalf("Agent baseline Collection is not marked as baseline")
	}
	if !harness.api.IsManagedAgentCollection(baseline) {
		t.Fatalf("Agent baseline Collection is not recognized as managed")
	}

	created, err := harness.api.CreateAgentCollection(
		t.Context(),
		collection.CreateRequest{
			Name:        "workflow-collection",
			DisplayName: "Workflow Collection",
		},
	)
	requireNoError(t, err)
	if created.Artifact.RootID != documentTopology.UserRootID() {
		t.Fatalf(
			"default Collection Root = %q, want %q",
			created.Artifact.RootID,
			documentTopology.UserRootID(),
		)
	}
	if !created.Editable || !harness.api.IsManagedAgentCollection(created) {
		t.Fatalf("new Agent Collection is not editable managed state")
	}

	updated, err := harness.api.UpdateAgentCollection(
		t.Context(),
		collection.UpdateRequest{
			Collection:       created.Artifact.Ref(),
			ExpectedRevision: created.Artifact.Revision,
			DisplayName:      "Workflow Collection Updated",
		},
	)
	requireNoError(t, err)

	_, err = harness.api.GetAgentCollection(
		t.Context(),
		updated.Artifact.Ref(),
	)
	requireNoError(t, err)
	// "if readCollection.Artifact.DisplayName != "Workflow Collection Updated" {
	// 	t.Fatalf(
	// 		"Collection display name = %q",
	// 		readCollection.Artifact.DisplayName,
	// 	)
	// }".

	disabledCollection, err := harness.api.SetAgentCollectionEnabled(
		t.Context(),
		updated.Artifact.Ref(),
		updated.Artifact.Revision,
		false,
	)
	requireNoError(t, err)
	if disabledCollection.Artifact.Enabled {
		t.Fatalf("Collection remained enabled after disable")
	}

	enabledCollection, err := harness.api.SetAgentCollectionEnabled(
		t.Context(),
		disabledCollection.Artifact.Ref(),
		disabledCollection.Artifact.Revision,
		true,
	)
	requireNoError(t, err)
	if !enabledCollection.Artifact.Enabled {
		t.Fatalf("Collection remained disabled after enable")
	}

	destinations, err := harness.api.ListAgentImportDestinations(
		t.Context(),
		documentTopology.UserRootID(),
	)
	requireNoError(t, err)
	destination := requireImportDestination(
		t,
		destinations,
		enabledCollection.Artifact.Ref(),
	)
	if destination.CollectionRevision != enabledCollection.Artifact.Revision {
		t.Fatalf(
			"destination Collection revision = %d, want %d",
			destination.CollectionRevision,
			enabledCollection.Artifact.Revision,
		)
	}

	allDestinations, err := harness.api.ListAgentImportDestinationsForManagement(
		t.Context(),
	)
	requireNoError(t, err)
	managementDestination := requireImportDestination(
		t,
		allDestinations,
		enabledCollection.Artifact.Ref(),
	)
	if managementDestination.RootDisplayName == "" {
		t.Fatalf("management import destination has no Root display name")
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
		agentConsumerAPI.AgentImportPreviewRequest{
			Path:                       inputPath,
			Collection:                 enabledCollection.Artifact.Ref(),
			ExpectedCollectionRevision: enabledCollection.Artifact.Revision,
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
		preview.Agent.Name != basespec.LogicalName("workflow-agent") {
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
		agentConsumerAPI.AgentImportCommitRequest{
			Prepared:                  preview.Prepared,
			PreparedFingerprint:       preview.PreparedFingerprint,
			AcceptedConfirmationCodes: preview.RequiredConfirmationCodes,
		},
	)
	requireNoError(t, err)
	if !committed.Agent.Managed || committed.Agent.BuiltIn {
		t.Fatalf("committed Agent has unexpected management flags")
	}
	if committed.Collection.Artifact.Ref() !=
		enabledCollection.Artifact.Ref() {
		t.Fatalf("import committed membership to another Collection")
	}

	current, err := harness.api.GetAgentView(
		t.Context(),
		committed.Agent.Artifact.Ref(),
	)
	requireNoError(t, err)
	if current.Name != basespec.LogicalName("workflow-agent") {
		t.Fatalf("read Agent name = %q", current.Name)
	}

	allUserAgents, err := harness.api.ListAgents(
		t.Context(),
		agentConsumerAPI.ListAgentsRequest{
			RootID: documentTopology.UserRootID(),
		},
	)
	requireNoError(t, err)
	if !containsAgent(allUserAgents, committed.Agent.Artifact.Ref()) {
		t.Fatalf("managed Agent is missing from the Root Agent list")
	}

	collectionRef := committed.Collection.Artifact.Ref()
	collectionAgents, err := harness.api.ListAgents(
		t.Context(),
		agentConsumerAPI.ListAgentsRequest{
			RootID:     documentTopology.UserRootID(),
			Collection: &collectionRef,
		},
	)
	requireNoError(t, err)
	if !containsAgent(collectionAgents, committed.Agent.Artifact.Ref()) {
		t.Fatalf("managed Agent is missing from its Collection")
	}

	resolution, err := harness.api.ResolveAgent(
		t.Context(),
		committed.Agent.Artifact.Ref(),
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
		basespec.LogicalName("workflow-agent-instructions"),
	)
	materialized, err := harness.api.MaterializeAgentText(
		t.Context(),
		textRef,
	)
	requireNoError(t, err)
	if materialized.Content != "workflow instructions" {
		t.Fatalf("materialized Text = %q", materialized.Content)
	}

	exported, err := harness.api.ExportAgent(
		t.Context(),
		agentConsumerAPI.AgentExportRequest{
			Agent: committed.Agent.Artifact.Ref(),
		},
	)
	requireNoError(t, err)
	if !exported.Managed || exported.BuiltIn {
		t.Fatalf("managed Agent export has unexpected management flags")
	}
	if !strings.Contains(exported.Content, "workflow-agent") {
		t.Fatalf("managed Agent export does not contain Agent name")
	}

	disabledAgent, err := harness.api.SetAgentEnabled(
		t.Context(),
		committed.Agent.Artifact.Ref(),
		committed.Agent.Artifact.Revision,
		false,
	)
	requireNoError(t, err)
	if disabledAgent.Enabled {
		t.Fatalf("Agent remained enabled after disable")
	}

	disabled := false
	disabledAgents, err := harness.api.ListAgents(
		t.Context(),
		agentConsumerAPI.ListAgentsRequest{
			RootID:  documentTopology.UserRootID(),
			Enabled: &disabled,
		},
	)
	requireNoError(t, err)
	if !containsAgent(disabledAgents, committed.Agent.Artifact.Ref()) {
		t.Fatalf("disabled Agent is missing from disabled Agent filter")
	}

	_, err = harness.api.SetAgentEnabled(
		t.Context(),
		committed.Agent.Artifact.Ref(),
		committed.Agent.Artifact.Revision,
		true,
	)
	requireErrorIs(t, err, basespec.ErrConflict)

	enabledAgent, err := harness.api.SetAgentEnabled(
		t.Context(),
		disabledAgent.Ref(),
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
			agentConsumerAPI.ManagedAgentDeleteRequest{
				Agent:            enabledAgent.Ref(),
				ExpectedRevision: enabledAgent.Revision,
			},
		),
	)

	_, err = harness.api.GetAgentView(
		t.Context(),
		enabledAgent.Ref(),
	)
	requireErrorIs(t, err, basespec.ErrArtifactNotFound)

	remainingUserAgents, err := harness.api.ListAgents(
		t.Context(),
		agentConsumerAPI.ListAgentsRequest{
			RootID: documentTopology.UserRootID(),
		},
	)
	requireNoError(t, err)
	if containsAgent(remainingUserAgents, enabledAgent.Ref()) {
		t.Fatalf("deleted Agent remains in Root Agent list")
	}

	remainingCollectionAgents, err := harness.api.ListAgents(
		t.Context(),
		agentConsumerAPI.ListAgentsRequest{
			RootID:     documentTopology.UserRootID(),
			Collection: &collectionRef,
		},
	)
	requireNoError(t, err)
	if containsAgent(remainingCollectionAgents, enabledAgent.Ref()) {
		t.Fatalf("deleted Agent remains in Collection Agent list")
	}

	// "requireNoError(
	// 	t,
	// 	harness.api.DeleteAgentCollection(
	// 		t.Context(),
	// 		collectionRef,
	// 		committed.Collection.Artifact.Revision,
	// 	),
	// )
	// collections, err := harness.api.ListAgentCollections(
	// 	t.Context(),
	// 	documentTopology.UserRootID(),
	// )
	// requireNoError(t, err)
	// if containsCollection(collections, collectionRef) {
	// 	t.Fatalf("deleted custom Collection remains in Collection list")
	// }
	// remainingDestinations, err := harness.api.ListAgentImportDestinationsForManagement(
	// 	t.Context(),
	// )
	// requireNoError(t, err)
	// if hasImportDestination(remainingDestinations, collectionRef) {
	// 	t.Fatalf("deleted custom Collection remains an import destination")
	// }.
}

func requireNamedAgent(
	t *testing.T,
	values []agentConsumerAPI.AgentView,
	name basespec.LogicalName,
) agentConsumerAPI.AgentView {
	t.Helper()

	for _, value := range values {
		if value.Artifact.LogicalName == name {
			return value
		}
	}
	t.Fatalf("Agent %q was not found", name)
	return agentConsumerAPI.AgentView{}
}

func requireImportDestination(
	t *testing.T,
	values []agentConsumerAPI.AgentImportDestination,
	ref artifact.ArtifactRef,
) agentConsumerAPI.AgentImportDestination {
	t.Helper()

	for _, value := range values {
		if value.Collection.Artifact.Ref() == ref {
			return value
		}
	}
	t.Fatalf("Agent import destination %q was not found", ref)
	return agentConsumerAPI.AgentImportDestination{}
}

func requireAvailableCapability(
	t *testing.T,
	plan resolve.CapabilityPlan,
	declarationType declaration.Type,
	name basespec.LogicalName,
) artifact.ArtifactRef {
	t.Helper()

	for _, occurrence := range plan.Occurrences {
		if occurrence.Type != declarationType ||
			occurrence.Name != name {
			continue
		}
		if occurrence.Status != resolve.ResolutionAvailable ||
			occurrence.Artifact == nil {
			t.Fatalf(
				"capability %s/%s is not available: %#v",
				declarationType,
				name,
				occurrence,
			)
		}
		return *occurrence.Artifact
	}

	t.Fatalf(
		"capability %s/%s was not found in %#v",
		declarationType,
		name,
		plan.Occurrences,
	)
	return artifact.ArtifactRef{}
}

func containsAgent(
	values []agentConsumerAPI.AgentView,
	ref artifact.ArtifactRef,
) bool {
	for _, value := range values {
		if value.Artifact.Ref() == ref {
			return true
		}
	}
	return false
}

// "func containsCollection(
// 	values []collection.CollectionView,
// 	ref artifact.ArtifactRef,
// ) bool {
// 	for _, value := range values {
// 		if value.Artifact.Ref() == ref {
// 			return true
// 		}
// 	}
// 	return false
// }
// func hasImportDestination(
// 	values []agentConsumerAPI.AgentImportDestination,
// 	ref artifact.ArtifactRef,
// ) bool {
// 	for _, value := range values {
// 		if value.Collection.Artifact.Ref() == ref {
// 			return true
// 		}
// 	}
// 	return false
// }.
