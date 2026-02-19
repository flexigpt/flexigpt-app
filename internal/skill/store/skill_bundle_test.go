package store

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/skill/spec"
)

func TestSkillStore_PutSkill_Errors(t *testing.T) {
	t.Parallel()
	s := newTestSkillStore(t)

	putBundle(t, s, "ben", "bundle-enabled", "Enabled Bundle", true)
	putBundle(t, s, "bdis", "bundle-disabled", "Disabled Bundle", false)

	skillRoot := t.TempDir()
	loc := writeSkillPackage(t, skillRoot, "putskill-ok", "desc", "BODY")

	tests := []struct {
		name   string
		req    *spec.PutSkillRequest
		wantIs error
	}{
		{"nil-req", nil, spec.ErrSkillInvalidRequest},
		{"nil-body", &spec.PutSkillRequest{BundleID: "ben", SkillSlug: "s1", Body: nil}, spec.ErrSkillInvalidRequest},
		{
			"empty-bundleid",
			&spec.PutSkillRequest{BundleID: "", SkillSlug: "s1", Body: &spec.PutSkillRequestBody{}},
			spec.ErrSkillInvalidRequest,
		},
		{
			"empty-skillSlug",
			&spec.PutSkillRequest{BundleID: "ben", SkillSlug: "", Body: &spec.PutSkillRequestBody{}},
			spec.ErrSkillInvalidRequest,
		},
		{
			"invalid-skillSlug",
			&spec.PutSkillRequest{BundleID: "ben", SkillSlug: "BAD SLUG", Body: &spec.PutSkillRequestBody{}},
			spec.ErrSkillInvalidRequest,
		},
		{
			"skillType-not-fs",
			&spec.PutSkillRequest{
				BundleID:  "ben",
				SkillSlug: "s1",
				Body: &spec.PutSkillRequestBody{
					SkillType: spec.SkillTypeEmbeddedFS,
					Location:  loc,
					Name:      "putskill-ok",
					IsEnabled: true,
				},
			},
			spec.ErrSkillInvalidRequest,
		},
		{
			"bundle-not-found",
			&spec.PutSkillRequest{
				BundleID:  "nope",
				SkillSlug: "s1",
				Body: &spec.PutSkillRequestBody{
					SkillType: spec.SkillTypeFS,
					Location:  loc,
					Name:      "putskill-ok",
					IsEnabled: true,
				},
			},
			spec.ErrSkillBundleNotFound,
		},
		{
			"bundle-disabled",
			&spec.PutSkillRequest{
				BundleID:  "bdis",
				SkillSlug: "s1",
				Body: &spec.PutSkillRequestBody{
					SkillType: spec.SkillTypeFS,
					Location:  loc,
					Name:      "putskill-ok",
					IsEnabled: true,
				},
			},
			spec.ErrSkillBundleDisabled,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := s.PutSkill(t.Context(), tc.req)
			if err == nil {
				t.Fatalf("expected error")
			}
			if tc.wantIs != nil && !errors.Is(err, tc.wantIs) {
				t.Fatalf("errors.Is(err,%v)=false; err=%v", tc.wantIs, err)
			}
		})
	}
}

func TestSkillStore_PutSkill_HappyPath_EnabledAndDisabled_RuntimeConverges(t *testing.T) {
	t.Parallel()
	s := newTestSkillStore(t)
	putBundle(t, s, "b1", "bundle-1", "Bundle 1", true)

	root := t.TempDir()

	locEnabled := writeSkillPackage(t, root, "user-put-enabled", "desc enabled", "BODY_ENABLED")
	_, err := s.PutSkill(t.Context(), &spec.PutSkillRequest{
		BundleID:  "b1",
		SkillSlug: "user-put-enabled",
		Body: &spec.PutSkillRequestBody{
			SkillType: spec.SkillTypeFS,
			Location:  locEnabled,
			Name:      "user-put-enabled",
			IsEnabled: true,
			Tags:      []string{"t1"},
		},
	})
	if err != nil {
		t.Fatalf("PutSkill(enabled): %v", err)
	}

	recs := listRuntimeSkills(t, s)
	mustHaveSkillDef(t, recs, agentskillsSpec.SkillDef{Type: "fs", Name: "user-put-enabled", Location: locEnabled})

	// Disabled-at-create: store persists it, but runtime must not keep it after resync.
	locDisabled := writeSkillPackage(t, root, "user-put-disabled", "desc disabled", "BODY_DISABLED")
	_, err = s.PutSkill(t.Context(), &spec.PutSkillRequest{
		BundleID:  "b1",
		SkillSlug: "user-put-disabled",
		Body: &spec.PutSkillRequestBody{
			SkillType: spec.SkillTypeFS,
			Location:  locDisabled,
			Name:      "user-put-disabled",
			IsEnabled: false,
			Tags:      []string{"t1"},
		},
	})
	if err != nil {
		t.Fatalf("PutSkill(disabled): %v", err)
	}

	recs = listRuntimeSkills(t, s)
	mustNotHaveSkillDef(t, recs, agentskillsSpec.SkillDef{Type: "fs", Name: "user-put-disabled", Location: locDisabled})

	// Store has it (GetSkill should fail because disabled).
	_, err = s.GetSkill(t.Context(), &spec.GetSkillRequest{BundleID: "b1", SkillSlug: "user-put-disabled"})
	if err == nil || !errors.Is(err, spec.ErrSkillDisabled) {
		t.Fatalf("expected ErrSkillDisabled, got %v", err)
	}
}

func TestSkillStore_PutSkill_RuntimeRejected_DoesNotPersist(t *testing.T) {
	t.Parallel()
	s := newTestSkillStore(t)
	putBundle(t, s, "b1", "bundle-1", "Bundle 1", true)

	// Create an empty directory that does NOT contain SKILL.md. Runtime indexing should reject it.
	locBad := filepath.Join(t.TempDir(), "bad-skill")
	if err := os.MkdirAll(locBad, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	_, err := s.PutSkill(t.Context(), &spec.PutSkillRequest{
		BundleID:  "b1",
		SkillSlug: "bad-skill",
		Body: &spec.PutSkillRequestBody{
			SkillType: spec.SkillTypeFS,
			Location:  locBad,
			Name:      "bad-skill",
			IsEnabled: true,
		},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, spec.ErrSkillInvalidRequest) {
		t.Fatalf("expected ErrSkillInvalidRequest, got %v", err)
	}

	// Must not be persisted.
	resp, err := s.ListSkills(t.Context(), &spec.ListSkillsRequest{
		BundleIDs:           []bundleitemutils.BundleID{"b1"},
		Types:               []spec.SkillType{spec.SkillTypeFS},
		IncludeDisabled:     true,
		IncludeMissing:      true,
		RecommendedPageSize: 50,
	})
	if err != nil {
		t.Fatalf("ListSkills: %v", err)
	}
	if len(resp.Body.SkillListItems) != 0 {
		t.Fatalf("expected 0 persisted skills after runtime rejection, got %d", len(resp.Body.SkillListItems))
	}
}

func TestSkillStore_PutSkill_DuplicateSlug_Conflict(t *testing.T) {
	t.Parallel()
	s := newTestSkillStore(t)
	putBundle(t, s, "b1", "bundle-1", "Bundle 1", true)

	root := t.TempDir()
	loc := writeSkillPackage(t, root, "dup-skill", "desc", "BODY")

	_, err := s.PutSkill(t.Context(), &spec.PutSkillRequest{
		BundleID:  "b1",
		SkillSlug: "dup-skill",
		Body: &spec.PutSkillRequestBody{
			SkillType: spec.SkillTypeFS,
			Location:  loc,
			Name:      "dup-skill",
			IsEnabled: true,
		},
	})
	if err != nil {
		t.Fatalf("PutSkill(1): %v", err)
	}

	_, err = s.PutSkill(t.Context(), &spec.PutSkillRequest{
		BundleID:  "b1",
		SkillSlug: "dup-skill",
		Body: &spec.PutSkillRequestBody{
			SkillType: spec.SkillTypeFS,
			Location:  loc,
			Name:      "dup-skill",
			IsEnabled: true,
		},
	})
	if err == nil || !errors.Is(err, spec.ErrSkillConflict) {
		t.Fatalf("expected ErrSkillConflict, got %v", err)
	}
}

func TestSkillStore_PatchSkill_EmptyPatchRejected(t *testing.T) {
	t.Parallel()
	s := newTestSkillStore(t)

	_, err := s.PatchSkill(t.Context(), &spec.PatchSkillRequest{
		BundleID:  "b1",
		SkillSlug: "s1",
		Body:      &spec.PatchSkillRequestBody{},
	})
	if err == nil || !errors.Is(err, spec.ErrSkillInvalidRequest) {
		t.Fatalf("expected ErrSkillInvalidRequest, got %v", err)
	}
}

func TestSkillStore_PatchSkill_DisabledBundleRejected(t *testing.T) {
	t.Parallel()
	s := newTestSkillStore(t)
	putBundle(t, s, "b1", "bundle-1", "Bundle 1", true)

	root := t.TempDir()
	loc := writeSkillPackage(t, root, "patch-skill", "desc", "BODY")

	_, err := s.PutSkill(t.Context(), &spec.PutSkillRequest{
		BundleID:  "b1",
		SkillSlug: "patch-skill",
		Body: &spec.PutSkillRequestBody{
			SkillType: spec.SkillTypeFS,
			Location:  loc,
			Name:      "patch-skill",
			IsEnabled: true,
		},
	})
	if err != nil {
		t.Fatalf("PutSkill: %v", err)
	}

	_, err = s.PatchSkillBundle(t.Context(), &spec.PatchSkillBundleRequest{
		BundleID: "b1",
		Body:     &spec.PatchSkillBundleRequestBody{IsEnabled: false},
	})
	if err != nil {
		t.Fatalf("PatchSkillBundle(disable): %v", err)
	}

	_, err = s.PatchSkill(t.Context(), &spec.PatchSkillRequest{
		BundleID:  "b1",
		SkillSlug: "patch-skill",
		Body:      &spec.PatchSkillRequestBody{IsEnabled: boolPtr(false)},
	})
	if err == nil || !errors.Is(err, spec.ErrSkillBundleDisabled) {
		t.Fatalf("expected ErrSkillBundleDisabled, got %v", err)
	}
}

func TestSkillStore_PatchSkill_EnableAndLocationChange_PresenceResetAndRuntimeDelta(t *testing.T) {
	t.Parallel()
	s := newTestSkillStore(t)
	putBundle(t, s, "b1", "bundle-1", "Bundle 1", true)

	root1 := filepath.Join(t.TempDir(), "v1")
	root2 := filepath.Join(t.TempDir(), "v2")
	loc1 := writeSkillPackage(t, root1, "patch-skill", "desc v1", "BODY_V1")
	loc2 := writeSkillPackage(t, root2, "patch-skill", "desc v2", "BODY_V2")

	_, err := s.PutSkill(t.Context(), &spec.PutSkillRequest{
		BundleID:  "b1",
		SkillSlug: "patch-skill",
		Body: &spec.PutSkillRequestBody{
			SkillType: spec.SkillTypeFS,
			Location:  loc1,
			Name:      "patch-skill",
			IsEnabled: false,
		},
	})
	if err != nil {
		t.Fatalf("PutSkill(disabled): %v", err)
	}

	// Enable -> should appear in runtime.
	_, err = s.PatchSkill(t.Context(), &spec.PatchSkillRequest{
		BundleID:  "b1",
		SkillSlug: "patch-skill",
		Body:      &spec.PatchSkillRequestBody{IsEnabled: boolPtr(true)},
	})
	if err != nil {
		t.Fatalf("PatchSkill(enable): %v", err)
	}
	recs := listRuntimeSkills(t, s)
	mustHaveSkillDef(t, recs, agentskillsSpec.SkillDef{Type: "fs", Name: "patch-skill", Location: loc1})

	// Location change -> must validate new, remove old, reset presence.
	_, err = s.PatchSkill(t.Context(), &spec.PatchSkillRequest{
		BundleID:  "b1",
		SkillSlug: "patch-skill",
		Body:      &spec.PatchSkillRequestBody{Location: strPtr(loc2)},
	})
	if err != nil {
		t.Fatalf("PatchSkill(location): %v", err)
	}
	recs = listRuntimeSkills(t, s)
	mustHaveSkillDef(t, recs, agentskillsSpec.SkillDef{Type: "fs", Name: "patch-skill", Location: loc2})
	mustNotHaveSkillDef(t, recs, agentskillsSpec.SkillDef{Type: "fs", Name: "patch-skill", Location: loc1})

	gs, err := s.GetSkill(t.Context(), &spec.GetSkillRequest{BundleID: "b1", SkillSlug: "patch-skill"})
	if err != nil {
		t.Fatalf("GetSkill: %v", err)
	}
	if gs.Body.Location != loc2 {
		t.Fatalf("location not updated: got=%q want=%q", gs.Body.Location, loc2)
	}
	if gs.Body.Presence == nil || gs.Body.Presence.Status != spec.SkillPresenceUnknown {
		t.Fatalf("presence not reset to unknown on location change: %+v", gs.Body.Presence)
	}
}

func TestSkillStore_RuntimeDuplicateSafeRemoval_PatchAndDelete(t *testing.T) {
	t.Parallel()
	s := newTestSkillStore(t)
	putBundle(t, s, "b1", "bundle-1", "Bundle 1", true)

	// Two store skills pointing to the same underlying runtime def.
	root1 := filepath.Join(t.TempDir(), "v1")
	root2 := filepath.Join(t.TempDir(), "v2")
	loc1 := writeSkillPackage(t, root1, "dupe-skill", "desc v1", "BODY_V1")
	loc2 := writeSkillPackage(t, root2, "dupe-skill", "desc v2", "BODY_V2")

	_, err := s.PutSkill(t.Context(), &spec.PutSkillRequest{
		BundleID:  "b1",
		SkillSlug: "dupe-1",
		Body: &spec.PutSkillRequestBody{
			SkillType: spec.SkillTypeFS,
			Location:  loc1,
			Name:      "dupe-skill",
			IsEnabled: true,
		},
	})
	if err != nil {
		t.Fatalf("PutSkill(dupe-1): %v", err)
	}
	_, err = s.PutSkill(t.Context(), &spec.PutSkillRequest{
		BundleID:  "b1",
		SkillSlug: "dupe-2",
		Body: &spec.PutSkillRequestBody{
			SkillType: spec.SkillTypeFS,
			Location:  loc1,
			Name:      "dupe-skill",
			IsEnabled: true,
		},
	})
	if err != nil {
		t.Fatalf("PutSkill(dupe-2): %v", err)
	}

	recs := listRuntimeSkills(t, s)
	mustHaveSkillDef(t, recs, agentskillsSpec.SkillDef{Type: "fs", Name: "dupe-skill", Location: loc1})

	// Patch only one location: old def must stay because dupe-2 still wants it.
	_, err = s.PatchSkill(t.Context(), &spec.PatchSkillRequest{
		BundleID:  "b1",
		SkillSlug: "dupe-1",
		Body:      &spec.PatchSkillRequestBody{Location: strPtr(loc2)},
	})
	if err != nil {
		t.Fatalf("PatchSkill(dupe-1 location): %v", err)
	}

	recs = listRuntimeSkills(t, s)
	mustHaveSkillDef(t, recs, agentskillsSpec.SkillDef{Type: "fs", Name: "dupe-skill", Location: loc1})
	mustHaveSkillDef(t, recs, agentskillsSpec.SkillDef{Type: "fs", Name: "dupe-skill", Location: loc2})

	// Delete the remaining reference to loc1 -> runtime must remove loc1 now.
	_, err = s.DeleteSkill(t.Context(), &spec.DeleteSkillRequest{BundleID: "b1", SkillSlug: "dupe-2"})
	if err != nil {
		t.Fatalf("DeleteSkill(dupe-2): %v", err)
	}
	recs = listRuntimeSkills(t, s)
	mustNotHaveSkillDef(t, recs, agentskillsSpec.SkillDef{Type: "fs", Name: "dupe-skill", Location: loc1})
	mustHaveSkillDef(t, recs, agentskillsSpec.SkillDef{Type: "fs", Name: "dupe-skill", Location: loc2})
}

func TestSkillStore_DeleteSkill_MissingPresenceBlocked(t *testing.T) {
	t.Parallel()
	s := newTestSkillStore(t)
	putBundle(t, s, "b1", "bundle-1", "Bundle 1", true)

	root := t.TempDir()
	loc := writeSkillPackage(t, root, "missing-skill", "desc", "BODY")

	_, err := s.PutSkill(t.Context(), &spec.PutSkillRequest{
		BundleID:  "b1",
		SkillSlug: "missing-skill",
		Body: &spec.PutSkillRequestBody{
			SkillType: spec.SkillTypeFS,
			Location:  loc,
			Name:      "missing-skill",
			IsEnabled: true,
		},
	})
	if err != nil {
		t.Fatalf("PutSkill: %v", err)
	}

	// Force presence=missing in the persisted store.
	all, err := readAllUserLocked(t, s, true)
	if err != nil {
		t.Fatalf("readAllUser: %v", err)
	}
	sk := all.Skills["b1"]["missing-skill"]
	sk.Presence = &spec.SkillPresence{Status: spec.SkillPresenceMissing}
	all.Skills["b1"]["missing-skill"] = sk
	writeAllUserLocked(t, s, all)

	_, err = s.DeleteSkill(t.Context(), &spec.DeleteSkillRequest{BundleID: "b1", SkillSlug: "missing-skill"})
	if err == nil || !errors.Is(err, spec.ErrSkillIsMissing) {
		t.Fatalf("expected ErrSkillIsMissing, got %v", err)
	}
}

func TestSkillStore_DeleteSkillBundle_NotEmpty(t *testing.T) {
	t.Parallel()
	s := newTestSkillStore(t)
	putBundle(t, s, "b1", "bundle-1", "Bundle 1", true)

	root := t.TempDir()
	loc := writeSkillPackage(t, root, "some-skill", "desc", "BODY")
	_, err := s.PutSkill(t.Context(), &spec.PutSkillRequest{
		BundleID:  "b1",
		SkillSlug: "some-skill",
		Body: &spec.PutSkillRequestBody{
			SkillType: spec.SkillTypeFS,
			Location:  loc,
			Name:      "some-skill",
			IsEnabled: true,
		},
	})
	if err != nil {
		t.Fatalf("PutSkill: %v", err)
	}

	_, err = s.DeleteSkillBundle(t.Context(), &spec.DeleteSkillBundleRequest{BundleID: "b1"})
	if err == nil || !errors.Is(err, spec.ErrSkillBundleNotEmpty) {
		t.Fatalf("expected ErrSkillBundleNotEmpty, got %v", err)
	}
}

func TestSkillStore_PutSkillBundle_SoftDeletedCannotBeRecreated(t *testing.T) {
	t.Parallel()
	s := newTestSkillStore(t)

	putBundle(t, s, "b1", "bundle-1", "Bundle 1", true)

	_, err := s.DeleteSkillBundle(t.Context(), &spec.DeleteSkillBundleRequest{BundleID: "b1"})
	if err != nil {
		t.Fatalf("DeleteSkillBundle: %v", err)
	}

	_, err = s.PutSkillBundle(t.Context(), &spec.PutSkillBundleRequest{
		BundleID: "b1",
		Body: &spec.PutSkillBundleRequestBody{
			Slug:        "bundle-1",
			DisplayName: "Bundle 1 (recreate)",
			IsEnabled:   true,
		},
	})
	if err == nil || !errors.Is(err, spec.ErrSkillBundleDeleting) {
		t.Fatalf("expected ErrSkillBundleDeleting, got %v", err)
	}
}

func TestSkillStore_withUserWriteSaga_InvalidArgs(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	var nilStore *SkillStore
	err := nilStore.withUserWriteSaga(ctx, "op", func(sc *skillStoreSchema) (userWriteSagaOutcome, error) {
		return userWriteSagaOutcome{}, nil
	})
	if err == nil || !errors.Is(err, spec.ErrSkillInvalidRequest) {
		t.Fatalf("expected ErrSkillInvalidRequest, got %v", err)
	}

	s := newTestSkillStore(t)
	err = s.withUserWriteSaga(ctx, "op", nil)
	if err == nil || !errors.Is(err, spec.ErrSkillInvalidRequest) {
		t.Fatalf("expected ErrSkillInvalidRequest, got %v", err)
	}
}

func TestSkillStore_enabledDefCountsInUserBundle_NilAndEmpty(t *testing.T) {
	t.Parallel()
	s := newTestSkillStore(t)

	got, err := s.enabledDefCountsInUserBundle(nil, "b1")
	if err != nil {
		t.Fatalf("enabledDefCountsInUserBundle: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty map, got %v", got)
	}

	sc := &skillStoreSchema{
		SchemaVersion: spec.SkillSchemaVersion,
		Bundles:       map[bundleitemutils.BundleID]spec.SkillBundle{},
		Skills:        map[bundleitemutils.BundleID]map[spec.SkillSlug]spec.Skill{},
	}
	got, err = s.enabledDefCountsInUserBundle(sc, "b1")
	if err != nil {
		t.Fatalf("enabledDefCountsInUserBundle: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty map, got %v", got)
	}
}

func TestSkillStore_enabledDefCountsInUserBundle_InvalidSkillDefFails(t *testing.T) {
	t.Parallel()
	s := newTestSkillStore(t)

	now := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)
	sc := &skillStoreSchema{
		SchemaVersion: spec.SkillSchemaVersion,
		Bundles: map[bundleitemutils.BundleID]spec.SkillBundle{
			"b1": {
				SchemaVersion: spec.SkillSchemaVersion,
				ID:            "b1",
				Slug:          "bundle-1",
				DisplayName:   "Bundle 1",
				IsEnabled:     true,
				CreatedAt:     now,
				ModifiedAt:    now,
			},
		},
		Skills: map[bundleitemutils.BundleID]map[spec.SkillSlug]spec.Skill{
			"b1": {
				"s1": {
					SchemaVersion: spec.SkillSchemaVersion,
					ID:            "id-1",
					Slug:          "s1",
					Type:          spec.SkillTypeFS,
					Location:      "", // invalid
					Name:          "n",
					IsEnabled:     true,
					IsBuiltIn:     false,
					CreatedAt:     now,
					ModifiedAt:    now,
				},
			},
		},
	}

	_, err := s.enabledDefCountsInUserBundle(sc, "b1")
	if err == nil || !errors.Is(err, spec.ErrSkillInvalidRequest) {
		t.Fatalf("expected ErrSkillInvalidRequest, got %v", err)
	}
}
