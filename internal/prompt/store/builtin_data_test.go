package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/builtin"
	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/prompt/spec"
	"github.com/ppipada/mapstore-go/uuidv7filename"
)

const (
	corrupted = "corrupted"
	demo      = "demo"
)

func TestNewBuiltInData(t *testing.T) {
	tests := []struct {
		name           string
		setupDir       func(t *testing.T) string
		snapshotMaxAge time.Duration
		wantErr        bool
		errContains    string
	}{
		{
			name: "happy_path",
			setupDir: func(t *testing.T) string {
				t.Helper()
				return t.TempDir()
			},
			snapshotMaxAge: time.Hour,
			wantErr:        false,
		},
		{
			name: "zero_snapshot_age_defaults_to_hour",
			setupDir: func(t *testing.T) string {
				t.Helper()
				return t.TempDir()
			},
			snapshotMaxAge: 0,
			wantErr:        false,
		},
		{
			name: "negative_snapshot_age_defaults_to_hour",
			setupDir: func(t *testing.T) string {
				t.Helper()
				return t.TempDir()
			},
			snapshotMaxAge: -1,
			wantErr:        false,
		},
		{
			name: "base_dir_is_file",
			setupDir: func(t *testing.T) string {
				t.Helper()
				tmp := t.TempDir()
				file := filepath.Join(tmp, "not-a-dir")
				if err := os.WriteFile(file, []byte("dummy"), 0o600); err != nil {
					t.Fatalf("setup: %v", err)
				}
				return file
			},
			snapshotMaxAge: time.Hour,
			wantErr:        true,
		},
		{
			name: "readonly_directory",
			setupDir: func(t *testing.T) string {
				t.Helper()
				tmp := t.TempDir()
				dir := filepath.Join(tmp, "readonly")
				if err := os.MkdirAll(dir, 0o444); err != nil {
					t.Fatalf("setup: %v", err)
				}
				return dir
			},
			snapshotMaxAge: time.Hour,
			// Unix-like systems will typically fail to create files in a 0444
			// directory; Windows does not honor these POSIX permission bits the
			// same way, so don't require an error there.
			wantErr: runtime.GOOS != "windows",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			dir := tt.setupDir(t)

			bi, err := NewBuiltInData(ctx, dir, tt.snapshotMaxAge)
			t.Cleanup(func() { _ = bi.Close() })

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify basic functionality.
			bundles, templates, _ := bi.ListBuiltInData(ctx)
			if len(bundles) == 0 {
				t.Error("expected at least one bundle")
			}
			if len(templates) == 0 {
				t.Error("expected at least one template")
			}

			// Verify overlay file creation.
			if _, err := os.Stat(filepath.Join(dir, spec.PromptBuiltInOverlayDBFileName)); err != nil {
				t.Errorf("overlay file not created: %v", err)
			}

			// Verify all bundles are marked as built-in.
			for id, bundle := range bundles {
				if !bundle.IsBuiltIn {
					t.Errorf("bundle %s should be marked as built-in", id)
				}
			}

			// Verify all templates are marked as built-in.
			for bid, tmap := range templates {
				for tid, template := range tmap {
					if !template.IsBuiltIn {
						t.Errorf("template %s/%s should be marked as built-in", bid, tid)
					}
				}
			}
		})
	}
}

func TestSetBundleEnabled(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, bi *BuiltInData) (bundleitemutils.BundleID, bool)
		wantErr bool
	}{
		{
			name: "enable_existing_bundle",
			setup: func(t *testing.T, bi *BuiltInData) (bundleitemutils.BundleID, bool) {
				t.Helper()
				bundles, _, _ := bi.ListBuiltInData(t.Context())
				id, bundle := anyBundle(bundles)
				return id, !bundle.IsEnabled
			},
			wantErr: false,
		},
		{
			name: "disable_existing_bundle",
			setup: func(t *testing.T, bi *BuiltInData) (bundleitemutils.BundleID, bool) {
				t.Helper()
				bundles, _, _ := bi.ListBuiltInData(t.Context())
				id, bundle := anyBundle(bundles)
				return id, !bundle.IsEnabled
			},
			wantErr: false,
		},
		{
			name: "nonexistent_bundle",
			setup: func(t *testing.T, bi *BuiltInData) (bundleitemutils.BundleID, bool) {
				t.Helper()
				return bundleitemutils.BundleID("does-not-exist"), true
			},
			wantErr: true,
		},
		{
			name: "empty_bundle_id",
			setup: func(t *testing.T, bi *BuiltInData) (bundleitemutils.BundleID, bool) {
				t.Helper()
				return bundleitemutils.BundleID(""), true
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			dir := t.TempDir()
			bi, err := NewBuiltInData(ctx, dir, 0)
			if err != nil {
				t.Fatalf("setup failed: %v", err)
			}
			t.Cleanup(func() { _ = bi.Close() })

			bundleID, enabled := tt.setup(t, bi)

			_, err = bi.SetBundleEnabled(ctx, bundleID, enabled)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify the change is reflected in the snapshot.
			bundles, _, _ := bi.ListBuiltInData(ctx)
			if got := bundles[bundleID].IsEnabled; got != enabled {
				t.Errorf("bundle enabled state = %v, want %v", got, enabled)
			}

			// Verify persistence to overlay store (not disk).
			flag, ok, err := bi.bundleOverlayFlags.GetFlag(ctx, builtInBundleID(bundleID))
			if err != nil {
				t.Fatalf("overlay store: %v", err)
			}
			if !ok || flag.Value != enabled {
				t.Errorf(
					"overlay store: present=%v val=%v, want present=true val=%v",
					ok,
					flag.Value,
					enabled,
				)
			}
		})
	}
}

func TestSetTemplateEnabled(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, bi *BuiltInData) (bundleitemutils.BundleID, spec.PromptTemplate, bool)
		wantErr bool
	}{
		{
			name: "enable_existing_template",
			setup: func(t *testing.T, bi *BuiltInData) (bundleitemutils.BundleID, spec.PromptTemplate, bool) {
				t.Helper()
				_, templates, _ := bi.ListBuiltInData(t.Context())
				bid, _, tpl := anyTemplate(templates)
				return bid, tpl, !tpl.IsEnabled
			},
			wantErr: false,
		},
		{
			name: "disable_existing_template",
			setup: func(t *testing.T, bi *BuiltInData) (bundleitemutils.BundleID, spec.PromptTemplate, bool) {
				t.Helper()
				_, templates, _ := bi.ListBuiltInData(t.Context())
				bid, _, tpl := anyTemplate(templates)
				return bid, tpl, !tpl.IsEnabled
			},
			wantErr: false,
		},
		{
			name: "nonexistent_bundle",
			setup: func(t *testing.T, bi *BuiltInData) (bundleitemutils.BundleID, spec.PromptTemplate, bool) {
				t.Helper()
				return bundleitemutils.BundleID(
						"does-not-exist",
					), spec.PromptTemplate{
						ID:      bundleitemutils.ItemID("template"),
						Slug:    "template",
						Version: "a",
					}, true
			},
			wantErr: true,
		},
		{
			name: "nonexistent_template",
			setup: func(t *testing.T, bi *BuiltInData) (bundleitemutils.BundleID, spec.PromptTemplate, bool) {
				t.Helper()
				bundles, _, _ := bi.ListBuiltInData(t.Context())
				bid, _ := anyBundle(bundles)
				return bid, spec.PromptTemplate{
						ID:      bundleitemutils.ItemID("does-not-exist"),
						Slug:    "does-not-exist",
						Version: "a",
					},
					true
			},
			wantErr: true,
		},
		{
			name: "empty_template_id",
			setup: func(t *testing.T, bi *BuiltInData) (bundleitemutils.BundleID, spec.PromptTemplate, bool) {
				t.Helper()
				bundles, _, _ := bi.ListBuiltInData(t.Context())
				bid, _ := anyBundle(bundles)
				return bid, spec.PromptTemplate{
					ID:      "",
					Slug:    "",
					Version: "a",
				}, true
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			dir := t.TempDir()
			bi, err := NewBuiltInData(ctx, dir, 0)
			if err != nil {
				t.Fatalf("setup failed: %v", err)
			}
			t.Cleanup(func() { _ = bi.Close() })

			bundleID, template, enabled := tt.setup(t, bi)

			_, err = bi.SetTemplateEnabled(ctx, bundleID, template.Slug, template.Version, enabled)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify the change is reflected in the snapshot.
			_, templates, _ := bi.ListBuiltInData(ctx)
			if got := templates[bundleID][template.ID].IsEnabled; got != enabled {
				t.Errorf("template enabled state = %v, want %v", got, enabled)
			}

			// Verify persistence to overlay store (not disk).
			flag, ok, err := bi.templateOverlayFlags.GetFlag(
				ctx,
				getTemplateKey(bundleID, template.ID),
			)
			if err != nil {
				t.Fatalf("overlay store: %v", err)
			}
			if !ok || flag.Value != enabled {
				t.Errorf(
					"overlay store: present=%v val=%v, want present=true val=%v",
					ok,
					flag.Value,
					enabled,
				)
			}
		})
	}
}

func TestList(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	bi, err := NewBuiltInData(ctx, dir, 0)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	t.Cleanup(func() { _ = bi.Close() })

	t.Run("returns_independent_copies", func(t *testing.T) {
		b1, t1, _ := bi.ListBuiltInData(ctx)
		b2, t2, _ := bi.ListBuiltInData(ctx)

		// Mutate first copy.
		for id := range b1 {
			b := b1[id]
			b.Slug = corrupted
			b1[id] = b
			break
		}
		for bid := range t1 {
			for tid := range t1[bid] {
				tpl := t1[bid][tid]
				tpl.Slug = corrupted
				t1[bid][tid] = tpl
				break
			}
			break
		}

		// Verify second copy is unaffected.
		for id, b := range b2 {
			if b.Slug == corrupted {
				t.Errorf("bundle %s shares memory with first copy", id)
			}
		}
		for bid, tm := range t2 {
			for tid, tpl := range tm {
				if tpl.Slug == corrupted {
					t.Errorf("template %s/%s shares memory with first copy", bid, tid)
				}
			}
		}
	})

	t.Run("returns_consistent_data", func(t *testing.T) {
		bundles, templates, _ := bi.ListBuiltInData(ctx)

		// Verify all bundles have corresponding template maps.
		for bundleID := range bundles {
			if _, exists := templates[bundleID]; !exists {
				t.Errorf("bundle %s has no corresponding template map", bundleID)
			}
		}

		// Verify all template maps have corresponding bundles.
		for bundleID := range templates {
			if _, exists := bundles[bundleID]; !exists {
				t.Errorf("template map for bundle %s has no corresponding bundle", bundleID)
			}
		}
	})

	t.Run("all_items_marked_builtin", func(t *testing.T) {
		bundles, templates, _ := bi.ListBuiltInData(ctx)

		for id, bundle := range bundles {
			if !bundle.IsBuiltIn {
				t.Errorf("bundle %s not marked as built-in", id)
			}
		}

		for bid, tmap := range templates {
			for tid, template := range tmap {
				if !template.IsBuiltIn {
					t.Errorf("template %s/%s not marked as built-in", bid, tid)
				}
			}
		}
	})
}

func TestConcurrency(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	bi, err := NewBuiltInData(ctx, dir, time.Millisecond*10) // Short rebuild interval
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	t.Cleanup(func() { _ = bi.Close() })

	bundles, templates, _ := bi.ListBuiltInData(ctx)
	if len(bundles) == 0 || len(templates) == 0 {
		t.Skip("no test data available")
	}

	bundleID, _ := anyBundle(bundles)
	templateBundleID, _, template := anyTemplate(templates)

	t.Run("concurrent_reads", func(t *testing.T) {
		var wg sync.WaitGroup
		const numReaders = 10

		for range numReaders {
			wg.Go(func() {
				for range 100 {
					bundles, templates, _ := bi.ListBuiltInData(ctx)
					if len(bundles) == 0 || len(templates) == 0 {
						t.Errorf("got empty results during concurrent read")
						return
					}
				}
			})
		}

		wg.Wait()
	})

	t.Run("concurrent_writes", func(t *testing.T) {
		var wg sync.WaitGroup
		const numWriters = 5

		for i := range numWriters {
			wg.Add(2)
			go func(i int) {
				defer wg.Done()
				enabled := i%2 == 0
				if _, err := bi.SetBundleEnabled(ctx, bundleID, enabled); err != nil {
					t.Errorf("SetBundleEnabled failed: %v", err)
				}
			}(i)

			go func(i int) {
				defer wg.Done()
				enabled := i%2 == 0
				if _, err := bi.SetTemplateEnabled(
					ctx,
					templateBundleID,
					template.Slug,
					template.Version,
					enabled,
				); err != nil {
					t.Errorf("SetTemplateEnabled failed: %v", err)
				}
			}(i)
		}

		wg.Wait()
	})

	t.Run("concurrent_read_write", func(t *testing.T) {
		var wg sync.WaitGroup
		done := make(chan struct{})

		// Start readers.
		for range 5 {
			wg.Go(func() {
				for {
					select {
					case <-done:
						return
					default:
						_, _, _ = bi.ListBuiltInData(ctx)
					}
				}
			})
		}

		// Start writers.
		for i := range 3 {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				for j := range 10 {
					enabled := (i+j)%2 == 0
					if _, e := bi.SetBundleEnabled(ctx, bundleID, enabled); e != nil {
						t.Errorf("SetBundleEnabled failed: %v", e)
					}
					if _, e := bi.SetTemplateEnabled(
						ctx,
						templateBundleID,
						template.Slug,
						template.Version,
						enabled,
					); e != nil {
						t.Errorf("SetTemplateEnabled failed: %v", e)
					}
				}
			}(i)
		}

		// Let it run for a bit.
		time.Sleep(100 * time.Millisecond)
		close(done)
		wg.Wait()
	})
}

func TestRebuildSnapshot(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	bi, err := NewBuiltInData(ctx, dir, time.Hour) // Long interval to control rebuilds
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	t.Cleanup(func() { _ = bi.Close() })

	bundles, _, _ := bi.ListBuiltInData(ctx)
	bundleID, bundle := anyBundle(bundles)

	// Change the overlay directly.
	_, err = bi.bundleOverlayFlags.SetFlag(ctx, builtInBundleID(bundleID), !bundle.IsEnabled)
	if err != nil {
		t.Fatalf("failed to set overlay: %v", err)
	}

	// Force rebuild.
	if err := bi.rebuildSnapshot(ctx); err != nil {
		t.Fatalf("rebuildSnapshot failed: %v", err)
	}

	// Verify the change is reflected.
	newBundles, _, _ := bi.ListBuiltInData(ctx)
	if newBundles[bundleID].IsEnabled == bundle.IsEnabled {
		t.Error("rebuild did not apply overlay changes")
	}
}

func TestAsyncRebuild(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	bi, err := NewBuiltInData(ctx, dir, time.Millisecond) // Very short interval
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	t.Cleanup(func() { _ = bi.Close() })

	bundles, _, _ := bi.ListBuiltInData(ctx)
	bundleID, bundle := anyBundle(bundles)

	// Trigger async rebuild by making the snapshot stale.
	time.Sleep(2 * time.Millisecond)

	// Make a change that should trigger rebuild.
	if _, err := bi.SetBundleEnabled(ctx, bundleID, !bundle.IsEnabled); err != nil {
		t.Fatalf("SetBundleEnabled failed: %v", err)
	}

	// Give async rebuild time to complete.
	time.Sleep(10 * time.Millisecond)

	// Verify the change is eventually reflected.
	newBundles, _, _ := bi.ListBuiltInData(ctx)
	if newBundles[bundleID].IsEnabled == bundle.IsEnabled {
		t.Error("async rebuild did not apply changes")
	}
}

func Test_NewBuiltInData_SyntheticFS_MissingBundlesJSON(t *testing.T) {
	_, err := newFromFS(t, fstest.MapFS{})
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("want fs.ErrNotExist, got %v", err)
	}
}

// Test when the bundles json file is present but contains invalid JSON.
func Test_NewBuiltInData_SyntheticFS_InvalidBundlesJSON(t *testing.T) {
	fsys := fstest.MapFS{
		builtin.BuiltInPromptBundlesJSON: {Data: []byte("{ oops ]")},
	}
	_, err := newFromFS(t, fsys)
	if err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Test when the manifest contains no bundles.
func Test_NewBuiltInData_SyntheticFS_NoBundlesInManifest(t *testing.T) {
	empty, _ := json.Marshal(
		spec.AllBundles{Bundles: map[bundleitemutils.BundleID]spec.PromptBundle{}},
	)
	fsys := fstest.MapFS{builtin.BuiltInPromptBundlesJSON: {Data: empty}}
	_, err := newFromFS(t, fsys)
	if err == nil || !strings.Contains(err.Error(), "contains no bundles") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Test when a bundle directory is present but the bundle has no templates.
func Test_NewBuiltInData_SyntheticFS_BundleHasNoTemplates(t *testing.T) {
	bundleID := newUUID(t)
	slug := demo

	dir := fmt.Sprintf("%s_%s", bundleID, slug)
	fsys := fstest.MapFS{
		builtin.BuiltInPromptBundlesJSON: {Data: buildManifest(bundleID, slug)},
		dir:                              &fstest.MapFile{Mode: fs.ModeDir},
	}
	_, err := newFromFS(t, fsys)
	if err == nil || !strings.Contains(err.Error(), "has no templates") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Test when a bundle directory exists that is not listed in the manifest.
func Test_NewBuiltInData_SyntheticFS_BundleDirNotInManifest(t *testing.T) {
	bundleID := newUUID(t)
	slug := demo

	ghostID := newUUID(t)
	ghostSlug := "ghost"
	dir := fmt.Sprintf("%s_%s", ghostID, ghostSlug)

	fn, raw, _ := buildTemplate(t, ghostSlug, "v1")
	fsys := fstest.MapFS{
		builtin.BuiltInPromptBundlesJSON: {Data: buildManifest(bundleID, slug)},
		dir:                              &fstest.MapFile{Mode: fs.ModeDir},
		dir + "/" + fn:                   {Data: raw},
	}
	_, err := newFromFS(t, fsys)
	if err == nil ||
		!strings.Contains(err.Error(), "not in "+builtin.BuiltInPromptBundlesJSON) {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Test when the directory slug doesn't match the manifest slug for that bundle.
func Test_NewBuiltInData_SyntheticFS_SlugMismatchBetweenDirAndManifest(t *testing.T) {
	bundleID := newUUID(t)
	slug := demo

	// Dir slug = wrong, manifest slug = demo.
	dir := fmt.Sprintf("%s_%s", bundleID, "wrong")

	fn, raw, _ := buildTemplate(t, "wrong", "v1")
	fsys := fstest.MapFS{
		builtin.BuiltInPromptBundlesJSON: {Data: buildManifest(bundleID, slug)},
		dir:                              &fstest.MapFile{Mode: fs.ModeDir},
		dir + "/" + fn:                   {Data: raw},
	}
	_, err := newFromFS(t, fsys)
	if err == nil || !strings.Contains(err.Error(), "dir slug") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Test when the filename implies a different slug/version than the JSON content.
func Test_NewBuiltInData_SyntheticFS_FilenameSlugVersionMismatch(t *testing.T) {
	bundleID := newUUID(t)
	slug := demo

	dir := fmt.Sprintf("%s_%s", bundleID, slug)

	// Template JSON says slug=demo,ver=v1.
	_, raw, _ := buildTemplate(t, slug, "v1")
	// But store it under file name demo_v2.json.
	fsys := fstest.MapFS{
		builtin.BuiltInPromptBundlesJSON: {Data: buildManifest(bundleID, slug)},
		dir:                              &fstest.MapFile{Mode: fs.ModeDir},
		dir + "/demo_v2.json":            {Data: raw},
	}
	_, err := newFromFS(t, fsys)
	if err == nil || !strings.Contains(err.Error(), "filename") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func Test_NewBuiltInData_SyntheticFS_HappyAndCRUD(t *testing.T) {
	ctx := t.Context()
	bid := newUUID(t)
	slug := demo

	// Build valid dir template.
	dir := fmt.Sprintf("%s_%s", bid, slug)
	fn, rawTpl, _ := buildTemplate(t, slug, "v1")

	mem := fstest.MapFS{
		builtin.BuiltInPromptBundlesJSON: {Data: buildManifest(bid, slug)},
		dir:                              &fstest.MapFile{Mode: fs.ModeDir},
		dir + "/" + fn:                   {Data: rawTpl},
	}

	tmp := t.TempDir()
	bi, err := NewBuiltInData(ctx, tmp, 0, WithBundlesFS(mem, "."))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Cleanup(func() { _ = bi.Close() })

	bundles, tpls, _ := bi.ListBuiltInData(ctx)
	if len(bundles) != 1 || len(tpls) != 1 {
		t.Fatalf("want 1/1 objects, got %d/%d", len(bundles), len(tpls))
	}

	bundleID, bundle := anyBundle(bundles)
	if _, err := bi.SetBundleEnabled(ctx, bundleID, !bundle.IsEnabled); err != nil {
		t.Fatalf("SetBundleEnabled: %v", err)
	}

	flag, ok, err := bi.bundleOverlayFlags.GetFlag(ctx, builtInBundleID(bundleID))
	if err != nil {
		t.Fatalf("bundleFlags.GetFlag: %v", err)
	}
	if !ok || flag.Value == bundle.IsEnabled {
		t.Fatalf(
			"bundle overlay not updated: present=%v, value=%v, want present=true, value=%v",
			ok,
			flag.Value,
			!bundle.IsEnabled,
		)
	}

	_, tmplID, tpl := anyTemplate(tpls)
	if _, err := bi.SetTemplateEnabled(ctx, bundleID, tpl.Slug, tpl.Version, !tpl.IsEnabled); err != nil {
		t.Fatalf("SetTemplateEnabled: %v", err)
	}

	tflag, ok, err := bi.templateOverlayFlags.GetFlag(ctx, getTemplateKey(bundleID, tmplID))
	if err != nil {
		t.Fatalf("templateFlags.GetFlag: %v", err)
	}
	if !ok || tflag.Value == tpl.IsEnabled {
		t.Fatalf(
			"template overlay not updated: present=%v, value=%v, want present=true, value=%v",
			ok,
			tflag.Value,
			!tpl.IsEnabled,
		)
	}
}

func anyTemplate(
	m map[bundleitemutils.BundleID]map[bundleitemutils.ItemID]spec.PromptTemplate,
) (bundleitemutils.BundleID, bundleitemutils.ItemID, spec.PromptTemplate) {
	for bid, tm := range m {
		for tid, tpl := range tm {
			return bid, tid, tpl
		}
	}
	return "", "", spec.PromptTemplate{}
}

func anyBundle(
	m map[bundleitemutils.BundleID]spec.PromptBundle,
) (bundleitemutils.BundleID, spec.PromptBundle) {
	for id, b := range m {
		return id, b
	}
	return "", spec.PromptBundle{}
}

// buildManifest returns the JSON for a manifest that holds exactly one bundle.
func buildManifest(bundleID, slug string) []byte {
	manifest := spec.AllBundles{
		Bundles: map[bundleitemutils.BundleID]spec.PromptBundle{
			bundleitemutils.BundleID(bundleID): {
				ID:        bundleitemutils.BundleID(bundleID),
				Slug:      bundleitemutils.BundleSlug(slug),
				IsEnabled: true,
			},
		},
	}
	b, _ := json.Marshal(manifest)
	return b
}

// buildTemplate returns filename (slug_version.json), raw JSON and the template ID.
func buildTemplate(t *testing.T, slug, ver string) (fileName string, raw []byte, tplID string) {
	t.Helper()
	tplID = newUUID(t)
	tpl := spec.PromptTemplate{
		SchemaVersion: spec.SchemaVersion,
		ID:            bundleitemutils.ItemID(tplID),
		Slug:          bundleitemutils.ItemSlug(slug),
		Version:       bundleitemutils.ItemVersion(ver),
		DisplayName:   slug,
		Blocks: []spec.MessageBlock{
			{
				ID:      spec.MessageBlockID("a"),
				Role:    spec.System,
				Content: "hey",
			},
		},
		IsEnabled:  true,
		CreatedAt:  time.Now(),
		ModifiedAt: time.Now(),
	}
	raw, _ = json.Marshal(tpl) // cannot fail
	fileName = fmt.Sprintf("%s_%s.json", slug, ver)
	return fileName, raw, tplID
}

// Constructor helper that injects the given fs.FS.
func newFromFS(t *testing.T, mem fs.FS) (*BuiltInData, error) {
	t.Helper()
	ctx := t.Context()
	tmp := t.TempDir()
	bi, err := NewBuiltInData(ctx, tmp, time.Hour, WithBundlesFS(mem, "."))
	// If the constructor returned an error but nevertheless created/opened
	// internal resources (for example the overlay sqlite DB), close them right
	// away instead of relying only on t.Cleanup. On Windows an open sqlite file
	// prevents the temporary directory from being removed.
	if err != nil {
		if bi != nil {
			_ = bi.Close()
		}
		return nil, err
	}
	t.Cleanup(func() { _ = bi.Close() })
	return bi, nil
}

// newUUID returns a v7-UUID as string or fails the test.
func newUUID(t *testing.T) string {
	t.Helper()
	u, err := uuidv7filename.NewUUIDv7String()
	if err != nil {
		t.Fatalf("uuidv7: %v", err)
	}
	return u
}
