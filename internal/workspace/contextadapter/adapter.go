package contextadapter

import (
	"context"
	"fmt"
	"sort"

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
	ConventionOrder  int                    `json:"conventionOrder"`
	OriginalBytes    int                    `json:"originalBytes"`
	IncludedBytes    int                    `json:"includedBytes"`
	Truncated        bool                   `json:"truncated"`
}

type ContextLoadPlan struct {
	RootID          artifactstore.RootID       `json:"rootID"`
	CatalogRevision uint64                     `json:"catalogRevision"`
	Contributions   []ContextContribution      `json:"contributions"`
	Prompt          string                     `json:"prompt"`
	Diagnostics     []artifactstore.Diagnostic `json:"diagnostics,omitempty"`
	Decisions       []CompositionDecision      `json:"decisions"`
	PromptBytes     int                        `json:"promptBytes"`
}

type ContextDocument struct {
	RecordID         artifactstore.RecordID     `json:"recordID"`
	RecordRevision   uint64                     `json:"recordRevision"`
	DefinitionDigest artifactstore.Digest       `json:"definitionDigest"`
	SourceID         artifactstore.SourceID     `json:"sourceID"`
	Locator          artifactstore.Locator      `json:"locator"`
	Priority         int                        `json:"priority"`
	Name             string                     `json:"name"`
	Role             string                     `json:"role"`
	MediaType        string                     `json:"mediaType"`
	Enabled          bool                       `json:"enabled"`
	State            record.State               `json:"state"`
	CatalogCurrent   bool                       `json:"catalogCurrent"`
	RuntimeAllowed   bool                       `json:"runtimeAllowed"`
	Diagnostics      []artifactstore.Diagnostic `json:"diagnostics,omitempty"`
}

type ContextInspection struct {
	RootID          artifactstore.RootID       `json:"rootID"`
	CatalogRevision uint64                     `json:"catalogRevision"`
	Contributions   []ContextContribution      `json:"contributions"`
	Diagnostics     []artifactstore.Diagnostic `json:"diagnostics,omitempty"`
}

type Adapter struct {
	query             *engine.QueryService
	runtimePolicy     engine.RuntimePolicy
	compositionPolicy CompositionPolicy
}

func NewAdapter(
	query *engine.QueryService,
	runtimePolicy engine.RuntimePolicy,
	compositionPolicy CompositionPolicy,
) (*Adapter, error) {
	if query == nil || runtimePolicy == nil {
		return nil, fmt.Errorf(
			"%w: Workspace context adapter query is nil",
			engine.ErrInvalidWorkspace,
		)
	}
	compositionPolicy = compositionPolicy.Normalized()
	if err := compositionPolicy.Validate(); err != nil {
		return nil, err
	}
	return &Adapter{
		query:             query,
		runtimePolicy:     runtimePolicy,
		compositionPolicy: compositionPolicy,
	}, nil
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
			if value.Enabled && value.State == record.StateAvailable {
				recordIDs = append(recordIDs, value.RecordID)
			}
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
		Diagnostics:     artifactstore.CloneDiagnostics(loadPlan.Diagnostics),
	}
	for _, item := range loadPlan.Items {
		if err := ValidateContextDefinition(item.Definition); err != nil {
			output.Diagnostics = artifactstore.AppendDiagnostics(
				output.Diagnostics,
				contextProjectionDiagnostic(item.Record, err),
			)
			continue
		}
		decision := p.runtimePolicy.Decide(ctx, engine.RuntimePolicyRequest{
			Use:              engine.RuntimeUseContextPrompt,
			Workspace:        workspaceValue,
			Record:           item.Record,
			DefinitionDigest: item.Definition.Digest,
			SourceID:         item.Source.ID,
			TrustReference:   workspaceValue.Data.TrustReference,
		})
		if err := decision.Validate(); err != nil {
			return ContextLoadPlan{}, err
		}
		if decision.Disposition != engine.RuntimeAllowed {
			output.Diagnostics = artifactstore.AppendDiagnostics(
				output.Diagnostics,
				engine.RuntimeDecisionDiagnostic(decision, item.Record),
			)
			status := CompositionDenied
			if decision.Disposition == engine.RuntimeUnavailable {
				status = CompositionUnavailable
			}
			output.Decisions = append(output.Decisions, CompositionDecision{
				RecordID: item.Record.ID,
				Status:   status,
				Code:     decision.Code,
			})
			continue
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
				ConventionOrder: contextRuntimeOrder(
					item.Record.Occurrence.Locator,
				),
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
	output.Contributions,
		output.Prompt,
		output.Diagnostics,
		output.Decisions = applyCompositionPolicy(
		p.compositionPolicy,
		output.Contributions,
		output.Diagnostics,
		output.Decisions,
	)
	output.PromptBytes = len(output.Prompt)
	return output, nil
}

func (p *Adapter) List(
	ctx context.Context,
	rootID artifactstore.RootID,
) ([]ContextDocument, error) {
	view, err := p.query.Catalog(ctx, rootID)
	if err != nil {
		return nil, err
	}
	priorities := attachmentPriorities(view.Workspace)
	output := make([]ContextDocument, 0)
	for _, resourceValue := range view.Resources {
		if resourceValue.Definition.Kind != contextKind ||
			resourceValue.Definition.SchemaID != contextSchemaID {
			continue
		}
		value, err := projectContextDocument(resourceValue, priorities)
		if err != nil {
			value.Diagnostics = artifactstore.AppendDiagnostics(
				value.Diagnostics,
				contextProjectionDiagnostic(resourceValue.Record, err),
			)
		}
		output = append(output, value)
	}
	sort.Slice(output, func(left, right int) bool {
		if output[left].Priority != output[right].Priority {
			return output[left].Priority > output[right].Priority
		}
		leftOrder := contextRuntimeOrder(output[left].Locator)
		rightOrder := contextRuntimeOrder(output[right].Locator)
		if leftOrder != rightOrder {
			return leftOrder < rightOrder
		}
		return output[left].RecordID < output[right].RecordID
	})
	return output, nil
}

func (p *Adapter) Load(
	ctx context.Context,
	rootID artifactstore.RootID,
	recordIDs []artifactstore.RecordID,
) (ContextInspection, error) {
	view, err := p.query.Catalog(ctx, rootID)
	if err != nil {
		return ContextInspection{}, err
	}
	requested := make(map[artifactstore.RecordID]struct{}, len(recordIDs))
	for _, recordID := range recordIDs {
		if err := artifactstore.ValidateRecordID(recordID); err != nil {
			return ContextInspection{}, err
		}
		if _, duplicate := requested[recordID]; duplicate {
			return ContextInspection{}, fmt.Errorf(
				"%w: duplicate Context record %q",
				engine.ErrInvalidWorkspace,
				recordID,
			)
		}
		requested[recordID] = struct{}{}
	}
	priorities := attachmentPriorities(view.Workspace)
	output := ContextInspection{
		RootID:          rootID,
		CatalogRevision: view.Catalog.Revision,
	}
	for _, resourceValue := range view.Resources {
		if resourceValue.Definition.Kind != contextKind ||
			resourceValue.Definition.SchemaID != contextSchemaID {
			continue
		}
		if len(requested) != 0 {
			if _, selected := requested[resourceValue.Record.ID]; !selected {
				continue
			}
		}
		contribution, err := projectContext(resourceValue, priorities)
		if err != nil {
			output.Diagnostics = artifactstore.AppendDiagnostics(
				output.Diagnostics,
				contextProjectionDiagnostic(resourceValue.Record, err),
			)
			continue
		}
		output.Contributions = append(output.Contributions, contribution)
		output.Diagnostics = artifactstore.AppendDiagnostics(
			output.Diagnostics,
			resourceValue.Record.Diagnostics...,
		)
	}
	sortContextContributions(output.Contributions)
	if len(requested) != 0 &&
		len(output.Contributions) != len(requested) {
		output.Diagnostics = artifactstore.AppendDiagnostics(
			output.Diagnostics,
			artifactstore.Diagnostic{
				Severity: artifactstore.DiagnosticError,
				Code:     engine.DiagnosticCodeRecordUnresolved,
				Message:  "one or more requested Context records were not available for inspection",
			},
		)
	}
	return output, nil
}

func projectContextDocument(
	value engine.Resource,
	priorities map[artifactstore.SourceID]int,
) (ContextDocument, error) {
	runtimeAllowed, dataErr := engine.RecordRuntimeAllowed(value.Record)
	output := ContextDocument{
		RecordID:         value.Record.ID,
		RecordRevision:   value.Record.Revision,
		DefinitionDigest: value.Definition.Digest,
		SourceID:         value.Source.ID,
		Locator:          value.Record.Occurrence.Locator,
		Priority:         priorities[value.Source.ID],
		Name:             value.Record.Name,
		Enabled:          value.Record.Enabled,
		State:            value.Record.State,
		CatalogCurrent:   value.CatalogCurrent,
		RuntimeAllowed:   runtimeAllowed,
		Diagnostics: artifactstore.AppendDiagnostics(
			value.Record.Diagnostics,
			value.Diagnostics...,
		),
	}
	if dataErr != nil {
		return output, dataErr
	}
	if err := ValidateContextDefinition(value.Definition); err != nil {
		return output, err
	}
	body, err := engine.DecodeDefinitionBody[contextDefinition](
		value.Definition.Body,
	)
	if err != nil {
		return output, err
	}
	output.Name = body.Name
	output.Role = body.Role
	output.MediaType = body.MediaType
	return output, nil
}

func projectContext(
	value engine.Resource,
	priorities map[artifactstore.SourceID]int,
) (ContextContribution, error) {
	if err := ValidateContextDefinition(value.Definition); err != nil {
		return ContextContribution{}, err
	}
	body, err := engine.DecodeDefinitionBody[contextDefinition](value.Definition.Body)
	if err != nil {
		return ContextContribution{}, err
	}
	return ContextContribution{
		RecordID:         value.Record.ID,
		DefinitionDigest: value.Definition.Digest,
		SourceID:         value.Source.ID,
		Locator:          value.Record.Occurrence.Locator,
		Priority:         priorities[value.Source.ID],
		ConventionOrder:  contextRuntimeOrder(value.Record.Occurrence.Locator),
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
		if values[left].ConventionOrder != values[right].ConventionOrder {
			return values[left].ConventionOrder <
				values[right].ConventionOrder
		}
		if values[left].SourceID != values[right].SourceID {
			return values[left].SourceID < values[right].SourceID
		}
		return values[left].Locator < values[right].Locator
	})
}

func contextRuntimeOrder(locator artifactstore.Locator) int {
	if convention, found := contextConventionFor(locator); found {
		return convention.RuntimeOrder
	}
	return 10_000
}

func contextProjectionDiagnostic(
	value record.Record,
	err error,
) artifactstore.Diagnostic {
	return artifactstore.Diagnostic{
		Severity: artifactstore.DiagnosticError,
		Code:     engine.DiagnosticCodeProjectionInvalid,
		Message:  err.Error(),
		Location: &artifactstore.DiagnosticLocation{
			Locator:            value.Occurrence.Locator,
			SubresourceLocator: value.Occurrence.SubresourceLocator,
		},
	}
}
