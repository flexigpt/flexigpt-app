package artifactbuiltin

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
)

const (
	MCPHostName    = "FlexiGPT"
	MCPHostVersion = "dev"

	BundleKind collection.CollectionKind = "mcp.bundle"
	ServerKind artifact.ArtifactKind     = "mcp.server"
	PolicyKind artifact.ArtifactKind     = "mcp.policy"

	BundleSchemaID schema.SchemaID = "mcp.bundle.v1"
	ServerSchemaID schema.SchemaID = "mcp.server.v1"
	PolicySchemaID schema.SchemaID = "mcp.policy.v1"

	MCPBuiltInInstallerName = "mcp.bundle"

	MCPBundleHydrationFingerprintSchemaVersion = "mcp.builtin-hydration/v1"

	MCPSchemaVersion = "v1"

	BundleSchemaURL = "https://schemas.flexigpt.dev/mcp/bundle/v1.json"
	ServerSchemaURL = "https://schemas.flexigpt.dev/mcp/server/v1.json"
	PolicySchemaURL = "https://schemas.flexigpt.dev/mcp/policy/v1.json"

	MCPServerSubresourceDirectory basespec.SubresourceLocator = "mcpServers"
	MCPPolicySubresourceDirectory basespec.SubresourceLocator = "policies"

	DecoderRevision                    = "mcp.bundle.discovery.v1"
	DecoderID       basespec.DecoderID = "mcp.bundle-json"
)

var (
	MCPBundleSchemaKey = schema.CollectionKey(
		BundleKind,
		BundleSchemaID,
		MCPSchemaVersion,
	)
	MCPServerSchemaKey = schema.ArtifactKey(
		ServerKind,
		ServerSchemaID,
		MCPSchemaVersion,
	)
	MCPPolicySchemaKey = schema.ArtifactKey(
		PolicyKind,
		PolicySchemaID,
		MCPSchemaVersion,
	)
)

func CheckCodecContext(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf(
			"%w: MCP schema codec context is nil",
			basespec.ErrInvalid,
		)
	}
	return ctx.Err()
}
