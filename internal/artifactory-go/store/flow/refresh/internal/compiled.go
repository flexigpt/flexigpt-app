package internal

import (
	"context"

	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	install "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	root "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	source "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func (s *Service) RegisterCompiledDocuments(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
	packages []install.CompiledPackage,
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
