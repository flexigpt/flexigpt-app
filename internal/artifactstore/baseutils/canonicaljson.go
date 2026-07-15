package baseutils

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"math"
	"math/big"
	"sort"
	"strconv"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/validate"
)

const placeholderDefinitionDigest spec.Digest = "sha256:0000000000000000000000000000000000000000000000000000000000000000"

// CanonicalizeDefinition normalizes portable JSON fields and calculates the
// digest of the immutable definition envelope. A supplied non-empty digest
// must match the calculated value.
func CanonicalizeDefinition(in spec.CanonicalDefinition) (spec.CanonicalDefinition, error) {
	out := cloneCanonicalDefinition(in)
	if out.Digest == "" {
		out.Digest = placeholderDefinitionDigest
	}
	if err := validate.ValidateCanonicalDefinition(out); err != nil {
		return spec.CanonicalDefinition{}, fmt.Errorf("canonical definition structure: %w", err)
	}
	sort.Slice(out.AssetManifest, func(left, right int) bool {
		return out.AssetManifest[left].Path < out.AssetManifest[right].Path
	})

	var err error
	out.Extensions, err = CanonicalizeJSON(out.Extensions)
	if err != nil {
		return spec.CanonicalDefinition{}, fmt.Errorf("canonicalize definition extensions: %w", err)
	}
	out.DefinitionJSON, err = CanonicalizeJSON(out.DefinitionJSON)
	if err != nil {
		return spec.CanonicalDefinition{}, fmt.Errorf("canonicalize definition JSON: %w", err)
	}
	if err := validate.ValidateCanonicalDefinition(out); err != nil {
		return spec.CanonicalDefinition{}, fmt.Errorf("canonical definition after ordering: %w", err)
	}

	payload, err := canonicalDefinitionPayload(out)
	if err != nil {
		return spec.CanonicalDefinition{}, err
	}
	digest := DigestBytes(payload)
	if in.Digest != "" && in.Digest != digest {
		return spec.CanonicalDefinition{}, fmt.Errorf(
			"%w: supplied %q, calculated %q",
			spec.ErrDigestMismatch,
			in.Digest,
			digest,
		)
	}
	out.Digest = digest
	if err := validate.ValidateCanonicalDefinition(out); err != nil {
		return spec.CanonicalDefinition{}, fmt.Errorf("canonical definition after normalization: %w", err)
	}
	return out, nil
}

func canonicalDefinitionPayload(definition spec.CanonicalDefinition) ([]byte, error) {
	envelope := struct {
		Kind                spec.ArtifactKind         `json:"kind"`
		SchemaID            spec.SchemaID             `json:"schemaID"`
		SchemaVersion       string                    `json:"schemaVersion"`
		LogicalName         spec.LogicalName          `json:"logicalName"`
		LogicalVersion      spec.LogicalVersion       `json:"logicalVersion,omitempty"`
		DisplayName         string                    `json:"displayName,omitempty"`
		Description         string                    `json:"description,omitempty"`
		Labels              map[string]string         `json:"labels,omitempty"`
		Extensions          json.RawMessage           `json:"extensions"`
		DefinitionJSON      json.RawMessage           `json:"definitionJSON"`
		DependencySelectors []spec.ArtifactSelector   `json:"dependencySelectors,omitempty"`
		AssetManifest       []spec.AssetManifestEntry `json:"assetManifest,omitempty"`
	}{
		Kind:                definition.Kind,
		SchemaID:            definition.SchemaID,
		SchemaVersion:       definition.SchemaVersion,
		LogicalName:         definition.LogicalName,
		LogicalVersion:      definition.LogicalVersion,
		DisplayName:         definition.DisplayName,
		Description:         definition.Description,
		Labels:              definition.Labels,
		Extensions:          definition.Extensions,
		DefinitionJSON:      definition.DefinitionJSON,
		DependencySelectors: definition.DependencySelectors,
		AssetManifest:       definition.AssetManifest,
	}
	raw, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("marshal canonical definition envelope: %w", err)
	}
	canonical, err := CanonicalizeJSON(raw)
	if err != nil {
		return nil, fmt.Errorf("canonicalize definition envelope: %w", err)
	}
	return canonical, nil
}

// CanonicalizeJSON returns deterministic JSON bytes. Objects are ordered by
// UTF-8 key order, arrays retain their input order, and exactly round-trippable
// JSON numbers are normalized through IEEE-754 binary64 representation.
// Numbers that would lose precision are rejected instead of being allowed to
// alias another canonical digest. This is a stable Artifact Store encoding,
// not a claim of general-purpose JCS conformance.
func CanonicalizeJSON(raw []byte) ([]byte, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()

	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("decode JSON: %w", err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return nil, err
	}

	var out bytes.Buffer
	if err := appendCanonicalJSON(&out, value); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func DigestBytes(value []byte) spec.Digest {
	sum := sha256.Sum256(value)
	return spec.Digest("sha256:" + hex.EncodeToString(sum[:]))
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if err == io.EOF {
		return nil
	}
	if err == nil {
		return errors.New("JSON contains trailing values")
	}
	return fmt.Errorf("decode trailing JSON: %w", err)
}

func appendCanonicalJSON(out *bytes.Buffer, value any) error {
	switch typed := value.(type) {
	case nil:
		out.WriteString("null")
	case bool:
		if typed {
			out.WriteString("true")
		} else {
			out.WriteString("false")
		}
	case string:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return err
		}
		out.Write(encoded)
	case json.Number:
		encoded, err := canonicalJSONNumber(string(typed))
		if err != nil {
			return err
		}
		out.WriteString(encoded)
	case float64:
		encoded, err := canonicalFloat(typed)
		if err != nil {
			return err
		}
		out.WriteString(encoded)
	case []any:
		out.WriteByte('[')
		for index, item := range typed {
			if index > 0 {
				out.WriteByte(',')
			}
			if err := appendCanonicalJSON(out, item); err != nil {
				return err
			}
		}
		out.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		out.WriteByte('{')
		for index, key := range keys {
			if index > 0 {
				out.WriteByte(',')
			}
			encodedKey, err := json.Marshal(key)
			if err != nil {
				return err
			}
			out.Write(encodedKey)
			out.WriteByte(':')
			if err := appendCanonicalJSON(out, typed[key]); err != nil {
				return err
			}
		}
		out.WriteByte('}')
	default:
		return fmt.Errorf("unsupported JSON value type %T", value)
	}
	return nil
}

func canonicalJSONNumber(value string) (string, error) {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return "", fmt.Errorf("parse JSON number %q: %w", value, err)
	}
	canonical, err := canonicalFloat(parsed)
	if err != nil {
		return "", err
	}
	originalValue, err := exactJSONNumber(value)
	if err != nil {
		return "", fmt.Errorf("parse exact JSON number %q: %w", value, err)
	}
	canonicalValue, err := exactJSONNumber(canonical)
	if err != nil {
		return "", fmt.Errorf("parse canonical JSON number %q: %w", canonical, err)
	}
	if originalValue.Cmp(canonicalValue) != 0 {
		return "", fmt.Errorf(
			"JSON number %q cannot be represented without precision loss",
			value,
		)
	}
	return canonical, nil
}

func canonicalFloat(value float64) (string, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return "", errors.New("non-finite JSON number")
	}
	if value == 0 {
		return "0", nil
	}
	return strconv.FormatFloat(value, 'g', -1, 64), nil
}

func exactJSONNumber(value string) (*big.Rat, error) {
	sign := 1
	if strings.HasPrefix(value, "-") {
		sign = -1
		value = strings.TrimPrefix(value, "-")
	}

	exponent := 0
	if index := strings.IndexAny(value, "eE"); index >= 0 {
		parsedExponent, err := strconv.Atoi(value[index+1:])
		if err != nil {
			return nil, err
		}
		if parsedExponent < -100_000 || parsedExponent > 100_000 {
			return nil, errors.New("JSON exponent is outside the supported range")
		}
		exponent = parsedExponent
		value = value[:index]
	}
	fractionDigits := 0
	if index := strings.IndexByte(value, '.'); index >= 0 {
		fractionDigits = len(value) - index - 1
		value = value[:index] + value[index+1:]
	}
	if value == "" {
		return nil, errors.New("JSON number has no digits")
	}

	numerator := new(big.Int)
	if _, ok := numerator.SetString(value, 10); !ok {
		return nil, errors.New("JSON number has invalid digits")
	}
	if sign < 0 {
		numerator.Neg(numerator)
	}

	scale := fractionDigits - exponent
	if scale <= 0 {
		multiplier := new(big.Int).Exp(
			big.NewInt(10),
			big.NewInt(int64(-scale)),
			nil,
		)
		numerator.Mul(numerator, multiplier)
		return new(big.Rat).SetInt(numerator), nil
	}

	denominator := new(big.Int).Exp(
		big.NewInt(10),
		big.NewInt(int64(scale)),
		nil,
	)
	return new(big.Rat).SetFrac(numerator, denominator), nil
}

func cloneCanonicalDefinition(in spec.CanonicalDefinition) spec.CanonicalDefinition {
	out := in
	out.Labels = cloneStringMap(in.Labels)
	out.Extensions = cloneRawJSON(in.Extensions)
	out.DefinitionJSON = cloneRawJSON(in.DefinitionJSON)
	out.DependencySelectors = make([]spec.ArtifactSelector, len(in.DependencySelectors))
	for index, selector := range in.DependencySelectors {
		out.DependencySelectors[index] = selector
		out.DependencySelectors[index].Labels = cloneStringMap(selector.Labels)
	}
	out.AssetManifest = append([]spec.AssetManifestEntry(nil), in.AssetManifest...)
	return out
}

func cloneRawJSON(in json.RawMessage) json.RawMessage {
	if in == nil {
		return nil
	}
	return append(json.RawMessage(nil), in...)
}

func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	maps.Copy(out, in)
	return out
}
