package artifactimpl

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/internal"
)

type (
	Synchronization = internal.Synchronization
	Service         = internal.Service
	Synchronizer    = internal.Synchronizer
)

var (
	NewService      = internal.NewService
	NewSynchronizer = internal.NewSynchronizer
)
