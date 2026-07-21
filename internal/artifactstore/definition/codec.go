package definition

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/jsoncanon"
)

const placeholderDigest artifactstore.Digest = artifactstore.DigestSHA256Prefix +
	"0000000000000000000000000000000000000000000000000000000000000000"

func EncodeFile(value Definition) ([]byte, error) {
	canonical, err := Canonicalize(value)
	if err != nil {
		return nil, err
	}
	file := File{
		Format:     FileFormatV1,
		Definition: canonical,
	}
	raw, err := json.Marshal(file)
	if err != nil {
		return nil, fmt.Errorf("marshal definition file: %w", err)
	}
	return jsoncanon.Canonicalize(raw)
}

func DecodeFile(raw []byte) (Definition, error) {
	canonicalFile, err := jsoncanon.Canonicalize(raw)
	if err != nil {
		return Definition{}, err
	}
	var file File
	if err := json.Unmarshal(canonicalFile, &file); err != nil {
		return Definition{}, fmt.Errorf("decode definition file: %w", err)
	}
	if err := file.Validate(); err != nil {
		return Definition{}, err
	}
	return Canonicalize(file.Definition)
}

func Canonicalize(input Definition) (Definition, error) {
	output := input
	output.Labels = sortedLabels(cloneLabels(input.Labels))
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

	calculated := DigestBytes(canonicalPayload)
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

func DigestBytes(content []byte) artifactstore.Digest {
	sum := sha256.Sum256(content)
	return artifactstore.Digest(
		artifactstore.DigestSHA256Prefix + hex.EncodeToString(sum[:]),
	)
}
