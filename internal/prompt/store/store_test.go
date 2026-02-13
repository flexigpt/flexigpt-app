// The test-suite exercises the public behaviour of PromptTemplateStore after
// the recent refactor that
//   * removed the "latest/active" template convenience (version is now
//     mandatory),
//   * introduced "fluid" pagination (bundles first, then user objects),
//   * added an immutable built-in catalogue that can only be enabled / disabled.
//
// The tests are still table driven and only use Go’s std-lib.

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
	"github.com/flexigpt/flexigpt-app/internal/prompt/spec"
)

func TestBundleCRUD(t *testing.T) {
	cases := []struct {
		name      string
		id        bundleitemutils.BundleID
		slug      bundleitemutils.BundleSlug
		display   string
		enabled   bool
		wantError bool
		expectMsg string
	}{
		{"valid", "b1", "slug", "Bundle", true, false, ""},
		{"disabled", "b2", "disabled", "Bundle", false, false, ""},
		{"missing id", "", "s", "d", true, true, "required"},
		{"missing slug", "b3", "", "d", true, true, "required"},
		{"missing display", "b4", "s", "", true, true, "required"},
		{"bad slug dot", "b5", "bad.slug", "d", true, true, "invalid"},
		{"bad slug space", "b6", "bad slug", "d", true, true, "invalid"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, clean := newTestStore(t)
			defer clean()

			_, err := s.PutPromptBundle(t.Context(), &spec.PutPromptBundleRequest{
				BundleID: tc.id,
				Body: &spec.PutPromptBundleRequestBody{
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

func TestBuiltInBundleGuards(t *testing.T) {
	s, clean := newTestStore(t)
	defer clean()

	bid, slug, _, ok := firstBuiltIn(t, s)
	if !ok {
		t.Skip("library compiled without built-in catalogue - skipping test")
	}

	// 1. Modifying a built-in bundle must be rejected.
	if _, err := s.PutPromptBundle(t.Context(), &spec.PutPromptBundleRequest{
		BundleID: bid,
		Body: &spec.PutPromptBundleRequestBody{
			Slug:        bundleitemutils.BundleSlug(slug), // any value
			DisplayName: "illegal update",
			IsEnabled:   true,
		},
	}); !errors.Is(err, spec.ErrBuiltInReadOnly) {
		t.Fatalf("expected ErrBuiltInReadOnly, got %v", err)
	}

	// 2. Deleting a built-in bundle must be rejected.
	if _, err := s.DeletePromptBundle(t.Context(), &spec.DeletePromptBundleRequest{
		BundleID: bid,
	}); !errors.Is(err, spec.ErrBuiltInReadOnly) {
		t.Fatalf("expected ErrBuiltInReadOnly on delete, got %v", err)
	}

	// 3. But enabling / disabling is allowed.
	_, err := s.PatchPromptBundle(t.Context(), &spec.PatchPromptBundleRequest{
		BundleID: bid,
		Body: &spec.PatchPromptBundleRequestBody{
			IsEnabled: false,
		},
	})
	if err != nil {
		t.Fatalf("PatchPromptBundle() on built-in failed: %v", err)
	}
}

func TestTemplateCRUD(t *testing.T) {
	s, clean := newTestStore(t)
	defer clean()

	mustPutBundle(t, s, "b1", "slug1", "Bundle", true)

	cases := []struct {
		name      string
		bid       bundleitemutils.BundleID
		slug      bundleitemutils.ItemSlug
		ver       bundleitemutils.ItemVersion
		display   string
		wantError bool
		msg       string
	}{
		{"valid", "b1", "tpl", "v1", "display", false, ""},
		{"missing id", "", "s", "v1", "d", true, "required"},
		{"missing slug", "b1", "", "v1", "d", true, "required"},
		{"missing ver", "b1", "s", "", "d", true, "required"},
		{"bad slug", "b1", "a.b", "v1", "d", true, "invalid"},
		{"bad ver", "b1", "s", "v&1", "d", true, "invalid"},
		{"unknown bundle", "x", "s", "v1", "d", true, "not found"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := s.PutPromptTemplate(t.Context(), &spec.PutPromptTemplateRequest{
				BundleID:     tc.bid,
				TemplateSlug: tc.slug,
				Version:      tc.ver,
				Body: &spec.PutPromptTemplateRequestBody{
					DisplayName: tc.display,

					IsEnabled: true,
					Blocks: []spec.MessageBlock{{
						ID:      "1",
						Role:    spec.User,
						Content: "hello",
					}},
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

func TestTemplateVersionConflict(t *testing.T) {
	s, clean := newTestStore(t)
	defer clean()

	mustPutBundle(t, s, "b1", "slug", "Bundle", true)
	mustPutTemplate(t, s, "b1", "tpl", "v1", "d", true)

	_, err := s.PutPromptTemplate(t.Context(), &spec.PutPromptTemplateRequest{
		BundleID:     "b1",
		TemplateSlug: "tpl",
		Version:      "v1",
		Body: &spec.PutPromptTemplateRequestBody{
			DisplayName: "dup",
			IsEnabled:   true,
			Blocks: []spec.MessageBlock{{
				ID:      "1",
				Role:    spec.User,
				Content: "hello",
			}},
		},
	})

	if !errors.Is(err, spec.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestTemplateDisabledBundleGuard(t *testing.T) {
	s, clean := newTestStore(t)
	defer clean()

	mustPutBundle(t, s, "b1", "slug", "Bundle", false)

	_, err := s.PutPromptTemplate(t.Context(), &spec.PutPromptTemplateRequest{
		BundleID:     "b1",
		TemplateSlug: "tpl",
		Version:      "v1",
		Body: &spec.PutPromptTemplateRequestBody{
			DisplayName: "d",
			IsEnabled:   true,
			Blocks: []spec.MessageBlock{{
				ID:      "1",
				Role:    spec.User,
				Content: "hello",
			}},
		},
	})
	if !errors.Is(err, spec.ErrBundleDisabled) {
		t.Fatalf("expected ErrBundleDisabled, got %v", err)
	}
}

func TestTemplateMultiVersionExact(t *testing.T) {
	s, clean := newTestStore(t)
	defer clean()

	mustPutBundle(t, s, "b1", "slug", "Bundle", true)

	vers := []string{"v1", "v2", "v3"}
	for _, v := range vers {
		mustPutTemplate(t, s, "b1", "tpl", bundleitemutils.ItemVersion(v), "disp "+v, true)
		time.Sleep(5 * time.Millisecond)
	}

	for _, v := range vers {
		resp, err := s.GetPromptTemplate(t.Context(), &spec.GetPromptTemplateRequest{
			BundleID:     "b1",
			TemplateSlug: "tpl",
			Version:      bundleitemutils.ItemVersion(v),
		})
		if err != nil {
			t.Fatalf("GetPromptTemplate(%s) failed: %v", v, err)
		}
		if resp.Body.Version != bundleitemutils.ItemVersion(v) {
			t.Fatalf("expected version %s, got %s", v, resp.Body.Version)
		}
	}

	// Omitted version must now fail.
	if _, err := s.GetPromptTemplate(t.Context(), &spec.GetPromptTemplateRequest{
		BundleID:     "b1",
		TemplateSlug: "tpl",
	}); !errors.Is(err, spec.ErrInvalidRequest) {
		t.Fatalf("expected ErrInvalidRequest for missing version, got %v", err)
	}
}

func TestBuiltInTemplateGuards(t *testing.T) {
	s, clean := newTestStore(t)
	defer clean()

	bid, slug, ver, ok := firstBuiltIn(t, s)
	if !ok {
		t.Skip("no built-in catalogue present")
	}

	// Create a template in a built-in bundle - forbidden.
	_, err := s.PutPromptTemplate(t.Context(), &spec.PutPromptTemplateRequest{
		BundleID:     bid,
		TemplateSlug: slug,
		Version:      "v-new",
		Body: &spec.PutPromptTemplateRequestBody{
			DisplayName: "illegal",
			IsEnabled:   true,
			Blocks: []spec.MessageBlock{{
				ID:      "1",
				Role:    spec.User,
				Content: "hello",
			}},
		},
	})
	if !errors.Is(err, spec.ErrBuiltInReadOnly) {
		t.Fatalf("expected ErrBuiltInReadOnly, got %v", err)
	}

	// Deleting a built-in template must fail.
	_, err = s.DeletePromptTemplate(t.Context(), &spec.DeletePromptTemplateRequest{
		BundleID:     bid,
		TemplateSlug: slug,
		Version:      ver,
	})
	if !errors.Is(err, spec.ErrBuiltInReadOnly) {
		t.Fatalf("expected ErrBuiltInReadOnly on delete, got %v", err)
	}

	// Patch (toggle enabled) is allowed.
	_, err = s.PatchPromptTemplate(t.Context(), &spec.PatchPromptTemplateRequest{
		BundleID:     bid,
		TemplateSlug: slug,
		Version:      ver,
		Body: &spec.PatchPromptTemplateRequestBody{
			IsEnabled: false,
		},
	})
	if err != nil {
		t.Fatalf("PatchPromptTemplate() for built-in failed: %v", err)
	}
}

func TestBundleListFiltering(t *testing.T) {
	s, clean := newTestStore(t)
	defer clean()

	builtInCnt, _ := builtinStatistics(t, s)

	// User data.
	mustPutBundle(t, s, "ub1", "slug1", "Bundle1", true)
	mustPutBundle(t, s, "ub2", "slug2", "Bundle2", false)

	tests := []struct {
		name            string
		includeDisabled bool
		filterIDs       []bundleitemutils.BundleID
		expectUser      int
	}{
		{"enabledOnly", false, nil, 1},
		{"allUsers", true, nil, 2},
		{"filterUser", true, []bundleitemutils.BundleID{"ub1"}, 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := s.ListPromptBundles(t.Context(), &spec.ListPromptBundlesRequest{
				IncludeDisabled: tc.includeDisabled,
				BundleIDs:       tc.filterIDs,
			})
			if err != nil {
				t.Fatalf("ListPromptBundles() failed: %v", err)
			}

			got := len(resp.Body.PromptBundles)
			want := tc.expectUser
			if tc.filterIDs == nil { // built-ins included
				want += builtInCnt
			}
			if got != want {
				t.Fatalf("expected %d bundles, got %d", want, got)
			}

			// Verify user bundles appear when expected.
			if tc.expectUser > 0 {
				find := func(id bundleitemutils.BundleID) bool {
					for _, b := range resp.Body.PromptBundles {
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

func TestTemplateListFiltering(t *testing.T) {
	s, clean := newTestStore(t)
	defer clean()

	mustPutBundle(t, s, "ub1", "slug1", "UserBundle", true)

	mustPutTemplate(t, s, "ub1", "t1", "v1", "t1", true, "tag1")
	mustPutTemplate(t, s, "ub1", "t2", "v1", "t2", false, "tag1", "tag2")

	tests := []struct {
		name            string
		includeDisabled bool
		tags            []string
		expect          int
	}{
		{"enabledOnly", false, nil, 1},
		{"withDisabled", true, nil, 2},
		{"tagFilter", true, []string{"tag2"}, 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := s.ListPromptTemplates(t.Context(), &spec.ListPromptTemplatesRequest{
				BundleIDs: []bundleitemutils.BundleID{
					"ub1",
				}, // exclude built-ins for determinism
				Tags:            tc.tags,
				IncludeDisabled: tc.includeDisabled,
			})
			if err != nil {
				t.Fatalf("ListPromptTemplates() failed: %v", err)
			}
			if len(resp.Body.PromptTemplateListItems) != tc.expect {
				t.Fatalf("expected %d items, got %d",
					tc.expect, len(resp.Body.PromptTemplateListItems))
			}
		})
	}
}

func TestBundlePagination(t *testing.T) {
	s, clean := newTestStore(t)
	defer clean()

	// User bundles.
	ids := make([]bundleitemutils.BundleID, 0, 30)
	for i := range 30 {
		id := bundleitemutils.BundleID(fmt.Sprintf("u%02d", i))
		ids = append(ids, id)
		mustPutBundle(t, s, id, bundleitemutils.BundleSlug("slug"+strconv.Itoa(i)), "b", true)
	}

	pageSize := 7
	token := ""
	collected := 0
	for {
		resp, err := s.ListPromptBundles(t.Context(), &spec.ListPromptBundlesRequest{
			PageSize:        pageSize,
			PageToken:       token,
			BundleIDs:       ids, // user bundles only
			IncludeDisabled: true,
		})
		if err != nil {
			t.Fatalf("ListPromptBundles() failed: %v", err)
		}
		collected += len(resp.Body.PromptBundles)

		if resp.Body.NextPageToken == nil || *resp.Body.NextPageToken == "" {
			break
		}
		token = *resp.Body.NextPageToken
	}

	if collected != len(ids) {
		t.Fatalf("pagination lost items: want %d, got %d", len(ids), collected)
	}
}

func TestTemplatePagination(t *testing.T) {
	s, clean := newTestStore(t)
	defer clean()

	mustPutBundle(t, s, "ub1", "slug1", "bundle", true)

	for i := range 23 {
		mustPutTemplate(t, s, "ub1",
			bundleitemutils.ItemSlug(fmt.Sprintf("t%02d", i)),
			"v1", "d", true)
	}

	const page = 6
	token := ""
	count := 0
	for {
		resp, err := s.ListPromptTemplates(t.Context(), &spec.ListPromptTemplatesRequest{
			RecommendedPageSize: page,
			PageToken:           token,
			BundleIDs:           []bundleitemutils.BundleID{"ub1"},
		})
		if err != nil {
			t.Fatalf("ListPromptTemplates() failed: %v", err)
		}
		count += len(resp.Body.PromptTemplateListItems)
		if resp.Body.NextPageToken == nil || *resp.Body.NextPageToken == "" {
			break
		}
		token = *resp.Body.NextPageToken
	}
	if count != 23 {
		t.Fatalf("pagination lost items: expected 23 got %d", count)
	}
}

func TestSoftDeleteBehaviour(t *testing.T) {
	s, clean := newTestStore(t)
	defer clean()

	mustPutBundle(t, s, "b1", "slug", "Bundle", true)

	_, err := s.DeletePromptBundle(t.Context(), &spec.DeletePromptBundleRequest{BundleID: "b1"})
	if err != nil {
		t.Fatalf("DeletePromptBundle() failed: %v", err)
	}

	// Verify state flag.
	if _, err := s.getUserBundle("b1"); !errors.Is(err, spec.ErrBundleDeleting) {
		t.Fatalf("expected ErrBundleDeleting, got %v", err)
	}

	// Fake grace-period expiration.
	raw, _ := s.bundleStore.GetKey([]string{"bundles", "b1"})
	if mp, ok := raw.(map[string]any); ok {
		mp["softDeletedAt"] = time.Now().Add(-2 * softDeleteGrace).UTC().Format(time.RFC3339Nano)
		_ = s.bundleStore.SetKey([]string{"bundles", "b1"}, mp)
	}

	s.sweepSoftDeleted()

	if _, err := s.bundleStore.GetKey([]string{"bundles", "b1"}); err == nil {
		t.Fatalf("bundle should have been purged")
	}
}

func TestConcurrentTemplatePut(t *testing.T) {
	s, clean := newTestStore(t)
	defer clean()

	mustPutBundle(t, s, "b1", "slug", "Bundle", true)

	errCh := make(chan error, 2)
	go func() {
		_, err := s.PutPromptTemplate(t.Context(), &spec.PutPromptTemplateRequest{
			BundleID:     "b1",
			TemplateSlug: "concurrent",
			Version:      "v1",
			Body: &spec.PutPromptTemplateRequestBody{
				DisplayName: "v1",
				IsEnabled:   true,
				Blocks: []spec.MessageBlock{{
					ID:      "1",
					Role:    spec.User,
					Content: "hello",
				}},
			},
		})
		errCh <- err
	}()
	go func() {
		_, err := s.PutPromptTemplate(t.Context(), &spec.PutPromptTemplateRequest{
			BundleID:     "b1",
			TemplateSlug: "concurrent",
			Version:      "v2",
			Body: &spec.PutPromptTemplateRequestBody{
				DisplayName: "v2",
				IsEnabled:   true,
				Blocks: []spec.MessageBlock{{
					ID:      "1",
					Role:    spec.User,
					Content: "hello",
				}},
			},
		})
		errCh <- err
	}()

	if e1, e2 := <-errCh, <-errCh; e1 != nil || e2 != nil {
		t.Fatalf("parallel PutPromptTemplate failed: %v / %v", e1, e2)
	}
}

func TestSlugVersionValidation(t *testing.T) {
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

// builtinStatistics returns how many built-in bundles / templates are embedded
// in the library.  The helper is used to keep assertions robust, independent
// from the actual catalogue size.
func builtinStatistics(t *testing.T, s *PromptTemplateStore) (bundleCnt, templateCnt int) {
	t.Helper()
	if s.builtinData == nil {
		return 0, 0
	}
	b, tm, _ := s.builtinData.ListBuiltInData(t.Context())
	bundleCnt = len(b)
	for _, tmplMap := range tm {
		templateCnt += len(tmplMap)
	}
	return bundleCnt, templateCnt
}

// located in that bundle.  Ok==false if no catalogue is available.
func firstBuiltIn(
	t *testing.T,
	s *PromptTemplateStore,
) (bid bundleitemutils.BundleID, slug bundleitemutils.ItemSlug, ver bundleitemutils.ItemVersion, ok bool) {
	t.Helper()
	if s.builtinData == nil {
		return bid, slug, ver, ok
	}
	_, tmplM, _ := s.builtinData.ListBuiltInData(t.Context())
	for bID, m := range tmplM {
		for _, tpl := range m {
			return bID, tpl.Slug, tpl.Version, true
		}
	}
	return bid, slug, ver, ok
}

// mustPutTemplate creates a template version or fails the test.
func mustPutTemplate(
	t *testing.T,
	s *PromptTemplateStore,
	bid bundleitemutils.BundleID,
	slug bundleitemutils.ItemSlug,
	ver bundleitemutils.ItemVersion,
	display string,
	enabled bool,
	tags ...string,
) {
	t.Helper()

	_, err := s.PutPromptTemplate(t.Context(), &spec.PutPromptTemplateRequest{
		BundleID:     bid,
		TemplateSlug: slug,
		Version:      ver,
		Body: &spec.PutPromptTemplateRequestBody{
			DisplayName: display,
			Description: "test template",
			IsEnabled:   enabled,
			Blocks: []spec.MessageBlock{{
				ID:      "b1",
				Role:    spec.User,
				Content: "hello",
			}},
			Tags: tags,
		},
	})
	if err != nil {
		t.Fatalf("PutPromptTemplate() failed: %v", err)
	}
}

// mustPutBundle creates a bundle or fails the test.
func mustPutBundle(
	t *testing.T,
	s *PromptTemplateStore,
	id bundleitemutils.BundleID,
	slug bundleitemutils.BundleSlug,
	display string,
	enabled bool,
) {
	t.Helper()

	_, err := s.PutPromptBundle(t.Context(), &spec.PutPromptBundleRequest{
		BundleID: id,
		Body: &spec.PutPromptBundleRequestBody{
			Slug:        slug,
			DisplayName: display,
			Description: "test bundle",
			IsEnabled:   enabled,
		},
	})
	if err != nil {
		t.Fatalf("PutPromptBundle() failed: %v", err)
	}
}

// newTestStore creates a store rooted in a temporary directory and returns a
// cleanup function that closes the store and removes the directory.
func newTestStore(t *testing.T) (s *PromptTemplateStore, cleanup func()) {
	t.Helper()

	dir := t.TempDir()
	s, err := NewPromptTemplateStore(dir)
	if err != nil {
		t.Fatalf("NewPromptTemplateStore() failed: %v", err)
	}
	return s, func() { s.Close(); _ = os.RemoveAll(dir) }
}
