package local

import (
	"io/fs"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider"
	overlay "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay/model"
	root "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/valuestore"
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

	Providers []provider.Provider

	ProtectedRootIDs []root.RootID
	RetainedRoots    []root.RootDraft

	ProtectedOverlayNamespaces []overlay.Namespace
	StoreOverlayNamespaces     []overlay.Namespace
	SecretValues               valuestore.ValueStore
}
