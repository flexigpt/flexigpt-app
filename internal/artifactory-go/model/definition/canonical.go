package definition

import (
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/schema"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const placeholderDigest cryptoutil.Digest = cryptoutil.DigestSHA256Prefix +
	"0000000000000000000000000000000000000000000000000000000000000000"

func Canonicalize(input Definition) (Definition, error) {
	output := input
	output.Labels = cloneLabels(input.Labels)
	output.Dependencies = cloneSelectors(input.Dependencies)
	output.Body = append(json.RawMessage(nil), input.Body...)

	body, err := jsonutil.CanonicalizeObject(
		output.Body,
		model.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return Definition{}, fmt.Errorf("canonicalize definition body: %w", err)
	}
	output.Body = json.RawMessage(body)

	suppliedDigest := output.Digest
	if output.Digest == "" {
		output.Digest = placeholderDigest
	}
	if err := validateDefinitionFields(output); err != nil {
		return Definition{}, err
	}

	payload := struct {
		Kind           artifact.ArtifactKind `json:"kind"`
		SchemaID       schema.SchemaID       `json:"schemaID"`
		SchemaVersion  string                `json:"schemaVersion"`
		LogicalName    model.LogicalName     `json:"logicalName"`
		LogicalVersion model.LogicalVersion  `json:"logicalVersion,omitempty"`
		DisplayName    string                `json:"displayName,omitempty"`
		Description    string                `json:"description,omitempty"`
		Labels         map[string]string     `json:"labels,omitempty"`
		Body           json.RawMessage       `json:"body"`
		Dependencies   []Selector            `json:"dependencies,omitempty"`
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

	calculated, err := canonicalPayloadDigest("definition", payload)
	if err != nil {
		return Definition{}, err
	}
	if suppliedDigest != "" && suppliedDigest != calculated {
		return Definition{}, fmt.Errorf(
			"%w: supplied definition digest %q, calculated %q",
			model.ErrDigestMismatch,
			suppliedDigest,
			calculated,
		)
	}
	output.Digest = calculated
	return output, nil
}

func canonicalPayloadDigest(
	subject string,
	payload any,
) (cryptoutil.Digest, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal %s payload: %w", subject, err)
	}
	canonicalPayload, err := jsonutil.Canonicalize(raw)
	if err != nil {
		return "", fmt.Errorf(
			"canonicalize %s payload: %w",
			subject,
			err,
		)
	}
	if len(canonicalPayload) > model.MaxDefinitionBytes {
		return "", fmt.Errorf(
			"%w: canonical %s exceeds %d bytes",
			model.ErrInvalid,
			subject,
			model.MaxDefinitionBytes,
		)
	}
	return cryptoutil.DigestBytes(canonicalPayload), nil
}
