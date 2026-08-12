package schema

import (
	"context"
	"fmt"
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/shareable"

	_ "embed"
)

//go:embed mcp-bundle-v1.schema.json
var BundleV1JSONSchema []byte

//go:embed mcp-server-v1.schema.json
var ServerV1JSONSchema []byte

//go:embed mcp-policy-v1.schema.json
var PolicyV1JSONSchema []byte

const (
	BundleKind basespec.CollectionKind = "mcp.bundle"
	ServerKind basespec.ArtifactKind   = "mcp.server"
	PolicyKind basespec.ArtifactKind   = "mcp.policy"

	BundleSchemaID basespec.SchemaID = "mcp.bundle.v1"
	ServerSchemaID basespec.SchemaID = "mcp.server.v1"
	PolicySchemaID basespec.SchemaID = "mcp.policy.v1"

	HydrationFingerprintSchemaVersion = "mcp.builtin-hydration/v1"

	MCPSchemaVersion = "v1"

	BundleSchemaURL = "https://schemas.flexigpt.dev/mcp/bundle/v1.json"
	ServerSchemaURL = "https://schemas.flexigpt.dev/mcp/server/v1.json"
	PolicySchemaURL = "https://schemas.flexigpt.dev/mcp/policy/v1.json"

	// BundleFileName is the preferred filename for newly authored managed
	// MCP Bundle packages. Existing source-controlled package registrations
	// can explicitly select one of the accepted compatibility filenames.
	BundleFileName          = "mcps.json"
	AlternateBundleFileName = "mcp.json"
	LegacyBundleFileName    = ".mcp.json"

	DecoderRevision                    = "mcp.bundle.discovery.v1"
	DecoderID       basespec.DecoderID = "mcp.bundle-json"
)

var bundleDocumentFileNames = [...]string{
	BundleFileName,
	AlternateBundleFileName,
	LegacyBundleFileName,
}

var MCPSchemaKey = shareable.SchemaKey{
	Entity:        shareable.EntityCollection,
	Kind:          BundleKind,
	SchemaID:      BundleSchemaID,
	SchemaVersion: MCPSchemaVersion,
}

func BundleDocumentFileNames() []string {
	return append([]string(nil), bundleDocumentFileNames[:]...)
}

func IsBundleDocumentFileName(value string) bool {
	for _, candidate := range bundleDocumentFileNames {
		if value == candidate {
			return true
		}
	}
	return false
}

func IsBundleDocumentLocator(value basespec.Locator) bool {
	return IsBundleDocumentFileName(path.Base(string(value)))
}

func CheckCodecContext(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf(
			"%w: MCP schema codec context is nil",
			basespec.ErrInvalid,
		)
	}
	return ctx.Err()
}
