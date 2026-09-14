package server

import (
	"encoding/json"
	"fmt"
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/mcpv1"
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
	value := ServerDocument{
		LogicalName:    name,
		LogicalVersion: version,
		DisplayName:    displayName,
		Description:    description,
		Labels:         maps.Clone(labels),
		MCPServer:      NormalizeCoreServer(core),
		Extension: NormalizeServerExtension(
			string(name),
			extension,
		),
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

	declaration, err := declarationForDocument(input)
	if err != nil {
		return definition.Definition{}, err
	}
	body, err := declaration.CanonicalJSON()
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

	declaration, err := mcpv1.DecodeMCPJSON(input.Body)
	if err != nil {
		return ServerDocument{}, err
	}
	if declaration.Name != string(input.LogicalName) ||
		declaration.Description != input.Description {
		return ServerDocument{}, fmt.Errorf(
			"%w: MCP Definition does not match declaration header",
			basespec.ErrInvalid,
		)
	}

	core, err := coreFromDeclaration(declaration)
	if err != nil {
		return ServerDocument{}, err
	}
	extension, err := extensionFromDeclaration(
		declaration,
		input.LogicalName,
	)
	if err != nil {
		return ServerDocument{}, err
	}
	document, err := NewDocument(
		input.LogicalName,
		input.LogicalVersion,
		input.DisplayName,
		input.Description,
		input.Labels,
		core,
		extension,
	)
	if err != nil {
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

	declaration := mcpv1.MCPDocument{
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
	}
	switch input.MCPServer.Type {
	case ServerTypeStdio:
		declaration.Transport = mcpv1.TransportStdio
	case ServerTypeHTTP:
		declaration.Transport = mcpv1.TransportStreamableHTTP
	default:
		return mcpv1.MCPDocument{}, fmt.Errorf(
			"%w: unsupported MCP server transport %q",
			basespec.ErrInvalid,
			input.MCPServer.Type,
		)
	}
	if err := declaration.Validate(); err != nil {
		return mcpv1.MCPDocument{}, err
	}
	return declaration, nil
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
	case mcpv1.TransportSSE:
		return CoreServer{}, fmt.Errorf(
			"%w: MCP SSE transport requires an MCP runtime adapter",
			basespec.ErrUnsupported,
		)
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

func LegacyServerDocument(
	name basespec.LogicalName,
	logicalVersion basespec.LogicalVersion,
	displayName string,
	description string,
	labels map[string]string,
	core CoreServer,
	extension ServerExtension,
) (ServerDocument, error) {
	return NewDocument(
		name,
		logicalVersion,
		displayName,
		description,
		labels,
		core,
		extension,
	)
}

func PolicyReferenceSelector(
	name basespec.LogicalName,
) definition.Selector {
	return definition.Selector{
		Kind:        mcpDomain.MCPPolicyArtifactKind,
		LogicalName: name,
	}
}
