package validate

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

// ValidateMCPConversationContext validates durable MCP conversation selection
// structure without requiring a live MCP runtime connection.
func ValidateMCPConversationContext(value spec.MCPConversationContext) error {
	servers := make(map[artifact.ArtifactRef]struct{}, len(value.Servers))
	for index, selection := range value.Servers {
		if err := validateMCPServerSelection(selection); err != nil {
			return fmt.Errorf("MCP servers[%d]: %w", index, err)
		}
		if _, duplicate := servers[selection.Server]; duplicate {
			return fmt.Errorf(
				"%w: duplicate MCP server selection %q",
				basespec.ErrInvalid,
				selection.Server.ArtifactID,
			)
		}
		servers[selection.Server] = struct{}{}
	}

	resources := make(map[string]struct{}, len(value.Resources))
	for index, resource := range value.Resources {
		if err := validateMCPResourceRef(resource); err != nil {
			return fmt.Errorf("MCP resources[%d]: %w", index, err)
		}
		key := mcpReferenceKey(resource.Server, resource.URI)
		if _, duplicate := resources[key]; duplicate {
			return fmt.Errorf(
				"%w: duplicate MCP resource %q",
				basespec.ErrInvalid,
				resource.URI,
			)
		}
		resources[key] = struct{}{}
	}

	templates := make(
		map[string]struct{},
		len(value.ResourceTemplates),
	)
	for index, selection := range value.ResourceTemplates {
		if err := validateMCPResourceTemplateSelection(selection); err != nil {
			return fmt.Errorf(
				"MCP resource templates[%d]: %w",
				index,
				err,
			)
		}
		key := mcpReferenceKey(
			selection.Server,
			selection.URITemplate,
		)
		if _, duplicate := templates[key]; duplicate {
			return fmt.Errorf(
				"%w: duplicate MCP resource template %q",
				basespec.ErrInvalid,
				selection.URITemplate,
			)
		}
		templates[key] = struct{}{}
	}

	prompts := make(map[string]struct{}, len(value.Prompts))
	for index, selection := range value.Prompts {
		if err := validateMCPPromptSelection(selection); err != nil {
			return fmt.Errorf("MCP prompts[%d]: %w", index, err)
		}
		key := mcpReferenceKey(
			selection.Server,
			selection.PromptName,
		)
		if _, duplicate := prompts[key]; duplicate {
			return fmt.Errorf(
				"%w: duplicate MCP prompt %q",
				basespec.ErrInvalid,
				selection.PromptName,
			)
		}
		prompts[key] = struct{}{}
	}

	return nil
}

// ValidateMCPAppContextUpdates validates application-originated MCP App
// context updates before they are inserted into model input.
func ValidateMCPAppContextUpdates(
	values []spec.MCPAppModelContextUpdate,
) error {
	for index, value := range values {
		if err := value.Server.Validate(); err != nil {
			return fmt.Errorf("MCP App context updates[%d]: %w", index, err)
		}
		if err := basespec.ValidateOptionalText(
			"MCP App instance ID",
			value.InstanceID,
			basespec.MaxDisplayNameBytes,
		); err != nil {
			return fmt.Errorf("MCP App context updates[%d]: %w", index, err)
		}
		if err := basespec.ValidateOptionalText(
			"MCP App resource URI",
			value.ResourceURI,
			basespec.MaxURIBytes,
		); err != nil {
			return fmt.Errorf("MCP App context updates[%d]: %w", index, err)
		}
	}
	return nil
}

// ValidateMCPProviderToolMapping validates one durable provider-tool mapping emitted during MCP
// inference hydration. These mappings bind later model tool calls to a
// specific Artifact-backed MCP server and discovered tool digest.
func ValidateMCPProviderToolMapping(m spec.MCPProviderToolMapping) error {
	if err := m.Server.Validate(); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"MCP provider tool name",
		m.ProviderToolName,
		basespec.MaxLocatorBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"MCP provider choice ID",
		m.ChoiceID,
		basespec.MaxLocatorBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"MCP tool name",
		m.ToolName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"MCP tool digest",
		m.ToolDigest,
		basespec.MaxFingerprintBytes,
	); err != nil {
		return err
	}
	if err := validateMCPApprovalRule(m.ApprovalRule); err != nil {
		return err
	}
	if err := validateMCPExecutionMode(m.ExecutionMode); err != nil {
		return err
	}
	if err := basespec.ValidateOptionalText(
		"MCP App resource URI",
		m.AppResourceURI,
		basespec.MaxURIBytes,
	); err != nil {
		return err
	}
	return validateMCPVisibility(m.Visibility)
}

// ValidateMCPInvocationSource validates the source of a tool invocation.
// The caller is still responsible for policy and approval enforcement.
func ValidateMCPInvocationSource(value spec.MCPInvocationSource) error {
	switch value {
	case spec.MCPInvocationSourceModel,
		spec.MCPInvocationSourceUser,
		spec.MCPInvocationSourceApp:
		return nil
	default:
		return fmt.Errorf(
			"%w: invalid MCP invocation source %q",
			basespec.ErrInvalid,
			value,
		)
	}
}

func validateMCPServerSelection(value spec.MCPServerSelection) error {
	if err := value.Server.Validate(); err != nil {
		return err
	}
	if err := basespec.ValidateOptionalText(
		"MCP snapshot digest",
		value.SnapshotDigest,
		basespec.MaxFingerprintBytes,
	); err != nil {
		return err
	}

	switch value.ToolExposure {
	case spec.MCPToolExposureNone, spec.MCPToolExposureAll:
		if len(value.SelectedTools) != 0 {
			return fmt.Errorf(
				"%w: selected MCP tools require selected tool exposure",
				basespec.ErrInvalid,
			)
		}

	case spec.MCPToolExposureSelected:
		if len(value.SelectedTools) == 0 {
			return fmt.Errorf(
				"%w: selected MCP tool exposure requires tools",
				basespec.ErrInvalid,
			)
		}

	default:
		return fmt.Errorf(
			"%w: invalid MCP tool exposure %q",
			basespec.ErrInvalid,
			value.ToolExposure,
		)
	}

	tools := make(map[string]struct{}, len(value.SelectedTools))
	for index, tool := range value.SelectedTools {
		if err := validateMCPToolSelection(tool); err != nil {
			return fmt.Errorf("selected tools[%d]: %w", index, err)
		}
		if tool.Server != value.Server {
			return fmt.Errorf(
				"%w: selected MCP tool belongs to another server",
				basespec.ErrInvalid,
			)
		}
		if _, duplicate := tools[tool.ToolName]; duplicate {
			return fmt.Errorf(
				"%w: duplicate selected MCP tool %q",
				basespec.ErrInvalid,
				tool.ToolName,
			)
		}
		tools[tool.ToolName] = struct{}{}
	}
	return nil
}

func validateMCPToolSelection(value spec.MCPToolSelection) error {
	if err := value.Server.Validate(); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"MCP tool name",
		value.ToolName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateOptionalText(
		"MCP provider tool name",
		value.ProviderToolName,
		basespec.MaxLocatorBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateOptionalText(
		"MCP tool choice ID",
		value.ChoiceID,
		basespec.MaxLocatorBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateOptionalText(
		"MCP tool digest",
		value.Digest,
		basespec.MaxFingerprintBytes,
	); err != nil {
		return err
	}
	if value.ApprovalRule != nil {
		if err := validateMCPApprovalRule(*value.ApprovalRule); err != nil {
			return err
		}
	}
	if value.ExecutionMode != nil {
		if err := validateMCPExecutionMode(*value.ExecutionMode); err != nil {
			return err
		}
	}
	if err := basespec.ValidateOptionalText(
		"MCP App resource URI",
		value.AppResourceURI,
		basespec.MaxURIBytes,
	); err != nil {
		return err
	}
	return validateMCPVisibility(value.Visibility)
}

func validateMCPResourceRef(value spec.MCPResourceRef) error {
	if err := value.Server.Validate(); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"MCP resource URI",
		value.URI,
		basespec.MaxURIBytes,
	); err != nil {
		return err
	}
	return basespec.ValidateOptionalText(
		"MCP resource digest",
		value.Digest,
		basespec.MaxFingerprintBytes,
	)
}

func validateMCPResourceTemplateSelection(
	value spec.MCPResourceTemplateSelection,
) error {
	ref := value.MCPResourceTemplateRef
	if err := ref.Server.Validate(); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"MCP resource URI template",
		ref.URITemplate,
		basespec.MaxURIBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateOptionalText(
		"MCP resource template digest",
		ref.Digest,
		basespec.MaxFingerprintBytes,
	); err != nil {
		return err
	}
	if err := validateMCPArgumentValues(value.ArgumentValues); err != nil {
		return err
	}
	return validateMCPArgumentDefinitions(ref.Arguments)
}

func validateMCPPromptSelection(value spec.MCPPromptSelection) error {
	if err := value.Server.Validate(); err != nil {
		return err
	}
	if err := basespec.ValidateRequiredText(
		"MCP prompt name",
		value.PromptName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateOptionalText(
		"MCP prompt digest",
		value.Digest,
		basespec.MaxFingerprintBytes,
	); err != nil {
		return err
	}
	if err := validateMCPArgumentValues(value.ArgumentValues); err != nil {
		return err
	}
	return validateMCPArgumentDefinitions(value.Arguments)
}

func validateMCPArgumentDefinitions(
	values map[string]spec.MCPArgumentDefinition,
) error {
	for name, value := range values {
		if err := basespec.ValidateRequiredText(
			"MCP argument name",
			name,
			basespec.MaxKindBytes,
		); err != nil {
			return err
		}
		if value.Name != "" && value.Name != name {
			return fmt.Errorf(
				"%w: MCP argument definition key %q differs from name %q",
				basespec.ErrInvalid,
				name,
				value.Name,
			)
		}
	}
	return nil
}

func validateMCPArgumentValues(values map[string]string) error {
	for name, value := range values {
		if err := basespec.ValidateRequiredText(
			"MCP argument value name",
			name,
			basespec.MaxKindBytes,
		); err != nil {
			return err
		}
		if !utf8.ValidString(value) ||
			len(value) > basespec.MaxDescriptionBytes {
			return fmt.Errorf(
				"%w: MCP argument value %q is invalid",
				basespec.ErrInvalid,
				name,
			)
		}
	}
	return nil
}

func validateMCPApprovalRule(value spec.MCPApprovalRule) error {
	switch value {
	case spec.MCPApprovalRuleAllow,
		spec.MCPApprovalRuleAsk,
		spec.MCPApprovalRuleDeny:
		return nil
	default:
		return fmt.Errorf(
			"%w: invalid MCP approval rule %q",
			basespec.ErrInvalid,
			value,
		)
	}
}

func validateMCPExecutionMode(value spec.MCPExecutionMode) error {
	switch value {
	case spec.MCPExecutionModeAuto, spec.MCPExecutionModeManual:
		return nil
	default:
		return fmt.Errorf(
			"%w: invalid MCP execution mode %q",
			basespec.ErrInvalid,
			value,
		)
	}
}

func validateMCPVisibility(values []string) error {
	seen := make(map[string]struct{}, len(values))
	for _, raw := range values {
		value := strings.ToLower(strings.TrimSpace(raw))
		switch value {
		case "model", "app":
		default:
			return fmt.Errorf(
				"%w: invalid MCP App visibility %q",
				basespec.ErrInvalid,
				raw,
			)
		}
		if _, duplicate := seen[value]; duplicate {
			return fmt.Errorf(
				"%w: duplicate MCP App visibility %q",
				basespec.ErrInvalid,
				value,
			)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func mcpReferenceKey(
	server artifact.ArtifactRef,
	value string,
) string {
	return string(server.RootID) +
		"\x00" +
		string(server.ArtifactID) +
		"\x00" +
		value
}
