package store

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/tool/spec"
)

func TestToolBundleCRUD(t *testing.T) {
	cases := []struct {
		name      string
		id        bundleitemutils.BundleID
		slug      bundleitemutils.BundleSlug
		display   string
		enabled   bool
		wantError bool
		expectMsg string
	}{
		{"Valid", "b1", "slug", "Bundle", true, false, ""},
		{"Disabled", "b2", "disabled", "Bundle", false, false, ""},
		{"MissingID", "", "s", "d", true, true, "required"},
		{"MissingSlug", "b3", "", "d", true, true, "required"},
		{"MissingDisplay", "b4", "s", "", true, true, "required"},
		{"BadSlugDot", "b5", "bad.slug", "d", true, true, "invalid"},
		{"BadSlugSpace", "b6", "bad slug", "d", true, true, "invalid"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, clean := newTestToolStore(t)
			defer clean()

			_, err := s.PutToolBundle(t.Context(), &spec.PutToolBundleRequest{
				BundleID: tc.id,
				Body: &spec.PutToolBundleRequestBody{
					Slug:        tc.slug,
					DisplayName: tc.display,
					IsEnabled:   tc.enabled,
				},
			})

			if tc.wantError {
				if err == nil || !strings.Contains(err.Error(), tc.expectMsg) {
					t.Fatalf("expected %q error, got %v", tc.expectMsg, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestToolBuiltInBundleGuards(t *testing.T) {
	s, clean := newTestToolStore(t)
	defer clean()

	bid, slug, _, ok := firstBuiltInTool(t, s)
	if !ok {
		t.Skip("No built-in catalogue present.")
	}

	// Modifying a built-in bundle must be rejected.
	_, err := s.PutToolBundle(t.Context(), &spec.PutToolBundleRequest{
		BundleID: bid,
		Body: &spec.PutToolBundleRequestBody{
			Slug:        bundleitemutils.BundleSlug((string(slug))),
			DisplayName: "illegal update",
			IsEnabled:   true,
		},
	})
	if !errors.Is(err, spec.ErrBuiltInReadOnly) {
		t.Fatalf("expected ErrBuiltInReadOnly, got %v", err)
	}

	// Deleting a built-in bundle must be rejected.
	_, err = s.DeleteToolBundle(t.Context(), &spec.DeleteToolBundleRequest{
		BundleID: bid,
	})
	if !errors.Is(err, spec.ErrBuiltInReadOnly) {
		t.Fatalf("expected ErrBuiltInReadOnly on delete, got %v", err)
	}

	// Enabling/disabling is allowed.
	_, err = s.PatchToolBundle(t.Context(), &spec.PatchToolBundleRequest{
		BundleID: bid,
		Body:     &spec.PatchToolBundleRequestBody{IsEnabled: false},
	})
	if err != nil {
		t.Fatalf("PatchToolBundle() on built-in failed: %v", err)
	}
}

func TestToolCRUD(t *testing.T) {
	s, clean := newTestToolStore(t)
	defer clean()

	mustPutToolBundle(t, s, "b1", "slug1", "Bundle", true)

	cases := []struct {
		name      string
		bid       bundleitemutils.BundleID
		slug      bundleitemutils.ItemSlug
		ver       bundleitemutils.ItemVersion
		display   string
		wantError bool
		msg       string
	}{
		{"Valid", "b1", "tool", "v1", "display", false, ""},
		{"MissingID", "", "s", "v1", "d", true, "required"},
		{"MissingSlug", "b1", "", "v1", "d", true, "required"},
		{"MissingVer", "b1", "s", "", "d", true, "required"},
		{"BadSlug", "b1", "a.b", "v1", "d", true, "invalid"},
		{"BadVer", "b1", "s", "v&1", "d", true, "invalid"},
		{"UnknownBundle", "x", "s", "v1", "d", true, "not found"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := s.PutTool(t.Context(), &spec.PutToolRequest{
				BundleID: tc.bid,
				ToolSlug: tc.slug,
				Version:  tc.ver,
				Body: &spec.PutToolRequestBody{
					DisplayName:  tc.display,
					IsEnabled:    true,
					UserCallable: true,
					LLMCallable:  true,

					ArgSchema: `{}`,

					Type:     spec.ToolTypeHTTP,
					HTTPImpl: dummyHTTPTool(),
				},
			})

			if tc.wantError {
				if err == nil || !strings.Contains(err.Error(), tc.msg) {
					t.Fatalf("expected %q error, got %v", tc.msg, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestToolVersionConflict(t *testing.T) {
	s, clean := newTestToolStore(t)
	defer clean()

	mustPutToolBundle(t, s, "b1", "slug", "Bundle", true)
	mustPutTool(t, s, "b1", "tool", "v1", "d", true)

	_, err := s.PutTool(t.Context(), &spec.PutToolRequest{
		BundleID: "b1", ToolSlug: "tool", Version: "v1",
		Body: &spec.PutToolRequestBody{
			DisplayName:  "dup",
			IsEnabled:    true,
			UserCallable: true,
			LLMCallable:  true,

			ArgSchema: `{}`,

			Type:     spec.ToolTypeHTTP,
			HTTPImpl: dummyHTTPTool(),
		},
	})
	if !errors.Is(err, spec.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestToolDisabledBundleGuard(t *testing.T) {
	s, clean := newTestToolStore(t)
	defer clean()

	mustPutToolBundle(t, s, "b1", "slug", "Bundle", false)

	_, err := s.PutTool(t.Context(), &spec.PutToolRequest{
		BundleID: "b1", ToolSlug: "tool", Version: "v1",
		Body: &spec.PutToolRequestBody{
			DisplayName:  "d",
			IsEnabled:    true,
			UserCallable: true,
			LLMCallable:  true,

			ArgSchema: `{}`,

			Type:     spec.ToolTypeHTTP,
			HTTPImpl: dummyHTTPTool(),
		},
	})
	if !errors.Is(err, spec.ErrBundleDisabled) {
		t.Fatalf("expected ErrBundleDisabled, got %v", err)
	}
}

func TestToolMultiVersionExact(t *testing.T) {
	s, clean := newTestToolStore(t)
	defer clean()

	mustPutToolBundle(t, s, "b1", "slug", "Bundle", true)

	vers := []string{"v1", "v2", "v3"}
	for _, v := range vers {
		mustPutTool(t, s, "b1", "tool", bundleitemutils.ItemVersion(v), "disp "+v, true)
		time.Sleep(5 * time.Millisecond)
	}

	for _, v := range vers {
		resp, err := s.GetTool(t.Context(), &spec.GetToolRequest{
			BundleID: "b1", ToolSlug: "tool", Version: bundleitemutils.ItemVersion(v),
		})
		if err != nil {
			t.Fatalf("GetTool(%s) failed: %v", v, err)
		}
		if resp.Body.Version != bundleitemutils.ItemVersion(v) {
			t.Fatalf("expected version %s, got %s", v, resp.Body.Version)
		}
	}

	// Omitted version must fail.
	_, err := s.GetTool(t.Context(), &spec.GetToolRequest{
		BundleID: "b1", ToolSlug: "tool", Version: "",
	})
	if !errors.Is(err, spec.ErrInvalidRequest) {
		t.Fatalf("expected ErrInvalidRequest for missing version, got %v", err)
	}
}

func TestToolBuiltInGuards(t *testing.T) {
	s, clean := newTestToolStore(t)
	defer clean()

	bid, slug, ver, ok := firstBuiltInTool(t, s)
	if !ok {
		t.Skip("No built-in catalogue present.")
	}

	// Creating a tool in a built-in bundle is forbidden.
	_, err := s.PutTool(t.Context(), &spec.PutToolRequest{
		BundleID: bid, ToolSlug: slug, Version: "v-new",
		Body: &spec.PutToolRequestBody{
			DisplayName:  "illegal",
			IsEnabled:    true,
			UserCallable: true,
			LLMCallable:  true,

			ArgSchema: `{}`,

			Type:     spec.ToolTypeHTTP,
			HTTPImpl: dummyHTTPTool(),
		},
	})
	if !errors.Is(err, spec.ErrBuiltInReadOnly) {
		t.Fatalf("expected ErrBuiltInReadOnly, got %v", err)
	}

	// Deleting a built-in tool must fail.
	_, err = s.DeleteTool(t.Context(), &spec.DeleteToolRequest{
		BundleID: bid, ToolSlug: slug, Version: ver,
	})
	if !errors.Is(err, spec.ErrBuiltInReadOnly) {
		t.Fatalf("expected ErrBuiltInReadOnly on delete, got %v", err)
	}

	// Patch (toggle enabled) is allowed.
	_, err = s.PatchTool(t.Context(), &spec.PatchToolRequest{
		BundleID: bid, ToolSlug: slug, Version: ver,
		Body: &spec.PatchToolRequestBody{IsEnabled: false},
	})
	if err != nil {
		t.Fatalf("PatchTool() for built-in failed: %v", err)
	}
}

func TestToolBundleListFiltering(t *testing.T) {
	s, clean := newTestToolStore(t)
	defer clean()

	builtInCnt, _ := builtinToolStatistics(t, s)

	mustPutToolBundle(t, s, "ub1", "slug1", "Bundle1", true)
	mustPutToolBundle(t, s, "ub2", "slug2", "Bundle2", false)

	tests := []struct {
		name            string
		includeDisabled bool
		filterIDs       []bundleitemutils.BundleID
		expectUser      int
	}{
		{"EnabledOnly", false, nil, 1},
		{"AllUsers", true, nil, 2},
		{"FilterUser", true, []bundleitemutils.BundleID{"ub1"}, 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := s.ListToolBundles(t.Context(), &spec.ListToolBundlesRequest{
				IncludeDisabled: tc.includeDisabled,
				BundleIDs:       tc.filterIDs,
			})
			if err != nil {
				t.Fatalf("ListToolBundles() failed: %v", err)
			}

			got := len(resp.Body.ToolBundles)
			want := tc.expectUser
			if tc.filterIDs == nil {
				want += builtInCnt
			}
			if got != want {
				t.Fatalf("expected %d bundles, got %d", want, got)
			}

			if tc.expectUser > 0 {
				find := func(id bundleitemutils.BundleID) bool {
					for _, b := range resp.Body.ToolBundles {
						if b.ID == id {
							return true
						}
					}
					return false
				}
				for _, id := range tc.filterIDs {
					if !find(id) {
						t.Fatalf("bundle %s missing from result", id)
					}
				}
			}
		})
	}
}

func TestToolListFiltering(t *testing.T) {
	s, clean := newTestToolStore(t)
	defer clean()

	mustPutToolBundle(t, s, "ub1", "slug1", "UserBundle", true)
	mustPutTool(t, s, "ub1", "t1", "v1", "t1", true, "tag1")
	mustPutTool(t, s, "ub1", "t2", "v1", "t2", false, "tag1", "tag2")

	tests := []struct {
		name            string
		includeDisabled bool
		tags            []string
		expect          int
	}{
		{"EnabledOnly", false, nil, 1},
		{"WithDisabled", true, nil, 2},
		{"TagFilter", true, []string{"tag2"}, 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := s.ListTools(t.Context(), &spec.ListToolsRequest{
				BundleIDs:       []bundleitemutils.BundleID{"ub1"},
				Tags:            tc.tags,
				IncludeDisabled: tc.includeDisabled,
			})
			if err != nil {
				t.Fatalf("ListTools() failed: %v", err)
			}
			if len(resp.Body.ToolListItems) != tc.expect {
				t.Fatalf("expected %d items, got %d", tc.expect, len(resp.Body.ToolListItems))
			}
		})
	}
}

func TestToolBundlePagination(t *testing.T) {
	s, clean := newTestToolStore(t)
	defer clean()

	ids := make([]bundleitemutils.BundleID, 0, 30)
	for i := range 30 {
		id := bundleitemutils.BundleID(fmt.Sprintf("u%02d", i))
		ids = append(ids, id)
		mustPutToolBundle(t, s, id, bundleitemutils.BundleSlug("slug"+strconv.Itoa(i)), "b", true)
	}

	pageSize := 7
	token := ""
	collected := 0
	for {
		resp, err := s.ListToolBundles(t.Context(), &spec.ListToolBundlesRequest{
			PageSize:        pageSize,
			PageToken:       token,
			BundleIDs:       ids,
			IncludeDisabled: true,
		})
		if err != nil {
			t.Fatalf("ListToolBundles() failed: %v", err)
		}
		collected += len(resp.Body.ToolBundles)

		if resp.Body.NextPageToken == nil || *resp.Body.NextPageToken == "" {
			break
		}
		token = *resp.Body.NextPageToken
	}

	if collected != len(ids) {
		t.Fatalf("pagination lost items: want %d, got %d", len(ids), collected)
	}
}

func TestToolPagination(t *testing.T) {
	s, clean := newTestToolStore(t)
	defer clean()

	mustPutToolBundle(t, s, "ub1", "slug1", "bundle", true)

	for i := range 23 {
		mustPutTool(t, s, "ub1",
			bundleitemutils.ItemSlug(fmt.Sprintf("t%02d", i)),
			"v1", "d", true)
	}

	const page = 6
	token := ""
	count := 0
	for {
		resp, err := s.ListTools(t.Context(), &spec.ListToolsRequest{
			RecommendedPageSize: page,
			PageToken:           token,
			BundleIDs:           []bundleitemutils.BundleID{"ub1"},
		})
		if err != nil {
			t.Fatalf("ListTools() failed: %v", err)
		}
		count += len(resp.Body.ToolListItems)
		if resp.Body.NextPageToken == nil || *resp.Body.NextPageToken == "" {
			break
		}
		token = *resp.Body.NextPageToken
	}
	if count != 23 {
		t.Fatalf("pagination lost items: expected 23 got %d", count)
	}
}

func TestToolSoftDeleteBehaviour(t *testing.T) {
	s, clean := newTestToolStore(t)
	defer clean()

	mustPutToolBundle(t, s, "b1", "slug", "Bundle", true)

	_, err := s.DeleteToolBundle(t.Context(), &spec.DeleteToolBundleRequest{BundleID: "b1"})
	if err != nil {
		t.Fatalf("DeleteToolBundle() failed: %v", err)
	}

	if _, err := s.getUserBundle("b1"); !errors.Is(err, spec.ErrBundleDeleting) {
		t.Fatalf("expected ErrBundleDeleting, got %v", err)
	}

	raw, _ := s.bundleStore.GetKey([]string{"bundles", "b1"})
	if mp, ok := raw.(map[string]any); ok {
		mp["softDeletedAt"] = time.Now().
			Add(-2 * softDeleteGraceTools).
			UTC().
			Format(time.RFC3339Nano)
		_ = s.bundleStore.SetKey([]string{"bundles", "b1"}, mp)
	}

	s.sweepSoftDeleted()

	if _, err := s.bundleStore.GetKey([]string{"bundles", "b1"}); err == nil {
		t.Fatalf("bundle should have been purged")
	}
}

func TestConcurrentToolPut(t *testing.T) {
	s, clean := newTestToolStore(t)
	defer clean()

	mustPutToolBundle(t, s, "b1", "slug", "Bundle", true)

	errCh := make(chan error, 2)
	go func() {
		_, err := s.PutTool(t.Context(), &spec.PutToolRequest{
			BundleID: "b1", ToolSlug: "concurrent", Version: "v1",
			Body: &spec.PutToolRequestBody{
				DisplayName:  "v1",
				IsEnabled:    true,
				UserCallable: true,
				LLMCallable:  true,

				ArgSchema: `{}`,

				Type:     spec.ToolTypeHTTP,
				HTTPImpl: dummyHTTPTool(),
			},
		})
		errCh <- err
	}()
	go func() {
		_, err := s.PutTool(t.Context(), &spec.PutToolRequest{
			BundleID: "b1", ToolSlug: "concurrent", Version: "v2",
			Body: &spec.PutToolRequestBody{
				DisplayName:  "v2",
				IsEnabled:    true,
				UserCallable: true,
				LLMCallable:  true,

				ArgSchema: `{}`,

				Type:     spec.ToolTypeHTTP,
				HTTPImpl: dummyHTTPTool(),
			},
		})
		errCh <- err
	}()

	if e1, e2 := <-errCh, <-errCh; e1 != nil || e2 != nil {
		t.Fatalf("parallel PutTool failed: %v / %v", e1, e2)
	}
}

func TestToolSlugVersionValidation(t *testing.T) {
	cases := []struct {
		slug  bundleitemutils.ItemSlug
		ver   bundleitemutils.ItemVersion
		valid bool
	}{
		{"abc", "v1", true},
		{"abc-def", "v1", true},
		{"", "v1", false},
		{"abc", "", false},
		{"bad.slug", "v1", false},
		{"abc", "v 1", false},
	}

	for _, c := range cases {
		errSlug := bundleitemutils.ValidateItemSlug(c.slug)
		errVer := bundleitemutils.ValidateItemVersion(c.ver)
		if c.valid && (errSlug != nil || errVer != nil) {
			t.Fatalf("expected valid slug/version (%s/%s) got errors %v %v",
				c.slug, c.ver, errSlug, errVer)
		}
		if !c.valid && errSlug == nil && errVer == nil {
			t.Fatalf("expected invalid for (%s/%s)", c.slug, c.ver)
		}
	}
}

func builtinToolStatistics(t *testing.T, s *ToolStore) (bundleCnt, toolCnt int) {
	t.Helper()
	if s.builtinData == nil {
		return 0, 0
	}
	b, tm, _ := s.builtinData.ListBuiltInToolData(t.Context())
	bundleCnt = len(b)
	for _, toolMap := range tm {
		toolCnt += len(toolMap)
	}
	return bundleCnt, toolCnt
}

func firstBuiltInTool(
	t *testing.T,
	s *ToolStore,
) (bid bundleitemutils.BundleID, slug bundleitemutils.ItemSlug, ver bundleitemutils.ItemVersion, ok bool) {
	t.Helper()
	if s.builtinData == nil {
		return bid, slug, ver, ok
	}
	_, tm, _ := s.builtinData.ListBuiltInToolData(t.Context())
	for bID, m := range tm {
		for _, tool := range m {
			return bID, tool.Slug, tool.Version, true
		}
	}
	return bid, slug, ver, ok
}

func mustPutTool(
	t *testing.T,
	s *ToolStore,
	bid bundleitemutils.BundleID,
	slug bundleitemutils.ItemSlug,
	ver bundleitemutils.ItemVersion,
	display string,
	enabled bool,
	tags ...string,
) {
	t.Helper()
	_, err := s.PutTool(t.Context(), &spec.PutToolRequest{
		BundleID: bid,
		ToolSlug: slug,
		Version:  ver,
		Body: &spec.PutToolRequestBody{
			DisplayName:  display,
			Description:  "test tool",
			IsEnabled:    enabled,
			Tags:         tags,
			UserCallable: true,
			LLMCallable:  true,

			ArgSchema: `{}`,

			Type:     spec.ToolTypeHTTP,
			HTTPImpl: dummyHTTPTool(),
		},
	})
	if err != nil {
		t.Fatalf("PutTool() failed: %v", err)
	}
}

func mustPutToolBundle(
	t *testing.T,
	s *ToolStore,
	id bundleitemutils.BundleID,
	slug bundleitemutils.BundleSlug,
	display string,
	enabled bool,
) {
	t.Helper()
	_, err := s.PutToolBundle(t.Context(), &spec.PutToolBundleRequest{
		BundleID: id,
		Body: &spec.PutToolBundleRequestBody{
			Slug:        slug,
			DisplayName: display,
			Description: "test bundle",
			IsEnabled:   enabled,
		},
	})
	if err != nil {
		t.Fatalf("PutToolBundle() failed: %v", err)
	}
}

func newTestToolStore(t *testing.T) (s *ToolStore, cleanup func()) {
	t.Helper()
	dir := t.TempDir()
	s, err := NewToolStore(dir)
	if err != nil {
		t.Fatalf("NewToolStore() failed: %v", err)
	}
	return s, func() { s.Close(); _ = os.RemoveAll(dir) }
}

func dummyHTTPTool() *spec.HTTPToolImpl {
	return &spec.HTTPToolImpl{
		Request: spec.HTTPRequest{
			Method:      "GET",
			URLTemplate: "https://example.com",
			TimeoutMS:   1000,
		},
		Response: spec.HTTPResponse{
			SuccessCodes: []int{200},
			ErrorMode:    "fail",
		},
	}
}
