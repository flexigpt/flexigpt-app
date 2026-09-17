package mcppolicyv1

import (
	_ "embed"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	MCPPolicyType          = declaration.TypeMCPPolicy
	MCPPolicySchemaID      = "artifact.mcp-policy.v1"
	MCPPolicySchemaVersion = declaration.SchemaVersionV1
)

//go:embed mcp-policy-v1.schema.json
var schemaJSON []byte

var compiledMCPPolicySchema = jsonutil.MustCompileJSONSchema(schemaJSON)

var MCPPolicySchemaKey = schema.ArtifactKey(
	artifact.ArtifactKind(MCPPolicyType),
	schema.SchemaID(MCPPolicySchemaID),
	MCPPolicySchemaVersion,
)

type MCPPolicyDocument struct {
	declaration.Header

	TrustLevel    string                                 `json:"trustLevel,omitempty"`
	DefaultPolicy *MCPPolicyDefaultPolicy                `json:"defaultPolicy,omitempty"`
	ToolPolicies  map[string]MCPPolicyToolPolicyOverride `json:"toolPolicies,omitempty"`
	AppsPolicy    *MCPPolicyAppsPolicy                   `json:"appsPolicy,omitempty"`
}

type MCPPolicyToolPolicyOverride struct {
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

func MCPPolicyJSONSchema() []byte {
	return append([]byte(nil), schemaJSON...)
}

func DecodeMCPPolicyJSON(raw []byte) (MCPPolicyDocument, error) {
	return decodeMCPPolicy(raw)
}

func DecodeMCPPolicyEntry(
	entry declaration.Entry,
) (MCPPolicyDocument, error) {
	var value MCPPolicyDocument
	if err := declaration.DecodeEntryDocumentInto(
		entry,
		compiledMCPPolicySchema,
		&value,
	); err != nil {
		return MCPPolicyDocument{}, err
	}
	if err := value.validateFields(); err != nil {
		return MCPPolicyDocument{}, err
	}
	return value, nil
}

func decodeMCPPolicy(
	raw []byte,
) (MCPPolicyDocument, error) {
	var value MCPPolicyDocument
	if err := declaration.DecodeDocumentInto(
		raw,
		compiledMCPPolicySchema,
		&value,
	); err != nil {
		return MCPPolicyDocument{}, err
	}
	if err := value.validateFields(); err != nil {
		return MCPPolicyDocument{}, err
	}
	return value, nil
}

func (v MCPPolicyDocument) Clone() (
	MCPPolicyDocument,
	error,
) {
	return jsonutil.CloneJSON(v)
}

func (v MCPPolicyDocument) Canonicalize() (
	MCPPolicyDocument,
	error,
) {
	return v.Clone()
}

func (v MCPPolicyDocument) CanonicalJSON() ([]byte, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	return declaration.CanonicalDocumentJSON(v)
}

func (v MCPPolicyDocument) CalculatedDigest() (
	cryptoutil.Digest,
	error,
) {
	return declaration.DocumentDigest(v)
}

func (v MCPPolicyDocument) Validate() error {
	return v.validate()
}

func (v MCPPolicyDocument) ValidateEntry() error {
	return v.validate()
}

func (v MCPPolicyDocument) validate() error {
	if err := declaration.ValidateDocument(
		compiledMCPPolicySchema,
		v,
	); err != nil {
		return fmt.Errorf("MCP Policy schema: %w", err)
	}
	return v.validateFields()
}

func (v MCPPolicyDocument) validateFields() error {
	if err := v.Header.Validate(declaration.HeaderValidation{
		ExpectedType: MCPPolicyType,
		RequireName:  true,
	}); err != nil {
		return err
	}
	if err := declaration.ValidateDeclarationLocatorExclusivity(
		"MCP Policy",
		v.Locator,
		v.hasPolicyFields(),
	); err != nil {
		return err
	}
	return v.validatePolicy()
}

func (v MCPPolicyDocument) validatePolicy() error {
	switch v.TrustLevel {
	case "", "trusted", "untrusted":
	default:
		return fmt.Errorf(
			"%w: invalid MCP policy trustLevel %q",
			basespec.ErrInvalid,
			v.TrustLevel,
		)
	}
	if len(v.ToolPolicies) > basespec.MaxDefinitionDependencies {
		return fmt.Errorf(
			"%w: MCP policy toolPolicies exceed %d entries",
			basespec.ErrInvalid,
			basespec.MaxDefinitionDependencies,
		)
	}
	for name, policy := range v.ToolPolicies {
		if err := basespec.ValidateRequiredText(
			"MCP policy tool name",
			name,
			basespec.MaxLogicalNameBytes,
		); err != nil {
			return err
		}
		if err := validateApprovalRule(
			"MCP policy approval rule",
			policy.ApprovalRule,
		); err != nil {
			return err
		}
		if err := validateExecutionMode(
			"MCP policy execution mode",
			policy.ExecutionMode,
		); err != nil {
			return err
		}
		if policy.ExpectedDigest != "" {
			if err := cryptoutil.ValidateDigest(
				cryptoutil.Digest(policy.ExpectedDigest),
			); err != nil {
				return err
			}
		}
	}
	if v.DefaultPolicy != nil {
		if err := validateApprovalRule(
			"MCP policy default approval rule",
			v.DefaultPolicy.DefaultApprovalRule,
		); err != nil {
			return err
		}
		if err := validateExecutionMode(
			"MCP policy default execution mode",
			v.DefaultPolicy.DefaultExecutionMode,
		); err != nil {
			return err
		}
	}
	return nil
}

func (v MCPPolicyDocument) hasPolicyFields() bool {
	return v.TrustLevel != "" ||
		v.DefaultPolicy != nil ||
		v.ToolPolicies != nil ||
		v.AppsPolicy != nil
}

func validateApprovalRule(
	label string,
	value string,
) error {
	switch value {
	case "", "allow", "ask", "deny":
		return nil
	default:
		return fmt.Errorf(
			"%w: %s %q is invalid",
			basespec.ErrInvalid,
			label,
			value,
		)
	}
}

func validateExecutionMode(
	label string,
	value string,
) error {
	switch value {
	case "", "auto", "manual":
		return nil
	default:
		return fmt.Errorf(
			"%w: %s %q is invalid",
			basespec.ErrInvalid,
			label,
			value,
		)
	}
}
