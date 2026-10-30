package workflow

import (
	"context"
	"testing"

	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
)

type testArtifactReader struct {
	record     artifactModel.Artifact
	definition definitionModel.Definition
}

func (r testArtifactReader) Get(
	_ context.Context,
	ref artifactModel.ArtifactRef,
) (artifactModel.Artifact, error) {
	if ref != r.record.Ref() {
		return artifactModel.Artifact{}, storeSpec.ErrArtifactNotFound
	}
	return r.record.Clone(), nil
}

func (r testArtifactReader) GetDefinition(
	_ context.Context,
	ref artifactModel.ArtifactRef,
) (definitionModel.Definition, error) {
	if ref != r.record.Ref() {
		return definitionModel.Definition{}, storeSpec.ErrArtifactNotFound
	}
	return r.definition.Clone(), nil
}

type emptyCatalog struct{}

func (emptyCatalog) FindByIdentity(
	_ context.Context,
	_ rootModel.RootID,
	_ artifactModel.ArtifactKind,
	_ storeSpec.LogicalName,
	_ catalogModel.ListOptions,
) ([]catalogModel.Entry, error) {
	return nil, nil
}

func (emptyCatalog) ListBySource(
	_ context.Context,
	_ rootModel.RootID,
	_ sourceModel.SourceID,
	_ catalogModel.ListOptions,
) ([]catalogModel.Entry, error) {
	return nil, nil
}

type testDirectProvider struct {
	available bool
}

func (testDirectProvider) ProviderIdentity() string {
	return "workflow-test"
}

func (p testDirectProvider) ResolveDirectCapability(
	_ context.Context,
	request composition.DirectCapabilityRequest,
) (composition.CapabilityTarget, bool, error) {
	if !p.available {
		return composition.CapabilityTarget{}, false, nil
	}
	if request.Type != declaration.TypeAgent &&
		request.Type != declaration.TypeMCP {
		return composition.CapabilityTarget{}, false, nil
	}

	return composition.CapabilityTarget{
		Form:             composition.TargetFormDirect,
		Type:             request.Type,
		Name:             request.Name,
		Provenance:       composition.TargetProvenanceDirectCapability,
		ProviderIdentity: p.ProviderIdentity(),
		ProviderLocalID:  string(request.Type) + "/" + string(request.Name),
		Evidence:         cryptoutil.DigestBytes([]byte("workflow-plan-test")),
	}, true, nil
}

func testWorkflow(t *testing.T) workflowv1.WorkflowDocument {
	t.Helper()

	document, err := workflowv1.DecodeWorkflowJSON([]byte(`{
		"apiVersion": "v1",
		"type": "workflow",
		"name": "plan-test",
		"inputSchema": true,
		"outputSchema": true,
		"steps": [],
		"output": null,
		"targets": {
			"writer": {
				"kind": "agent",
				"ref": "writer"
			},
			"mcp": {
				"kind": "tool",
				"ref": "publish",
				"server": {
					"name": "issue-server"
				}
			},
			"external": {
				"kind": "service",
				"ref": "releases.publish"
			}
		},
		"requirements": {
			"audit": {
				"kind": "activity",
				"ref": "audit.write"
			}
		}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	return document
}

func testResolver(
	t *testing.T,
	document workflowv1.WorkflowDocument,
	available bool,
) (*composition.Resolver, artifactModel.ArtifactRef) {
	t.Helper()

	definition, err := workflowv1.DefinitionForDocument(document)
	if err != nil {
		t.Fatal(err)
	}

	ref := artifactModel.ArtifactRef{
		RootID:     rootModel.RootID("0192c4c0-0000-7000-8000-000000000002"),
		ArtifactID: artifactModel.ArtifactID("0192c4c0-0002-7000-8000-000000000001"),
	}
	record := artifactModel.Artifact{
		ID:     ref.ArtifactID,
		RootID: ref.RootID,
	}
	record.Kind = artifactModel.ArtifactKind(workflowv1.WorkflowType)
	record.LogicalName = definition.LogicalName
	record.LogicalVersion = definition.LogicalVersion
	record.DisplayName = definition.DisplayName
	record.State = artifactModel.StateAvailable
	record.Enabled = true
	record.Revision = 1
	record.Binding.SourceID = sourceModel.SourceID("0192c4c0-0001-7000-8000-000000000001")
	record.Binding.Locator = storeSpec.Locator("workflow.yaml")
	digest := definition.Digest
	record.ResolvedDefinition = &digest

	interpretations, err := coreinterpretation.NewRegistry(
		workflowv1.Interpretation(),
	)
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := composition.New(composition.ResolverOptions{
		Artifacts: testArtifactReader{
			record:     record,
			definition: definition,
		},
		Catalog:         emptyCatalog{},
		Interpretations: interpretations,
		DirectCapabilities: []composition.DirectCapabilityProvider{
			testDirectProvider{available: available},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return resolver, ref
}

func TestResolvePlan(t *testing.T) {
	for _, available := range []bool{true, false} {
		name := "unavailable"
		if available {
			name = "available"
		}
		t.Run(name, func(t *testing.T) {
			document := testWorkflow(t)
			resolver, ref := testResolver(t, document, available)

			service, err := NewService(resolver)
			if err != nil {
				t.Fatal(err)
			}
			plan, err := service.ResolvePlan(t.Context(), ref)
			if err != nil {
				t.Fatal(err)
			}
			if plan.Workflow != ref {
				t.Fatalf("got Workflow %v, expected %v", plan.Workflow, ref)
			}
			if plan.Document.Name != "plan-test" {
				t.Fatalf("got Workflow name %q", plan.Document.Name)
			}
			if plan.CompositionComplete != available ||
				plan.Capabilities.Complete != available {
				t.Fatalf(
					"composition completeness is %v/%v, expected %v",
					plan.CompositionComplete,
					plan.Capabilities.Complete,
					available,
				)
			}
			if len(plan.Capabilities.Occurrences) != 2 {
				t.Fatalf(
					"got %d composition occurrences, expected Agent and MCP server",
					len(plan.Capabilities.Occurrences),
				)
			}

			expected := []struct {
				path            string
				usesComposition bool
			}{
				{path: "targets/external", usesComposition: false},
				{path: "targets/mcp", usesComposition: true},
				{path: "targets/writer", usesComposition: true},
				{path: "requirements/audit", usesComposition: false},
			}
			if len(plan.BindingRequirements) != len(expected) {
				t.Fatalf(
					"got %d binding requirements, expected %d",
					len(plan.BindingRequirements),
					len(expected),
				)
			}
			for index, want := range expected {
				got := plan.BindingRequirements[index]
				if got.Path != want.path ||
					got.UsesComposition != want.usesComposition {
					t.Fatalf("binding %d: got %+v, expected %+v", index, got, want)
				}
			}

			// The MCP Tool operation still needs binding preparation even when
			// its server relationship resolves. Its inventory value must also
			// be independent of the returned document.
			mcp := &plan.BindingRequirements[1]
			if mcp.Target.Server == nil {
				t.Fatal("MCP Tool binding has no server reference")
			}
			mcp.Target.Server.Name = "changed"
			if plan.Document.Targets["mcp"].Server.Name != "issue-server" {
				t.Fatal("binding inventory shares mutable data with the document")
			}
		})
	}
}

func TestServiceRequiresResolver(t *testing.T) {
	if _, err := NewService(nil); err == nil {
		t.Fatal("accepted a nil resolver")
	}

	var service *Service
	ref := artifactModel.ArtifactRef{
		RootID:     rootModel.RootID("0192c4c0-0000-7000-8000-000000000002"),
		ArtifactID: artifactModel.ArtifactID("0192c4c0-0002-7000-8000-000000000001"),
	}
	if _, err := service.ResolvePlan(t.Context(), ref); err == nil {
		t.Fatal("nil service accepted a plan request")
	}
}
