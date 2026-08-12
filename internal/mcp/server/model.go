package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/builtin/schema"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type InputBinding struct {
	Value     *string `json:"value,omitempty"`
	SecretRef string  `json:"secretRef,omitempty"`
}

type ServerData struct {
	SchemaVersion string `json:"schemaVersion"`

	SelectedConnectionProfile string                  `json:"selectedConnectionProfile,omitempty"`
	Inputs                    map[string]InputBinding `json:"inputs,omitempty"`
	AdditionalPolicies        []artifact.ArtifactRef  `json:"additionalPolicies,omitempty"`
}

func DefaultServerData() ServerData {
	return ServerData{
		SchemaVersion: schema.MCPSchemaVersion,
		Inputs:        map[string]InputBinding{},
	}
}

func EncodeServerData(
	input ServerData,
) (json.RawMessage, error) {
	value := input
	value.Inputs = maps.Clone(input.Inputs)
	value.AdditionalPolicies = append(
		[]artifact.ArtifactRef(nil),
		input.AdditionalPolicies...,
	)
	if err := ValidateServerData(value); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(canonical), nil
}

func DecodeServerData(
	raw json.RawMessage,
) (ServerData, error) {
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return ServerData{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()

	var value ServerData
	if err := decoder.Decode(&value); err != nil {
		return ServerData{}, fmt.Errorf(
			"%w: decode MCP installation data: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("MCP installation data has trailing JSON")
		}
		return ServerData{}, fmt.Errorf("%w: %w", basespec.ErrInvalid, err)
	}
	if err := ValidateServerData(value); err != nil {
		return ServerData{}, err
	}
	value.Inputs = maps.Clone(value.Inputs)
	value.AdditionalPolicies = append(
		[]artifact.ArtifactRef(nil),
		value.AdditionalPolicies...,
	)
	return value, nil
}

func ValidateServerData(value ServerData) error {
	if value.SchemaVersion != schema.MCPSchemaVersion {
		return fmt.Errorf(
			"%w: unsupported MCP installation schema %q",
			basespec.ErrInvalid,
			value.SchemaVersion,
		)
	}
	if err := basespec.ValidateOptionalText(
		"selected MCP connection profile",
		value.SelectedConnectionProfile,
		basespec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if len(value.Inputs) > basespec.MaxDefinitionDependencies {
		return fmt.Errorf(
			"%w: MCP installation inputs exceed %d entries",
			basespec.ErrInvalid,
			basespec.MaxDefinitionDependencies,
		)
	}
	if len(value.AdditionalPolicies) >
		basespec.MaxDefinitionDependencies {
		return fmt.Errorf(
			"%w: additional MCP policies exceed %d entries",
			basespec.ErrInvalid,
			basespec.MaxDefinitionDependencies,
		)
	}

	seen := make(map[artifact.ArtifactRef]struct{})
	for name, binding := range value.Inputs {
		if !installationInputNamePattern.MatchString(name) {
			return fmt.Errorf(
				"%w: invalid MCP installation input name %q",
				basespec.ErrInvalid,
				name,
			)
		}
		if binding.Value != nil && binding.SecretRef != "" {
			return fmt.Errorf(
				"%w: MCP input %q has both value and secretRef",
				basespec.ErrInvalid,
				name,
			)
		}
		if binding.Value == nil && binding.SecretRef == "" {
			return fmt.Errorf(
				"%w: MCP input %q has no value or secretRef",
				basespec.ErrInvalid,
				name,
			)
		}
		if binding.Value != nil {
			if err := basespec.ValidateOptionalText(
				"MCP installation input value",
				*binding.Value,
				basespec.MaxDescriptionBytes,
			); err != nil {
				return fmt.Errorf("MCP input %q: %w", name, err)
			}
		}
		if binding.SecretRef != "" {
			if err := basespec.ValidateRequiredText(
				"MCP installation secret reference",
				binding.SecretRef,
				basespec.MaxURIBytes,
			); err != nil {
				return fmt.Errorf("MCP input %q: %w", name, err)
			}
		}
	}
	for _, ref := range value.AdditionalPolicies {
		if err := ref.Validate(); err != nil {
			return err
		}
		if _, duplicate := seen[ref]; duplicate {
			return fmt.Errorf(
				"%w: duplicate additional MCP policy",
				basespec.ErrInvalid,
			)
		}
		seen[ref] = struct{}{}
	}
	return nil
}
