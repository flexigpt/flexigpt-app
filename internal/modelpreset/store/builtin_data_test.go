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
	"github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
	inferencegoSpec "github.com/flexigpt/inference-go/spec"
)

const (
	corrupted = "corrupted"
	windows   = "windows"
)

func TestNewBuiltInPresets(t *testing.T) {
	tests := []struct {
		name           string
		setupDir       func(*testing.T) string
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
		},
		{
			name: "zero_snapshot_age_defaults",
			setupDir: func(t *testing.T) string {
				t.Helper()
				return t.TempDir()
			},
			snapshotMaxAge: 0,
		},
		{
			name: "negative_snapshot_age_defaults",
			setupDir: func(t *testing.T) string {
				t.Helper()
				return t.TempDir()
			},
			snapshotMaxAge: -1,
		},
		{
			name: "base_dir_is_file",
			setupDir: func(t *testing.T) string {
				t.Helper()
				tmp := t.TempDir()
				f := filepath.Join(tmp, "file")
				_ = os.WriteFile(f, []byte("dummy"), 0o600)
				return f
			},
			wantErr: true,
		},
		{
			name: "readonly_directory",
			setupDir: func(t *testing.T) string {
				t.Helper()
				tmp := t.TempDir()
				ro := filepath.Join(tmp, "ro")
				_ = os.MkdirAll(ro, 0o444)
				return ro
			},
			// Unix-like systems will typically fail to create files in a 0444
			// directory; Windows does not honor these POSIX permission bits the
			// same way, so don't require an error there.
			wantErr: runtime.GOOS != windows,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			dir := tc.setupDir(t)
			bi, err := NewBuiltInPresets(ctx, dir, tc.snapshotMaxAge)
			// Always ensure any created instance is closed at test cleanup.
			// This guards against the case where NewBuiltInPresets unexpectedly
			// succeeds for a case that planned to fail (which would otherwise
			// leave resources open and cause TempDir removal to fail on Windows).
			t.Cleanup(func() {
				if bi != nil {
					_ = bi.Close()
				}
			})

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
					t.Fatalf("error %q does not contain %q", err, tc.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			prov, models, _ := bi.ListBuiltInPresets(ctx)
			if len(prov) == 0 || len(models) == 0 {
				t.Fatal("expected non-empty data")
			}
			if _, err := os.Stat(filepath.Join(dir, spec.ModelPresetsBuiltInOverlayDBFileName)); err != nil {
				t.Fatalf("overlay file missing: %v", err)
			}
			for n, p := range prov {
				if !p.IsBuiltIn {
					t.Errorf("provider %s not flagged built-in", n)
				}
			}
			for pn, mm := range models {
				for mid, mp := range mm {
					if !mp.IsBuiltIn {
						t.Errorf("model %s/%s not flagged built-in", pn, mid)
					}
				}
			}
		})
	}
}

func TestSetProviderEnabled(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*testing.T, *BuiltInPresets) (inferencegoSpec.ProviderName, bool)
		wantErr bool
	}{
		{
			name: "toggle_existing_provider",
			setup: func(t *testing.T, bi *BuiltInPresets) (inferencegoSpec.ProviderName, bool) {
				t.Helper()
				prov, _, _ := bi.ListBuiltInPresets(t.Context())
				id, p := anyProvider(prov)
				return id, !p.IsEnabled
			},
		},
		{
			name: "nonexistent_provider",
			setup: func(t *testing.T, _ *BuiltInPresets) (inferencegoSpec.ProviderName, bool) {
				t.Helper()
				return inferencegoSpec.ProviderName("ghost"), true
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			dir := t.TempDir()
			bi, _ := NewBuiltInPresets(ctx, dir, 0)
			t.Cleanup(func() {
				_ = bi.Close()
			})
			pname, enabled := tc.setup(t, bi)
			_, err := bi.SetProviderEnabled(ctx, pname, enabled)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected: %v", err)
			}
			prov, _, _ := bi.ListBuiltInPresets(ctx)
			if prov[pname].IsEnabled != enabled {
				t.Errorf("flag mismatch want %v", enabled)
			}
			// Check overlay state via providerOverlayFlags API.
			flag, ok, err := bi.providerOverlayFlags.GetFlag(ctx, builtInProviderKey(pname))
			if err != nil {
				t.Fatalf("providerOverlayFlags.GetFlag: %v", err)
			}
			if !ok || flag.Value != enabled {
				t.Errorf(
					"overlay mismatch: present=%v, value=%v, want present=true, value=%v",
					ok,
					flag.Value,
					enabled,
				)
			}
		})
	}
}

func TestSetModelPresetEnabled(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*testing.T, *BuiltInPresets) (inferencegoSpec.ProviderName, spec.ModelPreset, bool)
		wantErr bool
	}{
		{
			name: "toggle_existing_model",
			setup: func(t *testing.T, bi *BuiltInPresets) (inferencegoSpec.ProviderName, spec.ModelPreset, bool) {
				t.Helper()
				_, models, _ := bi.ListBuiltInPresets(t.Context())
				pn, _, mp := anyModel(models)
				return pn, mp, !mp.IsEnabled
			},
		},
		{
			name: "nonexistent_provider",
			setup: func(*testing.T, *BuiltInPresets) (inferencegoSpec.ProviderName, spec.ModelPreset, bool) {
				return inferencegoSpec.ProviderName("ghost"), spec.ModelPreset{ID: "m"}, true
			},
			wantErr: true,
		},
		{
			name: "nonexistent_model",
			setup: func(t *testing.T, bi *BuiltInPresets) (inferencegoSpec.ProviderName, spec.ModelPreset, bool) {
				t.Helper()
				prov, _, _ := bi.ListBuiltInPresets(t.Context())
				pn, _ := anyProvider(prov)
				return pn, spec.ModelPreset{ID: "ghost"}, true
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			dir := t.TempDir()
			bi, _ := NewBuiltInPresets(ctx, dir, 0)
			t.Cleanup(func() {
				_ = bi.Close()
			})
			pn, mp, enabled := tc.setup(t, bi)
			_, err := bi.SetModelPresetEnabled(ctx, pn, mp.ID, enabled)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected: %v", err)
			}
			_, models, _ := bi.ListBuiltInPresets(ctx)
			if models[pn][mp.ID].IsEnabled != enabled {
				t.Errorf("flag mismatch want %v", enabled)
			}
			// Check overlay state via modelOverlayFlags API.
			flag, ok, err := bi.modelOverlayFlags.GetFlag(ctx, getModelKey(pn, mp.ID))
			if err != nil {
				t.Fatalf("modelOverlayFlags.GetFlag: %v", err)
			}
			if !ok || flag.Value != enabled {
				t.Errorf(
					"overlay mismatch: present=%v, value=%v, want present=true, value=%v",
					ok,
					flag.Value,
					enabled,
				)
			}
		})
	}
}

func TestListBuiltInPresets(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	bi, _ := NewBuiltInPresets(ctx, dir, 0)
	t.Cleanup(func() {
		_ = bi.Close()
	})

	t.Run("independent_copies", func(t *testing.T) {
		p1, m1, _ := bi.ListBuiltInPresets(ctx)
		p2, m2, _ := bi.ListBuiltInPresets(ctx)

		for n := range p1 {
			p1[n] = spec.ProviderPreset{DisplayName: corrupted}
			break
		}
		for pn := range m1 {
			for mid := range m1[pn] {
				mp := m1[pn][mid]
				mp.DisplayName = corrupted
				m1[pn][mid] = mp
				break
			}
			break
		}
		for n, p := range p2 {
			if p.DisplayName == corrupted {
				t.Errorf("provider %s shares memory", n)
			}
		}
		for pn, mm := range m2 {
			for mid, mp := range mm {
				if mp.DisplayName == corrupted {
					t.Errorf("model %s/%s shares memory", pn, mid)
				}
			}
		}
	})

	t.Run("consistent_maps", func(t *testing.T) {
		prov, models, _ := bi.ListBuiltInPresets(ctx)
		for pn := range prov {
			if _, ok := models[pn]; !ok {
				t.Errorf("provider %s has no model map", pn)
			}
		}
		for pn := range models {
			if _, ok := prov[pn]; !ok {
				t.Errorf("model map %s has no provider", pn)
			}
		}
	})
}

func TestConcurrencyPresets(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	bi, _ := NewBuiltInPresets(ctx, dir, 10*time.Millisecond)
	t.Cleanup(func() {
		_ = bi.Close()
	})

	// Fetch sample keys.
	prov, models, _ := bi.ListBuiltInPresets(ctx)
	pname, _ := anyProvider(prov)
	pn, mid, _ := anyModel(models)

	// Concurrent reads.
	t.Run("reads", func(t *testing.T) {
		var wg sync.WaitGroup
		const loops = 10
		for range loops {
			wg.Go(func() {
				for range 100 {
					_, _, _ = bi.ListBuiltInPresets(ctx)
				}
			})
		}
		wg.Wait()
	})

	// Concurrent writes.
	t.Run("writes", func(t *testing.T) {
		var wg sync.WaitGroup
		const writers = 5
		for i := range writers {
			wg.Add(2)
			go func(i int) {
				defer wg.Done()
				_, _ = bi.SetProviderEnabled(ctx, pname, i%2 == 0)
			}(i)
			go func(i int) {
				defer wg.Done()
				_, _ = bi.SetModelPresetEnabled(ctx, pn, mid, i%2 == 0)
			}(i)
		}
		wg.Wait()
	})
}

func TestRebuildSnapshotPresets(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	bi, _ := NewBuiltInPresets(ctx, dir, time.Hour)
	t.Cleanup(func() {
		_ = bi.Close()
	})

	prov, _, _ := bi.ListBuiltInPresets(ctx)
	pn, p := anyProvider(prov)
	_, err := bi.providerOverlayFlags.SetFlag(ctx, builtInProviderKey(pn), !p.IsEnabled)
	if err != nil {
		t.Fatalf("providerOverlayFlags.SetFlag: %v", err)
	}

	bi.mu.Lock()
	_ = bi.rebuildSnapshot(ctx)
	bi.mu.Unlock()

	prov2, _, _ := bi.ListBuiltInPresets(ctx)
	if prov2[pn].IsEnabled == p.IsEnabled {
		t.Error("rebuild did not apply overlay")
	}
}

func TestAsyncRebuildPresets(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	bi, _ := NewBuiltInPresets(ctx, dir, time.Millisecond)
	t.Cleanup(func() {
		_ = bi.Close()
	})

	prov, _, _ := bi.ListBuiltInPresets(ctx)
	pn, p := anyProvider(prov)
	time.Sleep(2 * time.Millisecond)
	_, _ = bi.SetProviderEnabled(ctx, pn, !p.IsEnabled)
	time.Sleep(10 * time.Millisecond)

	prov2, _, _ := bi.ListBuiltInPresets(ctx)
	if prov2[pn].IsEnabled == p.IsEnabled {
		t.Error("async rebuild missed change")
	}
}

func Test_NewBuiltInPresets_SyntheticFS_MissingJSON(t *testing.T) {
	if runtime.GOOS == windows {
		t.Skip("custom fs test has some overlay race in win")
		return
	}
	_, err := newPresetsFromFS(t, fstest.MapFS{})
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("want fs.ErrNotExist, got %v", err)
	}
}

// Test when the json file is present but contains invalid JSON.
func Test_NewBuiltInPresets_SyntheticFS_InvalidJSON(t *testing.T) {
	if runtime.GOOS == windows {
		t.Skip("custom fs test has some overlay race in win")
		return
	}
	fsys := fstest.MapFS{builtin.BuiltInModelPresetsJSON: {Data: []byte("{ nope ]")}}
	_, err := newPresetsFromFS(t, fsys)
	if err == nil {
		t.Fatal("want error")
	}
}

// Test when the schema has no providers defined.
func Test_NewBuiltInPresets_SyntheticFS_NoProviders(t *testing.T) {
	if runtime.GOOS == windows {
		t.Skip("custom fs test has some overlay race in win")
		return
	}
	empty, _ := json.Marshal(spec.PresetsSchema{
		SchemaVersion:   spec.SchemaVersion,
		ProviderPresets: map[inferencegoSpec.ProviderName]spec.ProviderPreset{},
	})
	fsys := fstest.MapFS{builtin.BuiltInModelPresetsJSON: {Data: empty}}
	_, err := newPresetsFromFS(t, fsys)
	if err == nil || (!strings.Contains(err.Error(), "no providers") &&
		!strings.Contains(err.Error(), "no default provider in builtin")) {
		t.Fatalf("unexpected: %v", err)
	}
}

// Test when a provider exists but its default model preset id is missing.
func Test_NewBuiltInPresets_SyntheticFS_DefaultModelMissing(t *testing.T) {
	if runtime.GOOS == windows {
		t.Skip("custom fs test has some overlay race in win")
		return
	}
	provName := inferencegoSpec.ProviderName("demoProv")
	modelID := spec.ModelPresetID("model1")

	s := buildSchemaDefaultMissing(provName, modelID)
	fsys := fstest.MapFS{builtin.BuiltInModelPresetsJSON: {Data: s}}
	_, err := newPresetsFromFS(t, fsys)
	if err == nil || !strings.Contains(err.Error(), "defaultModelPresetID") {
		t.Fatalf("unexpected: %v", err)
	}
}

func Test_NewBuiltInPresets_SyntheticFS_HappyAndCRUD(t *testing.T) {
	ctx := t.Context()
	pn := inferencegoSpec.ProviderName("demo")
	mpid := spec.ModelPresetID("m1")

	schema := buildHappySchema(pn, mpid)
	fsys := fstest.MapFS{builtin.BuiltInModelPresetsJSON: {Data: schema}}

	dir := t.TempDir()
	bi, err := NewBuiltInPresets(ctx, dir, 0, WithModelPresetsFS(fsys, "."))
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	t.Cleanup(func() {
		_ = bi.Close()
	})

	prov, models, _ := bi.ListBuiltInPresets(ctx)
	if len(prov) != 1 || len(models) != 1 {
		t.Fatalf("want 1/1 objects, got %d/%d", len(prov), len(models))
	}

	// Toggle provider.
	p := prov[pn]
	_, _ = bi.SetProviderEnabled(ctx, pn, !p.IsEnabled)
	flag, ok, err := bi.providerOverlayFlags.GetFlag(ctx, builtInProviderKey(pn))
	if err != nil {
		t.Fatalf("providerOverlayFlags.GetFlag: %v", err)
	}
	if !ok || flag.Value == p.IsEnabled {
		t.Fatal("provider overlay not updated")
	}

	// Toggle model preset.
	mp := models[pn][mpid]
	_, _ = bi.SetModelPresetEnabled(ctx, pn, mpid, !mp.IsEnabled)
	mflag, ok, err := bi.modelOverlayFlags.GetFlag(ctx, getModelKey(pn, mpid))
	if err != nil {
		t.Fatalf("modelOverlayFlags.GetFlag: %v", err)
	}
	if !ok || mflag.Value == mp.IsEnabled {
		t.Fatal("model overlay not updated")
	}
}

func TestSetDefaultModelPreset(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*testing.T, *BuiltInPresets) (inferencegoSpec.ProviderName, spec.ModelPresetID)
		wantErr bool
	}{
		{
			name: "change_existing_provider",
			setup: func(t *testing.T, bi *BuiltInPresets) (inferencegoSpec.ProviderName, spec.ModelPresetID) {
				t.Helper()
				_, models, _ := bi.ListBuiltInPresets(t.Context())
				for pn, mm := range models {
					if len(mm) < 2 {
						continue // need at least 2 models to change default
					}
					ids := make([]spec.ModelPresetID, 0, len(mm))
					for mid := range mm {
						ids = append(ids, mid)
					}
					return pn, ids[1] // pick a non-default one
				}
				t.Skip("test data has no provider with ≥2 models")
				return "", ""
			},
		},
		{
			name: "nonexistent_provider",
			setup: func(*testing.T, *BuiltInPresets) (inferencegoSpec.ProviderName, spec.ModelPresetID) {
				return inferencegoSpec.ProviderName("ghost"), spec.ModelPresetID("m1")
			},
			wantErr: true,
		},
		{
			name: "nonexistent_model",
			setup: func(t *testing.T, bi *BuiltInPresets) (inferencegoSpec.ProviderName, spec.ModelPresetID) {
				t.Helper()
				prov, _, _ := bi.ListBuiltInPresets(t.Context())
				pn, _ := anyProvider(prov)
				return pn, spec.ModelPresetID("ghost")
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			dir := t.TempDir()
			bi, _ := NewBuiltInPresets(ctx, dir, 0)
			t.Cleanup(func() {
				_ = bi.Close()
			})

			pn, mid := tc.setup(t, bi)
			if pn == "" {
				return // skipped inside setup
			}

			_, err := bi.SetDefaultModelPreset(ctx, pn, mid)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected: %v", err)
			}

			// Snapshot must reflect the change.
			prov, _, _ := bi.ListBuiltInPresets(ctx)
			if prov[pn].DefaultModelPresetID != mid {
				t.Errorf("defaultModelPresetID not updated, want %s", mid)
			}

			// Overlay flag must be present and consistent.
			flag, ok, err := bi.providerDefaultModelIDOverlayFlags.GetFlag(
				ctx, builtInProviderDefaultModelIDKey(pn))
			if err != nil {
				t.Fatalf("providerDefaultModelIDOverlayFlags.GetFlag: %v", err)
			}
			if !ok || flag.Value != mid {
				t.Errorf("overlay mismatch: present=%v, value=%s, want present=true, value=%s",
					ok, flag.Value, mid)
			}
		})
	}
}

func TestRebuildSnapshot_DefaultModelPreset(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	bi, _ := NewBuiltInPresets(ctx, dir, time.Hour)
	t.Cleanup(func() {
		_ = bi.Close()
	})

	_, models, _ := bi.ListBuiltInPresets(ctx)
	pn, _, mp := anyModel(models)

	// Need another model ID to switch to – if there is none, skip.
	var newID spec.ModelPresetID
	for id := range models[pn] {
		if id != mp.ID {
			newID = id
			break
		}
	}
	if newID == "" {
		t.Skip("dataset has only one model – cannot test rebuild for default overlay")
	}

	// Directly set flag (bypassing helper) then call rebuild.
	_, _ = bi.providerDefaultModelIDOverlayFlags.SetFlag(
		ctx, builtInProviderDefaultModelIDKey(pn), newID)

	bi.mu.Lock()
	_ = bi.rebuildSnapshot(ctx)
	bi.mu.Unlock()

	prov, _, _ := bi.ListBuiltInPresets(ctx)
	if prov[pn].DefaultModelPresetID != newID {
		t.Error("rebuild did not apply default-model overlay")
	}
}

func TestAsyncRebuild_DefaultModelPreset(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	bi, _ := NewBuiltInPresets(ctx, dir, 5*time.Millisecond)
	t.Cleanup(func() {
		_ = bi.Close()
	})

	_, models, _ := bi.ListBuiltInPresets(ctx)
	pn, _, mp := anyModel(models)

	// Need another model ID to switch to.
	var newID spec.ModelPresetID
	for id := range models[pn] {
		if id != mp.ID {
			newID = id
			break
		}
	}
	if newID == "" {
		t.Skip("dataset has only one model – cannot test async rebuild")
	}

	time.Sleep(10 * time.Millisecond) // ensure snapshot considered stale

	_, _ = bi.SetDefaultModelPreset(ctx, pn, newID)
	time.Sleep(20 * time.Millisecond) // allow async worker to run

	prov, _, _ := bi.ListBuiltInPresets(ctx)
	if prov[pn].DefaultModelPresetID != newID {
		t.Error("async rebuild missed default-model change")
	}
}

func TestProviderModelSync_Scenarios(t *testing.T) {
	ctx := t.Context()
	dir := t.TempDir()
	bi, err := NewBuiltInPresets(ctx, dir, time.Hour)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	t.Cleanup(func() {
		_ = bi.Close()
	})

	prov, models, _ := bi.ListBuiltInPresets(ctx)
	pn, mid, mp := anyModel(models)
	pname, p := anyProvider(prov)

	// Try to find a provider with ≥2 models to test default-model flip.
	var pn2 inferencegoSpec.ProviderName
	var secondModelID spec.ModelPresetID
	for provName, mm := range models {
		if len(mm) >= 2 {
			pn2 = provName
			for id := range mm {
				if id != prov[provName].DefaultModelPresetID {
					secondModelID = id
					break
				}
			}
			break
		}
	}
	if pn2 == "" {
		t.Log(
			"dataset has no provider with ≥2 models; the default-model change subtest will be skipped",
		)
	}

	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "model_toggle_is_reflected_in_provider_snapshot",
			run: func(t *testing.T) {
				t.Helper()
				newEnabled := !mp.IsEnabled

				// Toggle model flag (provider-model sync target).
				if _, err := bi.SetModelPresetEnabled(ctx, pn, mid, newEnabled); err != nil {
					t.Fatalf("SetModelPresetEnabled: %v", err)
				}

				// Force a snapshot rebuild to ensure provider snapshot gets the model overlay applied.
				bi.mu.Lock()
				_ = bi.rebuildSnapshot(ctx)
				bi.mu.Unlock()

				prov2, models2, _ := bi.ListBuiltInPresets(ctx)

				// Model map reflects the new flag.
				if models2[pn][mid].IsEnabled != newEnabled {
					t.Fatalf(
						"models view mismatch: got %v, want %v",
						models2[pn][mid].IsEnabled,
						newEnabled,
					)
				}
				// Provider snapshot reflects the same flag for the same model.
				if prov2[pn].ModelPresets[mid].IsEnabled != newEnabled {
					t.Fatalf("provider snapshot not in sync: got %v, want %v",
						prov2[pn].ModelPresets[mid].IsEnabled, newEnabled)
				}

				// Check some other properties remain built-in and consistent across both views.
				if !prov2[pn].ModelPresets[mid].IsBuiltIn || !models2[pn][mid].IsBuiltIn {
					t.Fatal("model lost IsBuiltIn flag after overlay")
				}

				// ModifiedAt of the model should match the overlay flag's ModifiedAt as applied in snapshot.
				flag, ok, err := bi.modelOverlayFlags.GetFlag(ctx, getModelKey(pn, mid))
				if err != nil {
					t.Fatalf("modelOverlayFlags.GetFlag: %v", err)
				}
				if !ok {
					t.Fatal("expected model overlay flag to be present")
				}
				gotProvMod := prov2[pn].ModelPresets[mid].ModifiedAt
				gotModelsMod := models2[pn][mid].ModifiedAt
				if !gotProvMod.Equal(flag.ModifiedAt) || !gotModelsMod.Equal(flag.ModifiedAt) {
					t.Fatalf("ModifiedAt mismatch: provider=%v models=%v overlay=%v",
						gotProvMod, gotModelsMod, flag.ModifiedAt)
				}
			},
		},
		{
			name: "provider_toggle_does_not_change_models",
			run: func(t *testing.T) {
				t.Helper()
				// Take a stable snapshot of models for this provider.
				_, beforeModels, _ := bi.ListBuiltInPresets(ctx)
				before := beforeModels[pname]

				// Toggle the provider flag.
				newEnabled := !p.IsEnabled
				if _, err := bi.SetProviderEnabled(ctx, pname, newEnabled); err != nil {
					t.Fatalf("SetProviderEnabled: %v", err)
				}

				// Rebuild to ensure provider snapshot reflects the change.
				bi.mu.Lock()
				_ = bi.rebuildSnapshot(ctx)
				bi.mu.Unlock()

				_, afterModels, _ := bi.ListBuiltInPresets(ctx)
				after := afterModels[pname]

				// Models map should remain unchanged for this provider (no side effects of provider toggle).
				if len(before) != len(after) {
					t.Fatalf(
						"model count changed after provider toggle: before=%d after=%d",
						len(before),
						len(after),
					)
				}
				for id, bm := range before {
					am, ok := after[id]
					if !ok {
						t.Fatalf("model %q disappeared after provider toggle", id)
					}
					if bm.IsEnabled != am.IsEnabled {
						t.Fatalf(
							"model %q IsEnabled changed after provider toggle: %v -> %v",
							id,
							bm.IsEnabled,
							am.IsEnabled,
						)
					}
				}
			},
		},
		{
			name: "default_model_change_reflected_and_model_map_consistent",
			run: func(t *testing.T) {
				t.Helper()
				if pn2 == "" || secondModelID == "" {
					t.Skip("no provider with ≥2 models; skipping default model change scenario")
				}

				// Change provider default model.
				if _, err := bi.SetDefaultModelPreset(ctx, pn2, secondModelID); err != nil {
					t.Fatalf("SetDefaultModelPreset: %v", err)
				}

				// Force rebuild to apply overlay onto view.
				bi.mu.Lock()
				_ = bi.rebuildSnapshot(ctx)
				bi.mu.Unlock()

				prov2, models2, _ := bi.ListBuiltInPresets(ctx)
				if prov2[pn2].DefaultModelPresetID != secondModelID {
					t.Fatalf("defaultModelPresetID not updated: got=%s want=%s",
						prov2[pn2].DefaultModelPresetID, secondModelID)
				}

				// Consistency: provider.ModelPresets must have the exact same keys as models map.
				mpMap := prov2[pn2].ModelPresets
				if len(mpMap) != len(models2[pn2]) {
					t.Fatalf(
						"provider.ModelPresets key-count mismatch: %d vs %d",
						len(mpMap),
						len(models2[pn2]),
					)
				}
				for id := range models2[pn2] {
					if _, ok := mpMap[id]; !ok {
						t.Fatalf("provider.ModelPresets missing model %q present in models map", id)
					}
				}

				// Overlay flag must be present and consistent for default-model.
				flag, ok, err := bi.providerDefaultModelIDOverlayFlags.GetFlag(
					ctx,
					builtInProviderDefaultModelIDKey(pn2),
				)
				if err != nil {
					t.Fatalf("providerDefaultModelIDOverlayFlags.GetFlag: %v", err)
				}
				if !ok || flag.Value != secondModelID {
					t.Fatalf(
						"default-model overlay mismatch: present=%v value=%s want present=true value=%s",
						ok,
						flag.Value,
						secondModelID,
					)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, tc.run)
	}
}

func TestScopedModelIDs_AcrossProviders_OverlayIsolation(t *testing.T) {
	ctx := t.Context()

	p1 := inferencegoSpec.ProviderName("provA")
	p2 := inferencegoSpec.ProviderName("provB")
	commonID := spec.ModelPresetID("m1")
	otherID := spec.ModelPresetID("m2")

	// Build a schema where both providers have the same model ID "m1".
	s := buildSchemaScopedDuplicateIDs(p1, p2, commonID, otherID)
	fsys := fstest.MapFS{builtin.BuiltInModelPresetsJSON: {Data: s}}

	dir := t.TempDir()
	bi, err := NewBuiltInPresets(ctx, dir, time.Hour, WithModelPresetsFS(fsys, "."))
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	t.Cleanup(func() {
		_ = bi.Close()
	})

	// Sanity checks: both providers and the common model exist in their own namespaces.
	prov, models, _ := bi.ListBuiltInPresets(ctx)
	if len(prov) != 2 {
		t.Fatalf("want 2 providers, got %d", len(prov))
	}
	if _, ok := models[p1][commonID]; !ok {
		t.Fatalf("provider %s missing model %s", p1, commonID)
	}
	if _, ok := models[p2][commonID]; !ok {
		t.Fatalf("provider %s missing model %s", p2, commonID)
	}

	// Table-driven: toggle in p1, verify no effect on p2; then toggle in p2 and verify independence again.
	tests := []struct {
		name       string
		targetProv inferencegoSpec.ProviderName
		targetID   spec.ModelPresetID
		newEnabled bool
		otherProv  inferencegoSpec.ProviderName
	}{
		{
			name:       "toggle_common_model_in_first_provider_only",
			targetProv: p1,
			targetID:   commonID,
			newEnabled: false,
			otherProv:  p2,
		},
		{
			name:       "toggle_common_model_in_second_provider_only",
			targetProv: p2,
			targetID:   commonID,
			newEnabled: true,
			otherProv:  p1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Snapshot current state.
			_, before, _ := bi.ListBuiltInPresets(ctx)

			beforeOther := before[tc.otherProv][tc.targetID].IsEnabled

			// Apply overlay to target provider's common model ID.
			if _, err := bi.SetModelPresetEnabled(ctx, tc.targetProv, tc.targetID, tc.newEnabled); err != nil {
				t.Fatalf("SetModelPresetEnabled: %v", err)
			}

			// Force rebuild to propagate provider snapshot update.
			bi.mu.Lock()
			_ = bi.rebuildSnapshot(ctx)
			bi.mu.Unlock()

			prov2, after, _ := bi.ListBuiltInPresets(ctx)

			// Target provider changed as requested (both in models view and provider snapshot).
			if after[tc.targetProv][tc.targetID].IsEnabled != tc.newEnabled {
				t.Fatalf("target models view mismatch: got %v, want %v",
					after[tc.targetProv][tc.targetID].IsEnabled, tc.newEnabled)
			}
			if prov2[tc.targetProv].ModelPresets[tc.targetID].IsEnabled != tc.newEnabled {
				t.Fatalf("target provider snapshot mismatch: got %v, want %v",
					prov2[tc.targetProv].ModelPresets[tc.targetID].IsEnabled, tc.newEnabled)
			}

			// Other provider remains unaffected.
			if after[tc.otherProv][tc.targetID].IsEnabled != beforeOther {
				t.Fatalf("other provider's model was affected: before=%v after=%v",
					beforeOther, after[tc.otherProv][tc.targetID].IsEnabled)
			}
			if prov2[tc.otherProv].ModelPresets[tc.targetID].IsEnabled != beforeOther {
				t.Fatalf("other provider snapshot was affected: before=%v after=%v",
					beforeOther, prov2[tc.otherProv].ModelPresets[tc.targetID].IsEnabled)
			}

			// Overlay rows should be isolated by provider.
			flag1, ok1, err := bi.modelOverlayFlags.GetFlag(
				ctx,
				getModelKey(tc.targetProv, tc.targetID),
			)
			if err != nil {
				t.Fatalf("modelOverlayFlags.GetFlag(target): %v", err)
			}
			if !ok1 || flag1.Value != tc.newEnabled {
				t.Fatalf(
					"target overlay row missing or incorrect; present=%v value=%v want=%v",
					ok1,
					flag1.Value,
					tc.newEnabled,
				)
			}

			// Other provider should not have a flag unless it was changed earlier in the suite.
			flag2, ok2, err := bi.modelOverlayFlags.GetFlag(
				ctx,
				getModelKey(tc.otherProv, tc.targetID),
			)
			if err != nil {
				t.Fatalf("modelOverlayFlags.GetFlag(other): %v", err)
			}
			// We tolerate "ok2 == true but value == beforeOther" if previous subtest toggled it.
			if ok2 && flag2.Value != beforeOther {
				t.Fatalf(
					"unexpected overlay for other provider: present=%v value=%v want(before)=%v",
					ok2,
					flag2.Value,
					beforeOther,
				)
			}
		})
	}
}

// Helper: schema with same model ID present in two different providers (scoped uniqueness).
func buildSchemaScopedDuplicateIDs(
	p1, p2 inferencegoSpec.ProviderName,
	commonID spec.ModelPresetID,
	extraID spec.ModelPresetID,
) []byte {
	mCommon := makeModelPreset(commonID)
	mExtra := makeModelPreset(extraID)

	pp1 := spec.ProviderPreset{
		SchemaVersion:            spec.SchemaVersion,
		Name:                     p1,
		DisplayName:              "Provider A",
		SDKType:                  inferencegoSpec.ProviderSDKTypeOpenAIChatCompletions,
		IsEnabled:                true,
		CreatedAt:                time.Now(),
		ModifiedAt:               time.Now(),
		Origin:                   "https://example.com/a",
		ChatCompletionPathPrefix: spec.DefaultOpenAIChatCompletionsPrefix,
		DefaultModelPresetID:     commonID,
		ModelPresets: map[spec.ModelPresetID]spec.ModelPreset{
			commonID: mCommon,
			extraID:  mExtra,
		},
	}
	pp2 := spec.ProviderPreset{
		SchemaVersion:            spec.SchemaVersion,
		Name:                     p2,
		DisplayName:              "Provider B",
		SDKType:                  inferencegoSpec.ProviderSDKTypeOpenAIChatCompletions,
		IsEnabled:                true,
		CreatedAt:                time.Now(),
		ModifiedAt:               time.Now(),
		Origin:                   "https://example.com/b",
		ChatCompletionPathPrefix: spec.DefaultOpenAIChatCompletionsPrefix,
		DefaultModelPresetID:     commonID,
		ModelPresets:             map[spec.ModelPresetID]spec.ModelPreset{commonID: mCommon},
	}

	s := spec.PresetsSchema{
		SchemaVersion:   spec.SchemaVersion,
		DefaultProvider: p1,
		ProviderPresets: map[inferencegoSpec.ProviderName]spec.ProviderPreset{p1: pp1, p2: pp2},
	}
	b, _ := json.Marshal(s)
	return b
}

func anyProvider(
	m map[inferencegoSpec.ProviderName]spec.ProviderPreset,
) (inferencegoSpec.ProviderName, spec.ProviderPreset) {
	for n, p := range m {
		return n, p
	}
	return "", spec.ProviderPreset{}
}

func anyModel(m map[inferencegoSpec.ProviderName]map[spec.ModelPresetID]spec.ModelPreset,
) (inferencegoSpec.ProviderName, spec.ModelPresetID, spec.ModelPreset) {
	for pn, mm := range m {
		for mid, mp := range mm {
			return pn, mid, mp
		}
	}
	return "", "", spec.ModelPreset{}
}

func newPresetsFromFS(t *testing.T, mem fs.FS) (*BuiltInPresets, error) {
	t.Helper()
	ctx := t.Context()
	dir := t.TempDir()
	bi, err := NewBuiltInPresets(ctx, dir, time.Hour, WithModelPresetsFS(mem, "."))
	if err != nil {
		// If NewBuiltInPresets returned an error but allocated/started something,
		// make sure to close it immediately; otherwise tests on Windows can fail
		// during TempDir cleanup because the overlay sqlite file remains open.
		if bi != nil {
			_ = bi.Close()
			if runtime.GOOS == windows {
				// Give SQLite time to release handles on Windows.
				t.Log("modelpreset: sleeping in win")
				time.Sleep(time.Millisecond * 100)
			}
		}
		return nil, err
	}
	// Register normal cleanup for the successful case.
	t.Cleanup(func() {
		_ = bi.Close()
	})
	return bi, nil
}

func buildSchemaDefaultMissing(pn inferencegoSpec.ProviderName, mpid spec.ModelPresetID) []byte {
	model := makeModelPreset(mpid)
	pp := spec.ProviderPreset{
		SchemaVersion:            spec.SchemaVersion,
		Name:                     pn,
		DisplayName:              "Demo",
		SDKType:                  inferencegoSpec.ProviderSDKTypeOpenAIChatCompletions,
		IsEnabled:                true,
		CreatedAt:                time.Now(),
		ModifiedAt:               time.Now(),
		Origin:                   "https://x",
		ChatCompletionPathPrefix: spec.DefaultOpenAIChatCompletionsPrefix,
		DefaultModelPresetID:     "ghost",
		ModelPresets:             map[spec.ModelPresetID]spec.ModelPreset{mpid: model},
	}
	s := spec.PresetsSchema{
		SchemaVersion:   spec.SchemaVersion,
		DefaultProvider: pn,
		ProviderPresets: map[inferencegoSpec.ProviderName]spec.ProviderPreset{pn: pp},
	}
	b, _ := json.Marshal(s)
	return b
}

func buildHappySchema(pn inferencegoSpec.ProviderName, mpid spec.ModelPresetID) []byte {
	model := makeModelPreset(mpid)
	pp := spec.ProviderPreset{
		SchemaVersion:            spec.SchemaVersion,
		Name:                     pn,
		DisplayName:              "Demo",
		SDKType:                  inferencegoSpec.ProviderSDKTypeOpenAIChatCompletions,
		IsEnabled:                true,
		CreatedAt:                time.Now(),
		ModifiedAt:               time.Now(),
		Origin:                   "https://example.com",
		ChatCompletionPathPrefix: spec.DefaultOpenAIChatCompletionsPrefix,
		DefaultModelPresetID:     mpid,
		ModelPresets:             map[spec.ModelPresetID]spec.ModelPreset{mpid: model},
	}
	s := spec.PresetsSchema{
		SchemaVersion:   spec.SchemaVersion,
		DefaultProvider: pp.Name,
		ProviderPresets: map[inferencegoSpec.ProviderName]spec.ProviderPreset{pn: pp},
	}
	b, _ := json.Marshal(s)
	return b
}

func makeModelPreset(mpid spec.ModelPresetID) spec.ModelPreset {
	temp := 0.1
	return spec.ModelPreset{
		SchemaVersion: spec.SchemaVersion,
		ID:            mpid,
		Name:          spec.ModelName(mpid),
		DisplayName:   spec.ModelDisplayName("Demo Model"),
		Slug:          "demo",
		IsEnabled:     true,
		CreatedAt:     time.Now(),
		ModifiedAt:    time.Now(),
		Temperature:   &temp,
	}
}

func PrintJSON(v any) {
	p, err := json.MarshalIndent(v, "", "")
	if err == nil {
		fmt.Print("request params", "json", string(p))
	}
}
