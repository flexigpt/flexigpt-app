// Package refreshimpl is a temporary compatibility facade.
//
// Deprecated: use Refresh public contracts and Refresh compose capabilities
// after refresh persistence and publication values are extracted.
package refreshimpl

import "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh/internal"

type (
	RefreshStateReader = internal.RefreshStateReader
	ArtifactReader     = internal.ArtifactReader
	Publication        = internal.Publication
	Publisher          = internal.Publisher
	Service            = internal.Service
)

var NewService = internal.NewService
