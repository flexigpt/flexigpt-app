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

const (
	testBundleGroup = "bundles"

	testRequiredMsg = "required"
	testInvalidMsg  = "invalid"

	testBundleID1 = "b1"
	testBundleID2 = "b2"
	testBundleID3 = "b3"
	testBundleID4 = "b4"
	testBundleID5 = "b5"
	testBundleID6 = "b6"
	testBundleIDX = "x"

	testUserBundleID1 = "ub1"
	testUserBundleID2 = "ub2"

	testShortSlug = "s"

	testBundleSlug         = "slug"
	testBundleSlug1        = "slug1"
	testBundleSlugDisabled = "disabled"

	testBundleDisplay        = "Bundle"
	testBundleDisplay1       = "Bundle1"
	testBundleDisplay2       = "Bundle2"
	testUserBundleDisplay    = "UserBundle"
	testIllegalUpdateDisplay = "illegal update"

	testToolSlug           = "tool"
	testToolSlugConcurrent = "concurrent"
	testToolSlugT1         = "t1"
	testToolSlugT2         = "t2"

	testVersion1   = "v1"
	testVersion2   = "v2"
	testVersion3   = "v3"
	testVersionNew = "v-new"

	testDisplay      = "display"
	testDisplayShort = "d"
	testDisplayDup   = "dup"

	testTag1 = "tag1"
	testTag2 = "tag2"

	testABC               = "abc"
	testABCDef            = "abc-def"
	testBadSlugDot        = "bad.slug"
	testBadSlugSpace      = "bad slug"
	testBadVersionSpace   = "v 1"
	testArgSchema         = `{}`
	testBundleDescription = "test bundle"
	testToolDescription   = "test tool"
	testHTTPMethod        = "GET"
	testURLTemplate       = "https://example.com"
	testErrorModeFail     = "fail"
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
		{"Valid", testBundleID1, testBundleSlug, testBundleDisplay, true, false, ""},
		{"Disabled", testBundleID2, testBundleSlugDisabled, testBundleDisplay, false, false, ""},
		{"MissingID", "", testShortSlug, testDisplayShort, true, true, testRequiredMsg},
		{"MissingSlug", testBundleID3, "", testDisplayShort, true, true, testRequiredMsg},
		{"MissingDisplay", testBundleID4, testShortSlug, "", true, true, testRequiredMsg},
		{"BadSlugDot", testBundleID5, testBadSlugDot, testDisplayShort, true, true, testInvalidMsg},
		{"BadSlugSpace", testBundleID6, testBadSlugSpace, testDisplayShort, true, true, testInvalidMsg},
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
			Slug:        bundleitemutils.BundleSlug(string(slug)),
			DisplayName: testIllegalUpdateDisplay,
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

	mustPutToolBundle(t, s, testBundleID1, testBundleSlug1, testBundleDisplay, true)

	cases := []struct {
		name      string
		bid       bundleitemutils.BundleID
		slug      bundleitemutils.ItemSlug
		ver       bundleitemutils.ItemVersion
		display   string
		wantError bool
		msg       string
	}{
		{"Valid", testBundleID1, testToolSlug, testVersion1, testDisplay, false, ""},
		{"MissingID", "", testShortSlug, testVersion1, testDisplayShort, true, testRequiredMsg},
		{"MissingSlug", testBundleID1, "", testVersion1, testDisplayShort, true, testRequiredMsg},
		{"MissingVer", testBundleID1, testShortSlug, "", testDisplayShort, true, testRequiredMsg},
		{"BadSlug", testBundleID1, testBadSlugDot, testVersion1, testDisplayShort, true, testInvalidMsg},
		{"BadVer", testBundleID1, testShortSlug, testBadVersionSpace, testDisplayShort, true, testInvalidMsg},
		{"UnknownBundle", testBundleIDX, testShortSlug, testVersion1, testDisplayShort, true, "not found"},
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

					ArgSchema: testArgSchema,

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

	mustPutToolBundle(t, s, testBundleID1, testBundleSlug, testBundleDisplay, true)
	mustPutTool(t, s, testBundleID1, testToolSlug, testVersion1, testDisplayShort, true)

	_, err := s.PutTool(t.Context(), &spec.PutToolRequest{
		BundleID: testBundleID1, ToolSlug: testToolSlug, Version: testVersion1,
		Body: &spec.PutToolRequestBody{
			DisplayName:  testDisplayDup,
			IsEnabled:    true,
			UserCallable: true,
			LLMCallable:  true,

			ArgSchema: testArgSchema,

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

	mustPutToolBundle(t, s, testBundleID1, testBundleSlug, testBundleDisplay, false)

	_, err := s.PutTool(t.Context(), &spec.PutToolRequest{
		BundleID: testBundleID1, ToolSlug: testToolSlug, Version: testVersion1,
		Body: &spec.PutToolRequestBody{
			DisplayName:  testDisplayShort,
			IsEnabled:    true,
			UserCallable: true,
			LLMCallable:  true,

			ArgSchema: testArgSchema,

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

	mustPutToolBundle(t, s, testBundleID1, testBundleSlug, testBundleDisplay, true)

	vers := []string{testVersion1, testVersion2, testVersion3}
	for _, v := range vers {
		mustPutTool(t, s, testBundleID1, testToolSlug, bundleitemutils.ItemVersion(v), "disp "+v, true)
		time.Sleep(5 * time.Millisecond)
	}

	for _, v := range vers {
		resp, err := s.GetTool(t.Context(), &spec.GetToolRequest{
			BundleID: testBundleID1, ToolSlug: testToolSlug, Version: bundleitemutils.ItemVersion(v),
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
		BundleID: testBundleID1, ToolSlug: testToolSlug, Version: "",
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
		BundleID: bid, ToolSlug: slug, Version: testVersionNew,
		Body: &spec.PutToolRequestBody{
			DisplayName:  "illegal",
			IsEnabled:    true,
			UserCallable: true,
			LLMCallable:  true,

			ArgSchema: testArgSchema,

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

	mustPutToolBundle(t, s, testUserBundleID1, testBundleSlug1, testBundleDisplay1, true)
	mustPutToolBundle(t, s, testUserBundleID2, testBundleSlug, testBundleDisplay2, false)

	tests := []struct {
		name            string
		includeDisabled bool
		filterIDs       []bundleitemutils.BundleID
		expectUser      int
	}{
		{"EnabledOnly", false, nil, 1},
		{"AllUsers", true, nil, 2},
		{"FilterUser", true, []bundleitemutils.BundleID{testUserBundleID1}, 1},
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

	mustPutToolBundle(t, s, testUserBundleID1, testBundleSlug1, testUserBundleDisplay, true)
	mustPutTool(t, s, testUserBundleID1, testToolSlugT1, testVersion1, testToolSlugT1, true, testTag1)
	mustPutTool(t, s, testUserBundleID1, testToolSlugT2, testVersion1, testToolSlugT2, false, testTag1, testTag2)

	tests := []struct {
		name            string
		includeDisabled bool
		tags            []string
		expect          int
	}{
		{"EnabledOnly", false, nil, 1},
		{"WithDisabled", true, nil, 2},
		{"TagFilter", true, []string{testTag2}, 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := s.ListTools(t.Context(), &spec.ListToolsRequest{
				BundleIDs:       []bundleitemutils.BundleID{testUserBundleID1},
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
		mustPutToolBundle(t, s, id, bundleitemutils.BundleSlug(testBundleSlug+strconv.Itoa(i)), "b", true)
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

	mustPutToolBundle(t, s, testUserBundleID1, testBundleSlug1, "bundle", true)

	for i := range 23 {
		mustPutTool(t, s, testUserBundleID1,
			bundleitemutils.ItemSlug(fmt.Sprintf("t%02d", i)),
			testVersion1, testDisplayShort, true)
	}

	const page = 6
	token := ""
	count := 0
	for {
		resp, err := s.ListTools(t.Context(), &spec.ListToolsRequest{
			RecommendedPageSize: page,
			PageToken:           token,
			BundleIDs:           []bundleitemutils.BundleID{testUserBundleID1},
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

	mustPutToolBundle(t, s, testBundleID1, testBundleSlug, testBundleDisplay, true)

	_, err := s.DeleteToolBundle(t.Context(), &spec.DeleteToolBundleRequest{BundleID: testBundleID1})
	if err != nil {
		t.Fatalf("DeleteToolBundle() failed: %v", err)
	}

	if _, err := s.getUserBundle(testBundleID1); !errors.Is(err, spec.ErrBundleDeleting) {
		t.Fatalf("expected ErrBundleDeleting, got %v", err)
	}

	raw, _ := s.bundleStore.GetKey([]string{testBundleGroup, testBundleID1})
	if mp, ok := raw.(map[string]any); ok {
		mp["softDeletedAt"] = time.Now().
			Add(-2 * softDeleteGraceTools).
			UTC().
			Format(time.RFC3339Nano)
		_ = s.bundleStore.SetKey([]string{testBundleGroup, testBundleID1}, mp)
	}

	s.sweepSoftDeleted()

	if _, err := s.bundleStore.GetKey([]string{testBundleGroup, testBundleID1}); err == nil {
		t.Fatalf("bundle should have been purged")
	}
}

func TestConcurrentToolPut(t *testing.T) {
	s, clean := newTestToolStore(t)
	defer clean()

	mustPutToolBundle(t, s, testBundleID1, testBundleSlug, testBundleDisplay, true)

	errCh := make(chan error, 2)
	go func() {
		_, err := s.PutTool(t.Context(), &spec.PutToolRequest{
			BundleID: testBundleID1, ToolSlug: testToolSlugConcurrent, Version: testVersion1,
			Body: &spec.PutToolRequestBody{
				DisplayName:  testVersion1,
				IsEnabled:    true,
				UserCallable: true,
				LLMCallable:  true,

				ArgSchema: testArgSchema,

				Type:     spec.ToolTypeHTTP,
				HTTPImpl: dummyHTTPTool(),
			},
		})
		errCh <- err
	}()
	go func() {
		_, err := s.PutTool(t.Context(), &spec.PutToolRequest{
			BundleID: testBundleID1, ToolSlug: testToolSlugConcurrent, Version: testVersion2,
			Body: &spec.PutToolRequestBody{
				DisplayName:  testVersion2,
				IsEnabled:    true,
				UserCallable: true,
				LLMCallable:  true,

				ArgSchema: testArgSchema,

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
		{testABC, testVersion1, true},
		{testABC + "-" + "def", testVersion1, true},
		{"", testVersion1, false},
		{testABC, "", false},
		{testBadSlugDot, testVersion1, false},
		{testABC, testBadVersionSpace, false},
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
			Description:  testToolDescription,
			IsEnabled:    enabled,
			Tags:         tags,
			UserCallable: true,
			LLMCallable:  true,

			ArgSchema: testArgSchema,

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
			Description: testBundleDescription,
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
			Method:      testHTTPMethod,
			URLTemplate: testURLTemplate,
			TimeoutMS:   1000,
		},
		Response: spec.HTTPResponse{
			SuccessCodes: []int{200},
			ErrorMode:    testErrorModeFail,
		},
	}
}
