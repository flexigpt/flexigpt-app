package refreshimpl

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/api/installerapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/api/installerapi/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/basespec/source"
)

func (s *Service) RegisterCompiledDocuments(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
	packages []topology.CompiledPackage,
) error {
	if s == nil || s.discovery == nil {
		return basespec.ErrClosed
	}
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return err
	}
	return s.discovery.RegisterCompiledDocuments(
		rootID,
		sourceID,
		packages,
	)
}
