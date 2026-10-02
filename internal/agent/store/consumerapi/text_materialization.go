package consumerapi

import (
	"context"
	"fmt"

	artifact "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// MaterializeAgentText resolves one Text Artifact through the Artifact Store's
// verified resource boundary. It is exposed by Agent Store because Agent
// declarations can own contained Text declarations, while standalone Text
// management does not yet have a dedicated consumer API.
func (a *API) MaterializeAgentText(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (AgentTextMaterialization, error) {
	if a == nil || a.texts == nil {
		return AgentTextMaterialization{}, spec.ErrClosed
	}
	if ctx == nil {
		return AgentTextMaterialization{}, fmt.Errorf(
			"%w: Agent Text materialization context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return AgentTextMaterialization{}, err
	}
	if err := ref.Validate(); err != nil {
		return AgentTextMaterialization{}, err
	}

	value, err := a.texts.Resolve(ctx, ref)
	if err != nil {
		return AgentTextMaterialization{}, err
	}

	return AgentTextMaterialization{
		Artifact:         value.Artifact,
		ArtifactRevision: value.ArtifactRevision,
		DefinitionDigest: value.DefinitionDigest,
		Name:             spec.LogicalName(value.Name),
		Insert:           value.Insert,
		MediaType:        value.MediaType,
		Content:          value.Content,
		Locator:          value.Locator,
		BuiltIn:          value.Artifact.RootID == agentBuiltinRootID(),
	}, nil
}
