package store

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/builtin"
	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/skill/spec"
)

const (
	mutated = "MUTATED"
	badSlug = "BAD SLUG"
)

func TestNewBuiltInSkills_InvalidArgs(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	_, err := NewBuiltInSkills(ctx, "", time.Second)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, spec.ErrSkillInvalidRequest) {
		t.Fatalf("expected ErrSkillInvalidRequest, got %v", err)
	}
}

func TestBuiltInSkills_LoadFromFS_HappyPathAndClones(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	overlayDir := t.TempDir()

	td := filepath.Join(".", "testdata")
	fsys := os.DirFS(td)

	b, err := NewBuiltInSkills(
		ctx,
		overlayDir,
		5*time.Second,
		WithBuiltInSkillsFS(fsys, "."),
	)
	if err != nil {
		t.Fatalf("NewBuiltInSkills: %v", err)
	}
	t.Cleanup(func() {
		_ = b.Close()
	})

	bundles, skills, err := b.ListBuiltInSkills(ctx)
	if err != nil {
		t.Fatalf("ListBuiltInSkills: %v", err)
	}
	if len(bundles) != 2 {
		t.Fatalf("expected 2 bundles, got %d", len(bundles))
	}
	if len(skills) != 2 {
		t.Fatalf("expected 2 skills maps, got %d", len(skills))
	}

	// Verify built-in normalization is applied.
	sb1, ok := bundles[bundleitemutils.BundleID("builtin-bundle-1")]
	if !ok {
		t.Fatalf("missing builtin-bundle-1")
	}
	if !sb1.IsBuiltIn || sb1.SchemaVersion != spec.SkillSchemaVersion {
		t.Fatalf("bundle not normalized: %+v", sb1)
	}
	if sb1.Slug == "" || sb1.DisplayName == "" {
		t.Fatalf("unexpected empty fields: %+v", sb1)
	}

	// Clone safety: mutate returned maps and ensure store data doesn't change.
	sb1.DisplayName = mutated
	bundles[bundleitemutils.BundleID("builtin-bundle-1")] = sb1

	orig, err := b.GetBuiltInSkillBundle(ctx, bundleitemutils.BundleID("builtin-bundle-1"))
	if err != nil {
		t.Fatalf("GetBuiltInSkillBundle: %v", err)
	}
	if orig.DisplayName == mutated {
		t.Fatalf("bundle clone failed; mutation leaked into store")
	}

	// Clone safety for skills: mutate slices and nested pointers.
	sk, err := b.GetBuiltInSkill(ctx, bundleitemutils.BundleID("builtin-bundle-1"), spec.SkillSlug("hello"))
	if err != nil {
		t.Fatalf("GetBuiltInSkill: %v", err)
	}
	if !sk.IsBuiltIn || sk.Type != spec.SkillTypeEmbeddedFS {
		t.Fatalf("skill not normalized: %+v", sk)
	}

	// List returns cloneSkill; mutate list output then ensure Get returns unchanged.
	lsk := skills[bundleitemutils.BundleID("builtin-bundle-1")][spec.SkillSlug("hello")]
	if len(lsk.Tags) == 0 {
		t.Fatalf("expected tags")
	}
	lsk.Tags[0] = mutated
	if lsk.Presence == nil {
		t.Fatalf("expected presence")
	}
	lsk.Presence.Status = spec.SkillPresenceError
	skills[bundleitemutils.BundleID("builtin-bundle-1")][spec.SkillSlug("hello")] = lsk

	sk2, err := b.GetBuiltInSkill(ctx, bundleitemutils.BundleID("builtin-bundle-1"), spec.SkillSlug("hello"))
	if err != nil {
		t.Fatalf("GetBuiltInSkill: %v", err)
	}
	if sk2.Tags[0] == mutated {
		t.Fatalf("skill tags clone failed; mutation leaked into store")
	}
	if sk2.Presence != nil && sk2.Presence.Status == spec.SkillPresenceError {
		t.Fatalf("skill presence clone failed; mutation leaked into store")
	}
}

func TestBuiltInSkills_LoadFromFS_Errors(t *testing.T) {
	ctx := t.Context()

	mk := func(t *testing.T, raw string) fs.FS {
		t.Helper()
		return fstest.MapFS{
			builtin.BuiltInSkillBundlesJSON: &fstest.MapFile{Data: []byte(raw)},
		}
	}

	now := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)

	validSchema := func() skillStoreSchema {
		return skillStoreSchema{
			SchemaVersion: spec.SkillSchemaVersion,
			Bundles: map[bundleitemutils.BundleID]spec.SkillBundle{
				"b1": {
					SchemaVersion: spec.SkillSchemaVersion,
					ID:            "b1",
					Slug:          "ok-bundle",
					DisplayName:   "OK",
					Description:   "",
					IsEnabled:     true,
					CreatedAt:     now,
					ModifiedAt:    now,
				},
			},
			Skills: map[bundleitemutils.BundleID]map[spec.SkillSlug]spec.Skill{
				"b1": {
					"ok": {
						SchemaVersion: spec.SkillSchemaVersion,
						ID:            "s1",
						Slug:          "ok",
						Type:          spec.SkillTypeEmbeddedFS,
						Location:      "x",
						Name:          "n",
						IsEnabled:     true,
						CreatedAt:     now,
						ModifiedAt:    now,
					},
				},
			},
		}
	}

	marshal := func(t *testing.T, sc skillStoreSchema) string {
		t.Helper()
		b, err := json.Marshal(sc)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}
		return string(b)
	}

	tests := []struct {
		name    string
		rawFS   fs.FS
		wantSub string
	}{
		{
			name:    "invalid-json",
			rawFS:   mk(t, "{not json"),
			wantSub: "invalid character",
		},
		{
			name: "wrong-schema-version",
			rawFS: func() fs.FS {
				sc := validSchema()
				sc.SchemaVersion = "1900-01-01"
				return mk(t, marshal(t, sc))
			}(),
			wantSub: "schemaVersion",
		},
		{
			name: "no-bundles",
			rawFS: func() fs.FS {
				sc := validSchema()
				sc.Bundles = map[bundleitemutils.BundleID]spec.SkillBundle{}
				return mk(t, marshal(t, sc))
			}(),
			wantSub: "",
		},
		{
			name: "skills-reference-missing-bundle",
			rawFS: func() fs.FS {
				sc := validSchema()
				sc.Skills = map[bundleitemutils.BundleID]map[spec.SkillSlug]spec.Skill{
					"missing": {"ok": sc.Skills["b1"]["ok"]},
				}
				return mk(t, marshal(t, sc))
			}(),
			wantSub: "not present in bundles",
		},
		{
			name: "builtin-skill-wrong-type",
			rawFS: func() fs.FS {
				sc := validSchema()
				sk := sc.Skills["b1"]["ok"]
				sk.Type = spec.SkillTypeFS
				sc.Skills["b1"]["ok"] = sk
				return mk(t, marshal(t, sc))
			}(),
			wantSub: "type must be",
		},
		{
			name: "builtin-skill-map-key-slug-mismatch",
			rawFS: func() fs.FS {
				sc := validSchema()
				sk := sc.Skills["b1"]["ok"]
				sk.Slug = "DIFFERENT"
				sc.Skills["b1"]["ok"] = sk
				return mk(t, marshal(t, sc))
			}(),
			wantSub: "map key slug",
		},
		{
			name: "invalid-bundle-slug",
			rawFS: func() fs.FS {
				sc := validSchema()
				b := sc.Bundles["b1"]
				b.Slug = badSlug
				sc.Bundles["b1"] = b
				return mk(t, marshal(t, sc))
			}(),
			wantSub: "invalid slug",
		},
		{
			name: "invalid-skill-empty-location",
			rawFS: func() fs.FS {
				sc := validSchema()
				sk := sc.Skills["b1"]["ok"]
				sk.Location = ""
				sc.Skills["b1"]["ok"] = sk
				return mk(t, marshal(t, sc))
			}(),
			wantSub: "location is empty",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			overlayDir := t.TempDir()
			b, err := NewBuiltInSkills(
				ctx,
				overlayDir,
				time.Second,
				WithBuiltInSkillsFS(tc.rawFS, "."),
			)
			// Windows can't delete open files. Ensure any partially-open overlay DB
			// gets closed even when NewBuiltInSkills returns an error.
			if b != nil {
				t.Cleanup(func() {
					_ = b.Close()
					if runtime.GOOS == "windows" {
						// Give SQLite time to release handles on Windows.
						t.Log("skillstore: sleeping in win")
						time.Sleep(time.Millisecond * 100)
					}
				})
			}
			if tc.wantSub == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error")
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.wantSub)
			}
		})
	}
}

func TestBuiltInSkills_SetFlags_AndPersistence(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	overlayDir := t.TempDir()

	td := filepath.Join(".", "testdata")
	fsys := os.DirFS(td)

	b1, err := NewBuiltInSkills(ctx, overlayDir, time.Second, WithBuiltInSkillsFS(fsys, "."))
	if err != nil {
		t.Fatalf("NewBuiltInSkills: %v", err)
	}
	t.Cleanup(func() {
		_ = b1.Close()
	})

	// Toggle bundle.
	beforeBundle, err := b1.GetBuiltInSkillBundle(ctx, "builtin-bundle-1")
	if err != nil {
		t.Fatalf("GetBuiltInSkillBundle: %v", err)
	}
	afterBundle, err := b1.SetSkillBundleEnabled(ctx, "builtin-bundle-1", !beforeBundle.IsEnabled)
	if err != nil {
		t.Fatalf("SetSkillBundleEnabled: %v", err)
	}
	if afterBundle.IsEnabled == beforeBundle.IsEnabled {
		t.Fatalf("expected bundle enabled to toggle")
	}

	// Toggle skill.
	beforeSkill, err := b1.GetBuiltInSkill(ctx, "builtin-bundle-1", "hello")
	if err != nil {
		t.Fatalf("GetBuiltInSkill: %v", err)
	}
	afterSkill, err := b1.SetSkillEnabled(ctx, "builtin-bundle-1", "hello", !beforeSkill.IsEnabled)
	if err != nil {
		t.Fatalf("SetSkillEnabled: %v", err)
	}
	if afterSkill.IsEnabled == beforeSkill.IsEnabled {
		t.Fatalf("expected skill enabled to toggle")
	}

	// "Reopen" (new instance should see overlay flags).
	b2, err := NewBuiltInSkills(ctx, overlayDir, time.Second, WithBuiltInSkillsFS(fsys, "."))
	if err != nil {
		t.Fatalf("NewBuiltInSkills(reopen): %v", err)
	}
	t.Cleanup(func() {
		_ = b2.Close()
	})
	gotBundle, err := b2.GetBuiltInSkillBundle(ctx, "builtin-bundle-1")
	if err != nil {
		t.Fatalf("GetBuiltInSkillBundle(reopen): %v", err)
	}
	if gotBundle.IsEnabled != afterBundle.IsEnabled {
		t.Fatalf("bundle flag did not persist: got=%v want=%v", gotBundle.IsEnabled, afterBundle.IsEnabled)
	}
	gotSkill, err := b2.GetBuiltInSkill(ctx, "builtin-bundle-1", "hello")
	if err != nil {
		t.Fatalf("GetBuiltInSkill(reopen): %v", err)
	}
	if gotSkill.IsEnabled != afterSkill.IsEnabled {
		t.Fatalf("skill flag did not persist: got=%v want=%v", gotSkill.IsEnabled, afterSkill.IsEnabled)
	}
}

func TestBuiltInSkills_ConcurrentFlagUpdates(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	overlayDir := t.TempDir()

	// Generate a larger built-in schema in-memory for concurrency.
	now := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)
	sc := skillStoreSchema{
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
			"b1": {},
		},
	}
	for i := range 50 {
		slug := spec.SkillSlug("skill-" + strconv.Itoa(i))
		sc.Skills["b1"][slug] = spec.Skill{
			SchemaVersion: spec.SkillSchemaVersion,
			ID:            spec.SkillID("id-" + strconv.Itoa(i)),
			Slug:          slug,
			Type:          spec.SkillTypeEmbeddedFS,
			Location:      "loc-" + strconv.Itoa(i),
			Name:          "name-" + strconv.Itoa(i),
			IsEnabled:     true,
			CreatedAt:     now,
			ModifiedAt:    now,
		}
	}

	raw, err := json.Marshal(sc)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	fsys := fstest.MapFS{
		builtin.BuiltInSkillBundlesJSON: &fstest.MapFile{Data: raw},
	}

	b, err := NewBuiltInSkills(ctx, overlayDir, time.Second, WithBuiltInSkillsFS(fsys, "."))
	if err != nil {
		t.Fatalf("NewBuiltInSkills: %v", err)
	}
	t.Cleanup(func() {
		_ = b.Close()
	})

	var wg sync.WaitGroup
	errCh := make(chan error, 100)

	for slug := range sc.Skills["b1"] {
		wg.Add(1)
		go func(sl spec.SkillSlug) {
			defer wg.Done()
			_, err := b.SetSkillEnabled(ctx, "b1", sl, false)
			if err != nil {
				errCh <- err
			}
		}(slug)
	}

	wg.Wait()
	close(errCh)

	for e := range errCh {
		t.Fatalf("concurrent SetSkillEnabled error: %v", e)
	}

	// Spot-check one value deterministically.
	got, err := b.GetBuiltInSkill(ctx, "b1", "skill-0")
	if err != nil {
		t.Fatalf("GetBuiltInSkill: %v", err)
	}
	if got.IsEnabled {
		t.Fatalf("expected skill-0 to be disabled after concurrent updates")
	}
}

func TestResolveSkillsFS(t *testing.T) {
	fsys := fstest.MapFS{
		"a/skills.json": &fstest.MapFile{Data: []byte(`{}`)},
	}

	tests := []struct {
		name    string
		dir     string
		wantErr bool
	}{
		{"empty-dir", "", false},
		{"dot-dir", ".", false},
		{"subdir-ok", "a", false},
		{"subdir-missing", "nope", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := resolveSkillsFS(fsys, tc.dir)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

func TestGetBuiltInSkillKey(t *testing.T) {
	got := getBuiltInSkillKey("b1", "s1")
	if string(got) != "b1::s1" {
		t.Fatalf("got %q", got)
	}
}
