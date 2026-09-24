package consumerapi_test

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	agentConsumerAPI "github.com/flexigpt/flexigpt-app/internal/agent/store/consumerapi"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/collection"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

func TestWorkflow_ManagedAgentImportRejectsTamperedAndStalePreparedPlans(
	t *testing.T,
) {
	harness, collectionValue := newManagedImportFixture(
		t,
		"stale-import-collection",
	)

	path := writeSimpleManagedAgentImport(
		t,
		"stale-plan-agent",
	)
	preview := previewManagedAgentImport(
		t,
		harness,
		collectionValue,
		path,
	)
	if !preview.CanImport {
		t.Fatalf("initial import preview cannot import: %#v", preview.Issues)
	}

	_, err := harness.api.CommitAgentImport(
		t.Context(),
		agentConsumerAPI.AgentImportCommitRequest{
			Prepared:                  preview.Prepared + ".",
			PreparedFingerprint:       preview.PreparedFingerprint,
			AcceptedConfirmationCodes: preview.RequiredConfirmationCodes,
		},
	)
	requireErrorIs(t, err, basespec.ErrInvalid)

	_, err = harness.api.CommitAgentImport(
		t.Context(),
		agentConsumerAPI.AgentImportCommitRequest{
			Prepared: preview.Prepared,
			PreparedFingerprint: cryptoutil.DigestBytes(
				[]byte("wrong prepared fingerprint"),
			),
			AcceptedConfirmationCodes: preview.RequiredConfirmationCodes,
		},
	)
	requireErrorIs(t, err, basespec.ErrConflict)

	_, err = harness.api.SetAgentCollectionEnabled(
		t.Context(),
		collectionValue.Artifact.Ref(),
		collectionValue.Artifact.Revision,
		false,
	)
	requireNoError(t, err)

	_, err = harness.api.CommitAgentImport(
		t.Context(),
		agentConsumerAPI.AgentImportCommitRequest{
			Prepared:                  preview.Prepared,
			PreparedFingerprint:       preview.PreparedFingerprint,
			AcceptedConfirmationCodes: preview.RequiredConfirmationCodes,
		},
	)
	requireErrorIs(t, err, basespec.ErrConflict)

	agents, err := harness.api.ListAgents(
		t.Context(),
		agentConsumerAPI.ListAgentsRequest{
			RootID: documentTopology.UserRootID(),
		},
	)
	requireNoError(t, err)
	assertAgentNamedAbsent(
		t,
		agents,
		"stale-plan-agent",
	)
}

func TestWorkflow_ManagedAgentImportSurfacesIdentityAndBuiltinConflicts(
	t *testing.T,
) {
	harness, collectionValue := newManagedImportFixture(
		t,
		"identity-import-collection",
	)

	firstPath := writeSimpleManagedAgentImport(
		t,
		"duplicate-agent",
	)
	firstPreview := previewManagedAgentImport(
		t,
		harness,
		collectionValue,
		firstPath,
	)
	if !firstPreview.CanImport {
		t.Fatalf("first import preview cannot import: %#v", firstPreview.Issues)
	}

	firstCommit, err := harness.api.CommitAgentImport(
		t.Context(),
		agentConsumerAPI.AgentImportCommitRequest{
			Prepared:                  firstPreview.Prepared,
			PreparedFingerprint:       firstPreview.PreparedFingerprint,
			AcceptedConfirmationCodes: firstPreview.RequiredConfirmationCodes,
		},
	)
	requireNoError(t, err)

	duplicatePreview := previewManagedAgentImport(
		t,
		harness,
		firstCommit.Collection,
		firstPath,
	)
	if duplicatePreview.CanImport {
		t.Fatalf(
			"duplicate managed Agent import unexpectedly can import: %#v",
			duplicatePreview,
		)
	}
	requireImportIssue(
		t,
		duplicatePreview,
		"agent.import.identity-conflict",
	)

	builtinNamePath := writeSimpleManagedAgentImport(
		t,
		"local-dev-workspace",
	)
	builtinNamePreview := previewManagedAgentImport(
		t,
		harness,
		firstCommit.Collection,
		builtinNamePath,
	)
	if builtinNamePreview.CanImport {
		t.Fatalf(
			"reserved built-in Agent name unexpectedly can import: %#v",
			builtinNamePreview,
		)
	}
	requireImportIssue(
		t,
		builtinNamePreview,
		"agent.import.builtin-name-conflict",
	)
}

func TestWorkflow_ManagedAgentImportRequiresDeclaredConfirmation(
	t *testing.T,
) {
	harness, collectionValue := newManagedImportFixture(
		t,
		"confirmation-import-collection",
	)

	path := writeManagedAgentImport(
		t,
		"confirmed-tool-agent",
		`  - type: tool
    name: readfile
    overrides:
      autoExecute: true`,
	)
	preview := previewManagedAgentImport(
		t,
		harness,
		collectionValue,
		path,
	)
	if !preview.CanImport {
		t.Fatalf(
			"auto-execute Tool import preview cannot import: %#v",
			preview.Issues,
		)
	}
	if !preview.RequiresConfirmation {
		t.Fatalf("auto-execute Tool import did not require confirmation")
	}
	if !containsString(
		preview.RequiredConfirmationCodes,
		"agent.import.tool-auto-execute",
	) {
		t.Fatalf(
			"confirmation codes = %#v, want tool auto-execute code",
			preview.RequiredConfirmationCodes,
		)
	}

	_, err := harness.api.CommitAgentImport(
		t.Context(),
		agentConsumerAPI.AgentImportCommitRequest{
			Prepared:            preview.Prepared,
			PreparedFingerprint: preview.PreparedFingerprint,
		},
	)
	requireErrorIs(t, err, basespec.ErrConflict)

	committed, err := harness.api.CommitAgentImport(
		t.Context(),
		agentConsumerAPI.AgentImportCommitRequest{
			Prepared:                  preview.Prepared,
			PreparedFingerprint:       preview.PreparedFingerprint,
			AcceptedConfirmationCodes: preview.RequiredConfirmationCodes,
		},
	)
	requireNoError(t, err)
	if !committed.Agent.Managed {
		t.Fatalf("confirmed import did not create a managed Agent")
	}

	requireNoError(
		t,
		harness.api.DeleteManagedAgent(
			t.Context(),
			agentConsumerAPI.ManagedAgentDeleteRequest{
				Agent:            committed.Agent.Artifact.Ref(),
				ExpectedRevision: committed.Agent.Artifact.Revision,
			},
		),
	)
}

func TestWorkflow_ProtectedBuiltinAgentBoundaries(
	t *testing.T,
) {
	harness := newWorkflowHarness(t)
	harness.installBundledAgents(t)

	builtinAgents, err := harness.api.ListAgents(
		t.Context(),
		agentConsumerAPI.ListAgentsRequest{
			RootID: documentTopology.BuiltinRootID(),
		},
	)
	requireNoError(t, err)

	agent := requireNamedAgent(
		t,
		builtinAgents,
		"local-dev-workspace",
	)

	destinations, err := harness.api.ListAgentImportDestinations(
		t.Context(),
		documentTopology.BuiltinRootID(),
	)
	requireNoError(t, err)
	if len(destinations) != 0 {
		t.Fatalf(
			"protected built-in Root import destinations = %#v, want empty",
			destinations,
		)
	}

	disabled, err := harness.api.SetAgentEnabled(
		t.Context(),
		agent.Artifact.Ref(),
		agent.Artifact.Revision,
		false,
	)
	requireNoError(t, err)
	if disabled.Enabled {
		t.Fatalf("protected built-in Agent remained enabled after disable")
	}

	enabled, err := harness.api.SetAgentEnabled(
		t.Context(),
		disabled.Ref(),
		disabled.Revision,
		true,
	)
	requireNoError(t, err)
	if !enabled.Enabled {
		t.Fatalf("protected built-in Agent remained disabled after enable")
	}

	err = harness.api.DeleteManagedAgent(
		t.Context(),
		agentConsumerAPI.ManagedAgentDeleteRequest{
			Agent:            enabled.Ref(),
			ExpectedRevision: enabled.Revision,
		},
	)
	requireErrorIs(t, err, basespec.ErrProtected)

	_, err = harness.api.CreateAgentCollection(
		t.Context(),
		collection.CreateRequest{
			RootID:      documentTopology.BuiltinRootID(),
			Name:        "protected-test-collection",
			DisplayName: "Protected Test Collection",
		},
	)
	requireErrorIs(t, err, basespec.ErrProtected)
}

func newManagedImportFixture(
	t *testing.T,
	collectionName string,
) (*workflowHarness, collection.CollectionView) {
	t.Helper()

	harness := newWorkflowHarness(t)
	harness.installBundledAgents(t)

	baselineEnsurer, err := agentConsumerAPI.NewBaselineEnsurer(
		harness.api,
	)
	requireNoError(t, err)

	_, err = baselineEnsurer.EnsureAgentBaselineCollection(
		t.Context(),
		documentTopology.UserRootID(),
	)
	requireNoError(t, err)

	value, err := harness.api.CreateAgentCollection(
		t.Context(),
		collection.CreateRequest{
			RootID:      documentTopology.UserRootID(),
			Name:        basespec.LogicalName(collectionName),
			DisplayName: "Managed Import Guard Collection",
		},
	)
	requireNoError(t, err)

	return harness, value
}

func previewManagedAgentImport(
	t *testing.T,
	harness *workflowHarness,
	collectionValue collection.CollectionView,
	path string,
) agentConsumerAPI.AgentImportPreview {
	t.Helper()

	preview, err := harness.api.PreviewAgentImport(
		t.Context(),
		agentConsumerAPI.AgentImportPreviewRequest{
			Path:                       path,
			Collection:                 collectionValue.Artifact.Ref(),
			ExpectedCollectionRevision: collectionValue.Artifact.Revision,
		},
	)
	requireNoError(t, err)
	return preview
}

func writeSimpleManagedAgentImport(
	t *testing.T,
	name string,
) string {
	t.Helper()

	return writeManagedAgentImport(
		t,
		name,
		fmt.Sprintf(
			`  - type: text
    name: %q
    insert: instructions
    parameters:
      content: duplicate-safe workflow test instructions`,
			name+"-instructions",
		),
	)
}

func writeManagedAgentImport(
	t *testing.T,
	name string,
	members string,
) string {
	t.Helper()

	path := filepath.Join(
		t.TempDir(),
		name+".yaml",
	)
	content := fmt.Sprintf(
		"type: agent\nname: %q\ndisplayName: %q\nmembers:\n%s\n",
		name,
		"Managed "+name,
		members,
	)
	requireNoError(
		t,
		os.WriteFile(
			path,
			[]byte(content),
			0o600,
		),
	)
	return path
}

func requireImportIssue(
	t *testing.T,
	preview agentConsumerAPI.AgentImportPreview,
	code string,
) {
	t.Helper()

	for _, issue := range preview.Issues {
		if issue.Code == code {
			return
		}
	}
	t.Fatalf(
		"import issues %#v do not contain code %q",
		preview.Issues,
		code,
	)
}

func assertAgentNamedAbsent(
	t *testing.T,
	values []agentConsumerAPI.AgentView,
	name string,
) {
	t.Helper()

	for _, value := range values {
		if string(value.Artifact.LogicalName) == name {
			t.Fatalf(
				"unexpected Agent %q remains at %q",
				name,
				value.Artifact.Ref(),
			)
		}
	}
}

func containsString(
	values []string,
	expected string,
) bool {
	return slices.Contains(values, expected)
}
