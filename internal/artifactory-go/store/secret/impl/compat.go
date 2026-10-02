// Package localstate is a temporary compatibility facade for the current
// combined overlay and secret lifecycle implementation.
//
// Deprecated: this package will disappear after overlay, secret lifecycle,
// and artifact-cleanup behavior are split into their actual owners.
package localstate

import "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/internal"

type (
	ArtifactReader       = internal.ArtifactReader
	AttachBindingRequest = internal.AttachBindingRequest
	Repository           = internal.Repository
	Service              = internal.Service
)

var NewService = internal.NewService
