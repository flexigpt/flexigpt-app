package prompt

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/textv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/materializetext"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	workspaceRuntime "github.com/flexigpt/flexigpt-app/internal/workspace/runtime"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/workspace/store/domain"
)

type Contribution struct {
	Artifact         artifact.ArtifactRef     `json:"-"`
	ArtifactRevision uint64                   `json:"-"`
	DefinitionDigest cryptoutil.Digest        `json:"-"`
	Kind             artifact.ArtifactKind    `json:"-"`
	Name             string                   `json:"-"`
	Insert           declaration.InsertTarget `json:"-"`
	MediaType        string                   `json:"-"`
	Locator          basespec.Locator         `json:"-"`
	Content          string                   `json:"-"`
	OriginalBytes    int                      `json:"-"`
	IncludedBytes    int                      `json:"-"`
	Truncated        bool                     `json:"-"`
}

type Decision struct {
	Artifact      artifact.ArtifactRef               `json:"-"`
	Status        workspaceRuntime.CompositionStatus `json:"-"`
	Code          string                             `json:"-"`
	OriginalBytes int                                `json:"-"`
	IncludedBytes int                                `json:"-"`
}

type Plan struct {
	Workspace     artifact.ArtifactRef    `json:"-"`
	Contributions []Contribution          `json:"-"`
	Prompt        string                  `json:"-"`
	Diagnostics   []diagnostic.Diagnostic `json:"-"`
	Decisions     []Decision              `json:"-"`
}

type Adapter struct {
	artifacts compositionapi.ArtifactAPI
	text      *materializetext.Adapter
	engine    *workspaceRuntime.Engine
	policy    workspaceRuntime.CompositionPolicy
}

func New(
	artifacts compositionapi.ArtifactAPI,
	resources compositionapi.ResourceAPI,
	policy workspaceRuntime.CompositionPolicy,
) (*Adapter, error) {
	if artifacts == nil || resources == nil {
		return nil, fmt.Errorf(
			"%w: Workspace prompt adapter dependencies are incomplete",
			workspaceDomain.ErrInvalidWorkspace,
		)
	}
	policy = policy.Normalized()
	if err := policy.Validate(); err != nil {
		return nil, err
	}
	t, err := materializetext.NewAdapter(resources)
	if err != nil {
		return nil, err
	}
	return &Adapter{
		artifacts: artifacts,
		text:      t,
		engine:    workspaceRuntime.NewEngine(),
		policy:    policy,
	}, nil
}

// ComposeSelected composes exactly refs in order. The Workspace consumer API
// must authorize refs against the resolved capability plan before calling it.
// Protected built-in ArtifactRefs may therefore belong to another Root. An
// empty ref list means an intentionally empty selection.
func (a *Adapter) ComposeSelected(
	ctx context.Context,
	workspace workspaceDomain.Workspace,
	refs []artifact.ArtifactRef,
) (Plan, error) {
	return a.compose(ctx, workspace, refs)
}

func (a *Adapter) compose(
	ctx context.Context,
	workspace workspaceDomain.Workspace,
	refs []artifact.ArtifactRef,
) (Plan, error) {
	if a == nil || a.artifacts == nil || a.engine == nil {
		return Plan{}, basespec.ErrClosed
	}
	if err := ctx.Err(); err != nil {
		return Plan{}, err
	}

	selected, err := a.selection(
		ctx,
		workspace,
		refs,
	)
	if err != nil {
		return Plan{}, err
	}
	output := Plan{
		Workspace:     workspace.Ref(),
		Contributions: make([]Contribution, 0, len(selected)),
		Diagnostics:   make([]diagnostic.Diagnostic, 0),
		Decisions:     make([]Decision, 0, len(selected)),
	}

	runtimeValues := make(
		[]workspaceRuntime.ContextContribution,
		0,
		len(selected),
	)
	contributionsByID := make(
		map[string]Contribution,
		len(selected),
	)

	for _, record := range selected {
		if !record.Enabled {
			output.Diagnostics = diagnostic.Append(
				output.Diagnostics,
				artifactDiagnostic(
					record,
					workspaceDomain.DiagnosticCodeArtifactUnavailable,
					"Artifact is disabled",
				),
			)
			output.Decisions = append(output.Decisions, Decision{
				Artifact: record.Ref(),
				Status:   workspaceRuntime.CompositionDenied,
				Code:     workspaceDomain.DiagnosticCodeArtifactUnavailable,
			})
			continue
		}

		contribution, err := a.resolveContribution(ctx, record, workspace)
		if err != nil {
			output.Diagnostics = diagnostic.Append(
				output.Diagnostics,
				artifactDiagnostic(
					record,
					workspaceDomain.DiagnosticCodeArtifactUnavailable,
					err.Error(),
				),
			)
			output.Decisions = append(output.Decisions, Decision{
				Artifact: record.Ref(),
				Status:   workspaceRuntime.CompositionUnavailable,
				Code:     workspaceDomain.DiagnosticCodeArtifactUnavailable,
			})
			continue
		}

		id := contributionID(contribution.Artifact)
		contributionsByID[id] = contribution
		runtimeValues = append(runtimeValues, workspaceRuntime.ContextContribution{
			ID:      id,
			Kind:    string(contribution.Insert),
			Name:    contribution.Name,
			Locator: string(contribution.Locator),
			Content: contribution.Content,
		})
	}

	result, err := a.engine.Compose(a.policy, runtimeValues)
	if err != nil {
		return Plan{}, err
	}
	output.Contributions = make(
		[]Contribution,
		0,
		len(result.Contributions),
	)
	for _, value := range result.Contributions {
		contribution, found := contributionsByID[value.ID]
		if !found {
			return Plan{}, fmt.Errorf(
				"%w: prompt engine returned an unknown contribution",
				workspaceDomain.ErrInvalidWorkspace,
			)
		}
		contribution.Content = value.Content
		contribution.OriginalBytes = value.OriginalBytes
		contribution.IncludedBytes = value.IncludedBytes
		contribution.Truncated = value.Truncated
		output.Contributions = append(output.Contributions, contribution)
	}
	for _, value := range result.Decisions {
		contribution, found := contributionsByID[value.ID]
		if !found {
			return Plan{}, fmt.Errorf(
				"%w: prompt engine returned an unknown decision",
				workspaceDomain.ErrInvalidWorkspace,
			)
		}
		output.Decisions = append(output.Decisions, Decision{
			Artifact:      contribution.Artifact,
			Status:        value.Status,
			Code:          value.Code,
			OriginalBytes: value.OriginalBytes,
			IncludedBytes: value.IncludedBytes,
		})
	}
	for _, value := range result.Diagnostics {
		contribution, found := contributionsByID[value.ID]
		if !found {
			return Plan{}, fmt.Errorf(
				"%w: prompt engine returned an unknown diagnostic",
				workspaceDomain.ErrInvalidWorkspace,
			)
		}
		output.Diagnostics = diagnostic.Append(
			output.Diagnostics,
			artifactDiagnostic(
				selectedArtifact(contribution.Artifact, selected),
				value.Code,
				value.Message,
			),
		)
	}
	output.Prompt = result.Prompt
	return output, nil
}

func (a *Adapter) selection(
	ctx context.Context,
	workspace workspaceDomain.Workspace,
	refs []artifact.ArtifactRef,
) ([]artifact.Artifact, error) {
	if len(refs) == 0 {
		return []artifact.Artifact{}, nil
	}

	output := make([]artifact.Artifact, 0, len(refs))
	for _, ref := range refs {
		value, err := a.artifacts.Get(ctx, ref)
		if err != nil {
			return nil, err
		}
		output = append(output, value)
	}
	return output, nil
}

func (a *Adapter) resolveContribution(
	ctx context.Context,
	record artifact.Artifact,
	workspace workspaceDomain.Workspace,
) (Contribution, error) {
	switch record.Kind {
	case artifact.ArtifactKind(textv1.TextType):
		var value materializetext.Document
		var err error
		if record.RootID == workspace.Artifact.RootID &&
			workspace.CompositionSourceID != "" &&
			record.Binding.SourceID != workspace.CompositionSourceID {
			value, err = a.text.ResolveWithContentSource(
				ctx,
				record.Ref(),
				workspace.Artifact.RootID,
				workspace.CompositionSourceID,
			)
		} else {
			value, err = a.text.Resolve(ctx, record.Ref())
		}
		if err != nil {
			return Contribution{}, err
		}
		return Contribution{
			Artifact:         value.Artifact,
			ArtifactRevision: value.ArtifactRevision,
			DefinitionDigest: value.DefinitionDigest,
			Kind:             record.Kind,
			Name:             value.Name,
			Insert:           value.Insert,
			MediaType:        value.MediaType,
			Locator:          value.Locator,
			Content:          value.Content,
		}, nil

	default:
		return Contribution{}, fmt.Errorf(
			"%w: Artifact kind %q cannot contribute to Workspace prompt",
			basespec.ErrUnsupported,
			record.Kind,
		)
	}
}

func selectedArtifact(
	ref artifact.ArtifactRef,
	values []artifact.Artifact,
) artifact.Artifact {
	for _, value := range values {
		if value.Ref() == ref {
			return value
		}
	}
	return artifact.Artifact{
		ID:     ref.ArtifactID,
		RootID: ref.RootID,
	}
}

func contributionID(
	ref artifact.ArtifactRef,
) string {
	return string(ref.RootID) + "\x00" + string(ref.ArtifactID)
}

func artifactDiagnostic(
	value artifact.Artifact,
	code string,
	message string,
) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{
		Severity: diagnostic.SeverityWarning,
		Code:     code,
		Message:  diagnostic.BoundedMessage(message),
		Location: &diagnostic.Location{
			Locator:            value.Binding.Locator,
			SubresourceLocator: value.Binding.SubresourceLocator,
		},
	}
}
