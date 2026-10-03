package internal

import (
	"context"

	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func (s *Service) RegisterCompiledDocuments(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	packages []installModel.CompiledPackage,
) error {
	if s == nil || s.discovery == nil {
		return spec.ErrClosed
	}
	if err := installFlow.RequirePrivileged(ctx); err != nil {
		return err
	}
	return s.discovery.RegisterCompiledDocuments(
		rootID,
		sourceID,
		packages,
	)
}
