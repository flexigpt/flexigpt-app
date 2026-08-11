package artifact

import (
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/mcp/schema"
)

const (
	DecoderID basespec.DecoderID = "mcp.bundle-json"

	DecoderRevision = "mcp.bundle.discovery.v1"
)

func ServerSubresource(
	name basespec.LogicalName,
) basespec.SubresourceLocator {
	return basespec.SubresourceLocator(
		path.Join("mcpServers", string(name)),
	)
}

func PolicySubresource(
	name basespec.LogicalName,
) basespec.SubresourceLocator {
	return basespec.SubresourceLocator(
		path.Join("policies", string(name)),
	)
}

func IsMCPKind(kind basespec.ArtifactKind) bool {
	return kind == schema.ServerKind || kind == schema.PolicyKind
}
