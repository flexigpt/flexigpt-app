package providerapi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/agentv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/collectionv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/contextv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/instructionv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/loopv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/modelv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/teamv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/toolv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/workflowv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/workspacev1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
)

const CanonicalJSONDecoderID basespec.DecoderID = "workspace-artifact-json"

type CanonicalDecoder struct {
	canonicalizer providerapi.ExpectedCanonicalizer
}

func NewCanonicalDecoder() *CanonicalDecoder {
	return &CanonicalDecoder{}
}

func (*CanonicalDecoder) ID() basespec.DecoderID {
	return CanonicalJSONDecoderID
}

func (*CanonicalDecoder) Revision() string {
	return "workspace-artifact-json/v1"
}

func (*CanonicalDecoder) RequiredSchemaKeys() []schema.Key {
	return []schema.Key{
		instructionv1.InstructionSchemaKey,
		contextv1.ContextSchemaKey,
		toolv1.ToolSchemaKey,
		modelv1.ModelSchemaKey,
		collectionv1.CollectionSchemaKey,
		agentv1.AgentSchemaKey,
		teamv1.TeamSchemaKey,
		loopv1.LoopSchemaKey,
		workflowv1.WorkflowSchemaKey,
		workspacev1.WorkspaceSchemaKey,
	}
}

func (d *CanonicalDecoder) BindExpectedCanonicalizer(
	catalog providerapi.SchemaCatalog,
) error {
	if d == nil || catalog == nil {
		return fmt.Errorf(
			"%w: canonical declaration schema catalog is nil",
			basespec.ErrInvalid,
		)
	}
	d.canonicalizer = catalog
	return nil
}

func (*CanonicalDecoder) Recognize(
	_ context.Context,
	candidate providerapi.Candidate,
) providerapi.Recognition {
	var header struct {
		Type declaration.Type `json:"type"`
	}
	if err := json.Unmarshal(candidate.Content, &header); err != nil {
		return providerapi.RecognitionNone
	}
	if _, found := canonicalSchemaKey(header.Type); !found {
		return providerapi.RecognitionNone
	}
	return providerapi.RecognitionPreferred
}

func (d *CanonicalDecoder) Decode(
	ctx context.Context,
	candidate providerapi.Candidate,
) ([]providerapi.Decoded, []diagnostic.Diagnostic) {
	if d == nil || d.canonicalizer == nil {
		return nil, []diagnostic.Diagnostic{{
			Severity: diagnostic.SeverityError,
			Code:     "workspace.artifact-schema-unavailable",
			Message:  "canonical Artifact declaration schema registry is unavailable",
			Location: &diagnostic.Location{
				Locator: candidate.Locator,
			},
		}}
	}

	var header struct {
		Type declaration.Type `json:"type"`
	}
	if err := json.Unmarshal(candidate.Content, &header); err != nil {
		return nil, nil
	}
	key, found := canonicalSchemaKey(header.Type)
	if !found {
		return nil, nil
	}

	parsed, err := d.canonicalizer.CanonicalizeExpected(
		ctx,
		key,
		candidate.Content,
	)
	if err != nil {
		return nil, []diagnostic.Diagnostic{{
			Severity: diagnostic.SeverityError,
			Code:     "workspace.artifact-invalid",
			Message:  diagnostic.BoundedMessage(err.Error()),
			Location: &diagnostic.Location{
				Locator: candidate.Locator,
			},
		}}
	}

	root, err := declaration.DecodeEntryJSON(parsed.Raw)
	if err != nil {
		return nil, []diagnostic.Diagnostic{{
			Severity: diagnostic.SeverityError,
			Code:     "workspace.artifact-invalid",
			Message:  diagnostic.BoundedMessage(err.Error()),
			Location: &diagnostic.Location{
				Locator: candidate.Locator,
			},
		}}
	}
	entries, err := declaration.WalkNamedEntries(root)
	if err != nil {
		return nil, []diagnostic.Diagnostic{{
			Severity: diagnostic.SeverityError,
			Code:     "workspace.artifact-nested-invalid",
			Message:  diagnostic.BoundedMessage(err.Error()),
			Location: &diagnostic.Location{
				Locator: candidate.Locator,
			},
		}}
	}

	output := make([]providerapi.Decoded, 0, len(entries))
	for _, entry := range entries {
		value, err := DefinitionForEntry(entry.Entry)
		if err != nil {
			output = append(output, providerapi.Decoded{
				SubresourceLocator: entry.SubresourceLocator,
				Diagnostics: []diagnostic.Diagnostic{{
					Severity: diagnostic.SeverityError,
					Code:     "workspace.artifact-nested-invalid",
					Message:  diagnostic.BoundedMessage(err.Error()),
					Location: &diagnostic.Location{
						Locator:            candidate.Locator,
						SubresourceLocator: entry.SubresourceLocator,
					},
				}},
			})
			continue
		}
		output = append(output, providerapi.Decoded{
			SubresourceLocator: entry.SubresourceLocator,
			Definition:         value,
		})
	}
	return output, nil
}

func canonicalSchemaKey(
	declarationType declaration.Type,
) (schema.Key, bool) {
	switch declarationType {
	case declaration.TypeInstruction:
		return instructionv1.InstructionSchemaKey, true
	case declaration.TypeContext:
		return contextv1.ContextSchemaKey, true
	case declaration.TypeTool:
		return toolv1.ToolSchemaKey, true
	case declaration.TypeModel:
		return modelv1.ModelSchemaKey, true
	case declaration.TypeCollection:
		return collectionv1.CollectionSchemaKey, true
	case declaration.TypeAgent:
		return agentv1.AgentSchemaKey, true
	case declaration.TypeTeam:
		return teamv1.TeamSchemaKey, true
	case declaration.TypeLoop:
		return loopv1.LoopSchemaKey, true
	case declaration.TypeWorkflow:
		return workflowv1.WorkflowSchemaKey, true
	case declaration.TypeWorkspace:
		return workspacev1.WorkspaceSchemaKey, true
	default:
		return schema.Key{}, false
	}
}
