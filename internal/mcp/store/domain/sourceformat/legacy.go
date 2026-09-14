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

type legacyBundle struct {
	Kind          string `json:"kind"`
	SchemaID      string `json:"schemaID"`
	SchemaVersion string `json:"schemaVersion"`

	MCPServers      map[string]mcpDomainServer.CoreServer `json:"mcpServers"`
	BundleExtension legacyBundleExtension                 `json:"bundleExtension"`
}

type legacyBundleExtension struct {
	Servers  map[string]mcpDomainServer.ServerExtension `json:"servers,omitempty"`
	Policies map[string]json.RawMessage                 `json:"policies,omitempty"`
}

type legacyPolicy struct {
	LogicalName string                    `json:"logicalName"`
	Description string                    `json:"description,omitempty"`
	Body        mcppolicyv1.MCPPolicyBody `json:"body"`
}

func IsLegacyBundle(
	raw []byte,
) bool {
	var header struct {
		Kind string `json:"kind"`
	}
	return json.Unmarshal(raw, &header) == nil &&
		header.Kind == "mcp.bundle"
}

// DecodeLegacyBundle preserves the old physical mcps.json input format while
// normalizing every contained server and policy into ordinary flat Artifacts.
//
// Callers that need the final bundle semantics should use
// DecodeLegacyBundleWithCollection.
func DecodeLegacyBundle(
	raw []byte,
) ([]Decoded, error) {
	var bundle legacyBundle
	if err := jsonutil.DecodeCanonicalObjectInto(
		raw,
		&bundle,
		basespec.MaxDefinitionBytes,
	); err != nil {
		return nil, err
	}
	if bundle.Kind != "mcp.bundle" ||
		bundle.SchemaID != "mcp.bundle.v1" ||
		bundle.SchemaVersion != "v1" {
		return nil, fmt.Errorf(
			"%w: unsupported legacy MCP package format",
			basespec.ErrInvalid,
		)
	}
	if len(bundle.MCPServers) == 0 {
		return nil, fmt.Errorf(
			"%w: legacy MCP package has no servers",
			basespec.ErrInvalid,
		)
	}

	serverNames := make([]string, 0, len(bundle.MCPServers))
	for name := range bundle.MCPServers {
		serverNames = append(serverNames, name)
	}
	sort.Strings(serverNames)

	policyNames := make([]string, 0, len(bundle.BundleExtension.Policies))
	for name := range bundle.BundleExtension.Policies {
		policyNames = append(policyNames, name)
	}
	sort.Strings(policyNames)

	output := make([]Decoded, 0, len(serverNames)+len(policyNames))
	for _, name := range serverNames {
		logicalName := basespec.LogicalName(name)
		if err := logicalName.Validate(); err != nil {
			return nil, err
		}
		extension := bundle.BundleExtension.Servers[name]
		document, err := mcpDomainServer.LegacyServerDocument(
			logicalName,
			extension.LogicalVersion,
			extension.DisplayName,
			extension.Description,
			maps.Clone(extension.Labels),
			bundle.MCPServers[name],
			extension,
		)
		if err != nil {
			return nil, fmt.Errorf("legacy MCP server %q: %w", name, err)
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
		var legacy legacyPolicy
		if err := jsonutil.DecodeCanonicalObjectInto(
			bundle.BundleExtension.Policies[name],
			&legacy,
			basespec.MaxDefinitionBodyBytes,
		); err != nil {
			return nil, fmt.Errorf("legacy MCP policy %q: %w", name, err)
		}
		if legacy.LogicalName != "" &&
			legacy.LogicalName != name {
			return nil, fmt.Errorf(
				"%w: legacy MCP policy key %q differs from logicalName %q",
				basespec.ErrInvalid,
				name,
				legacy.LogicalName,
			)
		}
		document, err := mcpDomainPolicy.DocumentFromLegacyBody(
			logicalName,
			legacy.Description,
			legacy.Body,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"legacy MCP policy %q: %w",
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

// DecodeLegacyBundleWithCollection normalizes one legacy MCP bundle into its
// ordinary MCP and MCP policy Artifacts plus one canonical Collection Artifact.
//
// The legacy format has no portable bundle name, so the caller supplies a
// deterministic name derived from its physical source origin.
func DecodeLegacyBundleWithCollection(
	raw []byte,
	collectionName basespec.LogicalName,
) ([]Decoded, error) {
	if err := collectionName.Validate(); err != nil {
		return nil, err
	}
	output, err := DecodeLegacyBundle(raw)
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
	Type    string            `json:"type,omitempty"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
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
	var config configDocument
	if err := jsonutil.DecodeCanonicalObjectInto(
		raw,
		&config,
		basespec.MaxDefinitionBytes,
	); err != nil {
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
		switch input.Type {
		case "", "stdio":
			core.Type = mcpDomainServer.ServerTypeStdio
		case "http", "streamable-http":
			core.Type = mcpDomainServer.ServerTypeHTTP
		case "sse":
			decl := mcpv1.MCPDocument{
				APIVersion: mcpv1.MCPSchemaVersion,
				Type:       mcpv1.MCPType,
				Name:       name,
				Transport:  mcpv1.TransportSSE,
				Command:    input.Command,
				Args:       append([]string(nil), input.Args...),
				Env:        maps.Clone(input.Env),
				URL:        input.URL,
				Headers:    maps.Clone(input.Headers),
			}
			definitionValue, err := mcpv1.DefinitionForDeclaration(decl)
			if err != nil {
				return nil, fmt.Errorf(
					"decode MCP config SSE server %q: %w",
					name,
					err,
				)
			}
			output = append(output, Decoded{
				SubresourceLocator: basespec.SubresourceLocator(
					"mcpServers/" + name,
				),
				Definition: definitionValue,
			})
			continue

		default:
			return nil, fmt.Errorf(
				"%w: MCP config server %q has unsupported type %q",
				basespec.ErrInvalid,
				name,
				input.Type,
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

func MCPDocumentFromCanonical(
	input mcpv1.MCPDocument,
) (definition.Definition, error) {
	return mcpDomainServer.DefinitionForMCPDeclaration(input)
}
