package compositionapi

import (
	"io/fs"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/overlay"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/secretapi"
)

// Config contains application-composition inputs for one Artifact Store.
//
// Providers are fully constructed before Open is called. RetainedRoots are
// both lifecycle-policy declarations and initial generic Root declarations.
//
// SecretValues ownership transfers to Artifact Store after successful Open.
// Implementations must make Close safe to call more than once.
type Config struct {
	BaseDirectory string

	EmbeddedProviders map[string]fs.FS

	Providers []providerapi.Provider

	ProtectedRootIDs []root.RootID
	RetainedRoots    []root.RootDraft

	ProtectedOverlayNamespaces []overlay.Namespace
	StoreOverlayNamespaces     []overlay.Namespace
	SecretValues               secretapi.ValueStore
}
