package resource

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	refreshModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource/internal"
	resourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

type refreshInspector interface {
	InspectSourceMetadata(ctx context.Context, s sourceModel.Source) (refreshModel.Inspection, error)
}

type (
	ordinaryService   struct{ service *internal.Service }
	nativePathService struct{ service *internal.Service }
)

// NewService constructs separately typed ordinary and trusted native-path
// capabilities over one private verification implementation.
func NewService(
	artifacts artifact.Reader,
	definitions artifact.DefinitionReader,
	refresh refreshInspector,
	sources source.Runtime,
) (API, NativePathAPI, error) {
	if artifacts == nil || definitions == nil || refresh == nil || sources == nil {
		return nil, nil, fmt.Errorf("%w: Artifact resource service dependencies are incomplete", spec.ErrInvalid)
	}
	service, err := internal.NewService(artifacts, definitions, refresh, sources)
	if err != nil {
		return nil, nil, err
	}
	return &ordinaryService{service: service}, &nativePathService{service: service}, nil
}

func (s *ordinaryService) ResolveArtifact(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	options resourceModel.ResolveOptions,
) (resourceModel.ResolvedArtifact, error) {
	return s.service.ResolveArtifact(ctx, ref, options)
}

func (s *ordinaryService) ReadSourceEntry(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	locator spec.Locator,
	maximumBytes int64,
) (resourceModel.VerifiedEntry, error) {
	return s.service.ReadSourceEntry(ctx, rootID, sourceID, locator, maximumBytes)
}

func (s *ordinaryService) StatSourceEntry(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	locator spec.Locator,
) (sourceModel.Entry, error) {
	return s.service.StatSourceEntry(ctx, rootID, sourceID, locator)
}

func (s *ordinaryService) ReadSourceTree(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	base spec.Locator,
	include, exclude []string,
	maximumEntries int,
	maximumBytes int64,
) ([]resourceModel.VerifiedEntry, error) {
	return s.service.ReadSourceTree(ctx, rootID, sourceID, base, include, exclude, maximumEntries, maximumBytes)
}

func (s *ordinaryService) BeginVerificationSession(
	ctx context.Context,
) (context.Context, resourceModel.VerificationSession, error) {
	return s.service.BeginVerificationSession(ctx)
}

func (s *nativePathService) ResolveVerifiedLocalPath(
	ctx context.Context,
	resolved resourceModel.ResolvedArtifact,
	locator spec.Locator,
) (string, error) {
	return s.service.ResolveVerifiedLocalPath(ctx, resolved, locator)
}

func (s *nativePathService) SupportsLocalPath(kind sourceModel.SourceKind) bool {
	return s.service.SupportsLocalPath(kind)
}
