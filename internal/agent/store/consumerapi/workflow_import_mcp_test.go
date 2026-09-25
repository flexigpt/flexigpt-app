package consumerapi_test

import (
	"testing"

	agentConsumerAPI "github.com/flexigpt/flexigpt-app/internal/agent/store/consumerapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

func TestWorkflow_ManagedAgentImportInlineMCPSetupAndConfirmation(
	t *testing.T,
) {
	harness, collectionValue := newManagedImportFixture(
		t,
		"inline-mcp-collection",
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
		collectionValue,
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
		agentConsumerAPI.AgentExportRequest{
			Agent: committed.Agent.Artifact.Ref(),
		},
	)
	requireNoError(t, err)

	exportedSetup := requireMCPSetupDescriptor(
		t,
		exported.MCPSetupDescriptors,
		"local-stdio-mcp",
	)
	if exportedSetup.Artifact == nil {
		t.Fatal("exported MCP setup has no contained MCP Artifact reference")
	}
	if exportedSetup.Transport != "stdio" ||
		exportedSetup.Command != "workflow-mcp" {
		t.Fatalf("exported MCP setup = %#v", exportedSetup)
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

func TestWorkflow_ManagedAgentImportReportsExpectedSourceDigestMismatch(
	t *testing.T,
) {
	harness, collectionValue := newManagedImportFixture(
		t,
		"source-digest-collection",
	)

	path := writeSimpleManagedAgentImport(
		t,
		"source-digest-agent",
	)

	preview, err := harness.api.PreviewAgentImport(
		t.Context(),
		agentConsumerAPI.AgentImportPreviewRequest{
			Path:                       path,
			Collection:                 collectionValue.Artifact.Ref(),
			ExpectedCollectionRevision: collectionValue.Artifact.Revision,
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
	values []agentConsumerAPI.AgentMCPSetupDescriptor,
	name string,
) agentConsumerAPI.AgentMCPSetupDescriptor {
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
	return agentConsumerAPI.AgentMCPSetupDescriptor{}
}
