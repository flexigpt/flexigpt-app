package overlay

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/compose/local"
	artifact "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	overlay "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain"
	mcpDomainServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/server"
)

const (
	InstallationNamespace   overlay.Namespace = "mcp.installation"
	GlobalSettingsNamespace overlay.Namespace = "mcp.global"
)

func Namespaces() []overlay.Namespace {
	return []overlay.Namespace{
		InstallationNamespace,
	}
}

func StoreNamespaces() []overlay.Namespace {
	return []overlay.Namespace{
		GlobalSettingsNamespace,
	}
}

// ServerOverlay stores only non-secret protected MCP installation state.
//
// ServerData may contain MCP logical secret selectors such as `mcpv1:...`,
// but actual Artifact Store secret refs and values remain outside this payload.
type ServerOverlay struct {
	SchemaVersion string                     `json:"schemaVersion"`
	Revision      uint64                     `json:"revision"`
	ServerData    mcpDomainServer.ServerData `json:"serverData"`
}

func (value ServerOverlay) Validate() error {
	if value.SchemaVersion != mcpDomain.InstallationDataSchemaVersion {
		return fmt.Errorf(
			"%w: unsupported MCP server overlay schema %q",
			spec.ErrInvalid,
			value.SchemaVersion,
		)
	}
	if value.Revision == 0 {
		return fmt.Errorf(
			"%w: MCP server overlay revision is required",
			spec.ErrInvalid,
		)
	}
	return value.ServerData.Validate()
}

func (value ServerOverlay) Clone() ServerOverlay {
	output := value
	output.ServerData = value.ServerData.Clone()
	return output
}

type OverlayRepository interface {
	GetServerOverlay(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) (ServerOverlay, bool, error)

	PutServerOverlay(
		ctx context.Context,
		ref artifact.ArtifactRef,
		expectedArtifactRevision uint64,
		expectedOverlayRevision uint64,
		value ServerOverlay,
	) error

	DeleteServerOverlay(
		ctx context.Context,
		ref artifact.ArtifactRef,
		expectedArtifactRevision uint64,
		expectedOverlayRevision uint64,
	) error

	// PurgeServerLocalState removes protected overlays and all Artifact Store
	// secret bindings for a removed MCP server. It is used by managed package
	// deletion and built-in compiled package lifecycle cleanup.
	PurgeServerLocalState(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) error
}

type ArtifactOverlayDependencies struct {
	Artifacts        local.ArtifactAPI
	Protection       local.ProtectionAPI
	ProtectedOverlay local.ProtectedOverlayAPI
	LocalState       local.LocalStateMaintenanceAPI
}
