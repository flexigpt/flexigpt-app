package compositionapi

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/api/installerapi/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec"
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
