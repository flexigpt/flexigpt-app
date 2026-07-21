package workspace

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/record"
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

type ContextProvider struct {
	query *QueryService
}

func NewContextProvider(
	query *QueryService,
) (*ContextProvider, error) {
	if query == nil {
		return nil, fmt.Errorf(
			"%w: Workspace context provider query is nil",
			ErrInvalidWorkspace,
		)
	}
	return &ContextProvider{query: query}, nil
}

func (p *ContextProvider) List(
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
		if resourceValue.Definition.Kind != ContextKind ||
			resourceValue.Definition.SchemaID != ContextSchemaID ||
			resourceValue.Record.State != record.StateAvailable {
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

func (p *ContextProvider) Compose(
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

	workspaceValue, err := p.query.workspaces.Get(ctx, rootID)
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
		if item.Definition.Kind != ContextKind ||
			item.Definition.SchemaID != ContextSchemaID {
			return ContextLoadPlan{}, fmt.Errorf(
				"%w: record %q is not a Workspace context resource",
				ErrInvalidWorkspace,
				item.Record.ID,
			)
		}
		body, err := decodeDefinitionBody[ContextDefinition](
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

func projectContext(
	value Resource,
	priorities map[artifactstore.SourceID]int,
) (ContextContribution, error) {
	body, err := decodeDefinitionBody[ContextDefinition](
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

func attachmentPriorities(value Workspace) map[artifactstore.SourceID]int {
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
			output.WriteString("\n\n")
		}
		fmt.Fprintf(
			&output,
			"<<<WORKSPACE_CONTEXT name=%q role=%q source=%q>>>\n",
			value.Name,
			value.Role,
			value.Locator,
		)
		output.WriteString(strings.TrimSpace(value.Content))
		output.WriteString("\n<<<END_WORKSPACE_CONTEXT>>>")
	}
	return output.String()
}
