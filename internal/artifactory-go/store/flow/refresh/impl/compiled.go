package refreshimpl

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/topology"
	root "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	source "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func (s *Service) RegisterCompiledDocuments(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
	packages []topology.CompiledPackage,
) error {
	if s == nil || s.discovery == nil {
		return spec.ErrClosed
	}
	if err := install.RequirePrivileged(ctx); err != nil {
		return err
	}
	return s.discovery.RegisterCompiledDocuments(
		rootID,
		sourceID,
		packages,
	)
}
