package sourceformat

import (
	"encoding/json"
	"fmt"
	"maps"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/collectionv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcppolicyv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcpv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/decoder"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	mcpDomainPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/policy"
	mcpDomainServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/server"
)

type Decoded struct {
	SubresourceLocator basespec.SubresourceLocator
	Definition         definition.Definition
}

type mcpCollectionManifest struct {
	Kind          string `json:"kind"`
	SchemaID      string `json:"schemaID"`
	SchemaVersion string `json:"schemaVersion"`

	MCPServers          map[string]mcpDomainServer.CoreServer `json:"mcpServers"`
	CollectionExtension mcpCollectionExtension                `json:"bundleExtension"`
}

type mcpCollectionExtension struct {
	Servers  map[string]mcpDomainServer.ServerExtension `json:"servers,omitempty"`
	Policies map[string]json.RawMessage                 `json:"policies,omitempty"`
}

type sourcePolicy struct {
	LogicalName string                    `json:"logicalName"`
	Description string                    `json:"description,omitempty"`
	Body        mcppolicyv1.MCPPolicyBody `json:"body"`
}

func IsMCPCollection(
	raw []byte,
) bool {
	var header struct {
		Kind string `json:"kind"`
	}
	return json.Unmarshal(raw, &header) == nil &&
		isMCPCollectionKind(header.Kind)
}

func isMCPCollectionKind(kind string) bool {
	return kind == "mcp.collection" || kind == "mcp.bundle"
}

// DecodeMCPCollection normalizes one MCP Collection source document into
// ordinary MCP and MCP Policy Artifacts.
func DecodeMCPCollection(
	raw []byte,
) ([]Decoded, error) {
	collection, err := decodeMCPCollectionManifest(raw)
	if err != nil {
		return nil, err
	}
	if len(collection.MCPServers) == 0 {
		return nil, fmt.Errorf(
			"%w: MCP Collection has no servers",
			basespec.ErrInvalid,
		)
	}

	serverNames := make([]string, 0, len(collection.MCPServers))
	for name := range collection.MCPServers {
		serverNames = append(serverNames, name)
	}
	sort.Strings(serverNames)

	policyNames := make([]string, 0, len(collection.CollectionExtension.Policies))
	for name := range collection.CollectionExtension.Policies {
		policyNames = append(policyNames, name)
	}
	sort.Strings(policyNames)

	output := make([]Decoded, 0, len(serverNames)+len(policyNames))
	for _, name := range serverNames {
		logicalName := basespec.LogicalName(name)
		if err := logicalName.Validate(); err != nil {
			return nil, err
		}
		extension := collection.CollectionExtension.Servers[name]
		document, err := mcpDomainServer.NewDocument(
			logicalName,
			extension.LogicalVersion,
			extension.DisplayName,
			extension.Description,
			maps.Clone(extension.Labels),
			collection.MCPServers[name],
			extension,
		)
		if err != nil {
			return nil, fmt.Errorf("MCP Collection server %q: %w", name, err)
		}
		definitionValue, err := mcpDomainServer.DefinitionForDocument(
			document,
		)
		if err != nil {
			return nil, err
		}
		output = append(output, Decoded{
			SubresourceLocator: basespec.SubresourceLocator(
				"mcpServers/" + name,
			),
			Definition: definitionValue,
		})
	}

	for _, name := range policyNames {
		logicalName := basespec.LogicalName(name)
		if err := logicalName.Validate(); err != nil {
			return nil, err
		}
		var sourceValue sourcePolicy
		if err := jsonutil.DecodeCanonicalObjectInto(
			collection.CollectionExtension.Policies[name],
			&sourceValue,
			basespec.MaxDefinitionBodyBytes,
		); err != nil {
			return nil, fmt.Errorf("MCP Collection policy %q: %w", name, err)
		}
		if sourceValue.LogicalName != "" &&
			sourceValue.LogicalName != name {
			return nil, fmt.Errorf(
				"%w: MCP Collection policy key %q differs from logicalName %q",
				basespec.ErrInvalid,
				name,
				sourceValue.LogicalName,
			)
		}
		document, err := mcpDomainPolicy.DocumentFromSourceBody(
			logicalName,
			sourceValue.Description,
			sourceValue.Body,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"MCP Collection policy %q: %w",
				name,
				err,
			)
		}
		definitionValue, err := mcpDomainPolicy.DefinitionForDocument(
			document,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"legacy MCP policy %q: %w",
				name,
				err,
			)
		}
		output = append(output, Decoded{
			SubresourceLocator: basespec.SubresourceLocator(
				"policies/" + name,
			),
			Definition: definitionValue,
		})
	}

	for index := range output {
		if err := output[index].SubresourceLocator.Validate(); err != nil {
			return nil, err
		}
	}
	return output, nil
}

func decodeMCPCollectionManifest(
	raw []byte,
) (mcpCollectionManifest, error) {
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxDefinitionBytes,
	)
	if err != nil {
		return mcpCollectionManifest{}, err
	}
	var value mcpCollectionManifest
	if err := json.Unmarshal(canonical, &value); err != nil {
		return mcpCollectionManifest{}, err
	}
	if !isMCPCollectionKind(value.Kind) ||
		value.SchemaVersion != "v1" {
		return mcpCollectionManifest{}, fmt.Errorf(
			"%w: unsupported MCP Collection source format",
			basespec.ErrInvalid,
		)
	}
	return value, nil
}

// DecodeMCPCollectionWithCollection emits all normalized MCP members plus one
// canonical collection Artifact. The physical source spelling is normalized
// immediately and never becomes a persisted Artifact kind.
func DecodeMCPCollectionWithCollection(
	raw []byte,
	collectionName basespec.LogicalName,
) ([]Decoded, error) {
	if err := collectionName.Validate(); err != nil {
		return nil, err
	}
	output, err := DecodeMCPCollection(raw)
	if err != nil {
		return nil, err
	}

	members := make([]declaration.Entry, 0, len(output))
	for index, value := range output {
		switch declaration.Type(value.Definition.Kind) {
		case declaration.TypeMCP, declaration.TypeMCPPolicy:
		default:
			return nil, fmt.Errorf(
				"%w: legacy MCP decoded member %d has unsupported type %q",
				basespec.ErrInvalid,
				index,
				value.Definition.Kind,
			)
		}
		member, err := declaration.NewSymbolicEntry(
			declaration.Type(value.Definition.Kind),
			value.Definition.LogicalName,
		)
		if err != nil {
			return nil, err
		}
		members = append(members, member)
	}

	document := collectionv1.CollectionDocument{
		APIVersion: collectionv1.CollectionSchemaVersion,
		Type:       collectionv1.CollectionType,
		Name:       string(collectionName),
		Members:    members,
	}
	entry, err := declaration.NewEntry(document)
	if err != nil {
		return nil, err
	}
	definitionValue, err := decoder.DefinitionForEntry(entry)
	if err != nil {
		return nil, err
	}
	return append(output, Decoded{
		SubresourceLocator: "collection",
		Definition:         definitionValue,
	}), nil
}

type configDocument struct {
	MCPServers map[string]configServer `json:"mcpServers"`
}

type configServer struct {
	Type      string            `json:"type,omitempty"`
	Transport string            `json:"transport,omitempty"`
	Command   string            `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	URL       string            `json:"url,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
}

func IsMCPConfig(
	raw []byte,
) bool {
	var value map[string]json.RawMessage
	if err := json.Unmarshal(raw, &value); err != nil {
		return false
	}
	_, found := value["mcpServers"]
	return found
}

// DecodeMCPConfig decodes the standard multi-server .mcp.json source shape.
// Every selected server becomes a separate mcp Artifact at a stable
// mcpServers/<name> source subresource.
func DecodeMCPConfig(
	raw []byte,
) ([]Decoded, error) {
	config, err := decodeMCPConfigDocument(raw)
	if err != nil {
		return nil, err
	}
	if len(config.MCPServers) == 0 {
		return nil, fmt.Errorf(
			"%w: MCP configuration has no servers",
			basespec.ErrInvalid,
		)
	}

	names := make([]string, 0, len(config.MCPServers))
	for name := range config.MCPServers {
		names = append(names, name)
	}
	sort.Strings(names)

	output := make([]Decoded, 0, len(names))
	for _, name := range names {
		logicalName := basespec.LogicalName(name)
		if err := logicalName.Validate(); err != nil {
			return nil, err
		}
		input := config.MCPServers[name]
		core := mcpDomainServer.CoreServer{
			Command: input.Command,
			Args:    append([]string(nil), input.Args...),
			Env:     maps.Clone(input.Env),
			URL:     input.URL,
			Headers: maps.Clone(input.Headers),
		}
		transport := input.Transport
		if input.Type != "" {
			transport = input.Type
		}
		switch transport {
		case "":
			if input.URL != "" {
				core.Type = mcpDomainServer.ServerTypeHTTP
			} else {
				core.Type = mcpDomainServer.ServerTypeStdio
			}
		case "stdio":
			core.Type = mcpDomainServer.ServerTypeStdio
		case "http", "streamable-http", "streamableHttp":
			core.Type = mcpDomainServer.ServerTypeHTTP
		case "sse":
			core.Type = mcpDomainServer.ServerTypeSSE
		default:
			return nil, fmt.Errorf(
				"%w: MCP config server %q has unsupported type %q",
				basespec.ErrInvalid,
				name,
				transport,
			)
		}
		document, err := mcpDomainServer.NewDocument(
			logicalName,
			"",
			name,
			"",
			nil,
			core,
			mcpDomainServer.ServerExtension{},
		)
		if err != nil {
			return nil, err
		}
		definitionValue, err := mcpDomainServer.DefinitionForDocument(
			document,
		)
		if err != nil {
			return nil, err
		}
		output = append(output, Decoded{
			SubresourceLocator: basespec.SubresourceLocator(
				"mcpServers/" + name,
			),
			Definition: definitionValue,
		})
	}
	return output, nil
}

func decodeMCPConfigDocument(
	raw []byte,
) (configDocument, error) {
	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		basespec.MaxDefinitionBytes,
	)
	if err != nil {
		return configDocument{}, err
	}
	var value configDocument
	if err := json.Unmarshal(canonical, &value); err != nil {
		return configDocument{}, err
	}
	return value, nil
}

func MCPDocumentFromCanonical(
	input mcpv1.MCPDocument,
) (definition.Definition, error) {
	return mcpDomainServer.DefinitionForMCPDeclaration(input)
}
