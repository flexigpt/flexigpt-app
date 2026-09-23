package store

import (
	"context"
	"errors"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	toolSpec "github.com/flexigpt/flexigpt-app/internal/tool/spec"
)

func newBuiltInToolStore(t *testing.T) *ToolStore {
	t.Helper()

	store, err := NewToolStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewToolStore() error = %v", err)
	}
	t.Cleanup(store.Close)

	return store
}

func TestToolStoreListsOnlyBuiltInToolsAndPaginates(t *testing.T) {
	store := newBuiltInToolStore(t)
	ctx := t.Context()

	items := collectAllTools(t, ctx, store, true, 1)
	if len(items) == 0 {
		t.Fatal("ListTools() returned no built-in tools")
	}

	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		if !item.IsBuiltIn {
			t.Fatalf(
				"ListTools() returned non-built-in list item %s/%s@%s",
				item.BundleID,
				item.ToolSlug,
				item.ToolVersion,
			)
		}
		if !item.ToolDefinition.IsBuiltIn {
			t.Fatalf(
				"ListTools() returned non-built-in definition %s/%s@%s",
				item.BundleID,
				item.ToolSlug,
				item.ToolVersion,
			)
		}

		key := toolListItemKey(item)
		if _, duplicate := seen[key]; duplicate {
			t.Fatalf("ListTools() returned duplicate %q", key)
		}
		seen[key] = struct{}{}
	}

	target := items[0]
	response, err := store.GetTool(ctx, &toolSpec.GetToolRequest{
		BundleID: target.BundleID,
		ToolSlug: target.ToolSlug,
		Version:  target.ToolVersion,
	})
	if err != nil {
		t.Fatalf("GetTool() error = %v", err)
	}
	if response == nil || response.Body == nil {
		t.Fatal("GetTool() returned an empty body")
	}
	if !response.Body.IsBuiltIn {
		t.Fatal("GetTool() returned a non-built-in tool")
	}

	bundles, err := store.ListToolBundles(
		ctx,
		&toolSpec.ListToolBundlesRequest{
			IncludeDisabled: true,
			PageSize:        1,
		},
	)
	if err != nil {
		t.Fatalf("ListToolBundles() error = %v", err)
	}
	if bundles == nil || bundles.Body == nil {
		t.Fatal("ListToolBundles() returned an empty body")
	}
	if len(bundles.Body.ToolBundles) == 0 {
		t.Fatal("ListToolBundles() returned no built-in bundles")
	}
	for _, bundle := range bundles.Body.ToolBundles {
		if !bundle.IsBuiltIn {
			t.Fatalf("ListToolBundles() returned non-built-in bundle %q", bundle.ID)
		}
	}
}

func TestToolStoreEnablementOverlayAffectsLists(t *testing.T) {
	store := newBuiltInToolStore(t)
	ctx := t.Context()

	items := collectAllTools(t, ctx, store, true, 1)
	if len(items) == 0 {
		t.Fatal("ListTools() returned no built-in tools")
	}
	target := items[0]

	bundle, isBuiltIn, err := store.GetAnyToolBundle(ctx, target.BundleID)
	if err != nil {
		t.Fatalf("GetAnyToolBundle() error = %v", err)
	}
	if !isBuiltIn || !bundle.IsBuiltIn {
		t.Fatal("GetAnyToolBundle() did not return a built-in bundle")
	}

	originalBundleEnabled := bundle.IsEnabled
	originalToolEnabled := target.ToolDefinition.IsEnabled
	t.Cleanup(func() {
		cleanupCtx := t.Context()
		_, _ = store.PatchTool(cleanupCtx, &toolSpec.PatchToolRequest{
			BundleID: target.BundleID,
			ToolSlug: target.ToolSlug,
			Version:  target.ToolVersion,
			Body: &toolSpec.PatchToolRequestBody{
				IsEnabled: originalToolEnabled,
			},
		})
		_, _ = store.PatchToolBundle(cleanupCtx, &toolSpec.PatchToolBundleRequest{
			BundleID: target.BundleID,
			Body: &toolSpec.PatchToolBundleRequestBody{
				IsEnabled: originalBundleEnabled,
			},
		})
	})

	mustPatchBundleEnabled(t, ctx, store, target.BundleID, true)
	mustPatchToolEnabled(
		t,
		ctx,
		store,
		target.BundleID,
		target.ToolSlug,
		target.ToolVersion,
		true,
	)

	mustPatchBundleEnabled(t, ctx, store, target.BundleID, false)

	visibleBundles, err := store.ListToolBundles(
		ctx,
		&toolSpec.ListToolBundlesRequest{},
	)
	if err != nil {
		t.Fatalf("ListToolBundles() error = %v", err)
	}
	for _, visible := range visibleBundles.Body.ToolBundles {
		if visible.ID == target.BundleID {
			t.Fatalf("disabled bundle %q was returned by enabled-only list", target.BundleID)
		}
	}

	mustPatchBundleEnabled(t, ctx, store, target.BundleID, true)
	mustPatchToolEnabled(
		t,
		ctx,
		store,
		target.BundleID,
		target.ToolSlug,
		target.ToolVersion,
		false,
	)

	visibleTools := collectAllTools(t, ctx, store, false, 1)
	if containsTool(visibleTools, target) {
		t.Fatalf(
			"disabled tool %s/%s@%s was returned by enabled-only list",
			target.BundleID,
			target.ToolSlug,
			target.ToolVersion,
		)
	}

	allTools := collectAllTools(t, ctx, store, true, 1)
	disabled, found := findTool(allTools, target)
	if !found {
		t.Fatalf(
			"disabled tool %s/%s@%s was absent from includeDisabled list",
			target.BundleID,
			target.ToolSlug,
			target.ToolVersion,
		)
	}
	if disabled.ToolDefinition.IsEnabled {
		t.Fatalf(
			"tool %s/%s@%s remained enabled after PatchTool()",
			target.BundleID,
			target.ToolSlug,
			target.ToolVersion,
		)
	}
}

func TestToolStoreRejectsUnknownBundle(t *testing.T) {
	store := newBuiltInToolStore(t)

	_, _, err := store.GetAnyToolBundle(
		t.Context(),
		bundleitemutils.BundleID("not-a-built-in-bundle"),
	)
	if !errors.Is(err, errBundleNotFound) {
		t.Fatalf("GetAnyToolBundle() error = %v, want errBundleNotFound", err)
	}
}

func collectAllTools(
	t *testing.T,
	ctx context.Context,
	store *ToolStore,
	includeDisabled bool,
	pageSize int,
) []toolSpec.ToolListItem {
	t.Helper()

	var (
		items      []toolSpec.ToolListItem
		pageToken  string
		seenTokens = map[string]struct{}{}
	)

	for page := 0; ; page++ {
		if page > 4096 {
			t.Fatal("ListTools() exceeded pagination safety limit")
		}

		response, err := store.ListTools(ctx, &toolSpec.ListToolsRequest{
			IncludeDisabled:     includeDisabled,
			RecommendedPageSize: pageSize,
			PageToken:           pageToken,
		})
		if err != nil {
			t.Fatalf("ListTools() error = %v", err)
		}
		if response == nil || response.Body == nil {
			t.Fatal("ListTools() returned an empty body")
		}

		items = append(items, response.Body.ToolListItems...)

		if response.Body.NextPageToken == nil ||
			*response.Body.NextPageToken == "" {
			return items
		}

		next := *response.Body.NextPageToken
		if _, duplicate := seenTokens[next]; duplicate {
			t.Fatalf("ListTools() repeated page token %q", next)
		}
		seenTokens[next] = struct{}{}
		pageToken = next
	}
}

func mustPatchBundleEnabled(
	t *testing.T,
	ctx context.Context,
	store *ToolStore,
	bundleID bundleitemutils.BundleID,
	enabled bool,
) {
	t.Helper()

	_, err := store.PatchToolBundle(ctx, &toolSpec.PatchToolBundleRequest{
		BundleID: bundleID,
		Body: &toolSpec.PatchToolBundleRequestBody{
			IsEnabled: enabled,
		},
	})
	if err != nil {
		t.Fatalf("PatchToolBundle() error = %v", err)
	}
}

func mustPatchToolEnabled(
	t *testing.T,
	ctx context.Context,
	store *ToolStore,
	bundleID bundleitemutils.BundleID,
	toolSlug bundleitemutils.ItemSlug,
	version bundleitemutils.ItemVersion,
	enabled bool,
) {
	t.Helper()

	_, err := store.PatchTool(ctx, &toolSpec.PatchToolRequest{
		BundleID: bundleID,
		ToolSlug: toolSlug,
		Version:  version,
		Body: &toolSpec.PatchToolRequestBody{
			IsEnabled: enabled,
		},
	})
	if err != nil {
		t.Fatalf("PatchTool() error = %v", err)
	}
}

func toolListItemKey(item toolSpec.ToolListItem) string {
	return string(item.BundleID) + "\x00" +
		string(item.ToolSlug) + "\x00" +
		string(item.ToolVersion)
}

func containsTool(
	items []toolSpec.ToolListItem,
	target toolSpec.ToolListItem,
) bool {
	_, found := findTool(items, target)
	return found
}

func findTool(
	items []toolSpec.ToolListItem,
	target toolSpec.ToolListItem,
) (toolSpec.ToolListItem, bool) {
	targetKey := toolListItemKey(target)
	for _, item := range items {
		if toolListItemKey(item) == targetKey {
			return item, true
		}
	}
	return toolSpec.ToolListItem{}, false
}
