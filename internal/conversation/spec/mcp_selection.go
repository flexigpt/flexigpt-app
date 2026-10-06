package spec

import (
	"fmt"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/policy"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/server"
)

type MCPAppModelContextUpdate struct {
	InstanceID string             `json:"instanceID,omitempty"`
	Server     mcpServer.ServerID `json:"server"`

	ResourceURI string `json:"resourceUri,omitempty"`

	Content           []mcpServer.MCPContent `json:"content,omitempty"`
	StructuredContent any                    `json:"structuredContent,omitempty"`
	UpdatedAt         string                 `json:"updatedAt,omitempty"`
	RawArguments      jsonutil.JSONRawString `json:"rawArguments,omitempty"`
}

// Validate validates the server identity in an App-originated model-context update.
func (value MCPAppModelContextUpdate) Validate() error {
	return value.Server.Validate()
}

type MCPToolSelection struct {
	Server           mcpServer.ServerID `json:"server"`
	ToolName         string             `json:"toolName"`
	ProviderToolName string             `json:"providerToolName,omitempty"`
	ChoiceID         string             `json:"choiceID,omitempty"`
	Digest           string             `json:"digest,omitempty"`

	ApprovalRule  *mcpPolicy.MCPApprovalRule  `json:"approvalRule,omitempty"`
	ExecutionMode *mcpPolicy.MCPExecutionMode `json:"executionMode,omitempty"`

	AppResourceURI string   `json:"appResourceUri,omitempty"`
	Visibility     []string `json:"visibility,omitempty"`
}

// Validate validates a selected MCP tool and its optional policy tightening.
func (value MCPToolSelection) Validate() error {
	if err := value.Server.Validate(); err != nil {
		return err
	}
	if err := validateConversationRequiredText(
		"selected MCP tool name",
		value.ToolName,
		mcpServer.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := validateConversationOptionalText(
		"selected MCP provider tool name",
		value.ProviderToolName,
		mcpServer.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := validateConversationOptionalText(
		"selected MCP choice ID",
		value.ChoiceID,
		mcpServer.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if value.Digest != "" {
		if err := mcpServer.Digest(value.Digest).Validate(); err != nil {
			return err
		}
	}
	if err := validateMCPAppResourceURI(value.AppResourceURI); err != nil {
		return err
	}
	if value.ApprovalRule != nil {
		if err := value.ApprovalRule.Validate(); err != nil {
			return err
		}
	}
	if value.ExecutionMode != nil {
		if err := value.ExecutionMode.Validate(); err != nil {
			return err
		}
	}

	return validateMCPVisibility(value.Visibility)
}

func (value MCPToolSelection) sameVisibility(other []string) bool {
	selected := make(map[string]struct{}, len(value.Visibility))
	for _, raw := range value.Visibility {
		selected[strings.ToLower(strings.TrimSpace(raw))] = struct{}{}
	}

	mapped := make(map[string]struct{}, len(other))
	for _, raw := range other {
		mapped[strings.ToLower(strings.TrimSpace(raw))] = struct{}{}
	}

	if len(selected) != len(mapped) {
		return false
	}
	for visibility := range selected {
		if _, found := mapped[visibility]; !found {
			return false
		}
	}
	return true
}

type MCPProviderToolMapping struct {
	Server mcpServer.ServerID `json:"server"`

	ProviderToolName string `json:"providerToolName"`
	ChoiceID         string `json:"choiceID"`

	ToolName   string `json:"toolName"`
	ToolDigest string `json:"toolDigest"`

	ApprovalRule   mcpPolicy.MCPApprovalRule  `json:"approvalRule,omitempty"`
	ExecutionMode  mcpPolicy.MCPExecutionMode `json:"executionMode,omitempty"`
	AppResourceURI string                     `json:"appResourceUri,omitempty"`
	Visibility     []string                   `json:"visibility,omitempty"`
}

// Validate validates one durable provider-tool mapping.
func (value MCPProviderToolMapping) Validate() error {
	if err := value.Server.Validate(); err != nil {
		return err
	}
	if err := validateConversationRequiredText(
		"MCP provider tool name",
		value.ProviderToolName,
		mcpServer.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := validateConversationRequiredText(
		"MCP choice ID",
		value.ChoiceID,
		mcpServer.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := validateConversationRequiredText(
		"MCP tool name",
		value.ToolName,
		mcpServer.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := mcpServer.Digest(value.ToolDigest).Validate(); err != nil {
		return err
	}
	if err := value.ApprovalRule.Validate(); err != nil {
		return err
	}
	if err := value.ExecutionMode.Validate(); err != nil {
		return err
	}
	if err := validateMCPVisibility(value.Visibility); err != nil {
		return err
	}
	return validateMCPAppResourceURI(value.AppResourceURI)
}

func (value MCPProviderToolMapping) validateAgainst(
	selection MCPToolSelection,
) error {
	if selection.ProviderToolName != "" &&
		selection.ProviderToolName != value.ProviderToolName {
		return fmt.Errorf(
			"%w: mapped MCP provider tool identity changed for %q",
			mcpServer.ErrInvalid,
			value.ToolName,
		)
	}
	if selection.ChoiceID != "" && selection.ChoiceID != value.ChoiceID {
		return fmt.Errorf(
			"%w: mapped MCP choice identity changed for %q",
			mcpServer.ErrInvalid,
			value.ToolName,
		)
	}
	if selection.Digest != "" && selection.Digest != value.ToolDigest {
		return fmt.Errorf(
			"%w: mapped MCP tool digest changed for %q",
			mcpServer.ErrInvalid,
			value.ToolName,
		)
	}
	if selection.AppResourceURI != "" &&
		selection.AppResourceURI != value.AppResourceURI {
		return fmt.Errorf(
			"%w: mapped MCP App resource changed for %q",
			mcpServer.ErrInvalid,
			value.ToolName,
		)
	}
	if len(selection.Visibility) != 0 &&
		!selection.sameVisibility(value.Visibility) {
		return fmt.Errorf(
			"%w: mapped MCP App visibility changed for %q",
			mcpServer.ErrInvalid,
			value.ToolName,
		)
	}
	if selection.ApprovalRule != nil &&
		mcpPolicy.ApprovalRuleRank(value.ApprovalRule) <
			mcpPolicy.ApprovalRuleRank(*selection.ApprovalRule) {
		return fmt.Errorf(
			"%w: mapped MCP approval rule weakens conversation policy",
			mcpServer.ErrInvalid,
		)
	}
	if selection.ExecutionMode != nil &&
		mcpPolicy.ExecutionModeRank(value.ExecutionMode) <
			mcpPolicy.ExecutionModeRank(*selection.ExecutionMode) {
		return fmt.Errorf(
			"%w: mapped MCP execution mode weakens conversation policy",
			mcpServer.ErrInvalid,
		)
	}
	return nil
}

type MCPToolExposure string

const (
	MCPToolExposureNone     MCPToolExposure = "none"
	MCPToolExposureAll      MCPToolExposure = "all"
	MCPToolExposureSelected MCPToolExposure = "selected"
)

// Validate reports whether value is a supported MCP tool-exposure mode.
func (value MCPToolExposure) Validate() error {
	switch value {
	case MCPToolExposureNone, MCPToolExposureAll, MCPToolExposureSelected:
		return nil
	default:
		return fmt.Errorf(
			"%w: invalid MCP tool exposure %q",
			mcpServer.ErrInvalid,
			value,
		)
	}
}

type MCPServerSelection struct {
	Server mcpServer.ServerID `json:"server"`

	SnapshotDigest string `json:"snapshotDigest,omitempty"`

	ToolExposure  MCPToolExposure    `json:"toolExposure"`
	SelectedTools []MCPToolSelection `json:"selectedTools,omitempty"`

	IncludeServerInstructions bool `json:"includeServerInstructions,omitempty"`
}

// Validate validates a selected MCP server and its selected tools.
func (value MCPServerSelection) Validate() error {
	if err := value.Server.Validate(); err != nil {
		return err
	}
	if value.SnapshotDigest != "" {
		if err := mcpServer.Digest(value.SnapshotDigest).Validate(); err != nil {
			return err
		}
	}
	if err := value.ToolExposure.Validate(); err != nil {
		return err
	}

	if value.ToolExposure == MCPToolExposureNone &&
		len(value.SelectedTools) != 0 {
		return fmt.Errorf(
			"%w: selected MCP tools require selected tool exposure",
			mcpServer.ErrInvalid,
		)
	}
	if value.ToolExposure == MCPToolExposureSelected &&
		len(value.SelectedTools) == 0 {
		return fmt.Errorf(
			"%w: selected MCP tool exposure requires tools",
			mcpServer.ErrInvalid,
		)
	}

	tools := make(map[string]struct{}, len(value.SelectedTools))
	for index, tool := range value.SelectedTools {
		if err := tool.Validate(); err != nil {
			return fmt.Errorf("selected tools[%d]: %w", index, err)
		}
		if tool.Server != value.Server {
			return fmt.Errorf(
				"%w: selected MCP tool belongs to another server",
				mcpServer.ErrInvalid,
			)
		}
		if _, duplicate := tools[tool.ToolName]; duplicate {
			return fmt.Errorf(
				"%w: duplicate selected MCP tool %q",
				mcpServer.ErrInvalid,
				tool.ToolName,
			)
		}
		tools[tool.ToolName] = struct{}{}
	}
	return nil
}

func (value MCPServerSelection) selectedTool(
	name string,
) (MCPToolSelection, bool) {
	for _, tool := range value.SelectedTools {
		if tool.ToolName == name {
			return tool, true
		}
	}
	return MCPToolSelection{}, false
}

type MCPResourceTemplateSelection struct {
	mcpServer.MCPResourceTemplateRef

	ArgumentValues map[string]string `json:"argumentValues,omitempty"`
}

// Validate validates a selected MCP resource template.
func (value MCPResourceTemplateSelection) Validate() error {
	ref := value.MCPResourceTemplateRef
	if err := ref.Server.Validate(); err != nil {
		return err
	}
	return validateMCPArguments(value.ArgumentValues, ref.Arguments)
}

type MCPPromptSelection struct {
	mcpServer.MCPPromptRef

	ArgumentValues map[string]string `json:"argumentValues,omitempty"`
}

// Validate validates a selected MCP prompt.
func (value MCPPromptSelection) Validate() error {
	ref := value.MCPPromptRef
	if err := ref.Server.Validate(); err != nil {
		return err
	}
	return validateMCPArguments(value.ArgumentValues, ref.Arguments)
}

type MCPConversationContext struct {
	Servers           []MCPServerSelection           `json:"servers"`
	Resources         []mcpServer.MCPResourceRef     `json:"resources,omitempty"`
	ResourceTemplates []MCPResourceTemplateSelection `json:"resourceTemplates,omitempty"`
	Prompts           []MCPPromptSelection           `json:"prompts,omitempty"`
}

// Validate validates durable MCP conversation structure without requiring a
// live MCP runtime connection.
func (value MCPConversationContext) Validate() error {
	if len(value.Servers) > mcpServer.MaxDiscoveryCandidates ||
		len(value.Resources) > mcpServer.MaxDiscoveryCandidates ||
		len(value.ResourceTemplates) > mcpServer.MaxDiscoveryCandidates ||
		len(value.Prompts) > mcpServer.MaxDiscoveryCandidates {
		return fmt.Errorf(
			"%w: MCP conversation context exceeds entry limits",
			mcpServer.ErrInvalid,
		)
	}

	type referenceKey struct {
		server mcpServer.ServerID
		value  string
	}

	servers := make(map[mcpServer.ServerID]struct{}, len(value.Servers))
	for index, selection := range value.Servers {
		if err := selection.Validate(); err != nil {
			return fmt.Errorf("MCP servers[%d]: %w", index, err)
		}
		if _, duplicate := servers[selection.Server]; duplicate {
			return fmt.Errorf(
				"%w: duplicate MCP server selection %q",
				mcpServer.ErrInvalid,
				selection.Server,
			)
		}
		servers[selection.Server] = struct{}{}
	}

	resources := make(map[referenceKey]struct{}, len(value.Resources))
	for index, resource := range value.Resources {
		if err := resource.Server.Validate(); err != nil {
			return fmt.Errorf("MCP resources[%d]: %w", index, err)
		}
		if _, selected := servers[resource.Server]; !selected {
			return fmt.Errorf(
				"%w: MCP resource %q belongs to a server not selected by the conversation",
				mcpServer.ErrInvalid,
				resource.URI,
			)
		}

		key := referenceKey{
			server: resource.Server,
			value:  resource.URI,
		}
		if _, duplicate := resources[key]; duplicate {
			return fmt.Errorf(
				"%w: duplicate MCP resource %q",
				mcpServer.ErrInvalid,
				resource.URI,
			)
		}
		resources[key] = struct{}{}
	}

	templates := make(map[referenceKey]struct{}, len(value.ResourceTemplates))
	for index, selection := range value.ResourceTemplates {
		if err := selection.Validate(); err != nil {
			return fmt.Errorf("MCP resource templates[%d]: %w", index, err)
		}
		if _, selected := servers[selection.Server]; !selected {
			return fmt.Errorf(
				"%w: MCP resource template %q belongs to a server not selected by the conversation",
				mcpServer.ErrInvalid,
				selection.URITemplate,
			)
		}

		key := referenceKey{
			server: selection.Server,
			value:  selection.URITemplate,
		}
		if _, duplicate := templates[key]; duplicate {
			return fmt.Errorf(
				"%w: duplicate MCP resource template %q",
				mcpServer.ErrInvalid,
				selection.URITemplate,
			)
		}
		templates[key] = struct{}{}
	}

	prompts := make(map[referenceKey]struct{}, len(value.Prompts))
	for index, selection := range value.Prompts {
		if err := selection.Validate(); err != nil {
			return fmt.Errorf("MCP prompts[%d]: %w", index, err)
		}
		if _, selected := servers[selection.Server]; !selected {
			return fmt.Errorf(
				"%w: MCP prompt %q belongs to a server not selected by the conversation",
				mcpServer.ErrInvalid,
				selection.PromptName,
			)
		}

		key := referenceKey{
			server: selection.Server,
			value:  selection.PromptName,
		}
		if _, duplicate := prompts[key]; duplicate {
			return fmt.Errorf(
				"%w: duplicate MCP prompt %q",
				mcpServer.ErrInvalid,
				selection.PromptName,
			)
		}
		prompts[key] = struct{}{}
	}

	return nil
}

// ValidateProviderToolMappings validates durable provider-tool mappings against
// this exact durable MCP context.
func (value MCPConversationContext) ValidateProviderToolMappings(
	mappings []MCPProviderToolMapping,
) error {
	if err := value.Validate(); err != nil {
		return err
	}

	servers := value.serverSelections()
	providerNames := make(map[string]struct{}, len(mappings))
	choiceIDs := make(map[string]struct{}, len(mappings))

	for index, mapping := range mappings {
		if err := mapping.Validate(); err != nil {
			return fmt.Errorf("MCP provider tool mappings[%d]: %w", index, err)
		}
		if _, duplicate := providerNames[mapping.ProviderToolName]; duplicate {
			return fmt.Errorf(
				"%w: duplicate MCP provider tool name %q",
				mcpServer.ErrInvalid,
				mapping.ProviderToolName,
			)
		}
		if _, duplicate := choiceIDs[mapping.ChoiceID]; duplicate {
			return fmt.Errorf(
				"%w: duplicate MCP choice ID %q",
				mcpServer.ErrInvalid,
				mapping.ChoiceID,
			)
		}

		providerNames[mapping.ProviderToolName] = struct{}{}
		choiceIDs[mapping.ChoiceID] = struct{}{}

		server, selected := servers[mapping.Server]
		if !selected {
			return fmt.Errorf(
				"%w: mapped MCP tool %q belongs to an unselected server",
				mcpServer.ErrInvalid,
				mapping.ToolName,
			)
		}

		switch server.ToolExposure {
		case MCPToolExposureSelected:
			selection, found := server.selectedTool(mapping.ToolName)
			if !found {
				return fmt.Errorf(
					"%w: mapped MCP tool %q was not selected",
					mcpServer.ErrInvalid,
					mapping.ToolName,
				)
			}
			if err := mapping.validateAgainst(selection); err != nil {
				return err
			}

		case MCPToolExposureAll:
			// An all-tools selection may still contain per-tool policy
			// tightening that must apply to durable mappings.
			if selection, found := server.selectedTool(mapping.ToolName); found {
				if err := mapping.validateAgainst(selection); err != nil {
					return err
				}
			}

		default:
			return fmt.Errorf(
				"%w: mapped MCP tool %q has no enabled tool exposure",
				mcpServer.ErrInvalid,
				mapping.ToolName,
			)
		}
	}

	return nil
}

// ValidateAppContextUpdates validates App-originated model-context updates
// against this exact durable MCP context.
func (value MCPConversationContext) ValidateAppContextUpdates(
	updates []MCPAppModelContextUpdate,
) error {
	if err := value.Validate(); err != nil {
		return err
	}

	for index, update := range updates {
		if err := update.Validate(); err != nil {
			return fmt.Errorf("MCP App context updates[%d]: %w", index, err)
		}
	}

	servers := value.serverSelections()
	for index, update := range updates {
		if _, selected := servers[update.Server]; !selected {
			return fmt.Errorf(
				"%w: MCP App context update %d belongs to an unselected server",
				mcpServer.ErrInvalid,
				index,
			)
		}
	}

	return nil
}

func (value MCPConversationContext) serverSelections() map[mcpServer.ServerID]MCPServerSelection {
	selections := make(map[mcpServer.ServerID]MCPServerSelection, len(value.Servers))
	for _, selection := range value.Servers {
		selections[selection.Server] = selection
	}
	return selections
}

func validateMCPArguments(
	argumentValues map[string]string,
	definitions map[string]mcpServer.MCPArgumentDefinition,
) error {
	for name, argumentValue := range argumentValues {
		if !utf8.ValidString(argumentValue) ||
			len(argumentValue) > mcpServer.MaxDescriptionBytes {
			return fmt.Errorf(
				"%w: MCP argument value %q is invalid",
				mcpServer.ErrInvalid,
				name,
			)
		}
	}

	for name, definition := range definitions {
		if definition.Name != "" && definition.Name != name {
			return fmt.Errorf(
				"%w: MCP argument definition key %q differs from name %q",
				mcpServer.ErrInvalid,
				name,
				definition.Name,
			)
		}
	}

	return nil
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
				mcpServer.ErrInvalid,
				raw,
			)
		}
		if _, duplicate := seen[value]; duplicate {
			return fmt.Errorf(
				"%w: duplicate MCP App visibility %q",
				mcpServer.ErrInvalid,
				value,
			)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func validateConversationRequiredText(
	subject string,
	value string,
	maximum int,
) error {
	if value == "" {
		return fmt.Errorf(
			"%w: %s is required",
			mcpServer.ErrInvalid,
			subject,
		)
	}
	return validateConversationOptionalText(subject, value, maximum)
}

func validateConversationOptionalText(
	subject string,
	value string,
	maximum int,
) error {
	if value == "" {
		return nil
	}
	if !utf8.ValidString(value) ||
		strings.TrimSpace(value) != value ||
		len(value) > maximum ||
		strings.ContainsRune(value, 0) {
		return fmt.Errorf(
			"%w: %s is invalid",
			mcpServer.ErrInvalid,
			subject,
		)
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return fmt.Errorf(
				"%w: %s contains a control character",
				mcpServer.ErrInvalid,
				subject,
			)
		}
	}
	return nil
}

func validateMCPAppResourceURI(value string) error {
	if value == "" {
		return nil
	}
	if err := validateConversationOptionalText(
		"MCP App resource URI",
		value,
		mcpServer.MaxURIBytes,
	); err != nil {
		return err
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf(
			"%w: MCP App resource URI is invalid: %w",
			mcpServer.ErrInvalid,
			err,
		)
	}
	if parsed.Scheme != "ui" ||
		parsed.Host == "" ||
		parsed.User != nil ||
		parsed.Fragment != "" {
		return fmt.Errorf(
			"%w: MCP App resource URI must be a ui:// URI",
			mcpServer.ErrInvalid,
		)
	}
	return nil
}
