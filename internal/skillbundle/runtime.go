package skillbundle

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/skillartifact"
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
	if record.Kind != skillartifact.Kind ||
		!record.Enabled ||
		record.State != artifact.StateAvailable ||
		record.ResolvedDefinition == nil {
		return RuntimeSkill{}, fmt.Errorf(
			"%w: Skill Artifact %q is not enabled and available",
			basespec.ErrReferenceUnresolved,
			ref.ArtifactID,
		)
	}

	snapshot, err := catalog.ReadCurrent(ctx, a.dependencies.Catalogs, bundleRef)
	if err != nil {
		return RuntimeSkill{}, err
	}
	expectedGeneration, exists := snapshot.SourceGenerations[record.Binding.SourceID]
	if !exists {
		return RuntimeSkill{}, fmt.Errorf(
			"%w: Skill Artifact source has no current catalog generation",
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
	if err := skillartifact.ValidateDefinition(value); err != nil {
		return RuntimeSkill{}, err
	}
	body, err := skillartifact.DecodeBody(value.Body)
	if err != nil {
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
	localPaths, supported := a.dependencies.SourceRuntime.(source.LocalPathRuntime)
	if !supported || !localPaths.SupportsLocalPath(sourceValue.Kind) {
		return RuntimeSkill{}, fmt.Errorf(
			"%w: Skill source kind %q has no local runtime projection",
			basespec.ErrUnsupported,
			sourceValue.Kind,
		)
	}

	current, err := a.dependencies.SourceRuntime.Open(ctx, sourceValue)
	if err != nil {
		return RuntimeSkill{}, err
	}
	generation := current.Generation()
	confirmErr := current.Confirm(ctx)
	closeErr := current.Close()
	if err := errors.Join(confirmErr, closeErr); err != nil {
		return RuntimeSkill{}, err
	}
	if generation != expectedGeneration {
		return RuntimeSkill{}, fmt.Errorf(
			"%w: Skill source changed after catalog publication",
			basespec.ErrCatalogStale,
		)
	}

	location, err := localPaths.ResolveLocalPath(
		ctx,
		sourceValue,
		record.Binding.Locator,
	)
	if err != nil {
		return RuntimeSkill{}, err
	}
	if filepath.Base(location) != skillartifact.DefinitionFileName {
		return RuntimeSkill{}, fmt.Errorf(
			"%w: Skill binding does not resolve to %q",
			basespec.ErrInvalid,
			skillartifact.DefinitionFileName,
		)
	}

	versionInput := string(value.Digest) + "\x00" +
		string(record.ID) + "\x00" + strconv.FormatUint(record.Revision, 10) + "\x00" + generation

	return RuntimeSkill{
		Artifact:   record.Ref(),
		Collection: bundleRef,
		Name:       body.Name,
		Location:   filepath.Dir(location),
		Version: "skill.bundle:" + string(
			cryptoutil.DigestBytes([]byte(versionInput)),
		),
	}, nil
}

// ListRuntimeSkills is deliberately fail-closed. A collection reconciliation
// must not retain a previous runtime definition when a current Artifact can no
// longer be projected.
func (a *API) ListRuntimeSkills(
	ctx context.Context,
	bundle collection.CollectionRef,
) ([]RuntimeSkill, error) {
	records, err := a.ListSkills(ctx, bundle)
	if err != nil {
		return nil, err
	}

	output := make([]RuntimeSkill, 0, len(records))
	for _, record := range records {
		if !record.Enabled || record.State != artifact.StateAvailable {
			continue
		}
		value, err := a.LoadRuntimeSkill(ctx, record.Ref())
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
