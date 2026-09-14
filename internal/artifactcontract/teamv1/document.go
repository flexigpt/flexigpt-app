package teamv1

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
	TeamType          = artifactcontract.TypeTeam
	TeamSchemaID      = "artifact.team.v1"
	TeamSchemaVersion = artifactcontract.APIVersionV1
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
	artifactcontract.Header

	Members []artifactcontract.Entry `json:"members,omitempty"`
	Program *artifactcontract.Entry  `json:"program,omitempty"`
}

func TeamJSONSchema() []byte {
	return append([]byte(nil), schemaJSON...)
}

func DecodeTeamJSON(raw []byte) (TeamDocument, error) {
	return decodeTeam(raw, true)
}

func DecodeTeamEntry(
	entry artifactcontract.Entry,
) (TeamDocument, error) {
	raw, err := entry.CanonicalJSON()
	if err != nil {
		return TeamDocument{}, err
	}
	return decodeTeam(raw, false)
}

func decodeTeam(
	raw []byte,
	requireName bool,
) (TeamDocument, error) {
	var value TeamDocument
	if err := artifactcontract.DecodeDocumentInto(
		raw,
		compiledTeamSchema,
		&value,
	); err != nil {
		return TeamDocument{}, err
	}
	if err := value.validate(requireName); err != nil {
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
	return artifactcontract.CanonicalDocumentJSON(v)
}

func (v TeamDocument) CalculatedDigest() (
	cryptoutil.Digest,
	error,
) {
	return artifactcontract.DocumentDigest(v)
}

func (v TeamDocument) Validate() error {
	return v.validate(true)
}

func (v TeamDocument) ValidateEntry() error {
	return v.validate(false)
}

func (v TeamDocument) validate(requireName bool) error {
	if err := artifactcontract.ValidateDocument(
		compiledTeamSchema,
		v,
	); err != nil {
		return fmt.Errorf("team schema: %w", err)
	}
	if err := v.Header.Validate(artifactcontract.HeaderValidation{
		ExpectedType: TeamType,
		APIVersion:   TeamSchemaVersion,
		RequireName:  requireName,
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
	if err := artifactcontract.ValidateEntryTypes(
		"Team members",
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
		return fmt.Errorf("team program: %w", err)
	}
	switch v.Program.Header().Type {
	case artifactcontract.TypeLoop, artifactcontract.TypeWorkflow:
		return nil
	default:
		return fmt.Errorf(
			"%w: Team program has incompatible type %q",
			basespec.ErrInvalid,
			v.Program.Header().Type,
		)
	}
}
