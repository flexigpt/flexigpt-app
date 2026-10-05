package teamv1

import (
	_ "embed"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

const (
	TeamType          declaration.Type = "team"
	TeamSchemaID                       = "artifact.team.v1"
	TeamSchemaVersion                  = declaration.SchemaVersionV1
)

//go:embed team-v1.schema.json
var schemaJSON []byte

var compiledTeamSchema = jsonutil.MustCompileJSONSchema(schemaJSON)

var TeamSchemaKey = schemaModel.ArtifactKey(
	artifactModel.ArtifactKind(TeamType),
	schemaModel.SchemaID(TeamSchemaID),
	TeamSchemaVersion,
)

type TeamDocument struct {
	declaration.Header

	Members  []declaration.Entry `json:"members,omitempty"`
	Loop     *declaration.Entry  `json:"loop,omitempty"`
	Workflow *declaration.Entry  `json:"workflow,omitempty"`
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
	if err := declaration.DecodeEntryDocumentInto(
		entry,
		compiledTeamSchema,
		&value,
	); err != nil {
		return TeamDocument{}, err
	}
	if err := value.validateFields(); err != nil {
		return TeamDocument{}, err
	}
	return value, nil
}

func DecodeAdmittedTeamJSON(
	raw []byte,
) (TeamDocument, error) {
	entry, err := declaration.DecodeCanonicalEntryJSON(raw)
	if err != nil {
		return TeamDocument{}, err
	}
	return DecodeAdmittedTeamEntry(entry)
}

func DecodeAdmittedTeamEntry(
	entry declaration.Entry,
) (TeamDocument, error) {
	var value TeamDocument
	if err := declaration.DecodeAdmittedEntryInto(
		entry,
		&value,
	); err != nil {
		return TeamDocument{}, err
	}
	if err := value.validateFields(); err != nil {
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
	if err := value.validateFields(); err != nil {
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
	return v.validateFields()
}

func (v TeamDocument) validateFields() error {
	if err := v.Header.Validate(declaration.HeaderValidation{
		ExpectedType: TeamType,
		RequireName:  true,
	}); err != nil {
		return err
	}
	if err := declaration.ValidateDeclarationLocatorExclusivity(
		"Team",
		v.Locator,
		v.Members != nil || v.Loop != nil || v.Workflow != nil,
	); err != nil {
		return err
	}
	if err := declaration.ValidateMembersWithoutRelationshipBehavior(
		"Team members",
		v.Members,
		declaration.TypeAgent,
		declaration.TypePlugin,
	); err != nil {
		return err
	}
	if v.Loop != nil && v.Workflow != nil {
		return fmt.Errorf(
			"%w: Team cannot contain both loop and workflow",
			spec.ErrInvalid,
		)
	}
	if v.Loop != nil {
		if err := validateProgramMember("Team loop", *v.Loop, declaration.TypeLoop); err != nil {
			return err
		}
	}
	if v.Workflow != nil {
		if err := validateProgramMember(
			"Team workflow",
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
			spec.ErrInvalid,
			label,
			expected,
		)
	}
	return declaration.ValidateNoRelationshipBehavior(
		label,
		value,
	)
}
