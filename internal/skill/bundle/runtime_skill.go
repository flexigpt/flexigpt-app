package bundle

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"

	skillArtifact "github.com/flexigpt/flexigpt-app/internal/skill/artifact"
)

// RuntimeSkill is the typed Skill Bundle handoff to application composition.
// It is not persisted and is never exposed as a durable runtime identity.
type RuntimeSkill struct {
	Artifact   artifact.ArtifactRef
	Collection collection.CollectionRef

	Name     string
	Location string
	Version  string
}

// ListRuntimeSkills is deliberately fail-closed. A collection reconciliation
// must not retain a previous runtime definition when a current Artifact can no
// longer be projected.
func (a *API) ListRuntimeSkills(
	ctx context.Context,
	bundle collection.CollectionRef,
) (output []RuntimeSkill, returnErr error) {
	records, err := a.ListSkills(ctx, bundle)
	if err != nil {
		return nil, err
	}

	eligible := make([]artifact.ArtifactRef, 0, len(records))
	for _, record := range records {
		if !record.Enabled || record.State != artifact.StateAvailable {
			continue
		}
		eligible = append(eligible, record.Ref())
	}
	if len(eligible) == 0 {
		return []RuntimeSkill{}, nil
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

	output = make([]RuntimeSkill, 0, len(eligible))
	for _, ref := range eligible {
		value, err := a.LoadRuntimeSkill(sessionCtx, ref)
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

// LoadRuntimeSkill resolves one Artifact-backed Skill Bundle member.
//
// It verifies Artifact membership, Collection kind, Collection and Artifact
// enablement, catalog currentness, definition compatibility, and the source
// snapshot generation. Source adapters and MapStore remain responsible for
// their own containment and filesystem behavior.
func (a *API) LoadRuntimeSkill(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (RuntimeSkill, error) {
	if err := a.Ready(); err != nil {
		return RuntimeSkill{}, err
	}
	if err := ref.Validate(); err != nil {
		return RuntimeSkill{}, err
	}

	record, err := a.dependencies.Artifacts.Get(ctx, ref)
	if err != nil {
		return RuntimeSkill{}, err
	}
	bundleRef := collection.CollectionRef{
		RootID:       record.RootID,
		CollectionID: record.CollectionID,
	}
	bundle, err := a.GetBundle(ctx, bundleRef)
	if err != nil {
		return RuntimeSkill{}, err
	}
	if !bundle.Collection.Enabled {
		return RuntimeSkill{}, fmt.Errorf(
			"%w: skill bundle %q is disabled",
			basespec.ErrReferenceUnresolved,
			bundleRef.CollectionID,
		)
	}
	if record.Kind != skillArtifact.Kind ||
		!record.Enabled ||
		record.State != artifact.StateAvailable ||
		record.ResolvedDefinition == nil {
		return RuntimeSkill{}, fmt.Errorf(
			"%w: Skill Artifact %q is not enabled and available",
			basespec.ErrReferenceUnresolved,
			ref.ArtifactID,
		)
	}

	snapshot, err := a.currentBundleCatalog(ctx, bundle)
	if err != nil {
		return RuntimeSkill{}, err
	}
	occurrence, err := currentSkillOccurrence(snapshot, record)
	if err != nil {
		return RuntimeSkill{}, err
	}
	expectedGeneration := snapshot.SourceGenerations[record.Binding.SourceID]
	expectedSourceRevision := snapshot.SourceRevisions[record.Binding.SourceID]
	if expectedGeneration == "" || expectedSourceRevision == 0 {
		return RuntimeSkill{}, fmt.Errorf(
			"%w: Skill Artifact source has no current catalog state",
			basespec.ErrCatalogStale,
		)
	}

	value, err := definition.ReadCanonical(
		ctx,
		a.dependencies.Definitions,
		record.RootID,
		*record.ResolvedDefinition,
	)
	if err != nil {
		return RuntimeSkill{}, err
	}
	if err := skillArtifact.ValidateDefinition(value); err != nil {
		return RuntimeSkill{}, err
	}

	sourceValue, err := a.dependencies.SourceRuntime.Get(
		ctx,
		record.RootID,
		record.Binding.SourceID,
	)
	if err != nil {
		return RuntimeSkill{}, err
	}
	if sourceValue.Revision != expectedSourceRevision {
		return RuntimeSkill{}, fmt.Errorf(
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
		return RuntimeSkill{}, err
	}

	versionInput := string(value.Digest) + "\x00" +
		string(*occurrence.SourceContentDigest) + "\x00" +
		strconv.FormatUint(record.Revision, 10) + "\x00" +
		expectedGeneration

	return RuntimeSkill{
		Artifact:   record.Ref(),
		Collection: bundleRef,
		Name:       string(value.LogicalName),
		Location:   location,
		Version: "skill.bundle:" + string(
			cryptoutil.DigestBytes([]byte(versionInput)),
		),
	}, nil
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
			occurrence.Kind != skillArtifact.Kind ||
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
