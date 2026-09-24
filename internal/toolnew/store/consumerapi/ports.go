package consumerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
)

type BuiltinStore interface {
	InstallBuiltInPackage(
		ctx context.Context,
		request BuiltInPackageInstallRequest,
	) error

	RemoveBuiltInPackage(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
		address source.ManagedPackageAddress,
	) error

	EnsureBuiltInSourceCurrent(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
	) error
}

type builtinStore struct {
	api *API
}

func NewBuiltinStore(api *API) (BuiltinStore, error) {
	if api == nil {
		return nil, fmt.Errorf(
			"%w: Tool built-in Store requires an API",
			basespec.ErrInvalid,
		)
	}
	return &builtinStore{api: api}, nil
}

func (s *builtinStore) InstallBuiltInPackage(
	ctx context.Context,
	request BuiltInPackageInstallRequest,
) error {
	if s == nil || s.api == nil {
		return basespec.ErrClosed
	}
	return s.api.installBuiltInPackage(ctx, request)
}

func (s *builtinStore) RemoveBuiltInPackage(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
	address source.ManagedPackageAddress,
) error {
	if s == nil || s.api == nil {
		return basespec.ErrClosed
	}
	return s.api.removeBuiltInPackage(
		ctx,
		rootID,
		sourceID,
		address,
	)
}

func (s *builtinStore) EnsureBuiltInSourceCurrent(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
) error {
	if s == nil || s.api == nil {
		return basespec.ErrClosed
	}
	return s.api.ensureBuiltInSourceCurrent(
		ctx,
		rootID,
		sourceID,
	)
}
