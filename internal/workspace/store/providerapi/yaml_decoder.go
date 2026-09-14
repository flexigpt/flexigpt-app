package providerapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"path"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v4"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const CanonicalYAMLDecoderID basespec.DecoderID = "workspace-artifact-yaml"

const (
	maxYAMLAliases = 128
	maxYAMLNodes   = 100_000
)

type YAMLDecoder struct{}

func NewYAMLDecoder() *YAMLDecoder {
	return &YAMLDecoder{}
}

func (*YAMLDecoder) ID() basespec.DecoderID {
	return CanonicalYAMLDecoderID
}

func (*YAMLDecoder) Revision() string {
	return "workspace-artifact-yaml/v1"
}

func (*YAMLDecoder) Recognize(
	_ context.Context,
	candidate providerapi.Candidate,
) providerapi.Recognition {
	extension := strings.ToLower(path.Ext(string(candidate.Locator)))
	if extension != ".yaml" && extension != ".yml" {
		return providerapi.RecognitionNone
	}
	raw, err := canonicalJSONFromYAML(candidate.Content)
	if err != nil {
		return providerapi.RecognitionNone
	}
	var header struct {
		Type artifactcontract.Type `json:"type"`
	}
	if err := json.Unmarshal(raw, &header); err != nil {
		return providerapi.RecognitionNone
	}
	if err := header.Type.Validate(); err != nil {
		return providerapi.RecognitionNone
	}
	return providerapi.RecognitionPreferred
}

func (*YAMLDecoder) Decode(
	_ context.Context,
	candidate providerapi.Candidate,
) ([]providerapi.Decoded, []diagnostic.Diagnostic) {
	raw, err := canonicalJSONFromYAML(candidate.Content)
	if err != nil {
		return nil, yamlDiagnostic(candidate.Locator, "", err)
	}
	root, err := artifactcontract.DecodeEntryJSON(raw)
	if err != nil {
		return nil, yamlDiagnostic(candidate.Locator, "", err)
	}
	entries, err := artifactcontract.WalkNamedEntries(root)
	if err != nil {
		return nil, yamlDiagnostic(candidate.Locator, "", err)
	}

	output := make([]providerapi.Decoded, 0, len(entries))
	for _, entry := range entries {
		value, err := DefinitionForEntry(entry.Entry)
		if err != nil {
			output = append(output, providerapi.Decoded{
				SubresourceLocator: entry.SubresourceLocator,
				Diagnostics: yamlDiagnostic(
					candidate.Locator,
					entry.SubresourceLocator,
					err,
				),
			})
			continue
		}
		output = append(output, providerapi.Decoded{
			SubresourceLocator: entry.SubresourceLocator,
			Definition:         value,
		})
	}
	return output, nil
}

func yamlDiagnostic(
	locator basespec.Locator,
	subresource basespec.SubresourceLocator,
	err error,
) []diagnostic.Diagnostic {
	location := &diagnostic.Location{
		Locator: locator,
	}
	if subresource != "" {
		location.SubresourceLocator = subresource
	}
	return []diagnostic.Diagnostic{{
		Severity: diagnostic.SeverityError,
		Code:     "workspace.artifact-yaml-invalid",
		Message:  diagnostic.BoundedMessage(err.Error()),
		Location: location,
	}}
}

type yamlConverter struct {
	nodes   int
	aliases int
	active  map[*yaml.Node]struct{}
}

func canonicalJSONFromYAML(raw []byte) ([]byte, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		return nil, err
	}
	var trailing yaml.Node
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New(
				"canonical Artifact YAML cannot contain multiple documents",
			)
		}
		return nil, err
	}
	if document.Kind != yaml.DocumentNode ||
		len(document.Content) != 1 {
		return nil, errors.New(
			"canonical Artifact YAML must contain one document",
		)
	}

	converter := yamlConverter{
		active: make(map[*yaml.Node]struct{}),
	}
	value, err := converter.convert(document.Content[0], 0)
	if err != nil {
		return nil, err
	}
	return jsonutil.MarshalCanonicalObject(
		value,
		basespec.MaxDefinitionBytes,
	)
}

func (c *yamlConverter) convert(
	node *yaml.Node,
	depth int,
) (any, error) {
	if node == nil {
		return nil, errors.New("YAML node is nil")
	}
	c.nodes++
	if c.nodes > maxYAMLNodes {
		return nil, fmt.Errorf(
			"YAML exceeds %d nodes",
			maxYAMLNodes,
		)
	}
	if depth > basespec.MaxDiscoveryDepth {
		return nil, fmt.Errorf(
			"YAML exceeds %d nesting levels",
			basespec.MaxDiscoveryDepth,
		)
	}

	if node.Kind == yaml.AliasNode {
		c.aliases++
		if c.aliases > maxYAMLAliases {
			return nil, fmt.Errorf(
				"YAML exceeds %d aliases",
				maxYAMLAliases,
			)
		}
		if _, active := c.active[node.Alias]; active {
			return nil, errors.New("YAML alias cycle")
		}
		c.active[node.Alias] = struct{}{}
		value, err := c.convert(node.Alias, depth+1)
		delete(c.active, node.Alias)
		return value, err
	}

	switch node.Kind {
	case yaml.MappingNode:
		if len(node.Content)%2 != 0 {
			return nil, errors.New("YAML mapping has an incomplete key/value pair")
		}
		output := make(map[string]any, len(node.Content)/2)
		for index := 0; index < len(node.Content); index += 2 {
			key := node.Content[index]
			if key.Kind != yaml.ScalarNode ||
				key.Tag != "!!str" {
				return nil, errors.New(
					"canonical Artifact YAML requires string object keys",
				)
			}
			if _, duplicate := output[key.Value]; duplicate {
				return nil, fmt.Errorf(
					"duplicate YAML object key %q",
					key.Value,
				)
			}
			value, err := c.convert(node.Content[index+1], depth+1)
			if err != nil {
				return nil, err
			}
			output[key.Value] = value
		}
		return output, nil

	case yaml.SequenceNode:
		output := make([]any, len(node.Content))
		for index, child := range node.Content {
			value, err := c.convert(child, depth+1)
			if err != nil {
				return nil, err
			}
			output[index] = value
		}
		return output, nil

	case yaml.ScalarNode:
		return yamlScalar(node)

	default:
		return nil, fmt.Errorf(
			"unsupported YAML node kind %d",
			node.Kind,
		)
	}
}

func yamlScalar(node *yaml.Node) (any, error) {
	switch node.Tag {
	case "!!null":
		//nolint:nilnil // Exlicit.
		return nil, nil

	case "!!str", "":
		return node.Value, nil

	case "!!bool":
		value, err := strconv.ParseBool(strings.ToLower(node.Value))
		if err != nil {
			return nil, err
		}
		return value, nil

	case "!!int":
		if value, err := strconv.ParseInt(node.Value, 0, 64); err == nil {
			return value, nil
		}
		value, err := strconv.ParseUint(node.Value, 0, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid YAML integer %q", node.Value)
		}
		return value, nil

	case "!!float":
		value, err := strconv.ParseFloat(node.Value, 64)
		if err != nil ||
			math.IsInf(value, 0) ||
			math.IsNaN(value) {
			return nil, fmt.Errorf("invalid YAML float %q", node.Value)
		}
		return value, nil

	default:
		return nil, fmt.Errorf(
			"unsupported YAML scalar tag %q",
			node.Tag,
		)
	}
}
