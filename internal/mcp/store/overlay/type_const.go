package overlay

import (
	"context"
	"encoding/json"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	mcpDomainServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/server"
)

const settingsOverlayPrefix = "mcp.installation.v1/"

type SettingsValueStore interface {
	GetMCPInstallationValue(
		ctx context.Context,
		key string,
	) (json.RawMessage, bool, error)

	PutMCPInstallationValue(
		ctx context.Context,
		key string,
		expectedRevision uint64,
		value json.RawMessage,
	) error

	DeleteMCPInstallationValue(
		ctx context.Context,
		key string,
		expectedRevision uint64,
	) error
}

type SettingsPrefixValueStore interface {
	SettingsValueStore

	DeleteMCPInstallationPrefix(
		ctx context.Context,
		prefix string,
	) error
}

type ServerOverlay struct {
	SchemaVersion  string                     `json:"schemaVersion"`
	Revision       uint64                     `json:"revision"`
	RuntimeEnabled bool                       `json:"runtimeEnabled"`
	ServerData     mcpDomainServer.ServerData `json:"serverData"`
}

type OverlayRepository interface {
	GetServerOverlay(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) (ServerOverlay, bool, error)

	PutServerOverlay(
		ctx context.Context,
		ref artifact.ArtifactRef,
		expectedRevision uint64,
		value ServerOverlay,
	) error

	DeleteServerOverlay(
		ctx context.Context,
		ref artifact.ArtifactRef,
		expectedRevision uint64,
	) error
}

type RootPurger interface {
	PurgeRoot(
		ctx context.Context,
		rootID root.RootID,
	) error
}
