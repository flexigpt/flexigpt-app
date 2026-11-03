package sessionv3runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	toolAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool"
	toolv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool/contract/v1"
	inferenceSpec "github.com/flexigpt/inference-go/spec"
)

func (b *Binder) goToolChoices(ctx context.Context, req sessionv3.WorkDispatchRequest) ([]preparedChoice, error) {
	out := []preparedChoice{}
	if b.deps.Tools == nil {
		return out, nil
	}
	for _, sel := range req.Session.Defaults.Tools {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		ref := artifactModel.ArtifactRef{
			RootID:     rootModel.RootID(sel.RootID),
			ArtifactID: artifactModel.ArtifactID(sel.ArtifactID),
		}
		if err := ref.Validate(); err != nil {
			return nil, fmt.Errorf("selected Tool reference: %w", err)
		}
		target, err := b.deps.Tools.MapToolTarget(ctx, ref)
		if err != nil {
			return nil, fmt.Errorf(
				"map selected Tool %s/%s: %w",
				ref.RootID,
				ref.ArtifactID,
				err,
			)
		}
		resolved, err := b.deps.Tools.ResolveToolTarget(ctx, target)
		if err != nil {
			return nil, fmt.Errorf(
				"resolve selected Tool %s/%s: %w",
				ref.RootID,
				ref.ArtifactID,
				err,
			)
		}
		if err := validateV3ToolSelection(sel, resolved); err != nil {
			return nil, err
		}
		choice, call, ok := v3GoOrProviderChoice(resolved)
		if !ok {
			return nil, fmt.Errorf(
				"selected Tool %s/%s has no supported v3 binding",
				ref.RootID,
				ref.ArtifactID,
			)
		}
		out = append(out, preparedChoice{Choice: choice, Call: call})
	}
	return out, nil
}

func validateV3ToolSelection(
	selection sessionv3.ArtifactSelection,
	resolved toolAPI.ResolvedToolView,
) error {
	artifact := resolved.Tool.Artifact
	if selection.ExpectedRevision != 0 &&
		selection.ExpectedRevision != artifact.Revision {
		return errors.New("selected Tool revision changed")
	}
	if selection.ExpectedDigest != "" &&
		(artifact.ResolvedDefinition == nil ||
			selection.ExpectedDigest != string(*artifact.ResolvedDefinition)) {
		return errors.New("selected Tool Definition changed")
	}
	return nil
}

func v3GoOrProviderChoice(resolved toolAPI.ResolvedToolView) (inferenceSpec.ToolChoice, sessionv3.ToolCall, bool) {
	name := string(resolved.Tool.Name)
	if strings.TrimSpace(name) == "" {
		return inferenceSpec.ToolChoice{}, sessionv3.ToolCall{}, false
	}
	args := getEmptySchema()
	if len(resolved.Tool.InputSchema) != 0 {
		var err error
		args, err = jsonutil.DecodeJSONRaw[map[string]any](resolved.Tool.InputSchema)
		if err != nil {
			return inferenceSpec.ToolChoice{}, sessionv3.ToolCall{}, false
		}
	}
	digest := ""
	if resolved.Tool.Artifact.ResolvedDefinition != nil {
		digest = string(*resolved.Tool.Artifact.ResolvedDefinition)
	}
	pin := sessionv3.ArtifactPin{
		RootID:     string(resolved.Tool.Artifact.RootID),
		ArtifactID: string(resolved.Tool.Artifact.ID),
		Revision:   resolved.Tool.Artifact.Revision,
		Digest:     digest,
	}
	switch resolved.Tool.Implementation.Kind {
	case toolv1.ImplementationKindGo:
		function := strings.TrimSpace(resolved.Tool.Implementation.Function)
		if function == "" {
			return inferenceSpec.ToolChoice{}, sessionv3.ToolCall{}, false
		}
		id := stableV3ChoiceID(
			"v3-go",
			pin.RootID,
			pin.ArtifactID,
			name,
			pin.Digest,
		)
		choice := inferenceSpec.ToolChoice{
			Type:        inferenceSpec.ToolTypeFunction,
			ID:          id,
			Name:        name,
			Description: resolved.Tool.Description,
			Arguments:   args,
		}
		call := sessionv3.ToolCall{
			Name:        name,
			ChoiceID:    id,
			DisplayName: resolved.Tool.DisplayName,
			Description: resolved.Tool.Description,
			Binding: sessionv3.ToolBinding{
				Kind: sessionv3.ToolBindingKindGo,
				Go: &sessionv3.GoToolBinding{
					Target:      pin,
					LogicalName: name,
					Function:    function,
				},
			},
		}
		return choice, call, true
	case toolv1.ImplementationKindSDK:
		sdkType := strings.TrimSpace(string(resolved.Tool.Implementation.SDKToolType))
		if sdkType == "" {
			return inferenceSpec.ToolChoice{}, sessionv3.ToolCall{}, false
		}
		id := stableV3ChoiceID(
			"v3-sdk",
			pin.RootID,
			pin.ArtifactID,
			name,
			pin.Digest,
		)
		choice := inferenceSpec.ToolChoice{
			Type:        inferenceSpec.ToolType(sdkType),
			ID:          id,
			Name:        name,
			Description: resolved.Tool.Description,
			Arguments:   args,
		}
		call := sessionv3.ToolCall{
			Name:        name,
			ChoiceID:    id,
			DisplayName: resolved.Tool.DisplayName,
			Description: resolved.Tool.Description,
			Binding: sessionv3.ToolBinding{
				Kind: sessionv3.ToolBindingKindProvider,
				Provider: &sessionv3.ProviderToolBinding{
					Type: sdkType,
					Name: name,
				},
			},
		}
		return choice, call, true
	default:
		return inferenceSpec.ToolChoice{}, sessionv3.ToolCall{}, false
	}
}

func stableV3ChoiceID(prefix string, values ...string) string {
	var input strings.Builder
	input.WriteString(prefix)
	for _, value := range values {
		input.WriteString("\x00")
		input.WriteString(value)
	}
	sum := sha256.Sum256([]byte(input.String()))
	return prefix + "-" + hex.EncodeToString(sum[:])[:16]
}

func getEmptySchema() map[string]any {
	return map[string]any{"type": "object"}
}
