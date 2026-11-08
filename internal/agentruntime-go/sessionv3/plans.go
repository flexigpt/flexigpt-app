package sessionv3

import (
	"errors"
	"fmt"
	"strings"
)

type WorkflowContextRequirement struct {
	Name   string         `json:"name"`
	Schema CanonicalValue `json:"schema"`
}

type WorkflowCallablePort struct {
	Name      string                `json:"name"`
	Direction WorkflowPortDirection `json:"direction"`
	Schema    CanonicalValue        `json:"schema"`
	Stream    bool                  `json:"stream,omitempty"`
}

type WorkflowCapabilityRequirement struct {
	Name string `json:"name"`
}

type ModelDispatchPlan struct {
	Agent          *AgentDispatchPlan        `json:"agent,omitempty"`
	Workspace      *WorkspaceDispatchPlan    `json:"workspace,omitempty"`
	Model          ModelDispatchTarget       `json:"model"`
	ModelOverrides *ModelDefaultsOverride    `json:"modelOverrides,omitempty"`
	Instructions   []InstructionDispatchPlan `json:"instructions,omitempty"`
	Texts          []TextDispatchPlan        `json:"texts,omitempty"`
	Tools          []ToolDispatchPlan        `json:"tools,omitempty"`
	Skills         []SkillDispatchPlan       `json:"skills,omitempty"`
	MCP            []MCPDispatchPlan         `json:"mcp,omitempty"`
	Attachments    []AttachmentDispatchPlan  `json:"attachments,omitempty"`
	Messages       []MessageDispatchPlan     `json:"messages,omitempty"`
	Context        []ContextDispatchPlan     `json:"context,omitempty"`
	Channels       []ChannelDispatchPlan     `json:"channels,omitempty"`
	Ports          []PortPlan                `json:"ports,omitempty"`
	PortInputs     []PortInput               `json:"portInputs,omitempty"`
}

type AgentDispatchPlan struct {
	Artifact     ArtifactPin               `json:"artifact"`
	Model        *ModelDispatchTarget      `json:"model,omitempty"`
	Workspace    *WorkspaceDispatchPlan    `json:"workspace,omitempty"`
	Instructions []InstructionDispatchPlan `json:"instructions,omitempty"`
	Texts        []TextDispatchPlan        `json:"texts,omitempty"`
	Tools        []ToolDispatchPlan        `json:"tools,omitempty"`
	Skills       []SkillDispatchPlan       `json:"skills,omitempty"`
	MCP          []MCPDispatchPlan         `json:"mcp,omitempty"`
}

type WorkspaceDispatchPlan struct {
	Artifact     ArtifactPin               `json:"artifact"`
	Instructions []InstructionDispatchPlan `json:"instructions,omitempty"`
	Texts        []TextDispatchPlan        `json:"texts,omitempty"`
	Skills       []SkillDispatchPlan       `json:"skills,omitempty"`
	MCP          []MCPDispatchPlan         `json:"mcp,omitempty"`
	Context      []ContextDispatchPlan     `json:"context,omitempty"`
}

type ModelDispatchTarget struct {
	Artifact ArtifactPin `json:"artifact"`
}

type InstructionDispatchPlan struct {
	ID       string       `json:"id"`
	Label    string       `json:"label,omitempty"`
	Content  string       `json:"content"`
	Artifact *ArtifactPin `json:"artifact,omitempty"`
}

type TextDispatchPlan struct {
	Artifact ArtifactPin        `json:"artifact"`
	Insert   TextDispatchInsert `json:"insert"`
	Content  string             `json:"content,omitempty"`
}

type ToolDispatchPlan struct {
	Call            ToolCall `json:"call"`
	ProviderManaged bool     `json:"providerManaged,omitempty"`
}

type SkillDispatchPlan struct {
	Artifact ArtifactPin `json:"artifact"`
	Active   bool        `json:"active,omitempty"`
}

type MCPDispatchPlan struct {
	Server                    ArtifactPin                       `json:"server"`
	IncludeServerInstructions bool                              `json:"includeServerInstructions,omitempty"`
	ToolExposure              MCPToolExposure                   `json:"toolExposure"`
	Tools                     []MCPToolDispatchPlan             `json:"tools,omitempty"`
	Resources                 []MCPResourceDispatchPlan         `json:"resources,omitempty"`
	ResourceTemplates         []MCPResourceTemplateDispatchPlan `json:"resourceTemplates,omitempty"`
	Prompts                   []MCPPromptDispatchPlan           `json:"prompts,omitempty"`
}

type AttachmentDispatchPlan struct {
	MessageID  string     `json:"messageID,omitempty"`
	Attachment Attachment `json:"attachment"`
}

type MessageDispatchPlan struct {
	ID          string       `json:"id,omitempty"`
	Role        MessageRole  `json:"role"`
	Text        string       `json:"text,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

type ContextDispatchPlan struct {
	Name  string         `json:"name"`
	Value CanonicalValue `json:"value"`
}

type ChannelDispatchPlan struct {
	Name   string             `json:"name"`
	Cursor uint64             `json:"cursor,omitempty"`
	Access WorkflowPortAccess `json:"access"`
}

func (p ModelDispatchPlan) Validate() error {
	if err := validatePin(p.Model.Artifact); err != nil {
		return fmt.Errorf("model dispatch plan model: %w", err)
	}
	if p.Agent != nil {
		if err := p.Agent.Validate(); err != nil {
			return fmt.Errorf("model dispatch plan agent: %w", err)
		}
	}
	if p.Workspace != nil {
		if err := p.Workspace.Validate(); err != nil {
			return fmt.Errorf("model dispatch plan workspace: %w", err)
		}
	}

	instructionIDs := make(map[string]struct{}, len(p.Instructions))
	for _, value := range p.Instructions {
		if strings.TrimSpace(value.Content) == "" {
			return errors.New("model dispatch plan instruction has no content")
		}
		if value.ID != "" {
			if _, found := instructionIDs[value.ID]; found {
				return errors.New("model dispatch plan instruction is duplicated")
			}
			instructionIDs[value.ID] = struct{}{}
		}
		if value.Artifact != nil {
			if err := validatePin(*value.Artifact); err != nil {
				return fmt.Errorf(
					"model dispatch plan instruction artifact: %w",
					err,
				)
			}
		}
	}

	for _, value := range p.Texts {
		if err := validatePin(value.Artifact); err != nil {
			return fmt.Errorf("model dispatch plan text artifact: %w", err)
		}
		if strings.TrimSpace(string(value.Insert)) == "" {
			return errors.New("model dispatch plan text has no insert mode")
		}
	}

	toolIDs := make(map[string]struct{}, len(p.Tools))
	for _, value := range p.Tools {
		if err := value.Call.Validate(); err != nil {
			return fmt.Errorf("model dispatch plan tool: %w", err)
		}
		key := firstText(value.Call.ChoiceID, value.Call.Name)
		if key == "" {
			return errors.New("model dispatch plan tool has no choice identity")
		}
		if _, found := toolIDs[key]; found {
			return errors.New("model dispatch plan tool is duplicated")
		}
		toolIDs[key] = struct{}{}
	}

	for _, value := range p.Skills {
		if err := validatePin(value.Artifact); err != nil {
			return fmt.Errorf("model dispatch plan skill: %w", err)
		}
	}

	for _, value := range p.MCP {
		if err := validatePin(value.Server); err != nil {
			return fmt.Errorf("model dispatch plan mcp server: %w", err)
		}
		switch value.ToolExposure {
		case MCPToolExposureNone,
			MCPToolExposureAll,
			MCPToolExposureSelected:
		default:
			return errors.New("model dispatch plan has an invalid mcp exposure")
		}
	}

	for _, value := range p.Attachments {
		if err := value.Attachment.Validate(); err != nil {
			return fmt.Errorf("model dispatch plan attachment: %w", err)
		}
	}

	for _, value := range p.Messages {
		switch value.Role {
		case MessageRoleSystem,
			MessageRoleDeveloper,
			MessageRoleUser,
			MessageRoleAssistant:
		default:
			return errors.New("model dispatch plan message has an invalid role")
		}
		for _, attachment := range value.Attachments {
			if err := attachment.Validate(); err != nil {
				return fmt.Errorf(
					"model dispatch plan message attachment: %w",
					err,
				)
			}
		}
	}

	contextNames := make(map[string]struct{}, len(p.Context))
	for _, value := range p.Context {
		if err := validateID("model dispatch plan context name", value.Name); err != nil {
			return err
		}
		if !value.Value.Valid() {
			return errors.New("model dispatch plan context is invalid")
		}
		if _, found := contextNames[value.Name]; found {
			return errors.New("model dispatch plan context is duplicated")
		}
		contextNames[value.Name] = struct{}{}
	}

	channelNames := make(map[string]struct{}, len(p.Channels))
	for _, value := range p.Channels {
		if err := validateID("model dispatch plan channel name", value.Name); err != nil {
			return err
		}
		switch value.Access {
		case WorkflowPortAccessRead,
			WorkflowPortAccessWrite,
			WorkflowPortAccessReadWrite:
		default:
			return errors.New("model dispatch plan channel access is invalid")
		}
		if _, found := channelNames[value.Name]; found {
			return errors.New("model dispatch plan channel is duplicated")
		}
		channelNames[value.Name] = struct{}{}
	}

	portNames := make(map[string]struct{}, len(p.Ports))
	for _, value := range p.Ports {
		if err := value.Validate(); err != nil {
			return fmt.Errorf("model dispatch plan port: %w", err)
		}
		if _, found := portNames[value.Name]; found {
			return errors.New("model dispatch plan port is duplicated")
		}
		portNames[value.Name] = struct{}{}
	}
	return validateExtendedModelDispatchPlan(p)
}

func (p WorkflowContextRequirement) Validate() error {
	if err := validateID("workflow context name", p.Name); err != nil {
		return err
	}
	if len(p.Schema) == 0 || !p.Schema.Valid() {
		return errors.New("workflow context schema is invalid")
	}
	return nil
}

func (p WorkflowCallablePort) Validate() error {
	if err := validateID("workflow callable port name", p.Name); err != nil {
		return err
	}
	switch p.Direction {
	case WorkflowPortDirectionInput, WorkflowPortDirectionOutput:
	default:
		return errors.New("workflow callable port direction is invalid")
	}
	if len(p.Schema) == 0 || !p.Schema.Valid() {
		return errors.New("workflow callable port schema is invalid")
	}
	return nil
}

func (p WorkflowCapabilityRequirement) Validate() error {
	return validateID("workflow capability requirement", p.Name)
}

func workflowContextRequirement(
	values []WorkflowContextRequirement,
	name string,
) (WorkflowContextRequirement, bool) {
	for _, value := range values {
		if value.Name == name {
			return value, true
		}
	}
	return WorkflowContextRequirement{}, false
}

func workflowCallablePort(
	values []WorkflowCallablePort,
	name string,
) (WorkflowCallablePort, bool) {
	for _, value := range values {
		if value.Name == name {
			return value, true
		}
	}
	return WorkflowCallablePort{}, false
}
