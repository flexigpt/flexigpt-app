// Package sourceimpl is a temporary compatibility facade.
//
// Deprecated: source driver, repository, runtime, and composition contracts
// will move to source-owned public packages in the next split.
package sourceimpl

import "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/internal"

type (
	Reader                    = internal.Reader
	Repository                = internal.Repository
	Snapshot                  = internal.Snapshot
	Opener                    = internal.Opener
	LocalPathResolver         = internal.LocalPathResolver
	LocalPathCapability       = internal.LocalPathCapability
	ManagedPackageWriter      = internal.ManagedPackageWriter
	ManagedSourceBootstrapper = internal.ManagedSourceBootstrapper
	ManagedRootRemover        = internal.ManagedRootRemover
	Adapter                   = internal.Adapter
	PackageBatchWriter        = internal.PackageBatchWriter
	Registry                  = internal.Registry
	Runtime                   = internal.Runtime
	LocalPathRuntime          = internal.LocalPathRuntime
	VerificationSession       = internal.VerificationSession
	Service                   = internal.Service
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
