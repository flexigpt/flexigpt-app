package sourceformat

import (
	"encoding/json"
	"fmt"
	"maps"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/mcpv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type Decoded struct {
	SubresourceLocator basespec.SubresourceLocator
	Definition         definition.Definition
}

type configDocument struct {
	Kind       string                  `json:"kind,omitempty"`
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

// IsRetiredMCPCollection identifies only the removed proprietary source
// format. It is retained solely to produce an explicit unsupported-format
// diagnostic, never to decode or convert the document.
func IsRetiredMCPCollection(
	raw []byte,
) bool {
	var header struct {
		Kind string `json:"kind"`
	}
	return json.Unmarshal(raw, &header) == nil &&
		isRetiredMCPCollectionKind(header.Kind)
}

func isRetiredMCPCollectionKind(
	kind string,
) bool {
	return kind == "mcp.collection" || kind == "mcp.bundle"
}

// IsMCPConfig identifies the standard .mcp.json / mcp.json multi-server
// shape. Retired proprietary collection documents are explicitly excluded.
func IsMCPConfig(
	raw []byte,
) bool {
	var value struct {
		Kind       string          `json:"kind"`
		MCPServers json.RawMessage `json:"mcpServers"`
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		return false
	}
	return !isRetiredMCPCollectionKind(value.Kind) &&
		len(value.MCPServers) != 0
}

// DecodeMCPConfig decodes the standard multi-server .mcp.json source shape.
// Every configured server becomes a canonical mcp Artifact at a stable
// mcpServers/<name> source subresource.
func DecodeMCPConfig(
	raw []byte,
) ([]Decoded, error) {
	config, err := decodeMCPConfigDocument(raw)
	if err != nil {
		return nil, err
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

		transport := input.Transport
		if input.Type != "" {
			transport = input.Type
		}
		var t mcpv1.Transport
		switch transport {
		case "":
			if input.URL != "" {
				t = mcpv1.TransportStreamableHTTP
			} else {
				t = mcpv1.TransportStdio
			}
		case "stdio":
			t = mcpv1.TransportStdio

		case "http", "streamableHTTP":
			t = mcpv1.TransportStreamableHTTP

		default:
			return nil, fmt.Errorf(
				"%w: MCP config server %q has unsupported type %q",
				basespec.ErrInvalid,
				name,
				transport,
			)
		}

		document := mcpv1.MCPDocument{
			Type:        mcpv1.MCPType,
			Name:        string(logicalName),
			DisplayName: name,
			Transport:   t,
			Command:     input.Command,
			Args:        append([]string(nil), input.Args...),
			Env:         maps.Clone(input.Env),
			URL:         input.URL,
			Headers:     maps.Clone(input.Headers),
		}
		if err := document.Validate(); err != nil {
			return nil, err
		}
		definitionValue, err := mcpv1.DefinitionForDeclaration(document)
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
	if isRetiredMCPCollectionKind(value.Kind) {
		return configDocument{}, fmt.Errorf(
			"%w: proprietary MCP collection manifests are not supported",
			basespec.ErrUnsupported,
		)
	}
	return value, nil
}
