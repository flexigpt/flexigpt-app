package skill

import (
	"context"
	"sort"

	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	skillSource "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/source"
)

func (a *Service) ListSkills(
	ctx context.Context,
	request ListSkillsRequest,
) ([]SkillListItem, error) {
	entries, err := a.cat.ListByRoot(
		ctx,
		request.RootID,
		catalogModel.ListOptions{
			Kind:    skillSource.SkillArtifactKind,
			Enabled: request.Enabled,
		},
	)
	if err != nil {
		return nil, err
	}

	output := make([]SkillListItem, 0, len(entries))
	for _, entry := range entries {
		item := a.skillListItem(entry)
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

func (a *Service) skillListItem(
	entry catalogModel.Entry,
) SkillListItem {
	digest := cryptoutil.Digest("")
	description := ""
	if entry.Definition != nil {
		digest = entry.Definition.Digest
		description = entry.Definition.Description
	}

	managed := false
	if entry.State == artifactModel.StateAvailable &&
		a.support.PluginProfile.Source.MatchesSourceMetadata(entry.Source) &&
		entry.Binding.SubresourceLocator == "" {
		_, err := a.support.Package.AddressFromLocator(
			entry.Binding.Locator,
		)
		managed = err == nil
	}

	return SkillListItem{
		Ref:              entry.Ref(),
		Name:             entry.LogicalName,
		DisplayName:      entry.DisplayName,
		Description:      description,
		State:            entry.State,
		Enabled:          entry.Enabled,
		Revision:         entry.Revision,
		DefinitionDigest: digest,
		BuiltIn:          entry.Ref().RootID == a.support.BuiltinRoot,
		Managed:          managed,
	}
}
