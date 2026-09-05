package bundle

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"

	skillArtifact "github.com/flexigpt/flexigpt-app/internal/skill/store/artifact"
)

// ResolvedSkill is the typed Skill Bundle handoff to application composition.
// It is not persisted and is never exposed as a durable runtime identity.
type ResolvedSkill struct {
	Artifact   artifact.ArtifactRef
	Collection collection.CollectionRef

	Name     string
	Location string
	Version  string
}

// ListResolvedSkills is deliberately fail-closed. A collection reconciliation
// must not retain a previous runtime definition when a current Artifact can no
// longer be projected.
func (a *API) ListResolvedSkills(
	ctx context.Context,
	bundle collection.CollectionRef,
) ([]ResolvedSkill, error) {
	if err := a.Ready(); err != nil {
		return nil, err
	}
	if err := bundle.Validate(); err != nil {
		return nil, err
	}

	bundleValue, err := a.GetBundle(ctx, bundle)
	if err != nil {
		return nil, err
	}
	records, err := a.dependencies.Artifacts.ListByCollection(ctx, bundle)
	if err != nil {
		return nil, err
	}

	eligible := make([]artifact.Artifact, 0, len(records))
	for _, record := range records {
		if record.Kind != artifactbuiltin.AgentSkillArtifactKind ||
			!record.Enabled ||
			record.State != artifact.StateAvailable {
			continue
		}
		eligible = append(eligible, record)
	}
	if len(eligible) == 0 {
		return []ResolvedSkill{}, nil
	}
	if !bundleValue.Collection.Enabled {
		return nil, fmt.Errorf(
			"%w: skill bundle %q is disabled",
			basespec.ErrReferenceUnresolved,
			bundle.CollectionID,
		)
	}

	catalogValue, err := a.currentBundleCatalog(ctx, bundleValue)
	if err != nil {
		return nil, err
	}
	output := make([]ResolvedSkill, 0, len(eligible))
	for _, record := range eligible {
		value, err := a.resolvedSkillFromSnapshot(
			ctx,
			record,
			bundleValue,
			catalogValue,
		)
		if err != nil {
			return nil, err
		}
		output = append(output, value)
	}

	sort.Slice(output, func(left, right int) bool {
		if output[left].Name != output[right].Name {
			return output[left].Name < output[right].Name
		}
		return output[left].Artifact.ArtifactID <
			output[right].Artifact.ArtifactID
	})
	return output, nil
}

// ResolveSkill resolves one Artifact-backed Skill Bundle member.
//
// It verifies Artifact membership, Collection kind, Collection and Artifact
// enablement, catalog currentness, definition compatibility, and the source
// snapshot generation. Source adapters and MapStore remain responsible for
// their own containment and filesystem behavior.
func (a *API) ResolveSkill(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (ResolvedSkill, error) {
	if err := a.Ready(); err != nil {
		return ResolvedSkill{}, err
	}
	if err := ref.Validate(); err != nil {
		return ResolvedSkill{}, err
	}

	record, err := a.dependencies.Artifacts.Get(ctx, ref)
	if err != nil {
		return ResolvedSkill{}, err
	}

	bundleRef := collection.CollectionRef{
		RootID:       record.RootID,
		CollectionID: record.CollectionID,
	}
	bundle, err := a.GetBundle(ctx, bundleRef)
	if err != nil {
		return ResolvedSkill{}, err
	}

	snapshot, err := a.currentBundleCatalog(ctx, bundle)
	if err != nil {
		return ResolvedSkill{}, err
	}
	return a.resolvedSkillFromSnapshot(
		ctx,
		record,
		bundle,
		snapshot,
	)
}

func (a *API) resolvedSkillFromSnapshot(
	ctx context.Context,
	record artifact.Artifact,
	bundle Bundle,
	snapshot catalog.Snapshot,
) (ResolvedSkill, error) {
	bundleRef := bundle.Collection.Ref()
	if record.RootID != bundleRef.RootID ||
		record.CollectionID != bundleRef.CollectionID {
		return ResolvedSkill{}, fmt.Errorf(
			"%w: Skill Artifact belongs to another bundle",
			basespec.ErrInvalid,
		)
	}
	if !bundle.Collection.Enabled {
		return ResolvedSkill{}, fmt.Errorf(
			"%w: skill bundle %q is disabled",
			basespec.ErrReferenceUnresolved,
			bundleRef.CollectionID,
		)
	}
	if record.Kind != artifactbuiltin.AgentSkillArtifactKind ||
		!record.Enabled ||
		record.State != artifact.StateAvailable ||
		record.ResolvedDefinition == nil {
		return ResolvedSkill{}, fmt.Errorf(
			"%w: Skill Artifact %q is not enabled and available",
			basespec.ErrReferenceUnresolved,
			record.ID,
		)
	}

	occurrence, err := currentSkillOccurrence(snapshot, record)
	if err != nil {
		return ResolvedSkill{}, err
	}
	expectedGeneration := snapshot.SourceGenerations[record.Binding.SourceID]
	expectedSourceRevision := snapshot.SourceRevisions[record.Binding.SourceID]
	if expectedGeneration == "" || expectedSourceRevision == 0 {
		return ResolvedSkill{}, fmt.Errorf(
			"%w: Skill Artifact source has no current catalog state",
			basespec.ErrCatalogStale,
		)
	}

	value, err := definitionForArtifact(snapshot, record)
	if err != nil {
		return ResolvedSkill{}, err
	}
	if err := skillArtifact.ValidateDefinition(value); err != nil {
		return ResolvedSkill{}, err
	}

	resolved, err := a.dependencies.Resources.ResolveArtifact(
		ctx,
		record.Ref(),
		artifactstore.ResolveOptions{},
	)
	if err != nil {
		return ResolvedSkill{}, err
	}
	if resolved.Artifact.Revision != record.Revision ||
		resolved.Artifact.Binding != record.Binding ||
		resolved.Collection.Revision != bundle.Collection.Revision ||
		resolved.CatalogRevision != snapshot.Revision ||
		resolved.Definition.Digest != value.Digest {
		return ResolvedSkill{}, fmt.Errorf(
			"%w: Skill resource changed during runtime resolution",
			basespec.ErrCatalogStale,
		)
	}
	if resolved.Source.ID != record.Binding.SourceID ||
		resolved.Source.Revision != expectedSourceRevision ||
		resolved.SourceGeneration != expectedGeneration ||
		resolved.Occurrence.SourceContentDigest == nil ||
		*resolved.Occurrence.SourceContentDigest !=
			*occurrence.SourceContentDigest {
		return ResolvedSkill{}, fmt.Errorf(
			"%w: Skill source changed after catalog publication",
			basespec.ErrCatalogStale,
		)
	}

	packageLocator, err := skillArtifact.RuntimePackageLocator(
		record.Binding.Locator,
		record.Binding.SubresourceLocator,
	)
	if err != nil {
		return ResolvedSkill{}, err
	}
	location, err := a.dependencies.Resources.ResolveVerifiedLocalPath(
		ctx,
		resolved,
		packageLocator,
	)
	if err != nil {
		return ResolvedSkill{}, err
	}

	versionInput := string(value.Digest) + "\x00" +
		string(*occurrence.SourceContentDigest) + "\x00" +
		strconv.FormatUint(record.Revision, 10) + "\x00" +
		expectedGeneration

	return ResolvedSkill{
		Artifact:   record.Ref(),
		Collection: bundleRef,
		Name:       string(value.LogicalName),
		Location:   location,
		Version: "skill.bundle:" + string(
			cryptoutil.DigestBytes([]byte(versionInput)),
		),
	}, nil
}

func definitionForArtifact(
	snapshot catalog.Snapshot,
	record artifact.Artifact,
) (providerapi.Definition, error) {
	if record.ResolvedDefinition == nil {
		return providerapi.Definition{}, fmt.Errorf(
			"%w: Skill Artifact %q has no current definition",
			basespec.ErrReferenceUnresolved,
			record.ID,
		)
	}
	if snapshot.RootID != record.RootID ||
		snapshot.CollectionID != record.CollectionID {
		return providerapi.Definition{}, fmt.Errorf(
			"%w: Skill catalog belongs to another Collection",
			basespec.ErrInvalid,
		)
	}

	value, err := catalog.DefinitionForOccurrence(snapshot, catalog.OccurrenceKey{
		CollectionID:       record.CollectionID,
		SourceID:           record.Binding.SourceID,
		Locator:            record.Binding.Locator,
		SubresourceLocator: record.Binding.SubresourceLocator,
	})
	if err != nil {
		return providerapi.Definition{}, err
	}
	if value.Digest != *record.ResolvedDefinition {
		return providerapi.Definition{}, fmt.Errorf(
			"%w: Skill Artifact %q catalog definition changed",
			basespec.ErrConflict,
			record.ID,
		)
	}
	return value, nil
}

func currentSkillOccurrence(
	snapshot catalog.Snapshot,
	record artifact.Artifact,
) (catalog.Occurrence, error) {
	if record.ResolvedDefinition == nil {
		return catalog.Occurrence{}, fmt.Errorf(
			"%w: Skill Artifact has no resolved definition",
			basespec.ErrCatalogStale,
		)
	}
	key := catalog.OccurrenceKey{
		CollectionID:       record.CollectionID,
		SourceID:           record.Binding.SourceID,
		Locator:            record.Binding.Locator,
		SubresourceLocator: record.Binding.SubresourceLocator,
	}
	for _, occurrence := range snapshot.Occurrences {
		if occurrence.Key != key {
			continue
		}
		if occurrence.State != catalog.OccurrenceValid ||
			occurrence.Kind != artifactbuiltin.AgentSkillArtifactKind ||
			occurrence.DefinitionDigest == nil ||
			occurrence.SourceContentDigest == nil ||
			*occurrence.DefinitionDigest != *record.ResolvedDefinition {
			break
		}
		return catalog.CloneOccurrence(occurrence), nil
	}
	return catalog.Occurrence{}, fmt.Errorf(
		"%w: Skill Artifact does not match the current catalog occurrence",
		basespec.ErrCatalogStale,
	)
}
