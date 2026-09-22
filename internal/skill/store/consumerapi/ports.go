package consumerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/collection"
)

// BuiltinStore is the narrow Skill capability required by the protected
// built-in installer. It does not expose generic Store mutation internals.
type BuiltinStore interface {
	InstallBuiltInSkillPackage(
		ctx context.Context,

		request BuiltInSkillPackageInstallRequest,
	) ([]artifact.Artifact, error)

	RemoveBuiltInSkillPackage(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
		address source.ManagedPackageAddress,
	) error

	EnsureBuiltInSkillSourceCurrent(ctx context.Context, rootID root.RootID, sourceID source.SourceID) error
}

type BaselineEnsurer interface {
	EnsureSkillBaselineCollection(
		ctx context.Context,
		rootID root.RootID,
	) (collection.CollectionView, error)
}

type builtinStore struct {
	api *API
}

func NewBuiltinStore(api *API) (BuiltinStore, error) {
	if api == nil {
		return nil, fmt.Errorf(
			"%w: Skill built-in Store requires an API",
			basespec.ErrInvalid,
		)
	}
	return &builtinStore{api: api}, nil
}

func (s *builtinStore) InstallBuiltInSkillPackage(
	ctx context.Context,
	request BuiltInSkillPackageInstallRequest,
) ([]artifact.Artifact, error) {
	if s == nil || s.api == nil {
		return nil, basespec.ErrClosed
	}
	return s.api.installBuiltInSkillPackage(ctx, request)
}

func (s *builtinStore) RemoveBuiltInSkillPackage(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
	address source.ManagedPackageAddress,
) error {
	if s == nil || s.api == nil {
		return basespec.ErrClosed
	}
	return s.api.removeBuiltInSkillPackage(
		ctx, rootID, sourceID, address,
	)
}

func (s *builtinStore) EnsureBuiltInSkillSourceCurrent(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
) error {
	if s == nil || s.api == nil {
		return basespec.ErrClosed
	}
	return s.api.ensureBuiltInSkillSourceCurrent(
		ctx, rootID, sourceID,
	)
}

type baselineEnsurer struct {
	api *API
}

func NewBaselineEnsurer(api *API) (BaselineEnsurer, error) {
	if api == nil || api.collections == nil {
		return nil, fmt.Errorf(
			"%w: Skill baseline ensurer requires collections",
			basespec.ErrInvalid,
		)
	}
	return &baselineEnsurer{api: api}, nil
}

func (s *baselineEnsurer) EnsureSkillBaselineCollection(
	ctx context.Context,
	rootID root.RootID,
) (collection.CollectionView, error) {
	if s == nil || s.api == nil {
		return collection.CollectionView{}, basespec.ErrClosed
	}
	return s.api.ensureSkillBaselineCollection(ctx, rootID)
}
