package source

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/driver"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage"
)

// Owner-local aliases keep Source implementation details expressed through
// the public Source and driver contracts without recreating an impl facade.
type (
	Adapter                   = driver.Driver
	Snapshot                  = driver.Snapshot
	Opener                    = driver.Opener
	LocalPathResolver         = driver.LocalPathResolver
	LocalPathCapability       = driver.LocalPathCapability
	ManagedSourceBootstrapper = driver.ManagedSourceBootstrapper
	ManagedRootRemover        = driver.ManagedRootRemover
	ManagedPackageWriter      = managedpackage.Writer
)
