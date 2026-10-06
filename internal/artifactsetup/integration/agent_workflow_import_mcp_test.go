package integration

import (
	"strings"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	agentAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent"
)

func TestWorkflow_ManagedAgentImportInlineMCPSetupAndConfirmation(
	t *testing.T,
) {
	harness, pluginValue := newManagedImportFixture(
		t,
		"inline-mcp-plugin",
	)

	path := writeManagedAgentImport(
		t,
		"inline-stdio-mcp-agent",
		`  - type: mcp
    name: local-stdio-mcp
    parameters:
      transport: stdio
      command: workflow-mcp
      install:
        inputs:
          apiToken:
            kind: secret
            label: API token
            required: true`,
	)

	preview := previewManagedAgentImport(
		t,
		harness,
		pluginValue,
		path,
	)
	if !preview.CanImport {
		t.Fatalf(
			"inline MCP import preview cannot import: %#v",
			preview.Issues,
		)
	}
	if !preview.RequiresConfirmation {
		t.Fatal("inline stdio MCP import did not require confirmation")
	}
	if !containsString(
		preview.RequiredConfirmationCodes,
		"agent.import.mcp-stdio",
	) {
		t.Fatalf(
			"confirmation codes = %#v, want inline MCP stdio code",
			preview.RequiredConfirmationCodes,
		)
	}

	previewSetup := requireMCPSetupDescriptor(
		t,
		preview.MCPSetupDescriptors,
		"local-stdio-mcp",
	)
	if previewSetup.Artifact != nil {
		t.Fatalf("preview MCP setup unexpectedly contains an Artifact ref")
	}
	if previewSetup.Transport != "stdio" ||
		previewSetup.Command != "workflow-mcp" {
		t.Fatalf("preview MCP setup = %#v", previewSetup)
	}
	if len(previewSetup.Inputs) != 1 ||
		previewSetup.Inputs[0].Name != "apiToken" ||
		previewSetup.Inputs[0].Kind != "secret" ||
		!previewSetup.Inputs[0].Required {
		t.Fatalf("preview MCP setup inputs = %#v", previewSetup.Inputs)
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

	committedSetup := requireMCPSetupDescriptor(
		t,
		committed.MCPSetupDescriptors,
		"local-stdio-mcp",
	)
	if committedSetup.Artifact == nil {
		t.Fatal("committed MCP setup has no contained MCP Artifact reference")
	}

	exported, err := harness.api.ExportAgent(
		t.Context(),
		agentAPI.AgentExportRequest{
			Agent: committed.Agent.Ref,
		},
	)
	requireNoError(t, err)

	// Export is intentionally declaration-only. Runtime setup descriptors are
	// produced by import preview and commit, not by a read-only YAML export.
	if !strings.Contains(exported.Content, "local-stdio-mcp") ||
		!strings.Contains(exported.Content, "workflow-mcp") {
		t.Fatalf(
			"exported Agent YAML does not retain inline MCP declaration: %q",
			exported.Content,
		)
	}

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

func TestWorkflow_ManagedAgentImportReportsExpectedSourceDigestMismatch(
	t *testing.T,
) {
	harness, pluginValue := newManagedImportFixture(
		t,
		"source-digest-plugin",
	)

	path := writeSimpleManagedAgentImport(
		t,
		"source-digest-agent",
	)

	preview, err := harness.api.PreviewAgentImport(
		t.Context(),
		agentAPI.AgentImportPreviewRequest{
			Path:                   path,
			Plugin:                 pluginValue.Artifact.Ref(),
			ExpectedPluginRevision: pluginValue.Artifact.Revision,
			ExpectedSourceDigest: cryptoutil.DigestBytes(
				[]byte("different source bytes"),
			),
		},
	)
	requireNoError(t, err)

	if preview.CanImport {
		t.Fatalf(
			"source digest mismatch preview unexpectedly can import: %#v",
			preview,
		)
	}
	if preview.Prepared != "" || preview.PreparedFingerprint != "" {
		t.Fatal("source digest mismatch preview unexpectedly has a prepared plan")
	}
	requireImportIssue(
		t,
		preview,
		"agent.import.source-digest-mismatch",
	)
}

func requireMCPSetupDescriptor(
	t *testing.T,
	values []agentAPI.AgentMCPSetupDescriptor,
	name string,
) agentAPI.AgentMCPSetupDescriptor {
	t.Helper()

	for _, value := range values {
		if string(value.Name) == name {
			return value
		}
	}
	t.Fatalf(
		"MCP setup descriptor %q was not found in %#v",
		name,
		values,
	)
	return agentAPI.AgentMCPSetupDescriptor{}
}
