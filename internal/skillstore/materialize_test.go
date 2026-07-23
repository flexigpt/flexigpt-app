package skillstore

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestFSDigestSHA256_StableAndSensitive(t *testing.T) {
	t.Parallel()

	first := fstest.MapFS{
		"b.txt": &fstest.MapFile{Data: []byte("B")},
		"a.txt": &fstest.MapFile{Data: []byte("A")},
	}
	firstDigest, err := fsDigestSHA256(first)
	if err != nil {
		t.Fatalf("fsDigestSHA256(first): %v", err)
	}
	repeatedDigest, err := fsDigestSHA256(first)
	if err != nil {
		t.Fatalf("fsDigestSHA256(repeated): %v", err)
	}
	if firstDigest != repeatedDigest {
		t.Fatalf("digest is unstable: %q != %q", firstDigest, repeatedDigest)
	}

	second := fstest.MapFS{
		"b.txt": &fstest.MapFile{Data: []byte("B_CHANGED")},
		"a.txt": &fstest.MapFile{Data: []byte("A")},
	}
	secondDigest, err := fsDigestSHA256(second)
	if err != nil {
		t.Fatalf("fsDigestSHA256(second): %v", err)
	}
	if firstDigest == secondDigest {
		t.Fatal("expected a changed filesystem to have a changed digest")
	}
}

func TestCopyFSToDir_CopiesNestedFiles(t *testing.T) {
	t.Parallel()

	source := fstest.MapFS{
		"nested/note.txt": &fstest.MapFile{
			Data: []byte("hello\n"),
		},
	}
	destination := t.TempDir()

	if err := copyFSToDir(source, destination); err != nil {
		t.Fatalf("copyFSToDir: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(destination, "nested", "note.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(content) != "hello\n" {
		t.Fatalf("copied content = %q, want %q", content, "hello\n")
	}
}
