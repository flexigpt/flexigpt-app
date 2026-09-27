package compositionapi

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi/topology"
)

func (s *Store) RegisterCompiledPackages(
	ctx context.Context,
	values []topology.CompiledRegistration,
) error {
	if s == nil || s.components == nil {
		return basespec.ErrClosed
	}
	return s.components.RegisterCompiledPackages(ctx, values)
}

func (s *Store) HydrateCompiledPackages(
	ctx context.Context,
	plans []topology.CompiledPackagePlan,
) error {
	if s == nil || s.components == nil {
		return basespec.ErrClosed
	}
	return s.components.HydrateCompiledPackages(ctx, plans)
}
