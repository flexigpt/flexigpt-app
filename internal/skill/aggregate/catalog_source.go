package aggregate

import (
	"context"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	skillRuntime "github.com/flexigpt/flexigpt-app/internal/skill/runtime"
)

const artifactRootCatalogPrefix = "artifact-root:"

// CatalogSource maps one Root's currently enabled Skill Artifacts into the
// runtime-owned CatalogSource contract. The runtime treats CatalogID as opaque.
type CatalogSource struct {
	router *ArtifactRouter
}

func NewCatalogSource(
	router *ArtifactRouter,
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
	rootID, err := RootCatalogIDRoot(catalogID)
	if err != nil {
		return nil, err
	}
	values, err := s.router.ListRootSkills(ctx, rootID)
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

func RootCatalogID(
	rootID root.RootID,
) (skillRuntime.CatalogID, error) {
	if err := rootID.Validate(); err != nil {
		return "", err
	}
	return skillRuntime.CatalogID(
		artifactRootCatalogPrefix + string(rootID),
	), nil
}

func RootCatalogIDRoot(
	catalogID skillRuntime.CatalogID,
) (root.RootID, error) {
	raw, found := strings.CutPrefix(
		string(catalogID),
		artifactRootCatalogPrefix,
	)
	if !found {
		return "", fmt.Errorf(
			"%w: unsupported Skill catalog ID %q",
			basespec.ErrInvalid,
			catalogID,
		)
	}
	rootID := root.RootID(raw)
	if err := rootID.Validate(); err != nil {
		return "", err
	}
	return rootID, nil
}
