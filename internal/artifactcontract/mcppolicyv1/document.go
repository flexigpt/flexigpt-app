package mcppolicyv1

import (
	_ "embed"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	MCPPolicyKind          = "mcp.policy"
	MCPPolicySchemaID      = "mcp.policy.v1"
	MCPPolicySchemaVersion = "v1"
	MCPPolicySchemaURL     = "https://schemas.flexigpt.dev/mcp/policy/v1.json"
)

//go:embed mcp-policy-v1.schema.json
var schemaJSON []byte

var compiledMCPPolicySchema = jsonutil.MustCompileJSONSchema(
	schemaJSON,
)

var MCPPolicySchemaKey = schema.ArtifactKey(
	artifact.ArtifactKind(MCPPolicyKind),
	schema.SchemaID(MCPPolicySchemaID),
	MCPPolicySchemaVersion,
)

func MCPPolicyJSONSchema() []byte {
	return append([]byte(nil), schemaJSON...)
}

type MCPPolicyDocument struct {
	Digest         *string           `json:"digest,omitempty"`
	Kind           string            `json:"kind"`
	SchemaID       string            `json:"schemaID"`
	SchemaVersion  string            `json:"schemaVersion"`
	LogicalName    string            `json:"logicalName"`
	LogicalVersion string            `json:"logicalVersion,omitempty"`
	DisplayName    string            `json:"displayName,omitempty"`
	Description    string            `json:"description,omitempty"`
	Labels         map[string]string `json:"labels,omitempty"`
	Body           MCPPolicyBody     `json:"body"`
}

type MCPPolicyBody struct {
	TrustLevel    string                                 `json:"trustLevel,omitempty"`
	DefaultPolicy *MCPPolicyDefaultPolicy                `json:"defaultPolicy,omitempty"`
	ToolPolicies  map[string]MCPPolicyToolPolicyOverride `json:"toolPolicies,omitempty"`
	AppsPolicy    *MCPPolicyAppsPolicy                   `json:"appsPolicy,omitempty"`
}

type MCPPolicyToolPolicyOverride struct {
	ToolName         string `json:"toolName,omitempty"`
	ApprovalRule     string `json:"approvalRule,omitempty"`
	ExecutionMode    string `json:"executionMode,omitempty"`
	AllowStaleDigest *bool  `json:"allowStaleDigest,omitempty"`
	ExpectedDigest   string `json:"expectedDigest,omitempty"`
}

type MCPPolicyDefaultPolicy struct {
	DefaultApprovalRule           string `json:"defaultApprovalRule,omitempty"`
	DefaultExecutionMode          string `json:"defaultExecutionMode,omitempty"`
	RequireApprovalForUnknownRisk *bool  `json:"requireApprovalForUnknownRisk,omitempty"`
	RequireApprovalForWrite       *bool  `json:"requireApprovalForWrite,omitempty"`
	RequireApprovalForDestructive *bool  `json:"requireApprovalForDestructive,omitempty"`
}

type MCPPolicyAppsPolicy struct {
	Enabled                          *bool `json:"enabled,omitempty"`
	AllowAppInitiatedToolCalls       *bool `json:"allowAppInitiatedToolCalls,omitempty"`
	RequireApprovalForOpenLink       *bool `json:"requireApprovalForOpenLink,omitempty"`
	RequireApprovalForContextUpdates *bool `json:"requireApprovalForContextUpdates,omitempty"`
}

func DecodeMCPPolicyJSON(raw []byte) (MCPPolicyDocument, error) {
	return jsonutil.DecodeCanonicalObject[MCPPolicyDocument](
		raw,
		basespec.MaxDefinitionBytes,
	)
}

func (v MCPPolicyDocument) Clone() (MCPPolicyDocument, error) {
	return jsonutil.CloneJSON(v)
}

func (v MCPPolicyDocument) Canonicalize() (MCPPolicyDocument, error) {
	return jsonutil.CloneJSON(v)
}

func (v MCPPolicyDocument) CanonicalJSON() ([]byte, error) {
	return jsonutil.MarshalCanonicalObject(v, basespec.MaxDefinitionBytes)
}

func (v MCPPolicyDocument) CalculatedDigest() (
	cryptoutil.Digest,
	error,
) {
	payload := v
	payload.Digest = nil
	return cryptoutil.CanonicalDigest(payload)
}

func (v MCPPolicyDocument) Validate() error {
	if err := jsonutil.ValidateJSONSchema(
		compiledMCPPolicySchema,
		v,
		basespec.MaxDefinitionBytes,
	); err != nil {
		return fmt.Errorf("MCP Policy schema: %w", err)
	}
	return basespec.ValidatePortableMetadata(
		basespec.LogicalName(v.LogicalName),
		basespec.LogicalVersion(v.LogicalVersion),
		v.DisplayName,
		v.Description,
		v.Labels,
		v.Digest,
	)
}
