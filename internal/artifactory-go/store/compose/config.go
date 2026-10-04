package compose

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema"
	artifactcleanupFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/artifactcleanup"
	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	refreshFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay"
	overlayModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/value"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/driver"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
)

// ShutdownFunc closes deployment-owned provider resources after all business
// services have stopped being used. A deployment transfers this responsibility
// only after Open succeeds; on an Open failure it retains ownership and must
// clean up locally.
type ShutdownFunc func() error

// Config supplies the exact contracts required to construct one coherent
// Artifact Store aggregate. Repositories are deliberately named by their
// entity or flow responsibility: there is no metadata or local-state
// aggregate dependency.
type Config struct {
	RootRepository       root.Repository
	SourceRepository     source.Repository
	ArtifactRepository   artifact.Repository
	CatalogRepository    catalog.Repository
	DefinitionRepository definition.Repository

	RefreshStates    refreshFlow.StateReader
	RefreshPublisher refreshFlow.Repository

	OverlayRepository         overlay.Repository
	SecretBindingRepository   secret.BindingRepository
	SecretRuntimeRepository   secret.RuntimeRepository
	SecretLifecycleRepository secret.LifecycleRepository
	ArtifactCleanupRepository artifactcleanupFlow.Repository
	InstallationRepository    installFlow.Repository

	SourceDrivers []driver.Driver
	SchemaFactory schema.Factory
	SchemaCodecs  []schema.Codec
	Decoders      []ingest.Decoder

	Clock              clockutil.Clock
	ArtifactIDProvider artifact.IDProvider
	RootPolicy         root.Policy

	ProtectedOverlayNamespaces []overlayModel.Namespace
	StoreOverlayNamespaces     []overlayModel.Namespace
	SecretValues               value.ValueStore

	// Shutdown is invoked exactly once by Store.Close after successful
	// assembly. It is normally supplied by deployment composition and must
	// close only resources whose ownership was explicitly transferred.
	Shutdown ShutdownFunc
}
