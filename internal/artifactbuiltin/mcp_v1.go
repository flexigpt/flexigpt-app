package artifactbuiltin

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"

	_ "embed"
)

//go:embed mcp-bundle-v1.schema.json
var BundleV1JSONSchema []byte

//go:embed mcp-server-v1.schema.json
var ServerV1JSONSchema []byte

//go:embed mcp-policy-v1.schema.json
var PolicyV1JSONSchema []byte

const (
	MCPHostName    = "FlexiGPT"
	MCPHostVersion = "dev"

	BundleKind basespec.CollectionKind = "mcp.bundle"
	ServerKind basespec.ArtifactKind   = "mcp.server"
	PolicyKind basespec.ArtifactKind   = "mcp.policy"

	BundleSchemaID basespec.SchemaID = "mcp.bundle.v1"
	ServerSchemaID basespec.SchemaID = "mcp.server.v1"
	PolicySchemaID basespec.SchemaID = "mcp.policy.v1"

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
	MCPBundleSchemaKey = providerapi.CollectionSchemaKey(
		BundleKind,
		BundleSchemaID,
		MCPSchemaVersion,
	)
	MCPServerSchemaKey = providerapi.ArtifactSchemaKey(
		ServerKind,
		ServerSchemaID,
		MCPSchemaVersion,
	)
	MCPPolicySchemaKey = providerapi.ArtifactSchemaKey(
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
