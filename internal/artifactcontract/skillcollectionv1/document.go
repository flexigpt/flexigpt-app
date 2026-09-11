package skillcollectionv1

import (
	_ "embed"
	"errors"
	"fmt"
	"path"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	SkillCollectionKind          = "skill.bundle"
	SkillCollectionSchemaID      = "skill.bundle.v1"
	SkillCollectionSchemaVersion = "v1"

	SkillCollectionMemberFormat    = "agent.skill-entrypoint/v1"
	SkillCollectionMemberRole      = "agent.skill"
	SkillCollectionMemberMediaType = "text/markdown"
)

//go:embed skill-collection-v1.schema.json
var schemaJSON []byte

var compiledSkillCollectionSchema = jsonutil.MustCompileJSONSchema(
	schemaJSON,
)

var SkillCollectionSchemaKey = schema.CollectionKey(
	collection.CollectionKind(SkillCollectionKind),
	schema.SchemaID(SkillCollectionSchemaID),
	SkillCollectionSchemaVersion,
)

func SkillCollectionJSONSchema() []byte {
	return append([]byte(nil), schemaJSON...)
}

type SkillCollectionDocument struct {
	Digest         *string                 `json:"digest,omitempty"`
	Kind           string                  `json:"kind"`
	SchemaID       string                  `json:"schemaID"`
	SchemaVersion  string                  `json:"schemaVersion"`
	LogicalName    string                  `json:"logicalName"`
	LogicalVersion string                  `json:"logicalVersion,omitempty"`
	DisplayName    string                  `json:"displayName,omitempty"`
	Description    string                  `json:"description,omitempty"`
	Labels         map[string]string       `json:"labels,omitempty"`
	Body           SkillCollectionBody     `json:"body"`
	Members        []SkillCollectionMember `json:"members,omitempty"`
}

type SkillCollectionBody struct {
	MemberFormat string `json:"memberFormat"`
}

type SkillCollectionMember struct {
	Locator   string  `json:"locator,omitempty"`
	URI       string  `json:"uri,omitempty"`
	Digest    *string `json:"digest,omitempty"`
	MediaType string  `json:"mediaType"`
	Role      string  `json:"role"`
}

func DecodeSkillCollectionJSON(raw []byte) (SkillCollectionDocument, error) {
	return jsonutil.DecodeCanonicalObject[SkillCollectionDocument](
		raw,
		basespec.MaxDefinitionBytes,
	)
}

func (v SkillCollectionDocument) Clone() (SkillCollectionDocument, error) {
	return jsonutil.CloneJSON(v)
}

func (v SkillCollectionDocument) Canonicalize() (
	SkillCollectionDocument,
	error,
) {
	output, err := jsonutil.CloneJSON(v)
	if err != nil {
		return SkillCollectionDocument{}, err
	}

	sort.Slice(output.Members, func(left, right int) bool {
		return output.Members[left].Locator+"\x00"+output.Members[left].URI <
			output.Members[right].Locator+"\x00"+output.Members[right].URI
	})
	return output, nil
}

func (v SkillCollectionDocument) CanonicalJSON() ([]byte, error) {
	return jsonutil.MarshalCanonicalObject(v, basespec.MaxDefinitionBytes)
}

func (v SkillCollectionDocument) CalculatedDigest() (
	cryptoutil.Digest,
	error,
) {
	payload := v
	payload.Digest = nil
	return cryptoutil.CanonicalDigest(payload)
}

func (v SkillCollectionDocument) Validate() error {
	if err := jsonutil.ValidateJSONSchema(
		compiledSkillCollectionSchema,
		v,
		basespec.MaxDefinitionBytes,
	); err != nil {
		return fmt.Errorf("skill collection schema: %w", err)
	}
	if err := basespec.ValidatePortableMetadata(
		basespec.LogicalName(v.LogicalName),
		basespec.LogicalVersion(v.LogicalVersion),
		v.DisplayName,
		v.Description,
		v.Labels,
		v.Digest,
	); err != nil {
		return err
	}
	if _, err := jsonutil.MarshalCanonicalObject(
		v.Body,
		basespec.MaxDefinitionBodyBytes,
	); err != nil {
		return fmt.Errorf("invalid skill collection document: body: %w", err)
	}

	seen := make(map[string]struct{}, len(v.Members))
	for index, member := range v.Members {
		if err := basespec.ValidateContentReference(
			member.Locator,
			member.URI,
			member.Digest,
		); err != nil {
			return fmt.Errorf("skill collection members[%d]: %w", index, err)
		}
		if member.Locator != "" &&
			(path.Base(member.Locator) != "SKILL.md" ||
				path.Dir(member.Locator) == ".") {
			return errors.New("invalid skill collection document: member locator must identify a packaged SKILL.md")
		}

		identity := member.Locator + "\x00" + member.URI
		if _, exists := seen[identity]; exists {
			return fmt.Errorf("invalid skill collection document: duplicate member %d", index)
		}
		seen[identity] = struct{}{}
	}
	return nil
}
