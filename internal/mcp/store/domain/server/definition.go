package server

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcpv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain"
)

func NewDocument(
	name basespec.LogicalName,
	version basespec.LogicalVersion,
	displayName string,
	description string,
	labels map[string]string,
	core CoreServer,
	extension ServerExtension,
) (ServerDocument, error) {
	normalizedCore := NormalizeCoreServer(core)
	normalizedExtension := NormalizeServerExtension(
		string(name),
		extension,
	)
	normalizedExtension = withImplicitEnvironmentInputs(
		normalizedCore,
		normalizedExtension,
	)
	value := ServerDocument{
		LogicalName:    name,
		LogicalVersion: version,
		DisplayName:    displayName,
		Description:    description,
		Labels:         maps.Clone(labels),
		MCPServer:      normalizedCore,
		Extension:      normalizedExtension,
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
	body, err := decl.CanonicalJSON()
	if err != nil {
		return definition.Definition{}, err
	}

	labels := maps.Clone(input.Labels)
	if labels == nil {
		labels = map[string]string{}
	}
	labels[TransportLabelKey] = string(input.MCPServer.Type)
	labels[AuthModeLabelKey] = string(input.Extension.Auth.Mode)

	dependencies := []definition.Selector(nil)
	if input.Extension.Policy != nil {
		dependencies = append(dependencies, definition.Selector{
			Kind:        mcpDomain.MCPPolicyArtifactKind,
			LogicalName: input.Extension.Policy.Ref,
		})
	}

	value := definition.Definition{
		Kind:           mcpDomain.MCPArtifactKind,
		SchemaID:       mcpv1.MCPSchemaKey.SchemaID,
		SchemaVersion:  mcpv1.MCPSchemaKey.SchemaVersion,
		LogicalName:    input.LogicalName,
		LogicalVersion: input.LogicalVersion,
		DisplayName:    input.DisplayName,
		Description:    input.Description,
		Labels:         labels,
		Body:           body,
		Dependencies:   dependencies,
	}
	return definition.Canonicalize(value)
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
	extension, err := extensionFromDeclaration(
		decl,
		input.LogicalName,
	)
	if err != nil {
		return ServerDocument{}, err
	}
	logicalVersion := input.LogicalVersion
	if logicalVersion == "" {
		logicalVersion = extension.LogicalVersion
	}
	displayName := input.DisplayName
	if extension.DisplayName != "" {
		// Canonical mcp declarations keep MCP-specific display metadata in
		// the namespaced metadata extension. Generic Definition metadata
		// remains derived from the portable declaration header.
		displayName = extension.DisplayName
	}

	document, err := NewDocument(
		input.LogicalName,
		logicalVersion,
		displayName,
		input.Description,
		input.Labels,
		core,
		extension,
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
	extensionRaw, err := jsonutil.MarshalCanonicalObject(
		input.Extension,
		basespec.MaxDefinitionBodyBytes,
	)
	if err != nil {
		return mcpv1.MCPDocument{}, err
	}

	decl := mcpv1.MCPDocument{
		APIVersion:  mcpv1.MCPSchemaVersion,
		Type:        mcpv1.MCPType,
		Name:        string(input.LogicalName),
		Description: input.Description,
		Metadata: map[string]json.RawMessage{
			mcpDomain.RuntimeExtensionMetadataKey: extensionRaw,
		},
		Command: input.MCPServer.Command,
		Args:    append([]string(nil), input.MCPServer.Args...),
		Env:     maps.Clone(input.MCPServer.Env),
		URL:     input.MCPServer.URL,
		Headers: maps.Clone(input.MCPServer.Headers),
		Include: includeToDeclaration(input.Include),
	}
	switch input.MCPServer.Type {
	case ServerTypeStdio:
		decl.Transport = mcpv1.TransportStdio
	case ServerTypeHTTP:
		decl.Transport = mcpv1.TransportStreamableHTTP
	case ServerTypeSSE:
		decl.Transport = mcpv1.TransportSSE
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
	if input.Transport == "" &&
		input.Locator != nil &&
		input.Locator.Kind == declaration.LocatorKindCommand {
		output.Type = ServerTypeStdio
		output.Command = input.Locator.Command
		return output, validateCoreServer(output)
	}
	switch input.Transport {
	case mcpv1.TransportStdio:
		output.Type = ServerTypeStdio
	case mcpv1.TransportStreamableHTTP:
		output.Type = ServerTypeHTTP
	case mcpv1.TransportSSE:
		// The configured SDK transport supports standalone SSE fallback when
		// DisableStandaloneSSE is false. Preserve the semantic transport for
		// round-trip declaration serialization.
		output.Type = ServerTypeSSE
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

func extensionFromDeclaration(
	input mcpv1.MCPDocument,
	name basespec.LogicalName,
) (ServerExtension, error) {
	value := ServerExtension{}
	raw, found := input.Metadata[mcpDomain.RuntimeExtensionMetadataKey]
	if found {
		if err := jsonutil.DecodeCanonicalObjectInto(
			raw,
			&value,
			basespec.MaxDefinitionBodyBytes,
		); err != nil {
			return ServerExtension{}, fmt.Errorf(
				"MCP runtime metadata extension: %w",
				err,
			)
		}
	}
	value = NormalizeServerExtension(string(name), value)
	if err := validateExtension(string(name), value); err != nil {
		return ServerExtension{}, err
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

func cloneInclude(input *Include) *Include {
	if input == nil {
		return nil
	}
	return &Include{
		Tools:     slices.Clone(input.Tools),
		Resources: slices.Clone(input.Resources),
		Prompts:   slices.Clone(input.Prompts),
	}
}

// RebindLocatedDocument applies a local MCP declaration identity and optional
// include/runtime extension data to one server decoded from a local source.
func RebindLocatedDocument(
	input ServerDocument,
	identity definition.Definition,
	outer mcpv1.MCPDocument,
) (ServerDocument, error) {
	output, err := CanonicalizeServer(input)
	if err != nil {
		return ServerDocument{}, err
	}
	output.LogicalName = identity.LogicalName
	output.LogicalVersion = identity.LogicalVersion
	output.DisplayName = identity.DisplayName
	output.Description = identity.Description
	if output.DisplayName == "" {
		output.DisplayName = string(identity.LogicalName)
	}
	output.MCPServer, err = overlayLocatedCore(output.MCPServer, outer)
	if err != nil {
		return ServerDocument{}, err
	}
	if outer.Include != nil {
		output.Include = includeFromDeclaration(outer.Include)
	} else {
		output.Include = cloneInclude(input.Include)
	}
	outerExtension := false
	if _, found := outer.Metadata[mcpDomain.RuntimeExtensionMetadataKey]; found {
		extension, err := extensionFromDeclaration(
			outer,
			identity.LogicalName,
		)
		if err != nil {
			return ServerDocument{}, err
		}
		output.Extension = extension
		outerExtension = true
	}
	output.Extension = withImplicitEnvironmentInputs(
		output.MCPServer,
		output.Extension,
	)
	if outerExtension && output.Extension.DisplayName != "" {
		output.DisplayName = output.Extension.DisplayName
	}
	if err := output.Validate(); err != nil {
		return ServerDocument{}, err
	}
	return output, nil
}

func overlayLocatedCore(
	input CoreServer,
	outer mcpv1.MCPDocument,
) (CoreServer, error) {
	output := cloneCore(input)
	switch outer.Transport {
	case "":
	case mcpv1.TransportStdio:
		output.Type = ServerTypeStdio
		output.URL = ""
		output.Headers = nil
	case mcpv1.TransportStreamableHTTP:
		output.Type = ServerTypeHTTP
		output.Command = ""
		output.Args = nil
		output.Env = nil
	case mcpv1.TransportSSE:
		output.Type = ServerTypeSSE
		output.Command = ""
		output.Args = nil
		output.Env = nil
	default:
		return CoreServer{}, fmt.Errorf(
			"%w: unsupported located MCP transport %q",
			basespec.ErrInvalid,
			outer.Transport,
		)
	}

	if outer.Command != "" {
		output.Command = outer.Command
	}
	if outer.Args != nil {
		output.Args = slices.Clone(outer.Args)
	}
	if outer.Env != nil {
		output.Env = maps.Clone(outer.Env)
	}
	if outer.URL != "" {
		output.URL = outer.URL
	}
	if outer.Headers != nil {
		output.Headers = maps.Clone(outer.Headers)
	}
	return NormalizeCoreServer(output), nil
}
