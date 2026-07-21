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

const (
	skillBundleBen              = "ben"
	skillBundleBdis             = "bdis"
	skillBundleB1               = "b1"
	skillBundleS1               = "s1"
	skillBundleUserPutEnabled   = "user-put-enabled"
	skillBundleUserPutDisabled  = "user-put-disabled"
	skillBundleDupSkillSlug     = "dup-skill"
	skillBundlePatchSkillSlug   = "patch-skill"
	skillBundleDupeSkillName    = "dupe-skill"
	skillBundleDupe1Slug        = "dupe-1"
	skillBundleDupe2Slug        = "dupe-2"
	skillBundleMissingSkillSlug = "missing-skill"
	skillBundleBadSkillSlug     = "bad-skill"
)

func TestSkillStore_PutSkill_Errors(t *testing.T) {
	t.Parallel()
	s := newTestSkillStore(t)

	putBundle(t, s, skillBundleBen, "bundle-enabled", "Enabled Bundle", true)
	putBundle(t, s, skillBundleBdis, "bundle-disabled", "Disabled Bundle", false)

	skillRoot := t.TempDir()
	loc := writeSkillPackage(t, skillRoot, "putskill-ok", "desc", "BODY")

	tests := []struct {
		name   string
		req    *spec.PutSkillRequest
		wantIs error
	}{
		{"nil-req", nil, spec.ErrSkillInvalidRequest},
		{
			"nil-body",
			&spec.PutSkillRequest{BundleID: skillBundleBen, SkillSlug: skillBundleS1, Body: nil},
			spec.ErrSkillInvalidRequest,
		},
		{
			"empty-bundleid",
			&spec.PutSkillRequest{BundleID: "", SkillSlug: skillBundleS1, Body: &spec.PutSkillRequestBody{}},
			spec.ErrSkillInvalidRequest,
		},
		{
			"empty-skillSlug",
			&spec.PutSkillRequest{BundleID: skillBundleBen, SkillSlug: "", Body: &spec.PutSkillRequestBody{}},
			spec.ErrSkillInvalidRequest,
		},
		{
			"invalid-skillSlug",
			&spec.PutSkillRequest{BundleID: skillBundleBen, SkillSlug: badSlug, Body: &spec.PutSkillRequestBody{}},
			spec.ErrSkillInvalidRequest,
		},
		{
			"skillType-not-fs",
			&spec.PutSkillRequest{
				BundleID:  skillBundleBen,
				SkillSlug: skillBundleS1,
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
				BundleID:  testNope,
				SkillSlug: skillBundleS1,
				Body: &spec.PutSkillRequestBody{
					SkillType: spec.SkillTypeFS,
					Location:  loc,
					Name:      "putskill-ok",
					IsEnabled: true,
				},
			},
			spec.ErrSkillBundleNotFound,
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
	putBundle(t, s, skillBundleB1, testBundleSlug, testBundleDisplayName, true)

	root := t.TempDir()

	locEnabled := writeSkillPackage(t, root, skillBundleUserPutEnabled, "desc enabled", "BODY_ENABLED")
	_, err := s.PutSkill(t.Context(), &spec.PutSkillRequest{
		BundleID:  skillBundleB1,
		SkillSlug: skillBundleUserPutEnabled,
		Body: &spec.PutSkillRequestBody{
			SkillType: spec.SkillTypeFS,
			Location:  locEnabled,
			Name:      skillBundleUserPutEnabled,
			IsEnabled: true,
			Tags:      []string{"t1"},
		},
	})
	if err != nil {
		t.Fatalf("PutSkill(enabled): %v", err)
	}

	recs := runtimeRecs(t, s)
	mustHaveSkillDef(
		t,
		recs,
		agentskillsSpec.SkillDef{Type: "fs", Name: skillBundleUserPutEnabled, Location: locEnabled},
	)

	// Disabled-at-create: store persists it, but runtime must not keep it after resync.
	locDisabled := writeSkillPackage(t, root, skillBundleUserPutDisabled, "desc disabled", "BODY_DISABLED")
	_, err = s.PutSkill(t.Context(), &spec.PutSkillRequest{
		BundleID:  skillBundleB1,
		SkillSlug: skillBundleUserPutDisabled,
		Body: &spec.PutSkillRequestBody{
			SkillType: spec.SkillTypeFS,
			Location:  locDisabled,
			Name:      skillBundleUserPutDisabled,
			IsEnabled: false,
			Tags:      []string{"t1"},
		},
	})
	if err != nil {
		t.Fatalf("PutSkill(disabled): %v", err)
	}

	recs = runtimeRecs(t, s)
	mustNotHaveSkillDef(
		t,
		recs,
		agentskillsSpec.SkillDef{Type: "fs", Name: skillBundleUserPutDisabled, Location: locDisabled},
	)

	// Store has it (GetSkill should fail because disabled).
	_, err = s.GetSkill(
		t.Context(),
		&spec.GetSkillRequest{BundleID: skillBundleB1, SkillSlug: skillBundleUserPutDisabled},
	)
	if err == nil || !errors.Is(err, spec.ErrSkillDisabled) {
		t.Fatalf("expected ErrSkillDisabled, got %v", err)
	}
}

func TestSkillStore_PutSkill_RuntimeRejected_DoesNotPersist(t *testing.T) {
	t.Parallel()
	s := newTestSkillStore(t)
	putBundle(t, s, skillBundleB1, testBundleSlug, testBundleDisplayName, true)

	// Create an empty directory that does NOT contain SKILL.md. Runtime indexing should reject it.
	locBad := filepath.Join(t.TempDir(), skillBundleBadSkillSlug)
	if err := os.MkdirAll(locBad, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	_, err := s.PutSkill(t.Context(), &spec.PutSkillRequest{
		BundleID:  skillBundleB1,
		SkillSlug: skillBundleBadSkillSlug,
		Body: &spec.PutSkillRequestBody{
			SkillType: spec.SkillTypeFS,
			Location:  locBad,
			Name:      skillBundleBadSkillSlug,
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
		BundleIDs:           []bundleitemutils.BundleID{skillBundleB1},
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
	putBundle(t, s, skillBundleB1, testBundleSlug, testBundleDisplayName, true)

	root := t.TempDir()
	loc := writeSkillPackage(t, root, skillBundleDupSkillSlug, "desc", "BODY")

	_, err := s.PutSkill(t.Context(), &spec.PutSkillRequest{
		BundleID:  skillBundleB1,
		SkillSlug: skillBundleDupSkillSlug,
		Body: &spec.PutSkillRequestBody{
			SkillType: spec.SkillTypeFS,
			Location:  loc,
			Name:      skillBundleDupSkillSlug,
			IsEnabled: true,
		},
	})
	if err != nil {
		t.Fatalf("PutSkill(1): %v", err)
	}

	_, err = s.PutSkill(t.Context(), &spec.PutSkillRequest{
		BundleID:  skillBundleB1,
		SkillSlug: skillBundleDupSkillSlug,
		Body: &spec.PutSkillRequestBody{
			SkillType: spec.SkillTypeFS,
			Location:  loc,
			Name:      skillBundleDupSkillSlug,
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
		BundleID:  skillBundleB1,
		SkillSlug: skillBundleS1,
		Body:      &spec.PatchSkillRequestBody{},
	})
	if err == nil || !errors.Is(err, spec.ErrSkillInvalidRequest) {
		t.Fatalf("expected ErrSkillInvalidRequest, got %v", err)
	}
}

func TestSkillStore_PatchSkill_EnableAndLocationChange_PresenceResetAndRuntimeDelta(t *testing.T) {
	t.Parallel()
	s := newTestSkillStore(t)
	putBundle(t, s, skillBundleB1, testBundleSlug, testBundleDisplayName, true)

	root1 := filepath.Join(t.TempDir(), "v1")
	root2 := filepath.Join(t.TempDir(), "v2")
	loc1 := writeSkillPackage(t, root1, skillBundlePatchSkillSlug, "desc v1", "BODY_V1")
	loc2 := writeSkillPackage(t, root2, skillBundlePatchSkillSlug, "desc v2", "BODY_V2")

	_, err := s.PutSkill(t.Context(), &spec.PutSkillRequest{
		BundleID:  skillBundleB1,
		SkillSlug: skillBundlePatchSkillSlug,
		Body: &spec.PutSkillRequestBody{
			SkillType: spec.SkillTypeFS,
			Location:  loc1,
			Name:      skillBundlePatchSkillSlug,
			IsEnabled: false,
		},
	})
	if err != nil {
		t.Fatalf("PutSkill(disabled): %v", err)
	}

	// Enable -> should appear in runtime.
	_, err = s.PatchSkill(t.Context(), &spec.PatchSkillRequest{
		BundleID:  skillBundleB1,
		SkillSlug: skillBundlePatchSkillSlug,
		Body:      &spec.PatchSkillRequestBody{IsEnabled: new(true)},
	})
	if err != nil {
		t.Fatalf("PatchSkill(enable): %v", err)
	}
	recs := runtimeRecs(t, s)
	mustHaveSkillDef(t, recs, agentskillsSpec.SkillDef{Type: "fs", Name: skillBundlePatchSkillSlug, Location: loc1})

	// Location change -> must validate new, remove old, reset presence.
	_, err = s.PatchSkill(t.Context(), &spec.PatchSkillRequest{
		BundleID:  skillBundleB1,
		SkillSlug: skillBundlePatchSkillSlug,
		Body:      &spec.PatchSkillRequestBody{Location: new(loc2)},
	})
	if err != nil {
		t.Fatalf("PatchSkill(location): %v", err)
	}
	recs = runtimeRecs(t, s)
	mustHaveSkillDef(t, recs, agentskillsSpec.SkillDef{Type: "fs", Name: skillBundlePatchSkillSlug, Location: loc2})
	mustNotHaveSkillDef(t, recs, agentskillsSpec.SkillDef{Type: "fs", Name: skillBundlePatchSkillSlug, Location: loc1})

	gs, err := s.GetSkill(
		t.Context(),
		&spec.GetSkillRequest{BundleID: skillBundleB1, SkillSlug: skillBundlePatchSkillSlug},
	)
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
	putBundle(t, s, skillBundleB1, testBundleSlug, testBundleDisplayName, true)

	// Two store skills pointing to the same underlying runtime def.
	root1 := filepath.Join(t.TempDir(), "v1")
	root2 := filepath.Join(t.TempDir(), "v2")
	loc1 := writeSkillPackage(t, root1, skillBundleDupeSkillName, "desc v1", "BODY_V1")
	loc2 := writeSkillPackage(t, root2, skillBundleDupeSkillName, "desc v2", "BODY_V2")

	_, err := s.PutSkill(t.Context(), &spec.PutSkillRequest{
		BundleID:  skillBundleB1,
		SkillSlug: skillBundleDupe1Slug,
		Body: &spec.PutSkillRequestBody{
			SkillType: spec.SkillTypeFS,
			Location:  loc1,
			Name:      skillBundleDupeSkillName,
			IsEnabled: true,
		},
	})
	if err != nil {
		t.Fatalf("PutSkill(dupe-1): %v", err)
	}
	_, err = s.PutSkill(t.Context(), &spec.PutSkillRequest{
		BundleID:  skillBundleB1,
		SkillSlug: skillBundleDupe2Slug,
		Body: &spec.PutSkillRequestBody{
			SkillType: spec.SkillTypeFS,
			Location:  loc1,
			Name:      skillBundleDupeSkillName,
			IsEnabled: true,
		},
	})
	if err != nil {
		t.Fatalf("PutSkill(dupe-2): %v", err)
	}

	recs := runtimeRecs(t, s)
	mustHaveSkillDef(t, recs, agentskillsSpec.SkillDef{Type: "fs", Name: skillBundleDupeSkillName, Location: loc1})

	// Patch only one location: old def must stay because dupe-2 still wants it.
	_, err = s.PatchSkill(t.Context(), &spec.PatchSkillRequest{
		BundleID:  skillBundleB1,
		SkillSlug: skillBundleDupe1Slug,
		Body:      &spec.PatchSkillRequestBody{Location: new(loc2)},
	})
	if err != nil {
		t.Fatalf("PatchSkill(dupe-1 location): %v", err)
	}

	recs = runtimeRecs(t, s)
	mustHaveSkillDef(t, recs, agentskillsSpec.SkillDef{Type: "fs", Name: skillBundleDupeSkillName, Location: loc1})
	mustHaveSkillDef(t, recs, agentskillsSpec.SkillDef{Type: "fs", Name: skillBundleDupeSkillName, Location: loc2})

	// Delete the remaining reference to loc1 -> runtime must remove loc1 now.
	_, err = s.DeleteSkill(
		t.Context(),
		&spec.DeleteSkillRequest{BundleID: skillBundleB1, SkillSlug: skillBundleDupe2Slug},
	)
	if err != nil {
		t.Fatalf("DeleteSkill(dupe-2): %v", err)
	}
	recs = runtimeRecs(t, s)
	mustNotHaveSkillDef(t, recs, agentskillsSpec.SkillDef{Type: "fs", Name: skillBundleDupeSkillName, Location: loc1})
	mustHaveSkillDef(t, recs, agentskillsSpec.SkillDef{Type: "fs", Name: skillBundleDupeSkillName, Location: loc2})
}

func TestSkillStore_DeleteSkill_MissingPresenceBlocked(t *testing.T) {
	t.Parallel()
	s := newTestSkillStore(t)
	putBundle(t, s, skillBundleB1, testBundleSlug, testBundleDisplayName, true)

	root := t.TempDir()
	loc := writeSkillPackage(t, root, skillBundleMissingSkillSlug, "desc", "BODY")

	_, err := s.PutSkill(t.Context(), &spec.PutSkillRequest{
		BundleID:  skillBundleB1,
		SkillSlug: skillBundleMissingSkillSlug,
		Body: &spec.PutSkillRequestBody{
			SkillType: spec.SkillTypeFS,
			Location:  loc,
			Name:      skillBundleMissingSkillSlug,
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
	sk := all.Skills[skillBundleB1][skillBundleMissingSkillSlug]
	sk.Presence = &spec.SkillPresence{Status: spec.SkillPresenceMissing}
	all.Skills[skillBundleB1][skillBundleMissingSkillSlug] = sk
	writeAllUserLocked(t, s, all)

	_, err = s.DeleteSkill(
		t.Context(),
		&spec.DeleteSkillRequest{BundleID: skillBundleB1, SkillSlug: skillBundleMissingSkillSlug},
	)
	if err == nil || !errors.Is(err, spec.ErrSkillIsMissing) {
		t.Fatalf("expected ErrSkillIsMissing, got %v", err)
	}
}

func TestSkillStore_DeleteSkillBundle_NotEmpty(t *testing.T) {
	t.Parallel()
	s := newTestSkillStore(t)
	putBundle(t, s, skillBundleB1, testBundleSlug, testBundleDisplayName, true)

	root := t.TempDir()
	loc := writeSkillPackage(t, root, "some-skill", "desc", "BODY")
	_, err := s.PutSkill(t.Context(), &spec.PutSkillRequest{
		BundleID:  skillBundleB1,
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

	_, err = s.DeleteSkillBundle(t.Context(), &spec.DeleteSkillBundleRequest{BundleID: skillBundleB1})
	if err == nil || !errors.Is(err, spec.ErrSkillBundleNotEmpty) {
		t.Fatalf("expected ErrSkillBundleNotEmpty, got %v", err)
	}
}

func TestSkillStore_PutSkillBundle_SoftDeletedCannotBeRecreated(t *testing.T) {
	t.Parallel()
	s := newTestSkillStore(t)

	putBundle(t, s, skillBundleB1, testBundleSlug, testBundleDisplayName, true)

	_, err := s.DeleteSkillBundle(t.Context(), &spec.DeleteSkillBundleRequest{BundleID: skillBundleB1})
	if err != nil {
		t.Fatalf("DeleteSkillBundle: %v", err)
	}

	_, err = s.PutSkillBundle(t.Context(), &spec.PutSkillBundleRequest{
		BundleID: skillBundleB1,
		Body: &spec.PutSkillBundleRequestBody{
			Slug:        testBundleSlug,
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

	got, err := s.enabledDefCountsInUserBundle(nil, skillBundleB1)
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
	got, err = s.enabledDefCountsInUserBundle(sc, skillBundleB1)
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
			skillBundleB1: {
				SchemaVersion: spec.SkillSchemaVersion,
				ID:            skillBundleB1,
				Slug:          testBundleSlug,
				DisplayName:   testBundleDisplayName,
				IsEnabled:     true,
				CreatedAt:     now,
				ModifiedAt:    now,
			},
		},
		Skills: map[bundleitemutils.BundleID]map[spec.SkillSlug]spec.Skill{
			skillBundleB1: {
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

	_, err := s.enabledDefCountsInUserBundle(sc, skillBundleB1)
	if err == nil || !errors.Is(err, spec.ErrSkillInvalidRequest) {
		t.Fatalf("expected ErrSkillInvalidRequest, got %v", err)
	}
}

func runtimeRecs(t *testing.T, s *SkillStore) []agentskillsSpec.SkillRecord {
	t.Helper()
	if s == nil || s.runtime == nil {
		t.Fatalf("runtime not configured in test store")
	}
	recs, err := s.runtime.ListSkills(t.Context(), nil)
	if err != nil {
		t.Fatalf("runtime.ListSkills: %v", err)
	}
	return recs
}
