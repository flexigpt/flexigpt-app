package skillstore

import (
	"errors"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/skillstore/spec"
)

const (
	skillBundleBen              = "ben"
	skillBundleBdis             = "bdis"
	skillBundleB1               = "b1"
	skillBundleS1               = "s1"
	skillBundleDupSkillSlug     = "dup-skill"
	skillBundleMissingSkillSlug = "missing-skill"
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
		{"nil-req", nil, errSkillInvalidRequest},
		{
			"nil-body",
			&spec.PutSkillRequest{BundleID: skillBundleBen, SkillSlug: skillBundleS1, Body: nil},
			errSkillInvalidRequest,
		},
		{
			"empty-bundleid",
			&spec.PutSkillRequest{BundleID: "", SkillSlug: skillBundleS1, Body: &spec.PutSkillRequestBody{}},
			errSkillInvalidRequest,
		},
		{
			"empty-skillSlug",
			&spec.PutSkillRequest{BundleID: skillBundleBen, SkillSlug: "", Body: &spec.PutSkillRequestBody{}},
			errSkillInvalidRequest,
		},
		{
			"invalid-skillSlug",
			&spec.PutSkillRequest{BundleID: skillBundleBen, SkillSlug: badSlug, Body: &spec.PutSkillRequestBody{}},
			errSkillInvalidRequest,
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
			errSkillInvalidRequest,
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
			errSkillBundleNotFound,
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
	if err == nil || !errors.Is(err, errSkillConflict) {
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
	if err == nil || !errors.Is(err, errSkillInvalidRequest) {
		t.Fatalf("expected ErrSkillInvalidRequest, got %v", err)
	}
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
	if err == nil || !errors.Is(err, errSkillIsMissing) {
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
	if err == nil || !errors.Is(err, errSkillBundleNotEmpty) {
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
	if err == nil || !errors.Is(err, errSkillBundleDeleting) {
		t.Fatalf("expected ErrSkillBundleDeleting, got %v", err)
	}
}

func TestSkillStore_withUserWriteSaga_InvalidArgs(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	var nilStore *SkillStore
	err := nilStore.withUserWrite(ctx, "op", func(sc *skillStoreSchema) error {
		return nil
	})
	if err == nil || !errors.Is(err, errSkillInvalidRequest) {
		t.Fatalf("expected ErrSkillInvalidRequest, got %v", err)
	}

	s := newTestSkillStore(t)
	err = s.withUserWrite(ctx, "op", nil)
	if err == nil || !errors.Is(err, errSkillInvalidRequest) {
		t.Fatalf("expected ErrSkillInvalidRequest, got %v", err)
	}
}
