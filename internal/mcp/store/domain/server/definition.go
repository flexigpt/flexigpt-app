package server

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcpv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain"
)

func NewDocument(
	name basespec.LogicalName,
	version basespec.LogicalVersion,
	displayName string,
	description string,
	labels map[string]string,
	core CoreServer,
	configuration ServerConfiguration,
) (ServerDocument, error) {
	normalizedCore := NormalizeCoreServer(core)
	normalizedConfiguration := NormalizeServerConfiguration(configuration)
	normalizedConfiguration = withImplicitEnvironmentInputs(
		normalizedCore,
		normalizedConfiguration,
	)
	value := ServerDocument{
		LogicalName:    name,
		LogicalVersion: version,
		DisplayName:    displayName,
		Description:    description,
		Labels:         maps.Clone(labels),
		MCPServer:      normalizedCore,
		Configuration:  normalizedConfiguration,
	}
	if err := value.Validate(); err != nil {
		return ServerDocument{}, err
	}
	return value, nil
}

func DefinitionForDocument(
	input ServerDocument,
) (definition.Definition, error) {
	if err := input.Validate(); err != nil {
		return definition.Definition{}, err
	}

	decl, err := declarationForDocument(input)
	if err != nil {
		return definition.Definition{}, err
	}
	return mcpv1.DefinitionForDeclaration(decl)
}

func ServerDocumentFromDefinition(
	input definition.Definition,
) (ServerDocument, error) {
	if err := input.Validate(); err != nil {
		return ServerDocument{}, err
	}
	if input.Kind != mcpDomain.MCPArtifactKind ||
		input.SchemaID != mcpv1.MCPSchemaKey.SchemaID ||
		input.SchemaVersion != mcpv1.MCPSchemaKey.SchemaVersion {
		return ServerDocument{}, fmt.Errorf(
			"%w: Definition is not an MCP Artifact",
			basespec.ErrInvalid,
		)
	}

	decl, err := mcpv1.DecodeMCPJSON(input.Body)
	if err != nil {
		return ServerDocument{}, err
	}
	if decl.Name != string(input.LogicalName) ||
		decl.Description != input.Description {
		return ServerDocument{}, fmt.Errorf(
			"%w: MCP Definition does not match declaration header",
			basespec.ErrInvalid,
		)
	}

	core, err := coreFromDeclaration(decl)
	if err != nil {
		return ServerDocument{}, err
	}
	configuration, err := configurationFromDeclaration(
		decl,
	)
	if err != nil {
		return ServerDocument{}, err
	}

	logicalVersion := input.LogicalVersion
	document, err := NewDocument(
		input.LogicalName,
		logicalVersion,
		input.DisplayName,
		input.Description,
		input.Labels,
		core,
		configuration,
	)
	if err != nil {
		return ServerDocument{}, err
	}
	document.Include = includeFromDeclaration(decl.Include)
	if err := document.Validate(); err != nil {
		return ServerDocument{}, err
	}
	return document, nil
}

func DefinitionForMCPDeclaration(
	input mcpv1.MCPDocument,
) (definition.Definition, error) {
	return mcpv1.DefinitionForDeclaration(input)
}

func declarationForDocument(
	input ServerDocument,
) (mcpv1.MCPDocument, error) {
	decl := mcpv1.MCPDocument{
		Type:               mcpv1.MCPType,
		Name:               string(input.LogicalName),
		DisplayName:        input.DisplayName,
		Description:        input.Description,
		Labels:             maps.Clone(input.Labels),
		Command:            input.MCPServer.Command,
		Args:               append([]string(nil), input.MCPServer.Args...),
		Env:                maps.Clone(input.MCPServer.Env),
		URL:                input.MCPServer.URL,
		Headers:            maps.Clone(input.MCPServer.Headers),
		Include:            includeToDeclaration(input.Include),
		TimeoutMS:          input.Configuration.TimeoutMS,
		Auth:               authenticationToDeclaration(input.Configuration.Auth),
		Install:            installToDeclaration(input.Configuration.Install),
		ConnectionProfiles: profilesToDeclaration(input.Configuration.ConnectionProfiles),
		Policy:             policyToDeclaration(input.Configuration.Policy),
	}
	switch input.MCPServer.Type {
	case ServerTypeStdio:
		decl.Transport = mcpv1.TransportStdio
	case ServerTypeHTTP:
		decl.Transport = mcpv1.TransportStreamableHTTP

	default:
		return mcpv1.MCPDocument{}, fmt.Errorf(
			"%w: unsupported MCP server transport %q",
			basespec.ErrInvalid,
			input.MCPServer.Type,
		)
	}
	if err := decl.Validate(); err != nil {
		return mcpv1.MCPDocument{}, err
	}
	return decl, nil
}

func coreFromDeclaration(
	input mcpv1.MCPDocument,
) (CoreServer, error) {
	output := CoreServer{
		Command: input.Command,
		Args:    append([]string(nil), input.Args...),
		Env:     maps.Clone(input.Env),
		URL:     input.URL,
		Headers: maps.Clone(input.Headers),
	}
	switch input.Transport {
	case mcpv1.TransportStdio:
		output.Type = ServerTypeStdio
	case mcpv1.TransportStreamableHTTP:
		output.Type = ServerTypeHTTP

	default:
		return CoreServer{}, fmt.Errorf(
			"%w: MCP declaration has no executable transport",
			basespec.ErrReferenceUnresolved,
		)
	}
	if err := validateCoreServer(output); err != nil {
		return CoreServer{}, err
	}
	return output, nil
}

func configurationFromDeclaration(
	input mcpv1.MCPDocument,
) (ServerConfiguration, error) {
	value, err := serverConfigurationFromDeclaration(input)
	if err != nil {
		return ServerConfiguration{}, err
	}
	value = NormalizeServerConfiguration(value)
	if err := validateExtension("", value); err != nil {
		return ServerConfiguration{}, err
	}
	return value, nil
}

func PolicyReferenceSelector(
	name basespec.LogicalName,
) definition.Selector {
	return definition.Selector{
		Kind:        mcpDomain.MCPPolicyArtifactKind,
		LogicalName: name,
	}
}

func includeFromDeclaration(input *mcpv1.Include) *Include {
	if input == nil {
		return nil
	}
	return &Include{
		Tools:     slices.Clone(input.Tools),
		Resources: slices.Clone(input.Resources),
		Prompts:   slices.Clone(input.Prompts),
	}
}

func includeToDeclaration(input *Include) *mcpv1.Include {
	if input == nil {
		return nil
	}
	return &mcpv1.Include{
		Tools:     slices.Clone(input.Tools),
		Resources: slices.Clone(input.Resources),
		Prompts:   slices.Clone(input.Prompts),
	}
}

func authenticationToDeclaration(
	input AuthenticationDeclaration,
) *mcpv1.Auth {
	if input.Mode == "" {
		return nil
	}
	return &mcpv1.Auth{
		Mode:                        input.Mode,
		ClientCredentialsInput:      input.ClientCredentialsInput,
		ClientIDMetadataDocumentURL: input.ClientIDMetadataDocumentURL,
	}
}

func installToDeclaration(
	input InstallationDeclaration,
) *mcpv1.Install {
	if input.Note == "" &&
		len(input.Inputs) == 0 &&
		len(input.AllowEnvironment) == 0 {
		return nil
	}
	output := &mcpv1.Install{
		Note:             input.Note,
		Inputs:           make(map[string]mcpv1.InstallInput, len(input.Inputs)),
		AllowEnvironment: append([]string(nil), input.AllowEnvironment...),
	}
	for name, value := range input.Inputs {
		output.Inputs[name] = mcpv1.InstallInput{
			Kind:                 string(value.Kind),
			Label:                value.Label,
			Description:          value.Description,
			Note:                 value.Note,
			Placeholder:          value.Placeholder,
			Required:             pointerBool(value.Required),
			Default:              stringDefaultToJSON(value.Default),
			ClientSecretRequired: pointerBool(value.ClientSecretRequired),
		}
	}
	return output
}

func profilesToDeclaration(
	input map[string]ConnectionProfile,
) map[string]mcpv1.ConnectionProfile {
	if input == nil {
		return nil
	}
	output := make(map[string]mcpv1.ConnectionProfile, len(input))
	for name, profile := range input {
		value := mcpv1.ConnectionProfile{
			Platforms: append([]string(nil), profile.Platforms...),
		}
		if profile.Stdio != nil {
			stdio := &mcpv1.StdioProfile{
				Env:       maps.Clone(profile.Stdio.Env),
				RemoveEnv: append([]string(nil), profile.Stdio.RemoveEnv...),
			}
			if profile.Stdio.Command != nil {
				stdio.Command = *profile.Stdio.Command
			}
			if profile.Stdio.Args != nil {
				stdio.Args = append([]string(nil), (*profile.Stdio.Args)...)
			}
			value.Stdio = stdio
		}
		if profile.HTTP != nil {
			http := &mcpv1.HTTPProfile{
				Headers:       maps.Clone(profile.HTTP.Headers),
				RemoveHeaders: append([]string(nil), profile.HTTP.RemoveHeaders...),
			}
			if profile.HTTP.URL != nil {
				http.URL = *profile.HTTP.URL
			}
			value.HTTP = http
		}
		output[name] = value
	}
	return output
}

func policyToDeclaration(
	input *PolicyReference,
) *mcpv1.PolicyReference {
	if input == nil {
		return nil
	}
	required := input.Required
	return &mcpv1.PolicyReference{
		Name:     input.Name,
		Required: &required,
	}
}

func serverConfigurationFromDeclaration(
	input mcpv1.MCPDocument,
) (ServerConfiguration, error) {
	output := ServerConfiguration{
		TimeoutMS: input.TimeoutMS,
		Auth: AuthenticationDeclaration{
			Mode: mcpv1.HTTPAuthModeNone,
		},
		Install: InstallationDeclaration{
			Inputs: map[string]InputDeclaration{},
		},
	}
	if input.Auth != nil {
		output.Auth = AuthenticationDeclaration{
			Mode:                        input.Auth.Mode,
			ClientCredentialsInput:      input.Auth.ClientCredentialsInput,
			ClientIDMetadataDocumentURL: input.Auth.ClientIDMetadataDocumentURL,
		}
	}
	if input.Install != nil {
		output.Install.Note = input.Install.Note
		output.Install.AllowEnvironment = append(
			[]string(nil),
			input.Install.AllowEnvironment...,
		)
		for name, value := range input.Install.Inputs {
			defaultValue, err := jsonDefaultToString(value.Default)
			if err != nil {
				return ServerConfiguration{}, fmt.Errorf(
					"MCP installation input %q default: %w",
					name,
					err,
				)
			}
			output.Install.Inputs[name] = InputDeclaration{
				Kind:                 InputKind(value.Kind),
				Label:                value.Label,
				Description:          value.Description,
				Note:                 value.Note,
				Placeholder:          value.Placeholder,
				Required:             dereferenceBool(value.Required),
				Default:              defaultValue,
				ClientSecretRequired: dereferenceBool(value.ClientSecretRequired),
			}
		}
	}
	if input.ConnectionProfiles != nil {
		output.ConnectionProfiles = make(
			map[string]ConnectionProfile,
			len(input.ConnectionProfiles),
		)
		for name, value := range input.ConnectionProfiles {
			profile := ConnectionProfile{
				Platforms: append([]string(nil), value.Platforms...),
			}
			if value.Stdio != nil {
				stdio := &StdioProfile{
					Env:       maps.Clone(value.Stdio.Env),
					RemoveEnv: append([]string(nil), value.Stdio.RemoveEnv...),
				}
				if value.Stdio.Command != "" {
					command := value.Stdio.Command
					stdio.Command = &command
				}
				if value.Stdio.Args != nil {
					args := append([]string(nil), value.Stdio.Args...)
					stdio.Args = &args
				}
				profile.Stdio = stdio
			}
			if value.HTTP != nil {
				http := &HTTPProfile{
					Headers:       maps.Clone(value.HTTP.Headers),
					RemoveHeaders: append([]string(nil), value.HTTP.RemoveHeaders...),
				}
				if value.HTTP.URL != "" {
					url := value.HTTP.URL
					http.URL = &url
				}
				profile.HTTP = http
			}
			output.ConnectionProfiles[name] = profile
		}
	}
	if input.Policy != nil {
		output.Policy = &PolicyReference{
			Name:     input.Policy.Name,
			Required: input.Policy.Required == nil || *input.Policy.Required,
		}
	}
	return output, nil
}

func pointerBool(value bool) *bool {
	output := value
	return &output
}

func dereferenceBool(value *bool) bool {
	return value != nil && *value
}

func stringDefaultToJSON(value *string) *json.RawMessage {
	if value == nil {
		return nil
	}
	raw, err := json.Marshal(*value)
	if err != nil {
		panic(err)
	}
	output := json.RawMessage(raw)
	return &output
}

func jsonDefaultToString(value *json.RawMessage) (*string, error) {
	if value == nil {
		//nolint:nilnil // Nil to nil conversion.
		return nil, nil
	}
	var output string
	if err := json.Unmarshal(*value, &output); err != nil {
		return nil, fmt.Errorf(
			"%w: MCP runtime supports string installation defaults only",
			basespec.ErrUnsupported,
		)
	}
	return &output, nil
}
