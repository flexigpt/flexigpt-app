package secretimpl

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/internal"
)

// Deprecated: this compatibility façade disappears in Phase 3.
type (
	ArtifactReader = internal.ArtifactReader
	Repository     = internal.Repository
	Service        = internal.Service
)

var NewService = internal.NewService
