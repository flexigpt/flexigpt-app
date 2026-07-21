package definition

import (
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/jsoncanon"
)

const placeholderDigest artifactstore.Digest = artifactstore.DigestSHA256Prefix +
	"0000000000000000000000000000000000000000000000000000000000000000"

func Canonicalize(input Definition) (Definition, error) {
	output := input
	output.Labels = cloneLabels(input.Labels)
	output.Dependencies = cloneSelectors(input.Dependencies)
	output.Body = append(json.RawMessage(nil), input.Body...)

	body, err := jsoncanon.CanonicalizeObject(
		output.Body,
		artifactstore.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return Definition{}, fmt.Errorf("canonicalize definition body: %w", err)
	}
	output.Body = json.RawMessage(body)

	suppliedDigest := output.Digest
	if output.Digest == "" {
		output.Digest = placeholderDigest
	}
	if err := output.Validate(); err != nil {
		return Definition{}, err
	}

	payload := struct {
		Kind           artifactstore.ArtifactKind   `json:"kind"`
		SchemaID       artifactstore.SchemaID       `json:"schemaID"`
		SchemaVersion  string                       `json:"schemaVersion"`
		LogicalName    artifactstore.LogicalName    `json:"logicalName"`
		LogicalVersion artifactstore.LogicalVersion `json:"logicalVersion,omitempty"`
		DisplayName    string                       `json:"displayName,omitempty"`
		Description    string                       `json:"description,omitempty"`
		Labels         map[string]string            `json:"labels,omitempty"`
		Body           json.RawMessage              `json:"body"`
		Dependencies   []Selector                   `json:"dependencies,omitempty"`
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
		Dependencies:   output.Dependencies,
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return Definition{}, fmt.Errorf("marshal definition payload: %w", err)
	}
	canonicalPayload, err := jsoncanon.Canonicalize(raw)
	if err != nil {
		return Definition{}, fmt.Errorf("canonicalize definition payload: %w", err)
	}

	calculated := artifactstore.DigestBytes(canonicalPayload)
	if suppliedDigest != "" && suppliedDigest != calculated {
		return Definition{}, fmt.Errorf(
			"%w: supplied definition digest %q, calculated %q",
			artifactstore.ErrDigestMismatch,
			suppliedDigest,
			calculated,
		)
	}
	output.Digest = calculated

	if err := output.Validate(); err != nil {
		return Definition{}, err
	}
	return output, nil
}
