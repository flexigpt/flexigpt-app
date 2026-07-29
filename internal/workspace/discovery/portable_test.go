package discovery

import (
	"errors"
	"strings"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

func portableTestDefinition() CollectionDefinition {
	return CollectionDefinition{
		Kind:           "test.collection",
		SchemaID:       "test.schema",
		SchemaVersion:  "v1",
		LogicalName:    "portable-example",
		LogicalVersion: "1",
		DisplayName:    "Portable example",
		Labels:         map[string]string{"scope": "test"},
		Body:           []byte(`{"z":2,"a":1}`),
		Members: []ContentRef{
			{Locator: "z/member.txt", MediaType: "text/plain", Role: "support"},
			{URI: "https://example.com/member.json", MediaType: "application/json"},
		},
	}
}

func TestCanonicalizeCollectionDefinitionSortsMembersAndOwnsFields(t *testing.T) {
	t.Parallel()

	input := portableTestDefinition()
	first, err := CanonicalizeCollectionDefinition(input)
	if err != nil {
		t.Fatalf("CanonicalizeCollectionDefinition: %v", err)
	}
	if err := first.Validate(); err != nil {
		t.Fatalf("canonical definition validation: %v", err)
	}
	if string(first.Body) != `{"a":1,"z":2}` {
		t.Fatalf("canonical body=%s", first.Body)
	}

	reordered := portableTestDefinition()
	reordered.Members[0], reordered.Members[1] = reordered.Members[1], reordered.Members[0]
	second, err := CanonicalizeCollectionDefinition(reordered)
	if err != nil {
		t.Fatalf("canonicalize reordered: %v", err)
	}
	if first.Digest != second.Digest {
		t.Fatalf("digest=%q, want %q after member reordering", second.Digest, first.Digest)
	}

	input.Labels["scope"] = "changed"
	input.Body[2] = 'x'
	if first.Labels["scope"] != "test" || string(first.Body) != `{"a":1,"z":2}` {
		t.Fatalf("canonical result retained caller data: %#v", first)
	}

	mismatch := portableTestDefinition()
	mismatch.Digest = cryptoutil.DigestBytes([]byte("wrong"))
	if _, err := CanonicalizeCollectionDefinition(mismatch); !errors.Is(err, basespec.ErrDigestMismatch) {
		t.Fatalf("mismatched digest error=%v", err)
	}
}

func TestPortableReferencesAndRelativeResolutionRejectAmbiguity(t *testing.T) {
	t.Parallel()

	for _, reference := range []ContentRef{
		{Locator: "file.txt", URI: "https://example.com/file"},
		{URI: "file:///private/file"},
		{URI: "https://user@example.com/file"},
		{URI: "https://example.com/file#fragment"},
		{},
	} {
		if err := reference.Validate(); !errors.Is(err, basespec.ErrInvalid) {
			t.Fatalf("ContentRef(%#v) error=%v, want ErrInvalid", reference, err)
		}
	}
	if err := (ContentRef{URI: "https://example.com/file%23literal"}).Validate(); err != nil {
		t.Fatalf("percent-encoded hash URI error=%v", err)
	}

	resolved, err := resolveRelativeLocator("packages/example", "member.txt", false)
	if err != nil {
		t.Fatalf("resolveRelativeLocator: %v", err)
	}
	if resolved != "packages/example/member.txt" {
		t.Fatalf("resolved=%q", resolved)
	}
	root, err := resolveRelativeLocator("packages/example", ".", true)
	if err != nil {
		t.Fatalf("resolveRelativeLocator: %v", err)
	}
	if root != "packages/example" {
		t.Fatalf("directory root=%q", root)
	}
	if _, err := resolveRelativeLocator("packages/example", ".", false); !errors.Is(err, basespec.ErrInvalid) {
		t.Fatalf("file relative root error=%v", err)
	}

	ambiguous := portableTestDefinition()
	ambiguous.Members = []ContentRef{{Locator: "Files/Example.txt"}, {Locator: "files/example.TXT"}}
	if err := ambiguous.Validate(); !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("case-ambiguous members error=%v", err)
	}
}
