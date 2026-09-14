package compositionapi

import (
	"io/fs"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
)

// Config contains application-composition inputs for one Artifact Store.
//
// Providers are fully constructed before Open is called. RetainedRoots are
// both lifecycle-policy declarations and initial generic Root declarations.
type Config struct {
	BaseDirectory string

	EmbeddedProviders map[string]fs.FS

	Providers []providerapi.Provider

	ProtectedRootIDs []root.RootID
	RetainedRoots    []root.RootDraft
}
