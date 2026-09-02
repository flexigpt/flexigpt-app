package bundle

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
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
) (output []ResolvedSkill, returnErr error) {
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

	sessionCtx, session, err := source.NewVerificationSession(
		ctx,
	)
	if err != nil {
		return nil, err
	}

	defer func() {
		returnErr = errors.Join(
			returnErr,
			session.Close(sessionCtx),
		)
	}()

	sourcesByID := make(map[basespec.SourceID]source.Source)
	output = make([]ResolvedSkill, 0, len(eligible))
	for _, record := range eligible {
		sourceValue, found := sourcesByID[record.Binding.SourceID]
		if !found {
			sourceValue, err = a.dependencies.SourceRuntime.Get(
				sessionCtx,
				record.RootID,
				record.Binding.SourceID,
			)
			if err != nil {
				return nil, err
			}
			sourcesByID[record.Binding.SourceID] = sourceValue
		}

		value, err := a.resolvedSkillFromSnapshot(
			sessionCtx,
			record,
			bundleValue,
			catalogValue,
			sourceValue,
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
	sourceValue, err := a.dependencies.SourceRuntime.Get(
		ctx,
		record.RootID,
		record.Binding.SourceID,
	)
	if err != nil {
		return ResolvedSkill{}, err
	}
	return a.resolvedSkillFromSnapshot(
		ctx,
		record,
		bundle,
		snapshot,
		sourceValue,
	)
}

func (a *API) resolvedSkillFromSnapshot(
	ctx context.Context,
	record artifact.Artifact,
	bundle Bundle,
	snapshot catalog.Snapshot,
	sourceValue source.Source,
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

	if sourceValue.RootID != record.RootID ||
		sourceValue.ID != record.Binding.SourceID {
		return ResolvedSkill{}, fmt.Errorf(
			"%w: Skill source does not match its Artifact binding",
			basespec.ErrInvalid,
		)
	}
	if sourceValue.Revision != expectedSourceRevision {
		return ResolvedSkill{}, fmt.Errorf(
			"%w: Skill source changed after catalog publication",
			basespec.ErrCatalogStale,
		)
	}
	location, err := skillArtifact.ResolveRuntimePackage(
		ctx,
		a.dependencies.SourceRuntime,
		sourceValue,
		record.Binding.Locator,
		record.Binding.SubresourceLocator,
		expectedGeneration,
		*occurrence.SourceContentDigest,
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
) (definition.Definition, error) {
	if record.ResolvedDefinition == nil {
		return definition.Definition{}, fmt.Errorf(
			"%w: Skill Artifact %q has no current definition",
			basespec.ErrReferenceUnresolved,
			record.ID,
		)
	}
	if snapshot.RootID != record.RootID ||
		snapshot.CollectionID != record.CollectionID {
		return definition.Definition{}, fmt.Errorf(
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
		return definition.Definition{}, err
	}
	if value.Digest != *record.ResolvedDefinition {
		return definition.Definition{}, fmt.Errorf(
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
