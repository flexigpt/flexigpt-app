package store

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/assistantpreset/spec"
	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
)

func TestAssistantPresetStore_PutAssistantPresetBundle_Invalid(t *testing.T) {
	s := newTestStore(t, nil)
	validSlug := testBundleSlug(t, "put-invalid")

	tests := []struct {
		name            string
		req             *spec.PutAssistantPresetBundleRequest
		wantErrIs       error
		wantErrContains string
	}{
		{
			name:      "nil request",
			req:       nil,
			wantErrIs: spec.ErrInvalidRequest,
		},
		{
			name: "nil body",
			req: &spec.PutAssistantPresetBundleRequest{
				BundleID: testBundleID(t, "put-invalid-1"),
			},
			wantErrIs: spec.ErrInvalidRequest,
		},
		{
			name: "empty bundle id",
			req: &spec.PutAssistantPresetBundleRequest{
				Body: &spec.PutAssistantPresetBundleRequestBody{
					Slug:        validSlug,
					DisplayName: "bundle",
					IsEnabled:   true,
				},
			},
			wantErrIs: spec.ErrInvalidRequest,
		},
		{
			name: "empty display name",
			req: &spec.PutAssistantPresetBundleRequest{
				BundleID: testBundleID(t, "put-invalid-2"),
				Body: &spec.PutAssistantPresetBundleRequestBody{
					Slug:        validSlug,
					DisplayName: "   ",
					IsEnabled:   true,
				},
			},
			wantErrIs: spec.ErrInvalidRequest,
		},
		{
			name: "empty slug",
			req: &spec.PutAssistantPresetBundleRequest{
				BundleID: testBundleID(t, "put-invalid-3"),
				Body: &spec.PutAssistantPresetBundleRequestBody{
					DisplayName: "bundle",
					IsEnabled:   true,
				},
			},
			wantErrIs: spec.ErrInvalidRequest,
		},
		{
			name: "invalid slug",
			req: &spec.PutAssistantPresetBundleRequest{
				BundleID: testBundleID(t, "put-invalid-4"),
				Body: &spec.PutAssistantPresetBundleRequestBody{
					Slug:        "Bad Slug!",
					DisplayName: "bundle",
					IsEnabled:   true,
				},
			},
			wantErrContains: "slug",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.PutAssistantPresetBundle(t.Context(), tt.req)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
				t.Fatalf("err = %v, want errors.Is(..., %v)", err, tt.wantErrIs)
			}
			if tt.wantErrContains != "" && !contains(err.Error(), tt.wantErrContains) {
				t.Fatalf("err = %q, want substring %q", err.Error(), tt.wantErrContains)
			}
		})
	}
}

func TestAssistantPresetStore_PutAssistantPresetBundle_CreateReplaceAndBuiltInReadOnly(t *testing.T) {
	t.Run("create then replace preserves createdAt", func(t *testing.T) {
		s := newTestStore(t, nil)

		bundleID := testBundleID(t, "put-replace")
		slug := testBundleSlug(t, "put-replace")

		_, err := s.PutAssistantPresetBundle(t.Context(), &spec.PutAssistantPresetBundleRequest{
			BundleID: bundleID,
			Body: &spec.PutAssistantPresetBundleRequestBody{
				Slug:        slug,
				DisplayName: "Bundle One",
				Description: "first",
				IsEnabled:   true,
			},
		})
		if err != nil {
			t.Fatalf("first PutAssistantPresetBundle() error: %v", err)
		}

		all, err := s.readAllBundles(true)
		if err != nil {
			t.Fatalf("readAllBundles() error: %v", err)
		}
		first := all.Bundles[bundleID]
		if first.DisplayName != "Bundle One" {
			t.Fatalf("DisplayName = %q, want %q", first.DisplayName, "Bundle One")
		}
		createdAt := first.CreatedAt

		_, err = s.PutAssistantPresetBundle(t.Context(), &spec.PutAssistantPresetBundleRequest{
			BundleID: bundleID,
			Body: &spec.PutAssistantPresetBundleRequestBody{
				Slug:        slug,
				DisplayName: "Bundle Two",
				Description: "second",
				IsEnabled:   false,
			},
		})
		if err != nil {
			t.Fatalf("second PutAssistantPresetBundle() error: %v", err)
		}

		all, err = s.readAllBundles(true)
		if err != nil {
			t.Fatalf("readAllBundles() error: %v", err)
		}
		second := all.Bundles[bundleID]

		if second.DisplayName != "Bundle Two" {
			t.Fatalf("DisplayName = %q, want %q", second.DisplayName, "Bundle Two")
		}
		if second.Description != "second" {
			t.Fatalf("Description = %q, want %q", second.Description, "second")
		}
		if second.IsEnabled != false {
			t.Fatalf("IsEnabled = %v, want false", second.IsEnabled)
		}
		if !second.CreatedAt.Equal(createdAt) {
			t.Fatalf("CreatedAt changed: got %v, want %v", second.CreatedAt, createdAt)
		}
	})

	t.Run("changing slug on existing bundle is rejected", func(t *testing.T) {
		s := newTestStore(t, nil)

		bundleID := testBundleID(t, "put-immutable")
		mustPutBundle(t, s, bundleID, testBundleSlug(t, "put-immutable"), true)

		_, err := s.PutAssistantPresetBundle(t.Context(), &spec.PutAssistantPresetBundleRequest{
			BundleID: bundleID,
			Body: &spec.PutAssistantPresetBundleRequestBody{
				Slug:        testBundleSlug(t, "different"),
				DisplayName: "Different",
				IsEnabled:   true,
			},
		})
		if !errors.Is(err, spec.ErrInvalidRequest) {
			t.Fatalf("err = %v, want errors.Is(..., %v)", err, spec.ErrInvalidRequest)
		}
	})

	t.Run("built-in bundle is read-only", func(t *testing.T) {
		fixture := newSingleBuiltInFixture(t, true, true)
		s := newTestStore(t, fixture.fsys)

		_, err := s.PutAssistantPresetBundle(t.Context(), &spec.PutAssistantPresetBundleRequest{
			BundleID: fixture.bundle.ID,
			Body: &spec.PutAssistantPresetBundleRequestBody{
				Slug:        fixture.bundle.Slug,
				DisplayName: "try overwrite",
				IsEnabled:   true,
			},
		})
		if !errors.Is(err, spec.ErrBuiltInReadOnly) {
			t.Fatalf("err = %v, want errors.Is(..., %v)", err, spec.ErrBuiltInReadOnly)
		}
	})
}

func TestAssistantPresetStore_PatchAssistantPresetBundle(t *testing.T) {
	t.Run("invalid requests", func(t *testing.T) {
		s := newTestStore(t, nil)

		tests := []struct {
			name string
			req  *spec.PatchAssistantPresetBundleRequest
		}{
			{name: "nil request", req: nil},
			{
				name: "nil body",
				req:  &spec.PatchAssistantPresetBundleRequest{BundleID: testBundleID(t, "patch-bundle-1")},
			},
			{
				name: "empty bundle id",
				req: &spec.PatchAssistantPresetBundleRequest{
					Body: &spec.PatchAssistantPresetBundleRequestBody{IsEnabled: true},
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := s.PatchAssistantPresetBundle(t.Context(), tt.req)
				if !errors.Is(err, spec.ErrInvalidRequest) {
					t.Fatalf("err = %v, want errors.Is(..., %v)", err, spec.ErrInvalidRequest)
				}
			})
		}
	})

	t.Run("not found", func(t *testing.T) {
		s := newTestStore(t, nil)

		_, err := s.PatchAssistantPresetBundle(t.Context(), &spec.PatchAssistantPresetBundleRequest{
			BundleID: testBundleID(t, "missing-patch"),
			Body: &spec.PatchAssistantPresetBundleRequestBody{
				IsEnabled: false,
			},
		})
		if !errors.Is(err, spec.ErrBundleNotFound) {
			t.Fatalf("err = %v, want errors.Is(..., %v)", err, spec.ErrBundleNotFound)
		}
	})

	t.Run("soft deleted bundle returns deleting", func(t *testing.T) {
		s := newTestStore(t, nil)
		bundleID := testBundleID(t, "patch-soft-deleted")
		slug := testBundleSlug(t, "patch-soft-deleted")

		mustPutBundle(t, s, bundleID, slug, true)

		_, err := s.DeleteAssistantPresetBundle(t.Context(), &spec.DeleteAssistantPresetBundleRequest{
			BundleID: bundleID,
		})
		if err != nil {
			t.Fatalf("DeleteAssistantPresetBundle() error: %v", err)
		}

		_, err = s.PatchAssistantPresetBundle(t.Context(), &spec.PatchAssistantPresetBundleRequest{
			BundleID: bundleID,
			Body: &spec.PatchAssistantPresetBundleRequestBody{
				IsEnabled: true,
			},
		})
		if !errors.Is(err, spec.ErrBundleDeleting) {
			t.Fatalf("err = %v, want errors.Is(..., %v)", err, spec.ErrBundleDeleting)
		}
	})

	t.Run("user bundle success", func(t *testing.T) {
		s := newTestStore(t, nil)
		bundleID := testBundleID(t, "patch-user")
		slug := testBundleSlug(t, "patch-user")

		mustPutBundle(t, s, bundleID, slug, true)

		_, err := s.PatchAssistantPresetBundle(t.Context(), &spec.PatchAssistantPresetBundleRequest{
			BundleID: bundleID,
			Body: &spec.PatchAssistantPresetBundleRequestBody{
				IsEnabled: false,
			},
		})
		if err != nil {
			t.Fatalf("PatchAssistantPresetBundle() error: %v", err)
		}

		all, err := s.readAllBundles(true)
		if err != nil {
			t.Fatalf("readAllBundles() error: %v", err)
		}
		if all.Bundles[bundleID].IsEnabled {
			t.Fatal("bundle should be disabled")
		}
	})

	t.Run("built-in bundle success", func(t *testing.T) {
		fixture := newSingleBuiltInFixture(t, true, true)
		s := newTestStore(t, fixture.fsys)

		_, err := s.PatchAssistantPresetBundle(t.Context(), &spec.PatchAssistantPresetBundleRequest{
			BundleID: fixture.bundle.ID,
			Body: &spec.PatchAssistantPresetBundleRequestBody{
				IsEnabled: false,
			},
		})
		if err != nil {
			t.Fatalf("PatchAssistantPresetBundle() error: %v", err)
		}

		got, err := s.builtinData.GetBuiltInBundle(t.Context(), fixture.bundle.ID)
		if err != nil {
			t.Fatalf("GetBuiltInBundle() error: %v", err)
		}
		if got.IsEnabled {
			t.Fatal("built-in bundle should be disabled")
		}
	})
}

func TestAssistantPresetStore_DeleteAssistantPresetBundle(t *testing.T) {
	t.Run("invalid requests", func(t *testing.T) {
		s := newTestStore(t, nil)

		tests := []struct {
			name string
			req  *spec.DeleteAssistantPresetBundleRequest
		}{
			{name: "nil request", req: nil},
			{name: "empty bundle id", req: &spec.DeleteAssistantPresetBundleRequest{}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := s.DeleteAssistantPresetBundle(t.Context(), tt.req)
				if !errors.Is(err, spec.ErrInvalidRequest) {
					t.Fatalf("err = %v, want errors.Is(..., %v)", err, spec.ErrInvalidRequest)
				}
			})
		}
	})

	t.Run("built-in read-only", func(t *testing.T) {
		fixture := newSingleBuiltInFixture(t, true, true)
		s := newTestStore(t, fixture.fsys)

		_, err := s.DeleteAssistantPresetBundle(t.Context(), &spec.DeleteAssistantPresetBundleRequest{
			BundleID: fixture.bundle.ID,
		})
		if !errors.Is(err, spec.ErrBuiltInReadOnly) {
			t.Fatalf("err = %v, want errors.Is(..., %v)", err, spec.ErrBuiltInReadOnly)
		}
	})

	t.Run("not found", func(t *testing.T) {
		s := newTestStore(t, nil)

		_, err := s.DeleteAssistantPresetBundle(t.Context(), &spec.DeleteAssistantPresetBundleRequest{
			BundleID: testBundleID(t, "missing-delete"),
		})
		if !errors.Is(err, spec.ErrBundleNotFound) {
			t.Fatalf("err = %v, want errors.Is(..., %v)", err, spec.ErrBundleNotFound)
		}
	})

	t.Run("not empty", func(t *testing.T) {
		s := newTestStore(t, nil)

		bundleID := testBundleID(t, "not-empty")
		slug := testBundleSlug(t, "not-empty")
		itemSlug := testItemSlug(t, "not-empty")
		version := testItemVersion(t)

		mustPutBundle(t, s, bundleID, slug, true)
		mustPutPreset(t, s, bundleID, itemSlug, version, true)

		_, err := s.DeleteAssistantPresetBundle(t.Context(), &spec.DeleteAssistantPresetBundleRequest{
			BundleID: bundleID,
		})
		if !errors.Is(err, spec.ErrBundleNotEmpty) {
			t.Fatalf("err = %v, want errors.Is(..., %v)", err, spec.ErrBundleNotEmpty)
		}
	})

	t.Run("success marks soft deleted and disabled", func(t *testing.T) {
		s := newTestStore(t, nil)

		bundleID := testBundleID(t, "delete-success")
		slug := testBundleSlug(t, "delete-success")
		mustPutBundle(t, s, bundleID, slug, true)

		_, err := s.DeleteAssistantPresetBundle(t.Context(), &spec.DeleteAssistantPresetBundleRequest{
			BundleID: bundleID,
		})
		if err != nil {
			t.Fatalf("DeleteAssistantPresetBundle() error: %v", err)
		}

		all, err := s.readAllBundles(true)
		if err != nil {
			t.Fatalf("readAllBundles() error: %v", err)
		}
		got := all.Bundles[bundleID]

		if got.IsEnabled {
			t.Fatal("bundle should be disabled after soft delete")
		}
		if got.SoftDeletedAt == nil || got.SoftDeletedAt.IsZero() {
			t.Fatal("SoftDeletedAt should be set")
		}
	})
}

func TestAssistantPresetStore_ListAssistantPresetBundles_Filtering(t *testing.T) {
	fixture := newSingleBuiltInFixture(t, true, true)
	s := newTestStore(t, fixture.fsys)

	enabled := newTestBundle(t, "enabled", true)
	disabled := newTestBundle(t, "disabled", false)
	deleted := newTestBundle(t, "deleted", true)

	enabled.ModifiedAt = fixedTestTime().Add(3 * time.Hour)
	disabled.ModifiedAt = fixedTestTime().Add(2 * time.Hour)
	deleted.ModifiedAt = fixedTestTime().Add(time.Hour)
	softDeletedAt := fixedTestTime().Add(4 * time.Hour)
	deleted.SoftDeletedAt = &softDeletedAt

	if err := s.writeAllBundles(spec.AllBundles{
		SchemaVersion: spec.SchemaVersion,
		Bundles: map[bundleitemutils.BundleID]spec.AssistantPresetBundle{
			enabled.ID:  enabled,
			disabled.ID: disabled,
			deleted.ID:  deleted,
		},
	}); err != nil {
		t.Fatalf("writeAllBundles() error: %v", err)
	}

	tests := []struct {
		name    string
		req     *spec.ListAssistantPresetBundlesRequest
		wantIDs map[bundleitemutils.BundleID]struct{}
	}{
		{
			name: "default excludes disabled and soft deleted",
			req:  &spec.ListAssistantPresetBundlesRequest{},
			wantIDs: map[bundleitemutils.BundleID]struct{}{
				fixture.bundle.ID: {},
				enabled.ID:        {},
			},
		},
		{
			name: "include disabled includes disabled but not soft deleted",
			req: &spec.ListAssistantPresetBundlesRequest{
				IncludeDisabled: true,
			},
			wantIDs: map[bundleitemutils.BundleID]struct{}{
				fixture.bundle.ID: {},
				enabled.ID:        {},
				disabled.ID:       {},
			},
		},
		{
			name: "bundle id filter",
			req: &spec.ListAssistantPresetBundlesRequest{
				IncludeDisabled: true,
				BundleIDs:       []bundleitemutils.BundleID{disabled.ID},
			},
			wantIDs: map[bundleitemutils.BundleID]struct{}{
				disabled.ID: {},
			},
		},
		{
			name: "invalid page token falls back",
			req: &spec.ListAssistantPresetBundlesRequest{
				PageToken: "%%%not-base64%%%",
			},
			wantIDs: map[bundleitemutils.BundleID]struct{}{
				fixture.bundle.ID: {},
				enabled.ID:        {},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := s.ListAssistantPresetBundles(t.Context(), tt.req)
			if err != nil {
				t.Fatalf("ListAssistantPresetBundles() error: %v", err)
			}
			if resp == nil || resp.Body == nil {
				t.Fatal("nil response body")
			}

			got := collectBundleIDs(resp.Body.AssistantPresetBundles)
			if len(got) != len(tt.wantIDs) {
				t.Fatalf("got %d bundle IDs, want %d", len(got), len(tt.wantIDs))
			}
			for id := range tt.wantIDs {
				if _, ok := got[id]; !ok {
					t.Fatalf("missing bundle ID %q", id)
				}
			}
		})
	}
}

func TestAssistantPresetStore_ListAssistantPresetBundles_PaginationAndTieBreak(t *testing.T) {
	s := newTestStore(t, nil)

	base := fixedTestTime()

	b1 := newTestBundle(t, "page-1", true)
	b2 := newTestBundle(t, "page-2", true)
	b3 := newTestBundle(t, "page-3", true)

	b1.ID = "a-id"
	b2.ID = "b-id"
	b3.ID = "c-id"

	b1.ModifiedAt = base.Add(2 * time.Hour)
	b2.ModifiedAt = base.Add(2 * time.Hour) // same timestamp as b1, tie-break by ID ascending
	b3.ModifiedAt = base.Add(time.Hour)

	if err := s.writeAllBundles(spec.AllBundles{
		SchemaVersion: spec.SchemaVersion,
		Bundles: map[bundleitemutils.BundleID]spec.AssistantPresetBundle{
			b1.ID: b1,
			b2.ID: b2,
			b3.ID: b3,
		},
	}); err != nil {
		t.Fatalf("writeAllBundles() error: %v", err)
	}

	resp1, err := s.ListAssistantPresetBundles(t.Context(), &spec.ListAssistantPresetBundlesRequest{
		PageSize:        2,
		IncludeDisabled: true,
	})
	if err != nil {
		t.Fatalf("ListAssistantPresetBundles(page1) error: %v", err)
	}
	if resp1 == nil || resp1.Body == nil {
		t.Fatal("page1 nil body")
	}
	if len(resp1.Body.AssistantPresetBundles) != 2 {
		t.Fatalf("page1 len = %d, want 2", len(resp1.Body.AssistantPresetBundles))
	}
	if resp1.Body.AssistantPresetBundles[0].ID != b1.ID || resp1.Body.AssistantPresetBundles[1].ID != b2.ID {
		t.Fatalf("page1 order = [%q, %q], want [%q, %q]",
			resp1.Body.AssistantPresetBundles[0].ID,
			resp1.Body.AssistantPresetBundles[1].ID,
			b1.ID,
			b2.ID,
		)
	}
	if resp1.Body.NextPageToken == nil || *resp1.Body.NextPageToken == "" {
		t.Fatal("expected next page token on first page")
	}

	resp2, err := s.ListAssistantPresetBundles(t.Context(), &spec.ListAssistantPresetBundlesRequest{
		PageToken: *resp1.Body.NextPageToken,
	})
	if err != nil {
		t.Fatalf("ListAssistantPresetBundles(page2) error: %v", err)
	}
	if resp2 == nil || resp2.Body == nil {
		t.Fatal("page2 nil body")
	}
	if len(resp2.Body.AssistantPresetBundles) != 1 {
		t.Fatalf("page2 len = %d, want 1", len(resp2.Body.AssistantPresetBundles))
	}
	if resp2.Body.AssistantPresetBundles[0].ID != b3.ID {
		t.Fatalf("page2 ID = %q, want %q", resp2.Body.AssistantPresetBundles[0].ID, b3.ID)
	}
	if resp2.Body.NextPageToken != nil {
		t.Fatal("expected nil next page token on last page")
	}
}

func TestAssistantPresetStore_PutAssistantPreset_Invalid(t *testing.T) {
	s := newTestStore(t, nil)

	enabledBundleID := testBundleID(t, "put-preset-enabled")
	disabledBundleID := testBundleID(t, "put-preset-disabled")
	mustPutBundle(t, s, enabledBundleID, testBundleSlug(t, "put-preset-enabled"), true)
	mustPutBundle(t, s, disabledBundleID, testBundleSlug(t, "put-preset-disabled"), false)

	version := testItemVersion(t)

	tests := []struct {
		name            string
		req             *spec.PutAssistantPresetRequest
		wantErrIs       error
		wantErrContains string
	}{
		{
			name:      "nil request",
			req:       nil,
			wantErrIs: spec.ErrInvalidRequest,
		},
		{
			name: "nil body",
			req: &spec.PutAssistantPresetRequest{
				BundleID:            enabledBundleID,
				AssistantPresetSlug: testItemSlug(t, "put-preset-a"),
				Version:             version,
			},
			wantErrIs: spec.ErrInvalidRequest,
		},
		{
			name: "missing fields",
			req: &spec.PutAssistantPresetRequest{
				Body: &spec.PutAssistantPresetRequestBody{
					DisplayName: "preset",
					IsEnabled:   true,
				},
			},
			wantErrIs: spec.ErrInvalidRequest,
		},
		{
			name: "empty display name",
			req: &spec.PutAssistantPresetRequest{
				BundleID:            enabledBundleID,
				AssistantPresetSlug: testItemSlug(t, "put-preset-b"),
				Version:             version,
				Body: &spec.PutAssistantPresetRequestBody{
					DisplayName: "   ",
					IsEnabled:   true,
				},
			},
			wantErrIs: spec.ErrInvalidRequest,
		},
		{
			name: "invalid slug",
			req: &spec.PutAssistantPresetRequest{
				BundleID:            enabledBundleID,
				AssistantPresetSlug: "Bad Slug!",
				Version:             version,
				Body: &spec.PutAssistantPresetRequestBody{
					DisplayName: "preset",
					IsEnabled:   true,
				},
			},
			wantErrContains: "slug",
		},
		{
			name: "invalid version",
			req: &spec.PutAssistantPresetRequest{
				BundleID:            enabledBundleID,
				AssistantPresetSlug: testItemSlug(t, "put-preset-c"),
				Version:             "bad version!",
				Body: &spec.PutAssistantPresetRequestBody{
					DisplayName: "preset",
					IsEnabled:   true,
				},
			},
			wantErrContains: "version",
		},
		{
			name: "bundle not found",
			req: &spec.PutAssistantPresetRequest{
				BundleID:            testBundleID(t, "missing-put-preset"),
				AssistantPresetSlug: testItemSlug(t, "put-preset-d"),
				Version:             version,
				Body: &spec.PutAssistantPresetRequestBody{
					DisplayName: "preset",
					IsEnabled:   true,
				},
			},
			wantErrIs: spec.ErrBundleNotFound,
		},
		{
			name: "bundle disabled",
			req: &spec.PutAssistantPresetRequest{
				BundleID:            disabledBundleID,
				AssistantPresetSlug: testItemSlug(t, "put-preset-e"),
				Version:             version,
				Body: &spec.PutAssistantPresetRequestBody{
					DisplayName: "preset",
					IsEnabled:   true,
				},
			},
			wantErrIs: spec.ErrBundleDisabled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.PutAssistantPreset(t.Context(), tt.req)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
				t.Fatalf("err = %v, want errors.Is(..., %v)", err, tt.wantErrIs)
			}
			if tt.wantErrContains != "" && !contains(err.Error(), tt.wantErrContains) {
				t.Fatalf("err = %q, want substring %q", err.Error(), tt.wantErrContains)
			}
		})
	}
}

func TestAssistantPresetStore_PutAssistantPreset_HappyConflictCloneAndBuiltInReadOnly(t *testing.T) {
	t.Run("built-in bundle is read-only", func(t *testing.T) {
		fixture := newSingleBuiltInFixture(t, true, true)
		s := newTestStore(t, fixture.fsys)

		_, err := s.PutAssistantPreset(t.Context(), &spec.PutAssistantPresetRequest{
			BundleID:            fixture.bundle.ID,
			AssistantPresetSlug: testItemSlug(t, "builtin-put"),
			Version:             testItemVersion(t),
			Body: &spec.PutAssistantPresetRequestBody{
				DisplayName: "preset",
				IsEnabled:   true,
			},
		})
		if !errors.Is(err, spec.ErrBuiltInReadOnly) {
			t.Fatalf("err = %v, want errors.Is(..., %v)", err, spec.ErrBuiltInReadOnly)
		}
	})

	t.Run("success, clone of pointer field, conflict on duplicate", func(t *testing.T) {
		s := newTestStore(t, nil)

		bundleID := testBundleID(t, "put-happy")
		mustPutBundle(t, s, bundleID, testBundleSlug(t, "put-happy"), true)

		slug := testItemSlug(t, "put-happy")
		version := testItemVersion(t)
		include := true

		req := &spec.PutAssistantPresetRequest{
			BundleID:            bundleID,
			AssistantPresetSlug: slug,
			Version:             version,
			Body: &spec.PutAssistantPresetRequestBody{
				DisplayName:                      "Preset Happy",
				Description:                      "desc",
				IsEnabled:                        true,
				StartingIncludeModelSystemPrompt: &include,
			},
		}

		_, err := s.PutAssistantPreset(t.Context(), req)
		if err != nil {
			t.Fatalf("PutAssistantPreset() error: %v", err)
		}

		include = false

		got := mustGetAssistantPreset(t, s, bundleID, slug, version)
		if got.StartingIncludeModelSystemPrompt == nil || *got.StartingIncludeModelSystemPrompt != true {
			t.Fatalf("stored StartingIncludeModelSystemPrompt = %v, want true", got.StartingIncludeModelSystemPrompt)
		}

		_, err = s.PutAssistantPreset(t.Context(), req)
		if !errors.Is(err, spec.ErrConflict) {
			t.Fatalf("err = %v, want errors.Is(..., %v)", err, spec.ErrConflict)
		}
	})
}

func TestAssistantPresetStore_GetPatchDeleteAssistantPreset_InvalidRequests(t *testing.T) {
	s := newTestStore(t, nil)
	version := testItemVersion(t)

	tests := []struct {
		name      string
		call      func() error
		wantErrIs error
	}{
		{
			name: "get nil request",
			call: func() error {
				_, err := s.GetAssistantPreset(t.Context(), nil)
				return err
			},
			wantErrIs: spec.ErrInvalidRequest,
		},
		{
			name: "get missing fields",
			call: func() error {
				_, err := s.GetAssistantPreset(t.Context(), &spec.GetAssistantPresetRequest{})
				return err
			},
			wantErrIs: spec.ErrInvalidRequest,
		},
		{
			name: "patch nil request",
			call: func() error {
				_, err := s.PatchAssistantPreset(t.Context(), nil)
				return err
			},
			wantErrIs: spec.ErrInvalidRequest,
		},
		{
			name: "patch nil body",
			call: func() error {
				_, err := s.PatchAssistantPreset(t.Context(), &spec.PatchAssistantPresetRequest{
					BundleID:            testBundleID(t, "patch-preset-invalid"),
					AssistantPresetSlug: testItemSlug(t, "patch-preset-invalid"),
					Version:             version,
				})
				return err
			},
			wantErrIs: spec.ErrInvalidRequest,
		},
		{
			name: "delete nil request",
			call: func() error {
				_, err := s.DeleteAssistantPreset(t.Context(), nil)
				return err
			},
			wantErrIs: spec.ErrInvalidRequest,
		},
		{
			name: "delete missing fields",
			call: func() error {
				_, err := s.DeleteAssistantPreset(t.Context(), &spec.DeleteAssistantPresetRequest{})
				return err
			},
			wantErrIs: spec.ErrInvalidRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			if !errors.Is(err, tt.wantErrIs) {
				t.Fatalf("err = %v, want errors.Is(..., %v)", err, tt.wantErrIs)
			}
		})
	}
}

func TestAssistantPresetStore_GetPatchDeleteAssistantPreset_Paths(t *testing.T) {
	t.Run("user preset lifecycle", func(t *testing.T) {
		s := newTestStore(t, nil)

		bundleID := testBundleID(t, "lifecycle")
		bundleSlug := testBundleSlug(t, "lifecycle")
		itemSlug := testItemSlug(t, "lifecycle")
		version := testItemVersion(t)

		mustPutBundle(t, s, bundleID, bundleSlug, true)
		mustPutPreset(t, s, bundleID, itemSlug, version, true)

		got := mustGetAssistantPreset(t, s, bundleID, itemSlug, version)
		if !got.IsEnabled {
			t.Fatal("expected preset enabled initially")
		}

		_, err := s.PatchAssistantPreset(t.Context(), &spec.PatchAssistantPresetRequest{
			BundleID:            bundleID,
			AssistantPresetSlug: itemSlug,
			Version:             version,
			Body: &spec.PatchAssistantPresetRequestBody{
				IsEnabled: false,
			},
		})
		if err != nil {
			t.Fatalf("PatchAssistantPreset() error: %v", err)
		}

		got = mustGetAssistantPreset(t, s, bundleID, itemSlug, version)
		if got.IsEnabled {
			t.Fatal("expected preset disabled after patch")
		}

		_, err = s.DeleteAssistantPreset(t.Context(), &spec.DeleteAssistantPresetRequest{
			BundleID:            bundleID,
			AssistantPresetSlug: itemSlug,
			Version:             version,
		})
		if err != nil {
			t.Fatalf("DeleteAssistantPreset() error: %v", err)
		}

		_, err = s.GetAssistantPreset(t.Context(), &spec.GetAssistantPresetRequest{
			BundleID:            bundleID,
			AssistantPresetSlug: itemSlug,
			Version:             version,
		})
		if !errors.Is(err, spec.ErrAssistantPresetNotFound) {
			t.Fatalf("err = %v, want errors.Is(..., %v)", err, spec.ErrAssistantPresetNotFound)
		}
	})

	t.Run("patch missing preset", func(t *testing.T) {
		s := newTestStore(t, nil)

		bundleID := testBundleID(t, "missing-preset")
		mustPutBundle(t, s, bundleID, testBundleSlug(t, "missing-preset"), true)

		_, err := s.PatchAssistantPreset(t.Context(), &spec.PatchAssistantPresetRequest{
			BundleID:            bundleID,
			AssistantPresetSlug: testItemSlug(t, "missing-preset"),
			Version:             testItemVersion(t),
			Body: &spec.PatchAssistantPresetRequestBody{
				IsEnabled: false,
			},
		})
		if !errors.Is(err, spec.ErrAssistantPresetNotFound) {
			t.Fatalf("err = %v, want errors.Is(..., %v)", err, spec.ErrAssistantPresetNotFound)
		}
	})

	t.Run("patch preset fails when bundle disabled", func(t *testing.T) {
		s := newTestStore(t, nil)

		bundleID := testBundleID(t, "disabled-bundle")
		bundleSlug := testBundleSlug(t, "disabled-bundle")
		itemSlug := testItemSlug(t, "disabled-bundle")
		version := testItemVersion(t)

		mustPutBundle(t, s, bundleID, bundleSlug, true)
		mustPutPreset(t, s, bundleID, itemSlug, version, true)

		_, err := s.PatchAssistantPresetBundle(t.Context(), &spec.PatchAssistantPresetBundleRequest{
			BundleID: bundleID,
			Body: &spec.PatchAssistantPresetBundleRequestBody{
				IsEnabled: false,
			},
		})
		if err != nil {
			t.Fatalf("PatchAssistantPresetBundle() error: %v", err)
		}

		_, err = s.PatchAssistantPreset(t.Context(), &spec.PatchAssistantPresetRequest{
			BundleID:            bundleID,
			AssistantPresetSlug: itemSlug,
			Version:             version,
			Body: &spec.PatchAssistantPresetRequestBody{
				IsEnabled: false,
			},
		})
		if !errors.Is(err, spec.ErrBundleDisabled) {
			t.Fatalf("err = %v, want errors.Is(..., %v)", err, spec.ErrBundleDisabled)
		}
	})

	t.Run("built-in get and patch work, delete is read-only", func(t *testing.T) {
		fixture := newSingleBuiltInFixture(t, true, true)
		s := newTestStore(t, fixture.fsys)

		resp, err := s.GetAssistantPreset(t.Context(), &spec.GetAssistantPresetRequest{
			BundleID:            fixture.bundle.ID,
			AssistantPresetSlug: fixture.preset.Slug,
			Version:             fixture.preset.Version,
		})
		if err != nil {
			t.Fatalf("GetAssistantPreset() error: %v", err)
		}
		if resp == nil || resp.Body == nil {
			t.Fatal("nil get response")
		}
		if !resp.Body.IsBuiltIn {
			t.Fatal("expected built-in preset")
		}

		_, err = s.PatchAssistantPreset(t.Context(), &spec.PatchAssistantPresetRequest{
			BundleID:            fixture.bundle.ID,
			AssistantPresetSlug: fixture.preset.Slug,
			Version:             fixture.preset.Version,
			Body: &spec.PatchAssistantPresetRequestBody{
				IsEnabled: false,
			},
		})
		if err != nil {
			t.Fatalf("PatchAssistantPreset() error: %v", err)
		}

		resp, err = s.GetAssistantPreset(t.Context(), &spec.GetAssistantPresetRequest{
			BundleID:            fixture.bundle.ID,
			AssistantPresetSlug: fixture.preset.Slug,
			Version:             fixture.preset.Version,
		})
		if err != nil {
			t.Fatalf("GetAssistantPreset() after patch error: %v", err)
		}
		if resp.Body.IsEnabled {
			t.Fatal("expected built-in preset disabled after patch")
		}

		_, err = s.DeleteAssistantPreset(t.Context(), &spec.DeleteAssistantPresetRequest{
			BundleID:            fixture.bundle.ID,
			AssistantPresetSlug: fixture.preset.Slug,
			Version:             fixture.preset.Version,
		})
		if !errors.Is(err, spec.ErrBuiltInReadOnly) {
			t.Fatalf("err = %v, want errors.Is(..., %v)", err, spec.ErrBuiltInReadOnly)
		}
	})
}

func TestAssistantPresetStore_ListAssistantPresets_PaginationAcrossBuiltInAndUser(t *testing.T) {
	fixture := newSingleBuiltInFixture(t, true, true)
	s := newTestStore(t, fixture.fsys)

	bundleID := testBundleID(t, "user-list-page")
	mustPutBundle(t, s, bundleID, testBundleSlug(t, "user-list-page"), true)
	mustPutPreset(t, s, bundleID, testItemSlug(t, "user-list-page"), testItemVersion(t), true)

	resp1, err := s.ListAssistantPresets(t.Context(), &spec.ListAssistantPresetsRequest{
		RecommendedPageSize: 1,
	})
	if err != nil {
		t.Fatalf("ListAssistantPresets(page1) error: %v", err)
	}
	if resp1 == nil || resp1.Body == nil {
		t.Fatal("page1 nil body")
	}
	if len(resp1.Body.AssistantPresetListItems) != 1 {
		t.Fatalf("page1 len = %d, want 1", len(resp1.Body.AssistantPresetListItems))
	}
	if !resp1.Body.AssistantPresetListItems[0].IsBuiltIn {
		t.Fatal("first page should return built-in item first")
	}
	if resp1.Body.NextPageToken == nil || *resp1.Body.NextPageToken == "" {
		t.Fatal("expected next page token after built-in-first page")
	}

	resp2, err := s.ListAssistantPresets(t.Context(), &spec.ListAssistantPresetsRequest{
		PageToken: *resp1.Body.NextPageToken,
	})
	if err != nil {
		t.Fatalf("ListAssistantPresets(page2) error: %v", err)
	}
	if resp2 == nil || resp2.Body == nil {
		t.Fatal("page2 nil body")
	}
	if len(resp2.Body.AssistantPresetListItems) != 1 {
		t.Fatalf("page2 len = %d, want 1", len(resp2.Body.AssistantPresetListItems))
	}
	if resp2.Body.AssistantPresetListItems[0].IsBuiltIn {
		t.Fatal("second page should contain user item")
	}
	if resp2.Body.NextPageToken != nil {
		t.Fatal("expected nil next page token on last page")
	}
}

func TestAssistantPresetStore_ListAssistantPresets_Filtering(t *testing.T) {
	s := newTestStore(t, nil)

	bundleA := testBundleID(t, "list-filter-a")
	bundleB := testBundleID(t, "list-filter-b")
	version := testItemVersion(t)

	mustPutBundle(t, s, bundleA, testBundleSlug(t, "list-filter-a"), true)
	mustPutPreset(t, s, bundleA, testItemSlug(t, "list-filter-a1"), version, true)
	mustPutPreset(t, s, bundleA, testItemSlug(t, "list-filter-a2"), version, false)

	mustPutBundle(t, s, bundleB, testBundleSlug(t, "list-filter-b"), true)
	mustPutPreset(t, s, bundleB, testItemSlug(t, "list-filter-b1"), version, true)

	_, err := s.PatchAssistantPresetBundle(t.Context(), &spec.PatchAssistantPresetBundleRequest{
		BundleID: bundleB,
		Body: &spec.PatchAssistantPresetBundleRequestBody{
			IsEnabled: false,
		},
	})
	if err != nil {
		t.Fatalf("PatchAssistantPresetBundle() error: %v", err)
	}

	tests := []struct {
		name     string
		req      *spec.ListAssistantPresetsRequest
		wantKeys map[string]struct{}
	}{
		{
			name: "default excludes disabled presets and disabled bundles",
			req:  &spec.ListAssistantPresetsRequest{},
			wantKeys: map[string]struct{}{
				string(bundleA) + "|" + string(testItemSlug(t, "list-filter-a1")) + "|" + string(version): {},
			},
		},
		{
			name: "include disabled includes all user items",
			req: &spec.ListAssistantPresetsRequest{
				IncludeDisabled: true,
			},
			wantKeys: map[string]struct{}{
				string(bundleA) + "|" + string(testItemSlug(t, "list-filter-a1")) + "|" + string(version): {},
				string(bundleA) + "|" + string(testItemSlug(t, "list-filter-a2")) + "|" + string(version): {},
				string(bundleB) + "|" + string(testItemSlug(t, "list-filter-b1")) + "|" + string(version): {},
			},
		},
		{
			name: "bundle filter with disabled bundle excluded",
			req: &spec.ListAssistantPresetsRequest{
				BundleIDs: []bundleitemutils.BundleID{bundleB},
			},
			wantKeys: map[string]struct{}{},
		},
		{
			name: "bundle filter with include disabled returns disabled bundle item",
			req: &spec.ListAssistantPresetsRequest{
				BundleIDs:       []bundleitemutils.BundleID{bundleB},
				IncludeDisabled: true,
			},
			wantKeys: map[string]struct{}{
				string(bundleB) + "|" + string(testItemSlug(t, "list-filter-b1")) + "|" + string(version): {},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := s.ListAssistantPresets(t.Context(), tt.req)
			if err != nil {
				t.Fatalf("ListAssistantPresets() error: %v", err)
			}
			if resp == nil || resp.Body == nil {
				t.Fatal("nil response body")
			}

			got := collectPresetKeys(resp.Body.AssistantPresetListItems)
			if len(got) != len(tt.wantKeys) {
				t.Fatalf("got %d items, want %d", len(got), len(tt.wantKeys))
			}
			for k := range tt.wantKeys {
				if _, ok := got[k]; !ok {
					t.Fatalf("missing key %q", k)
				}
			}
		})
	}
}

func TestAssistantPresetStore_FindAssistantPreset(t *testing.T) {
	s := newTestStore(t, nil)

	bundle := newTestBundle(t, "find", true)
	dirInfo := mustBuildBundleDir(t, bundle.ID, bundle.Slug)
	slug := testItemSlug(t, "find")
	version := testItemVersion(t)
	fileInfo := mustBuildItemFile(t, slug, version)

	tests := []struct {
		name      string
		setup     func(t *testing.T)
		slug      bundleitemutils.ItemSlug
		version   bundleitemutils.ItemVersion
		wantErrIs error
		wantPath  string
	}{
		{
			name:      "empty slug",
			slug:      "",
			version:   version,
			wantErrIs: spec.ErrInvalidRequest,
		},
		{
			name:      "empty version",
			slug:      slug,
			version:   "",
			wantErrIs: spec.ErrInvalidRequest,
		},
		{
			name:      "not found",
			slug:      slug,
			version:   version,
			wantErrIs: spec.ErrAssistantPresetNotFound,
		},
		{
			name: "raw slug mismatch",
			setup: func(t *testing.T) {
				t.Helper()
				if err := SetPreparedData(s, fileInfo.FileName, dirInfo.DirName, map[string]any{
					"slug":    "other-slug",
					"version": string(version),
				}); err != nil {
					t.Fatalf("SetPreparedData() error: %v", err)
				}
			},
			slug:      slug,
			version:   version,
			wantErrIs: spec.ErrAssistantPresetNotFound,
		},
		{
			name: "raw version mismatch",
			setup: func(t *testing.T) {
				t.Helper()
				if err := SetPreparedData(s, fileInfo.FileName, dirInfo.DirName, map[string]any{
					"slug":    string(slug),
					"version": "other-version",
				}); err != nil {
					t.Fatalf("SetPreparedData() error: %v", err)
				}
			},
			slug:      slug,
			version:   version,
			wantErrIs: spec.ErrAssistantPresetNotFound,
		},
		{
			name: "success",
			setup: func(t *testing.T) {
				t.Helper()
				if err := SetPreparedData(s, fileInfo.FileName, dirInfo.DirName, map[string]any{
					"slug":    string(slug),
					"version": string(version),
				}); err != nil {
					t.Fatalf("SetPreparedData() error: %v", err)
				}
			},
			slug:     slug,
			version:  version,
			wantPath: filepath.Join(dirInfo.DirName, fileInfo.FileName),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(t)
			}

			gotInfo, gotPath, err := s.findAssistantPreset(dirInfo, tt.slug, tt.version)
			if tt.wantErrIs != nil {
				if !errors.Is(err, tt.wantErrIs) {
					t.Fatalf("err = %v, want errors.Is(..., %v)", err, tt.wantErrIs)
				}
				return
			}
			if err != nil {
				t.Fatalf("findAssistantPreset() error: %v", err)
			}
			if gotInfo.FileName != fileInfo.FileName {
				t.Fatalf("FileName = %q, want %q", gotInfo.FileName, fileInfo.FileName)
			}
			if gotPath != tt.wantPath {
				t.Fatalf("path = %q, want %q", gotPath, tt.wantPath)
			}
		})
	}
}

func TestAssistantPresetStore_SweepSoftDeleted(t *testing.T) {
	tests := []struct {
		name         string
		prepareFile  bool
		softDelete   time.Time
		wantExists   bool
		bundleSuffix string
	}{
		{
			name:         "old empty bundle is hard deleted",
			softDelete:   time.Now().UTC().Add(-(softDeleteGraceAssistantPresetBundles + time.Hour)),
			bundleSuffix: "sweepoldempty",
			wantExists:   false,
		},
		{
			name:         "old non-empty bundle is retained",
			prepareFile:  true,
			softDelete:   time.Now().UTC().Add(-(softDeleteGraceAssistantPresetBundles + time.Hour)),
			bundleSuffix: "sweepoldnonempty",
			wantExists:   true,
		},
		{
			name:         "recent soft-deleted bundle is retained",
			bundleSuffix: "sweeprecent",
			softDelete:   time.Now().UTC().Add(-(softDeleteGraceAssistantPresetBundles - time.Hour)),
			wantExists:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestStore(t, nil)

			bundle := newTestBundle(t, tt.bundleSuffix, true)
			bundle.SoftDeletedAt = &tt.softDelete
			bundle.IsEnabled = false

			if err := s.writeAllBundles(spec.AllBundles{
				SchemaVersion: spec.SchemaVersion,
				Bundles: map[bundleitemutils.BundleID]spec.AssistantPresetBundle{
					bundle.ID: bundle,
				},
			}); err != nil {
				t.Fatalf("writeAllBundles() error: %v", err)
			}

			if tt.prepareFile {
				dirInfo := mustBuildBundleDir(t, bundle.ID, bundle.Slug)
				fileInfo := mustBuildItemFile(t, testItemSlug(t, "sweep"), testItemVersion(t))
				if err := SetPreparedData(s, fileInfo.FileName, dirInfo.DirName, map[string]any{
					"slug":    string(testItemSlug(t, "sweep")),
					"version": string(testItemVersion(t)),
				}); err != nil {
					t.Fatalf("SetPreparedData() error: %v", err)
				}
			}

			s.sweepSoftDeleted()

			all, err := s.readAllBundles(true)
			if err != nil {
				t.Fatalf("readAllBundles() error: %v", err)
			}
			_, ok := all.Bundles[bundle.ID]
			if ok != tt.wantExists {
				t.Fatalf("bundle exists = %v, want %v", ok, tt.wantExists)
			}
		})
	}
}

func TestStoreHelpers(t *testing.T) {
	t.Run("timePtr", func(t *testing.T) {
		tests := []struct {
			name string
			in   time.Time
			nil  bool
		}{
			{name: "zero", in: time.Time{}, nil: true},
			{name: "non-zero", in: fixedTestTime()},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := timePtr(tt.in)
				if tt.nil {
					if got != nil {
						t.Fatalf("got %v, want nil", got)
					}
					return
				}
				if got == nil {
					t.Fatal("got nil, want non-nil")
				}
				if !got.Equal(tt.in) {
					t.Fatalf("got %v, want %v", *got, tt.in)
				}
			})
		}
	})

	t.Run("isSoftDeletedAssistantPresetBundle", func(t *testing.T) {
		zero := time.Time{}
		now := fixedTestTime()

		tests := []struct {
			name   string
			bundle spec.AssistantPresetBundle
			want   bool
		}{
			{name: "nil time", bundle: spec.AssistantPresetBundle{}, want: false},
			{name: "zero time ptr", bundle: spec.AssistantPresetBundle{SoftDeletedAt: &zero}, want: false},
			{name: "non-zero time ptr", bundle: spec.AssistantPresetBundle{SoftDeletedAt: &now}, want: true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := isSoftDeletedAssistantPresetBundle(tt.bundle); got != tt.want {
					t.Fatalf("got %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("toAssistantPresetListItem", func(t *testing.T) {
		preset := newTestPreset(t, "list-item", true)

		got := toAssistantPresetListItem("bundle-a", "bundle-slug-a", preset)

		if got.BundleID != "bundle-a" {
			t.Fatalf("BundleID = %q, want %q", got.BundleID, "bundle-a")
		}
		if got.BundleSlug != "bundle-slug-a" {
			t.Fatalf("BundleSlug = %q, want %q", got.BundleSlug, "bundle-slug-a")
		}
		if got.AssistantPresetSlug != preset.Slug {
			t.Fatalf("AssistantPresetSlug = %q, want %q", got.AssistantPresetSlug, preset.Slug)
		}
		if got.AssistantPresetVersion != preset.Version {
			t.Fatalf("AssistantPresetVersion = %q, want %q", got.AssistantPresetVersion, preset.Version)
		}
		if got.DisplayName != preset.DisplayName {
			t.Fatalf("DisplayName = %q, want %q", got.DisplayName, preset.DisplayName)
		}
		if got.ModifiedAt == nil || !got.ModifiedAt.Equal(preset.ModifiedAt) {
			t.Fatalf("ModifiedAt = %v, want %v", got.ModifiedAt, preset.ModifiedAt)
		}

		preset.ModifiedAt = time.Time{}
		got = toAssistantPresetListItem("bundle-a", "bundle-slug-a", preset)
		if got.ModifiedAt != nil {
			t.Fatalf("ModifiedAt = %v, want nil for zero time", got.ModifiedAt)
		}
	})
}

func TestAssistantPresetStore_PutAssistantPreset_ConcurrentSameSlugVersion(t *testing.T) {
	s := newTestStore(t, nil)

	bundleID := testBundleID(t, "concurrent-put")
	bundleSlug := testBundleSlug(t, "concurrent-put")
	itemSlug := testItemSlug(t, "concurrent-put")
	version := testItemVersion(t)

	mustPutBundle(t, s, bundleID, bundleSlug, true)

	errCh := make(chan error, 2)
	start := make(chan struct{})

	put := func() {
		<-start
		_, err := s.PutAssistantPreset(t.Context(), &spec.PutAssistantPresetRequest{
			BundleID:            bundleID,
			AssistantPresetSlug: itemSlug,
			Version:             version,
			Body: &spec.PutAssistantPresetRequestBody{
				DisplayName: "Concurrent preset",
				IsEnabled:   true,
			},
		})
		errCh <- err
	}

	go put()
	go put()
	close(start)

	err1 := <-errCh
	err2 := <-errCh

	successes := 0
	conflicts := 0

	for _, err := range []error{err1, err2} {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, spec.ErrConflict):
			conflicts++
		default:
			t.Fatalf("unexpected concurrent put error: %v", err)
		}
	}

	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes=%d conflicts=%d, want 1 and 1", successes, conflicts)
	}
}

func contains(s, sub string) bool {
	return sub != "" || (len(s) >= len(sub) && filepath.Base(sub) == sub && stringContains(s, sub))
}

func stringContains(s, sub string) bool {
	return sub != "" || (len(s) >= len(sub) && (s == sub || s != "" && (indexOf(s, sub) >= 0)))
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
