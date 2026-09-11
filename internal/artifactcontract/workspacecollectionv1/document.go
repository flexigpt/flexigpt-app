package workspacecollectionv1

import (
	_ "embed"
	"errors"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/schema"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	WorkspaceCollectionKind          = "workspace.collection"
	WorkspaceCollectionSchemaID      = "workspace.collection.v1"
	WorkspaceCollectionSchemaVersion = "v1"
)

var errInvalid = errors.New("invalid workspace collection document")

//go:embed workspace-collection-v1.schema.json
var schemaJSON []byte

var compiledWorkspaceCollectionSchema = jsonutil.MustCompileJSONSchema(
	schemaJSON,
)

var WorkspaceCollectionSchemaKey = schema.CollectionKey(
	collection.CollectionKind(WorkspaceCollectionKind),
	schema.SchemaID(WorkspaceCollectionSchemaID),
	WorkspaceCollectionSchemaVersion,
)

func WorkspaceCollectionJSONSchema() []byte {
	return append([]byte(nil), schemaJSON...)
}

type WorkspaceCollectionDocument struct {
	Digest         *string                     `json:"digest,omitempty"`
	Kind           string                      `json:"kind"`
	SchemaID       string                      `json:"schemaID"`
	SchemaVersion  string                      `json:"schemaVersion"`
	LogicalName    string                      `json:"logicalName"`
	LogicalVersion string                      `json:"logicalVersion,omitempty"`
	DisplayName    string                      `json:"displayName,omitempty"`
	Description    string                      `json:"description,omitempty"`
	Labels         map[string]string           `json:"labels,omitempty"`
	Body           WorkspaceCollectionBody     `json:"body"`
	Members        []WorkspaceCollectionMember `json:"members,omitempty"`
}

type WorkspaceCollectionBody struct {
	Discovery WorkspaceCollectionDiscovery `json:"discovery"`
}

type WorkspaceCollectionDiscovery struct {
	AdditionalLocators []string                           `json:"additionalLocators,omitempty"`
	AdditionalRoots    []WorkspaceCollectionDirectoryRoot `json:"additionalRoots,omitempty"`
	IncludeReadme      *bool                              `json:"includeReadme,omitempty"`
}

type WorkspaceCollectionDirectoryRoot struct {
	Root            string   `json:"root"`
	Recursive       *bool    `json:"recursive,omitempty"`
	IncludePatterns []string `json:"includePatterns,omitempty"`
}

type WorkspaceCollectionMember struct {
	Locator   string  `json:"locator,omitempty"`
	URI       string  `json:"uri,omitempty"`
	Digest    *string `json:"digest,omitempty"`
	MediaType string  `json:"mediaType,omitempty"`
	Role      string  `json:"role,omitempty"`
}

func DecodeWorkspaceCollectionJSON(
	raw []byte,
) (WorkspaceCollectionDocument, error) {
	return jsonutil.DecodeCanonicalObject[WorkspaceCollectionDocument](
		raw,
		basespec.MaxDefinitionBytes,
	)
}

func (v WorkspaceCollectionDocument) Clone() (
	WorkspaceCollectionDocument,
	error,
) {
	return jsonutil.CloneJSON(v)
}

func (v WorkspaceCollectionDocument) Canonicalize() (
	WorkspaceCollectionDocument,
	error,
) {
	output, err := jsonutil.CloneJSON(v)
	if err != nil {
		return WorkspaceCollectionDocument{}, err
	}

	sort.Slice(output.Members, func(left, right int) bool {
		return output.Members[left].Locator+"\x00"+output.Members[left].URI <
			output.Members[right].Locator+"\x00"+output.Members[right].URI
	})
	return output, nil
}

func (v WorkspaceCollectionDocument) CanonicalJSON() ([]byte, error) {
	return jsonutil.MarshalCanonicalObject(v, basespec.MaxDefinitionBytes)
}

func (v WorkspaceCollectionDocument) CalculatedDigest() (
	cryptoutil.Digest,
	error,
) {
	payload := v
	payload.Digest = nil
	return cryptoutil.CanonicalDigest(payload)
}

func (v WorkspaceCollectionDocument) Validate() error {
	if err := jsonutil.ValidateJSONSchema(
		compiledWorkspaceCollectionSchema,
		v,
		basespec.MaxDefinitionBytes,
	); err != nil {
		return fmt.Errorf("workspace collection schema: %w", err)
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
		return fmt.Errorf("%w: body: %w", errInvalid, err)
	}
	if err := validateWorkspaceCollectionDiscovery(v.Body.Discovery); err != nil {
		return err
	}

	seen := make(map[string]struct{}, len(v.Members))
	for index, member := range v.Members {
		if err := basespec.ValidateContentReference(
			member.Locator,
			member.URI,
			member.Digest,
		); err != nil {
			return fmt.Errorf(
				"workspace collection members[%d]: %w",
				index,
				err,
			)
		}
		if err := basespec.ValidateOptionalText(
			"workspace collection member media type",
			member.MediaType,
			256,
		); err != nil {
			return err
		}
		if member.Role != "" {
			if err := basespec.ValidateIdentifier(
				"workspace collection member role",
				member.Role,
				basespec.MaxKindBytes,
			); err != nil {
				return err
			}
		}

		identity := member.Locator + "\x00" + member.URI
		if _, exists := seen[identity]; exists {
			return fmt.Errorf(
				"%w: duplicate member %d",
				errInvalid,
				index,
			)
		}
		seen[identity] = struct{}{}
	}
	return nil
}

func validateWorkspaceCollectionDiscovery(
	value WorkspaceCollectionDiscovery,
) error {
	if len(value.AdditionalLocators) > basespec.MaxDiscoveryCandidates {
		return fmt.Errorf(
			"%w: additional locators exceed %d entries",
			errInvalid,
			basespec.MaxDiscoveryCandidates,
		)
	}
	if len(value.AdditionalRoots) > basespec.MaxDiscoveryCandidates {
		return fmt.Errorf(
			"%w: additional roots exceed %d entries",
			errInvalid,
			basespec.MaxDiscoveryCandidates,
		)
	}

	locators := make(
		map[basespec.Locator]struct{},
		len(value.AdditionalLocators),
	)
	for _, rawLocator := range value.AdditionalLocators {
		locator := basespec.Locator(rawLocator)
		if err := locator.ValidatePortable(false); err != nil {
			return err
		}
		if _, exists := locators[locator]; exists {
			return fmt.Errorf(
				"%w: duplicate additional locator %q",
				errInvalid,
				locator,
			)
		}
		locators[locator] = struct{}{}
	}

	roots := make(
		map[basespec.Locator]struct{},
		len(value.AdditionalRoots),
	)
	for index, root := range value.AdditionalRoots {
		locator := basespec.Locator(root.Root)
		if err := locator.ValidatePortable(true); err != nil {
			return fmt.Errorf(
				"workspace additional roots[%d]: %w",
				index,
				err,
			)
		}
		if _, exists := roots[locator]; exists {
			return fmt.Errorf(
				"%w: duplicate additional root %q",
				errInvalid,
				locator,
			)
		}
		roots[locator] = struct{}{}

		patterns := make(map[string]struct{}, len(root.IncludePatterns))
		for _, pattern := range root.IncludePatterns {
			if err := basespec.ValidateIncludePattern(pattern); err != nil {
				return err
			}
			if _, exists := patterns[pattern]; exists {
				return fmt.Errorf(
					"%w: duplicate include pattern %q",
					errInvalid,
					pattern,
				)
			}
			patterns[pattern] = struct{}{}
		}
	}
	return nil
}
