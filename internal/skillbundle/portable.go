package skillbundle

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/skillartifact"
)

const (
	PortableBundleSchemaID      basespec.SchemaID = "skill.bundle.v1"
	PortableBundleSchemaVersion basespec.SchemaID = "v1"
	portableMemberFormat        basespec.SchemaID = "agent.skill-package/v1"
	portableSkillMediaType      basespec.SchemaID = "text/markdown"
)

type PortableBundleMetadata struct {
	LogicalName    basespec.LogicalName
	LogicalVersion basespec.LogicalVersion
	DisplayName    string
	Description    string
	Labels         map[string]string
}

type portableBundleBody struct {
	MemberFormat string `json:"memberFormat"`
}

// NewPortableBundleDefinition creates a canonical shareable Skill Bundle
// manifest. Members identify package SKILL.md documents; the package closure
// itself remains the responsibility of future import/export code.
func NewPortableBundleDefinition(
	metadata PortableBundleMetadata,
	members []definition.ContentRef,
) (definition.CollectionDefinition, error) {
	body, err := json.Marshal(portableBundleBody{
		MemberFormat: string(portableMemberFormat),
	})
	if err != nil {
		return definition.CollectionDefinition{}, err
	}
	return CanonicalizePortableBundleDefinition(
		definition.CollectionDefinition{
			Kind:           CollectionKind,
			SchemaID:       PortableBundleSchemaID,
			SchemaVersion:  string(PortableBundleSchemaVersion),
			LogicalName:    metadata.LogicalName,
			LogicalVersion: metadata.LogicalVersion,
			DisplayName:    metadata.DisplayName,
			Description:    metadata.Description,
			Labels:         metadata.Labels,
			Body:           body,
			Members:        members,
		},
	)
}

func CanonicalizePortableBundleDefinition(
	input definition.CollectionDefinition,
) (definition.CollectionDefinition, error) {
	canonical, err := definition.CanonicalizeCollectionDefinition(input)
	if err != nil {
		return definition.CollectionDefinition{}, err
	}
	if err := ValidatePortableBundleDefinition(canonical); err != nil {
		return definition.CollectionDefinition{}, err
	}
	return canonical, nil
}

func ValidatePortableBundleDefinition(
	value definition.CollectionDefinition,
) error {
	if err := value.Validate(); err != nil {
		return err
	}
	if value.Kind != CollectionKind ||
		value.SchemaID != PortableBundleSchemaID ||
		value.SchemaVersion != string(PortableBundleSchemaVersion) {
		return fmt.Errorf(
			"%w: unsupported portable Skill Bundle schema",
			basespec.ErrInvalid,
		)
	}

	var body portableBundleBody
	decoder := json.NewDecoder(bytes.NewReader(value.Body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		return fmt.Errorf("%w: decode portable Skill Bundle body: %w", basespec.ErrInvalid, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: portable Skill Bundle body has trailing JSON", basespec.ErrInvalid)
	}
	if body.MemberFormat != string(portableMemberFormat) {
		return fmt.Errorf(
			"%w: unsupported portable Skill Bundle member format %q",
			basespec.ErrInvalid,
			body.MemberFormat,
		)
	}

	for index, member := range value.Members {
		if member.Role != string(skillartifact.Kind) ||
			member.MediaType != string(portableSkillMediaType) ||
			member.Digest == nil ||
			member.SubresourceLocator != "" {
			return fmt.Errorf(
				"%w: portable Skill Bundle member %d is not an integrity-pinned SKILL.md member",
				basespec.ErrInvalid,
				index,
			)
		}
		if member.Locator != "" {
			if !strings.EqualFold(
				path.Base(string(member.Locator)),
				skillartifact.DefinitionFileName,
			) || path.Dir(string(member.Locator)) == "." {
				return fmt.Errorf(
					"%w: portable Skill Bundle member %d must locate a package %q",
					basespec.ErrInvalid,
					index,
					skillartifact.DefinitionFileName,
				)
			}
		}
	}
	return nil
}

func MarshalPortableBundleDefinition(
	input definition.CollectionDefinition,
) ([]byte, error) {
	canonical, err := CanonicalizePortableBundleDefinition(input)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(canonical)
	if err != nil {
		return nil, err
	}
	return jsonutil.Canonicalize(raw)
}

func ParsePortableBundleDefinition(
	raw []byte,
) (definition.CollectionDefinition, error) {
	canonicalJSON, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxDefinitionBytes,
	)
	if err != nil {
		return definition.CollectionDefinition{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(canonicalJSON))
	decoder.DisallowUnknownFields()
	var value definition.CollectionDefinition
	if err := decoder.Decode(&value); err != nil {
		return definition.CollectionDefinition{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return definition.CollectionDefinition{}, fmt.Errorf(
			"%w: portable Skill Bundle JSON has trailing values",
			basespec.ErrInvalid,
		)
	}
	return CanonicalizePortableBundleDefinition(value)
}
