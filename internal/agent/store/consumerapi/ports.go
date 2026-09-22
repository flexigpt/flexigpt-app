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

// BuiltinStore is the narrow Agent Store capability used by the trusted
// protected-topology Agent installer.
type BuiltinStore interface {
	InstallBuiltInAgentPackage(
		ctx context.Context,
		request BuiltInAgentPackageInstallRequest,
	) ([]artifact.Artifact, error)

	ValidateBuiltInAgentPackage(
		ctx context.Context,
		request BuiltInAgentPackageInstallRequest,
	) error

	RemoveBuiltInAgentPackage(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
		address source.ManagedPackageAddress,
	) error

	EnsureBuiltInAgentSourceCurrent(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
	) error
}

type BaselineEnsurer interface {
	EnsureAgentBaselineCollection(
		ctx context.Context,
		rootID root.RootID,
	) (collection.CollectionView, error)
}

type builtinStore struct {
	api *API
}

func NewBuiltinStore(
	api *API,
) (BuiltinStore, error) {
	if api == nil {
		return nil, fmt.Errorf(
			"%w: Agent built-in Store requires an API",
			basespec.ErrInvalid,
		)
	}
	return &builtinStore{api: api}, nil
}

func (s *builtinStore) InstallBuiltInAgentPackage(
	ctx context.Context,
	request BuiltInAgentPackageInstallRequest,
) ([]artifact.Artifact, error) {
	if s == nil || s.api == nil {
		return nil, basespec.ErrClosed
	}
	return s.api.installBuiltInAgentPackage(ctx, request)
}

func (s *builtinStore) ValidateBuiltInAgentPackage(
	ctx context.Context,
	request BuiltInAgentPackageInstallRequest,
) error {
	if s == nil || s.api == nil {
		return basespec.ErrClosed
	}
	return s.api.validateBuiltInAgentPackage(ctx, request)
}

func (s *builtinStore) RemoveBuiltInAgentPackage(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
	address source.ManagedPackageAddress,
) error {
	if s == nil || s.api == nil {
		return basespec.ErrClosed
	}
	return s.api.removeBuiltInAgentPackage(
		ctx, rootID, sourceID, address,
	)
}

func (s *builtinStore) EnsureBuiltInAgentSourceCurrent(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
) error {
	if s == nil || s.api == nil {
		return basespec.ErrClosed
	}
	return s.api.ensureBuiltInAgentSourceCurrent(
		ctx, rootID, sourceID,
	)
}

type baselineEnsurer struct {
	api *API
}

func NewBaselineEnsurer(
	api *API,
) (BaselineEnsurer, error) {
	if api == nil || api.collections == nil {
		return nil, fmt.Errorf(
			"%w: Agent baseline ensurer requires collections",
			basespec.ErrInvalid,
		)
	}
	return &baselineEnsurer{api: api}, nil
}

func (s *baselineEnsurer) EnsureAgentBaselineCollection(
	ctx context.Context,
	rootID root.RootID,
) (collection.CollectionView, error) {
	if s == nil || s.api == nil {
		return collection.CollectionView{}, basespec.ErrClosed
	}
	return s.api.ensureAgentBaselineCollection(ctx, rootID)
}
