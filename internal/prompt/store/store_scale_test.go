// Scale-tests for PromptTemplateStore (lots of bundles / lots of templates).
// Updated for the current behaviour:
//
//   • template "active/latest" behaviour was removed - we now check
//     presence per slug instead of "latest wins".
//   • built-in catalogue exists - all scale checks explicitly restrict the
//     BundleIDs filter to the bundles that are created in-test so that the
//     reference numbers stay deterministic.
//   • helper loops have been fixed (no more "range <int>").

package store

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/prompt/spec"
)

func TestScale_LotsOfBundles(t *testing.T) {
	const (
		nBundles      = 256
		nonEmptyEvery = 10 // every 10-th bundle gets one template
		deleteEvery   = 5  // request delete on every 5-th bundle
		patchEvery    = 2  // toggle enabled every 2-nd bundle
		concurrency   = 20
	)

	s, clean := newTestStore(t)
	defer clean()
	ctx := t.Context()
	errCh := make(chan error, 8_000)

	createJobs := make(chan int, nBundles)
	for i := range nBundles {
		createJobs <- i
	}
	close(createJobs)

	var wg sync.WaitGroup
	wg.Add(concurrency)
	for range concurrency {
		go func() {
			defer wg.Done()
			for i := range createJobs {
				bID := bundleitemutils.BundleID("bundle-" + strconv.Itoa(i))

				_, err := s.PutPromptBundle(ctx, &spec.PutPromptBundleRequest{
					BundleID: bID,
					Body: &spec.PutPromptBundleRequestBody{
						Slug:        bundleitemutils.BundleSlug("slug-" + strconv.Itoa(i)),
						DisplayName: "Bundle " + strconv.Itoa(i),
						IsEnabled:   i%3 == 0, // enabled for 0,3,6,...
					},
				})
				if err != nil {
					errCh <- err
					continue
				}

				// One template?.
				if i%nonEmptyEvery == 0 {
					_, err = s.PutPromptTemplate(ctx, &spec.PutPromptTemplateRequest{
						BundleID:     bID,
						TemplateSlug: "t",
						Version:      "v1",
						Body: &spec.PutPromptTemplateRequestBody{
							Kind:        spec.PromptTemplateKindGeneric,
							IsResolved:  true,
							DisplayName: "dummy",
							IsEnabled:   true,
							Blocks: []spec.MessageBlock{{
								ID:      "1",
								Role:    spec.User,
								Content: "hi",
							}},
						},
					})
					if err != nil && !errors.Is(err, spec.ErrBundleDisabled) {
						errCh <- err
					}
				}
			}
		}()
	}
	wg.Wait()

	modJobs := make(chan int, nBundles)
	for i := range nBundles {
		modJobs <- i
	}
	close(modJobs)

	wg.Add(concurrency)
	for range concurrency {
		go func() {
			defer wg.Done()
			for i := range modJobs {
				bID := bundleitemutils.BundleID("bundle-" + strconv.Itoa(i))

				if i%patchEvery == 0 {
					_, err := s.PatchPromptBundle(ctx, &spec.PatchPromptBundleRequest{
						BundleID: bID,
						Body: &spec.PatchPromptBundleRequestBody{
							IsEnabled: i%3 != 0, // invert initial state
						},
					})
					if err != nil {
						errCh <- err
					}
				}

				if i%deleteEvery == 0 {
					_, err := s.DeletePromptBundle(ctx, &spec.DeletePromptBundleRequest{
						BundleID: bID,
					})

					nonEmpty := (i%nonEmptyEvery == 0) && (i%3 == 0) // bundle had template
					if nonEmpty && err == nil {
						errCh <- fmt.Errorf("expected delete to fail for non-empty bundle %s", bID)
					}
					if !nonEmpty && err != nil {
						errCh <- fmt.Errorf("delete empty bundle %s failed: %w", bID, err)
					}
				}
			}
		}()
	}
	wg.Wait()

	resp, err := s.ListPromptBundles(ctx, &spec.ListPromptBundlesRequest{})
	if err != nil {
		errCh <- err
	} else {
		for _, b := range resp.Body.PromptBundles {
			if !b.IsEnabled {
				errCh <- fmt.Errorf("disabled bundle %s in enabled-only list", b.ID)
			}
		}
	}

	close(errCh)
	drainErrors(t, errCh)
}

// Lots of templates.
func TestScale_LotsOfTemplates(t *testing.T) {
	const (
		nBundles    = 3
		nSlugs      = 120
		nVersions   = 4
		concurrency = 40
	)

	s, clean := newTestStore(t)
	defer clean()
	ctx := t.Context()
	errCh := make(chan error, 20_000)

	userBundles := make([]bundleitemutils.BundleID, 0, nBundles)
	for i := range nBundles {
		bID := bundleitemutils.BundleID(fmt.Sprintf("b%d", i))
		userBundles = append(userBundles, bID)

		_, err := s.PutPromptBundle(ctx, &spec.PutPromptBundleRequest{
			BundleID: bID,
			Body: &spec.PutPromptBundleRequestBody{
				Slug:        bundleitemutils.BundleSlug(fmt.Sprintf("slug-%s", bID)),
				DisplayName: fmt.Sprintf("Bundle %d", i),
				IsEnabled:   true,
			},
		})
		if err != nil {
			t.Fatalf("create bundle: %v", err)
		}
	}

	type job struct{ b, s, v int }
	insertJobs := make(chan job, nBundles*nSlugs*nVersions)
	for b := range nBundles {
		for sIdx := range nSlugs {
			for v := 1; v <= nVersions; v++ {
				insertJobs <- job{b, sIdx, v}
			}
		}
	}
	close(insertJobs)

	var wg sync.WaitGroup
	wg.Add(concurrency)
	for range concurrency {
		go func() {
			defer wg.Done()
			for j := range insertJobs {
				_, err := s.PutPromptTemplate(ctx, &spec.PutPromptTemplateRequest{
					BundleID:     bundleitemutils.BundleID(fmt.Sprintf("b%d", j.b)),
					TemplateSlug: bundleitemutils.ItemSlug(fmt.Sprintf("tpl-%d", j.s)),
					Version:      bundleitemutils.ItemVersion(fmt.Sprintf("v%d", j.v)),
					Body: &spec.PutPromptTemplateRequestBody{
						Kind:        spec.PromptTemplateKindGeneric,
						IsResolved:  true,
						DisplayName: "T",
						IsEnabled:   true,
						Blocks: []spec.MessageBlock{{
							ID:      "1",
							Role:    spec.User,
							Content: "hi",
						}},
					},
				})
				if err != nil {
					errCh <- err
				}
				time.Sleep(500 * time.Microsecond) // help ModifiedAt monotonicity
			}
		}()
	}
	wg.Wait()

	base := spec.ListPromptTemplatesRequest{
		BundleIDs: userBundles,
	}
	all, err := collectTemplates(ctx, s, base, 100)
	if err != nil {
		errCh <- err
	} else if got, want := uniqCntSlug(all), nBundles*nSlugs; got != want {
		errCh <- fmt.Errorf("slug uniqueness mismatch got=%d want=%d", got, want)
	}

	modJobs := make(chan job, nBundles*nSlugs)
	for b := range nBundles {
		for sIdx := range nSlugs {
			if sIdx%2 == 0 { // disable v4
				modJobs <- job{b, sIdx, 4}
			}
			if sIdx%3 == 0 { // delete v1
				modJobs <- job{b, sIdx, 1}
			}
		}
	}
	close(modJobs)

	wg.Add(concurrency)
	for range concurrency {
		go func() {
			defer wg.Done()
			for j := range modJobs {
				bID := bundleitemutils.BundleID(fmt.Sprintf("b%d", j.b))
				slug := bundleitemutils.ItemSlug(fmt.Sprintf("tpl-%d", j.s))

				if j.v == 4 { // patch disable
					_, err := s.PatchPromptTemplate(ctx, &spec.PatchPromptTemplateRequest{
						BundleID:     bID,
						TemplateSlug: slug,
						Version:      "v4",
						Body: &spec.PatchPromptTemplateRequestBody{
							IsEnabled: false,
						},
					})
					if err != nil {
						errCh <- err
					}
				} else { // delete v1
					_, err := s.DeletePromptTemplate(ctx, &spec.DeletePromptTemplateRequest{
						BundleID:     bID,
						TemplateSlug: slug,
						Version:      "v1",
					})
					if err != nil {
						errCh <- err
					}
				}
			}
		}()
	}
	wg.Wait()

	base = spec.ListPromptTemplatesRequest{
		IncludeDisabled: true,
		BundleIDs:       userBundles,
	}
	all, err = collectTemplates(ctx, s, base, 150)
	if err != nil {
		errCh <- err
	} else {
		total := nBundles * nSlugs * nVersions
		deleted := nBundles * (nSlugs / 3) // v1 removed for every 3-rd slug
		want := total - deleted
		if got := uniqCntVersion(all); got != want {
			errCh <- fmt.Errorf("version uniqueness mismatch got=%d want=%d", got, want)
		}
	}

	base = spec.ListPromptTemplatesRequest{
		BundleIDs:           userBundles,
		RecommendedPageSize: 70,
	}
	all, err = collectTemplates(ctx, s, base, 70)
	if err != nil {
		errCh <- err
	} else if got, want := uniqCntSlug(all), nBundles*nSlugs; got != want {
		errCh <- fmt.Errorf("pagination slug mismatch got=%d want=%d", got, want)
	}

	close(errCh)
	drainErrors(t, errCh)
}

// collectTemplates pages through ListPromptTemplates until NextPageToken=="".
func collectTemplates(
	ctx context.Context,
	s *PromptTemplateStore,
	baseReq spec.ListPromptTemplatesRequest,
	pageHint int,
) ([]spec.PromptTemplateListItem, error) {
	var (
		out   []spec.PromptTemplateListItem
		token string
	)

	for {
		req := baseReq // copy
		req.RecommendedPageSize = pageHint
		req.PageToken = token

		resp, err := s.ListPromptTemplates(ctx, &req)
		if err != nil {
			return nil, err
		}
		out = append(out, resp.Body.PromptTemplateListItems...)

		if resp.Body.NextPageToken == nil || *resp.Body.NextPageToken == "" {
			break
		}
		token = *resp.Body.NextPageToken
	}
	return out, nil
}

func uniqCntSlug(list []spec.PromptTemplateListItem) int {
	m := make(map[string]struct{}, len(list))
	for _, it := range list {
		k := string(it.BundleID) + "|" + string(it.TemplateSlug)
		m[k] = struct{}{}
	}
	return len(m)
}

func uniqCntVersion(list []spec.PromptTemplateListItem) int {
	m := make(map[string]struct{}, len(list))
	for _, it := range list {
		k := string(it.BundleID) + "|" + string(it.TemplateSlug) + "|" + string(it.TemplateVersion)
		m[k] = struct{}{}
	}
	return len(m)
}

func drainErrors(t *testing.T, ch <-chan error) {
	t.Helper()
	for err := range ch {
		if err != nil {
			t.Errorf("scale-test error: %v", err)
		}
	}
}
