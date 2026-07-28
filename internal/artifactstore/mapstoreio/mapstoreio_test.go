package mapstoreio

import (
	"bytes"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

type codecPayload struct {
	Name string `json:"name"`
}

func TestBoundedJSONEncoderDecoderHappyAndErrorPaths(t *testing.T) {
	t.Parallel()

	codec := BoundedJSONEncoderDecoder{MaximumBytes: 128}
	var encoded bytes.Buffer
	if err := codec.Encode(&encoded, codecPayload{Name: "value"}); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	var decoded codecPayload
	if err := codec.Decode(&encoded, &decoded); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if decoded.Name != "value" {
		t.Fatalf("decoded=%#v", decoded)
	}

	if err := codec.Decode(bytes.NewBufferString(`{"unknown":true}`), &decoded); err == nil {
		t.Fatal("Decode accepted an unknown field")
	}
	if err := codec.Decode(bytes.NewBufferString(`{"name":"value"}{}`), &decoded); err == nil {
		t.Fatal("Decode accepted trailing JSON")
	}
	if err := (BoundedJSONEncoderDecoder{MaximumBytes: 2}).Encode(
		&bytes.Buffer{},
		codecPayload{Name: "value"},
	); err == nil {
		t.Fatal("Encode accepted oversized content")
	}
	if err := (BoundedJSONEncoderDecoder{MaximumBytes: 0}).Encode(&bytes.Buffer{}, codecPayload{}); err == nil {
		t.Fatal("Encode accepted zero maximum")
	}
	if err := codec.Encode(nil, codecPayload{}); err == nil {
		t.Fatal("Encode accepted nil writer")
	}
	if err := codec.Decode(nil, &decoded); err == nil {
		t.Fatal("Decode accepted nil reader")
	}
}

func TestRawEncoderDecoderPreservesBinaryContentAndOwnership(t *testing.T) {
	t.Parallel()

	input := []byte{0, 1, 2, 0xff}
	data := RawData(input)
	input[0] = 99
	value, err := RawBytes(data, 16)
	if err != nil {
		t.Fatalf("RawBytes: %v", err)
	}
	if !bytes.Equal(value, []byte{0, 1, 2, 0xff}) {
		t.Fatalf("RawData retained caller mutation: %v", value)
	}

	codec := RawEncoderDecoder{MaximumBytes: 16}
	var encoded bytes.Buffer
	if err := codec.Encode(&encoded, data); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	var decoded map[string]any
	if err := codec.Decode(&encoded, &decoded); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	got, err := RawBytes(decoded, 16)
	if err != nil {
		t.Fatalf("RawBytes decoded: %v", err)
	}
	if !bytes.Equal(got, value) {
		t.Fatalf("decoded bytes=%v, want %v", got, value)
	}
	if _, err := RawBytes(map[string]any{"other": "value"}, 16); err == nil {
		t.Fatal("RawBytes accepted invalid key")
	}
	if err := (RawEncoderDecoder{MaximumBytes: 2}).Encode(&bytes.Buffer{}, data); err == nil {
		t.Fatal("Raw Encode accepted oversized content")
	}
}

func TestPrivatePathsRejectEscapesAndRemainConcurrentSafe(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	location, err := EnsurePrivateSubdirectory(base, filepath.Join("one", "two"))
	if err != nil {
		t.Fatalf("EnsurePrivateSubdirectory: %v", err)
	}
	info, err := os.Stat(location)
	if err != nil || !info.IsDir() {
		t.Fatalf("private directory info=%v, err=%v", info, err)
	}
	file, err := PrivateFilePath(base, filepath.Join("one", "two"), "record.json", true)
	if err != nil {
		t.Fatalf("PrivateFilePath: %v", err)
	}
	if filepath.Dir(file) != location {
		t.Fatalf("file parent=%q, want %q", filepath.Dir(file), location)
	}
	for _, relative := range []string{"..", filepath.Join("..", "escape")} {
		if _, err := EnsurePrivateSubdirectory(base, relative); err == nil {
			t.Fatalf("EnsurePrivateSubdirectory(%q) succeeded", relative)
		}
	}
	for _, name := range []string{"", ".", "..", "nested/file"} {
		if _, err := PrivateFilePath(base, "", name, false); err == nil {
			t.Fatalf("PrivateFilePath accepted name %q", name)
		}
	}

	const workers = 24
	var group sync.WaitGroup
	errorsSeen := make(chan error, workers)
	for range workers {
		group.Go(func() {
			_, err := EnsurePrivateSubdirectory(base, filepath.Join("shared", "nested"))
			if err != nil {
				errorsSeen <- err
			}
		})
	}
	group.Wait()
	close(errorsSeen)
	for err := range errorsSeen {
		t.Fatalf("concurrent directory preparation failed: %v", err)
	}

	notDirectory := filepath.Join(base, "file")
	if err := os.WriteFile(notDirectory, []byte("x"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, err := PreparePrivateDirectory(notDirectory); err == nil {
		t.Fatal("PreparePrivateDirectory accepted a regular file")
	}
}
