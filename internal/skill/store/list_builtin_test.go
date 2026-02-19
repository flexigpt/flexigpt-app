package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/builtin"
	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/skill/spec"
)

func TestSkillStore_ListSkillBundles_BuiltInAndUser_SortPaging_AndTokenErrors(t *testing.T) {
	t.Parallel()
	s := newTestSkillStore(t)

	// Override built-ins deterministically (MapFS).
	now := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)
	biSchema := skillStoreSchema{
		SchemaVersion: spec.SkillSchemaVersion,
		Bundles: map[bundleitemutils.BundleID]spec.SkillBundle{
			"bi-1": {
				SchemaVersion: spec.SkillSchemaVersion,
				ID:            "bi-1",
				Slug:          "bi-1",
				DisplayName:   "BuiltIn 1",
				IsEnabled:     true,
				IsBuiltIn:     true,
				CreatedAt:     now,
				ModifiedAt:    now.Add(20 * time.Second),
			},
			"bi-2": {
				SchemaVersion: spec.SkillSchemaVersion,
				ID:            "bi-2",
				Slug:          "bi-2",
				DisplayName:   "BuiltIn 2 (disabled)",
				IsEnabled:     false,
				IsBuiltIn:     true,
				CreatedAt:     now,
				ModifiedAt:    now.Add(30 * time.Second),
			},
		},
		Skills: map[bundleitemutils.BundleID]map[spec.SkillSlug]spec.Skill{
			"bi-1": {},
			"bi-2": {},
		},
	}
	raw, err := json.Marshal(biSchema)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	s.builtin.skillsFS = fstest.MapFS{
		builtin.BuiltInSkillBundlesJSON: &fstest.MapFile{Data: raw},
	}
	s.builtin.skillsDir = "."
	if err := s.builtin.loadFromFS(t.Context()); err != nil {
		t.Fatalf("builtin.loadFromFS: %v", err)
	}

	// User snapshot with two bundles at same ModifiedAt => tie-break by ID asc.
	userMod := now.Add(25 * time.Second)
	user := skillStoreSchema{
		SchemaVersion: spec.SkillSchemaVersion,
		Bundles: map[bundleitemutils.BundleID]spec.SkillBundle{
			"u-1": {
				SchemaVersion: spec.SkillSchemaVersion,
				ID:            "u-1",
				Slug:          "u-1",
				DisplayName:   "User 1",
				IsEnabled:     true,
				IsBuiltIn:     false,
				CreatedAt:     userMod,
				ModifiedAt:    userMod,
			},
			"u-2": {
				SchemaVersion: spec.SkillSchemaVersion,
				ID:            "u-2",
				Slug:          "u-2",
				DisplayName:   "User 2",
				IsEnabled:     true,
				IsBuiltIn:     false,
				CreatedAt:     userMod,
				ModifiedAt:    userMod,
			},
		},
		Skills: map[bundleitemutils.BundleID]map[spec.SkillSlug]spec.Skill{
			"u-1": {},
			"u-2": {},
		},
	}
	writeAllUserLocked(t, s, user)

	resp1, err := s.ListSkillBundles(t.Context(), &spec.ListSkillBundlesRequest{
		IncludeDisabled: false,
		PageSize:        1,
	})
	if err != nil {
		t.Fatalf("ListSkillBundles(page1): %v", err)
	}
	if len(resp1.Body.SkillBundles) != 1 {
		t.Fatalf("expected 1 bundle on page1, got %d", len(resp1.Body.SkillBundles))
	}
	if resp1.Body.SkillBundles[0].ID != "u-1" {
		t.Fatalf("expected u-1 first (newest; tie by ID), got %q", resp1.Body.SkillBundles[0].ID)
	}
	if resp1.Body.NextPageToken == nil {
		t.Fatalf("expected NextPageToken")
	}

	resp2, err := s.ListSkillBundles(t.Context(), &spec.ListSkillBundlesRequest{
		PageToken: *resp1.Body.NextPageToken,
	})
	if err != nil {
		t.Fatalf("ListSkillBundles(page2): %v", err)
	}
	if len(resp2.Body.SkillBundles) != 1 {
		t.Fatalf("expected 1 bundle on page2, got %d", len(resp2.Body.SkillBundles))
	}
	if resp2.Body.SkillBundles[0].ID != "u-2" {
		t.Fatalf("expected u-2 second, got %q", resp2.Body.SkillBundles[0].ID)
	}
	if resp2.Body.NextPageToken == nil {
		t.Fatalf("expected NextPageToken on page2 (still more)")
	}

	resp3, err := s.ListSkillBundles(t.Context(), &spec.ListSkillBundlesRequest{
		PageToken: *resp2.Body.NextPageToken,
	})
	if err != nil {
		t.Fatalf("ListSkillBundles(page3): %v", err)
	}
	if len(resp3.Body.SkillBundles) != 1 {
		t.Fatalf("expected 1 bundle on page3, got %d", len(resp3.Body.SkillBundles))
	}
	if resp3.Body.SkillBundles[0].ID != "bi-1" {
		t.Fatalf("expected bi-1 third, got %q", resp3.Body.SkillBundles[0].ID)
	}
	if resp3.Body.NextPageToken != nil {
		t.Fatalf("expected no NextPageToken at end, got %q", *resp3.Body.NextPageToken)
	}

	t.Run("token-bad-cursor-time", func(t *testing.T) {
		t.Parallel()
		badTok := jsonutil.Base64JSONEncode(spec.SkillBundlePageToken{
			IncludeDisabled: true,
			PageSize:        10,
			CursorMod:       "not-a-time",
			CursorID:        "x",
		})
		_, err := s.ListSkillBundles(t.Context(), &spec.ListSkillBundlesRequest{PageToken: badTok})
		if err == nil || !errors.Is(err, spec.ErrSkillInvalidRequest) {
			t.Fatalf("expected ErrSkillInvalidRequest, got %v", err)
		}
	})
}

func TestSkillStore_ListSkills_BuiltInPaging_PhaseSwitch_AndTokenErrors(t *testing.T) {
	t.Parallel()
	s := newTestSkillStore(t)

	// Override built-ins with deterministic fixture.
	fsys := os.DirFS(filepath.Join(".", "testdata", "builtinspaging"))
	s.builtin.skillsFS = fsys
	s.builtin.skillsDir = "."
	if err := s.builtin.loadFromFS(t.Context()); err != nil {
		t.Fatalf("builtin.loadFromFS: %v", err)
	}

	// Deterministic user snapshot (no runtime involved here; listing reads JSON).
	now := time.Date(2026, 2, 10, 0, 0, 20, 0, time.UTC)
	user := skillStoreSchema{
		SchemaVersion: spec.SkillSchemaVersion,
		Bundles: map[bundleitemutils.BundleID]spec.SkillBundle{
			"ub1": {
				SchemaVersion: spec.SkillSchemaVersion,
				ID:            "ub1",
				Slug:          "user-bundle-1",
				DisplayName:   "User Bundle 1",
				IsEnabled:     true,
				IsBuiltIn:     false,
				CreatedAt:     now,
				ModifiedAt:    now,
			},
		},
		Skills: map[bundleitemutils.BundleID]map[spec.SkillSlug]spec.Skill{
			"ub1": {
				"user-old": {
					SchemaVersion: spec.SkillSchemaVersion,
					ID:            "u-old",
					Slug:          "user-old",
					Type:          spec.SkillTypeFS,
					Location:      "/tmp/user-old",
					Name:          "user-old",
					Tags:          []string{"t1"},
					Presence:      &spec.SkillPresence{Status: spec.SkillPresenceUnknown},
					IsEnabled:     true,
					IsBuiltIn:     false,
					CreatedAt:     now,
					ModifiedAt:    now.Add(1 * time.Second),
				},
				"user-new": {
					SchemaVersion: spec.SkillSchemaVersion,
					ID:            "u-new",
					Slug:          "user-new",
					Type:          spec.SkillTypeFS,
					Location:      "/tmp/user-new",
					Name:          "user-new",
					Tags:          []string{"t1"},
					Presence:      &spec.SkillPresence{Status: spec.SkillPresenceUnknown},
					IsEnabled:     true,
					IsBuiltIn:     false,
					CreatedAt:     now,
					ModifiedAt:    now.Add(2 * time.Second),
				},
			},
		},
	}
	writeAllUserLocked(t, s, user)

	// Page size 2: expect (builtins: bi-b1/skill-a, bi-b2/skill-d) because:
	// - bi-b1/skill-b-missing excluded (IncludeMissing=false)
	// - bi-b1/skill-c-disabled excluded (IncludeDisabled=false).
	resp1, err := s.ListSkills(t.Context(), &spec.ListSkillsRequest{
		RecommendedPageSize: 2,
		IncludeDisabled:     false,
		IncludeMissing:      false,
	})
	if err != nil {
		t.Fatalf("ListSkills(page1): %v", err)
	}
	if len(resp1.Body.SkillListItems) != 2 {
		t.Fatalf("expected 2 items on page1, got %d", len(resp1.Body.SkillListItems))
	}
	if resp1.Body.NextPageToken == nil {
		t.Fatalf("expected NextPageToken on page1")
	}
	if !resp1.Body.SkillListItems[0].IsBuiltIn || !resp1.Body.SkillListItems[1].IsBuiltIn {
		t.Fatalf("expected page1 to be built-in items: %+v", resp1.Body.SkillListItems)
	}

	// Page2: should contain remaining built-in (skill-e), then first user item (user-new).
	resp2, err := s.ListSkills(t.Context(), &spec.ListSkillsRequest{PageToken: *resp1.Body.NextPageToken})
	if err != nil {
		t.Fatalf("ListSkills(page2): %v", err)
	}
	if len(resp2.Body.SkillListItems) != 2 {
		t.Fatalf("expected 2 items on page2, got %d", len(resp2.Body.SkillListItems))
	}
	if !resp2.Body.SkillListItems[0].IsBuiltIn {
		t.Fatalf("expected first item of page2 to be built-in, got %+v", resp2.Body.SkillListItems[0])
	}
	if resp2.Body.SkillListItems[1].IsBuiltIn || resp2.Body.SkillListItems[1].SkillSlug != "user-new" {
		t.Fatalf("expected second item of page2 to be user-new, got %+v", resp2.Body.SkillListItems[1])
	}
	if resp2.Body.NextPageToken == nil {
		t.Fatalf("expected NextPageToken on page2 (more users)")
	}

	// Page3: last user item.
	resp3, err := s.ListSkills(t.Context(), &spec.ListSkillsRequest{PageToken: *resp2.Body.NextPageToken})
	if err != nil {
		t.Fatalf("ListSkills(page3): %v", err)
	}
	if len(resp3.Body.SkillListItems) != 1 {
		t.Fatalf("expected 1 item on page3, got %d", len(resp3.Body.SkillListItems))
	}
	if resp3.Body.SkillListItems[0].SkillSlug != "user-old" {
		t.Fatalf("expected user-old on page3, got %+v", resp3.Body.SkillListItems[0])
	}
	if resp3.Body.NextPageToken != nil {
		t.Fatalf("expected no NextPageToken on last page, got %q", *resp3.Body.NextPageToken)
	}

	// Token error cases.
	t.Run("token-invalid-phase", func(t *testing.T) {
		t.Parallel()
		bad := jsonutil.Base64JSONEncode(spec.SkillPageToken{Phase: "nope"})
		_, err := s.ListSkills(t.Context(), &spec.ListSkillsRequest{PageToken: bad})
		if err == nil || !errors.Is(err, spec.ErrSkillInvalidRequest) {
			t.Fatalf("expected ErrSkillInvalidRequest, got %v", err)
		}
	})

	t.Run("token-bad-builtin-cursor", func(t *testing.T) {
		t.Parallel()
		bad := jsonutil.Base64JSONEncode(spec.SkillPageToken{
			Phase:         spec.ListSkillPhaseBuiltIn,
			BuiltInCursor: "missing-separator",
		})
		_, err := s.ListSkills(t.Context(), &spec.ListSkillsRequest{PageToken: bad})
		if err == nil || !errors.Is(err, spec.ErrSkillInvalidRequest) {
			t.Fatalf("expected ErrSkillInvalidRequest, got %v", err)
		}
	})

	t.Run("token-bad-user-cursor", func(t *testing.T) {
		t.Parallel()
		bad := jsonutil.Base64JSONEncode(spec.SkillPageToken{
			Phase:  spec.ListSkillPhaseUser,
			DirTok: "bad",
		})
		_, err := s.ListSkills(t.Context(), &spec.ListSkillsRequest{PageToken: bad})
		if err == nil || !errors.Is(err, spec.ErrSkillInvalidRequest) {
			t.Fatalf("expected ErrSkillInvalidRequest, got %v", err)
		}
	})
}
