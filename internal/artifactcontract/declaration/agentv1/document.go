package agentv1

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	AgentType          = declaration.TypeAgent
	AgentSchemaID      = "artifact.agent.v1"
	AgentSchemaVersion = declaration.SchemaVersionV1
)

//go:embed agent-v1.schema.json
var schemaJSON []byte

var compiledAgentSchema = jsonutil.MustCompileJSONSchema(schemaJSON)

var AgentSchemaKey = schema.ArtifactKey(
	artifact.ArtifactKind(AgentType),
	schema.SchemaID(AgentSchemaID),
	AgentSchemaVersion,
)

type Prompt struct {
	MediaType string `json:"mediaType,omitempty"`
	Content   string `json:"content"`
}

type AgentDocument struct {
	declaration.Header

	Members  []declaration.Entry `json:"members,omitempty"`
	Prompt   *Prompt             `json:"prompt,omitempty"`
	Loop     *declaration.Entry  `json:"loop,omitempty"`
	Workflow *declaration.Entry  `json:"workflow,omitempty"`
}

func AgentJSONSchema() []byte {
	return append([]byte(nil), schemaJSON...)
}

func DecodeAgentJSON(raw []byte) (AgentDocument, error) {
	return decodeAgent(raw)
}

func DecodeAgentEntry(
	entry declaration.Entry,
) (AgentDocument, error) {
	var value AgentDocument
	if err := declaration.DecodeEntryDocumentInto(
		entry,
		compiledAgentSchema,
		&value,
	); err != nil {
		return AgentDocument{}, err
	}
	if err := value.validateFields(); err != nil {
		return AgentDocument{}, err
	}
	return value, nil
}

func decodeAgent(
	raw []byte,
) (AgentDocument, error) {
	var value AgentDocument
	if err := declaration.DecodeDocumentInto(
		raw,
		compiledAgentSchema,
		&value,
	); err != nil {
		return AgentDocument{}, err
	}
	if err := value.validateFields(); err != nil {
		return AgentDocument{}, err
	}
	return value, nil
}

func (v AgentDocument) Clone() (AgentDocument, error) {
	return jsonutil.CloneJSON(v)
}

func (v AgentDocument) Canonicalize() (AgentDocument, error) {
	return v.Clone()
}

func (v AgentDocument) CanonicalJSON() ([]byte, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	return declaration.CanonicalDocumentJSON(v)
}

func (v AgentDocument) CalculatedDigest() (
	cryptoutil.Digest,
	error,
) {
	return declaration.DocumentDigest(v)
}

func (v AgentDocument) Validate() error {
	return v.validate()
}

func (v AgentDocument) ValidateEntry() error {
	return v.validate()
}

func (v AgentDocument) validate() error {
	if err := declaration.ValidateDocument(
		compiledAgentSchema,
		v,
	); err != nil {
		return fmt.Errorf("agent schema: %w", err)
	}
	return v.validateFields()
}

func (v AgentDocument) validateFields() error {
	if err := v.Header.Validate(declaration.HeaderValidation{
		ExpectedType: AgentType,
		RequireName:  true,
	}); err != nil {
		return err
	}
	if err := declaration.ValidateDeclarationLocatorExclusivity(
		"Agent",
		v.Locator,
		v.Members != nil ||
			v.Prompt != nil ||
			v.Loop != nil ||
			v.Workflow != nil,
	); err != nil {
		return err
	}
	if err := declaration.ValidateMemberTypes(
		"Agent members",
		v.Members,
		declaration.TypeText,
		declaration.TypeModel,
		declaration.TypeSkill,
		declaration.TypeTool,
		declaration.TypeMCP,
		declaration.TypeMCPPolicy,
		declaration.TypePlugin,
		declaration.TypeAgent,
	); err != nil {
		return err
	}
	for index, member := range v.Members {
		if err := validateMemberRelationship(member); err != nil {
			return fmt.Errorf("agent members[%d]: %w", index, err)
		}
	}
	if v.Prompt != nil {
		if err := declaration.ValidateOptionalMediaType(v.Prompt.MediaType); err != nil {
			return err
		}
		content := v.Prompt.Content
		if err := declaration.ValidateOptionalContent(&content); err != nil {
			return err
		}
	}
	if v.Loop != nil && v.Workflow != nil {
		return fmt.Errorf(
			"%w: Agent cannot contain both loop and workflow",
			basespec.ErrInvalid,
		)
	}
	if v.Loop != nil {
		if err := validateProgramMember("Agent loop", *v.Loop, declaration.TypeLoop); err != nil {
			return err
		}
	}
	if v.Workflow != nil {
		if err := validateProgramMember(
			"Agent workflow",
			*v.Workflow,
			declaration.TypeWorkflow,
		); err != nil {
			return err
		}
	}
	return nil
}

func validateProgramMember(
	label string,
	value declaration.Entry,
	expected declaration.Type,
) error {
	form, err := value.MemberForm()
	if err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	if form == declaration.MemberSelector ||
		value.Header().Type != expected {
		return fmt.Errorf(
			"%w: %s must be one named or contained %q member",
			basespec.ErrInvalid,
			label,
			expected,
		)
	}
	return nil
}

func validateMemberRelationship(member declaration.Entry) error {
	relationship, err := member.Relationship()
	if err != nil {
		return err
	}

	switch member.Header().Type {
	case declaration.TypeModel:
		if len(relationship.Use) != 0 {
			return fmt.Errorf(
				"%w: Agent Model member cannot contain use",
				basespec.ErrInvalid,
			)
		}
		return validateBooleanOverride(
			relationship.Overrides,
			"includeSystemPrompt",
		)

	case declaration.TypeTool:
		if len(relationship.Use) != 0 {
			return fmt.Errorf(
				"%w: Agent Tool member cannot contain use",
				basespec.ErrInvalid,
			)
		}
		return validateBooleanOverride(
			relationship.Overrides,
			"autoExecute",
		)

	case declaration.TypeSkill:
		if len(relationship.Overrides) != 0 {
			return fmt.Errorf(
				"%w: agent skill member cannot contain overrides",
				basespec.ErrInvalid,
			)
		}
		if len(relationship.Use) == 0 {
			return nil
		}
		if len(relationship.Use) != 1 {
			return fmt.Errorf(
				"%w: agent skill use supports only mode",
				basespec.ErrInvalid,
			)
		}
		raw, found := relationship.Use["mode"]
		if !found {
			return fmt.Errorf(
				"%w: agent skill use requires mode",
				basespec.ErrInvalid,
			)
		}
		var mode string
		if err := json.Unmarshal(raw, &mode); err != nil {
			return fmt.Errorf("agent skill use mode: %w", err)
		}
		switch mode {
		case "available", "active", "instructions":
			return nil
		default:
			return fmt.Errorf(
				"%w: unsupported agent skill use mode %q",
				basespec.ErrInvalid,
				mode,
			)
		}

	default:
		if len(relationship.Overrides) != 0 ||
			len(relationship.Use) != 0 {
			return fmt.Errorf(
				"%w: Agent %q member does not support overrides or use",
				basespec.ErrInvalid,
				member.Header().Type,
			)
		}
		return nil
	}
}

func validateBooleanOverride(
	values map[string]json.RawMessage,
	allowed string,
) error {
	for name, raw := range values {
		if name != allowed {
			return fmt.Errorf(
				"%w: unsupported Agent member override %q",
				basespec.ErrInvalid,
				name,
			)
		}
		var value bool
		if err := json.Unmarshal(raw, &value); err != nil {
			return fmt.Errorf(
				"agent member override %q: %w",
				name,
				err,
			)
		}
	}
	return nil
}
