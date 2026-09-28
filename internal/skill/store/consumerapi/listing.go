package consumerapi

import (
	"context"
	"fmt"
	"sort"

	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/collection"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	skillDomain "github.com/flexigpt/flexigpt-app/internal/skill/store/domain"
)

func (a *API) ListSkills(
	ctx context.Context,
	request ListSkillsRequest,
) ([]SkillListItem, error) {
	if a == nil || a.artifacts == nil {
		return nil, basespec.ErrClosed
	}
	if ctx == nil {
		return nil, fmt.Errorf(
			"%w: Skill list context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := request.RootID.Validate(); err != nil {
		return nil, err
	}

	entries, err := a.artifacts.ListByRoot(
		ctx,
		request.RootID,
		catalog.ListOptions{
			Kind:    skillDomain.SkillArtifactKind,
			Enabled: request.Enabled,
		},
	)
	if err != nil {
		return nil, err
	}

	output := make([]SkillListItem, 0, len(entries))
	for _, entry := range entries {
		if entry.Kind != skillDomain.SkillArtifactKind {
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
	entry catalog.Entry,
) SkillListItem {
	digest := cryptoutil.Digest("")
	description := ""
	if entry.Definition != nil {
		digest = entry.Definition.Digest
		description = entry.Definition.Description
	}

	managed := false
	if entry.State == artifact.StateAvailable &&
		entry.Source.Kind == source.SourceKindManagedDirectory &&
		entry.Source.StorageKey == collection.SkillManagedSourceStorageKey &&
		entry.Binding.SubresourceLocator == "" {
		_, err := skillDomain.ManagedPackageAddressFromSkillLocator(
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
		BuiltIn:          entry.Ref().RootID == documentTopology.BuiltinRootID(),
		Managed:          managed,
	}
}
