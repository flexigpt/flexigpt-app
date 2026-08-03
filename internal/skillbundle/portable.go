package skillbundle

import (
	"fmt"
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/skillartifact"
)

const (
	PortableBundleSchemaVersion                   = "v1"
	portableMemberFormat                          = "agent.skill-entrypoint/v1"
	portableSkillMediaType                        = "text/markdown"
	PortableCollectionFileName                    = "collection.json"
	PortableBundleSchemaID      basespec.SchemaID = "skill.bundle.v1"
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

// NewPortableBundleDefinition creates a canonical shareable Skill Bundle JSON
// manifest. ContentRef.Digest is the digest of the raw SKILL.md member bytes,
// not the digest of the derived Artifact Definition.
//
// Members identify package entrypoint documents; the package closure itself remains the responsibility of future
// import/export code.
func NewPortableBundleDefinition(
	metadata PortableBundleMetadata,
	members []definition.ContentRef,
) (definition.CollectionDefinition, error) {
	body, err := definition.EncodeBody(portableBundleBody{MemberFormat: portableMemberFormat})
	if err != nil {
		return definition.CollectionDefinition{}, err
	}
	return CanonicalizePortableBundleDefinition(
		definition.CollectionDefinition{
			Kind:           CollectionKind,
			SchemaID:       PortableBundleSchemaID,
			SchemaVersion:  PortableBundleSchemaVersion,
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

func ValidatePortableBundleMetadata(
	metadata PortableBundleMetadata,
) error {
	_, err := NewPortableBundleDefinition(metadata, nil)
	return err
}

func ValidatePortableBundleDefinition(
	value definition.CollectionDefinition,
) error {
	if err := value.Validate(); err != nil {
		return err
	}
	if value.Kind != CollectionKind ||
		value.SchemaID != PortableBundleSchemaID ||
		value.SchemaVersion != PortableBundleSchemaVersion {
		return fmt.Errorf(
			"%w: unsupported portable Skill Bundle schema",
			basespec.ErrInvalid,
		)
	}

	body, err := definition.DecodeBody[portableBundleBody](value.Body)
	if err != nil {
		return err
	}
	if body.MemberFormat != portableMemberFormat {
		return fmt.Errorf(
			"%w: unsupported portable Skill Bundle member format %q",
			basespec.ErrInvalid,
			body.MemberFormat,
		)
	}

	for index, member := range value.Members {
		if member.Role != string(skillartifact.Kind) ||
			member.MediaType != portableSkillMediaType ||
			member.Digest == nil ||
			member.SubresourceLocator != "" {
			return fmt.Errorf(
				"%w: portable Skill Bundle member %d is not an integrity-pinned SKILL.md member",
				basespec.ErrInvalid,
				index,
			)
		}
		if member.Locator != "" {
			if path.Base(string(member.Locator)) != skillartifact.DefinitionFileName ||
				path.Dir(string(member.Locator)) == "." {
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
	return definition.MarshalCollectionDefinition(canonical)
}

func ParsePortableBundleDefinition(
	raw []byte,
) (definition.CollectionDefinition, error) {
	value, err := definition.ParseCollectionDefinition(raw)
	if err != nil {
		return definition.CollectionDefinition{}, err
	}
	return CanonicalizePortableBundleDefinition(value)
}
