package store

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/tool/spec"
)

const (
	maxPageSizeTools      = 256
	defPageSizeTools      = 25
	builtInSnapshotMaxAge = time.Hour
)

var (
	errInvalidRequest        = errors.New("invalid request")
	errInvalidDir            = errors.New("invalid directory")
	errBuiltInBundleNotFound = errors.New("bundle not found in built-in data")
	errBundleNotFound        = errors.New("bundle not found")
	errToolNotFound          = errors.New("tool not found")
)

// ToolStore exposes only the immutable built-in tool catalogue and its
// enable/disable overlay. User-authored bundles and tools are unsupported.
type ToolStore struct {
	baseDir     string
	builtinData *BuiltInToolData
}

// Option configures a ToolStore instance.
type Option func(*ToolStore) error

// NewToolStore initializes the built-in tool catalogue and overlay.
func NewToolStore(baseDir string, opts ...Option) (*ToolStore, error) {
	store := &ToolStore{
		baseDir: filepath.Clean(baseDir),
	}
	for _, option := range opts {
		if err := option(store); err != nil {
			return nil, err
		}
	}

	builtinData, err := NewBuiltInToolData(
		context.Background(),
		store.baseDir,
		builtInSnapshotMaxAge,
		WithLLMToolsGoBuiltins(true),
	)
	if err != nil {
		return nil, err
	}

	store.builtinData = builtinData
	return store, nil
}

// Close releases the built-in overlay resources.
func (ts *ToolStore) Close() {
	if ts == nil || ts.builtinData == nil {
		return
	}
	_ = ts.builtinData.Close()
	ts.builtinData = nil
}

// ListBuiltInTools returns every built-in Tool, including disabled items.
func (ts *ToolStore) ListBuiltInTools(
	ctx context.Context,
) ([]spec.ToolListItem, error) {
	items, _, err := ts.allBuiltInToolItems(ctx)
	if err != nil {
		return nil, err
	}
	return items, nil
}

// PatchToolBundle changes only the enablement overlay of a built-in bundle.
func (ts *ToolStore) PatchToolBundle(
	ctx context.Context,
	req *spec.PatchToolBundleRequest,
) (*spec.PatchToolBundleResponse, error) {
	if req == nil || req.Body == nil || req.BundleID == "" {
		return nil, fmt.Errorf("%w: bundleID required", errInvalidRequest)
	}
	if ts == nil || ts.builtinData == nil {
		return nil, fmt.Errorf("%w: %s", errBundleNotFound, req.BundleID)
	}
	if _, err := ts.builtinData.GetBuiltInToolBundle(ctx, req.BundleID); err != nil {
		return nil, fmt.Errorf("%w: %s", errBundleNotFound, req.BundleID)
	}
	if _, err := ts.builtinData.SetToolBundleEnabled(
		ctx,
		req.BundleID,
		req.Body.IsEnabled,
	); err != nil {
		return nil, err
	}
	return &spec.PatchToolBundleResponse{}, nil
}

// PatchTool changes only the enablement overlay of a built-in tool.
func (ts *ToolStore) PatchTool(
	ctx context.Context,
	req *spec.PatchToolRequest,
) (*spec.PatchToolResponse, error) {
	if req == nil || req.Body == nil ||
		req.BundleID == "" || req.ToolSlug == "" || req.Version == "" {
		return nil, fmt.Errorf(
			"%w: bundleID, toolSlug, version required",
			errInvalidRequest,
		)
	}
	if err := bundleitemutils.ValidateItemSlug(req.ToolSlug); err != nil {
		return nil, err
	}
	if err := bundleitemutils.ValidateItemVersion(req.Version); err != nil {
		return nil, err
	}

	bundle, isBuiltIn, err := ts.GetAnyToolBundle(ctx, req.BundleID)
	if err != nil {
		return nil, err
	}
	if !isBuiltIn {
		return nil, fmt.Errorf("%w: %s", errBundleNotFound, req.BundleID)
	}

	if _, err := ts.builtinData.SetToolEnabled(
		ctx,
		bundle.ID,
		req.ToolSlug,
		req.Version,
		req.Body.IsEnabled,
	); err != nil {
		return nil, err
	}
	return &spec.PatchToolResponse{}, nil
}

// ListToolBundles lists built-in bundles only.
func (ts *ToolStore) ListToolBundles(
	ctx context.Context,
	req *spec.ListToolBundlesRequest,
) (*spec.ListToolBundlesResponse, error) {
	var (
		pageSize        = defPageSizeTools
		includeDisabled bool
		wantIDs         = map[bundleitemutils.BundleID]struct{}{}
		cursorMod       time.Time
		cursorID        bundleitemutils.BundleID
	)

	if req != nil && req.PageToken != "" {
		token, err := jsonutil.Base64JSONDecode[spec.BundlePageToken](
			req.PageToken,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"%w: invalid bundle page token: %w",
				errInvalidRequest,
				err,
			)
		}

		pageSize = token.PageSize
		if pageSize <= 0 || pageSize > maxPageSizeTools {
			pageSize = defPageSizeTools
		}
		includeDisabled = token.IncludeDisabled
		if token.CursorMod != "" {
			cursorMod, _ = time.Parse(time.RFC3339Nano, token.CursorMod)
			cursorID = token.CursorID
		}
		for _, id := range token.BundleIDs {
			wantIDs[id] = struct{}{}
		}
	} else if req != nil {
		if req.PageSize > 0 && req.PageSize <= maxPageSizeTools {
			pageSize = req.PageSize
		}
		includeDisabled = req.IncludeDisabled
		for _, id := range req.BundleIDs {
			wantIDs[id] = struct{}{}
		}
	}

	bundles, _, err := ts.builtInSnapshot(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]spec.ToolBundle, 0, len(bundles))
	for _, bundle := range bundles {
		if !bundle.IsBuiltIn {
			continue
		}
		if len(wantIDs) != 0 {
			if _, found := wantIDs[bundle.ID]; !found {
				continue
			}
		}
		if !includeDisabled && !bundle.IsEnabled {
			continue
		}
		filtered = append(filtered, bundle)
	}

	sort.Slice(filtered, func(left, right int) bool {
		if filtered[left].ModifiedAt.Equal(filtered[right].ModifiedAt) {
			return string(filtered[left].ID) < string(filtered[right].ID)
		}
		return filtered[left].ModifiedAt.After(filtered[right].ModifiedAt)
	})

	start := 0
	if cursorID != "" {
		for index, bundle := range filtered {
			if bundle.ModifiedAt.Equal(cursorMod) &&
				bundle.ID == cursorID {
				start = index + 1
				break
			}
		}
	}
	if start > len(filtered) {
		start = len(filtered)
	}

	end := min(start+pageSize, len(filtered))

	var nextToken *string
	if end < len(filtered) {
		ids := make([]bundleitemutils.BundleID, 0, len(wantIDs))
		for id := range wantIDs {
			ids = append(ids, id)
		}
		slices.Sort(ids)

		encoded := jsonutil.Base64JSONEncode(spec.BundlePageToken{
			BundleIDs:       ids,
			IncludeDisabled: includeDisabled,
			PageSize:        pageSize,
			CursorMod:       filtered[end-1].ModifiedAt.Format(time.RFC3339Nano),
			CursorID:        filtered[end-1].ID,
		})
		nextToken = &encoded
	}

	return &spec.ListToolBundlesResponse{
		Body: &spec.ListToolBundlesResponseBody{
			ToolBundles:   filtered[start:end],
			NextPageToken: nextToken,
		},
	}, nil
}

// GetTool retrieves a built-in tool version.
func (ts *ToolStore) GetTool(
	ctx context.Context,
	req *spec.GetToolRequest,
) (*spec.GetToolResponse, error) {
	if req == nil || req.BundleID == "" || req.ToolSlug == "" || req.Version == "" {
		return nil, fmt.Errorf(
			"%w: bundleID, toolSlug, version required",
			errInvalidRequest,
		)
	}
	if err := bundleitemutils.ValidateItemSlug(req.ToolSlug); err != nil {
		return nil, err
	}
	if err := bundleitemutils.ValidateItemVersion(req.Version); err != nil {
		return nil, err
	}

	bundle, isBuiltIn, err := ts.GetAnyToolBundle(ctx, req.BundleID)
	if err != nil {
		return nil, err
	}
	if !isBuiltIn {
		return nil, fmt.Errorf("%w: %s", errBundleNotFound, req.BundleID)
	}

	tool, err := ts.builtinData.GetBuiltInTool(
		ctx,
		bundle.ID,
		req.ToolSlug,
		req.Version,
	)
	if err != nil {
		return nil, err
	}
	if !tool.IsBuiltIn {
		return nil, fmt.Errorf(
			"%w: non-built-in tool returned for %s/%s@%s",
			errToolNotFound,
			req.BundleID,
			req.ToolSlug,
			req.Version,
		)
	}

	return &spec.GetToolResponse{Body: &tool}, nil
}

// ListTools lists built-in tools only.
func (ts *ToolStore) ListTools(
	ctx context.Context,
	req *spec.ListToolsRequest,
) (*spec.ListToolsResponse, error) {
	token := spec.ToolPageToken{}
	if req != nil && req.PageToken != "" {
		decoded, err := jsonutil.Base64JSONDecode[spec.ToolPageToken](
			req.PageToken,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"%w: invalid tool page token: %w",
				errInvalidRequest,
				err,
			)
		}
		token = decoded
	} else if req != nil {
		token.RecommendedPageSize = req.RecommendedPageSize
		token.IncludeDisabled = req.IncludeDisabled
		token.BundleIDs = slices.Clone(req.BundleIDs)
		slices.Sort(token.BundleIDs)
		token.Tags = slices.Clone(req.Tags)
		sort.Strings(token.Tags)
	}

	pageSize := token.RecommendedPageSize
	if pageSize <= 0 || pageSize > maxPageSizeTools {
		pageSize = defPageSizeTools
	}

	bundleFilter := make(map[bundleitemutils.BundleID]struct{}, len(token.BundleIDs))
	for _, id := range token.BundleIDs {
		bundleFilter[id] = struct{}{}
	}
	tagFilter := make(map[string]struct{}, len(token.Tags))
	for _, tag := range token.Tags {
		tagFilter[tag] = struct{}{}
	}

	items, bundles, err := ts.allBuiltInToolItems(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]spec.ToolListItem, 0, len(items))
	for _, item := range items {
		bundle, found := bundles[item.BundleID]
		if !found || !bundle.IsBuiltIn {
			continue
		}
		if len(bundleFilter) != 0 {
			if _, found := bundleFilter[item.BundleID]; !found {
				continue
			}
		}
		if !token.IncludeDisabled &&
			(!bundle.IsEnabled || !item.ToolDefinition.IsEnabled) {
			continue
		}
		if len(tagFilter) != 0 {
			matched := false
			for _, tag := range item.ToolDefinition.Tags {
				if _, found := tagFilter[tag]; found {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		filtered = append(filtered, item)
	}

	if token.Offset < 0 || token.Offset > len(filtered) {
		return nil, fmt.Errorf("%w: invalid tool page offset", errInvalidRequest)
	}

	start := token.Offset
	end := min(start+pageSize, len(filtered))

	var nextToken *string
	if end < len(filtered) {
		token.Offset = end
		encoded := jsonutil.Base64JSONEncode(token)
		nextToken = &encoded
	}

	return &spec.ListToolsResponse{
		Body: &spec.ListToolsResponseBody{
			ToolListItems: filtered[start:end],
			NextPageToken: nextToken,
		},
	}, nil
}

// GetAnyToolBundle returns a built-in bundle only.
func (ts *ToolStore) GetAnyToolBundle(
	ctx context.Context,
	id bundleitemutils.BundleID,
) (spec.ToolBundle, bool, error) {
	if id == "" {
		return spec.ToolBundle{}, false, fmt.Errorf(
			"%w: bundleID required",
			errInvalidRequest,
		)
	}
	if ts == nil || ts.builtinData == nil {
		return spec.ToolBundle{}, false, fmt.Errorf(
			"%w: %s",
			errBundleNotFound,
			id,
		)
	}

	bundle, err := ts.builtinData.GetBuiltInToolBundle(ctx, id)
	if err != nil {
		return spec.ToolBundle{}, false, fmt.Errorf(
			"%w: %s",
			errBundleNotFound,
			id,
		)
	}
	if !bundle.IsBuiltIn {
		return spec.ToolBundle{}, false, fmt.Errorf(
			"%w: %s",
			errBundleNotFound,
			id,
		)
	}

	return bundle, true, nil
}

func (ts *ToolStore) builtInSnapshot(
	ctx context.Context,
) (
	bundles map[bundleitemutils.BundleID]spec.ToolBundle,
	tools map[bundleitemutils.BundleID]map[bundleitemutils.ItemID]spec.Tool,
	err error,
) {
	if ts == nil || ts.builtinData == nil {
		return nil, nil, fmt.Errorf(
			"%w: built-in Tool data is unavailable",
			errToolNotFound,
		)
	}
	return ts.builtinData.ListBuiltInToolData(ctx)
}

func (ts *ToolStore) allBuiltInToolItems(
	ctx context.Context,
) (
	[]spec.ToolListItem,
	map[bundleitemutils.BundleID]spec.ToolBundle,
	error,
) {
	bundles, tools, err := ts.builtInSnapshot(ctx)
	if err != nil {
		return nil, nil, err
	}

	bundleIDs := make([]bundleitemutils.BundleID, 0, len(bundles))
	for bundleID := range bundles {
		bundleIDs = append(bundleIDs, bundleID)
	}
	slices.Sort(bundleIDs)

	items := make([]spec.ToolListItem, 0)
	for _, bundleID := range bundleIDs {
		bundle := bundles[bundleID]
		if !bundle.IsBuiltIn {
			continue
		}

		toolIDs := make([]bundleitemutils.ItemID, 0, len(tools[bundleID]))
		for toolID := range tools[bundleID] {
			toolIDs = append(toolIDs, toolID)
		}
		slices.SortFunc(toolIDs, func(
			left bundleitemutils.ItemID,
			right bundleitemutils.ItemID,
		) int {
			return strings.Compare(string(left), string(right))
		})

		for _, toolID := range toolIDs {
			tool := tools[bundleID][toolID]
			if !tool.IsBuiltIn {
				continue
			}
			items = append(items, spec.ToolListItem{
				BundleID:       bundleID,
				BundleSlug:     bundle.Slug,
				ToolSlug:       tool.Slug,
				ToolVersion:    tool.Version,
				IsBuiltIn:      true,
				ToolDefinition: tool,
			})
		}
	}

	return items, bundles, nil
}
