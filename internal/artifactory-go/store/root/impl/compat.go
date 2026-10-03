// Package rootimpl is a temporary compatibility facade.
//
// Deprecated: new Root composition must eventually depend on Root public
// contracts and Root compose capabilities rather than this package.
package rootimpl

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/internal"
)

type (
	Service       = internal.Service
	SetRootPolicy = internal.SetRootPolicy
)

var (
	NewService         = internal.NewService
	NewSetRootPolicy   = internal.NewSetRootPolicy
	RequireMutableRoot = internal.RequireMutableRoot
)
