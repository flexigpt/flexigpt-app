package agent_test

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	agentAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent"
	pluginAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
)

func TestWorkflow_ManagedAgentImportRejectsTamperedAndStalePreparedPlans(
	t *testing.T,
) {
	harness, pluginValue := newManagedImportFixture(
		t,
		"stale-import-plugin",
	)

	path := writeSimpleManagedAgentImport(
		t,
		"stale-plan-agent",
	)
	preview := previewManagedAgentImport(
		t,
		harness,
		pluginValue,
		path,
	)
	if !preview.CanImport {
		t.Fatalf("initial import preview cannot import: %#v", preview.Issues)
	}

	_, err := harness.api.CommitAgentImport(
		t.Context(),
		agentAPI.AgentImportCommitRequest{
			Prepared:                  preview.Prepared + ".",
			PreparedFingerprint:       preview.PreparedFingerprint,
			AcceptedConfirmationCodes: preview.RequiredConfirmationCodes,
		},
	)
	requireErrorIs(t, err, spec.ErrInvalid)

	_, err = harness.api.CommitAgentImport(
		t.Context(),
		agentAPI.AgentImportCommitRequest{
			Prepared: preview.Prepared,
			PreparedFingerprint: cryptoutil.DigestBytes(
				[]byte("wrong prepared fingerprint"),
			),
			AcceptedConfirmationCodes: preview.RequiredConfirmationCodes,
		},
	)
	requireErrorIs(t, err, spec.ErrConflict)

	_, err = harness.api.SetAgentPluginEnabled(
		t.Context(),
		pluginValue.Artifact.Ref(),
		pluginValue.Artifact.Revision,
		false,
	)
	requireNoError(t, err)

	_, err = harness.api.CommitAgentImport(
		t.Context(),
		agentAPI.AgentImportCommitRequest{
			Prepared:                  preview.Prepared,
			PreparedFingerprint:       preview.PreparedFingerprint,
			AcceptedConfirmationCodes: preview.RequiredConfirmationCodes,
		},
	)
	requireErrorIs(t, err, spec.ErrConflict)

	agents, err := harness.api.ListAgents(
		t.Context(),
		agentAPI.ListAgentsRequest{
			RootID: topology.UserRootID(),
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
	harness, pluginValue := newManagedImportFixture(
		t,
		"identity-import-plugin",
	)

	firstPath := writeSimpleManagedAgentImport(
		t,
		"duplicate-agent",
	)
	firstPreview := previewManagedAgentImport(
		t,
		harness,
		pluginValue,
		firstPath,
	)
	if !firstPreview.CanImport {
		t.Fatalf("first import preview cannot import: %#v", firstPreview.Issues)
	}

	firstCommit, err := harness.api.CommitAgentImport(
		t.Context(),
		agentAPI.AgentImportCommitRequest{
			Prepared:                  firstPreview.Prepared,
			PreparedFingerprint:       firstPreview.PreparedFingerprint,
			AcceptedConfirmationCodes: firstPreview.RequiredConfirmationCodes,
		},
	)
	requireNoError(t, err)

	duplicatePreview := previewManagedAgentImport(
		t,
		harness,
		firstCommit.Plugin,
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
		firstCommit.Plugin,
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
	harness, pluginValue := newManagedImportFixture(
		t,
		"confirmation-import-plugin",
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
		pluginValue,
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
		agentAPI.AgentImportCommitRequest{
			Prepared:            preview.Prepared,
			PreparedFingerprint: preview.PreparedFingerprint,
		},
	)
	requireErrorIs(t, err, spec.ErrConflict)

	committed, err := harness.api.CommitAgentImport(
		t.Context(),
		agentAPI.AgentImportCommitRequest{
			Prepared:                  preview.Prepared,
			PreparedFingerprint:       preview.PreparedFingerprint,
			AcceptedConfirmationCodes: preview.RequiredConfirmationCodes,
		},
	)
	requireNoError(t, err)

	requireNoError(
		t,
		harness.api.DeleteManagedAgent(
			t.Context(),
			agentAPI.ManagedAgentDeleteRequest{
				Agent:            committed.Agent.Ref,
				ExpectedRevision: committed.Agent.Revision,
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
		agentAPI.ListAgentsRequest{
			RootID: topology.BuiltinRootID(),
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
		topology.BuiltinRootID(),
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
		agent.Ref,
		agent.Revision,
		false,
	)
	requireNoError(t, err)
	if disabled.Enabled {
		t.Fatalf("protected built-in Agent remained enabled after disable")
	}

	enabled, err := harness.api.SetAgentEnabled(
		t.Context(),
		disabled.Ref,
		disabled.Revision,
		true,
	)
	requireNoError(t, err)
	if !enabled.Enabled {
		t.Fatalf("protected built-in Agent remained disabled after enable")
	}

	err = harness.api.DeleteManagedAgent(
		t.Context(),
		agentAPI.ManagedAgentDeleteRequest{
			Agent:            enabled.Ref,
			ExpectedRevision: enabled.Revision,
		},
	)
	requireErrorIs(t, err, spec.ErrProtected)

	_, err = harness.api.CreateAgentPlugin(
		t.Context(),
		pluginAPI.CreateRequest{
			RootID:      topology.BuiltinRootID(),
			Name:        "protected-test-plugin",
			DisplayName: "Protected Test Plugin",
		},
	)
	requireErrorIs(t, err, spec.ErrProtected)
}

func newManagedImportFixture(
	t *testing.T,
	pluginName string,
) (*workflowHarness, pluginAPI.PluginView) {
	t.Helper()

	harness := newWorkflowHarness(t)
	harness.installBundledAgents(t)

	baselineEnsurer, err := agentAPI.NewBaselineEnsurer(
		harness.api,
	)
	requireNoError(t, err)

	_, err = baselineEnsurer.EnsureAgentBaselinePlugin(
		t.Context(),
		topology.UserRootID(),
	)
	requireNoError(t, err)

	value, err := harness.api.CreateAgentPlugin(
		t.Context(),
		pluginAPI.CreateRequest{
			RootID:      topology.UserRootID(),
			Name:        spec.LogicalName(pluginName),
			DisplayName: "Managed Import Guard Plugin",
		},
	)
	requireNoError(t, err)

	return harness, value
}

func previewManagedAgentImport(
	t *testing.T,
	harness *workflowHarness,
	pluginValue pluginAPI.PluginView,
	path string,
) agentAPI.AgentImportPreview {
	t.Helper()

	preview, err := harness.api.PreviewAgentImport(
		t.Context(),
		agentAPI.AgentImportPreviewRequest{
			Path:                   path,
			Plugin:                 pluginValue.Artifact.Ref(),
			ExpectedPluginRevision: pluginValue.Artifact.Revision,
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
	preview agentAPI.AgentImportPreview,
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
	values []agentAPI.AgentListItem,
	name string,
) {
	t.Helper()

	for _, value := range values {
		if string(value.Name) == name {
			t.Fatalf(
				"unexpected Agent %q remains at %q",
				name,
				value.Ref,
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
