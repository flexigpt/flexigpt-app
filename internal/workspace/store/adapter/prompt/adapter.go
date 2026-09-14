package prompt

import (
	"context"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/contextv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/instructionv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	workspaceRuntime "github.com/flexigpt/flexigpt-app/internal/workspace/runtime"
	contextAdapter "github.com/flexigpt/flexigpt-app/internal/workspace/store/adapter/context"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/adapter/instruction"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/workspace/store/domain"
)

type Contribution struct {
	Artifact         artifact.ArtifactRef
	ArtifactRevision uint64
	DefinitionDigest string
	Kind             artifact.ArtifactKind
	Name             string
	MediaType        string
	Locator          basespec.Locator
	Content          string
	OriginalBytes    int
	IncludedBytes    int
	Truncated        bool
}

type Decision struct {
	Artifact      artifact.ArtifactRef
	Status        workspaceRuntime.CompositionStatus
	Code          string
	OriginalBytes int
	IncludedBytes int
}

type Plan struct {
	Workspace     workspaceDomain.WorkspaceRef
	Contributions []Contribution
	Prompt        string
	Diagnostics   []diagnostic.Diagnostic
	Decisions     []Decision
}

type Adapter struct {
	artifacts    compositionapi.ArtifactAPI
	instructions *instruction.Adapter
	contexts     *contextAdapter.Adapter
	engine       *workspaceRuntime.Engine
	policy       workspaceRuntime.CompositionPolicy
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
	instructions, err := instruction.New(artifacts, resources)
	if err != nil {
		return nil, err
	}
	contexts, err := contextAdapter.New(resources)
	if err != nil {
		return nil, err
	}
	return &Adapter{
		artifacts:    artifacts,
		instructions: instructions,
		contexts:     contexts,
		engine:       workspaceRuntime.NewEngine(),
		policy:       policy,
	}, nil
}

func (a *Adapter) Compose(
	ctx context.Context,
	workspace workspaceDomain.Workspace,
	refs []artifact.ArtifactRef,
) (Plan, error) {
	if a == nil || a.artifacts == nil || a.engine == nil {
		return Plan{}, basespec.ErrClosed
	}
	if err := workspace.Validate(); err != nil {
		return Plan{}, err
	}
	if err := ctx.Err(); err != nil {
		return Plan{}, err
	}

	selected, err := a.selection(ctx, workspace, refs)
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
	contributionsByID := make(map[string]int)

	for _, record := range selected {
		settings, err := workspaceDomain.DecodeArtifactData(record.Data)
		if err != nil {
			output.Diagnostics = diagnostic.Append(
				output.Diagnostics,
				artifactDiagnostic(
					record,
					workspaceDomain.DiagnosticCodeProjectionInvalid,
					err.Error(),
				),
			)
			output.Decisions = append(output.Decisions, Decision{
				Artifact: record.Ref(),
				Status:   workspaceRuntime.CompositionUnavailable,
				Code:     workspaceDomain.DiagnosticCodeProjectionInvalid,
			})
			continue
		}
		if settings.RuntimeDisabled {
			output.Diagnostics = diagnostic.Append(
				output.Diagnostics,
				artifactDiagnostic(
					record,
					workspaceDomain.DiagnosticCodeRuntimeDisabled,
					"runtime use is disabled for this Workspace Artifact",
				),
			)
			output.Decisions = append(output.Decisions, Decision{
				Artifact: record.Ref(),
				Status:   workspaceRuntime.CompositionDenied,
				Code:     workspaceDomain.DiagnosticCodeRuntimeDisabled,
			})
			continue
		}

		contribution, err := a.resolveContribution(ctx, record)
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
		contributionsByID[id] = len(output.Contributions)
		output.Contributions = append(output.Contributions, contribution)
		runtimeValues = append(runtimeValues, workspaceRuntime.ContextContribution{
			ID:      id,
			Kind:    string(contribution.Kind),
			Name:    contribution.Name,
			Locator: string(contribution.Locator),
			Content: contribution.Content,
		})
	}

	result, err := a.engine.Compose(a.policy, runtimeValues)
	if err != nil {
		return Plan{}, err
	}
	for _, value := range result.Contributions {
		index, found := contributionsByID[value.ID]
		if !found {
			return Plan{}, fmt.Errorf(
				"%w: prompt engine returned an unknown contribution",
				workspaceDomain.ErrInvalidWorkspace,
			)
		}
		output.Contributions[index].Content = value.Content
		output.Contributions[index].OriginalBytes = value.OriginalBytes
		output.Contributions[index].IncludedBytes = value.IncludedBytes
		output.Contributions[index].Truncated = value.Truncated
	}
	for _, value := range result.Decisions {
		index, found := contributionsByID[value.ID]
		if !found {
			return Plan{}, fmt.Errorf(
				"%w: prompt engine returned an unknown decision",
				workspaceDomain.ErrInvalidWorkspace,
			)
		}
		output.Decisions = append(output.Decisions, Decision{
			Artifact:      output.Contributions[index].Artifact,
			Status:        value.Status,
			Code:          value.Code,
			OriginalBytes: value.OriginalBytes,
			IncludedBytes: value.IncludedBytes,
		})
	}
	for _, value := range result.Diagnostics {
		index, found := contributionsByID[value.ID]
		if !found {
			return Plan{}, fmt.Errorf(
				"%w: prompt engine returned an unknown diagnostic",
				workspaceDomain.ErrInvalidWorkspace,
			)
		}
		output.Diagnostics = diagnostic.Append(
			output.Diagnostics,
			artifactDiagnostic(
				selectedArtifact(output.Contributions[index].Artifact, selected),
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
		values, err := a.artifacts.ListByRoot(
			ctx,
			workspace.Artifact.RootID,
		)
		if err != nil {
			return nil, err
		}
		output := make([]artifact.Artifact, 0)
		for _, value := range values {
			if value.Kind != artifact.ArtifactKind(instructionv1.InstructionType) &&
				value.Kind != artifact.ArtifactKind(contextv1.ContextType) {
				continue
			}
			if !value.Enabled || value.State != artifact.StateAvailable {
				continue
			}
			output = append(output, value)
		}
		sort.Slice(output, func(left, right int) bool {
			leftInstruction := output[left].Kind ==
				artifact.ArtifactKind(instructionv1.InstructionType)
			rightInstruction := output[right].Kind ==
				artifact.ArtifactKind(instructionv1.InstructionType)
			if leftInstruction != rightInstruction {
				return leftInstruction
			}
			if output[left].Binding.SourceID != output[right].Binding.SourceID {
				return output[left].Binding.SourceID <
					output[right].Binding.SourceID
			}
			if output[left].Binding.Locator != output[right].Binding.Locator {
				return output[left].Binding.Locator <
					output[right].Binding.Locator
			}
			return output[left].ID < output[right].ID
		})
		return output, nil
	}

	seen := make(map[artifact.ArtifactID]struct{}, len(refs))
	output := make([]artifact.Artifact, 0, len(refs))
	for _, ref := range refs {
		if err := ref.Validate(); err != nil {
			return nil, err
		}
		if ref.RootID != workspace.Artifact.RootID {
			return nil, fmt.Errorf(
				"%w: selected Artifact belongs to another Root",
				workspaceDomain.ErrReferenceUnresolved,
			)
		}
		if _, duplicate := seen[ref.ArtifactID]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate selected Workspace Artifact",
				workspaceDomain.ErrInvalidWorkspace,
			)
		}
		seen[ref.ArtifactID] = struct{}{}
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
) (Contribution, error) {
	switch record.Kind {
	case artifact.ArtifactKind(instructionv1.InstructionType):
		value, err := a.instructions.Resolve(ctx, record.Ref())
		if err != nil {
			return Contribution{}, err
		}
		return Contribution{
			Artifact:         value.Artifact,
			ArtifactRevision: value.ArtifactRevision,
			DefinitionDigest: value.DefinitionDigest,
			Kind:             record.Kind,
			Name:             value.Name,
			MediaType:        value.MediaType,
			Locator:          value.Locator,
			Content:          value.Content,
		}, nil

	case artifact.ArtifactKind(contextv1.ContextType):
		value, err := a.contexts.Resolve(ctx, record.Ref())
		if err != nil {
			return Contribution{}, err
		}
		return Contribution{
			Artifact:         value.Artifact,
			ArtifactRevision: value.ArtifactRevision,
			DefinitionDigest: value.DefinitionDigest,
			Kind:             record.Kind,
			Name:             value.Name,
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

func contributionID(ref artifact.ArtifactRef) string {
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
