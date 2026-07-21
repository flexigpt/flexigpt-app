package jsoncanon

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

func Canonicalize(raw []byte) ([]byte, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()

	value, err := decodeValue(decoder)
	if err != nil {
		return nil, fmt.Errorf("decode JSON: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return nil, err
	}

	var output bytes.Buffer
	if err := appendCanonical(&output, value); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func CanonicalizeObject(raw []byte, maximum int) ([]byte, error) {
	if len(raw) == 0 {
		raw = []byte("{}")
	}
	if len(raw) > maximum {
		return nil, fmt.Errorf("JSON object exceeds %d bytes", maximum)
	}
	canonical, err := Canonicalize(raw)
	if err != nil {
		return nil, err
	}
	if len(canonical) == 0 || canonical[0] != '{' {
		return nil, errors.New("JSON value must be an object")
	}
	return canonical, nil
}

func Equal(left, right []byte) bool {
	leftCanonical, err := Canonicalize(left)
	if err != nil {
		return false
	}
	rightCanonical, err := Canonicalize(right)
	if err != nil {
		return false
	}
	return bytes.Equal(leftCanonical, rightCanonical)
}

func decodeValue(decoder *json.Decoder) (any, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}

	delimiter, isDelimiter := token.(json.Delim)
	if !isDelimiter {
		return token, nil
	}

	switch delimiter {
	case '{':
		object := make(map[string]any)
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return nil, fmt.Errorf("decode object key: %w", err)
			}
			key, ok := keyToken.(string)
			if !ok {
				return nil, errors.New("JSON object key is not a string")
			}
			if _, exists := object[key]; exists {
				return nil, fmt.Errorf("duplicate JSON object key %q", key)
			}
			value, err := decodeValue(decoder)
			if err != nil {
				return nil, fmt.Errorf(
					"decode object value for %q: %w",
					key,
					err,
				)
			}
			object[key] = value
		}
		end, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("decode object terminator: %w", err)
		}
		if end != json.Delim('}') {
			return nil, errors.New("invalid JSON object terminator")
		}
		return object, nil

	case '[':
		array := make([]any, 0)
		for decoder.More() {
			value, err := decodeValue(decoder)
			if err != nil {
				return nil, fmt.Errorf("decode array value: %w", err)
			}
			array = append(array, value)
		}
		end, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("decode array terminator: %w", err)
		}
		if end != json.Delim(']') {
			return nil, errors.New("invalid JSON array terminator")
		}
		return array, nil

	default:
		return nil, fmt.Errorf("unexpected JSON delimiter %q", delimiter)
	}
}

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	switch err {
	case io.EOF:
		return nil
	case nil:
		return errors.New("JSON contains trailing values")
	default:
		return fmt.Errorf("decode trailing JSON: %w", err)
	}
}

func appendCanonical(output *bytes.Buffer, value any) error {
	switch typed := value.(type) {
	case nil:
		output.WriteString("null")

	case bool:
		if typed {
			output.WriteString("true")
		} else {
			output.WriteString("false")
		}

	case string:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return err
		}
		output.Write(encoded)

	case json.Number:
		encoded, err := canonicalNumber(string(typed))
		if err != nil {
			return err
		}
		output.WriteString(encoded)

	case []any:
		output.WriteByte('[')
		for index, item := range typed {
			if index > 0 {
				output.WriteByte(',')
			}
			if err := appendCanonical(output, item); err != nil {
				return err
			}
		}
		output.WriteByte(']')

	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		output.WriteByte('{')
		for index, key := range keys {
			if index > 0 {
				output.WriteByte(',')
			}
			encodedKey, err := json.Marshal(key)
			if err != nil {
				return err
			}
			output.Write(encodedKey)
			output.WriteByte(':')
			if err := appendCanonical(output, typed[key]); err != nil {
				return err
			}
		}
		output.WriteByte('}')

	default:
		return fmt.Errorf("unsupported JSON value type %T", value)
	}
	return nil
}

func canonicalNumber(input string) (string, error) {
	negative := false
	if strings.HasPrefix(input, "-") {
		negative = true
		input = strings.TrimPrefix(input, "-")
	}

	var exponent int64
	if index := strings.IndexAny(input, "eE"); index >= 0 {
		parsed, err := strconv.ParseInt(input[index+1:], 10, 32)
		if err != nil {
			return "", fmt.Errorf("parse JSON number exponent: %w", err)
		}
		if parsed < -100_000 || parsed > 100_000 {
			return "", errors.New("JSON exponent is outside the supported range")
		}
		exponent = parsed
		input = input[:index]
	}

	integer := input
	fraction := ""
	if before, after, found := strings.Cut(input, "."); found {
		integer = before
		fraction = after
	}
	if integer == "" || strings.HasSuffix(input, ".") {
		return "", errors.New("invalid JSON decimal representation")
	}

	digits := integer + fraction
	for _, digit := range digits {
		if digit < '0' || digit > '9' {
			return "", errors.New("JSON number contains an invalid digit")
		}
	}

	digits = strings.TrimLeft(digits, "0")
	if digits == "" {
		return "0", nil
	}

	scale := exponent - int64(len(fraction))
	for len(digits) > 1 && digits[len(digits)-1] == '0' {
		digits = digits[:len(digits)-1]
		scale++
	}

	exponent10 := scale + int64(len(digits)-1)

	var output strings.Builder
	if negative {
		output.WriteByte('-')
	}
	output.WriteByte(digits[0])
	if len(digits) > 1 {
		output.WriteByte('.')
		output.WriteString(digits[1:])
	}
	if exponent10 != 0 {
		output.WriteByte('e')
		output.WriteString(strconv.FormatInt(exponent10, 10))
	}
	return output.String(), nil
}
