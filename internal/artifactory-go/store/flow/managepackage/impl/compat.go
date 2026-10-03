// Package managedpackageimpl is a temporary compatibility facade.
//
// Deprecated: use ManagedPackage public contracts and composition
// capabilities after source mutation ports are extracted.
package managedpackageimpl

import "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managepackage/internal"

type (
	SourceState               = internal.SourceState
	GetSourceStateFunc        = internal.GetSourceStateFunc
	PublishPackageFunc        = internal.PublishPackageFunc
	RemovePackageFunc         = internal.RemovePackageFunc
	PruneDiscoveryLocatorFunc = internal.PruneDiscoveryLocatorFunc
	ArtifactCommands          = internal.ArtifactCommands
	SourceRunner              = internal.SourceRunner
	Dependencies              = internal.Dependencies
	Service                   = internal.Service
)

var NewService = internal.NewService
