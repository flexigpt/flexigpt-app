package collection

import (
	"errors"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

func TestCollectionAndAttachmentValidationCloneAndBoundaries(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	retired := now.Add(time.Minute)
	value := Collection{
		ID:          "019d3150-6a41-7a6b-a34e-d9032342bc31",
		RootID:      "019d3150-6a42-7a6b-a34e-d9032342bc31",
		Kind:        "test.collection",
		DisplayName: "Collection",
		Description: "Description",
		Enabled:     false,
		Data:        []byte(`{"z":2,"a":1}`),
		Revision:    1,
		CreatedAt:   now,
		ModifiedAt:  now,
		RetiredAt:   &retired,
	}
	if err := value.Validate(); err != nil {
		t.Fatalf("Collection.Validate: %v", err)
	}
	cloned := value.Clone()
	expectedRetiredAt := retired
	value.Data[2] = 'x'
	*value.RetiredAt = value.RetiredAt.Add(time.Minute)
	if string(cloned.Data) != `{"z":2,"a":1}` || cloned.RetiredAt == nil || !cloned.RetiredAt.Equal(expectedRetiredAt) {
		t.Fatalf("Collection.Clone=%#v", cloned)
	}
	invalid := value
	invalid.Enabled = true
	if err := invalid.Validate(); !errors.Is(err, basespec.ErrInvalid) {
		t.Fatalf("enabled retired collection error=%v", err)
	}
	invalid = cloned
	invalid.Data = []byte(`[]`)
	if err := invalid.Validate(); !errors.Is(err, basespec.ErrInvalid) {
		t.Fatalf("array collection data error=%v", err)
	}

	attachment := Attachment{
		RootID:       value.RootID,
		CollectionID: value.ID,
		SourceID:     "019d3150-6a43-7a6b-a34e-d9032342bc31",
		Role:         "primary",
		Enabled:      true,
		Data:         []byte(`{"enabled":true}`),
		Revision:     1,
		CreatedAt:    now,
		ModifiedAt:   now,
	}
	if err := attachment.Validate(); err != nil {
		t.Fatalf("Attachment.Validate: %v", err)
	}
	attachmentClone := attachment.Clone()
	attachment.Data[2] = 'x'
	if string(attachmentClone.Data) != `{"enabled":true}` {
		t.Fatalf("Attachment.Clone=%#v", attachmentClone)
	}
	attachment.Data = []byte(`null`)
	if err := attachment.Validate(); !errors.Is(err, basespec.ErrInvalid) {
		t.Fatalf("non-object attachment data error=%v", err)
	}
}
