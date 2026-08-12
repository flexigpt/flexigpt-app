package overlay

import (
	"context"
	"encoding/json"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/mcp/server"
)

const settingsOverlayPrefix = "mcp.installation.v1/"

// SettingsValueStore is the narrow application-settings port required by MCP
// installation overlays. The application adapter must persist values through
// the existing Setting Store and implement compare-and-swap atomically.
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

// SettingsPrefixValueStore is optional. It is used by protected hydration to
// remove stale installation overlays for a reset protected Root.
type SettingsPrefixValueStore interface {
	SettingsValueStore

	DeleteMCPInstallationPrefix(
		ctx context.Context,
		prefix string,
	) error
}

// ServerOverlay intentionally stores ServerData as a named nested value.
// Anonymous embedding would produce two schemaVersion fields at JSON encoding
// time and causes the embedded ServerData schemaVersion to decode as empty.
type ServerOverlay struct {
	SchemaVersion  string            `json:"schemaVersion"`
	Revision       uint64            `json:"revision"`
	RuntimeEnabled bool              `json:"runtimeEnabled"`
	ServerData     server.ServerData `json:"serverData"`
}

type BundleOverlay struct {
	SchemaVersion  string `json:"schemaVersion"`
	Revision       uint64 `json:"revision"`
	RuntimeEnabled bool   `json:"runtimeEnabled"`
}

type OverlayRepository interface {
	GetServerOverlay(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) (ServerOverlay, bool, error)

	GetBundleOverlay(
		ctx context.Context,
		rootID basespec.RootID,
		collectionID basespec.CollectionID,
	) (BundleOverlay, bool, error)

	PutServerOverlay(
		ctx context.Context,
		ref artifact.ArtifactRef,
		expectedRevision uint64,
		value ServerOverlay,
	) error

	PutBundleOverlay(
		ctx context.Context,
		rootID basespec.RootID,
		collectionID basespec.CollectionID,
		expectedRevision uint64,
		value BundleOverlay,
	) error

	DeleteServerOverlay(
		ctx context.Context,
		ref artifact.ArtifactRef,
		expectedRevision uint64,
	) error

	DeleteBundleOverlay(
		ctx context.Context,
		rootID basespec.RootID,
		collectionID basespec.CollectionID,
		expectedRevision uint64,
	) error
}
