package engine

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source/fsdir"
)

func TestWorkspaceDataCodecsRejectUnknownAndOwnData(t *testing.T) {
	t.Parallel()

	input := CollectionData{
		DiscoveryPolicyRevision: "policy.v1",
		Discovery: DiscoveryPreferences{
			AdditionalLocators: []basespec.Locator{"docs/guide.md"},
			AdditionalRoots: []DiscoveryRoot{{
				Root:            "docs",
				Recursive:       true,
				IncludePatterns: []string{"*.md"},
			}},
			IncludeReadme: true,
		},
	}
	raw, err := encodeCollectionData(input)
	if err != nil {
		t.Fatalf("encodeCollectionData: %v", err)
	}
	if string(
		raw,
	) != `{"discovery":{"additionalLocators":["docs/guide.md"],"additionalRoots":[{"includePatterns":["*.md"],"recursive":true,"root":"docs"}],"includeReadme":true},"discoveryPolicyRevision":"policy.v1"}` {
		t.Fatalf("encoded collection data=%s", raw)
	}
	decoded, err := decodeCollectionData(raw)
	if err != nil {
		t.Fatalf("decodeCollectionData: %v", err)
	}
	decoded.Discovery.AdditionalLocators[0] = "changed.md"
	if input.Discovery.AdditionalLocators[0] != "docs/guide.md" {
		t.Fatalf("decodeCollectionData reused input storage: %#v", input)
	}
	if _, err := decodeCollectionData([]byte(`{"discoveryPolicyRevision":"policy.v1","extra":true}`)); err == nil {
		t.Fatal("unknown collection data field was accepted")
	}
	if _, err := encodeCollectionData(
		CollectionData{DiscoveryPolicyRevision: " "},
	); !errors.Is(
		err,
		basespec.ErrInvalid,
	) {
		t.Fatalf("invalid policy revision error=%v", err)
	}
	if _, err := encodeCollectionData(
		CollectionData{
			DiscoveryPolicyRevision: "policy.v1",
			Discovery: DiscoveryPreferences{
				AdditionalLocators: []basespec.Locator{"same.md", "same.md"},
			},
		},
	); !errors.Is(
		err,
		basespec.ErrInvalid,
	) {
		t.Fatalf("duplicate locator error=%v", err)
	}

	attachmentRaw, err := encodeAttachmentData(AttachmentData{})
	if err != nil || string(attachmentRaw) != `{}` {
		t.Fatalf("encodeAttachmentData=%s err=%v", attachmentRaw, err)
	}
	if _, err := decodeAttachmentData([]byte(`{"recursive":true,"unknown":true}`)); err == nil {
		t.Fatal("unknown attachment data field was accepted")
	}
	truth := true
	if err := validateAttachmentDataForRole(
		RolePrimary,
		AttachmentData{Recursive: &truth},
	); !errors.Is(
		err,
		ErrInvalidWorkspace,
	) {
		t.Fatalf("primary override error=%v, want ErrInvalidWorkspace", err)
	}
	if err := validateAttachmentDataForRole(RoleLibrary, AttachmentData{Recursive: &truth}); err != nil {
		t.Fatalf("library override error=%v", err)
	}

	artifactRaw, err := EncodeArtifactData(ArtifactData{RuntimeDisabled: true})
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
		ErrInvalidWorkspace,
	) {
		t.Fatalf("unknown artifact data error=%v", err)
	}
	if _, err := ArtifactRuntimeDisabled(
		artifact.Artifact{Data: []byte(`[]`)},
	); !strings.Contains(err.Error(), "JSON value must be an object") {
		t.Fatalf("invalid artifact runtime data error=%v", err)
	}
}

func TestDecodeDefinitionBodyAndWorkspaceStateBoundaries(t *testing.T) {
	t.Parallel()

	type body struct {
		Name string `json:"name"`
	}
	decoded, err := DecodeDefinitionBody[body](json.RawMessage(`{"name":"ok"}`))
	if err != nil || decoded.Name != "ok" {
		t.Fatalf("DecodeDefinitionBody=%#v err=%v", decoded, err)
	}
	for _, raw := range []json.RawMessage{
		[]byte(`{"name":"ok","unknown":true}`),
		[]byte(`{"name":"ok"} {}`),
		[]byte(`[]`),
	} {
		if _, err := DecodeDefinitionBody[body](raw); !errors.Is(err, ErrInvalidWorkspace) {
			t.Errorf("DecodeDefinitionBody(%s) error=%v, want ErrInvalidWorkspace", raw, err)
		}
	}

	workspace := validationTestCollection(t)
	mode, primary, err := validateWorkspaceState(
		workspace,
		CollectionData{DiscoveryPolicyRevision: "policy.v1"},
		nil,
		nil,
	)
	if err != nil || mode != ModeEmpty || primary != "" {
		t.Fatalf("empty workspace mode=%q primary=%q err=%v", mode, primary, err)
	}

	attachmentData, err := encodeAttachmentData(AttachmentData{})
	if err != nil {
		t.Fatalf("encode primary attachment: %v", err)
	}
	primarySource := validationTestSource(true, fsdir.Kind)
	primaryAttachment := collection.Attachment{
		RootID:       workspace.RootID,
		CollectionID: workspace.ID,
		SourceID:     primarySource.ID,
		Role:         RolePrimary,
		Enabled:      true,
		Data:         attachmentData,
		Revision:     1,
		CreatedAt:    workspace.CreatedAt,
		ModifiedAt:   workspace.ModifiedAt,
	}
	mode, primary, err = validateWorkspaceState(
		workspace,
		CollectionData{DiscoveryPolicyRevision: "policy.v1"},
		[]collection.Attachment{primaryAttachment},
		[]source.Summary{primarySource},
	)
	if err != nil || mode != ModeFilesystem || primary != primarySource.ID {
		t.Fatalf("filesystem workspace mode=%q primary=%q err=%v", mode, primary, err)
	}

	disabledSource := primarySource
	disabledSource.Enabled = false
	if _, _, err := validateWorkspaceState(
		workspace,
		CollectionData{DiscoveryPolicyRevision: "policy.v1"},
		[]collection.Attachment{primaryAttachment},
		[]source.Summary{disabledSource},
	); !errors.Is(
		err,
		ErrInvalidWorkspace,
	) {
		t.Fatalf("disabled primary error=%v, want ErrInvalidWorkspace", err)
	}
	second := primaryAttachment
	second.SourceID = "019d3150-6f04-7a6b-a34e-d9032342bc31"
	secondSource := validationTestSource(true, fsdir.Kind)
	secondSource.ID = second.SourceID
	if _, _, err := validateWorkspaceState(
		workspace,
		CollectionData{DiscoveryPolicyRevision: "policy.v1"},
		[]collection.Attachment{primaryAttachment, second},
		[]source.Summary{primarySource, secondSource},
	); !errors.Is(
		err,
		ErrInvalidWorkspace,
	) {
		t.Fatalf("multiple primary error=%v, want ErrInvalidWorkspace", err)
	}
}

func validationTestCollection(t *testing.T) collection.Collection {
	t.Helper()
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	raw, err := encodeCollectionData(CollectionData{DiscoveryPolicyRevision: "policy.v1"})
	if err != nil {
		t.Fatalf("encode test collection data: %v", err)
	}
	return collection.Collection{
		ID:          "019d3150-6f01-7a6b-a34e-d9032342bc31",
		RootID:      "019d3150-6f02-7a6b-a34e-d9032342bc31",
		Kind:        CollectionKind,
		DisplayName: "Workspace",
		Enabled:     true,
		Data:        raw,
		Revision:    1,
		CreatedAt:   now,
		ModifiedAt:  now,
	}
}

func validationTestSource(enabled bool, kind basespec.SourceKind) source.Summary {
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	return source.Summary{
		ID:          "019d3150-6f03-7a6b-a34e-d9032342bc31",
		RootID:      "019d3150-6f02-7a6b-a34e-d9032342bc31",
		Kind:        kind,
		DisplayName: "Source",
		Enabled:     enabled,
		Revision:    1,
		CreatedAt:   now,
		ModifiedAt:  now,
	}
}
