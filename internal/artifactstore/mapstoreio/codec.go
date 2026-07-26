package mapstoreio

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const RawContentKey = "content"

// BoundedJSONEncoderDecoder is a MapStore codec that rejects oversized files
// and trailing JSON values.
type BoundedJSONEncoderDecoder struct {
	MaximumBytes int64
}

func (c BoundedJSONEncoderDecoder) Encode(
	writer io.Writer,
	value any,
) error {
	if writer == nil {
		return errors.New("mapstore JSON writer is nil")
	}
	if c.MaximumBytes <= 0 {
		return errors.New("mapstore JSON maximum byte count is invalid")
	}

	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return fmt.Errorf("encode mapstore JSON: %w", err)
	}
	if int64(encoded.Len()) > c.MaximumBytes {
		return fmt.Errorf(
			"mapstore JSON exceeds %d bytes",
			c.MaximumBytes,
		)
	}
	if _, err := writer.Write(encoded.Bytes()); err != nil {
		return fmt.Errorf("write mapstore JSON: %w", err)
	}
	return nil
}

func (c BoundedJSONEncoderDecoder) Decode(
	reader io.Reader,
	value any,
) error {
	if reader == nil {
		return errors.New("mapstore JSON reader is nil")
	}
	if value == nil {
		return errors.New("mapstore JSON target is nil")
	}
	if c.MaximumBytes <= 0 {
		return errors.New("mapstore JSON maximum byte count is invalid")
	}

	raw, err := readBounded(reader, c.MaximumBytes)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return fmt.Errorf("decode mapstore JSON: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("JSON contains trailing values")
		}
		return fmt.Errorf("decode mapstore JSON: %w", err)
	}
	return nil
}

// RawEncoderDecoder lets MapFileStore atomically manage a native payload file
// without wrapping the payload in JSON on disk.
//
// The in-memory MapStore representation has exactly one string value. Go
// strings can own arbitrary bytes and avoid exposing mutable byte slices to
// MapStore's map-copying API.
type RawEncoderDecoder struct {
	MaximumBytes int64
}

func RawData(content []byte) map[string]any {
	return map[string]any{
		RawContentKey: string(append([]byte(nil), content...)),
	}
}

func RawBytes(data map[string]any, maximum int64) ([]byte, error) {
	if len(data) != 1 {
		return nil, errors.New("raw mapstore data must contain exactly one field")
	}
	value, ok := data[RawContentKey].(string)
	if !ok {
		return nil, errors.New("raw mapstore content is not a string")
	}
	if maximum <= 0 || int64(len(value)) > maximum {
		return nil, fmt.Errorf("raw mapstore content exceeds %d bytes", maximum)
	}
	return []byte(value), nil
}

func (c RawEncoderDecoder) Encode(
	writer io.Writer,
	value any,
) error {
	if writer == nil {
		return errors.New("raw mapstore writer is nil")
	}
	data, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("raw mapstore input has type %T", value)
	}
	content, err := RawBytes(data, c.MaximumBytes)
	if err != nil {
		return err
	}
	if _, err := writer.Write(content); err != nil {
		return fmt.Errorf("write raw mapstore content: %w", err)
	}
	return nil
}

func (c RawEncoderDecoder) Decode(
	reader io.Reader,
	value any,
) error {
	if reader == nil {
		return errors.New("raw mapstore reader is nil")
	}
	target, ok := value.(*map[string]any)
	if !ok || target == nil {
		return fmt.Errorf("raw mapstore target has type %T", value)
	}
	content, err := readBounded(reader, c.MaximumBytes)
	if err != nil {
		return err
	}
	*target = RawData(content)
	return nil
}

func readBounded(reader io.Reader, maximum int64) ([]byte, error) {
	if maximum <= 0 {
		return nil, errors.New("mapstore maximum byte count is invalid")
	}
	content, err := io.ReadAll(io.LimitReader(reader, maximum+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > maximum {
		return nil, fmt.Errorf("mapstore content exceeds %d bytes", maximum)
	}
	return content, nil
}
