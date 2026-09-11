package mcpbundlev1

import (
	_ "embed"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/mcppolicyv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/mcpserverv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	MCPBundleKind          = "mcp.bundle"
	MCPBundleSchemaID      = "mcp.bundle.v1"
	MCPBundleSchemaVersion = "v1"
	MCPBundleSchemaURL     = "https://schemas.flexigpt.dev/mcp/bundle/v1.json"
)

//go:embed mcp-bundle-v1.schema.json
var schemaJSON []byte

var compiledMCPBundleSchema = jsonutil.MustCompileJSONSchema(
	schemaJSON,
)

var MCPBundleSchemaKey = schema.CollectionKey(
	collection.CollectionKind(MCPBundleKind),
	schema.SchemaID(MCPBundleSchemaID),
	MCPBundleSchemaVersion,
)

func MCPBundleJSONSchema() []byte {
	return append([]byte(nil), schemaJSON...)
}

type MCPBundleDocument struct {
	Digest          *string                                    `json:"digest,omitempty"`
	Kind            string                                     `json:"kind"`
	SchemaID        string                                     `json:"schemaID"`
	SchemaVersion   string                                     `json:"schemaVersion"`
	LogicalName     string                                     `json:"logicalName"`
	LogicalVersion  string                                     `json:"logicalVersion,omitempty"`
	DisplayName     string                                     `json:"displayName,omitempty"`
	Description     string                                     `json:"description,omitempty"`
	Labels          map[string]string                          `json:"labels,omitempty"`
	MCPServers      map[string]mcpserverv1.MCPServerCoreServer `json:"mcpServers"`
	BundleExtension *MCPBundleExtension                        `json:"bundleExtension,omitempty"`
}

type MCPBundleExtension struct {
	Servers  map[string]mcpserverv1.MCPServerExtension `json:"servers,omitempty"`
	Policies map[string]mcppolicyv1.MCPPolicyDocument  `json:"policies,omitempty"`
}

func DecodeMCPBundleJSON(raw []byte) (MCPBundleDocument, error) {
	return jsonutil.DecodeCanonicalObject[MCPBundleDocument](
		raw,
		basespec.MaxDefinitionBytes,
	)
}

func (v MCPBundleDocument) Clone() (MCPBundleDocument, error) {
	return jsonutil.CloneJSON(v)
}

func (v MCPBundleDocument) Canonicalize() (MCPBundleDocument, error) {
	return jsonutil.CloneJSON(v)
}

func (v MCPBundleDocument) CanonicalJSON() ([]byte, error) {
	return jsonutil.MarshalCanonicalObject(v, basespec.MaxDefinitionBytes)
}

func (v MCPBundleDocument) CalculatedDigest() (
	cryptoutil.Digest,
	error,
) {
	payload := v
	payload.Digest = nil
	return cryptoutil.CanonicalDigest(payload)
}

func (v MCPBundleDocument) Validate() error {
	if err := jsonutil.ValidateJSONSchema(
		compiledMCPBundleSchema,
		v,
		basespec.MaxDefinitionBytes,
	); err != nil {
		return fmt.Errorf("MCP Bundle schema: %w", err)
	}
	if err := basespec.ValidatePortableMetadata(
		basespec.LogicalName(v.LogicalName),
		basespec.LogicalVersion(v.LogicalVersion),
		v.DisplayName,
		v.Description,
		v.Labels,
		v.Digest,
	); err != nil {
		return err
	}

	if v.BundleExtension == nil {
		return nil
	}
	for name, policy := range v.BundleExtension.Policies {
		if err := policy.Validate(); err != nil {
			return fmt.Errorf("MCP Bundle policy %q: %w", name, err)
		}
	}
	return nil
}
