package aggregate

import (
	"context"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	skillRuntime "github.com/flexigpt/flexigpt-app/internal/skill/runtime"
	skillStore "github.com/flexigpt/flexigpt-app/internal/skill/store"
)

const artifactCollectionCatalogPrefix = "artifact-collection:"

// CatalogSource adapts Artifact Store Collections to the runtime-owned
// CatalogSource contract. Runtime treats CatalogID as an opaque string.
type CatalogSource struct {
	router *skillStore.ArtifactRouter
}

func NewCatalogSource(
	router *skillStore.ArtifactRouter,
) (*CatalogSource, error) {
	if router == nil {
		return nil, fmt.Errorf(
			"%w: Artifact Skill router is nil",
			basespec.ErrInvalid,
		)
	}
	return &CatalogSource{router: router}, nil
}

func (s *CatalogSource) Skills(
	ctx context.Context,
	catalogID skillRuntime.CatalogID,
) ([]skillRuntime.SkillRegistration, error) {
	if s == nil || s.router == nil {
		return nil, fmt.Errorf(
			"%w: Artifact Skill catalog source is unavailable",
			basespec.ErrClosed,
		)
	}

	ref, err := collectionRefForCatalogID(catalogID)
	if err != nil {
		return nil, err
	}

	values, err := s.router.ListCollectionSkills(ctx, ref)
	if err != nil {
		return nil, err
	}

	output := make([]skillRuntime.SkillRegistration, 0, len(values))
	for _, value := range values {
		output = append(output, skillRuntime.SkillRegistration{
			Definition: value.Definition,
			Revision:   value.Version,
		})
	}
	return output, nil
}

func CollectionCatalogID(
	ref collection.CollectionRef,
) (skillRuntime.CatalogID, error) {
	if err := ref.Validate(); err != nil {
		return "", err
	}
	return skillRuntime.CatalogID(
		artifactCollectionCatalogPrefix +
			string(ref.RootID) + ":" +
			string(ref.CollectionID),
	), nil
}

func collectionRefForCatalogID(
	catalogID skillRuntime.CatalogID,
) (collection.CollectionRef, error) {
	raw, found := strings.CutPrefix(
		string(catalogID),
		artifactCollectionCatalogPrefix,
	)
	if !found {
		return collection.CollectionRef{}, fmt.Errorf(
			"%w: unsupported Skill catalog ID %q",
			basespec.ErrInvalid,
			catalogID,
		)
	}

	rootID, collectionID, found := strings.Cut(raw, ":")
	if !found {
		return collection.CollectionRef{}, fmt.Errorf(
			"%w: malformed Skill catalog ID %q",
			basespec.ErrInvalid,
			catalogID,
		)
	}

	ref := collection.CollectionRef{
		RootID:       basespec.RootID(rootID),
		CollectionID: basespec.CollectionID(collectionID),
	}
	if err := ref.Validate(); err != nil {
		return collection.CollectionRef{}, err
	}
	return ref, nil
}
