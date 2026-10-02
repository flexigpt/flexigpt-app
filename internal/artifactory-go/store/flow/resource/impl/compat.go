// Package resource is a temporary compatibility facade.
//
// Deprecated: use Resource public contracts and Resource compose
// capabilities after cross-entity source runtime dependencies are narrowed.
package resource

import "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource/internal"

type Service = internal.Service

var NewService = internal.NewService
