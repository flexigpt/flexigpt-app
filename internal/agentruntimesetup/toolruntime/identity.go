package toolruntime

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const maximumToolSchemaBytes = 1 << 20

// StableChoiceID creates one durable provider Tool choice identity from a
// namespace and semantic inputs. Both Session runtimes use this function.
func StableChoiceID(prefix string, values ...string) string {
	inputLength := len(prefix) + len(values)
	for _, value := range values {
		inputLength += len(value)
	}

	input := make([]byte, 0, inputLength)
	input = append(input, prefix...)
	for _, value := range values {
		input = append(input, 0)
		input = append(input, value...)
	}

	digest := strings.TrimPrefix(
		string(cryptoutil.DigestBytes(input)),
		cryptoutil.DigestSHA256Prefix,
	)
	return prefix + "-" + digest[:16]
}

// InputSchema validates and decodes a JSON Schema object used for function
// Tool arguments. A missing or true schema means an unconstrained object.
// False cannot be represented safely as an advertised function schema.
func InputSchema(raw json.RawMessage) (map[string]any, error) {
	raw = bytes.TrimSpace(raw)
	switch {
	case len(raw) == 0, bytes.Equal(raw, []byte("true")):
		return map[string]any{"type": "object"}, nil
	case bytes.Equal(raw, []byte("false")):
		return nil, errors.New("false Tool input schema cannot be advertised")
	}

	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		maximumToolSchemaBytes,
	)
	if err != nil {
		return nil, err
	}

	value, err := jsonutil.DecodeJSONRaw[map[string]any](
		json.RawMessage(canonical),
	)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return map[string]any{"type": "object"}, nil
	}
	return value, nil
}
