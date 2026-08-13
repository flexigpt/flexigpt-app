package artifactbuiltin

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/url"
	"path"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	SkillCollectionV1FileName      = "collection.json"
	SkillCollectionV1Kind          = "skill.bundle"
	SkillCollectionV1SchemaID      = "skill.bundle.v1"
	SkillCollectionV1SchemaVersion = "v1"
	SkillCollectionV1MemberFormat  = "agent.skill-entrypoint/v1"
	SkillCollectionV1MemberRole    = "agent.skill"
)

const (
	maxSkillCollectionV1Members      = 100_000
	maxSkillCollectionV1MediaTypeLen = 256
)

//go:embed skill-collection-v1.schema.json
var skillCollectionV1JSONSchema []byte

func SkillCollectionV1JSONSchema() []byte {
	return append([]byte(nil), skillCollectionV1JSONSchema...)
}

type skillCollectionV1Body struct {
	MemberFormat string `json:"memberFormat"`
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

// ParseSkillCollectionV1 validates structural and semantic input but does not
// calculate or verify a supplied document digest. Call
// CanonicalizeSkillCollectionV1 when the digest must be calculated or
// verified.
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

	if err := value.Validate(); err != nil {
		return SkillCollectionV1{}, err
	}
	return value.Clone(), nil
}

func MarshalSkillCollectionV1(value SkillCollectionV1) ([]byte, error) {
	canonical, err := CanonicalizeSkillCollectionV1(value)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(canonical)
	if err != nil {
		return nil, err
	}
	return jsonutil.CanonicalizeObject(raw, basespec.MaxDefinitionBytes)
}

// CanonicalizeSkillCollectionV1 validates a shareable collection document,
// deterministically orders its members, calculates its digest, and verifies a
// supplied digest when one is present.
//
// Member digests intentionally remain optional. They can be supplied by a
// sharer or enriched later by package hydration when member bytes exist.
func CanonicalizeSkillCollectionV1(
	input SkillCollectionV1,
) (SkillCollectionV1, error) {
	output := input.Clone()

	body, err := canonicalSkillCollectionV1Body(output.Body)
	if err != nil {
		return SkillCollectionV1{}, err
	}
	output.Body = json.RawMessage(body)

	sort.Slice(output.Members, func(left, right int) bool {
		return skillCollectionV1MemberIdentity(output.Members[left]) <
			skillCollectionV1MemberIdentity(output.Members[right])
	})

	if err := output.Validate(); err != nil {
		return SkillCollectionV1{}, err
	}

	supplied := ""
	if output.Digest != nil {
		supplied = *output.Digest
	}

	calculated, err := skillCollectionV1Digest(output)
	if err != nil {
		return SkillCollectionV1{}, err
	}
	if supplied != "" && supplied != string(calculated) {
		return SkillCollectionV1{}, fmt.Errorf(
			"%w: supplied skill collection digest %q, calculated %q",
			basespec.ErrDigestMismatch,
			supplied,
			calculated,
		)
	}

	digest := string(calculated)
	output.Digest = &digest
	if err := output.Validate(); err != nil {
		return SkillCollectionV1{}, err
	}
	return output.Clone(), nil
}

// Validate accepts both shareable input documents without a digest and
// canonical documents with a digest. CanonicalizeSkillCollectionV1 additionally
// calculates and verifies the digest.
func (v SkillCollectionV1) Validate() error {
	if err := v.ValidateEnvelope(); err != nil {
		return err
	}
	if err := ValidateShareableCollectionMetadata(
		v.LogicalName,
		v.LogicalVersion,
		v.DisplayName,
		v.Description,
		v.Labels,
	); err != nil {
		return err
	}
	if v.Digest != nil {
		if err := cryptoutil.ValidateDigest(
			cryptoutil.Digest(*v.Digest),
		); err != nil {
			return fmt.Errorf("skill collection digest: %w", err)
		}
	}
	if _, err := canonicalSkillCollectionV1Body(v.Body); err != nil {
		return err
	}
	if len(v.Members) > maxSkillCollectionV1Members {
		return fmt.Errorf(
			"%w: skill collection members exceed %d entries",
			basespec.ErrInvalid,
			maxSkillCollectionV1Members,
		)
	}

	seen := make(map[string]struct{}, len(v.Members))
	for index, member := range v.Members {
		if err := validateSkillCollectionV1Member(member); err != nil {
			return fmt.Errorf("skill collection members[%d]: %w", index, err)
		}
		identity := skillCollectionV1MemberIdentity(member)
		if _, duplicate := seen[identity]; duplicate {
			return fmt.Errorf(
				"%w: duplicate skill collection member %d",
				basespec.ErrInvalid,
				index,
			)
		}
		seen[identity] = struct{}{}
	}
	return nil
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

func canonicalSkillCollectionV1Body(
	raw json.RawMessage,
) ([]byte, error) {
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: skill collection body: %w",
			basespec.ErrInvalid,
			err,
		)
	}

	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()

	var body skillCollectionV1Body
	if err := decoder.Decode(&body); err != nil {
		return nil, fmt.Errorf(
			"%w: decode skill collection body: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("skill collection body contains trailing JSON")
		}
		return nil, fmt.Errorf(
			"%w: decode skill collection body: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if body.MemberFormat != SkillCollectionV1MemberFormat {
		return nil, fmt.Errorf(
			"%w: unsupported skill collection member format %q",
			basespec.ErrInvalid,
			body.MemberFormat,
		)
	}
	return canonical, nil
}

func validateSkillCollectionV1Member(value ContentRef) error {
	switch {
	case value.Locator != "" && value.URI != "":
		return fmt.Errorf(
			"%w: skill collection member cannot contain both locator and URI",
			basespec.ErrInvalid,
		)

	case value.Locator != "":
		locator := basespec.Locator(value.Locator)
		if err := basespec.ValidatePortableLocator(locator, false); err != nil {
			return err
		}
		if path.Base(value.Locator) != "SKILL.md" ||
			path.Dir(value.Locator) == "." {
			return fmt.Errorf(
				"%w: skill collection member locator must identify a packaged SKILL.md",
				basespec.ErrInvalid,
			)
		}

	case value.URI != "":
		if err := validateSkillCollectionV1URI(value.URI); err != nil {
			return err
		}

	default:
		return fmt.Errorf(
			"%w: skill collection member requires a locator or URI",
			basespec.ErrInvalid,
		)
	}

	if value.SubresourceLocator != "" {
		return fmt.Errorf(
			"%w: skill collection members cannot use subresource locators",
			basespec.ErrInvalid,
		)
	}
	if value.Digest != nil {
		if err := cryptoutil.ValidateDigest(
			cryptoutil.Digest(*value.Digest),
		); err != nil {
			return err
		}
	}
	if value.MediaType != "text/markdown" {
		return fmt.Errorf(
			"%w: skill collection member media type must be %q",
			basespec.ErrInvalid,
			"text/markdown",
		)
	}
	if len(value.MediaType) > maxSkillCollectionV1MediaTypeLen {
		return fmt.Errorf(
			"%w: skill collection member media type exceeds %d bytes",
			basespec.ErrInvalid,
			maxSkillCollectionV1MediaTypeLen,
		)
	}
	if value.Role != SkillCollectionV1MemberRole {
		return fmt.Errorf(
			"%w: skill collection member role must be %q",
			basespec.ErrInvalid,
			SkillCollectionV1MemberRole,
		)
	}
	return nil
}

func validateSkillCollectionV1URI(value string) error {
	if err := basespec.ValidateRequiredText(
		"skill collection member URI",
		value,
		basespec.MaxURIBytes,
	); err != nil {
		return err
	}
	if strings.Contains(value, "#") {
		return fmt.Errorf(
			"%w: skill collection member URI cannot contain a fragment",
			basespec.ErrInvalid,
		)
	}

	parsed, err := url.ParseRequestURI(value)
	if err != nil {
		return fmt.Errorf(
			"%w: invalid skill collection member URI: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if !parsed.IsAbs() || parsed.Scheme == "" || parsed.User != nil ||
		parsed.Fragment != "" || strings.EqualFold(parsed.Scheme, "file") {
		return fmt.Errorf(
			"%w: invalid skill collection member URI",
			basespec.ErrInvalid,
		)
	}
	return nil
}

func skillCollectionV1MemberIdentity(value ContentRef) string {
	return value.Locator + "\x00" +
		value.URI + "\x00" +
		value.SubresourceLocator
}

func skillCollectionV1Digest(
	value SkillCollectionV1,
) (cryptoutil.Digest, error) {
	payload := struct {
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
	}{
		Kind:           value.Kind,
		SchemaID:       value.SchemaID,
		SchemaVersion:  value.SchemaVersion,
		LogicalName:    value.LogicalName,
		LogicalVersion: value.LogicalVersion,
		DisplayName:    value.DisplayName,
		Description:    value.Description,
		Labels:         value.Labels,
		Body:           value.Body,
		Members:        value.Members,
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal skill collection digest payload: %w", err)
	}
	canonical, err := jsonutil.Canonicalize(raw)
	if err != nil {
		return "", fmt.Errorf(
			"canonicalize skill collection digest payload: %w",
			err,
		)
	}
	if len(canonical) > basespec.MaxDefinitionBytes {
		return "", fmt.Errorf(
			"%w: canonical skill collection exceeds %d bytes",
			basespec.ErrInvalid,
			basespec.MaxDefinitionBytes,
		)
	}
	return cryptoutil.DigestBytes(canonical), nil
}
