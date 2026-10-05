package consumerapi

import (
	"context"
	"sort"

	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
	toolDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool/domain"
)

func (a *API) listCollectionTools(
	ctx context.Context,
	value plugin.CollectionView,
) ([]ToolListItem, error) {
	allowed := make(map[spec.LogicalName]struct{}, len(value.Members))
	for _, member := range value.Members {
		if member.Type == "tool" {
			allowed[member.Name] = struct{}{}
		}
	}

	entries, err := a.cat.ListByRoot(
		ctx,
		a.builtinRoot,
		catalogModel.ListOptions{Kind: toolDomain.ToolArtifactKind},
	)
	if err != nil {
		return nil, err
	}

	output := make([]ToolListItem, 0, len(allowed))
	for _, entry := range entries {
		if entry.Kind != toolDomain.ToolArtifactKind ||
			entry.State != artifactModel.StateAvailable {
			continue
		}
		if _, found := allowed[entry.LogicalName]; !found {
			continue
		}

		item := ToolListItem{
			Ref:         entry.Ref(),
			Name:        entry.LogicalName,
			DisplayName: entry.DisplayName,
			State:       entry.State,
			Enabled:     entry.Enabled,
			Revision:    entry.Revision,
			BuiltIn:     true,
		}
		if entry.Definition != nil {
			item.DefinitionDigest = entry.Definition.Digest
			item.Description = entry.Definition.Description
		}
		output = append(output, item)
	}

	sort.Slice(output, func(left, right int) bool {
		if output[left].Name != output[right].Name {
			return output[left].Name < output[right].Name
		}
		return output[left].Ref.ArtifactID <
			output[right].Ref.ArtifactID
	})
	return output, nil
}
