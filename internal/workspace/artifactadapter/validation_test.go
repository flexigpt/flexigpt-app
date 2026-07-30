package artifactadapter

import (
	"errors"
	"strings"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/workspace/attachmentdata"
	"github.com/flexigpt/flexigpt-app/internal/workspace/collectiondata"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

func TestWorkspaceDataCodecsRejectUnknownAndOwnData(t *testing.T) {
	t.Parallel()

	input := spec.CollectionData{
		DiscoveryPolicyRevision: "policy.v1",
		Discovery: spec.DiscoveryPreferences{
			AdditionalLocators: []basespec.Locator{"docs/guide.md"},
			AdditionalRoots: []spec.DiscoveryRoot{{
				Root:            "docs",
				Recursive:       true,
				IncludePatterns: []string{"*.md"},
			}},
			IncludeReadme: true,
		},
	}
	raw, err := collectiondata.EncodeCollectionData(input)
	if err != nil {
		t.Fatalf("encodeCollectionData: %v", err)
	}
	if string(
		raw,
	) != `{"discovery":{"additionalLocators":["docs/guide.md"],"additionalRoots":[{"includePatterns":["*.md"],"recursive":true,"root":"docs"}],"includeReadme":true},"discoveryPolicyRevision":"policy.v1"}` {
		t.Fatalf("encoded collection data=%s", raw)
	}
	decoded, err := collectiondata.DecodeCollectionData(raw)
	if err != nil {
		t.Fatalf("decodeCollectionData: %v", err)
	}
	decoded.Discovery.AdditionalLocators[0] = "changed.md"
	if input.Discovery.AdditionalLocators[0] != "docs/guide.md" {
		t.Fatalf("decodeCollectionData reused input storage: %#v", input)
	}
	if _, err := collectiondata.DecodeCollectionData(
		[]byte(`{"discoveryPolicyRevision":"policy.v1","extra":true}`),
	); err == nil {
		t.Fatal("unknown collection data field was accepted")
	}
	if _, err := collectiondata.EncodeCollectionData(
		spec.CollectionData{DiscoveryPolicyRevision: " "},
	); !errors.Is(
		err,
		basespec.ErrInvalid,
	) {
		t.Fatalf("invalid policy revision error=%v", err)
	}
	if _, err := collectiondata.EncodeCollectionData(
		spec.CollectionData{
			DiscoveryPolicyRevision: "policy.v1",
			Discovery: spec.DiscoveryPreferences{
				AdditionalLocators: []basespec.Locator{"same.md", "same.md"},
			},
		},
	); !errors.Is(
		err,
		basespec.ErrInvalid,
	) {
		t.Fatalf("duplicate locator error=%v", err)
	}

	attachmentRaw, err := attachmentdata.EncodeAttachmentData(spec.AttachmentData{})
	if err != nil || string(attachmentRaw) != `{}` {
		t.Fatalf("encodeAttachmentData=%s err=%v", attachmentRaw, err)
	}
	if _, err := attachmentdata.DecodeAttachmentData([]byte(`{"recursive":true,"unknown":true}`)); err == nil {
		t.Fatal("unknown attachment data field was accepted")
	}
	truth := true
	if err := attachmentdata.ValidateAttachmentDataForRole(
		spec.RolePrimary,
		spec.AttachmentData{Recursive: &truth},
	); !errors.Is(
		err,
		spec.ErrInvalidWorkspace,
	) {
		t.Fatalf("primary override error=%v, want spec.ErrInvalidWorkspace", err)
	}
	if err := attachmentdata.ValidateAttachmentDataForRole(
		spec.RoleLibrary,
		spec.AttachmentData{Recursive: &truth},
	); err != nil {
		t.Fatalf("library override error=%v", err)
	}

	artifactRaw, err := EncodeArtifactData(spec.ArtifactData{RuntimeDisabled: true})
	if err != nil || string(artifactRaw) != `{"runtimeDisabled":true}` {
		t.Fatalf("EncodeArtifactData=%s err=%v", artifactRaw, err)
	}
	artifactData, err := DecodeArtifactData(artifactRaw)
	if err != nil || !artifactData.RuntimeDisabled {
		t.Fatalf("DecodeArtifactData=%#v err=%v", artifactData, err)
	}
	if _, err := DecodeArtifactData(
		[]byte(`{"runtimeDisabled":false,"unknown":true}`),
	); !errors.Is(
		err,
		spec.ErrInvalidWorkspace,
	) {
		t.Fatalf("unknown artifact data error=%v", err)
	}
	if _, err := ArtifactRuntimeDisabled(
		artifact.Artifact{Data: []byte(`[]`)},
	); !strings.Contains(err.Error(), "JSON value must be an object") {
		t.Fatalf("invalid artifact runtime data error=%v", err)
	}
}
