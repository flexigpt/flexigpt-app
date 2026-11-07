package sessionv3

import (
	"errors"
	"fmt"
	"strings"
)

type TextDispatchInsert string

const (
	TextDispatchInsertInstructions TextDispatchInsert = "instructions"
	TextDispatchInsertUserMessage  TextDispatchInsert = "user-message"
)

type MCPToolDispatchPlan struct {
	Name           string  `json:"name"`
	ExpectedDigest string  `json:"expectedDigest,omitempty"`
	ApprovalRule   *string `json:"approvalRule,omitempty"`
	ExecutionMode  *string `json:"executionMode,omitempty"`
}

type MCPResourceDispatchPlan struct {
	URI            string `json:"uri"`
	ExpectedDigest string `json:"expectedDigest,omitempty"`
}

type MCPResourceTemplateDispatchPlan struct {
	URITemplate    string            `json:"uriTemplate"`
	ExpectedDigest string            `json:"expectedDigest,omitempty"`
	Arguments      map[string]string `json:"arguments,omitempty"`
}

type MCPPromptDispatchPlan struct {
	PromptName     string            `json:"promptName"`
	ExpectedDigest string            `json:"expectedDigest,omitempty"`
	Arguments      map[string]string `json:"arguments,omitempty"`
}

type ServiceDispatchPlan struct {
	ID             string         `json:"id"`
	BindingVersion string         `json:"bindingVersion"`
	Input          CanonicalValue `json:"input"`
	Context        CanonicalValue `json:"context,omitempty"`
	Ports          []PortPlan     `json:"ports,omitempty"`
	PortInputs     []PortInput    `json:"portInputs,omitempty"`
	Effect         EffectContract `json:"effect"`
}

type ActivityDispatchPlan struct {
	ID             string         `json:"id"`
	BindingVersion string         `json:"bindingVersion"`
	Input          CanonicalValue `json:"input"`
	Context        CanonicalValue `json:"context,omitempty"`
	Ports          []PortPlan     `json:"ports,omitempty"`
	PortInputs     []PortInput    `json:"portInputs,omitempty"`
	Effect         EffectContract `json:"effect"`
}

func (p AgentDispatchPlan) Validate() error {
	if err := validatePin(p.Artifact); err != nil {
		return err
	}
	if p.Model != nil {
		if err := validatePin(p.Model.Artifact); err != nil {
			return err
		}
	}
	if p.Workspace != nil {
		if err := p.Workspace.Validate(); err != nil {
			return err
		}
	}
	if err := validateDispatchInstructions(p.Instructions); err != nil {
		return err
	}
	if err := validateDispatchTexts(p.Texts); err != nil {
		return err
	}
	if err := validateDispatchTools(p.Tools); err != nil {
		return err
	}
	if err := validateDispatchSkills(p.Skills); err != nil {
		return err
	}
	return validateDispatchMCP(p.MCP)
}

func (p WorkspaceDispatchPlan) Validate() error {
	if err := validatePin(p.Artifact); err != nil {
		return err
	}
	if err := validateDispatchInstructions(p.Instructions); err != nil {
		return err
	}
	if err := validateDispatchTexts(p.Texts); err != nil {
		return err
	}
	if err := validateDispatchSkills(p.Skills); err != nil {
		return err
	}
	if err := validateDispatchMCP(p.MCP); err != nil {
		return err
	}
	return validateDispatchContext(p.Context)
}

func (p ServiceDispatchPlan) Validate() error {
	return validateExternalDispatchPlan(
		"service dispatch plan",
		p.ID,
		p.BindingVersion,
		p.Input,
		p.Context,
		p.Ports,
		p.PortInputs,
		p.Effect,
	)
}

func (p ActivityDispatchPlan) Validate() error {
	return validateExternalDispatchPlan(
		"activity dispatch plan",
		p.ID,
		p.BindingVersion,
		p.Input,
		p.Context,
		p.Ports,
		p.PortInputs,
		p.Effect,
	)
}

func validateExternalDispatchPlan(
	label string,
	id string,
	version string,
	input CanonicalValue,
	contextValue CanonicalValue,
	ports []PortPlan,
	portInputs []PortInput,
	effect EffectContract,
) error {
	if err := validateID(label+" id", id); err != nil {
		return err
	}
	if err := validateID(label+" binding version", version); err != nil {
		return err
	}
	if !input.Valid() || !contextValue.Valid() {
		return errors.New("external dispatch plan contains invalid canonical values")
	}
	if err := effect.Validate(); err != nil {
		return err
	}
	if err := validateUniquePortPlans(ports); err != nil {
		return err
	}
	return validateUniquePortInputs(portInputs)
}

func validateExtendedModelDispatchPlan(
	plan ModelDispatchPlan,
) error {
	if err := validateDispatchTools(plan.Tools); err != nil {
		return err
	}
	if err := validateDispatchMCP(plan.MCP); err != nil {
		return err
	}
	if err := validateDispatchContext(plan.Context); err != nil {
		return err
	}
	if err := validateUniquePortInputs(plan.PortInputs); err != nil {
		return err
	}

	attachments := make(map[string]struct{}, len(plan.Attachments))
	for _, attachment := range plan.Attachments {
		if err := attachment.Attachment.Validate(); err != nil {
			return fmt.Errorf("model dispatch plan attachment: %w", err)
		}
		key := attachment.MessageID + "\x00" + attachment.Attachment.Digest
		if _, duplicate := attachments[key]; duplicate {
			return errors.New("model dispatch plan attachment is duplicated")
		}
		attachments[key] = struct{}{}
	}

	for _, message := range plan.Messages {
		if message.ID != "" {
			if err := validateID("model dispatch plan message id", message.ID); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateDispatchInstructions(
	values []InstructionDispatchPlan,
) error {
	ids := make(map[string]struct{}, len(values))
	for _, value := range values {
		if strings.TrimSpace(value.Content) == "" {
			return errors.New("dispatch instruction has no content")
		}
		if value.ID != "" {
			if _, duplicate := ids[value.ID]; duplicate {
				return errors.New("dispatch instruction is duplicated")
			}
			ids[value.ID] = struct{}{}
		}
		if value.Artifact != nil {
			if err := validatePin(*value.Artifact); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateDispatchTexts(
	values []TextDispatchPlan,
) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if err := validatePin(value.Artifact); err != nil {
			return err
		}
		switch value.Insert {
		case TextDispatchInsertInstructions, TextDispatchInsertUserMessage:
		default:
			return errors.New("text dispatch plan insert is invalid")
		}
		key := value.Artifact.RootID + "\x00" +
			value.Artifact.ArtifactID + "\x00" +
			string(value.Insert)
		if _, duplicate := seen[key]; duplicate {
			return errors.New("text dispatch plan is duplicated")
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validateDispatchTools(
	values []ToolDispatchPlan,
) error {
	for _, value := range values {
		if err := value.Call.Validate(); err != nil {
			return err
		}
		provider := value.Call.Binding.Kind == ToolBindingKindProvider
		if provider != value.ProviderManaged {
			return errors.New("tool dispatch plan provider ownership differs from its binding")
		}
	}
	return nil
}

func validateDispatchSkills(
	values []SkillDispatchPlan,
) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if err := validatePin(value.Artifact); err != nil {
			return err
		}
		key := value.Artifact.RootID + "\x00" + value.Artifact.ArtifactID
		if _, duplicate := seen[key]; duplicate {
			return errors.New("skill dispatch plan is duplicated")
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validateDispatchMCP(
	values []MCPDispatchPlan,
) error {
	seenServers := make(map[string]struct{}, len(values))
	for _, value := range values {
		if err := validatePin(value.Server); err != nil {
			return err
		}
		switch value.ToolExposure {
		case MCPToolExposureNone,
			MCPToolExposureAll,
			MCPToolExposureSelected:
		default:
			return errors.New("mcp dispatch plan has an invalid tool exposure")
		}
		serverKey := value.Server.RootID + "\x00" + value.Server.ArtifactID
		if _, duplicate := seenServers[serverKey]; duplicate {
			return errors.New("mcp dispatch plan server is duplicated")
		}
		seenServers[serverKey] = struct{}{}

		tools := make(map[string]struct{}, len(value.Tools))
		for _, tool := range value.Tools {
			if err := validateID("mcp dispatch plan tool name", tool.Name); err != nil {
				return err
			}
			if _, duplicate := tools[tool.Name]; duplicate {
				return errors.New("mcp dispatch plan tool is duplicated")
			}
			tools[tool.Name] = struct{}{}
		}

		resources := make(map[string]struct{}, len(value.Resources))
		for _, resource := range value.Resources {
			if strings.TrimSpace(resource.URI) == "" {
				return errors.New("mcp dispatch plan resource has no uri")
			}
			if _, duplicate := resources[resource.URI]; duplicate {
				return errors.New("mcp dispatch plan resource is duplicated")
			}
			resources[resource.URI] = struct{}{}
		}

		templates := make(map[string]struct{}, len(value.ResourceTemplates))
		for _, template := range value.ResourceTemplates {
			if strings.TrimSpace(template.URITemplate) == "" {
				return errors.New("mcp dispatch plan resource template has no uri template")
			}
			if _, duplicate := templates[template.URITemplate]; duplicate {
				return errors.New("mcp dispatch plan resource template is duplicated")
			}
			templates[template.URITemplate] = struct{}{}
		}

		prompts := make(map[string]struct{}, len(value.Prompts))
		for _, prompt := range value.Prompts {
			if err := validateID("mcp dispatch plan prompt name", prompt.PromptName); err != nil {
				return err
			}
			if _, duplicate := prompts[prompt.PromptName]; duplicate {
				return errors.New("mcp dispatch plan prompt is duplicated")
			}
			prompts[prompt.PromptName] = struct{}{}
		}
	}
	return nil
}

func validateDispatchContext(
	values []ContextDispatchPlan,
) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if err := validateID("dispatch context name", value.Name); err != nil {
			return err
		}
		if !value.Value.Valid() {
			return errors.New("dispatch context value is invalid")
		}
		if _, duplicate := seen[value.Name]; duplicate {
			return errors.New("dispatch context is duplicated")
		}
		seen[value.Name] = struct{}{}
	}
	return nil
}
