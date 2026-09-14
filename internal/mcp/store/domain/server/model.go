package server

import (
	"encoding/json"
	"fmt"
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain"
	mcpDomainSecret "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/secret"
)

const installationDataNamespace = "flexigpt.dev/mcp-installation-v1"

var legacyInstallationDataKeys = []string{
	"schemaVersion",
	"selectedConnectionProfile",
	"inputs",
	"additionalPolicies",
}

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
		SchemaVersion: mcpDomain.InstallationDataSchemaVersion,
		Inputs:        map[string]InputBinding{},
	}
}

func EncodeServerData(
	input ServerData,
) (json.RawMessage, error) {
	return MergeServerData(
		json.RawMessage(jsonutil.EmptyObject),
		input,
	)
}

// MergeServerData updates MCP-owned installation data while preserving other
// Artifact consumers' namespaced local state.
func MergeServerData(
	raw json.RawMessage,
	input ServerData,
) (json.RawMessage, error) {
	value := input
	value.Inputs = maps.Clone(input.Inputs)
	value.AdditionalPolicies = append(
		[]artifact.ArtifactRef(nil),
		input.AdditionalPolicies...,
	)
	if err := value.Validate(); err != nil {
		return nil, err
	}
	fields, err := artifact.DecodeDataObject(raw)
	if err != nil {
		return nil, err
	}
	payload, err := jsonutil.MarshalCanonicalObject(
		value,
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return nil, err
	}
	for _, key := range legacyInstallationDataKeys {
		delete(fields, key)
	}
	fields[installationDataNamespace] = payload
	return artifact.EncodeDataObject(fields)
}

func DecodeServerData(
	raw json.RawMessage,
) (ServerData, error) {
	fields, err := artifact.DecodeDataObject(raw)
	if err != nil {
		return ServerData{}, err
	}
	if payload, found := fields[installationDataNamespace]; found {
		return decodeServerDataPayload(payload)
	}
	if !containsLegacyInstallationData(fields) {
		return DefaultServerData(), nil
	}
	legacy, err := artifact.EncodeDataObject(fields)
	if err != nil {
		return ServerData{}, err
	}
	return decodeServerDataPayload(legacy)
}

func containsLegacyInstallationData(
	values map[string]json.RawMessage,
) bool {
	for _, key := range legacyInstallationDataKeys {
		if _, found := values[key]; found {
			return true
		}
	}
	return false
}

func decodeServerDataPayload(
	raw json.RawMessage,
) (ServerData, error) {
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return ServerData{}, err
	}
	if string(canonical) == jsonutil.EmptyObject {
		return DefaultServerData(), nil
	}
	var value ServerData
	if err := jsonutil.DecodeCanonicalObjectBytesInto(
		canonical,
		&value,
		basespec.MaxLocalDataBytes,
	); err != nil {
		return ServerData{}, fmt.Errorf(
			"%w: decode MCP installation data: %w",
			basespec.ErrInvalid,
			err,
		)
	}

	if err := value.Validate(); err != nil {
		return ServerData{}, err
	}
	value.Inputs = maps.Clone(value.Inputs)
	value.AdditionalPolicies = append(
		[]artifact.ArtifactRef(nil),
		value.AdditionalPolicies...,
	)
	return value, nil
}

func (value ServerData) Validate() error {
	if value.SchemaVersion != mcpDomain.InstallationDataSchemaVersion {
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
	if len(value.Inputs) > basespec.MaxDefinitionDependencies ||
		len(value.AdditionalPolicies) > basespec.MaxDefinitionDependencies {
		return fmt.Errorf(
			"%w: MCP installation data exceeds entry limits",
			basespec.ErrInvalid,
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
		if binding.SecretRef != "" {
			if _, err := mcpDomainSecret.ParseMCPSecretRef(binding.SecretRef); err != nil {
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
				"%w: duplicate additional MCP Policy",
				basespec.ErrInvalid,
			)
		}
		seen[ref] = struct{}{}
	}
	return nil
}
