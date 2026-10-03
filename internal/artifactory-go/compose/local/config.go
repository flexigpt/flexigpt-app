package local

import (
	"io/fs"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema"
	overlayModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/value"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest"
)

type Config struct {
	BaseDirectory string

	EmbeddedProviders map[string]fs.FS

	// SchemaCodecs and Decoders are explicit registrations. Artifact Store
	// deliberately has no generic artifact-provider descriptor.
	SchemaCodecs []schema.Codec
	Decoders     []ingest.Decoder

	ProtectedRootIDs []rootModel.RootID
	RetainedRoots    []rootModel.RootDraft

	ProtectedOverlayNamespaces []overlayModel.Namespace
	StoreOverlayNamespaces     []overlayModel.Namespace
	SecretValues               value.ValueStore
}
