package contextadapter

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/record"

	"github.com/flexigpt/flexigpt-app/internal/workspace/engine"
)

type ContextContribution struct {
	RecordID         artifactstore.RecordID `json:"recordID"`
	DefinitionDigest artifactstore.Digest   `json:"definitionDigest"`
	SourceID         artifactstore.SourceID `json:"sourceID"`
	Locator          artifactstore.Locator  `json:"locator"`
	Priority         int                    `json:"priority"`
	Name             string                 `json:"name"`
	Role             string                 `json:"role"`
	MediaType        string                 `json:"mediaType"`
	Content          string                 `json:"content"`
}

type ContextLoadPlan struct {
	RootID          artifactstore.RootID       `json:"rootID"`
	CatalogRevision uint64                     `json:"catalogRevision"`
	Contributions   []ContextContribution      `json:"contributions"`
	Prompt          string                     `json:"prompt"`
	Diagnostics     []artifactstore.Diagnostic `json:"diagnostics,omitempty"`
}

type Adapter struct {
	query *engine.QueryService
}

func NewAdapter(
	query *engine.QueryService,
) (*Adapter, error) {
	if query == nil {
		return nil, fmt.Errorf(
			"%w: Workspace context adapter query is nil",
			engine.ErrInvalidWorkspace,
		)
	}
	return &Adapter{query: query}, nil
}

func (p *Adapter) Compose(
	ctx context.Context,
	rootID artifactstore.RootID,
	recordIDs []artifactstore.RecordID,
) (ContextLoadPlan, error) {
	if len(recordIDs) == 0 {
		values, err := p.List(ctx, rootID)
		if err != nil {
			return ContextLoadPlan{}, err
		}
		for _, value := range values {
			recordIDs = append(recordIDs, value.RecordID)
		}
	}

	loadPlan, err := p.query.ComposeLoadPlan(ctx, rootID, recordIDs)
	if err != nil {
		return ContextLoadPlan{}, err
	}

	workspaceValue, err := p.query.GetWorkspace(ctx, rootID)
	if err != nil {
		return ContextLoadPlan{}, err
	}
	priorities := attachmentPriorities(workspaceValue)

	output := ContextLoadPlan{
		RootID:          rootID,
		CatalogRevision: loadPlan.CatalogRevision,
		Diagnostics:     loadPlan.Diagnostics,
	}
	for _, item := range loadPlan.Items {
		if item.Definition.Kind != contextKind ||
			item.Definition.SchemaID != contextSchemaID {
			return ContextLoadPlan{}, fmt.Errorf(
				"%w: record %q is not a Workspace context resource",
				engine.ErrInvalidWorkspace,
				item.Record.ID,
			)
		}
		body, err := engine.DecodeDefinitionBody[contextDefinition](
			item.Definition.Body,
		)
		if err != nil {
			return ContextLoadPlan{}, err
		}
		output.Contributions = append(
			output.Contributions,
			ContextContribution{
				RecordID:         item.Record.ID,
				DefinitionDigest: item.Definition.Digest,
				SourceID:         item.Source.ID,
				Locator:          item.Record.Occurrence.Locator,
				Priority:         priorities[item.Source.ID],
				Name:             body.Name,
				Role:             body.Role,
				MediaType:        body.MediaType,
				Content:          body.Content,
			},
		)
	}
	sortContextContributions(output.Contributions)
	output.Prompt = composeContextPrompt(output.Contributions)
	return output, nil
}

func (p *Adapter) List(
	ctx context.Context,
	rootID artifactstore.RootID,
) ([]ContextContribution, error) {
	view, err := p.query.Catalog(ctx, rootID)
	if err != nil {
		return nil, err
	}
	priorities := attachmentPriorities(view.Workspace)
	output := make([]ContextContribution, 0)
	for _, resourceValue := range view.Resources {
		if resourceValue.Definition.Kind != contextKind ||
			resourceValue.Definition.SchemaID != contextSchemaID ||
			resourceValue.Record.State != record.StateAvailable ||
			!resourceValue.Record.Enabled {
			continue
		}
		value, err := projectContext(resourceValue, priorities)
		if err != nil {
			return nil, err
		}
		output = append(output, value)
	}
	sortContextContributions(output)
	return output, nil
}

func projectContext(
	value engine.Resource,
	priorities map[artifactstore.SourceID]int,
) (ContextContribution, error) {
	body, err := engine.DecodeDefinitionBody[contextDefinition](
		value.Definition.Body,
	)
	if err != nil {
		return ContextContribution{}, err
	}
	return ContextContribution{
		RecordID:         value.Record.ID,
		DefinitionDigest: value.Definition.Digest,
		SourceID:         value.Source.ID,
		Locator:          value.Record.Occurrence.Locator,
		Priority:         priorities[value.Source.ID],
		Name:             body.Name,
		Role:             body.Role,
		MediaType:        body.MediaType,
		Content:          body.Content,
	}, nil
}

func attachmentPriorities(value engine.Workspace) map[artifactstore.SourceID]int {
	output := make(map[artifactstore.SourceID]int, len(value.Attachments))
	for _, attachment := range value.Attachments {
		if attachment.Enabled {
			output[attachment.SourceID] = attachment.Priority
		}
	}
	return output
}

func sortContextContributions(values []ContextContribution) {
	sort.Slice(values, func(left, right int) bool {
		if values[left].Priority != values[right].Priority {
			return values[left].Priority > values[right].Priority
		}
		if values[left].SourceID != values[right].SourceID {
			return values[left].SourceID < values[right].SourceID
		}
		return values[left].Locator < values[right].Locator
	})
}

func composeContextPrompt(values []ContextContribution) string {
	var output strings.Builder
	for index, value := range values {
		if index > 0 {
			output.WriteString(contextPromptSeparator)
		}
		fmt.Fprintf(
			&output,
			contextPromptStartFormat,
			value.Name,
			value.Role,
			value.Locator,
		)
		output.WriteString(strings.TrimSpace(value.Content))
		output.WriteString(contextPromptEndMarker)
	}
	return output.String()
}
