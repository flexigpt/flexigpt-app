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
	"github.com/flexigpt/flexigpt-app/internal/tool/spec"
)

// Scale-test: create, patch, and delete lots of tool bundles.
func TestScale_LotsOfToolBundles(t *testing.T) {
	const (
		nBundles      = 256
		nonEmptyEvery = 10 // Every 10-th bundle gets one tool.
		deleteEvery   = 5  // Request delete on every 5-th bundle.
		patchEvery    = 2  // Toggle enabled every 2-nd bundle.
		concurrency   = 20
	)

	s, clean := newTestToolStore(t)
	defer clean()

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

				_, err := s.PutToolBundle(t.Context(), &spec.PutToolBundleRequest{
					BundleID: bID,
					Body: &spec.PutToolBundleRequestBody{
						Slug:        bundleitemutils.BundleSlug("slug-" + strconv.Itoa(i)),
						DisplayName: "Bundle " + strconv.Itoa(i),
						IsEnabled:   i%3 == 0, // Enabled for 0,3,6,...
					},
				})
				if err != nil {
					errCh <- err
					continue
				}

				// One tool?
				if i%nonEmptyEvery == 0 {
					_, err = s.PutTool(t.Context(), &spec.PutToolRequest{
						BundleID: bID,
						ToolSlug: "t",
						Version:  "v1",
						Body: &spec.PutToolRequestBody{
							DisplayName:  "dummy",
							IsEnabled:    true,
							Type:         spec.ToolTypeHTTP,
							UserCallable: true,
							LLMCallable:  true,

							HTTPImpl:  dummyHTTPTool(),
							ArgSchema: `{}`,
						},
					})
					if err != nil && !errors.Is(err, errBundleDisabled) {
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
					_, err := s.PatchToolBundle(t.Context(), &spec.PatchToolBundleRequest{
						BundleID: bID,
						Body: &spec.PatchToolBundleRequestBody{
							IsEnabled: i%3 != 0, // Invert initial state.
						},
					})
					if err != nil {
						errCh <- err
					}
				}

				if i%deleteEvery == 0 {
					_, err := s.DeleteToolBundle(t.Context(), &spec.DeleteToolBundleRequest{
						BundleID: bID,
					})

					nonEmpty := (i%nonEmptyEvery == 0) && (i%3 == 0) // Bundle had tool.
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

	resp, err := s.ListToolBundles(t.Context(), &spec.ListToolBundlesRequest{})
	if err != nil {
		errCh <- err
	} else {
		for _, b := range resp.Body.ToolBundles {
			if !b.IsEnabled {
				errCh <- fmt.Errorf("disabled bundle %s in enabled-only list", b.ID)
			}
		}
	}

	close(errCh)
	drainErrors(t, errCh)
}

// Scale-test: create, patch, and delete lots of tools.
func TestScale_LotsOfTools(t *testing.T) {
	const (
		nBundles    = 3
		nSlugs      = 120
		nVersions   = 4
		concurrency = 40
	)

	s, clean := newTestToolStore(t)
	defer clean()

	errCh := make(chan error, 20_000)

	userBundles := make([]bundleitemutils.BundleID, 0, nBundles)
	for i := range nBundles {
		bID := bundleitemutils.BundleID(fmt.Sprintf("b%d", i))
		userBundles = append(userBundles, bID)

		_, err := s.PutToolBundle(t.Context(), &spec.PutToolBundleRequest{
			BundleID: bID,
			Body: &spec.PutToolBundleRequestBody{
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
				_, err := s.PutTool(t.Context(), &spec.PutToolRequest{
					BundleID: bundleitemutils.BundleID(fmt.Sprintf("b%d", j.b)),
					ToolSlug: bundleitemutils.ItemSlug(fmt.Sprintf("tool-%d", j.s)),
					Version:  bundleitemutils.ItemVersion(fmt.Sprintf("v%d", j.v)),
					Body: &spec.PutToolRequestBody{
						DisplayName:  "T",
						IsEnabled:    true,
						Type:         spec.ToolTypeHTTP,
						UserCallable: true,
						LLMCallable:  true,

						HTTPImpl:  dummyHTTPTool(),
						ArgSchema: `{}`,
					},
				})
				if err != nil {
					errCh <- err
				}
				time.Sleep(500 * time.Microsecond) // Help ModifiedAt monotonicity.
			}
		}()
	}
	wg.Wait()

	base := spec.ListToolsRequest{
		BundleIDs: userBundles,
	}
	all, err := collectTools(t.Context(), s, base, 100)
	if err != nil {
		errCh <- err
	} else if got, want := uniqCntToolSlug(all), nBundles*nSlugs; got != want {
		errCh <- fmt.Errorf("slug uniqueness mismatch got=%d want=%d", got, want)
	}

	modJobs := make(chan job, nBundles*nSlugs)
	for b := range nBundles {
		for sIdx := range nSlugs {
			if sIdx%2 == 0 { // Disable v4.
				modJobs <- job{b, sIdx, 4}
			}
			if sIdx%3 == 0 { // Delete v1.
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
				slug := bundleitemutils.ItemSlug(fmt.Sprintf("tool-%d", j.s))

				if j.v == 4 { // Patch disable.
					_, err := s.PatchTool(t.Context(), &spec.PatchToolRequest{
						BundleID: bID,
						ToolSlug: slug,
						Version:  "v4",
						Body: &spec.PatchToolRequestBody{
							IsEnabled: false,
						},
					})
					if err != nil {
						errCh <- err
					}
				} else { // Delete v1.
					_, err := s.DeleteTool(t.Context(), &spec.DeleteToolRequest{
						BundleID: bID,
						ToolSlug: slug,
						Version:  "v1",
					})
					if err != nil {
						errCh <- err
					}
				}
			}
		}()
	}
	wg.Wait()

	base = spec.ListToolsRequest{
		IncludeDisabled: true,
		BundleIDs:       userBundles,
	}
	all, err = collectTools(t.Context(), s, base, 150)
	if err != nil {
		errCh <- err
	} else {
		total := nBundles * nSlugs * nVersions
		deleted := nBundles * (nSlugs / 3) // v1 removed for every 3rd slug.
		want := total - deleted
		if got := uniqCntToolVersion(all); got != want {
			errCh <- fmt.Errorf("version uniqueness mismatch got=%d want=%d", got, want)
		}
	}

	base = spec.ListToolsRequest{
		BundleIDs:           userBundles,
		RecommendedPageSize: 70,
	}
	all, err = collectTools(t.Context(), s, base, 70)
	if err != nil {
		errCh <- err
	} else if got, want := uniqCntToolSlug(all), nBundles*nSlugs; got != want {
		errCh <- fmt.Errorf("pagination slug mismatch got=%d want=%d", got, want)
	}

	close(errCh)
	drainErrors(t, errCh)
}

// collectTools pages through ListTools until NextPageToken=="".
func collectTools(
	ctx context.Context,
	s *ToolStore,
	baseReq spec.ListToolsRequest,
	pageHint int,
) ([]spec.ToolListItem, error) {
	var (
		out   []spec.ToolListItem
		token string
	)

	for {
		req := baseReq // Copy.
		req.RecommendedPageSize = pageHint
		req.PageToken = token

		resp, err := s.ListTools(ctx, &req)
		if err != nil {
			return nil, err
		}
		out = append(out, resp.Body.ToolListItems...)

		if resp.Body.NextPageToken == nil || *resp.Body.NextPageToken == "" {
			break
		}
		token = *resp.Body.NextPageToken
	}
	return out, nil
}

// uniqCntToolSlug counts unique (bundleID, toolSlug) pairs.
func uniqCntToolSlug(list []spec.ToolListItem) int {
	m := make(map[string]struct{}, len(list))
	for _, it := range list {
		k := string(it.BundleID) + "|" + string(it.ToolSlug)
		m[k] = struct{}{}
	}
	return len(m)
}

// uniqCntToolVersion counts unique (bundleID, toolSlug, toolVersion) triples.
func uniqCntToolVersion(list []spec.ToolListItem) int {
	m := make(map[string]struct{}, len(list))
	for _, it := range list {
		k := string(it.BundleID) + "|" + string(it.ToolSlug) + "|" + string(it.ToolVersion)
		m[k] = struct{}{}
	}
	return len(m)
}

// drainErrors logs all errors from the channel.
func drainErrors(t *testing.T, ch <-chan error) {
	t.Helper()
	for err := range ch {
		if err != nil {
			t.Errorf("scale-test error: %v", err)
		}
	}
}
