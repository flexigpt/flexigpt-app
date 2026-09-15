package agentv1

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
	AgentType          = declaration.TypeAgent
	AgentSchemaID      = "artifact.agent.v1"
	AgentSchemaVersion = declaration.APIVersionV1
)

//go:embed agent-v1.schema.json
var schemaJSON []byte

var compiledAgentSchema = jsonutil.MustCompileJSONSchema(schemaJSON)

var AgentSchemaKey = schema.ArtifactKey(
	artifact.ArtifactKind(AgentType),
	schema.SchemaID(AgentSchemaID),
	AgentSchemaVersion,
)

type AgentDocument struct {
	declaration.Header

	Members []declaration.Entry `json:"members,omitempty"`
	Program *declaration.Entry  `json:"program,omitempty"`
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
	if err := entry.DecodeInto(&value); err != nil {
		return AgentDocument{}, err
	}
	if err := value.ValidateEntry(); err != nil {
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
	if err := value.validate(); err != nil {
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
	if err := v.Header.Validate(declaration.HeaderValidation{
		ExpectedType: AgentType,
		APIVersion:   AgentSchemaVersion,
	}); err != nil {
		return err
	}
	if len(v.Members) > basespec.MaxDefinitionDependencies {
		return fmt.Errorf(
			"%w: agent members exceed %d entries",
			basespec.ErrInvalid,
			basespec.MaxDefinitionDependencies,
		)
	}
	if err := declaration.ValidateEntryTypes(
		"Agent members",
		v.Members,
		declaration.TypeInstruction,
		declaration.TypeContext,
		declaration.TypeModel,
		declaration.TypeSkill,
		declaration.TypeTool,
		declaration.TypeMCP,
		declaration.TypeMCPPolicy,
		declaration.TypeCollection,
		declaration.TypeAgent,
	); err != nil {
		return err
	}
	if v.Program == nil {
		return nil
	}
	if err := v.Program.Validate(); err != nil {
		return fmt.Errorf("agent program: %w", err)
	}
	switch v.Program.Header().Type {
	case declaration.TypeLoop, declaration.TypeWorkflow:
		return nil
	default:
		return fmt.Errorf(
			"%w: agent program has incompatible type %q",
			basespec.ErrInvalid,
			v.Program.Header().Type,
		)
	}
}
