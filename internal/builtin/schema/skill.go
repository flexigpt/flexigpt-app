package schema

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	SkillCollectionV1Kind          = "skill.bundle"
	SkillCollectionV1SchemaID      = "skill.bundle.v1"
	SkillCollectionV1SchemaVersion = "v1"
	SkillCollectionV1MemberFormat  = "agent.skill-entrypoint/v1"
	SkillCollectionV1MemberRole    = "agent.skill"
)

//go:embed skill-collection-v1.schema.json
var skillCollectionV1JSONSchema []byte

func SkillCollectionV1JSONSchema() []byte {
	return append([]byte(nil), skillCollectionV1JSONSchema...)
}

// SkillCollectionV1 is the public Go wire schema for collection.json.
//
// It intentionally uses ordinary strings and json.RawMessage rather than
// internal Artifact Store value types. Semantic conversion and validation are
// supplied by the registered artifact-family codec.
type SkillCollectionV1 struct {
	Digest         *string           `json:"digest,omitempty"`
	Kind           string            `json:"kind"`
	SchemaID       string            `json:"schemaID"`
	SchemaVersion  string            `json:"schemaVersion"`
	LogicalName    string            `json:"logicalName"`
	LogicalVersion string            `json:"logicalVersion,omitempty"`
	DisplayName    string            `json:"displayName,omitempty"`
	Description    string            `json:"description,omitempty"`
	Labels         map[string]string `json:"labels,omitempty"`
	Body           json.RawMessage   `json:"body"`
	Members        []ContentRef      `json:"members,omitempty"`
}

type ContentRef struct {
	Locator            string  `json:"locator,omitempty"`
	URI                string  `json:"uri,omitempty"`
	SubresourceLocator string  `json:"subresourceLocator,omitempty"`
	Digest             *string `json:"digest,omitempty"`
	MediaType          string  `json:"mediaType,omitempty"`
	Role               string  `json:"role,omitempty"`
}

func ParseSkillCollectionV1(raw []byte) (SkillCollectionV1, error) {
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxDefinitionBytes,
	)
	if err != nil {
		return SkillCollectionV1{}, err
	}

	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()

	var value SkillCollectionV1
	if err := decoder.Decode(&value); err != nil {
		return SkillCollectionV1{}, fmt.Errorf(
			"%w: decode skill collection v1: %w",
			basespec.ErrInvalid,
			err,
		)
	}

	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("skill collection JSON contains trailing values")
		}
		return SkillCollectionV1{}, fmt.Errorf(
			"%w: decode skill collection v1: %w",
			basespec.ErrInvalid,
			err,
		)
	}

	if err := value.ValidateEnvelope(); err != nil {
		return SkillCollectionV1{}, err
	}
	return value.Clone(), nil
}

func MarshalSkillCollectionV1(value SkillCollectionV1) ([]byte, error) {
	value = value.Clone()
	if err := value.ValidateEnvelope(); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return jsonutil.CanonicalizeObject(raw, basespec.MaxDefinitionBytes)
}

func (v SkillCollectionV1) ValidateEnvelope() error {
	if v.Kind != SkillCollectionV1Kind {
		return fmt.Errorf(
			"%w: skill collection kind must be %q",
			basespec.ErrInvalid,
			SkillCollectionV1Kind,
		)
	}
	if v.SchemaID != SkillCollectionV1SchemaID {
		return fmt.Errorf(
			"%w: skill collection schema ID must be %q",
			basespec.ErrInvalid,
			SkillCollectionV1SchemaID,
		)
	}
	if v.SchemaVersion != SkillCollectionV1SchemaVersion {
		return fmt.Errorf(
			"%w: skill collection schema version must be %q",
			basespec.ErrInvalid,
			SkillCollectionV1SchemaVersion,
		)
	}
	if err := basespec.ValidateLogicalName(
		basespec.LogicalName(v.LogicalName),
	); err != nil {
		return err
	}
	if err := basespec.ValidateLogicalVersion(
		basespec.LogicalVersion(v.LogicalVersion),
		true,
	); err != nil {
		return err
	}
	if err := basespec.ValidateOptionalText(
		"skill collection display name",
		v.DisplayName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateOptionalText(
		"skill collection description",
		v.Description,
		basespec.MaxDescriptionBytes,
	); err != nil {
		return err
	}
	if _, err := jsonutil.CanonicalizeObject(
		v.Body,
		basespec.MaxDefinitionBodyBytes,
	); err != nil {
		return fmt.Errorf("%w: skill collection body: %w", basespec.ErrInvalid, err)
	}
	return nil
}

func (v SkillCollectionV1) Clone() SkillCollectionV1 {
	output := v
	output.Labels = maps.Clone(v.Labels)
	output.Body = append(json.RawMessage(nil), v.Body...)
	if v.Digest != nil {
		digest := *v.Digest
		output.Digest = &digest
	}
	if v.Members != nil {
		output.Members = make([]ContentRef, len(v.Members))
		for index, member := range v.Members {
			output.Members[index] = member.Clone()
		}
	}
	return output
}

func (v ContentRef) Clone() ContentRef {
	output := v
	if v.Digest != nil {
		digest := *v.Digest
		output.Digest = &digest
	}
	return output
}
