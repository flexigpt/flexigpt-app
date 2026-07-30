package skillbundle

import (
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

func TestPortableBundleJSONRoundTrip(t *testing.T) {
	t.Parallel()

	contentDigest := cryptoutil.DigestBytes([]byte("SKILL.md bytes"))
	value, err := NewPortableBundleDefinition(
		PortableBundleMetadata{
			LogicalName: "example-bundle",
			DisplayName: "Example Bundle",
		},
		[]definition.ContentRef{{
			Locator:   basespec.Locator("skills/example/SKILL.md"),
			Digest:    &contentDigest,
			MediaType: portableSkillMediaType,
			Role:      string("agent.skill"),
		}},
	)
	if err != nil {
		t.Fatalf("create portable bundle definition: %v", err)
	}

	raw, err := MarshalPortableBundleDefinition(value)
	if err != nil {
		t.Fatalf("marshal portable bundle definition: %v", err)
	}
	parsed, err := ParsePortableBundleDefinition(raw)
	if err != nil {
		t.Fatalf("parse portable bundle definition: %v", err)
	}
	if parsed.Digest != value.Digest {
		t.Fatalf("digest mismatch: got %q, want %q", parsed.Digest, value.Digest)
	}
}
