package sessionv3runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
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
			continue
		}
		target := composition.CapabilityTarget{
			Form:     composition.TargetFormArtifact,
			Type:     declaration.TypeTool,
			Artifact: &ref,
		}
		resolved, err := b.deps.Tools.ResolveToolTarget(ctx, target)
		if err != nil {
			continue
		}
		choice, call, ok := v3GoOrProviderChoice(resolved)
		if !ok {
			continue
		}
		out = append(out, preparedChoice{Choice: choice, Call: call})
	}
	return out, nil
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
		id := stableV3ChoiceID("v3-go", name, pin.Digest)
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
		id := stableV3ChoiceID("v3-sdk", name, pin.Digest)
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

func stableV3ChoiceID(prefix, name, digest string) string {
	sum := sha256.Sum256([]byte(prefix + "\x00" + name + "\x00" + digest))
	return prefix + "-" + hex.EncodeToString(sum[:])[:16]
}

func getEmptySchema() map[string]any {
	return map[string]any{"type": "object"}
}
