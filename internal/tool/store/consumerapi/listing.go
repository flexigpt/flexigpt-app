package consumerapi

import (
	"context"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/catalog"
	"github.com/flexigpt/flexigpt-app/internal/collection"
	toolDomain "github.com/flexigpt/flexigpt-app/internal/tool/store/domain"
)

func (a *API) listCollectionTools(
	ctx context.Context,
	value collection.CollectionView,
) ([]ToolListItem, error) {
	allowed := make(map[basespec.LogicalName]struct{}, len(value.Members))
	for _, member := range value.Members {
		if member.Type == "tool" {
			allowed[member.Name] = struct{}{}
		}
	}

	entries, err := a.artifacts.ListByRoot(
		ctx,
		a.builtinRoot,
		catalog.ListOptions{},
	)
	if err != nil {
		return nil, err
	}

	output := make([]ToolListItem, 0, len(allowed))
	for _, entry := range entries {
		if entry.Kind != toolDomain.ToolArtifactKind ||
			entry.State != artifact.StateAvailable {
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
