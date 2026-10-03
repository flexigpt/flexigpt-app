package resource

import (
	"context"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	resourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

type API interface {
	ResolveArtifact(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
		options resourceModel.ResolveOptions,
	) (resourceModel.ResolvedArtifact, error)

	ResolveVerifiedLocalPath(
		ctx context.Context,
		resolved resourceModel.ResolvedArtifact,
		localLocator spec.Locator,
	) (string, error)

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
		include []string,
		exclude []string,
		maximumEntries int,
		maximumBytes int64,
	) ([]resourceModel.VerifiedEntry, error)

	SupportsLocalPath(kind sourceModel.SourceKind) bool
}
