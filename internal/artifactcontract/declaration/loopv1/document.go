package loopv1

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
	LoopType          = declaration.TypeLoop
	LoopSchemaID      = "artifact.loop.v1"
	LoopSchemaVersion = declaration.SchemaVersionV1
)

//go:embed loop-v1.schema.json
var schemaJSON []byte

var compiledLoopSchema = jsonutil.MustCompileJSONSchema(schemaJSON)

var LoopSchemaKey = schema.ArtifactKey(
	artifact.ArtifactKind(LoopType),
	schema.SchemaID(LoopSchemaID),
	LoopSchemaVersion,
)

type LoopDocument struct {
	declaration.Header

	Body          *declaration.Entry       `json:"body,omitempty"`
	MaxIterations int                      `json:"maxIterations,omitempty"`
	Until         *declaration.OutputMatch `json:"until,omitempty"`
}

func LoopJSONSchema() []byte {
	return append([]byte(nil), schemaJSON...)
}

func DecodeLoopJSON(raw []byte) (LoopDocument, error) {
	return decodeLoop(raw)
}

func DecodeLoopEntry(
	entry declaration.Entry,
) (LoopDocument, error) {
	var value LoopDocument
	if err := declaration.DecodeEntryDocumentInto(
		entry,
		compiledLoopSchema,
		&value,
	); err != nil {
		return LoopDocument{}, err
	}
	if err := value.validateFields(); err != nil {
		return LoopDocument{}, err
	}
	return value, nil
}

func decodeLoop(
	raw []byte,
) (LoopDocument, error) {
	var value LoopDocument
	if err := declaration.DecodeDocumentInto(
		raw,
		compiledLoopSchema,
		&value,
	); err != nil {
		return LoopDocument{}, err
	}
	if err := value.validateFields(); err != nil {
		return LoopDocument{}, err
	}
	return value, nil
}

func (v LoopDocument) Clone() (LoopDocument, error) {
	return jsonutil.CloneJSON(v)
}

func (v LoopDocument) Canonicalize() (LoopDocument, error) {
	return v.Clone()
}

func (v LoopDocument) CanonicalJSON() ([]byte, error) {
	if err := v.Validate(); err != nil {
		return nil, err
	}
	return declaration.CanonicalDocumentJSON(v)
}

func (v LoopDocument) CalculatedDigest() (
	cryptoutil.Digest,
	error,
) {
	return declaration.DocumentDigest(v)
}

func (v LoopDocument) Validate() error {
	return v.validate()
}

func (v LoopDocument) ValidateEntry() error {
	return v.validate()
}

func (v LoopDocument) validate() error {
	if err := declaration.ValidateDocument(compiledLoopSchema, v); err != nil {
		return fmt.Errorf("loop schema: %w", err)
	}
	return v.validateFields()
}

func (v LoopDocument) validateFields() error {
	if err := v.Header.Validate(declaration.HeaderValidation{
		ExpectedType: LoopType,
		RequireName:  true,
	}); err != nil {
		return err
	}
	if err := declaration.ValidateDeclarationLocatorExclusivity(
		"Loop",
		v.Locator,
		v.Body != nil ||
			v.MaxIterations != 0 ||
			v.Until != nil,
	); err != nil {
		return err
	}

	if v.Body != nil {
		form, err := v.Body.MemberForm()
		if err != nil {
			return fmt.Errorf("loop body: %w", err)
		}
		if form == declaration.MemberSelector {
			return fmt.Errorf(
				"%w: Loop body cannot be a member selector",
				basespec.ErrInvalid,
			)
		}
		if err := declaration.ValidateNoRelationshipBehavior(
			"Loop body",
			*v.Body,
		); err != nil {
			return err
		}
	}
	if v.Until != nil {
		if err := declaration.ValidateOutputMatch(
			"Loop until",
			*v.Until,
		); err != nil {
			return err
		}
	}
	if v.Body == nil &&
		v.Locator == nil &&
		v.Name != "" {
		return fmt.Errorf(
			"%w: inline standalone Loop requires body",
			basespec.ErrInvalid,
		)
	}
	return nil
}
