package refreshimpl

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/api/installerapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/api/installerapi/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/source"
)

func (s *Service) RegisterCompiledDocuments(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
	packages []topology.CompiledPackage,
) error {
	if s == nil || s.discovery == nil {
		return model.ErrClosed
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
