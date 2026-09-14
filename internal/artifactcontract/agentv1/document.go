package agentv1

import (
	_ "embed"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	AgentType          = artifactcontract.TypeAgent
	AgentSchemaID      = "artifact.agent.v1"
	AgentSchemaVersion = artifactcontract.APIVersionV1
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
	artifactcontract.Header

	Members []artifactcontract.Entry `json:"members,omitempty"`
	Program *artifactcontract.Entry  `json:"program,omitempty"`
}

func AgentJSONSchema() []byte {
	return append([]byte(nil), schemaJSON...)
}

func DecodeAgentJSON(raw []byte) (AgentDocument, error) {
	return decodeAgent(raw, true)
}

func DecodeAgentEntry(
	entry artifactcontract.Entry,
) (AgentDocument, error) {
	raw, err := entry.CanonicalJSON()
	if err != nil {
		return AgentDocument{}, err
	}
	return decodeAgent(raw, false)
}

func decodeAgent(
	raw []byte,
	requireName bool,
) (AgentDocument, error) {
	var value AgentDocument
	if err := artifactcontract.DecodeDocumentInto(
		raw,
		compiledAgentSchema,
		&value,
	); err != nil {
		return AgentDocument{}, err
	}
	if err := value.validate(requireName); err != nil {
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
	return artifactcontract.CanonicalDocumentJSON(v)
}

func (v AgentDocument) CalculatedDigest() (
	cryptoutil.Digest,
	error,
) {
	return artifactcontract.DocumentDigest(v)
}

func (v AgentDocument) Validate() error {
	return v.validate(true)
}

func (v AgentDocument) ValidateEntry() error {
	return v.validate(false)
}

func (v AgentDocument) validate(requireName bool) error {
	if err := artifactcontract.ValidateDocument(
		compiledAgentSchema,
		v,
	); err != nil {
		return fmt.Errorf("agent schema: %w", err)
	}
	if err := v.Header.Validate(artifactcontract.HeaderValidation{
		ExpectedType: AgentType,
		APIVersion:   AgentSchemaVersion,
		RequireName:  requireName,
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
	if err := artifactcontract.ValidateEntryTypes(
		"Agent members",
		v.Members,
		artifactcontract.TypeInstruction,
		artifactcontract.TypeContext,
		artifactcontract.TypeModel,
		artifactcontract.TypeSkill,
		artifactcontract.TypeTool,
		artifactcontract.TypeMCP,
		artifactcontract.TypeCollection,
		artifactcontract.TypeAgent,
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
	case artifactcontract.TypeLoop, artifactcontract.TypeWorkflow:
		return nil
	default:
		return fmt.Errorf(
			"%w: agent program has incompatible type %q",
			basespec.ErrInvalid,
			v.Program.Header().Type,
		)
	}
}
