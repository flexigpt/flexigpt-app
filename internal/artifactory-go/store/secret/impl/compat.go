package secretimpl

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/internal"
)

// Deprecated: this compatibility façade disappears when Overlay, Secret, and
// ArtifactCleanup implementations are physically split.
type (
	ArtifactReader = internal.ArtifactReader
	Repository     = internal.Repository
	Service        = internal.Service
)

var NewService = internal.NewService
