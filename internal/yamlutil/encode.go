package yamlutil

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"go.yaml.in/yaml/v4"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	tagStr  = "!!str"
	tagBool = "!!bool"
)

// CanonicalObjectYAML converts one JSON object into deterministic YAML.
//
// The input is canonicalized through jsonutil first. The function is intended
// for portable declaration export and deliberately accepts only JSON objects.
func CanonicalObjectYAML(
	raw []byte,
	maximumBytes int,
) ([]byte, error) {
	canonical, err := jsonutil.CanonicalizeObject(raw, maximumBytes)
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.UseNumber()

	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, errors.New("json contains trailing values")
		}
		return nil, err
	}

	node, err := yamlNodeForJSON(value)
	if err != nil {
		return nil, err
	}

	var output bytes.Buffer
	encoder := yaml.NewEncoder(&output)
	encoder.SetIndent(2)
	if err := encoder.Encode(node); err != nil {
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}

	valueBytes := output.Bytes()
	if len(valueBytes) == 0 || len(valueBytes) > maximumBytes {
		return nil, fmt.Errorf(
			"YAML output exceeds %d bytes",
			maximumBytes,
		)
	}
	return append([]byte(nil), valueBytes...), nil
}

func yamlNodeForJSON(
	value any,
) (*yaml.Node, error) {
	switch typed := value.(type) {
	case nil:
		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!null",
			Value: "null",
		}, nil

	case bool:
		if typed {
			return &yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   tagBool,
				Value: "true",
			}, nil
		}
		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   tagBool,
			Value: "false",
		}, nil

	case string:
		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   tagStr,
			Value: typed,
		}, nil

	case json.Number:
		tag := "!!float"
		if !strings.ContainsAny(string(typed), ".eE") {
			tag = "!!int"
		}
		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   tag,
			Value: string(typed),
		}, nil

	case []any:
		output := &yaml.Node{
			Kind: yaml.SequenceNode,
			Tag:  "!!seq",
		}
		for _, child := range typed {
			valueNode, err := yamlNodeForJSON(child)
			if err != nil {
				return nil, err
			}
			output.Content = append(output.Content, valueNode)
		}
		return output, nil

	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		output := &yaml.Node{
			Kind: yaml.MappingNode,
			Tag:  "!!map",
		}
		for _, key := range keys {
			valueNode, err := yamlNodeForJSON(typed[key])
			if err != nil {
				return nil, err
			}
			output.Content = append(
				output.Content,
				&yaml.Node{
					Kind:  yaml.ScalarNode,
					Tag:   tagStr,
					Value: key,
				},
				valueNode,
			)
		}
		return output, nil

	default:
		return nil, fmt.Errorf(
			"unsupported canonical JSON value type %T",
			value,
		)
	}
}
