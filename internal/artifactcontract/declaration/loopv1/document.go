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
	LoopSchemaVersion = declaration.APIVersionV1
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
	return decodeLoop(raw, true, false, false)
}

func DecodeLoopEntry(
	entry declaration.Entry,
	implicitBody bool,
) (LoopDocument, error) {
	var value LoopDocument
	if err := entry.DecodeInto(&value); err != nil {
		return LoopDocument{}, err
	}
	if err := value.validate(false, implicitBody, entry.IsSymbolic()); err != nil {
		return LoopDocument{}, err
	}
	return value, nil
}

func decodeLoop(
	raw []byte,
	requireName bool,
	implicitBody bool,
	symbolic bool,
) (LoopDocument, error) {
	var value LoopDocument
	if err := declaration.DecodeDocumentInto(
		raw,
		compiledLoopSchema,
		&value,
	); err != nil {
		return LoopDocument{}, err
	}
	if err := value.validate(requireName, implicitBody, symbolic); err != nil {
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
	return v.validate(true, false, false)
}

func (v LoopDocument) ValidateEntry(
	implicitBody bool,
) error {
	return v.validate(false, implicitBody, false)
}

func (v LoopDocument) validate(
	requireName bool,
	implicitBody bool,
	symbolic bool,
) error {
	if err := declaration.ValidateDocument(
		compiledLoopSchema,
		v,
	); err != nil {
		return fmt.Errorf("loop schema: %w", err)
	}
	if err := v.Header.Validate(declaration.HeaderValidation{
		ExpectedType: LoopType,
		APIVersion:   LoopSchemaVersion,
		RequireName:  requireName,
	}); err != nil {
		return err
	}
	if v.MaxIterations < 0 {
		return fmt.Errorf(
			"%w: Loop maxIterations cannot be negative",
			basespec.ErrInvalid,
		)
	}
	if v.Body != nil {
		if err := v.Body.Validate(); err != nil {
			return fmt.Errorf("loop body: %w", err)
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
		!implicitBody &&
		!symbolic {
		return fmt.Errorf(
			"%w: inline standalone Loop requires body",
			basespec.ErrInvalid,
		)
	}
	return nil
}
