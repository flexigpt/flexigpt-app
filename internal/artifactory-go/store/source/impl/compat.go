package sourceimpl

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/driver"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/internal"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage"
)

// Package sourceimpl is a temporary compatibility facade.
//
// Deprecated: import source, source/driver, and source/managedpackage public
// contracts directly. This package disappears in Phase 3.
type (
	Reader                    = source.Reader
	Repository                = source.Repository
	Runtime                   = source.Runtime
	LocalPathRuntime          = source.LocalPathRuntime
	Snapshot                  = driver.Snapshot
	Opener                    = driver.Opener
	LocalPathResolver         = driver.LocalPathResolver
	LocalPathCapability       = driver.LocalPathCapability
	ManagedPackageWriter      = managedpackage.Writer
	PackageBatchWriter        = managedpackage.BatchWriter
	ManagedSourceBootstrapper = driver.ManagedSourceBootstrapper
	ManagedRootRemover        = driver.ManagedRootRemover
	Adapter                   = driver.Driver

	Registry            = internal.Registry
	VerificationSession = internal.VerificationSession
	Service             = internal.Service
)

var (
	NewRegistry                 = internal.NewRegistry
	NewRuntime                  = internal.NewRuntime
	ReadSnapshotEntry           = internal.ReadSnapshotEntry
	ReadVerifiedSnapshotEntry   = internal.ReadVerifiedSnapshotEntry
	VerifySnapshotContentDigest = internal.VerifySnapshotContentDigest
	NewVerificationSession      = internal.NewVerificationSession
	ResolveVerifiedLocalPath    = internal.ResolveVerifiedLocalPath
	NewService                  = internal.NewService
)
