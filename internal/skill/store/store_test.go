package store

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/skill/spec"
)

func TestCloneHelpers_DeepCopy(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)
	lc := now.Add(time.Second)
	p := &spec.SkillPresence{
		Status:        spec.SkillPresencePresent,
		LastCheckedAt: &lc,
		LastSeenAt:    &lc,
		MissingSince:  &lc,
	}
	orig := spec.Skill{
		SchemaVersion: spec.SkillSchemaVersion,
		ID:            "id",
		Slug:          "slug",
		Type:          spec.SkillTypeFS,
		Location:      "loc",
		Name:          "name",
		Tags:          []string{"a", "b"},
		Presence:      p,
		IsEnabled:     true,
		IsBuiltIn:     false,
		CreatedAt:     now,
		ModifiedAt:    now,
	}

	c := cloneSkill(orig)
	c.Tags[0] = "MUT"
	if orig.Tags[0] == "MUT" {
		t.Fatalf("tags not deep-cloned")
	}
	c.Presence.Status = spec.SkillPresenceError
	if orig.Presence.Status == spec.SkillPresenceError {
		t.Fatalf("presence not deep-cloned")
	}
	if c.Presence.LastCheckedAt == orig.Presence.LastCheckedAt {
		t.Fatalf("time pointers not cloned")
	}

	sbNow := now
	borig := spec.SkillBundle{
		SchemaVersion: spec.SkillSchemaVersion,
		ID:            "b1",
		Slug:          "bundle",
		DisplayName:   "d",
		IsEnabled:     false,
		CreatedAt:     now,
		ModifiedAt:    now,
		SoftDeletedAt: &sbNow,
	}
	bc := cloneBundle(borig)
	if bc.SoftDeletedAt == borig.SoftDeletedAt {
		t.Fatalf("bundle SoftDeletedAt pointer not cloned")
	}
}

func TestSkillCursor_RoundTripAndErrors(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)
	cur := buildSkillCursor("b1", "s1", now)

	parsed, err := parseSkillCursor(cur)
	if err != nil {
		t.Fatalf("parseSkillCursor: %v", err)
	}
	if !parsed.ModTime.Equal(now) || parsed.BundleID != "b1" || parsed.SkillSlug != "s1" {
		t.Fatalf("parsed mismatch: %+v", parsed)
	}

	bad := []string{"", "a|b", "not-a-time|b|c", "2026-01-01T00:00:00Z|b|c|d"}
	for _, s := range bad {
		if _, err := parseSkillCursor(s); err == nil {
			t.Fatalf("expected error for cursor %q", s)
		}
	}
}

func TestSkillStore_PutSkillBundle_Table(t *testing.T) {
	t.Parallel()
	s := newTestSkillStore(t)

	tests := []struct {
		name    string
		req     *spec.PutSkillBundleRequest
		wantErr error
	}{
		{"nil-req", nil, spec.ErrSkillInvalidRequest},
		{"nil-body", &spec.PutSkillBundleRequest{BundleID: "b1", Body: nil}, spec.ErrSkillInvalidRequest},
		{
			"empty-bundleid",
			&spec.PutSkillBundleRequest{
				BundleID: "",
				Body:     &spec.PutSkillBundleRequestBody{Slug: "x", DisplayName: "d", IsEnabled: true},
			},
			spec.ErrSkillInvalidRequest,
		},
		{
			"missing-slug",
			&spec.PutSkillBundleRequest{
				BundleID: "b1",
				Body:     &spec.PutSkillBundleRequestBody{Slug: "", DisplayName: "d", IsEnabled: true},
			},
			spec.ErrSkillInvalidRequest,
		},
		{
			"missing-displayName",
			&spec.PutSkillBundleRequest{
				BundleID: "b1",
				Body:     &spec.PutSkillBundleRequestBody{Slug: "x", DisplayName: "", IsEnabled: true},
			},
			spec.ErrSkillInvalidRequest,
		},
		{
			"invalid-slug",
			&spec.PutSkillBundleRequest{
				BundleID: "b1",
				Body:     &spec.PutSkillBundleRequestBody{Slug: badSlug, DisplayName: "d", IsEnabled: true},
			},
			nil, /* exact error comes from bundleitemutils */
		},
		{
			"happy",
			&spec.PutSkillBundleRequest{
				BundleID: "user-b1",
				Body: &spec.PutSkillBundleRequestBody{
					Slug:        "user-bundle",
					DisplayName: "User Bundle",
					IsEnabled:   true,
				},
			},
			nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := s.PutSkillBundle(t.Context(), tc.req)
			if tc.wantErr == nil {
				if tc.name == "invalid-slug" {
					if err == nil {
						t.Fatalf("expected error")
					}
					return
				}
				if err != nil {
					t.Fatalf("unexpected err: %v", err)
				}
				return
			}
			if err == nil || !errors.Is(err, tc.wantErr) {
				t.Fatalf("err=%v want=%v", err, tc.wantErr)
			}
		})
	}

	// Replace preserves CreatedAt.
	putBundle(t, s, "user-b2", "user-bundle-2", "Bundle 2", true)
	resp1, err := s.ListSkillBundles(t.Context(), &spec.ListSkillBundlesRequest{
		BundleIDs:       []bundleitemutils.BundleID{"user-b2"},
		IncludeDisabled: true,
	})
	if err != nil {
		t.Fatalf("ListSkillBundles: %v", err)
	}
	if len(resp1.Body.SkillBundles) != 1 {
		t.Fatalf("expected 1 bundle")
	}
	created1 := resp1.Body.SkillBundles[0].CreatedAt

	time.Sleep(2 * time.Millisecond)
	putBundle(t, s, "user-b2", "user-bundle-2", "Bundle 2 updated", true)

	resp2, err := s.ListSkillBundles(t.Context(), &spec.ListSkillBundlesRequest{
		BundleIDs:       []bundleitemutils.BundleID{"user-b2"},
		IncludeDisabled: true,
	})
	if err != nil {
		t.Fatalf("ListSkillBundles: %v", err)
	}
	created2 := resp2.Body.SkillBundles[0].CreatedAt
	if !created2.Equal(created1) {
		t.Fatalf("CreatedAt changed on replace: %v -> %v", created1, created2)
	}
}

func TestSkillStore_PatchAndDeleteSkillBundle_UserPaths(t *testing.T) {
	t.Parallel()
	s := newTestSkillStore(t)

	putBundle(t, s, "b1", "bundle-1", "Bundle 1", true)

	// Patch: disable.
	_, err := s.PatchSkillBundle(t.Context(), &spec.PatchSkillBundleRequest{
		BundleID: "b1",
		Body:     &spec.PatchSkillBundleRequestBody{IsEnabled: false},
	})
	if err != nil {
		t.Fatalf("PatchSkillBundle: %v", err)
	}

	// Delete should succeed only if empty.
	_, err = s.DeleteSkillBundle(t.Context(), &spec.DeleteSkillBundleRequest{BundleID: "b1"})
	if err != nil {
		t.Fatalf("DeleteSkillBundle: %v", err)
	}

	// Bundle is soft-deleted; patch/delete again should error with deleting.
	_, err = s.PatchSkillBundle(t.Context(), &spec.PatchSkillBundleRequest{
		BundleID: "b1",
		Body:     &spec.PatchSkillBundleRequestBody{IsEnabled: true},
	})
	if err == nil || !errors.Is(err, spec.ErrSkillBundleDeleting) {
		t.Fatalf("expected ErrSkillBundleDeleting, got %v", err)
	}
	_, err = s.DeleteSkillBundle(t.Context(), &spec.DeleteSkillBundleRequest{BundleID: "b1"})
	if err == nil || !errors.Is(err, spec.ErrSkillBundleDeleting) {
		t.Fatalf("expected ErrSkillBundleDeleting, got %v", err)
	}
}

func TestSkillStore_SweepSoftDeleted_HardDeletesAfterGrace(t *testing.T) {
	t.Parallel()
	s := newTestSkillStore(t)

	putBundle(t, s, "b1", "bundle-1", "Bundle 1", true)

	_, err := s.DeleteSkillBundle(t.Context(), &spec.DeleteSkillBundleRequest{BundleID: "b1"})
	if err != nil {
		t.Fatalf("DeleteSkillBundle: %v", err)
	}

	// Force softDeletedAt older than grace, then run sweep.
	s.writeMu.Lock()
	s.mu.Lock()
	all, err := s.readAllUser(true)
	if err != nil {
		s.mu.Unlock()
		s.writeMu.Unlock()
		t.Fatalf("readAllUser: %v", err)
	}
	b := all.Bundles["b1"]
	old := time.Now().UTC().Add(-(softDeleteGraceSkills + time.Hour))
	b.SoftDeletedAt = &old
	all.Bundles["b1"] = b
	if err := s.writeAllUser(all); err != nil {
		s.mu.Unlock()
		s.writeMu.Unlock()
		t.Fatalf("writeAllUser: %v", err)
	}
	s.mu.Unlock()
	s.writeMu.Unlock()

	s.sweepSoftDeleted()

	resp, err := s.ListSkillBundles(t.Context(), &spec.ListSkillBundlesRequest{
		BundleIDs:       []bundleitemutils.BundleID{"b1"},
		IncludeDisabled: true,
	})
	if err != nil {
		t.Fatalf("ListSkillBundles: %v", err)
	}
	if len(resp.Body.SkillBundles) != 0 {
		t.Fatalf("expected hard-deleted bundle to disappear, got %+v", resp.Body.SkillBundles)
	}
}

func TestSkillStore_GetSkill_DisabledChecks(t *testing.T) {
	t.Parallel()
	s := newTestSkillStore(t)

	putBundle(t, s, "b1", "bundle-1", "Bundle 1", true)
	skillBaseDir := t.TempDir()
	err := putSkill(t, s, "b1", "s1", skillBaseDir, "s1", "mySkill1", "My Skill 1", false)
	if err != nil {
		t.Fatalf("PutSkill: %v", err)
	}

	_, err = s.GetSkill(t.Context(), &spec.GetSkillRequest{BundleID: "b1", SkillSlug: "s1"})
	if err == nil || !errors.Is(err, spec.ErrSkillDisabled) {
		t.Fatalf("expected ErrSkillDisabled, got %v", err)
	}

	// Disable bundle and ensure ErrSkillBundleDisabled.
	_, err = s.PatchSkillBundle(t.Context(), &spec.PatchSkillBundleRequest{
		BundleID: "b1",
		Body:     &spec.PatchSkillBundleRequestBody{IsEnabled: false},
	})
	if err != nil {
		t.Fatalf("PatchSkillBundle: %v", err)
	}

	_, err = s.GetSkill(t.Context(), &spec.GetSkillRequest{BundleID: "b1", SkillSlug: "s1"})
	if err == nil || !errors.Is(err, spec.ErrSkillBundleDisabled) {
		t.Fatalf("expected ErrSkillBundleDisabled, got %v", err)
	}
}

func TestSkillStore_ListSkillBundles_FiltersAndPaging(t *testing.T) {
	t.Parallel()
	s := newTestSkillStore(t)

	putBundle(t, s, "b1", "bundle-1", "Bundle 1", true)
	time.Sleep(2 * time.Millisecond)
	putBundle(t, s, "b2", "bundle-2", "Bundle 2", false)
	time.Sleep(2 * time.Millisecond)
	putBundle(t, s, "b3", "bundle-3", "Bundle 3", true)

	// Filter to our IDs only to avoid built-in variability.
	req := &spec.ListSkillBundlesRequest{
		BundleIDs:       []bundleitemutils.BundleID{"b1", "b2", "b3"},
		IncludeDisabled: false,
		PageSize:        1,
	}
	resp1, err := s.ListSkillBundles(t.Context(), req)
	if err != nil {
		t.Fatalf("ListSkillBundles: %v", err)
	}
	if len(resp1.Body.SkillBundles) != 1 {
		t.Fatalf("expected 1 bundle on page1, got %d", len(resp1.Body.SkillBundles))
	}
	for _, b := range resp1.Body.SkillBundles {
		if b.ID != "b1" && b.ID != "b2" && b.ID != "b3" {
			t.Fatalf("unexpected bundle id %q", b.ID)
		}
		if !b.IsEnabled {
			t.Fatalf("includeDisabled=false returned disabled bundle %q", b.ID)
		}
	}

	if resp1.Body.NextPageToken == nil {
		t.Fatalf("expected NextPageToken for pageSize=1 with 2 enabled bundles")
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
	for _, b := range resp2.Body.SkillBundles {
		if b.ID != "b1" && b.ID != "b2" && b.ID != "b3" {
			t.Fatalf("unexpected bundle id %q", b.ID)
		}
		if !b.IsEnabled {
			t.Fatalf("includeDisabled=false returned disabled bundle %q", b.ID)
		}
	}
	if resp2.Body.NextPageToken != nil {
		t.Fatalf("expected no NextPageToken after final page, got %q", *resp2.Body.NextPageToken)
	}

	// Invalid token.
	_, err = s.ListSkillBundles(t.Context(), &spec.ListSkillBundlesRequest{PageToken: "!!!!"})
	if err == nil || !errors.Is(err, spec.ErrSkillInvalidRequest) {
		t.Fatalf("expected ErrSkillInvalidRequest, got %v", err)
	}
}

func TestSkillStore_ListSkills_UserOnlyFiltersAndPaging(t *testing.T) {
	t.Parallel()
	s := newTestSkillStore(t)

	putBundle(t, s, "b1", "bundle-1", "Bundle 1", true)
	skillBaseDir := t.TempDir()

	err := putSkill(t, s, "b1", "s1", skillBaseDir, "s1", "mySkill1", "My Skill 1", true)
	if err != nil {
		t.Fatalf("PutSkill: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	err = putSkill(t, s, "b1", "s2", skillBaseDir, "s2", "mySkill2", "My Skill 2", true)
	if err != nil {
		t.Fatalf("PutSkill: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	err = putSkill(t, s, "b1", "s3", skillBaseDir, "s3", "mySkill3", "My Skill 3", false)
	if err != nil {
		t.Fatalf("PutSkill: %v", err)
	}

	// Mark s2 missing (so it should be excluded when IncludeMissing=false).
	s.writeMu.Lock()
	s.mu.Lock()
	all, err := s.readAllUser(true)
	if err != nil {
		s.mu.Unlock()
		s.writeMu.Unlock()
		t.Fatalf("readAllUser: %v", err)
	}
	sk2 := all.Skills["b1"]["s2"]
	sk2.Presence = &spec.SkillPresence{Status: spec.SkillPresenceMissing}
	all.Skills["b1"]["s2"] = sk2
	if err := s.writeAllUser(all); err != nil {
		s.mu.Unlock()
		s.writeMu.Unlock()
		t.Fatalf("writeAllUser: %v", err)
	}
	s.mu.Unlock()
	s.writeMu.Unlock()

	// Add another enabled + non-missing skill so paging is guaranteed with pageSize=1.
	time.Sleep(2 * time.Millisecond)
	err = putSkill(t, s, "b1", "s4", skillBaseDir, "s4", "mySkill4", "My Skill 4", true)
	if err != nil {
		t.Fatalf("PutSkill: %v", err)
	}

	// List only user FS skills (built-ins are embeddedfs), and only for our bundle.
	resp1, err := s.ListSkills(t.Context(), &spec.ListSkillsRequest{
		BundleIDs:           []bundleitemutils.BundleID{"b1"},
		Types:               []spec.SkillType{spec.SkillTypeFS},
		IncludeDisabled:     false,
		IncludeMissing:      false,
		RecommendedPageSize: 1,
	})
	if err != nil {
		t.Fatalf("ListSkills: %v", err)
	}
	if len(resp1.Body.SkillListItems) != 1 {
		t.Fatalf("expected 1 item on page1, got %d", len(resp1.Body.SkillListItems))
	}
	it := resp1.Body.SkillListItems[0]
	if it.BundleID != "b1" || it.IsBuiltIn {
		t.Fatalf("unexpected item: %+v", it)
	}
	if it.SkillDefinition.Type != spec.SkillTypeFS {
		t.Fatalf("unexpected type: %v", it.SkillDefinition.Type)
	}
	if !it.SkillDefinition.IsEnabled {
		t.Fatalf("includeDisabled=false returned disabled skill")
	}
	if it.SkillDefinition.Presence != nil && it.SkillDefinition.Presence.Status == spec.SkillPresenceMissing {
		t.Fatalf("includeMissing=false returned missing skill")
	}

	if resp1.Body.NextPageToken == nil {
		t.Fatalf("expected next token")
	}
	resp2, err := s.ListSkills(t.Context(), &spec.ListSkillsRequest{PageToken: *resp1.Body.NextPageToken})
	if err != nil {
		t.Fatalf("ListSkills(page2): %v", err)
	}
	if len(resp2.Body.SkillListItems) != 1 {
		t.Fatalf("expected 1 item on page2, got %d", len(resp2.Body.SkillListItems))
	}
	it2 := resp2.Body.SkillListItems[0]
	if it2.BundleID != "b1" || it2.IsBuiltIn {
		t.Fatalf("unexpected page2 item: %+v", it2)
	}
	if !it2.SkillDefinition.IsEnabled {
		t.Fatalf("includeDisabled=false returned disabled skill on page2")
	}
	if it2.SkillDefinition.Presence != nil && it2.SkillDefinition.Presence.Status == spec.SkillPresenceMissing {
		t.Fatalf("includeMissing=false returned missing skill on page2")
	}
	if resp2.Body.NextPageToken != nil {
		t.Fatalf("expected no NextPageToken after final page, got %q", *resp2.Body.NextPageToken)
	}

	// Invalid token.
	_, err = s.ListSkills(t.Context(), &spec.ListSkillsRequest{PageToken: "!!!!"})
	if err == nil || !errors.Is(err, spec.ErrSkillInvalidRequest) {
		t.Fatalf("expected ErrSkillInvalidRequest, got %v", err)
	}
}

func TestSkillStore_ConcurrentPutAndList(t *testing.T) {
	t.Parallel()
	s := newTestSkillStore(t)

	putBundle(t, s, "b1", "bundle-1", "Bundle 1", true)

	ctx := t.Context()
	const n = 30

	var wg sync.WaitGroup
	errCh := make(chan error, n+100)
	skillBaseDir := t.TempDir()

	// Writer goroutines.
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			slug := "sk-" + strconv.Itoa(i)
			err := putSkill(t, s, "b1", slug, skillBaseDir, slug, "my "+slug, "My "+slug, true)
			if err != nil {
				errCh <- err
			}
		}(i)
	}

	// Concurrent lister.
	wg.Go(func() {
		for range 50 {
			_, err := s.ListSkills(ctx, &spec.ListSkillsRequest{
				BundleIDs:           []bundleitemutils.BundleID{"b1"},
				Types:               []spec.SkillType{spec.SkillTypeFS},
				IncludeDisabled:     true,
				IncludeMissing:      true,
				RecommendedPageSize: 25,
			})
			if err != nil {
				errCh <- err
				return
			}
		}
	})

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatalf("concurrency error: %v", err)
	}
}

func TestSkillStore_BuiltInReadOnly_Guards(t *testing.T) {
	t.Parallel()
	s := newTestSkillStore(t)

	if s.builtin == nil {
		t.Skip("builtin store not initialized")
	}

	ctx := t.Context()
	bundles, skills, err := s.builtin.ListBuiltInSkills(ctx)
	if err != nil {
		t.Fatalf("ListBuiltInSkills: %v", err)
	}
	if len(bundles) == 0 {
		t.Skip("no built-in bundles available")
	}

	// Pick any built-in bundle & skill.
	var bid bundleitemutils.BundleID
	for id := range bundles {
		bid = id
		break
	}
	var slug spec.SkillSlug
	if sm := skills[bid]; len(sm) > 0 {
		for sl := range sm {
			slug = sl
			break
		}
	} else {
		t.Skip("picked built-in bundle has no skills")
	}

	// PutSkillBundle on built-in bundleID should be read-only error.
	_, err = s.PutSkillBundle(ctx, &spec.PutSkillBundleRequest{
		BundleID: bid,
		Body: &spec.PutSkillBundleRequestBody{
			Slug:        "user-try",
			DisplayName: "User Try",
			IsEnabled:   true,
		},
	})
	if err == nil || !errors.Is(err, spec.ErrSkillBuiltInReadOnly) {
		t.Fatalf("expected ErrSkillBuiltInReadOnly, got %v", err)
	}

	// Delete built-in bundle should be read-only.
	_, err = s.DeleteSkillBundle(ctx, &spec.DeleteSkillBundleRequest{BundleID: bid})
	if err == nil || !errors.Is(err, spec.ErrSkillBuiltInReadOnly) {
		t.Fatalf("expected ErrSkillBuiltInReadOnly, got %v", err)
	}

	// PutSkill into built-in bundle should be read-only.
	_, err = s.PutSkill(ctx, &spec.PutSkillRequest{
		BundleID:  bid,
		SkillSlug: "user-skill",
		Body: &spec.PutSkillRequestBody{
			SkillType: spec.SkillTypeFS,
			Location:  "/tmp/x",
			Name:      "X",
			IsEnabled: true,
		},
	})
	if err == nil || !errors.Is(err, spec.ErrSkillBuiltInReadOnly) {
		t.Fatalf("expected ErrSkillBuiltInReadOnly, got %v", err)
	}

	// PatchSkill built-in cannot modify location.
	loc := "/tmp/nope"
	_, err = s.PatchSkill(ctx, &spec.PatchSkillRequest{
		BundleID:  bid,
		SkillSlug: slug,
		Body: &spec.PatchSkillRequestBody{
			Location: &loc,
		},
	})
	if err == nil || !errors.Is(err, spec.ErrSkillBuiltInReadOnly) {
		t.Fatalf("expected ErrSkillBuiltInReadOnly, got %v", err)
	}

	// DeleteSkill built-in should be read-only.
	_, err = s.DeleteSkill(ctx, &spec.DeleteSkillRequest{BundleID: bid, SkillSlug: slug})
	if err == nil || !errors.Is(err, spec.ErrSkillBuiltInReadOnly) {
		t.Fatalf("expected ErrSkillBuiltInReadOnly, got %v", err)
	}

	// PatchSkillBundle for built-in should succeed (enable/disable via overlay).
	orig, err := s.builtin.GetBuiltInSkillBundle(ctx, bid)
	if err != nil {
		t.Fatalf("GetBuiltInSkillBundle: %v", err)
	}
	_, err = s.PatchSkillBundle(ctx, &spec.PatchSkillBundleRequest{
		BundleID: bid,
		Body:     &spec.PatchSkillBundleRequestBody{IsEnabled: !orig.IsEnabled},
	})
	if err != nil {
		t.Fatalf("PatchSkillBundle(builtin): %v", err)
	}
	// Revert to avoid test pollution (though temp dir makes it isolated).
	_, _ = s.PatchSkillBundle(ctx, &spec.PatchSkillBundleRequest{
		BundleID: bid,
		Body:     &spec.PatchSkillBundleRequestBody{IsEnabled: orig.IsEnabled},
	})
}

func TestSkillStore_readAllUser_HardeningAndCorruptionDetection(t *testing.T) {
	s := newTestSkillStore(t)

	now := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)

	t.Run("missing-schemaVersion-defaults-and-normalizes-isBuiltIn", func(t *testing.T) {
		setUserStoreAllLocked(t, s, map[string]any{
			"bundles": map[string]any{
				"b1": map[string]any{
					"schemaVersion": spec.SkillSchemaVersion,
					"id":            "b1",
					"slug":          "bundle-1",
					"displayName":   "Bundle 1",
					"description":   "",
					"isEnabled":     true,
					"isBuiltIn":     true, // should be normalized to false on read
					"createdAt":     now,
					"modifiedAt":    now,
				},
			},
			"skills": map[string]any{
				"b1": map[string]any{
					"s1": map[string]any{
						"schemaVersion": spec.SkillSchemaVersion,
						"id":            "s1",
						"slug":          "s1",
						"type":          "fs",
						"location":      "/tmp/x",
						"name":          "n",
						"displayName":   "",
						"description":   "",
						"tags":          []any{},
						"presence":      map[string]any{"status": "unknown"},
						"isEnabled":     true,
						"isBuiltIn":     true, // should be normalized to false on read
						"createdAt":     now,
						"modifiedAt":    now,
					},
				},
			},
		})

		sc, err := readAllUserLocked(t, s, true)
		if err != nil {
			t.Fatalf("readAllUser: %v", err)
		}
		if sc.SchemaVersion != spec.SkillSchemaVersion {
			t.Fatalf("schemaVersion not defaulted: got=%q want=%q", sc.SchemaVersion, spec.SkillSchemaVersion)
		}
		if sc.Bundles["b1"].IsBuiltIn {
			t.Fatalf("expected bundle IsBuiltIn normalized to false")
		}
		if sc.Skills["b1"]["s1"].IsBuiltIn {
			t.Fatalf("expected skill IsBuiltIn normalized to false")
		}
	})

	t.Run("schemaVersion-mismatch-errors", func(t *testing.T) {
		setUserStoreAllLocked(t, s, map[string]any{
			"schemaVersion": "1900-01-01",
			"bundles":       map[string]any{},
			"skills":        map[string]any{},
		})
		_, err := readAllUserLocked(t, s, true)
		if err == nil || !strings.Contains(err.Error(), "schemaVersion") {
			t.Fatalf("expected schemaVersion error, got %v", err)
		}
	})

	t.Run("bundle-key-mismatch-errors", func(t *testing.T) {
		setUserStoreAllLocked(t, s, map[string]any{
			"schemaVersion": spec.SkillSchemaVersion,
			"bundles": map[string]any{
				"b1": map[string]any{
					"schemaVersion": spec.SkillSchemaVersion,
					"id":            "DIFF",
					"slug":          "bundle-1",
					"displayName":   "Bundle 1",
					"isEnabled":     true,
					"createdAt":     now,
					"modifiedAt":    now,
				},
			},
			"skills": map[string]any{"b1": map[string]any{}},
		})

		_, err := readAllUserLocked(t, s, true)
		if err == nil || !strings.Contains(err.Error(), "bundle key") {
			t.Fatalf("expected bundle key mismatch error, got %v", err)
		}
	})

	t.Run("skills-reference-missing-bundle-errors", func(t *testing.T) {
		setUserStoreAllLocked(t, s, map[string]any{
			"schemaVersion": spec.SkillSchemaVersion,
			"bundles":       map[string]any{},
			"skills": map[string]any{
				"missing": map[string]any{},
			},
		})

		_, err := readAllUserLocked(t, s, true)
		if err == nil || !strings.Contains(err.Error(), "skills reference missing bundle") {
			t.Fatalf("expected skills reference missing bundle error, got %v", err)
		}
	})

	t.Run("skill-key-mismatch-errors", func(t *testing.T) {
		setUserStoreAllLocked(t, s, map[string]any{
			"schemaVersion": spec.SkillSchemaVersion,
			"bundles": map[string]any{
				"b1": map[string]any{
					"schemaVersion": spec.SkillSchemaVersion,
					"id":            "b1",
					"slug":          "bundle-1",
					"displayName":   "Bundle 1",
					"isEnabled":     true,
					"createdAt":     now,
					"modifiedAt":    now,
				},
			},
			"skills": map[string]any{
				"b1": map[string]any{
					"s1": map[string]any{
						"schemaVersion": spec.SkillSchemaVersion,
						"id":            "s1",
						"slug":          "s2", // mismatch
						"type":          "fs",
						"location":      "/tmp/x",
						"name":          "n",
						"isEnabled":     true,
						"createdAt":     now,
						"modifiedAt":    now,
					},
				},
			},
		})

		_, err := readAllUserLocked(t, s, true)
		if err == nil || !strings.Contains(err.Error(), "skill key") {
			t.Fatalf("expected skill key mismatch error, got %v", err)
		}
	})

	t.Run("missing-bundles-and-skills-maps-normalize", func(t *testing.T) {
		setUserStoreAllLocked(t, s, map[string]any{
			"schemaVersion": spec.SkillSchemaVersion,
		})

		sc, err := readAllUserLocked(t, s, true)
		if err != nil {
			t.Fatalf("readAllUser: %v", err)
		}
		if sc.Bundles == nil || sc.Skills == nil {
			t.Fatalf("expected bundles/skills maps to be non-nil after normalization")
		}
	})
}
