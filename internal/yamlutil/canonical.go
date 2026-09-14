// Package yamlutil converts one bounded YAML mapping document into canonical
// JSON object bytes.
//
// It intentionally supports only the YAML subset that can be represented
// safely as JSON. In particular, object keys must be strings, duplicate keys
// are rejected, aliases are bounded, and YAML-only numeric forms are rejected.
package yamlutil

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"go.yaml.in/yaml/v4"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	maximumAliases = 128
	maximumNodes   = 100_000
	maximumDepth   = 256
)

// CanonicalObjectJSON converts one YAML mapping document into canonical JSON.
//
// The maximum applies to both YAML input bytes and canonical JSON output bytes.
// The caller owns domain-specific limits beyond this conversion boundary.
func CanonicalObjectJSON(
	raw []byte,
	maximumBytes int,
) ([]byte, error) {
	if maximumBytes <= 0 {
		return nil, errors.New("YAML byte limit is invalid")
	}
	if len(raw) == 0 {
		return nil, errors.New("YAML document is empty")
	}
	if len(raw) > maximumBytes {
		return nil, fmt.Errorf(
			"YAML document exceeds %d bytes",
			maximumBytes,
		)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(raw))

	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, errors.New("YAML document is empty")
		}
		return nil, fmt.Errorf("decode YAML document: %w", err)
	}

	var trailing yaml.Node
	if err := decoder.Decode(&trailing); err != nil {
		if !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf(
				"decode trailing YAML document: %w",
				err,
			)
		}
	} else {
		return nil, errors.New(
			"YAML input cannot contain multiple documents",
		)
	}

	if document.Kind != yaml.DocumentNode ||
		len(document.Content) != 1 {
		return nil, errors.New(
			"YAML input must contain exactly one document",
		)
	}

	state := converter{
		activeAliases: make(map[*yaml.Node]struct{}),
	}
	value, err := state.convert(document.Content[0], 0)
	if err != nil {
		return nil, err
	}

	canonical, err := jsonutil.MarshalCanonicalObject(
		value,
		maximumBytes,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"convert YAML document to canonical JSON object: %w",
			err,
		)
	}
	return append([]byte(nil), canonical...), nil
}

type converter struct {
	nodes         int
	aliases       int
	activeAliases map[*yaml.Node]struct{}
}

func (c *converter) convert(
	node *yaml.Node,
	depth int,
) (any, error) {
	if node == nil {
		return nil, errors.New("YAML node is nil")
	}
	c.nodes++
	if c.nodes > maximumNodes {
		return nil, fmt.Errorf(
			"YAML document exceeds %d nodes",
			maximumNodes,
		)
	}
	if depth > maximumDepth {
		return nil, fmt.Errorf(
			"YAML document exceeds %d nesting levels",
			maximumDepth,
		)
	}

	if node.Kind == yaml.AliasNode {
		if node.Alias == nil {
			return nil, errors.New("YAML alias has no target")
		}
		c.aliases++
		if c.aliases > maximumAliases {
			return nil, fmt.Errorf(
				"YAML document exceeds %d aliases",
				maximumAliases,
			)
		}
		if _, active := c.activeAliases[node.Alias]; active {
			return nil, errors.New("YAML alias cycle")
		}
		c.activeAliases[node.Alias] = struct{}{}
		value, err := c.convert(node.Alias, depth+1)
		delete(c.activeAliases, node.Alias)
		return value, err
	}

	switch node.Kind {
	case yaml.MappingNode:
		if len(node.Content)%2 != 0 {
			return nil, errors.New(
				"YAML mapping has an incomplete key/value pair",
			)
		}

		output := make(map[string]any, len(node.Content)/2)
		for index := 0; index < len(node.Content); index += 2 {
			key := node.Content[index]
			if key.Kind != yaml.ScalarNode ||
				(key.Tag != "!!str" && key.Tag != "") {
				return nil, errors.New(
					"YAML object keys must be strings",
				)
			}
			if _, duplicate := output[key.Value]; duplicate {
				return nil, fmt.Errorf(
					"duplicate YAML object key %q",
					key.Value,
				)
			}

			value, err := c.convert(
				node.Content[index+1],
				depth+1,
			)
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
		return convertScalar(node)

	default:
		return nil, fmt.Errorf(
			"unsupported YAML node kind %d",
			node.Kind,
		)
	}
}

func convertScalar(
	node *yaml.Node,
) (any, error) {
	switch node.Tag {
	case "!!null":
		//nolint:nilnil // Explicit.
		return nil, nil

	case "", "!!str":
		return node.Value, nil

	case "!!bool":
		switch strings.ToLower(node.Value) {
		case "true":
			return true, nil
		case "false":
			return false, nil
		default:
			return nil, fmt.Errorf(
				"invalid YAML boolean %q",
				node.Value,
			)
		}

	case "!!int", "!!float":
		// Do not parse through float64. That would corrupt exact integer
		// values above IEEE-754 precision. Portable YAML numeric values
		// must use JSON number syntax.
		if !json.Valid([]byte(
			`{"value":` + node.Value + `}`,
		)) {
			return nil, fmt.Errorf(
				"YAML number %q is not valid JSON number syntax",
				node.Value,
			)
		}
		return json.Number(node.Value), nil

	default:
		return nil, fmt.Errorf(
			"unsupported YAML scalar tag %q",
			node.Tag,
		)
	}
}
