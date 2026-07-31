package definition

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	MaxCollectionMembers      = 100_000
	MaxPortableMediaTypeBytes = 256
)

// ContentRef identifies portable member content. Locator is relative to the
// containing document or package root. URI is an external portable acquisition
// reference and must be resolved by an explicit policy-controlled resolver.
//
// When Locator and URI are empty, SubresourceLocator identifies embedded
// content in the containing document.
type ContentRef struct {
	Locator            basespec.Locator            `json:"locator,omitempty"`
	URI                string                      `json:"uri,omitempty"`
	SubresourceLocator basespec.SubresourceLocator `json:"subresourceLocator,omitempty"`
	Digest             *cryptoutil.Digest          `json:"digest,omitempty"`
	MediaType          string                      `json:"mediaType,omitempty"`
	Role               string                      `json:"role,omitempty"`
}

func (r ContentRef) Validate() error {
	switch {
	case r.Locator != "" && r.URI != "":
		return fmt.Errorf(
			"%w: portable content reference cannot contain both locator and URI",
			basespec.ErrInvalid,
		)

	case r.Locator != "":
		if err := basespec.ValidatePortableLocator(r.Locator, false); err != nil {
			return err
		}

	case r.URI != "":
		if err := validateURI(r.URI); err != nil {
			return err
		}

	case r.SubresourceLocator == "":
		return fmt.Errorf(
			"%w: portable content reference requires a locator, URI, or embedded subresource",
			basespec.ErrInvalid,
		)
	}

	if err := basespec.ValidatePortableSubresourceLocator(
		r.SubresourceLocator,
	); err != nil {
		return err
	}
	if r.Digest != nil {
		if err := cryptoutil.ValidateDigest(*r.Digest); err != nil {
			return err
		}
	}
	if err := basespec.ValidateOptionalText(
		"portable content media type",
		r.MediaType,
		MaxPortableMediaTypeBytes,
	); err != nil {
		return err
	}
	if r.Role != "" {
		if err := basespec.ValidateIdentifier(
			"portable content role",
			r.Role,
			basespec.MaxKindBytes,
		); err != nil {
			return err
		}
	}
	return nil
}

func (r ContentRef) Clone() ContentRef {
	output := r
	output.Digest = cryptoutil.CloneDigest(r.Digest)
	return output
}

func (r ContentRef) identity() string {
	return string(r.Locator) + "\x00" +
		r.URI + "\x00" +
		string(r.SubresourceLocator)
}

func validateURI(value string) error {
	if err := basespec.ValidateRequiredText(
		"portable content URI",
		value,
		basespec.MaxURIBytes,
	); err != nil {
		return err
	}
	if strings.Contains(value, "#") {
		return fmt.Errorf(
			"%w: portable content URI must use subresourceLocator instead of a fragment",
			basespec.ErrInvalid,
		)
	}

	parsed, err := url.ParseRequestURI(value)
	if err != nil {
		return fmt.Errorf(
			"%w: portable content URI is invalid: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if !parsed.IsAbs() || parsed.Scheme == "" {
		return fmt.Errorf(
			"%w: portable content URI must be absolute",
			basespec.ErrInvalid,
		)
	}
	if parsed.User != nil {
		return fmt.Errorf(
			"%w: portable content URI cannot contain user information",
			basespec.ErrInvalid,
		)
	}
	if parsed.Fragment != "" {
		return fmt.Errorf(
			"%w: portable content URI must use subresourceLocator instead of a fragment",
			basespec.ErrInvalid,
		)
	}
	if strings.EqualFold(parsed.Scheme, "file") {
		return fmt.Errorf(
			"%w: portable content URI cannot use the file scheme",
			basespec.ErrInvalid,
		)
	}
	return nil
}

// CollectionDefinition is the generic portable envelope for a shareable
// Collection. Its Body remains domain-owned, while Members remain visible to
// generic transfer and integrity code.
type CollectionDefinition struct {
	Digest         cryptoutil.Digest       `json:"digest,omitempty"`
	Kind           basespec.CollectionKind `json:"kind"`
	SchemaID       basespec.SchemaID       `json:"schemaID"`
	SchemaVersion  string                  `json:"schemaVersion"`
	LogicalName    basespec.LogicalName    `json:"logicalName"`
	LogicalVersion basespec.LogicalVersion `json:"logicalVersion,omitempty"`
	DisplayName    string                  `json:"displayName,omitempty"`
	Description    string                  `json:"description,omitempty"`
	Labels         map[string]string       `json:"labels,omitempty"`
	Body           json.RawMessage         `json:"body"`
	Members        []ContentRef            `json:"members,omitempty"`
}

func (d CollectionDefinition) Validate() error {
	if err := cryptoutil.ValidateDigest(d.Digest); err != nil {
		return fmt.Errorf("portable collection definition: %w", err)
	}
	if err := basespec.ValidateCollectionKind(d.Kind); err != nil {
		return fmt.Errorf("portable collection definition: %w", err)
	}
	if err := basespec.ValidateSchemaID(d.SchemaID); err != nil {
		return fmt.Errorf("portable collection definition: %w", err)
	}
	if err := basespec.ValidateRequiredText(
		"portable collection schema version",
		d.SchemaVersion,
		basespec.MaxVersionBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateLogicalName(d.LogicalName); err != nil {
		return fmt.Errorf("portable collection definition: %w", err)
	}
	if err := basespec.ValidateLogicalVersion(d.LogicalVersion, true); err != nil {
		return fmt.Errorf("portable collection definition: %w", err)
	}
	if err := basespec.ValidateOptionalText(
		"portable collection display name",
		d.DisplayName,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := basespec.ValidateOptionalText(
		"portable collection description",
		d.Description,
		basespec.MaxDescriptionBytes,
	); err != nil {
		return err
	}
	if err := validateLabels("portable collection", d.Labels); err != nil {
		return err
	}
	if _, err := jsonutil.CanonicalizeObject(
		d.Body,
		basespec.MaxDefinitionBodyBytes,
	); err != nil {
		return fmt.Errorf(
			"%w: portable collection body: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if len(d.Members) > MaxCollectionMembers {
		return fmt.Errorf(
			"%w: portable collection members exceed %d entries",
			basespec.ErrInvalid,
			MaxCollectionMembers,
		)
	}

	seen := make(map[string]struct{}, len(d.Members))
	for index, member := range d.Members {
		if err := member.Validate(); err != nil {
			return fmt.Errorf("portable collection members[%d]: %w", index, err)
		}
		identity := member.identity()
		if _, duplicate := seen[identity]; duplicate {
			return fmt.Errorf(
				"%w: duplicate portable collection member %d",
				basespec.ErrInvalid,
				index,
			)
		}
		seen[identity] = struct{}{}
	}
	return nil
}

func (d CollectionDefinition) Clone() CollectionDefinition {
	output := d
	output.Labels = cloneLabels(d.Labels)
	output.Body = append(json.RawMessage(nil), d.Body...)
	if d.Members != nil {
		output.Members = make([]ContentRef, len(d.Members))
		for index, member := range d.Members {
			output.Members[index] = member.Clone()
		}
	}
	return output
}

// CanonicalizeCollectionDefinition returns an independently owned canonical
// portable Collection Definition and verifies a supplied digest when present.
func CanonicalizeCollectionDefinition(
	input CollectionDefinition,
) (CollectionDefinition, error) {
	output := input.Clone()

	sort.Slice(output.Members, func(left, right int) bool {
		return output.Members[left].identity() <
			output.Members[right].identity()
	})

	body, err := jsonutil.CanonicalizeObject(
		output.Body,
		basespec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return CollectionDefinition{}, fmt.Errorf(
			"canonicalize portable collection body: %w",
			err,
		)
	}
	output.Body = json.RawMessage(body)

	suppliedDigest := output.Digest
	if output.Digest == "" {
		output.Digest = placeholderDigest
	}
	if err := output.Validate(); err != nil {
		return CollectionDefinition{}, err
	}

	payload := struct {
		Kind           basespec.CollectionKind `json:"kind"`
		SchemaID       basespec.SchemaID       `json:"schemaID"`
		SchemaVersion  string                  `json:"schemaVersion"`
		LogicalName    basespec.LogicalName    `json:"logicalName"`
		LogicalVersion basespec.LogicalVersion `json:"logicalVersion,omitempty"`
		DisplayName    string                  `json:"displayName,omitempty"`
		Description    string                  `json:"description,omitempty"`
		Labels         map[string]string       `json:"labels,omitempty"`
		Body           json.RawMessage         `json:"body"`
		Members        []ContentRef            `json:"members,omitempty"`
	}{
		Kind:           output.Kind,
		SchemaID:       output.SchemaID,
		SchemaVersion:  output.SchemaVersion,
		LogicalName:    output.LogicalName,
		LogicalVersion: output.LogicalVersion,
		DisplayName:    output.DisplayName,
		Description:    output.Description,
		Labels:         output.Labels,
		Body:           output.Body,
		Members:        output.Members,
	}
	calculated, err := canonicalPayloadDigest(
		"portable collection definition",
		payload,
	)
	if err != nil {
		return CollectionDefinition{}, err
	}
	if suppliedDigest != "" && suppliedDigest != calculated {
		return CollectionDefinition{}, fmt.Errorf(
			"%w: supplied portable collection digest %q, calculated %q",
			basespec.ErrDigestMismatch,
			suppliedDigest,
			calculated,
		)
	}
	output.Digest = calculated
	if err := output.Validate(); err != nil {
		return CollectionDefinition{}, err
	}
	return output, nil
}
