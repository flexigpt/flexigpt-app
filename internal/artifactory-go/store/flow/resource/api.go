package resource

import (
	"context"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	resourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// API is the ordinary portable resource capability. Native filesystem paths
// are intentionally absent and require NativePathAPI.
type API interface {
	ResolveArtifact(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
		options resourceModel.ResolveOptions,
	) (resourceModel.ResolvedArtifact, error)
	ReadSourceEntry(
		ctx context.Context,
		rootID rootModel.RootID,
		sourceID sourceModel.SourceID,
		locator spec.Locator,
		maximumBytes int64,
	) (resourceModel.VerifiedEntry, error)
	StatSourceEntry(
		ctx context.Context,
		rootID rootModel.RootID,
		sourceID sourceModel.SourceID,
		locator spec.Locator,
	) (sourceModel.Entry, error)
	ReadSourceTree(
		ctx context.Context,
		rootID rootModel.RootID,
		sourceID sourceModel.SourceID,
		base spec.Locator,
		include, exclude []string,
		maximumEntries int,
		maximumBytes int64,
	) ([]resourceModel.VerifiedEntry, error)
}
