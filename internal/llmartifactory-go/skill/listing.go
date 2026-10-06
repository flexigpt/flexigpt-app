package skill

import (
	"context"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/managedfs"
	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	pluginAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
	skillPackage "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/package"
	skillSource "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/source"
)

func (a *Service) ListSkills(
	ctx context.Context,
	request ListSkillsRequest,
) ([]SkillListItem, error) {
	if a == nil || a.artifacts == nil {
		return nil, spec.ErrClosed
	}
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: Skill list context is nil",
			spec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := request.RootID.Validate(); err != nil {
		return nil, err
	}

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
		if entry.Kind != skillSource.SkillArtifactKind {
			continue
		}
		if request.Enabled != nil && entry.Enabled != *request.Enabled {
			continue
		}

		item := skillListItem(entry)
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

func skillListItem(
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
		entry.Source.Kind == managedfs.Kind &&
		entry.Source.StorageKey == pluginAPI.SkillManagedPluginSourceStorageKey &&
		entry.Binding.SubresourceLocator == "" {
		_, err := skillPackage.ManagedPackageAddressFromSkillLocator(
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
		BuiltIn:          entry.Ref().RootID == topology.BuiltinRootID(),
		Managed:          managed,
	}
}
