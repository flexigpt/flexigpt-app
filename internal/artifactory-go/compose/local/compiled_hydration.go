package local

import (
	"context"

	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func (s *Store) RegisterCompiledPackages(
	ctx context.Context,
	values []installModel.CompiledRegistration,
) error {
	if s == nil || s.components == nil {
		return spec.ErrClosed
	}
	return s.components.RegisterCompiledPackages(ctx, values)
}

func (s *Store) HydrateCompiledPackages(
	ctx context.Context,
	plans []installModel.CompiledPackagePlan,
) error {
	if s == nil || s.components == nil {
		return spec.ErrClosed
	}
	return s.components.HydrateCompiledPackages(ctx, plans)
}
