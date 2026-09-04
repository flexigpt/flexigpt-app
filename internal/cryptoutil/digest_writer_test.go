package cryptoutil

import "testing"

func TestDigestWriterMatchesDigestBytes(t *testing.T) {
	t.Parallel()

	writer := NewDigestWriter()
	for _, part := range [][]byte{
		[]byte("workspace"),
		[]byte("\x00"),
		[]byte("artifact"),
		[]byte("\x00"),
		[]byte("content"),
	} {
		if _, err := writer.Write(part); err != nil {
			t.Fatalf("write digest content: %v", err)
		}
	}

	want := DigestBytes([]byte("workspace\x00artifact\x00content"))
	if got := writer.Digest(); got != want {
		t.Fatalf("digest = %q, want %q", got, want)
	}
}

func TestDigestWriterDoesNotMutateAfterDigest(t *testing.T) {
	t.Parallel()

	writer := NewDigestWriter()
	if _, err := writer.Write([]byte("first")); err != nil {
		t.Fatalf("write first value: %v", err)
	}
	first := writer.Digest()

	if _, err := writer.Write([]byte("second")); err != nil {
		t.Fatalf("write second value: %v", err)
	}
	second := writer.Digest()

	if first != DigestBytes([]byte("first")) {
		t.Fatalf("first digest = %q", first)
	}
	if second != DigestBytes([]byte("firstsecond")) {
		t.Fatalf("second digest = %q", second)
	}
}
