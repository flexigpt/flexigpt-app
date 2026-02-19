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
	"github.com/flexigpt/flexigpt-app/internal/tool/spec"
	"github.com/ppipada/mapstore-go/uuidv7filename"
)

const (
	corrupted = "corrupted"
	demo      = "demo"
)

func TestNewBuiltInToolData(t *testing.T) {
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

			bi, err := NewBuiltInToolData(ctx, dir, tt.snapshotMaxAge, WithLLMToolsGoBuiltins(true))
			t.Cleanup(func() {
				_ = bi.Close()
			})

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
			bundles, tools, _ := bi.ListBuiltInToolData(ctx)
			if len(bundles) == 0 {
				t.Error("expected at least one bundle")
			}
			if len(tools) == 0 {
				t.Error("expected at least one tool")
			}

			// Verify overlay file creation.
			if _, err := os.Stat(filepath.Join(dir, spec.ToolBuiltInOverlayDBFileName)); err != nil {
				t.Errorf("overlay file not created: %v", err)
			}

			// Verify all bundles are marked as built-in.
			for id, bundle := range bundles {
				if !bundle.IsBuiltIn {
					t.Errorf("bundle %s should be marked as built-in", id)
				}
			}

			// Verify all tools are marked as built-in.
			for bid, tmap := range tools {
				for tid, tool := range tmap {
					if !tool.IsBuiltIn {
						t.Errorf("tool %s/%s should be marked as built-in", bid, tid)
					}
				}
			}
		})
	}
}

func TestSetToolBundleEnabled(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, bi *BuiltInToolData) (bundleitemutils.BundleID, bool)
		wantErr bool
	}{
		{
			name: "enable_existing_bundle",
			setup: func(t *testing.T, bi *BuiltInToolData) (bundleitemutils.BundleID, bool) {
				t.Helper()
				bundles, _, _ := bi.ListBuiltInToolData(t.Context())
				id, bundle := anyToolBundle(bundles)
				return id, !bundle.IsEnabled
			},
			wantErr: false,
		},
		{
			name: "disable_existing_bundle",
			setup: func(t *testing.T, bi *BuiltInToolData) (bundleitemutils.BundleID, bool) {
				t.Helper()
				bundles, _, _ := bi.ListBuiltInToolData(t.Context())
				id, bundle := anyToolBundle(bundles)
				return id, !bundle.IsEnabled
			},
			wantErr: false,
		},
		{
			name: "nonexistent_bundle",
			setup: func(t *testing.T, bi *BuiltInToolData) (bundleitemutils.BundleID, bool) {
				t.Helper()
				return bundleitemutils.BundleID("does-not-exist"), true
			},
			wantErr: true,
		},
		{
			name: "empty_bundle_id",
			setup: func(t *testing.T, bi *BuiltInToolData) (bundleitemutils.BundleID, bool) {
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
			bi, err := NewBuiltInToolData(ctx, dir, 0, WithLLMToolsGoBuiltins(true))
			if err != nil {
				t.Fatalf("setup failed: %v", err)
			}
			t.Cleanup(func() {
				_ = bi.Close()
			})

			bundleID, enabled := tt.setup(t, bi)

			_, err = bi.SetToolBundleEnabled(ctx, bundleID, enabled)

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
			bundles, _, _ := bi.ListBuiltInToolData(ctx)
			if got := bundles[bundleID].IsEnabled; got != enabled {
				t.Errorf("bundle enabled state = %v, want %v", got, enabled)
			}

			// Verify overlay state via bundleOverlayFlags API.
			flag, ok, err := bi.bundleOverlayFlags.GetFlag(ctx, builtInToolBundleID(bundleID))
			if err != nil {
				t.Fatalf("bundleOverlayFlags.GetFlag: %v", err)
			}
			if !ok || flag.Value != enabled {
				t.Errorf(
					"overlay: present=%v val=%v, want present=true val=%v",
					ok,
					flag.Value,
					enabled,
				)
			}
		})
	}
}

func TestSetToolEnabled(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, bi *BuiltInToolData) (bundleitemutils.BundleID, spec.Tool, bool)
		wantErr bool
	}{
		{
			name: "enable_existing_tool",
			setup: func(t *testing.T, bi *BuiltInToolData) (bundleitemutils.BundleID, spec.Tool, bool) {
				t.Helper()
				_, tools, _ := bi.ListBuiltInToolData(t.Context())
				bid, _, tool := anyTool(tools)
				return bid, tool, !tool.IsEnabled
			},
			wantErr: false,
		},
		{
			name: "disable_existing_tool",
			setup: func(t *testing.T, bi *BuiltInToolData) (bundleitemutils.BundleID, spec.Tool, bool) {
				t.Helper()
				_, tools, _ := bi.ListBuiltInToolData(t.Context())
				bid, _, tool := anyTool(tools)
				return bid, tool, !tool.IsEnabled
			},
			wantErr: false,
		},
		{
			name: "nonexistent_bundle",
			setup: func(t *testing.T, bi *BuiltInToolData) (bundleitemutils.BundleID, spec.Tool, bool) {
				t.Helper()
				return bundleitemutils.BundleID(
					"does-not-exist",
				), buildValidTool(t, "tool", "a", "tool"), true
			},
			wantErr: true,
		},
		{
			name: "nonexistent_tool",
			setup: func(t *testing.T, bi *BuiltInToolData) (bundleitemutils.BundleID, spec.Tool, bool) {
				t.Helper()
				bundles, _, _ := bi.ListBuiltInToolData(t.Context())
				bid, _ := anyToolBundle(bundles)
				return bid, buildValidTool(t, "does-not-exist", "a", "does-not-exist"), true
			},
			wantErr: true,
		},
		{
			name: "empty_tool_id",
			setup: func(t *testing.T, bi *BuiltInToolData) (bundleitemutils.BundleID, spec.Tool, bool) {
				t.Helper()
				bundles, _, _ := bi.ListBuiltInToolData(t.Context())
				bid, _ := anyToolBundle(bundles)
				return bid, buildValidTool(t, "", "a", ""), true
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			dir := t.TempDir()
			bi, err := NewBuiltInToolData(ctx, dir, 0, WithLLMToolsGoBuiltins(true))
			if err != nil {
				t.Fatalf("setup failed: %v", err)
			}
			t.Cleanup(func() {
				_ = bi.Close()
			})

			bundleID, tool, enabled := tt.setup(t, bi)

			_, err = bi.SetToolEnabled(ctx, bundleID, tool.Slug, tool.Version, enabled)

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
			_, tools, _ := bi.ListBuiltInToolData(ctx)
			if got := tools[bundleID][tool.ID].IsEnabled; got != enabled {
				t.Errorf("tool enabled state = %v, want %v", got, enabled)
			}

			// Verify overlay state via toolOverlayFlags API.
			flag, ok, err := bi.toolOverlayFlags.GetFlag(ctx, getToolKey(bundleID, tool.ID))
			if err != nil {
				t.Fatalf("toolOverlayFlags.GetFlag: %v", err)
			}
			if !ok || flag.Value != enabled {
				t.Errorf(
					"overlay: present=%v val=%v, want present=true val=%v",
					ok,
					flag.Value,
					enabled,
				)
			}
		})
	}
}

func TestListBuiltInToolData(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	bi, err := NewBuiltInToolData(ctx, dir, 0, WithLLMToolsGoBuiltins(true))
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	t.Cleanup(func() {
		_ = bi.Close()
	})

	t.Run("returns_independent_copies", func(t *testing.T) {
		b1, t1, _ := bi.ListBuiltInToolData(ctx)
		b2, t2, _ := bi.ListBuiltInToolData(ctx)

		// Mutate first copy.
		for id := range b1 {
			b := b1[id]
			b.Slug = corrupted
			b1[id] = b
			break
		}
		for bid := range t1 {
			for tid := range t1[bid] {
				tool := t1[bid][tid]
				tool.Slug = corrupted
				t1[bid][tid] = tool
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
			for tid, tool := range tm {
				if tool.Slug == corrupted {
					t.Errorf("tool %s/%s shares memory with first copy", bid, tid)
				}
			}
		}
	})

	t.Run("returns_consistent_data", func(t *testing.T) {
		bundles, tools, _ := bi.ListBuiltInToolData(ctx)

		// Verify all bundles have corresponding tool maps.
		for bundleID := range bundles {
			if _, exists := tools[bundleID]; !exists {
				t.Errorf("bundle %s has no corresponding tool map", bundleID)
			}
		}

		// Verify all tool maps have corresponding bundles.
		for bundleID := range tools {
			if _, exists := bundles[bundleID]; !exists {
				t.Errorf("tool map for bundle %s has no corresponding bundle", bundleID)
			}
		}
	})

	t.Run("all_items_marked_builtin", func(t *testing.T) {
		bundles, tools, _ := bi.ListBuiltInToolData(ctx)

		for id, bundle := range bundles {
			if !bundle.IsBuiltIn {
				t.Errorf("bundle %s not marked as built-in", id)
			}
		}

		for bid, tmap := range tools {
			for tid, tool := range tmap {
				if !tool.IsBuiltIn {
					t.Errorf("tool %s/%s not marked as built-in", bid, tid)
				}
			}
		}
	})
}

func TestToolConcurrency(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	bi, err := NewBuiltInToolData(ctx, dir, time.Millisecond*10, WithLLMToolsGoBuiltins(true))
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	t.Cleanup(func() {
		_ = bi.Close()
	})

	bundles, tools, _ := bi.ListBuiltInToolData(ctx)
	if len(bundles) == 0 || len(tools) == 0 {
		t.Skip("no test data available")
	}

	bundleID, _ := anyToolBundle(bundles)
	toolBundleID, _, tool := anyTool(tools)

	t.Run("concurrent_reads", func(t *testing.T) {
		var wg sync.WaitGroup
		const numReaders = 10

		for range numReaders {
			wg.Go(func() {
				for range 100 {
					bundles, tools, _ := bi.ListBuiltInToolData(ctx)
					if len(bundles) == 0 || len(tools) == 0 {
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
				if _, err := bi.SetToolBundleEnabled(ctx, bundleID, enabled); err != nil {
					t.Errorf("SetToolBundleEnabled failed: %v", err)
				}
			}(i)

			go func(i int) {
				defer wg.Done()
				enabled := i%2 == 0
				if _, err := bi.SetToolEnabled(ctx, toolBundleID, tool.Slug, tool.Version, enabled); err != nil {
					t.Errorf("SetToolEnabled failed: %v", err)
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
						_, _, _ = bi.ListBuiltInToolData(ctx)
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
					if _, e := bi.SetToolBundleEnabled(ctx, bundleID, enabled); e != nil {
						t.Errorf("SetToolBundleEnabled failed: %v", e)
					}
					if _, e := bi.SetToolEnabled(ctx, toolBundleID, tool.Slug, tool.Version, enabled); e != nil {
						t.Errorf("SetToolEnabled failed: %v", e)
					}
				}
			}(i)
		}

		time.Sleep(100 * time.Millisecond)
		close(done)
		wg.Wait()
	})
}

func TestRebuildToolSnapshot(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	bi, err := NewBuiltInToolData(ctx, dir, time.Hour, WithLLMToolsGoBuiltins(true))
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	t.Cleanup(func() {
		_ = bi.Close()
	})

	bundles, _, _ := bi.ListBuiltInToolData(ctx)
	bundleID, bundle := anyToolBundle(bundles)

	// Change the overlay directly.
	_, err = bi.bundleOverlayFlags.SetFlag(ctx, builtInToolBundleID(bundleID), !bundle.IsEnabled)
	if err != nil {
		t.Fatalf("failed to set overlay: %v", err)
	}

	// Force rebuild.
	if err := bi.rebuildSnapshot(ctx); err != nil {
		t.Fatalf("rebuildSnapshot failed: %v", err)
	}

	// Verify the change is reflected.
	newBundles, _, _ := bi.ListBuiltInToolData(ctx)
	if newBundles[bundleID].IsEnabled == bundle.IsEnabled {
		t.Error("rebuild did not apply overlay changes")
	}
}

func TestAsyncToolRebuild(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	bi, err := NewBuiltInToolData(ctx, dir, time.Millisecond, WithLLMToolsGoBuiltins(true))
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	t.Cleanup(func() {
		_ = bi.Close()
	})

	bundles, _, _ := bi.ListBuiltInToolData(ctx)
	bundleID, bundle := anyToolBundle(bundles)

	time.Sleep(2 * time.Millisecond)

	if _, err := bi.SetToolBundleEnabled(ctx, bundleID, !bundle.IsEnabled); err != nil {
		t.Fatalf("SetToolBundleEnabled failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	newBundles, _, _ := bi.ListBuiltInToolData(ctx)
	if newBundles[bundleID].IsEnabled == bundle.IsEnabled {
		t.Error("async rebuild did not apply changes")
	}
}

func Test_NewBuiltInToolData_SyntheticFS_MissingBundlesJSON(t *testing.T) {
	_, err := newToolFromFS(t, fstest.MapFS{})
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("want fs.ErrNotExist, got %v", err)
	}
}

// Test when the tool bundles json file is present but contains invalid JSON.
func Test_NewBuiltInToolData_SyntheticFS_InvalidBundlesJSON(t *testing.T) {
	fsys := fstest.MapFS{
		builtin.BuiltInToolBundlesJSON: {Data: []byte("{ oops ]")},
	}
	_, err := newToolFromFS(t, fsys)
	if err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Test when the manifest contains no bundles.
func Test_NewBuiltInToolData_SyntheticFS_NoBundlesInManifest(t *testing.T) {
	empty, _ := json.Marshal(
		spec.AllBundles{Bundles: map[bundleitemutils.BundleID]spec.ToolBundle{}},
	)
	fsys := fstest.MapFS{builtin.BuiltInToolBundlesJSON: {Data: empty}}
	_, err := newToolFromFS(t, fsys)
	if err == nil || !strings.Contains(err.Error(), "contains no bundles") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Test when a bundle directory is present but the bundle has no tools.
func Test_NewBuiltInToolData_SyntheticFS_BundleHasNoTools(t *testing.T) {
	bundleID := newUUID(t)
	slug := demo

	dir := fmt.Sprintf("%s_%s", bundleID, slug)
	fsys := fstest.MapFS{
		builtin.BuiltInToolBundlesJSON: {Data: buildToolManifest(bundleID, slug)},
		dir:                            &fstest.MapFile{Mode: fs.ModeDir},
	}
	_, err := newToolFromFS(t, fsys)
	if err == nil || !strings.Contains(err.Error(), "has no tools") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Test when a bundle directory exists that is not listed in the manifest.
func Test_NewBuiltInToolData_SyntheticFS_BundleDirNotInManifest(t *testing.T) {
	bundleID := newUUID(t)
	slug := demo

	ghostID := newUUID(t)
	ghostSlug := "ghost"
	dir := fmt.Sprintf("%s_%s", ghostID, ghostSlug)

	fn, raw, _ := buildTool(t, ghostSlug, "v1")
	fsys := fstest.MapFS{
		builtin.BuiltInToolBundlesJSON: {Data: buildToolManifest(bundleID, slug)},
		dir:                            &fstest.MapFile{Mode: fs.ModeDir},
		dir + "/" + fn:                 {Data: raw},
	}
	_, err := newToolFromFS(t, fsys)
	if err == nil ||
		!strings.Contains(err.Error(), "not in "+builtin.BuiltInToolBundlesJSON) {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Test when the directory slug doesn't match the manifest slug for that bundle.
func Test_NewBuiltInToolData_SyntheticFS_SlugMismatchBetweenDirAndManifest(t *testing.T) {
	bundleID := newUUID(t)
	slug := demo

	// Dir slug = wrong, manifest slug = demo.
	dir := fmt.Sprintf("%s_%s", bundleID, "wrong")

	fn, raw, _ := buildTool(t, "wrong", "v1")
	fsys := fstest.MapFS{
		builtin.BuiltInToolBundlesJSON: {Data: buildToolManifest(bundleID, slug)},
		dir:                            &fstest.MapFile{Mode: fs.ModeDir},
		dir + "/" + fn:                 {Data: raw},
	}
	_, err := newToolFromFS(t, fsys)
	if err == nil || !strings.Contains(err.Error(), "dir slug") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Test when the filename implies a different slug/version than the JSON content.
func Test_NewBuiltInToolData_SyntheticFS_FilenameSlugVersionMismatch(t *testing.T) {
	bundleID := newUUID(t)
	slug := demo

	dir := fmt.Sprintf("%s_%s", bundleID, slug)

	_, raw, _ := buildTool(t, slug, "v1")
	fsys := fstest.MapFS{
		builtin.BuiltInToolBundlesJSON: {Data: buildToolManifest(bundleID, slug)},
		dir:                            &fstest.MapFile{Mode: fs.ModeDir},
		dir + "/demo_v2.json":          {Data: raw},
	}
	_, err := newToolFromFS(t, fsys)
	if err == nil || !strings.Contains(err.Error(), "filename") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Test when two tools in the same bundle share the same tool ID.
func Test_NewBuiltInToolData_SyntheticFS_DuplicateToolID(t *testing.T) {
	bundleID := newUUID(t)
	slug := demo

	dir := fmt.Sprintf("%s_%s", bundleID, slug)
	fn, raw, toolID := buildTool(t, slug, "v1")
	fn2, raw2, _ := buildToolWithID(t, slug, "v2", toolID)
	fsys := fstest.MapFS{
		builtin.BuiltInToolBundlesJSON: {Data: buildToolManifest(bundleID, slug)},
		dir:                            &fstest.MapFile{Mode: fs.ModeDir},
		dir + "/" + fn:                 {Data: raw},
		dir + "/" + fn2:                {Data: raw2},
	}
	_, err := newToolFromFS(t, fsys)
	if err == nil || !strings.Contains(err.Error(), "duplicate tool ID") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func Test_NewBuiltInToolData_SyntheticFS_HappyAndCRUD(t *testing.T) {
	ctx := t.Context()
	bid := newUUID(t)
	slug := demo

	dir := fmt.Sprintf("%s_%s", bid, slug)
	fn, rawTool, _ := buildTool(t, slug, "v1")

	mem := fstest.MapFS{
		builtin.BuiltInToolBundlesJSON: {Data: buildToolManifest(bid, slug)},
		dir:                            &fstest.MapFile{Mode: fs.ModeDir},
		dir + "/" + fn:                 {Data: rawTool},
	}

	tmp := t.TempDir()
	bi, err := NewBuiltInToolData(ctx, tmp, 0, WithToolBundlesFS(mem, "."))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Cleanup(func() {
		_ = bi.Close()
	})

	bundles, tools, err := bi.ListBuiltInToolData(ctx)
	if err != nil || len(bundles) != 1 || len(tools) != 1 {
		t.Fatalf("want 1/1 objects, got %d/%d", len(bundles), len(tools))
	}

	bundleID, bundle := anyToolBundle(bundles)
	if _, err := bi.SetToolBundleEnabled(ctx, bundleID, !bundle.IsEnabled); err != nil {
		t.Fatalf("SetToolBundleEnabled: %v", err)
	}
	flag, ok, err := bi.bundleOverlayFlags.GetFlag(ctx, builtInToolBundleID(bundleID))
	if err != nil {
		t.Fatalf("bundleOverlayFlags.GetFlag: %v", err)
	}
	if !ok || flag.Value == bundle.IsEnabled {
		t.Fatalf("bundle overlay not updated")
	}

	_, toolID, tool := anyTool(tools)
	if _, err := bi.SetToolEnabled(ctx, bundleID, tool.Slug, tool.Version, !tool.IsEnabled); err != nil {
		t.Fatalf("SetToolEnabled: %v", err)
	}
	tflag, ok, err := bi.toolOverlayFlags.GetFlag(ctx, getToolKey(bundleID, toolID))
	if err != nil {
		t.Fatalf("toolOverlayFlags.GetFlag: %v", err)
	}
	if !ok || tflag.Value == tool.IsEnabled {
		t.Fatalf("tool overlay not updated")
	}
}

// anyTool returns any tool from the map.
func anyTool(
	m map[bundleitemutils.BundleID]map[bundleitemutils.ItemID]spec.Tool,
) (bundleitemutils.BundleID, bundleitemutils.ItemID, spec.Tool) {
	for bid, tm := range m {
		for tid, tool := range tm {
			return bid, tid, tool
		}
	}
	return "", "", spec.Tool{}
}

// anyToolBundle returns any bundle from the map.
func anyToolBundle(
	m map[bundleitemutils.BundleID]spec.ToolBundle,
) (bundleitemutils.BundleID, spec.ToolBundle) {
	for id, b := range m {
		return id, b
	}
	return "", spec.ToolBundle{}
}

// buildToolManifest returns the JSON for a manifest that holds exactly one bundle.
func buildToolManifest(bundleID, slug string) []byte {
	now := time.Now().UTC()
	manifest := spec.AllBundles{
		Bundles: map[bundleitemutils.BundleID]spec.ToolBundle{
			bundleitemutils.BundleID(bundleID): {
				SchemaVersion: spec.SchemaVersion,
				ID:            bundleitemutils.BundleID(bundleID),
				Slug:          bundleitemutils.BundleSlug(slug),
				DisplayName:   slug,
				Description:   "desc",
				IsEnabled:     true,
				IsBuiltIn:     true,
				CreatedAt:     now,
				ModifiedAt:    now,
			},
		},
	}
	b, _ := json.Marshal(manifest)
	return b
}

// buildTool returns filename (slug_version.json), raw JSON and the tool ID.
func buildTool(t *testing.T, slug, ver string) (fileName string, raw []byte, toolID string) {
	t.Helper()
	toolID = newUUID(t)
	tool := buildValidTool(t, slug, ver, toolID)
	raw, _ = json.Marshal(tool)
	fileName = fmt.Sprintf("%s_%s.json", slug, ver)
	return fileName, raw, toolID
}

// buildToolWithID returns a tool with a specific ID.
func buildToolWithID(
	t *testing.T,
	slug, ver, id string,
) (fileName string, raw []byte, toolID string) {
	t.Helper()
	toolID = id
	tool := buildValidTool(t, slug, ver, toolID)
	raw, _ = json.Marshal(tool)
	fileName = fmt.Sprintf("%s_%s.json", slug, ver)
	return fileName, raw, toolID
}

// buildValidTool returns a fully valid spec.Tool for the test suite.
func buildValidTool(t *testing.T, slug, ver, id string) spec.Tool {
	t.Helper()
	now := time.Now().UTC()
	return spec.Tool{
		SchemaVersion: spec.SchemaVersion,
		ID:            bundleitemutils.ItemID(id),
		Slug:          bundleitemutils.ItemSlug(slug),
		Version:       bundleitemutils.ItemVersion(ver),
		DisplayName:   slug,
		Description:   "desc",
		UserCallable:  true,
		LLMCallable:   true,
		Tags:          []string{"tag1", "tag2"},
		ArgSchema:     json.RawMessage(`{"type":"object"}`),
		LLMToolType:   spec.ToolStoreChoiceTypeFunction,
		Type:          spec.ToolTypeGo,
		GoImpl:        &spec.GoToolImpl{Func: "github.com/acme/flexigpt/tools.Demo"},
		HTTPImpl:      nil,
		IsEnabled:     true,
		IsBuiltIn:     true,
		CreatedAt:     now,
		ModifiedAt:    now,
	}
}

// newToolFromFS constructs BuiltInToolData from a synthetic fs.FS.
func newToolFromFS(t *testing.T, mem fs.FS) (*BuiltInToolData, error) {
	t.Helper()
	ctx := t.Context()
	tmp := t.TempDir()
	bi, err := NewBuiltInToolData(ctx, tmp, time.Hour, WithToolBundlesFS(mem, "."))

	// If NewBuiltInToolData returned a non-nil instance together with an error,
	// close it immediately so any OS file handles (e.g. sqlite file) are
	// released before the temp dir cleanup runs (Windows cannot remove open
	// files). If construction succeeded, register a cleanup to close it at
	// test end.
	if bi != nil {
		if err != nil {
			_ = bi.Close()
			if runtime.GOOS == "windows" {
				// Give SQLite time to release handles on Windows.
				t.Log("toolstore: sleeping in win")
				time.Sleep(time.Millisecond * 100)
			}
			return nil, err
		}
		t.Cleanup(func() { _ = bi.Close() })
	}
	return bi, err
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
