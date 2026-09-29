package overlay

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestRegistrationAndGroupCreation(t *testing.T) {
	st, _ := tmpStore(t, WithKeyType[BundleID](), WithKeyType[TemplateID]())
	defer st.db.Close()

	// Check groups exist in SQLite.
	groups := queryGroups(t, st)
	if !containsString(groups, "bundles") {
		t.Fatalf("expected group %q to exist in db", "bundles")
	}
	if !containsString(groups, "templates") {
		t.Fatalf("expected group %q to exist in db", "templates")
	}
	if containsString(groups, "other") {
		t.Fatalf("group %q must NOT exist (was never registered)", "other")
	}

	// Set and get a flag.
	if _, err := st.SetFlag(t.Context(), BundleID("foo"), json.RawMessage(`true`)); err != nil {
		t.Fatalf("SetFlag: %v", err)
	}
	f, ok, err := st.GetFlag(t.Context(), BundleID("foo"))
	if err != nil || !ok {
		t.Fatalf("GetFlag: %v %v", err, ok)
	}
	var v bool
	if err := json.Unmarshal(f.Value, &v); err != nil || !v {
		t.Fatalf("unexpected value: %v %v", err, v)
	}
}

func TestDuplicateRegistrationFails(t *testing.T) {
	_, err := NewOverlayStore(t.Context(), t.TempDir()+"/o.db",
		WithKeyType[BundleID](),
		WithKeyType[DuplicateBundleID](),
	)
	if err == nil {
		t.Fatal("expected duplicate registration to error")
	}
}

func TestNoRegistrationMeansNoGroups(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "overlay.db")
	// No options.
	_, err := NewOverlayStore(t.Context(), path)
	if err == nil {
		t.Errorf("NewOverlayStore should fail with no gorups")
	}
}

func TestCRUDAndBoundaries(t *testing.T) {
	st, _ := tmpStore(
		t,
		WithKeyType[BundleID](),
		WithKeyType[TemplateID](),
		WithKeyType[CompositeKey](),
	)
	defer st.db.Close()

	type testCase struct {
		name string
		key  Key
		def  bool
		set  []bool
	}
	tests := []testCase{
		{"bundle/normal", BundleID("core"), false, []bool{true, false, true}},
		{"bundle/empty", BundleID(""), true, []bool{false, true}},
		{"template/empty", TemplateID(""), false, []bool{true}},
		{"template/zero", TemplateID(""), false, []bool{}},
		{"composite/zero", CompositeKey{}, true, []bool{false}},
		{"composite/long", CompositeKey{"a", string(make([]byte, 1000))}, false, []bool{true}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := getEnabled(t, st, tc.key, tc.def)
			if err != nil {
				t.Fatalf("GetFlag 1: %v", err)
			}
			if got != tc.def {
				t.Fatalf("initial get = %v, want %v", got, tc.def)
			}

			for i, v := range tc.set {
				if _, err := st.SetFlag(t.Context(), tc.key, marshalBool(v)); err != nil {
					t.Fatalf("SetFlag #%d: %v", i, err)
				}
				got, err := getEnabled(t, st, tc.key, !v)
				if err != nil {
					t.Fatalf("GetFlag #%d: %v", i, err)
				}
				if got != v {
					t.Fatalf("after set #%d get = %v, want %v", i, got, v)
				}
			}

			if err := st.DeleteKey(t.Context(), tc.key); err != nil {
				t.Fatalf("Delete: %v", err)
			}
			got, err = getEnabled(t, st, tc.key, tc.def)
			if err != nil {
				t.Fatalf("GetFlag after delete: %v", err)
			}
			if got != tc.def {
				t.Fatalf("after delete get = %v, want default %v", got, tc.def)
			}
		})
	}
}

func TestUnregisteredKeyErrors(t *testing.T) {
	st, _ := tmpStore(t, WithKeyType[BundleID]())
	wantErr := "not registered"
	if _, err := st.SetFlag(t.Context(), TemplateID("x"), marshalBool(true)); err == nil ||
		!containsErr(err, wantErr) {
		t.Fatalf("SetFlag with unregistered key should fail, got %v", err)
	}
	_, _, err := st.GetFlag(t.Context(), TemplateID("x"))
	if err == nil || !containsErr(err, wantErr) {
		t.Fatalf("GetFlag with unregistered key should fail, got %v", err)
	}
	if err := st.DeleteKey(t.Context(), TemplateID("x")); err == nil || !containsErr(err, wantErr) {
		t.Fatalf("Delete with unregistered key should fail, got %v", err)
	}
}

func TestPersistenceAcrossReopen(t *testing.T) {
	st, path := tmpStore(t, WithKeyType[BundleID]())
	defer st.db.Close()
	if _, err := st.SetFlag(t.Context(), BundleID("persist"), marshalBool(true)); err != nil {
		t.Fatalf("set: %v", err)
	}
	st.db.Close()

	st2, err := NewOverlayStore(t.Context(), path, WithKeyType[BundleID]())
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	t.Cleanup(func() {
		_ = st2.Close()
	})

	flag, ok, err := st2.GetFlag(t.Context(), BundleID("persist"))
	if err != nil {
		t.Fatalf("get after reopen: %v", err)
	}
	if !ok {
		t.Fatal("value lost after reopen")
	}
	var v bool
	if err := json.Unmarshal(flag.Value, &v); err != nil || !v {
		t.Fatalf("unexpected value after reopen: %v %v", err, v)
	}
}

func TestConcurrentAccess(t *testing.T) {
	const n = 100
	st, _ := tmpStore(t, WithKeyType[BundleID]())
	defer st.db.Close()
	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		go func() {
			defer wg.Done()
			k := BundleID("id-" + BundleID(strconv.Itoa(i)).ID())
			val := i%2 == 0
			if _, err := st.SetFlag(t.Context(), k, marshalBool(val)); err != nil {
				t.Errorf("set: %v", err)
			}
			got, err := getEnabled(t, st, k, !val)
			if err != nil {
				t.Errorf("get: %v", err)
			}
			if got != val {
				t.Errorf("concurrent mismatch: key=%v want=%v got=%v", k, val, got)
			}
		}()
	}
	wg.Wait()
}

func TestDefensiveEmptyKey(t *testing.T) {
	st, _ := tmpStore(t, WithKeyType[BundleID]())
	defer st.db.Close()
	if _, err := st.SetFlag(t.Context(), BundleID(""), marshalBool(true)); err != nil {
		t.Fatalf("SetFlag: %v", err)
	}
	ok, err := getEnabled(t, st, BundleID(""), false)
	if err != nil || !ok {
		t.Fatalf("GetFlag: %v %v", ok, err)
	}
	if err := st.DeleteKey(t.Context(), BundleID("")); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	ok, err = getEnabled(t, st, BundleID(""), false)
	if err != nil || ok {
		t.Fatalf("GetFlag after delete: %v %v", ok, err)
	}
}

func TestToggleBackAndForth(t *testing.T) {
	st, _ := tmpStore(t, WithKeyType[BundleID]())
	defer st.db.Close()
	k := BundleID("toggle")
	for i := range 10 {
		val := i%2 == 0
		if _, err := st.SetFlag(t.Context(), k, marshalBool(val)); err != nil {
			t.Fatalf("SetFlag: %v", err)
		}
		got, err := getEnabled(t, st, k, !val)
		if err != nil {
			t.Fatalf("GetFlag: %v", err)
		}
		if got != val {
			t.Fatalf("toggle #%d: got %v want %v", i, got, val)
		}
	}
}

func TestLargeKey(t *testing.T) {
	st, _ := tmpStore(t, WithKeyType[BundleID]())
	defer st.db.Close()
	large := BundleID(string(make([]byte, 4096)))
	if _, err := st.SetFlag(t.Context(), large, marshalBool(true)); err != nil {
		t.Fatalf("SetFlag: %v", err)
	}
	ok, err := getEnabled(t, st, large, false)
	if err != nil || !ok {
		t.Fatalf("GetFlag: %v %v", ok, err)
	}
}

func TestSetAfterDelete(t *testing.T) {
	st, _ := tmpStore(t, WithKeyType[BundleID]())
	defer st.db.Close()
	k := BundleID("foo")
	if _, err := st.SetFlag(t.Context(), k, marshalBool(true)); err != nil {
		t.Fatalf("SetFlag: %v", err)
	}
	if err := st.DeleteKey(t.Context(), k); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := st.SetFlag(t.Context(), k, marshalBool(false)); err != nil {
		t.Fatalf("SetFlag after delete: %v", err)
	}
	ok, err := getEnabled(t, st, k, true)
	if err != nil || ok {
		t.Fatalf("GetFlag after re-set: %v %v", ok, err)
	}
}

func TestSetFlagTimestamps(t *testing.T) {
	st, _ := tmpStore(t, WithKeyType[BundleID]())
	defer st.db.Close()
	k := BundleID("ts")
	f1, err := st.SetFlag(t.Context(), k, marshalBool(true))
	if err != nil {
		t.Fatalf("SetFlag: %v", err)
	}
	if !f1.CreatedAt.Equal(f1.ModifiedAt) {
		t.Fatalf("on create timestamps differ: %+v", f1)
	}
	time.Sleep(10 * time.Millisecond)
	f2, err := st.SetFlag(t.Context(), k, marshalBool(false))
	if err != nil {
		t.Fatalf("SetFlag: %v", err)
	}
	if !f2.ModifiedAt.After(f2.CreatedAt) {
		t.Fatalf("modified_at not updated: %+v", f2)
	}
}

func TestTypedGroup_Bool(t *testing.T) {
	st, _ := tmpStore(t, WithKeyType[BundleID]())
	defer st.db.Close()
	grp, err := NewTypedGroup[BundleID, bool](t.Context(), st)
	if err != nil {
		t.Fatalf("NewTypedGroup: %v", err)
	}
	k := BundleID("typedbool")
	_, ok, err := grp.GetFlag(t.Context(), k)
	if err != nil {
		t.Fatalf("GetFlag: %v", err)
	}
	if ok {
		t.Fatalf("should not exist yet")
	}
	tf2, err := grp.SetFlag(t.Context(), k, true)
	if err != nil {
		t.Fatalf("SetFlag: %v", err)
	}
	if !tf2.Value {
		t.Fatalf("SetFlag: value not true")
	}
	tf3, ok, err := grp.GetFlag(t.Context(), k)
	if err != nil || !ok || !tf3.Value {
		t.Fatalf("GetFlag: %v %v %v", err, ok, tf3.Value)
	}
	if err := grp.DeleteKey(t.Context(), k); err != nil {
		t.Fatalf("DeleteKey: %v", err)
	}
	_, ok, err = grp.GetFlag(t.Context(), k)
	if err != nil {
		t.Fatalf("GetFlag after delete: %v", err)
	}
	if ok {
		t.Fatalf("should not exist after delete")
	}
}

func TestTypedGroup_IntStringStructSliceMap(t *testing.T) {
	st, _ := tmpStore(t, WithKeyType[TemplateID]())
	defer st.db.Close()

	type MyStruct struct {
		A int
		B string
	}
	typeCases := []struct {
		name string
		val  any
	}{
		{"int", 42},
		{"string", "hello"},
		{"struct", MyStruct{A: 1, B: "x"}},
		{"slice", []int{1, 2, 3}},
		{"map", map[string]int{"a": 1, "b": 2}},
	}
	for _, tc := range typeCases {
		t.Run(tc.name, func(t *testing.T) {
			switch v := tc.val.(type) {
			case int:
				grp, err := NewTypedGroup[TemplateID, int](t.Context(), st)
				if err != nil {
					t.Fatal(err)
				}
				k := TemplateID("int")
				_, err = grp.SetFlag(t.Context(), k, v)
				if err != nil {
					t.Fatal(err)
				}
				tf, ok, err := grp.GetFlag(t.Context(), k)
				if err != nil || !ok || tf.Value != v {
					t.Fatalf("GetFlag: %v %v %v", err, ok, tf.Value)
				}
			case string:
				grp, err := NewTypedGroup[TemplateID, string](t.Context(), st)
				if err != nil {
					t.Fatal(err)
				}
				k := TemplateID("str")
				_, err = grp.SetFlag(t.Context(), k, v)
				if err != nil {
					t.Fatal(err)
				}
				tf, ok, err := grp.GetFlag(t.Context(), k)
				if err != nil || !ok || tf.Value != v {
					t.Fatalf("GetFlag: %v %v %v", err, ok, tf.Value)
				}
			case MyStruct:
				grp, err := NewTypedGroup[TemplateID, MyStruct](t.Context(), st)
				if err != nil {
					t.Fatal(err)
				}
				k := TemplateID("struct")
				_, err = grp.SetFlag(t.Context(), k, v)
				if err != nil {
					t.Fatal(err)
				}
				tf, ok, err := grp.GetFlag(t.Context(), k)
				if err != nil || !ok || tf.Value != v {
					t.Fatalf("GetFlag: %v %v %+v", err, ok, tf.Value)
				}
			case []int:
				grp, err := NewTypedGroup[TemplateID, []int](t.Context(), st)
				if err != nil {
					t.Fatal(err)
				}
				k := TemplateID("slice")
				_, err = grp.SetFlag(t.Context(), k, v)
				if err != nil {
					t.Fatal(err)
				}
				tf, ok, err := grp.GetFlag(t.Context(), k)
				if err != nil || !ok || !equalIntSlice(tf.Value, v) {
					t.Fatalf("GetFlag: %v %v %v", err, ok, tf.Value)
				}
			case map[string]int:
				grp, err := NewTypedGroup[TemplateID, map[string]int](t.Context(), st)
				if err != nil {
					t.Fatal(err)
				}
				k := TemplateID("map")
				_, err = grp.SetFlag(t.Context(), k, v)
				if err != nil {
					t.Fatal(err)
				}
				tf, ok, err := grp.GetFlag(t.Context(), k)
				if err != nil || !ok || !equalStringIntMap(tf.Value, v) {
					t.Fatalf("GetFlag: %v %v %v", err, ok, tf.Value)
				}
			}
		})
	}
}

func TestTypedGroup_UnmarshalError(t *testing.T) {
	st, _ := tmpStore(t, WithKeyType[BundleID]())
	defer st.db.Close()
	// Write a string, try to read as int.
	grpStr, err := NewTypedGroup[BundleID, string](t.Context(), st)
	if err != nil {
		t.Fatal(err)
	}
	k := BundleID("badtype")
	_, err = grpStr.SetFlag(t.Context(), k, "hello")
	if err != nil {
		t.Fatal(err)
	}
	grpInt, err := NewTypedGroup[BundleID, int](t.Context(), st)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = grpInt.GetFlag(t.Context(), k)
	if err == nil {
		t.Fatal("expected unmarshal error")
	}
}

func TestFilePermissionError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "overlay.db")
	// Create file with no write permission.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o400)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	f.Close()
	_, err = NewOverlayStore(t.Context(), path, WithKeyType[BundleID]())
	if err == nil {
		t.Fatalf("NewOverlayStore: did not get error in new store creation")
	}
}

func TestCorruptedFile(t *testing.T) {
	// Not relevant for SQLite, but test opening a non-SQLite file.
	dir := t.TempDir()
	path := filepath.Join(dir, "overlay.db")
	if err := os.WriteFile(path, []byte("not a sqlite db"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := NewOverlayStore(t.Context(), path, WithKeyType[BundleID]())
	if err == nil {
		t.Fatal("expected NewOverlayStore to fail on corrupted file")
	}
}

func tmpStore(t *testing.T, opts ...Option) (s *Store, path string) {
	t.Helper()
	dir := t.TempDir()
	path = filepath.Join(dir, "overlay.db")
	s, err := NewOverlayStore(t.Context(), path, opts...)
	if err != nil {
		t.Fatalf("NewOverlayStore failed: %v", err)
	}
	t.Cleanup(func() {
		_ = s.Close()
	})
	return s, path
}

func getEnabled(t *testing.T, st *Store, k Key, def bool) (bool, error) {
	t.Helper()
	f, ok, err := st.GetFlag(t.Context(), k)
	if err != nil {
		return def, err
	}
	if !ok {
		return def, nil
	}
	var v bool
	if err := json.Unmarshal(f.Value, &v); err != nil {
		return def, err
	}
	return v, nil
}

func queryGroups(t *testing.T, st *Store) []string {
	t.Helper()
	rows, err := st.db.QueryContext(t.Context(), "SELECT group_id FROM groups")
	if err != nil || rows.Err() != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out = append(out, s)
	}
	return out
}
