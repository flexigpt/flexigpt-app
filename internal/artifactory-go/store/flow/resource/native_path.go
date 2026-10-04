package resource

import (
	"context"

	resourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// NativePathAPI is a separately supplied trusted runtime capability. A native
// filesystem path is not part of ordinary portable resource access.
type NativePathAPI interface {
	ResolveVerifiedLocalPath(
		ctx context.Context,
		resolved resourceModel.ResolvedArtifact,
		localLocator spec.Locator,
	) (string, error)
	SupportsLocalPath(kind sourceModel.SourceKind) bool
}
