package teamv1

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
	TeamType          = declaration.TypeTeam
	TeamSchemaID      = "artifact.team.v1"
	TeamSchemaVersion = declaration.APIVersionV1
)

//go:embed team-v1.schema.json
var schemaJSON []byte

var compiledTeamSchema = jsonutil.MustCompileJSONSchema(schemaJSON)

var TeamSchemaKey = schema.ArtifactKey(
	artifact.ArtifactKind(TeamType),
	schema.SchemaID(TeamSchemaID),
	TeamSchemaVersion,
)

type TeamDocument struct {
	declaration.Header

	Members []declaration.Entry `json:"members,omitempty"`
	Program *declaration.Entry  `json:"program,omitempty"`
}

func TeamJSONSchema() []byte {
	return append([]byte(nil), schemaJSON...)
}

func DecodeTeamJSON(raw []byte) (TeamDocument, error) {
	return decodeTeam(raw)
}

func DecodeTeamEntry(
	entry declaration.Entry,
) (TeamDocument, error) {
	var value TeamDocument
	if err := entry.DecodeInto(&value); err != nil {
		return TeamDocument{}, err
	}
	if err := value.ValidateEntry(); err != nil {
		return TeamDocument{}, err
	}
	return value, nil
}

func decodeTeam(
	raw []byte,
) (TeamDocument, error) {
	var value TeamDocument
	if err := declaration.DecodeDocumentInto(
		raw,
		compiledTeamSchema,
		&value,
	); err != nil {
		return TeamDocument{}, err
	}
	if err := value.validate(); err != nil {
		return TeamDocument{}, err
	}
	return value, nil
}

func (v TeamDocument) Clone() (TeamDocument, error) {
	return jsonutil.CloneJSON(v)
}

func (v TeamDocument) Canonicalize() (TeamDocument, error) {
	return v.Clone()
}

func (v TeamDocument) CanonicalJSON() ([]byte, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	return declaration.CanonicalDocumentJSON(v)
}

func (v TeamDocument) CalculatedDigest() (
	cryptoutil.Digest,
	error,
) {
	return declaration.DocumentDigest(v)
}

func (v TeamDocument) Validate() error {
	return v.validate()
}

func (v TeamDocument) ValidateEntry() error {
	return v.validate()
}

func (v TeamDocument) validate() error {
	if err := declaration.ValidateDocument(
		compiledTeamSchema,
		v,
	); err != nil {
		return fmt.Errorf("team schema: %w", err)
	}
	if err := v.Header.Validate(declaration.HeaderValidation{
		ExpectedType: TeamType,
		APIVersion:   TeamSchemaVersion,
	}); err != nil {
		return err
	}
	if len(v.Members) > basespec.MaxDefinitionDependencies {
		return fmt.Errorf(
			"%w: Team members exceed %d entries",
			basespec.ErrInvalid,
			basespec.MaxDefinitionDependencies,
		)
	}
	if err := declaration.ValidateEntryTypes(
		"Team members",
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
		return fmt.Errorf("team program: %w", err)
	}
	switch v.Program.Header().Type {
	case declaration.TypeLoop, declaration.TypeWorkflow:
		return nil
	default:
		return fmt.Errorf(
			"%w: Team program has incompatible type %q",
			basespec.ErrInvalid,
			v.Program.Header().Type,
		)
	}
}
