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
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/shareable"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	WorkspaceCollectionV1Kind          basespec.CollectionKind = "workspace.collection"
	WorkspaceCollectionV1SchemaID      basespec.SchemaID       = "workspace.collection.v1"
	WorkspaceCollectionV1SchemaVersion                         = "v1"

	maxWorkspaceCollectionV1Members      = 100_000
	maxWorkspaceCollectionV1MediaTypeLen = 256
)

//go:embed workspace-collection-v1.schema.json
var workspaceCollectionV1JSONSchema []byte

var WorkspaceCollectionV1SchemaKey = shareable.SchemaKey{
	Entity:        shareable.EntityCollection,
	Kind:          WorkspaceCollectionV1Kind,
	SchemaID:      WorkspaceCollectionV1SchemaID,
	SchemaVersion: WorkspaceCollectionV1SchemaVersion,
}

func WorkspaceCollectionV1JSONSchema() []byte {
	return append([]byte(nil), workspaceCollectionV1JSONSchema...)
}

// WorkspaceCollectionV1 is the portable workspace.json schema model.
//
// Digest is optional on input. CanonicalizeWorkspaceCollectionV1 always
// calculates and returns a digest.
type WorkspaceCollectionV1 struct {
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

type WorkspaceDirectoryRootV1 struct {
	Root            string   `json:"root"`
	Recursive       bool     `json:"recursive,omitempty"`
	IncludePatterns []string `json:"includePatterns,omitempty"`
}
type WorkspaceDiscoveryV1 struct {
	AdditionalLocators []string                   `json:"additionalLocators,omitempty"`
	AdditionalRoots    []WorkspaceDirectoryRootV1 `json:"additionalRoots,omitempty"`
	IncludeReadme      bool                       `json:"includeReadme,omitempty"`
}
type WorkspaceCollectionV1Body struct {
	Discovery WorkspaceDiscoveryV1 `json:"discovery"`
}

func ParseWorkspaceCollectionV1(
	raw []byte,
) (WorkspaceCollectionV1, error) {
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxDefinitionBytes,
	)
	if err != nil {
		return WorkspaceCollectionV1{}, err
	}

	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()

	var value WorkspaceCollectionV1
	if err := decoder.Decode(&value); err != nil {
		return WorkspaceCollectionV1{}, fmt.Errorf(
			"%w: decode workspace collection v1: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("workspace collection JSON contains trailing values")
		}
		return WorkspaceCollectionV1{}, fmt.Errorf(
			"%w: decode workspace collection v1: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if err := value.Validate(); err != nil {
		return WorkspaceCollectionV1{}, err
	}
	return value.Clone(), nil
}

func MarshalWorkspaceCollectionV1(
	value WorkspaceCollectionV1,
) ([]byte, error) {
	canonical, err := CanonicalizeWorkspaceCollectionV1(value)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(canonical)
	if err != nil {
		return nil, err
	}
	return jsonutil.CanonicalizeObject(raw, basespec.MaxDefinitionBytes)
}

func EncodeWorkspaceCollectionV1Body(
	value WorkspaceCollectionV1Body,
) (json.RawMessage, error) {
	if err := validateWorkspaceDiscoveryV1(value.Discovery); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(canonical), nil
}

func DecodeWorkspaceCollectionV1Body(
	raw json.RawMessage,
) (WorkspaceCollectionV1Body, error) {
	value, _, err := decodeWorkspaceCollectionV1Body(raw)
	return value, err
}

func CanonicalizeWorkspaceCollectionV1(
	input WorkspaceCollectionV1,
) (WorkspaceCollectionV1, error) {
	output := input.Clone()

	_, body, err := decodeWorkspaceCollectionV1Body(output.Body)
	if err != nil {
		return WorkspaceCollectionV1{}, err
	}
	output.Body = json.RawMessage(body)

	sort.Slice(output.Members, func(left, right int) bool {
		return workspaceCollectionV1MemberIdentity(output.Members[left]) <
			workspaceCollectionV1MemberIdentity(output.Members[right])
	})
	if err := output.Validate(); err != nil {
		return WorkspaceCollectionV1{}, err
	}

	supplied := ""
	if output.Digest != nil {
		supplied = *output.Digest
	}
	calculated, err := workspaceCollectionV1Digest(output)
	if err != nil {
		return WorkspaceCollectionV1{}, err
	}
	if supplied != "" && supplied != string(calculated) {
		return WorkspaceCollectionV1{}, fmt.Errorf(
			"%w: supplied workspace collection digest %q, calculated %q",
			basespec.ErrDigestMismatch,
			supplied,
			calculated,
		)
	}

	digest := string(calculated)
	output.Digest = &digest
	if err := output.Validate(); err != nil {
		return WorkspaceCollectionV1{}, err
	}
	return output.Clone(), nil
}

func (v WorkspaceCollectionV1) Validate() error {
	if v.Kind != string(WorkspaceCollectionV1Kind) {
		return fmt.Errorf(
			"%w: workspace collection kind must be %q",
			basespec.ErrInvalid,
			WorkspaceCollectionV1Kind,
		)
	}
	if v.SchemaID != string(WorkspaceCollectionV1SchemaID) {
		return fmt.Errorf(
			"%w: workspace collection schema ID must be %q",
			basespec.ErrInvalid,
			WorkspaceCollectionV1SchemaID,
		)
	}
	if v.SchemaVersion != WorkspaceCollectionV1SchemaVersion {
		return fmt.Errorf(
			"%w: workspace collection schema version must be %q",
			basespec.ErrInvalid,
			WorkspaceCollectionV1SchemaVersion,
		)
	}
	if err := basespec.ValidatePortableMetadata(
		basespec.LogicalName(v.LogicalName),
		basespec.LogicalVersion(v.LogicalVersion),
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
			return fmt.Errorf("workspace collection digest: %w", err)
		}
	}
	if _, _, err := decodeWorkspaceCollectionV1Body(v.Body); err != nil {
		return err
	}
	if len(v.Members) > maxWorkspaceCollectionV1Members {
		return fmt.Errorf(
			"%w: workspace collection members exceed %d entries",
			basespec.ErrInvalid,
			maxWorkspaceCollectionV1Members,
		)
	}

	seen := make(map[string]struct{}, len(v.Members))
	for index, member := range v.Members {
		if err := validateWorkspaceCollectionV1Member(member); err != nil {
			return fmt.Errorf(
				"workspace collection members[%d]: %w",
				index,
				err,
			)
		}
		identity := workspaceCollectionV1MemberIdentity(member)
		if _, duplicate := seen[identity]; duplicate {
			return fmt.Errorf(
				"%w: duplicate workspace collection member %d",
				basespec.ErrInvalid,
				index,
			)
		}
		seen[identity] = struct{}{}
	}
	return nil
}

func (v WorkspaceCollectionV1) Clone() WorkspaceCollectionV1 {
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

func decodeWorkspaceCollectionV1Body(
	raw json.RawMessage,
) (WorkspaceCollectionV1Body, []byte, error) {
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return WorkspaceCollectionV1Body{}, nil, fmt.Errorf(
			"%w: workspace collection body: %w",
			basespec.ErrInvalid,
			err,
		)
	}

	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()
	var body WorkspaceCollectionV1Body
	if err := decoder.Decode(&body); err != nil {
		return WorkspaceCollectionV1Body{}, nil, fmt.Errorf(
			"%w: decode workspace collection body: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("workspace collection body contains trailing JSON")
		}
		return WorkspaceCollectionV1Body{}, nil, fmt.Errorf(
			"%w: decode workspace collection body: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if err := validateWorkspaceDiscoveryV1(body.Discovery); err != nil {
		return WorkspaceCollectionV1Body{}, nil, err
	}
	return body, canonical, nil
}

func validateWorkspaceDiscoveryV1(value WorkspaceDiscoveryV1) error {
	if len(value.AdditionalLocators) > basespec.MaxDiscoveryCandidates {
		return fmt.Errorf(
			"%w: workspace additional locators exceed %d entries",
			basespec.ErrInvalid,
			basespec.MaxDiscoveryCandidates,
		)
	}
	if len(value.AdditionalRoots) > basespec.MaxDiscoveryCandidates {
		return fmt.Errorf(
			"%w: workspace additional roots exceed %d entries",
			basespec.ErrInvalid,
			basespec.MaxDiscoveryCandidates,
		)
	}

	locators := make(map[basespec.Locator]struct{}, len(value.AdditionalLocators))
	for _, rawLocator := range value.AdditionalLocators {
		locator := basespec.Locator(rawLocator)
		if err := basespec.ValidatePortableLocator(locator, false); err != nil {
			return err
		}
		if _, duplicate := locators[locator]; duplicate {
			return fmt.Errorf(
				"%w: duplicate workspace additional locator %q",
				basespec.ErrInvalid,
				locator,
			)
		}
		locators[locator] = struct{}{}
	}

	roots := make(map[basespec.Locator]struct{}, len(value.AdditionalRoots))
	for index, root := range value.AdditionalRoots {
		locator := basespec.Locator(root.Root)
		if err := basespec.ValidatePortableLocator(locator, true); err != nil {
			return fmt.Errorf("workspace additional roots[%d]: %w", index, err)
		}
		if _, duplicate := roots[locator]; duplicate {
			return fmt.Errorf(
				"%w: duplicate workspace additional root %q",
				basespec.ErrInvalid,
				locator,
			)
		}
		roots[locator] = struct{}{}

		patterns := make(map[string]struct{}, len(root.IncludePatterns))
		for _, pattern := range root.IncludePatterns {
			if err := basespec.ValidateIncludePattern(pattern); err != nil {
				return err
			}
			if _, duplicate := patterns[pattern]; duplicate {
				return fmt.Errorf(
					"%w: duplicate workspace include pattern %q",
					basespec.ErrInvalid,
					pattern,
				)
			}
			patterns[pattern] = struct{}{}
		}
	}
	return nil
}

func validateWorkspaceCollectionV1Member(value ContentRef) error {
	switch {
	case value.Locator != "" && value.URI != "":
		return fmt.Errorf(
			"%w: workspace collection member cannot contain both locator and URI",
			basespec.ErrInvalid,
		)
	case value.Locator != "":
		if err := basespec.ValidatePortableLocator(
			basespec.Locator(value.Locator),
			false,
		); err != nil {
			return err
		}
	case value.URI != "":
		if err := validateWorkspaceCollectionV1URI(value.URI); err != nil {
			return err
		}
	default:
		return fmt.Errorf(
			"%w: workspace collection member requires a locator or URI",
			basespec.ErrInvalid,
		)
	}

	if value.SubresourceLocator != "" {
		return fmt.Errorf(
			"%w: workspace collection members cannot use subresource locators",
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
	if err := basespec.ValidateOptionalText(
		"workspace collection member media type",
		value.MediaType,
		maxWorkspaceCollectionV1MediaTypeLen,
	); err != nil {
		return err
	}
	if value.Role != "" {
		if err := basespec.ValidateIdentifier(
			"workspace collection member role",
			value.Role,
			basespec.MaxKindBytes,
		); err != nil {
			return err
		}
	}
	return nil
}

func validateWorkspaceCollectionV1URI(value string) error {
	if err := basespec.ValidateRequiredText(
		"workspace collection member URI",
		value,
		basespec.MaxURIBytes,
	); err != nil {
		return err
	}
	if strings.Contains(value, "#") {
		return fmt.Errorf(
			"%w: workspace collection member URI cannot contain a fragment",
			basespec.ErrInvalid,
		)
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil {
		return fmt.Errorf(
			"%w: invalid workspace collection member URI: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if !parsed.IsAbs() || parsed.Scheme == "" || parsed.User != nil ||
		parsed.Fragment != "" || strings.EqualFold(parsed.Scheme, "file") {
		return fmt.Errorf(
			"%w: invalid workspace collection member URI",
			basespec.ErrInvalid,
		)
	}
	return nil
}

func workspaceCollectionV1MemberIdentity(value ContentRef) string {
	return value.Locator + "\x00" +
		value.URI + "\x00" +
		value.SubresourceLocator
}

func workspaceCollectionV1Digest(
	value WorkspaceCollectionV1,
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
		return "", fmt.Errorf("marshal workspace collection digest payload: %w", err)
	}
	canonical, err := jsonutil.Canonicalize(raw)
	if err != nil {
		return "", fmt.Errorf(
			"canonicalize workspace collection digest payload: %w",
			err,
		)
	}
	if len(canonical) > basespec.MaxDefinitionBytes {
		return "", fmt.Errorf(
			"%w: canonical workspace collection exceeds %d bytes",
			basespec.ErrInvalid,
			basespec.MaxDefinitionBytes,
		)
	}
	return cryptoutil.DigestBytes(canonical), nil
}
