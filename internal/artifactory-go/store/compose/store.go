package compose

import (
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema"
	artifactcleanupFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/artifactcleanup"
	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	managepackageFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managepackage"
	refreshFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh"
	resourceFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
)

// Store is the sole assembled Artifact Store capability surface. Its fields
// are named entity and flow APIs; it neither exposes provider repositories nor
// forwards through a second local deployment handle.
type Store struct {
	Roots       root.API
	Sources     source.API
	Artifacts   artifact.API
	Catalog     catalog.API
	Definitions definition.API
	Schemas     schema.API

	Refresh                refreshFlow.API
	Resources              resourceFlow.API
	TrustedNativeResources resourceFlow.NativePathAPI
	ManagedPackages        managepackageFlow.API
	Topology               installFlow.API
	ArtifactCleanup        artifactcleanupFlow.API

	ProtectedOverlays overlay.API
	StoreOverlays     overlay.StoreAPI
	SecretBindings    secret.API
	SecretRuntime     secret.RuntimeAPI
	SecretLifecycle   secret.LifecycleAPI
	Protection        root.ProtectionAPI

	shutdown  ShutdownFunc
	closeOnce sync.Once
	closeErr  error
}
