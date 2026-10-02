// Package artifactimpl is a temporary compatibility facade.
//
// Deprecated: use Artifact public contracts and Artifact composition
// capabilities after the repository/API split.
package artifactimpl

import "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/internal"

type (
	SourceStateUpdate = internal.SourceStateUpdate
	Synchronization   = internal.Synchronization
	Reader            = internal.Reader
	CatalogReader     = internal.CatalogReader
	Repository        = internal.Repository
	DefinitionReader  = internal.DefinitionReader
	Service           = internal.Service
	Synchronizer      = internal.Synchronizer
)

var (
	NewService      = internal.NewService
	NewSynchronizer = internal.NewSynchronizer
)
