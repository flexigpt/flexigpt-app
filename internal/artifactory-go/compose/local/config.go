package local

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/fsdir"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/iofs"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema"
	overlayModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/value"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/driver"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
)

// Config declares local-deployment concerns and application registrations.
// Generic service assembly receives the resulting explicit contracts rather
// than importing local providers.
type Config struct {
	BaseDirectory string

	// EmbeddedProviders explicitly selects mutable or immutable provider
	// behavior for every application-supplied embedded filesystem.
	EmbeddedProviders         map[string]iofs.ProviderRegistration
	AdditionalSourceDrivers   []driver.Driver
	FilesystemTraversalPolicy *fsdir.TraversalPolicy

	// SchemaCodecs and Decoders are explicit registrations. Artifact Store
	// deliberately has no generic artifact-provider descriptor.
	SchemaCodecs []schema.Codec
	Decoders     []ingest.Decoder

	Clock              clockutil.Clock
	ArtifactIDProvider artifact.IDProvider

	ProtectedRootIDs []rootModel.RootID
	RetainedRoots    []rootModel.RootDraft

	ProtectedOverlayNamespaces []overlayModel.Namespace
	StoreOverlayNamespaces     []overlayModel.Namespace

	// SecretValues stays caller-owned if Open fails. Its ownership transfers to
	// the returned generic Store only after successful local opening, and that
	// handle closes it exactly once.
	SecretValues value.ValueStore
}
