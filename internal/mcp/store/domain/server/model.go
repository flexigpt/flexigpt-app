package server

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain"
	mcpDomainSecret "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/secret"
)

const installationDataNamespace = "flexigpt.site/mcp-installation-v1"

type InputBinding struct {
	Value     *string `json:"value,omitempty"`
	SecretRef string  `json:"secretRef,omitempty"`
}

type ServerData struct {
	SchemaVersion string `json:"schemaVersion"`

	SelectedConnectionProfile string                      `json:"selectedConnectionProfile,omitempty"`
	Inputs                    map[string]InputBinding     `json:"inputs,omitempty"`
	AdditionalPolicies        []artifactModel.ArtifactRef `json:"additionalPolicies,omitempty"`
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
		[]artifactModel.ArtifactRef(nil),
		input.AdditionalPolicies...,
	)
	if err := value.Validate(); err != nil {
		return nil, err
	}
	fields, err := artifactModel.DecodeDataObject(raw)
	if err != nil {
		return nil, err
	}
	payload, err := jsonutil.MarshalCanonicalObject(
		value,
		spec.MaxLocalDataBytes,
	)
	if err != nil {
		return nil, err
	}
	fields[installationDataNamespace] = payload
	return artifactModel.EncodeDataObject(fields)
}

func DecodeServerData(
	raw json.RawMessage,
) (ServerData, error) {
	fields, err := artifactModel.DecodeDataObject(raw)
	if err != nil {
		return ServerData{}, err
	}
	if payload, found := fields[installationDataNamespace]; found {
		return decodeServerDataPayload(payload)
	}
	return DefaultServerData(), nil
}

func decodeServerDataPayload(
	raw json.RawMessage,
) (ServerData, error) {
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		spec.MaxLocalDataBytes,
	)
	if err != nil {
		return ServerData{}, err
	}
	if string(canonical) == jsonutil.EmptyObject {
		return DefaultServerData(), nil
	}
	canonical, err = stripRetiredRuntimeEnabled(canonical)
	if err != nil {
		return ServerData{}, err
	}

	var value ServerData
	if err := jsonutil.DecodeCanonicalObjectBytesInto(
		canonical,
		&value,
		spec.MaxLocalDataBytes,
	); err != nil {
		return ServerData{}, fmt.Errorf(
			"%w: decode MCP installation data: %w",
			spec.ErrInvalid,
			err,
		)
	}

	if err := value.Validate(); err != nil {
		return ServerData{}, err
	}
	value.Inputs = maps.Clone(value.Inputs)
	value.AdditionalPolicies = append(
		[]artifactModel.ArtifactRef(nil),
		value.AdditionalPolicies...,
	)
	return value, nil
}

func (value ServerData) Clone() ServerData {
	output := value
	output.Inputs = maps.Clone(value.Inputs)
	for name, binding := range output.Inputs {
		binding.Value = cloneStringPointer(binding.Value)
		output.Inputs[name] = binding
	}
	output.AdditionalPolicies = slices.Clone(value.AdditionalPolicies)
	return output
}

func (value ServerData) Validate() error {
	if value.SchemaVersion != mcpDomain.InstallationDataSchemaVersion {
		return fmt.Errorf(
			"%w: unsupported MCP installation schema %q",
			spec.ErrInvalid,
			value.SchemaVersion,
		)
	}
	if err := spec.ValidateOptionalText(
		"selected MCP connection profile",
		value.SelectedConnectionProfile,
		spec.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if len(value.Inputs) > spec.MaxDefinitionDependencies ||
		len(value.AdditionalPolicies) > spec.MaxDefinitionDependencies {
		return fmt.Errorf(
			"%w: MCP installation data exceeds entry limits",
			spec.ErrInvalid,
		)
	}
	seen := make(map[artifactModel.ArtifactRef]struct{})
	for name, binding := range value.Inputs {
		if !installationInputNamePattern.MatchString(name) {
			return fmt.Errorf(
				"%w: invalid MCP installation input name %q",
				spec.ErrInvalid,
				name,
			)
		}
		if binding.Value != nil && binding.SecretRef != "" {
			return fmt.Errorf(
				"%w: MCP input %q has both value and secretRef",
				spec.ErrInvalid,
				name,
			)
		}
		if binding.Value == nil && binding.SecretRef == "" {
			return fmt.Errorf(
				"%w: MCP input %q has no value or secretRef",
				spec.ErrInvalid,
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
				spec.ErrInvalid,
			)
		}
		seen[ref] = struct{}{}
	}
	return nil
}

// SecretReferences returns the unique opaque secret references held by local
// server installation data. It never resolves or returns secret values.
func (value ServerData) SecretReferences() ([]string, error) {
	if err := value.Validate(); err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	for _, binding := range value.Inputs {
		if binding.SecretRef == "" {
			continue
		}
		seen[binding.SecretRef] = struct{}{}
	}

	output := make([]string, 0, len(seen))
	for value := range seen {
		output = append(output, value)
	}
	sort.Strings(output)
	return output, nil
}

func (value ServerData) ValidateFor(
	server artifactModel.ArtifactRef,
	document ServerDocument,
) error {
	if err := server.Validate(); err != nil {
		return err
	}
	if err := document.Validate(); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}

	secretTargets, err := document.SecretInputTargets()
	if err != nil {
		return err
	}

	if value.SelectedConnectionProfile != "" {
		if _, found := document.Configuration.ConnectionProfiles[value.SelectedConnectionProfile]; !found {
			return fmt.Errorf(
				"%w: selected MCP connection profile %q does not exist",
				spec.ErrReferenceUnresolved,
				value.SelectedConnectionProfile,
			)
		}
	}

	for name, binding := range value.Inputs {
		if !installationInputNamePattern.MatchString(name) {
			return fmt.Errorf(
				"%w: invalid MCP installation input name %q",
				spec.ErrInvalid,
				name,
			)
		}

		declaration, declared := document.Configuration.Install.Inputs[name]
		if !declared {
			return fmt.Errorf(
				"%w: MCP installation input %q is not declared by the server",
				spec.ErrInvalid,
				name,
			)
		}

		switch declaration.Kind {
		case InputText, InputPath:
			if binding.SecretRef != "" {
				return fmt.Errorf(
					"%w: MCP input %q must use a local value, not a secret reference",
					spec.ErrInvalid,
					name,
				)
			}
			if binding.Value == nil {
				return fmt.Errorf(
					"%w: MCP input %q requires a local value",
					spec.ErrInvalid,
					name,
				)
			}

		case InputSecret:
			if binding.Value != nil || strings.TrimSpace(binding.SecretRef) == "" {
				return fmt.Errorf(
					"%w: MCP secret input %q requires exactly one secret reference",
					spec.ErrInvalid,
					name,
				)
			}
			target, found := secretTargets[name]
			if !found {
				return fmt.Errorf(
					"%w: MCP secret input %q is not used by a permitted connection target",
					spec.ErrInvalid,
					name,
				)
			}
			if err := target.matches(server, binding.SecretRef); err != nil {
				return fmt.Errorf("MCP secret input %q: %w", name, err)
			}

		case InputOAuthClientCredentials:
			if binding.Value != nil || strings.TrimSpace(binding.SecretRef) == "" {
				return fmt.Errorf(
					"%w: MCP OAuth client input %q requires exactly one secret reference",
					spec.ErrInvalid,
					name,
				)
			}
			ref, err := mcpDomainSecret.ParseMCPSecretRef(binding.SecretRef)
			if err != nil {
				return fmt.Errorf("MCP OAuth client input %q: %w", name, err)
			}
			if err := ref.Matches(
				server,
				mcpDomainSecret.MCPSecretKindOAuthClientCredentials,
				"clientCredentials",
			); err != nil {
				return fmt.Errorf("MCP OAuth client input %q: %w", name, err)
			}

		default:
			return fmt.Errorf(
				"%w: MCP input %q has unsupported kind %q",
				spec.ErrInvalid,
				name,
				declaration.Kind,
			)
		}
	}

	seen := make(map[artifactModel.ArtifactRef]struct{}, len(value.AdditionalPolicies))
	for _, ref := range value.AdditionalPolicies {
		if err := ref.Validate(); err != nil {
			return err
		}
		if ref.RootID != server.RootID {
			return fmt.Errorf(
				"%w: additional MCP policy belongs to another Root",
				spec.ErrInvalid,
			)
		}
		if _, duplicate := seen[ref]; duplicate {
			return fmt.Errorf(
				"%w: duplicate additional MCP policy Artifact",
				spec.ErrInvalid,
			)
		}
		seen[ref] = struct{}{}
	}

	return nil
}

// stripRetiredRuntimeEnabled accepts persisted installation records produced
// before MCP runtime enablement was removed. The field is ignored on read and
// omitted by the next installation-data write.
func stripRetiredRuntimeEnabled(
	raw []byte,
) ([]byte, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	delete(fields, "runtimeEnabled")
	return jsonutil.MarshalCanonicalObject(
		fields,
		spec.MaxLocalDataBytes,
	)
}
