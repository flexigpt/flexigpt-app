package store

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/ppipada/mapstore-go"
	"github.com/ppipada/mapstore-go/jsonencdec"

	"github.com/flexigpt/flexigpt-app/internal/setting/spec"
)

// Helper to compute SHA256 hex (same as computeSHA in production code).
func expectedSHA(in string) string {
	sum := sha256.Sum256([]byte(in))
	return hex.EncodeToString(sum[:])
}

func TestComputeSHA(t *testing.T) {
	cases := []struct {
		in string
	}{
		{in: ""},
		{in: "abc"},
		{in: "some long secret"},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got := computeSHA(tc.in)
			want := expectedSHA(tc.in)
			if got != want {
				t.Fatalf("computeSHA(%q) = %q, want %q", tc.in, got, want)
			}
		})
	}
}

func TestValidateTheme_TableDriven(t *testing.T) {
	cases := []struct {
		name    string
		theme   *spec.AppTheme
		wantErr bool
	}{
		{"nil", nil, true},
		{"system-valid", &spec.AppTheme{Type: spec.ThemeSystem, Name: spec.ThemeNameSystem}, false},
		{"system-invalid-name", &spec.AppTheme{Type: spec.ThemeSystem, Name: "applight"}, true},
		{"light-valid", &spec.AppTheme{Type: spec.ThemeLight, Name: spec.ThemeNameLight}, false},
		{"light-invalid-name", &spec.AppTheme{Type: spec.ThemeLight, Name: "wrong"}, true},
		{"dark-valid", &spec.AppTheme{Type: spec.ThemeDark, Name: spec.ThemeNameDark}, false},
		{"other-valid", &spec.AppTheme{Type: spec.ThemeOther, Name: "custom-theme"}, false},
		{"other-invalid-empty", &spec.AppTheme{Type: spec.ThemeOther, Name: ""}, true},
		{"unknown-type", &spec.AppTheme{Type: spec.ThemeType("unknown"), Name: "x"}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateTheme(tc.theme)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateTheme(%v) returned err=%v, wantErr=%v", tc.theme, err, tc.wantErr)
			}
		})
	}
}

func TestIsBuiltInKey_TableDriven(t *testing.T) {
	// Preserve old BuiltInAuthKeys and restore at end.
	old := BuiltInAuthKeys
	defer func() { BuiltInAuthKeys = old }()

	BuiltInAuthKeys = map[spec.AuthKeyType][]spec.AuthKeyName{
		"provider": {"alpha", "beta"},
		"other":    {"one"},
	}

	cases := []struct {
		typ  spec.AuthKeyType
		name spec.AuthKeyName
		want bool
	}{
		{typ: "provider", name: "alpha", want: true},
		{typ: "provider", name: "gamma", want: false},
		{typ: "other", name: "one", want: true},
		{typ: "missing", name: "one", want: false},
	}

	for _, tc := range cases {
		t.Run(string(tc.typ)+"/"+string(tc.name), func(t *testing.T) {
			got := isBuiltInKey(tc.typ, tc.name)
			if got != tc.want {
				t.Fatalf("isBuiltInKey(%q, %q) = %v, want %v", tc.typ, tc.name, got, tc.want)
			}
		})
	}
}

func TestValueEncDecGetter(t *testing.T) {
	enc := jsonencdec.JSONEncoderDecoder{}
	s := &SettingStore{encEncrypt: enc}

	cases := []struct {
		name    string
		path    []string
		wantNil bool
	}{
		{"secret-path", []string{"authKeys", "provider", "k", "secret"}, false},
		{"wrong-last", []string{"authKeys", "provider", "k", "sha256"}, true},
		{"short-path", []string{"authKeys", "provider"}, true},
		{"other-root", []string{"appTheme"}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := s.valueEncDecGetter(tc.path)
			if (got == nil) != tc.wantNil {
				t.Fatalf("valueEncDecGetter(%v) returned nil=%v, wantNil=%v", tc.path, got == nil, tc.wantNil)
			}
			if got != nil {
				// Just ensure the returned type matches our encoder type.
				if reflect.TypeOf(got) != reflect.TypeOf(enc) {
					t.Fatalf("unexpected encoder type: %T", got)
				}
			}
		})
	}
}

// integrationTestStore sets up a real mapstore.MapFileStore backed by a temp file.
// Returns the SettingStore (with store and encEncrypt populated) and a cleanup func.
func integrationTestStore(t *testing.T, defaultMap map[string]any) (store *SettingStore, cleanup func()) {
	t.Helper()
	tmpDir := t.TempDir()

	file := filepath.Join(tmpDir, "settings.json")
	encoder := jsonencdec.JSONEncoderDecoder{}

	// Value encoder/decoder getter: use JSON encoding for secret values only.
	valueEncDecGetter := func(path []string) mapstore.IOEncoderDecoder {
		if len(path) == 4 && path[0] == "authKeys" && path[3] == "secret" {
			return jsonencdec.JSONEncoderDecoder{}
		}
		return nil
	}

	fs, err := mapstore.NewMapFileStore(file, defaultMap, encoder,
		mapstore.WithCreateIfNotExists(true),
		mapstore.WithFileAutoFlush(true),
		mapstore.WithValueEncDecGetter(valueEncDecGetter),
	)
	if err != nil {
		// Clean up tmpDir before failing.
		_ = os.RemoveAll(tmpDir)
		t.Fatalf("NewMapFileStore: %v", err)
	}

	store = &SettingStore{
		store:      fs,
		encEncrypt: encoder,
	}

	cleanup = func() {
		_ = fs.Close()
		_ = os.RemoveAll(tmpDir)
	}

	return store, cleanup
}

func TestSettingStore_AuthKeyLifecycleAndSettings(t *testing.T) {
	// Start with a minimal default map having an "old" schema version to
	// exercise migration bump as well.
	defaultMap := map[string]any{
		"schemaVersion": "v-old",
		"appTheme": map[string]any{
			"type": string(spec.ThemeSystem),
			"name": string(spec.ThemeNameSystem),
		},
		"authKeys": map[string]any{},
	}

	store, cleanup := integrationTestStore(t, defaultMap)
	defer cleanup()

	// Ensure BuiltInAuthKeys deterministic for tests; restore at end.
	oldBuiltins := BuiltInAuthKeys
	defer func() { BuiltInAuthKeys = oldBuiltins }()
	BuiltInAuthKeys = map[spec.AuthKeyType][]spec.AuthKeyName{} // start empty

	ctx := t.Context()

	// GetSettings initially: should return appTheme and no authKeys.
	out, err := store.GetSettings(ctx, nil)
	if err != nil {
		t.Fatalf("GetSettings error: %v", err)
	}
	if out == nil || out.Body == nil {
		t.Fatalf("GetSettings returned nil body")
	}
	if out.Body.AppTheme.Type != spec.ThemeSystem {
		t.Fatalf("expected initial theme type %v, got %v", spec.ThemeSystem, out.Body.AppTheme.Type)
	}
	if len(out.Body.AuthKeys) != 0 {
		t.Fatalf("expected no auth keys initially, got %d", len(out.Body.AuthKeys))
	}

	// Invalid SetAuthKey requests should return ErrInvalidArgument.
	badReqs := []*spec.SetAuthKeyRequest{
		nil,
		{Type: "", KeyName: "", Body: nil},
		{Type: spec.AuthKeyTypeProvider, KeyName: "", Body: &spec.SetAuthKeyRequestBody{Secret: "x"}},
		{Type: "", KeyName: "k", Body: &spec.SetAuthKeyRequestBody{Secret: "x"}},
		{Type: spec.AuthKeyTypeProvider, KeyName: "k", Body: nil},
	}
	for i, br := range badReqs {
		_, err := store.SetAuthKey(ctx, br)
		if !errors.Is(err, spec.ErrInvalidArgument) {
			t.Fatalf("bad request %d: expected ErrInvalidArgument, got %v", i, err)
		}
	}

	// Create a key and verify GetAuthKey and GetSettings reflect it.
	setReq := &spec.SetAuthKeyRequest{
		Type:    spec.AuthKeyTypeProvider,
		KeyName: "key1",
		Body:    &spec.SetAuthKeyRequestBody{Secret: "secret123"},
	}
	if _, err := store.SetAuthKey(ctx, setReq); err != nil {
		t.Fatalf("SetAuthKey failed: %v", err)
	}

	got, err := store.GetAuthKey(ctx, &spec.GetAuthKeyRequest{Type: setReq.Type, KeyName: setReq.KeyName})
	if err != nil {
		t.Fatalf("GetAuthKey failed: %v", err)
	}
	if got == nil || got.Body == nil {
		t.Fatalf("GetAuthKey returned nil body")
	}
	if got.Body.Secret != "secret123" {
		t.Fatalf("GetAuthKey secret mismatch: got %q", got.Body.Secret)
	}
	if got.Body.SHA256 != computeSHA("secret123") {
		t.Fatalf("GetAuthKey sha mismatch: got %q want %q", got.Body.SHA256, computeSHA("secret123"))
	}
	if !got.Body.NonEmpty {
		t.Fatalf("GetAuthKey NonEmpty expected true")
	}

	// GetSettings should show metadata (sha, nonEmpty) but not the secret.
	all, err := store.GetSettings(ctx, nil)
	if err != nil {
		t.Fatalf("GetSettings after SetAuthKey failed: %v", err)
	}
	found := false
	for _, meta := range all.Body.AuthKeys {
		if meta.Type == setReq.Type && meta.KeyName == setReq.KeyName {
			found = true
			if meta.SHA256 != computeSHA("secret123") {
				t.Fatalf("GetSettings sha mismatch: got %q want %q", meta.SHA256, computeSHA("secret123"))
			}
			if !meta.NonEmpty {
				t.Fatalf("GetSettings nonEmpty expected true")
			}
		}
	}
	if !found {
		t.Fatalf("auth key meta not found in GetSettings")
	}

	// Test built-in key: when BuiltInAuthKeys contains a key, DeleteAuthKey must be read-only.
	BuiltInAuthKeys = map[spec.AuthKeyType][]spec.AuthKeyName{
		spec.AuthKeyTypeProvider: {"built-in"},
	}

	if _, err := store.SetAuthKey(
		ctx,
		&spec.SetAuthKeyRequest{
			Type:    spec.AuthKeyTypeProvider,
			KeyName: "built-in",
			Body:    &spec.SetAuthKeyRequestBody{Secret: "s"},
		},
	); err != nil {
		t.Fatalf("SetAuthKey for built-in failed: %v", err)
	}
	_, err = store.DeleteAuthKey(ctx, &spec.DeleteAuthKeyRequest{Type: spec.AuthKeyTypeProvider, KeyName: "built-in"})
	if !errors.Is(err, spec.ErrBuiltInAuthKeyReadOnly) {
		t.Fatalf("expected ErrBuiltInAuthKeyReadOnly when deleting built-in, got %v", err)
	}

	if _, err := store.GetAuthKey(
		ctx,
		&spec.GetAuthKeyRequest{Type: spec.AuthKeyTypeProvider, KeyName: "built-in"},
	); err != nil {
		t.Fatalf("expected built-in key to remain after failed delete, GetAuthKey error: %v", err)
	}

	// Delete a non-built-in key: create a custom type with a single key then delete it.
	if _, err := store.SetAuthKey(
		ctx,
		&spec.SetAuthKeyRequest{
			Type:    spec.AuthKeyType("customType"),
			KeyName: "only",
			Body:    &spec.SetAuthKeyRequestBody{Secret: "v"},
		},
	); err != nil {
		t.Fatalf("SetAuthKey for customType failed: %v", err)
	}
	if _, err := store.DeleteAuthKey(
		ctx,
		&spec.DeleteAuthKeyRequest{Type: spec.AuthKeyType("customType"), KeyName: "only"},
	); err != nil {
		t.Fatalf("DeleteAuthKey failed: %v", err)
	}
	// After deletion, GetAuthKey should return not found.
	if _, err := store.GetAuthKey(
		ctx,
		&spec.GetAuthKeyRequest{Type: spec.AuthKeyType("customType"), KeyName: "only"},
	); !errors.Is(err, spec.ErrAuthKeyNotFound) {
		t.Fatalf("expected ErrAuthKeyNotFound after delete, got %v", err)
	}
	// And the type map should be removed entirely (no "customType" key under authKeys).
	raw, err := store.store.GetAll(false)
	if err != nil {
		t.Fatalf("store.GetAll failed: %v", err)
	}
	if akRaw, ok := raw["authKeys"].(map[string]any); ok {
		if _, ok2 := akRaw["customType"]; ok2 {
			t.Fatalf("expected 'customType' removed from authKeys map after deleting its only key")
		}
	}

	// Test migration: create a fresh store with missing built-in keys and different schemaVersion.
	{
		defaultMap2 := map[string]any{
			"schemaVersion": "old-schema",
			"appTheme": map[string]any{
				"type": string(spec.ThemeSystem),
				"name": string(spec.ThemeNameSystem),
			},
			"authKeys": map[string]any{},
		}
		store2, cleanup2 := integrationTestStore(t, defaultMap2)
		defer cleanup2()

		// Force BuiltInAuthKeys to include two entries to be added by Migrate.
		old2 := BuiltInAuthKeys
		BuiltInAuthKeys = map[spec.AuthKeyType][]spec.AuthKeyName{
			spec.AuthKeyTypeProvider: {"p1", "p2"},
		}
		defer func() { BuiltInAuthKeys = old2 }()

		if err := store2.Migrate(t.Context()); err != nil {
			t.Fatalf("Migrate failed: %v", err)
		}

		raw2, err := store2.store.GetAll(false)
		if err != nil {
			t.Fatalf("store2.GetAll failed: %v", err)
		}
		// SchemaVersion should be updated to current.
		if raw2["schemaVersion"] != spec.SchemaVersion {
			t.Fatalf("schemaVersion not updated by Migrate: got %v want %v", raw2["schemaVersion"], spec.SchemaVersion)
		}
		akRaw, ok := raw2["authKeys"].(map[string]any)
		if !ok {
			t.Fatalf("authKeys missing after Migrate")
		}
		provRaw, ok := akRaw[string(spec.AuthKeyTypeProvider)].(map[string]any)
		if !ok {
			t.Fatalf("provider map missing after Migrate")
		}
		for _, name := range []string{"p1", "p2"} {
			entry, ok := provRaw[name].(map[string]any)
			if !ok {
				t.Fatalf("built-in key %s missing in provider map", name)
			}
			// Check expected fields exist: secret, sha256, nonEmpty.
			if entry["secret"] != "" {
				t.Fatalf("expected empty secret for %s, got %v", name, entry["secret"])
			}
			if entry["sha256"] != computeSHA("") {
				t.Fatalf("expected sha256 of empty string for %s, got %v", name, entry["sha256"])
			}
			if entry["nonEmpty"] != false {
				t.Fatalf("expected nonEmpty=false for %s, got %v", name, entry["nonEmpty"])
			}
		}
	}

	// Test SetAppTheme invalid and valid cases.
	// Invalid: nil request.
	if _, err := store.SetAppTheme(ctx, nil); !errors.Is(err, spec.ErrInvalidArgument) {
		t.Fatalf("SetAppTheme(nil) expected ErrInvalidArgument, got %v", err)
	}
	// Invalid: wrong name for system.
	if _, err := store.SetAppTheme(
		ctx,
		&spec.SetAppThemeRequest{Body: &spec.SetAppThemeRequestBody{Type: spec.ThemeSystem, Name: "applight"}},
	); err == nil {
		t.Fatalf("SetAppTheme with invalid theme expected error, got nil")
	}
	// Valid: set to light.
	if _, err := store.SetAppTheme(
		ctx,
		&spec.SetAppThemeRequest{Body: &spec.SetAppThemeRequestBody{Type: spec.ThemeLight, Name: spec.ThemeNameLight}},
	); err != nil {
		t.Fatalf("SetAppTheme valid failed: %v", err)
	}
	// Verify persisted appTheme.
	rawFinal, err := store.store.GetAll(false)
	if err != nil {
		t.Fatalf("store.GetAll failed: %v", err)
	}
	if atRaw, ok := rawFinal["appTheme"].(map[string]any); ok {
		if atRaw["type"] != string(spec.ThemeLight) || atRaw["name"] != string(spec.ThemeNameLight) {
			t.Fatalf("appTheme not updated as expected: %v", atRaw)
		}
	} else {
		t.Fatalf("appTheme missing after SetAppTheme")
	}
}

func TestGetSettings_Sorting(t *testing.T) {
	defaultMap := map[string]any{
		"schemaVersion": spec.SchemaVersion,
		"appTheme": map[string]any{
			"type": string(spec.ThemeSystem),
			"name": string(spec.ThemeNameSystem),
		},
		"authKeys": map[string]any{},
	}
	store, cleanup := integrationTestStore(t, defaultMap)
	defer cleanup()

	ctx := t.Context()

	// Create keys out-of-order across types and names.
	cases := []spec.SetAuthKeyRequest{
		{Type: spec.AuthKeyType("b"), KeyName: "z", Body: &spec.SetAuthKeyRequestBody{Secret: "s1"}},
		{Type: spec.AuthKeyType("a"), KeyName: "b", Body: &spec.SetAuthKeyRequestBody{Secret: "s2"}},
		{Type: spec.AuthKeyType("a"), KeyName: "a", Body: &spec.SetAuthKeyRequestBody{Secret: "s3"}},
	}
	for _, c := range cases {
		// Create local copy to take address.
		req := c
		if _, err := store.SetAuthKey(ctx, &req); err != nil {
			t.Fatalf("SetAuthKey failed: %v", err)
		}
	}

	out, err := store.GetSettings(ctx, nil)
	if err != nil {
		t.Fatalf("GetSettings failed: %v", err)
	}
	if out == nil || out.Body == nil {
		t.Fatalf("GetSettings returned nil body")
	}
	// Expect order: type "a" keys "a", "b", then type "b" key "z".
	wantOrder := []struct {
		typ  spec.AuthKeyType
		name spec.AuthKeyName
	}{
		{typ: spec.AuthKeyType("a"), name: "a"},
		{typ: spec.AuthKeyType("a"), name: "b"},
		{typ: spec.AuthKeyType("b"), name: "z"},
	}
	if len(out.Body.AuthKeys) != len(wantOrder) {
		t.Fatalf("unexpected number of auth keys: got %d want %d", len(out.Body.AuthKeys), len(wantOrder))
	}
	for i, w := range wantOrder {
		got := out.Body.AuthKeys[i]
		if got.Type != w.typ || got.KeyName != w.name {
			t.Fatalf("auth key[%d] = (%v,%v) want (%v,%v)", i, got.Type, got.KeyName, w.typ, w.name)
		}
	}

	// Extra sanity: ensure result is sorted even if we scramble insertion order
	// (we already added them out of order; but ensure sorted property via code).
	sorted := make([]spec.AuthKeyMeta, len(out.Body.AuthKeys))
	copy(sorted, out.Body.AuthKeys)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Type == sorted[j].Type {
			return sorted[i].KeyName < sorted[j].KeyName
		}
		return sorted[i].Type < sorted[j].Type
	})
	for i := range sorted {
		if sorted[i] != out.Body.AuthKeys[i] {
			t.Fatalf("GetSettings did not return sorted auth keys")
		}
	}
}
