package internal

import (
	sourceStore "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/driver"
	managedpackage "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage"
)

// These aliases intentionally keep current source implementation files small
// during the contract extraction. The actual contracts now belong to source,
// source/driver, and source/managedpackage.
type (
	Reader                    = sourceStore.Reader
	Repository                = sourceStore.Repository
	Runtime                   = sourceStore.Runtime
	LocalPathRuntime          = sourceStore.LocalPathRuntime
	Snapshot                  = driver.Snapshot
	Opener                    = driver.Opener
	LocalPathResolver         = driver.LocalPathResolver
	LocalPathCapability       = driver.LocalPathCapability
	ManagedSourceBootstrapper = driver.ManagedSourceBootstrapper
	ManagedRootRemover        = driver.ManagedRootRemover
	Adapter                   = driver.Driver
	ManagedPackageWriter      = managedpackage.Writer
)
