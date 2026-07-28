// Package portable defines Artifact Store's portable, content-addressed
// collection envelope and source-independent member references.
package portable

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/url"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/jsoncanon"
)

const (
	MaxCollectionMembers      = 100_000
	MaxPortableMediaTypeBytes = 256
)

const placeholderDigest artifactstore.Digest = artifactstore.DigestSHA256Prefix +
	"0000000000000000000000000000000000000000000000000000000000000000"

// ContentRef identifies portable member content. Locator is relative to the
// containing document or package root. URI is an external portable acquisition
// reference and must be resolved by an explicit policy-controlled resolver.
//
// When Locator and URI are empty, SubresourceLocator identifies embedded
// content in the containing document.
type ContentRef struct {
	Locator            artifactstore.Locator            `json:"locator,omitempty"`
	URI                string                           `json:"uri,omitempty"`
	SubresourceLocator artifactstore.SubresourceLocator `json:"subresourceLocator,omitempty"`
	Digest             *artifactstore.Digest            `json:"digest,omitempty"`
	MediaType          string                           `json:"mediaType,omitempty"`
	Role               string                           `json:"role,omitempty"`
}

func (r ContentRef) Validate() error {
	switch {
	case r.Locator != "" && r.URI != "":
		return fmt.Errorf(
			"%w: portable content reference cannot contain both locator and URI",
			artifactstore.ErrInvalid,
		)

	case r.Locator != "":
		if err := artifactstore.ValidatePortableLocator(r.Locator, false); err != nil {
			return err
		}

	case r.URI != "":
		if err := validateURI(r.URI); err != nil {
			return err
		}

	case r.SubresourceLocator == "":
		return fmt.Errorf(
			"%w: portable content reference requires a locator, URI, or embedded subresource",
			artifactstore.ErrInvalid,
		)
	}

	if err := artifactstore.ValidatePortableSubresourceLocator(
		r.SubresourceLocator,
	); err != nil {
		return err
	}
	if r.Digest != nil {
		if err := artifactstore.ValidateDigest(*r.Digest); err != nil {
			return err
		}
	}
	if err := artifactstore.ValidateOptionalText(
		"portable content media type",
		r.MediaType,
		MaxPortableMediaTypeBytes,
	); err != nil {
		return err
	}
	if r.Role != "" {
		if err := artifactstore.ValidateIdentifier(
			"portable content role",
			r.Role,
			artifactstore.MaxKindBytes,
		); err != nil {
			return err
		}
	}
	return nil
}

func (r ContentRef) Clone() ContentRef {
	output := r
	if r.Digest != nil {
		value := *r.Digest
		output.Digest = &value
	}
	return output
}

func (r ContentRef) identity() string {
	return string(r.Locator) + "\x00" +
		r.URI + "\x00" +
		string(r.SubresourceLocator)
}

func validateURI(value string) error {
	if err := artifactstore.ValidateRequiredText(
		"portable content URI",
		value,
		artifactstore.MaxURIBytes,
	); err != nil {
		return err
	}
	// ParseRequestURI deliberately treats fragments as irrelevant to an HTTP
	// request target, so reject a raw fragment before parsing. A percent-encoded
	// hash remains valid path data and does not trigger this guard.
	if strings.Contains(value, "#") {
		return fmt.Errorf(
			"%w: portable content URI must use subresourceLocator instead of a fragment",
			artifactstore.ErrInvalid,
		)
	}

	parsed, err := url.ParseRequestURI(value)
	if err != nil {
		return fmt.Errorf(
			"%w: portable content URI is invalid: %w",
			artifactstore.ErrInvalid,
			err,
		)
	}
	if !parsed.IsAbs() || parsed.Scheme == "" {
		return fmt.Errorf(
			"%w: portable content URI must be absolute",
			artifactstore.ErrInvalid,
		)
	}
	if parsed.User != nil {
		return fmt.Errorf(
			"%w: portable content URI cannot contain user information",
			artifactstore.ErrInvalid,
		)
	}
	if parsed.Fragment != "" {
		return fmt.Errorf(
			"%w: portable content URI must use subresourceLocator instead of a fragment",
			artifactstore.ErrInvalid,
		)
	}
	if strings.EqualFold(parsed.Scheme, "file") {
		return fmt.Errorf(
			"%w: portable content URI cannot use the file scheme",
			artifactstore.ErrInvalid,
		)
	}
	return nil
}

// CollectionDefinition is the generic portable envelope for a shareable
// Collection. Its Body remains domain-owned, while Members remain visible to
// generic transfer and integrity code.
type CollectionDefinition struct {
	Digest         artifactstore.Digest         `json:"digest,omitempty"`
	Kind           artifactstore.CollectionKind `json:"kind"`
	SchemaID       artifactstore.SchemaID       `json:"schemaID"`
	SchemaVersion  string                       `json:"schemaVersion"`
	LogicalName    artifactstore.LogicalName    `json:"logicalName"`
	LogicalVersion artifactstore.LogicalVersion `json:"logicalVersion,omitempty"`
	DisplayName    string                       `json:"displayName,omitempty"`
	Description    string                       `json:"description,omitempty"`
	Labels         map[string]string            `json:"labels,omitempty"`
	Body           json.RawMessage              `json:"body"`
	Members        []ContentRef                 `json:"members,omitempty"`
}

func (d CollectionDefinition) Validate() error {
	if err := artifactstore.ValidateDigest(d.Digest); err != nil {
		return fmt.Errorf("portable collection definition: %w", err)
	}
	if err := artifactstore.ValidateCollectionKind(d.Kind); err != nil {
		return fmt.Errorf("portable collection definition: %w", err)
	}
	if err := artifactstore.ValidateSchemaID(d.SchemaID); err != nil {
		return fmt.Errorf("portable collection definition: %w", err)
	}
	if err := artifactstore.ValidateRequiredText(
		"portable collection schema version",
		d.SchemaVersion,
		artifactstore.MaxVersionBytes,
	); err != nil {
		return err
	}
	if err := artifactstore.ValidateLogicalName(d.LogicalName); err != nil {
		return fmt.Errorf("portable collection definition: %w", err)
	}
	if err := artifactstore.ValidateLogicalVersion(d.LogicalVersion, true); err != nil {
		return fmt.Errorf("portable collection definition: %w", err)
	}
	if err := artifactstore.ValidateOptionalText(
		"portable collection display name",
		d.DisplayName,
		artifactstore.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := artifactstore.ValidateOptionalText(
		"portable collection description",
		d.Description,
		artifactstore.MaxDescriptionBytes,
	); err != nil {
		return err
	}
	if len(d.Labels) > artifactstore.MaxLabels {
		return fmt.Errorf(
			"%w: portable collection labels exceed %d entries",
			artifactstore.ErrInvalid,
			artifactstore.MaxLabels,
		)
	}
	for key, value := range d.Labels {
		if err := artifactstore.ValidateIdentifier(
			"portable collection label key",
			key,
			artifactstore.MaxKindBytes,
		); err != nil {
			return err
		}
		if err := artifactstore.ValidateRequiredText(
			"portable collection label value",
			value,
			artifactstore.MaxLabelValueBytes,
		); err != nil {
			return err
		}
	}
	if _, err := jsoncanon.CanonicalizeObject(
		d.Body,
		artifactstore.MaxDefinitionBodyBytes,
	); err != nil {
		return fmt.Errorf(
			"%w: portable collection body: %w",
			artifactstore.ErrInvalid,
			err,
		)
	}
	if len(d.Members) > MaxCollectionMembers {
		return fmt.Errorf(
			"%w: portable collection members exceed %d entries",
			artifactstore.ErrInvalid,
			MaxCollectionMembers,
		)
	}

	seen := make(map[string]struct{}, len(d.Members))
	seenPortablePaths := make(map[string]struct{}, len(d.Members))
	for index, member := range d.Members {
		if err := member.Validate(); err != nil {
			return fmt.Errorf("portable collection members[%d]: %w", index, err)
		}
		identity := member.identity()
		if _, duplicate := seen[identity]; duplicate {
			return fmt.Errorf(
				"%w: duplicate portable collection member %d",
				artifactstore.ErrInvalid,
				index,
			)
		}
		seen[identity] = struct{}{}

		if member.Locator != "" {
			portablePath := strings.ToLower(string(member.Locator)) + "\x00" +
				strings.ToLower(string(member.SubresourceLocator))
			if _, duplicate := seenPortablePaths[portablePath]; duplicate {
				return fmt.Errorf(
					"%w: portable collection members contain a case-ambiguous locator at index %d",
					artifactstore.ErrInvalid,
					index,
				)
			}
			seenPortablePaths[portablePath] = struct{}{}
		}
	}
	return nil
}

func (d CollectionDefinition) Clone() CollectionDefinition {
	output := d
	output.Labels = maps.Clone(d.Labels)
	output.Body = append(json.RawMessage(nil), d.Body...)
	output.Members = make([]ContentRef, len(d.Members))
	for index, member := range d.Members {
		output.Members[index] = member.Clone()
	}
	return output
}

// CanonicalizeCollectionDefinition returns an independently owned canonical
// portable Collection Definition and verifies a supplied digest when present.
func CanonicalizeCollectionDefinition(
	input CollectionDefinition,
) (CollectionDefinition, error) {
	output := input.Clone()

	// Generic Members describe a content closure rather than domain display
	// ordering. Domain-defined ordering belongs in Body. Sorting here makes
	// equivalent portable closures produce the same digest.
	sort.Slice(output.Members, func(left, right int) bool {
		return output.Members[left].identity() < output.Members[right].identity()
	})

	body, err := jsoncanon.CanonicalizeObject(
		output.Body,
		artifactstore.MaxDefinitionBodyBytes,
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
		Kind           artifactstore.CollectionKind `json:"kind"`
		SchemaID       artifactstore.SchemaID       `json:"schemaID"`
		SchemaVersion  string                       `json:"schemaVersion"`
		LogicalName    artifactstore.LogicalName    `json:"logicalName"`
		LogicalVersion artifactstore.LogicalVersion `json:"logicalVersion,omitempty"`
		DisplayName    string                       `json:"displayName,omitempty"`
		Description    string                       `json:"description,omitempty"`
		Labels         map[string]string            `json:"labels,omitempty"`
		Body           json.RawMessage              `json:"body"`
		Members        []ContentRef                 `json:"members,omitempty"`
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
	raw, err := json.Marshal(payload)
	if err != nil {
		return CollectionDefinition{}, fmt.Errorf(
			"marshal portable collection definition: %w",
			err,
		)
	}
	canonical, err := jsoncanon.Canonicalize(raw)
	if err != nil {
		return CollectionDefinition{}, fmt.Errorf(
			"canonicalize portable collection definition: %w",
			err,
		)
	}

	calculated := artifactstore.DigestBytes(canonical)
	if suppliedDigest != "" && suppliedDigest != calculated {
		return CollectionDefinition{}, fmt.Errorf(
			"%w: supplied portable collection digest %q, calculated %q",
			artifactstore.ErrDigestMismatch,
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
