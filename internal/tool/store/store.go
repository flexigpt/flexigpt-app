package store

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/debug"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/tool/fts"
	"github.com/flexigpt/flexigpt-app/internal/tool/goregistry"
	"github.com/flexigpt/flexigpt-app/internal/tool/httprunner"
	"github.com/flexigpt/flexigpt-app/internal/tool/spec"
	"github.com/flexigpt/flexigpt-app/internal/tool/storehelper"
	"github.com/flexigpt/llmtools-go"
	"github.com/ppipada/mapstore-go"
	"github.com/ppipada/mapstore-go/ftsengine"
	"github.com/ppipada/mapstore-go/jsonencdec"
	"github.com/ppipada/mapstore-go/uuidv7filename"
)

const (
	fetchBatchTools       = 512            // Directory-store batch size.
	maxPageSizeTools      = 256            // Hard page limit.
	defPageSizeTools      = 25             // Default page size.
	softDeleteGraceTools  = 48 * time.Hour // Soft-delete grace interval.
	cleanupIntervalTools  = 24 * time.Hour // Sweep interval.
	builtInSnapshotMaxAge = time.Hour
)

// ToolStore provides CRUD, soft-delete and optional FTS for Tool bundles.
type ToolStore struct {
	baseDir string

	// Meta-data and raw file stores.
	bundleStore *mapstore.MapFileStore
	toolStore   *mapstore.MapDirectoryStore

	// Built-in overlay.
	builtinData *BuiltInToolData

	enableFTS bool
	fts       *ftsengine.Engine

	pp mapstore.PartitionProvider

	slugLock *storehelper.SlugLocks

	// Cleanup loop plumbing.
	cleanOnce sync.Once
	cleanKick chan struct{}
	cleanCtx  context.Context
	cleanStop context.CancelFunc
	wg        sync.WaitGroup

	// Sweep coordination with CRUD ops.
	sweepMu sync.RWMutex
}

// Option configures a ToolStore instance.
type Option func(*ToolStore) error

// WithFTS toggles full-text search indexing.
func WithFTS(enabled bool) Option {
	return func(ts *ToolStore) error {
		ts.enableFTS = enabled
		return nil
	}
}

// NewToolStore initialises a ToolStore rooted at baseDir.
func NewToolStore(baseDir string, opts ...Option) (*ToolStore, error) {
	ts := &ToolStore{
		baseDir: filepath.Clean(baseDir),
		pp:      &bundleitemutils.BundlePartitionProvider{},
	}
	for _, o := range opts {
		if err := o(ts); err != nil {
			return nil, err
		}
	}
	ctx := context.Background()

	// Built-in overlay.
	bi, err := NewBuiltInToolData(ctx, ts.baseDir, builtInSnapshotMaxAge, WithLLMToolsGoBuiltins(true))
	if err != nil {
		return nil, err
	}
	ts.builtinData = bi

	// Bundle meta-file.
	def, _ := jsonencdec.StructWithJSONTagsToMap(
		spec.AllBundles{Bundles: map[bundleitemutils.BundleID]spec.ToolBundle{}},
	)
	ts.bundleStore, err = mapstore.NewMapFileStore(
		filepath.Join(ts.baseDir, spec.ToolBundlesMetaFileName),
		def,
		jsonencdec.JSONEncoderDecoder{},
		mapstore.WithCreateIfNotExists(true),
		mapstore.WithFileAutoFlush(true),
		mapstore.WithFileLogger(slog.Default()),
	)
	if err != nil {
		return nil, err
	}

	// FTS engine.
	if ts.enableFTS {
		var lister fts.ToolBuiltInLister
		if ts.builtinData != nil {
			lister = ts.builtinData.ListBuiltInToolData
		}
		ts.fts, err = fts.InitToolFTSListeners(ts.baseDir, lister)
		if err != nil {
			return nil, err
		}
	}

	// Directory store.
	dirOpts := []mapstore.DirOption{mapstore.WithDirLogger(slog.Default())}
	if ts.fts != nil {
		dirOpts = append(dirOpts, mapstore.WithDirFileListeners(fts.NewUserToolsFTSListener(ts.fts)))
	}
	ts.toolStore, err = mapstore.NewMapDirectoryStore(
		ts.baseDir,
		true,
		ts.pp,
		jsonencdec.JSONEncoderDecoder{},
		dirOpts...)
	if err != nil {
		return nil, err
	}

	ts.slugLock = storehelper.NewSlugLocks()
	ts.startCleanupLoop()

	slog.Info("tool-store ready", "baseDir", ts.baseDir, "fts", ts.enableFTS)
	return ts, nil
}

// Close shuts down the background sweep.
func (ts *ToolStore) Close() {
	if ts.cleanStop != nil {
		ts.cleanStop()
	}
	ts.wg.Wait()
}

// PutToolBundle creates or replaces a bundle.
func (ts *ToolStore) PutToolBundle(
	ctx context.Context, req *spec.PutToolBundleRequest,
) (*spec.PutToolBundleResponse, error) {
	if req == nil || req.Body == nil ||
		req.BundleID == "" || req.Body.Slug == "" || req.Body.DisplayName == "" {
		return nil, fmt.Errorf("%w: id, slug & displayName required", spec.ErrInvalidRequest)
	}
	if err := bundleitemutils.ValidateBundleSlug(req.Body.Slug); err != nil {
		return nil, err
	}

	// Built-ins are immutable.
	if ts.builtinData != nil {
		if _, err := ts.builtinData.GetBuiltInToolBundle(ctx, req.BundleID); err == nil {
			return nil, fmt.Errorf("%w: bundleID %q", spec.ErrBuiltInReadOnly, req.BundleID)
		}
	}

	ts.sweepMu.Lock()
	defer ts.sweepMu.Unlock()

	all, err := ts.readAllBundles(false)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	createdAt := now
	if ex, ok := all.Bundles[req.BundleID]; ok && !ex.CreatedAt.IsZero() {
		createdAt = ex.CreatedAt
	}

	b := spec.ToolBundle{
		SchemaVersion: spec.SchemaVersion,
		ID:            req.BundleID,
		Slug:          req.Body.Slug,
		DisplayName:   req.Body.DisplayName,
		Description:   req.Body.Description,
		IsEnabled:     req.Body.IsEnabled,
		IsBuiltIn:     false,
		CreatedAt:     createdAt,
		ModifiedAt:    now,
		SoftDeletedAt: nil,
	}

	all.Bundles[req.BundleID] = b
	if err := ts.writeAllBundles(all); err != nil {
		return nil, err
	}
	slog.Info("putToolBundle", "bundleID", req.BundleID)
	return &spec.PutToolBundleResponse{}, nil
}

// PatchToolBundle toggles the enabled flag.
func (ts *ToolStore) PatchToolBundle(
	ctx context.Context, req *spec.PatchToolBundleRequest,
) (*spec.PatchToolBundleResponse, error) {
	if req == nil || req.Body == nil || req.BundleID == "" {
		return nil, fmt.Errorf("%w: bundleID required", spec.ErrInvalidRequest)
	}

	// Built-in?
	if ts.builtinData != nil {
		if _, err := ts.builtinData.GetBuiltInToolBundle(ctx, req.BundleID); err == nil {
			if _, err := ts.builtinData.SetToolBundleEnabled(ctx, req.BundleID, req.Body.IsEnabled); err != nil {
				return nil, err
			}
			slog.Info(
				"patchToolBundle (builtin)",
				"id",
				req.BundleID,
				"enabled",
				req.Body.IsEnabled,
			)
			return &spec.PatchToolBundleResponse{}, nil
		}
	}

	ts.sweepMu.Lock()
	defer ts.sweepMu.Unlock()

	all, err := ts.readAllBundles(false)
	if err != nil {
		return nil, err
	}
	bl, ok := all.Bundles[req.BundleID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", spec.ErrBundleNotFound, req.BundleID)
	}
	bl.IsEnabled = req.Body.IsEnabled
	bl.ModifiedAt = time.Now().UTC()
	all.Bundles[req.BundleID] = bl
	if err := ts.writeAllBundles(all); err != nil {
		return nil, err
	}
	slog.Info("patchToolBundle", "id", req.BundleID, "enabled", req.Body.IsEnabled)
	return &spec.PatchToolBundleResponse{}, nil
}

// DeleteToolBundle performs soft delete (only if empty).
func (ts *ToolStore) DeleteToolBundle(
	ctx context.Context, req *spec.DeleteToolBundleRequest,
) (*spec.DeleteToolBundleResponse, error) {
	if req == nil || req.BundleID == "" {
		return nil, fmt.Errorf("%w: bundleID required", spec.ErrInvalidRequest)
	}

	// Built-ins are immutable.
	if ts.builtinData != nil {
		if _, err := ts.builtinData.GetBuiltInToolBundle(ctx, req.BundleID); err == nil {
			return nil, fmt.Errorf("%w: bundleID %q", spec.ErrBuiltInReadOnly, req.BundleID)
		}
	}

	ts.sweepMu.Lock()
	defer ts.sweepMu.Unlock()

	all, err := ts.readAllBundles(false)
	if err != nil {
		return nil, err
	}
	b, ok := all.Bundles[req.BundleID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", spec.ErrBundleNotFound, req.BundleID)
	}
	if isSoftDeletedTool(b) {
		return nil, fmt.Errorf("%w: %s", spec.ErrBundleDeleting, req.BundleID)
	}

	dirInfo, derr := bundleitemutils.BuildBundleDir(b.ID, b.Slug)
	if derr != nil {
		return nil, derr
	}
	files, _, err := ts.toolStore.ListFiles(
		mapstore.ListingConfig{FilterPartitions: []string{dirInfo.DirName}, PageSize: 1}, "",
	)
	if err != nil {
		return nil, err
	}
	if len(files) > 0 {
		return nil, fmt.Errorf("%w: %s", spec.ErrBundleNotEmpty, req.BundleID)
	}

	now := time.Now().UTC()
	b.IsEnabled = false
	b.SoftDeletedAt = &now
	all.Bundles[req.BundleID] = b
	if err := ts.writeAllBundles(all); err != nil {
		return nil, err
	}

	ts.kickCleanupLoop()
	slog.Info("deleteToolBundle", "bundleID", req.BundleID)
	return &spec.DeleteToolBundleResponse{}, nil
}

// ListToolBundles returns bundles with filtering & pagination.
func (ts *ToolStore) ListToolBundles(
	ctx context.Context, req *spec.ListToolBundlesRequest,
) (*spec.ListToolBundlesResponse, error) {
	var (
		pageSize        = defPageSizeTools
		includeDisabled bool
		wantIDs         = map[bundleitemutils.BundleID]struct{}{}
		cursorMod       time.Time
		cursorID        bundleitemutils.BundleID
	)

	// Token overrides parameters.
	if req != nil && req.PageToken != "" {
		if tok, err := jsonutil.Base64JSONDecode[spec.BundlePageToken](req.PageToken); err == nil {
			pageSize = tok.PageSize
			if pageSize <= 0 || pageSize > maxPageSizeTools {
				pageSize = defPageSizeTools
			}
			includeDisabled = tok.IncludeDisabled
			if tok.CursorMod != "" {
				cursorMod, _ = time.Parse(time.RFC3339Nano, tok.CursorMod)
				cursorID = tok.CursorID
			}
			for _, id := range tok.BundleIDs {
				wantIDs[id] = struct{}{}
			}
		}
	} else if req != nil { // First page.
		if req.PageSize > 0 && req.PageSize <= maxPageSizeTools {
			pageSize = req.PageSize
		}
		includeDisabled = req.IncludeDisabled
		for _, id := range req.BundleIDs {
			wantIDs[id] = struct{}{}
		}
	}

	// Collect bundles (built-in + user).
	bundles := make([]spec.ToolBundle, 0)
	if ts.builtinData != nil {
		bi, _, _ := ts.builtinData.ListBuiltInToolData(ctx)
		for _, b := range bi {
			bundles = append(bundles, b)
		}
	}
	user, err := ts.readAllBundles(false)
	if err != nil {
		return nil, err
	}
	for _, b := range user.Bundles {
		if isSoftDeletedTool(b) {
			continue
		}
		bundles = append(bundles, b)
	}

	// Filtering.
	filtered := make([]spec.ToolBundle, 0, len(bundles))
	for _, b := range bundles {
		if len(wantIDs) > 0 {
			if _, ok := wantIDs[b.ID]; !ok {
				continue
			}
		}
		if !includeDisabled && !b.IsEnabled {
			continue
		}
		filtered = append(filtered, b)
	}

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].ModifiedAt.Equal(filtered[j].ModifiedAt) {
			return filtered[i].ID < filtered[j].ID
		}
		return filtered[i].ModifiedAt.After(filtered[j].ModifiedAt)
	})

	start := 0
	if !cursorMod.IsZero() || cursorID != "" {
		for i, b := range filtered {
			if b.ModifiedAt.Before(cursorMod) ||
				(b.ModifiedAt.Equal(cursorMod) && b.ID == cursorID) {
				start = i + 1
				break
			}
		}
	}
	end := min(start+pageSize, len(filtered))

	var nextTok *string
	if end < len(filtered) {
		ids := make([]bundleitemutils.BundleID, 0, len(wantIDs))
		for id := range wantIDs {
			ids = append(ids, id)
		}
		slices.Sort(ids)

		next := jsonutil.Base64JSONEncode(spec.BundlePageToken{
			BundleIDs:       ids,
			IncludeDisabled: includeDisabled,
			PageSize:        pageSize,
			CursorMod:       filtered[end-1].ModifiedAt.Format(time.RFC3339Nano),
			CursorID:        filtered[end-1].ID,
		})
		nextTok = &next
	}

	return &spec.ListToolBundlesResponse{
		Body: &spec.ListToolBundlesResponseBody{
			ToolBundles:   filtered[start:end],
			NextPageToken: nextTok,
		},
	}, nil
}

// PutTool creates a new tool version (immutable). Only HTTPImpl function tools are allowed to be added as of now.
func (ts *ToolStore) PutTool(
	ctx context.Context, req *spec.PutToolRequest,
) (*spec.PutToolResponse, error) {
	if req == nil || req.Body == nil ||
		req.BundleID == "" || req.ToolSlug == "" || req.Version == "" {
		return nil, fmt.Errorf("%w: bundleID, toolSlug, version required", spec.ErrInvalidRequest)
	}
	if req.Body.Type != spec.ToolTypeHTTP {
		return nil, fmt.Errorf("%w: only custom http tools can be added", spec.ErrInvalidRequest)
	}
	if !req.Body.UserCallable && !req.Body.LLMCallable {
		return nil, fmt.Errorf("%w: a tool needs to be callable", spec.ErrInvalidRequest)
	}

	if err := bundleitemutils.ValidateItemSlug(req.ToolSlug); err != nil {
		return nil, err
	}
	if err := bundleitemutils.ValidateItemVersion(req.Version); err != nil {
		return nil, err
	}

	bundle, isBI, err := ts.getAnyBundle(ctx, req.BundleID)
	if err != nil {
		return nil, err
	}
	if isBI {
		return nil, fmt.Errorf("%w: bundleID %q", spec.ErrBuiltInReadOnly, req.BundleID)
	}
	if !bundle.IsEnabled {
		return nil, fmt.Errorf("%w: %s", spec.ErrBundleDisabled, req.BundleID)
	}

	dirInfo, _ := bundleitemutils.BuildBundleDir(bundle.ID, bundle.Slug)

	// Per-slug lock.
	lock := ts.slugLock.LockKey(bundle.ID, req.ToolSlug)
	lock.Lock()
	defer lock.Unlock()

	finf, _ := bundleitemutils.BuildItemFileInfo(req.ToolSlug, req.Version)
	list, _, _ := ts.toolStore.ListFiles(
		mapstore.ListingConfig{
			FilterPartitions: []string{dirInfo.DirName},
			FilenamePrefix:   finf.FileName,
			PageSize:         10,
		}, "",
	)
	for _, ex := range list {
		if filepath.Base(ex.BaseRelativePath) == finf.FileName {
			return nil, fmt.Errorf("%w: slug+version exists", spec.ErrConflict)
		}
	}

	now := time.Now().UTC()
	uuid, _ := uuidv7filename.NewUUIDv7String()
	argSchemaStr := req.Body.ArgSchema
	if argSchemaStr == "" {
		argSchemaStr = "{}"
	}

	t := spec.Tool{
		SchemaVersion: spec.SchemaVersion,
		ID:            bundleitemutils.ItemID(uuid),
		Slug:          req.ToolSlug,
		Version:       req.Version,
		DisplayName:   req.Body.DisplayName,
		Description:   req.Body.Description,
		Tags:          req.Body.Tags,

		UserCallable: req.Body.UserCallable,
		LLMCallable:  req.Body.LLMCallable,
		AutoExecReco: req.Body.AutoExecReco,

		ArgSchema: json.RawMessage(argSchemaStr),

		LLMToolType: spec.ToolStoreChoiceTypeFunction,
		Type:        req.Body.Type,
		HTTPImpl:    req.Body.HTTPImpl,
		IsEnabled:   req.Body.IsEnabled,
		IsBuiltIn:   false,
		CreatedAt:   now,
		ModifiedAt:  now,
	}

	if err := storehelper.ValidateTool(&t); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	mp, _ := jsonencdec.StructWithJSONTagsToMap(t)
	if err := ts.toolStore.SetFileData(
		bundleitemutils.GetBundlePartitionFileKey(finf.FileName, dirInfo.DirName),
		mp,
	); err != nil {
		return nil, err
	}
	slog.Info("putTool", "bundleID", req.BundleID, "slug", req.ToolSlug, "ver", req.Version)
	return &spec.PutToolResponse{}, nil
}

// PatchTool toggles enabled flag on a tool version.
func (ts *ToolStore) PatchTool(
	ctx context.Context, req *spec.PatchToolRequest,
) (*spec.PatchToolResponse, error) {
	if req == nil || req.Body == nil ||
		req.BundleID == "" || req.ToolSlug == "" || req.Version == "" {
		return nil, fmt.Errorf("%w: bundleID, toolSlug, version required", spec.ErrInvalidRequest)
	}
	if err := bundleitemutils.ValidateItemSlug(req.ToolSlug); err != nil {
		return nil, err
	}
	if err := bundleitemutils.ValidateItemVersion(req.Version); err != nil {
		return nil, err
	}

	bundle, isBI, err := ts.getAnyBundle(ctx, req.BundleID)
	if err != nil {
		return nil, err
	}
	if isBI {
		tool, err := ts.builtinData.SetToolEnabled(ctx,
			bundle.ID, req.ToolSlug, req.Version, req.Body.IsEnabled,
		)
		if err != nil {
			return nil, err
		}
		if ts.fts != nil {
			if err := fts.ReindexOneBuiltInTool(ctx, bundle.ID, bundle.Slug, tool, ts.fts); err != nil {
				slog.Warn("builtin-fts reindex(one) failed",
					"bundleID", bundle.ID, "toolID", tool.ID, "err", err)
			}
		}
		slog.Info("patchTool (builtin)", "bundleID", req.BundleID, "slug", req.ToolSlug,
			"ver", req.Version, "enabled", req.Body.IsEnabled)
		return &spec.PatchToolResponse{}, nil
	}
	if !bundle.IsEnabled {
		return nil, fmt.Errorf("%w: %s", spec.ErrBundleDisabled, req.BundleID)
	}

	dirInfo, _ := bundleitemutils.BuildBundleDir(bundle.ID, bundle.Slug)
	lock := ts.slugLock.LockKey(bundle.ID, req.ToolSlug)
	lock.Lock()
	defer lock.Unlock()

	finf, _ := bundleitemutils.BuildItemFileInfo(req.ToolSlug, req.Version)
	key := bundleitemutils.GetBundlePartitionFileKey(finf.FileName, dirInfo.DirName)

	raw, err := ts.toolStore.GetFileData(key, false)
	if err != nil {
		return nil, err
	}
	var tool spec.Tool
	if err := jsonencdec.MapToStructWithJSONTags(raw, &tool); err != nil {
		return nil, err
	}
	tool.IsEnabled = req.Body.IsEnabled
	tool.ModifiedAt = time.Now().UTC()

	mp, _ := jsonencdec.StructWithJSONTagsToMap(tool)
	if err := ts.toolStore.SetFileData(key, mp); err != nil {
		return nil, err
	}

	slog.Info("patchTool", "bundleID", req.BundleID, "slug", req.ToolSlug,
		"ver", req.Version, "enabled", req.Body.IsEnabled)
	return &spec.PatchToolResponse{}, nil
}

// DeleteTool removes a tool version permanently.
func (ts *ToolStore) DeleteTool(
	ctx context.Context, req *spec.DeleteToolRequest,
) (*spec.DeleteToolResponse, error) {
	if req == nil || req.BundleID == "" || req.ToolSlug == "" || req.Version == "" {
		return nil, fmt.Errorf("%w: bundleID, toolSlug, version required", spec.ErrInvalidRequest)
	}
	if err := bundleitemutils.ValidateItemSlug(req.ToolSlug); err != nil {
		return nil, err
	}
	if err := bundleitemutils.ValidateItemVersion(req.Version); err != nil {
		return nil, err
	}
	bundle, isBI, err := ts.getAnyBundle(ctx, req.BundleID)
	if err != nil {
		return nil, err
	}
	if isBI {
		return nil, fmt.Errorf("%w: bundleID %q", spec.ErrBuiltInReadOnly, req.BundleID)
	}

	dirInfo, _ := bundleitemutils.BuildBundleDir(bundle.ID, bundle.Slug)
	lock := ts.slugLock.LockKey(bundle.ID, req.ToolSlug)
	lock.Lock()
	defer lock.Unlock()

	finf, _ := bundleitemutils.BuildItemFileInfo(req.ToolSlug, req.Version)
	if err := ts.toolStore.DeleteFile(
		bundleitemutils.GetBundlePartitionFileKey(finf.FileName, dirInfo.DirName),
	); err != nil {
		return nil, err
	}
	slog.Info("deleteTool", "bundleID", req.BundleID, "slug", req.ToolSlug, "ver", req.Version)
	return &spec.DeleteToolResponse{}, nil
}

// InvokeTool locates a tool version and executes it according to its type.
// - Validates struct (ValidateTool), bundle/tool enabled state.
// - Dispatches to HTTP or Go runner with functional options constructed from the request body.
func (ts *ToolStore) InvokeTool(
	ctx context.Context,
	req *spec.InvokeToolRequest,
) (*spec.InvokeToolResponse, error) {
	if req == nil || req.Body == nil ||
		req.BundleID == "" || req.ToolSlug == "" || req.Version == "" {
		return nil, fmt.Errorf(
			"%w: bundleID, toolSlug, version and body required",
			spec.ErrInvalidRequest,
		)
	}
	if err := bundleitemutils.ValidateItemSlug(req.ToolSlug); err != nil {
		return nil, err
	}
	if err := bundleitemutils.ValidateItemVersion(req.Version); err != nil {
		return nil, err
	}
	args := json.RawMessage(req.Body.Args)

	// Load bundle and tool; re-use existing GetTool for a single source of truth.
	bundle, isBI, err := ts.getAnyBundle(ctx, req.BundleID)
	if err != nil {
		return nil, err
	}
	if !bundle.IsEnabled {
		return nil, fmt.Errorf("%w: bundle %s", spec.ErrBundleDisabled, req.BundleID)
	}

	gtResp, err := ts.GetTool(ctx, &spec.GetToolRequest{
		BundleID: req.BundleID,
		ToolSlug: req.ToolSlug,
		Version:  req.Version,
	})
	if err != nil {
		return nil, err
	}
	tool := gtResp.Body
	if tool == nil {
		return nil, fmt.Errorf("%w: nil tool body", spec.ErrToolNotFound)
	}
	if !tool.IsEnabled {
		return nil, fmt.Errorf(
			"%w: %s/%s@%s",
			spec.ErrToolDisabled,
			req.BundleID,
			req.ToolSlug,
			req.Version,
		)
	}

	// Defensive validation of the tool record.
	if err := storehelper.ValidateTool(tool); err != nil {
		return nil, fmt.Errorf("tool validation failed: %w", err)
	}

	var (
		outputs []spec.ToolStoreOutputUnion
		md      map[string]any
		isError bool
		errMsg  string
	)

	switch tool.Type {
	case spec.ToolTypeHTTP:
		var hopts []httprunner.HTTPOption
		if req.Body.HTTPOptions != nil {
			if req.Body.HTTPOptions.TimeoutMs > 0 {
				hopts = append(hopts, httprunner.WithHTTPTimeoutMs(req.Body.HTTPOptions.TimeoutMs))
			}
			if len(req.Body.HTTPOptions.ExtraHeaders) > 0 {
				hopts = append(
					hopts,
					httprunner.WithHTTPExtraHeaders(req.Body.HTTPOptions.ExtraHeaders),
				)
			}
			if len(req.Body.HTTPOptions.Secrets) > 0 {
				hopts = append(hopts, httprunner.WithHTTPSecrets(req.Body.HTTPOptions.Secrets))
			}
		}
		r, configErr := httprunner.NewHTTPToolRunner(*tool.HTTPImpl, hopts...)
		if configErr != nil {
			return nil, configErr
		}

		outputs, md, err = r.Run(ctx, args)

	case spec.ToolTypeGo:
		var gopts []llmtools.CallOption
		if req.Body.GoOptions != nil && req.Body.GoOptions.TimeoutMs != 0 {
			gopts = append(
				gopts,
				llmtools.WithCallTimeout(time.Duration(req.Body.GoOptions.TimeoutMs)*time.Millisecond),
			)
		}
		outputs, err = goregistry.CallUsingDefaultGoRegistry(
			ctx,
			strings.TrimSpace(tool.GoImpl.Func),
			args,
			gopts...,
		)

		md = map[string]any{
			"type":     "go",
			"funcName": tool.GoImpl.Func,
		}
	case spec.ToolTypeSDK:
		// SDK-backed tools are not invoked through ToolStore; they are surfaced to the model as provider server tools.
		// Invoking them directly is a misuse.
		return nil, fmt.Errorf("unsupported tool type for InvokeTool: %s", tool.Type)
	default:
		return nil, fmt.Errorf("unsupported tool type: %s", tool.Type)
	}
	if err != nil {
		// Treat errors from the underlying tool runner as tool-level errors that are surfaced in the InvokeToolResponse
		// instead of failing the entire call.
		isError = true
		errMsg = err.Error()

	}

	return &spec.InvokeToolResponse{
		Body: &spec.InvokeToolResponseBody{
			Outputs:      outputs,
			Meta:         md,
			IsBuiltIn:    isBI,
			IsError:      isError,
			ErrorMessage: errMsg,
		},
	}, nil
}

// GetTool retrieves a specific tool version.
func (ts *ToolStore) GetTool(
	ctx context.Context, req *spec.GetToolRequest,
) (*spec.GetToolResponse, error) {
	if req == nil || req.BundleID == "" || req.ToolSlug == "" || req.Version == "" {
		return nil, fmt.Errorf("%w: bundleID, toolSlug, version required", spec.ErrInvalidRequest)
	}
	bundle, isBI, err := ts.getAnyBundle(ctx, req.BundleID)
	if err != nil {
		return nil, err
	}
	if isBI {
		tool, err := ts.builtinData.GetBuiltInTool(ctx, bundle.ID, req.ToolSlug, req.Version)
		if err != nil {
			return nil, err
		}
		return &spec.GetToolResponse{Body: &tool}, nil
	}
	dirInfo, _ := bundleitemutils.BuildBundleDir(bundle.ID, bundle.Slug)
	lock := ts.slugLock.LockKey(bundle.ID, req.ToolSlug)
	lock.RLock()
	defer lock.RUnlock()

	finf, _, err := ts.findTool(dirInfo, req.ToolSlug, req.Version)
	if err != nil {
		return nil, err
	}
	raw, err := ts.toolStore.GetFileData(
		bundleitemutils.GetBundlePartitionFileKey(finf.FileName, dirInfo.DirName), false,
	)
	if err != nil {
		return nil, err
	}
	var t spec.Tool
	if err := jsonencdec.MapToStructWithJSONTags(raw, &t); err != nil {
		return nil, err
	}
	return &spec.GetToolResponse{Body: &t}, nil
}

// ListTools enumerates every stored tool version subject to filters.
func (ts *ToolStore) ListTools(
	ctx context.Context, req *spec.ListToolsRequest,
) (*spec.ListToolsResponse, error) {
	// Initialise / resume paging.
	tok := spec.ToolPageToken{}
	if req != nil && req.PageToken != "" {
		_ = func() error {
			t, err := jsonutil.Base64JSONDecode[spec.ToolPageToken](req.PageToken)
			if err == nil {
				tok = t
			}
			return err
		}()
	}
	if req != nil && req.PageToken == "" {
		tok.RecommendedPageSize = req.RecommendedPageSize
		tok.IncludeDisabled = req.IncludeDisabled
		tok.BundleIDs = slices.Clone(req.BundleIDs)
		slices.Sort(tok.BundleIDs)
		tok.Tags = slices.Clone(req.Tags)
		sort.Strings(tok.Tags)
	}

	pageHint := tok.RecommendedPageSize
	if pageHint <= 0 || pageHint > maxPageSizeTools {
		pageHint = defPageSizeTools
	}

	// Constant filters.
	bFilter := map[bundleitemutils.BundleID]struct{}{}
	for _, id := range tok.BundleIDs {
		bFilter[id] = struct{}{}
	}
	tagFilter := map[string]struct{}{}
	for _, t := range tok.Tags {
		tagFilter[t] = struct{}{}
	}

	include := func(bid bundleitemutils.BundleID, tool *spec.Tool) bool {
		if len(bFilter) > 0 {
			if _, ok := bFilter[bid]; !ok {
				return false
			}
		}
		if !tok.IncludeDisabled && !tool.IsEnabled {
			return false
		}
		if len(tagFilter) > 0 {
			match := false
			for _, tg := range tool.Tags {
				if _, ok := tagFilter[tg]; ok {
					match = true
					break
				}
			}
			if !match {
				return false
			}
		}
		return true
	}

	var out []spec.ToolListItem
	scannedUsers := false

	// Built-ins first.
	if ts.builtinData == nil {
		tok.BuiltInDone = true
	} else if !tok.BuiltInDone {
		biBundles, biTools, _ := ts.builtinData.ListBuiltInToolData(ctx)

		bidList := make([]bundleitemutils.BundleID, 0, len(biBundles))
		for bid := range biBundles {
			bidList = append(bidList, bid)
		}
		slices.Sort(bidList)

		for _, bid := range bidList {
			bundle := biBundles[bid]

			// Enforce bundle filter and enabled-state.
			if len(bFilter) > 0 {
				if _, ok := bFilter[bid]; !ok {
					continue
				}
			}
			if !tok.IncludeDisabled && !bundle.IsEnabled {
				continue
			}

			bslug := bundle.Slug

			ids := make([]bundleitemutils.ItemID, 0, len(biTools[bid]))
			for tid := range biTools[bid] {
				ids = append(ids, tid)
			}
			slices.SortFunc(ids, func(a, b bundleitemutils.ItemID) int {
				return strings.Compare(string(a), string(b))
			})
			for _, tid := range ids {
				tool := biTools[bid][tid]
				if include(bid, &tool) {
					out = append(out, spec.ToolListItem{
						BundleID:       bid,
						BundleSlug:     bslug,
						ToolSlug:       tool.Slug,
						ToolVersion:    tool.Version,
						IsBuiltIn:      true,
						ToolDefinition: tool,
					})
				}
			}
		}
		tok.BuiltInDone = true
	}

	allUserBundles, err := ts.readAllBundles(false)
	if err != nil {
		return nil, err
	}
	userBundles := allUserBundles.Bundles

	// User tools until pageHint filled.
	for len(out) < pageHint {
		files, next, err := ts.toolStore.ListFiles(
			mapstore.ListingConfig{
				PageSize:  fetchBatchTools,
				SortOrder: mapstore.SortOrderDescending,
			}, tok.DirTok,
		)
		if err != nil {
			return nil, err
		}
		for _, f := range files {
			fn := filepath.Base(f.BaseRelativePath)
			dir := filepath.Base(filepath.Dir(f.BaseRelativePath))

			if _, err := bundleitemutils.ParseItemFileName(fn); err != nil {
				continue
			}
			bdi, err := bundleitemutils.ParseBundleDir(dir)
			if err != nil {
				continue
			}
			bundle, ok := userBundles[bdi.ID]
			if !ok || isSoftDeletedTool(bundle) {
				// Unknown or soft-deleted bundle; skip its tools.
				continue
			}
			if len(bFilter) > 0 {
				if _, ok := bFilter[bdi.ID]; !ok {
					continue
				}
			}
			if !tok.IncludeDisabled && !bundle.IsEnabled {
				continue
			}

			raw, err := ts.toolStore.GetFileData(
				bundleitemutils.GetBundlePartitionFileKey(fn, dir), false,
			)
			if err != nil {
				continue
			}
			var tool spec.Tool
			if err := jsonencdec.MapToStructWithJSONTags(raw, &tool); err != nil {
				continue
			}
			if !include(bdi.ID, &tool) {
				continue
			}
			out = append(out, spec.ToolListItem{
				BundleID:       bdi.ID,
				BundleSlug:     bdi.Slug,
				ToolSlug:       tool.Slug,
				ToolVersion:    tool.Version,
				IsBuiltIn:      false,
				ToolDefinition: tool,
			})
		}
		tok.DirTok = next
		scannedUsers = true
		if tok.DirTok == "" {
			break
		}
	}

	var nextTok *string
	if tok.DirTok != "" || !scannedUsers { // Need more pages.
		s := jsonutil.Base64JSONEncode(tok)
		nextTok = &s
	}

	return &spec.ListToolsResponse{
		Body: &spec.ListToolsResponseBody{
			ToolListItems: out,
			NextPageToken: nextTok,
		},
	}, nil
}

// SearchTools executes a full-text query via FTS.
func (ts *ToolStore) SearchTools(
	ctx context.Context, req *spec.SearchToolsRequest,
) (*spec.SearchToolsResponse, error) {
	if req == nil || req.Query == "" {
		return nil, fmt.Errorf("%w: query required", spec.ErrInvalidRequest)
	}
	if ts.fts == nil {
		return nil, spec.ErrFTSDisabled
	}

	pageSize := defPageSizeTools
	if req.PageSize > 0 && req.PageSize <= maxPageSizeTools {
		pageSize = req.PageSize
	}
	searchPageSize := pageSize * 2 // Over-fetch to compensate filtering.

	hits, next, err := ts.fts.Search(ctx, req.Query, req.PageToken, searchPageSize)
	if err != nil {
		return nil, err
	}

	items := make([]spec.ToolListItem, 0, len(hits))
	for _, h := range hits {
		// Built-in?
		if strings.HasPrefix(h.ID, fts.BuiltInDocPrefix) {
			if ts.builtinData == nil {
				continue
			}
			rel := strings.TrimPrefix(h.ID, fts.BuiltInDocPrefix)
			dir := filepath.Dir(rel)
			file := filepath.Base(rel)

			bdi, err := bundleitemutils.ParseBundleDir(dir)
			if err != nil {
				continue
			}
			finf, err := bundleitemutils.ParseItemFileName(file)
			if err != nil {
				continue
			}
			bundle, err := ts.builtinData.GetBuiltInToolBundle(ctx, bdi.ID)
			if err != nil {
				continue
			}
			t, err := ts.builtinData.GetBuiltInTool(ctx, bdi.ID, finf.Slug, finf.Version)
			if err != nil {
				continue
			}
			if !req.IncludeDisabled && (!t.IsEnabled || !bundle.IsEnabled) {
				continue
			}
			items = append(items, spec.ToolListItem{
				BundleID:       bdi.ID,
				BundleSlug:     bdi.Slug,
				ToolSlug:       t.Slug,
				ToolVersion:    t.Version,
				IsBuiltIn:      true,
				ToolDefinition: t,
			})
			if len(items) >= pageSize {
				break
			}
			continue
		}

		// User-tool.
		dir := filepath.Base(filepath.Dir(h.ID))
		bdi, err := bundleitemutils.ParseBundleDir(dir)
		if err != nil {
			continue
		}
		bundle, _, err := ts.getAnyBundle(ctx, bdi.ID)
		if err != nil {
			continue
		}
		fn := filepath.Base(h.ID)
		_, err = bundleitemutils.ParseItemFileName(fn)
		if err != nil {
			continue
		}

		raw, err := ts.toolStore.GetFileData(
			bundleitemutils.GetBundlePartitionFileKey(fn, dir), false,
		)
		if err != nil {
			continue
		}
		var t spec.Tool
		if err := jsonencdec.MapToStructWithJSONTags(raw, &t); err != nil {
			continue
		}
		if !req.IncludeDisabled && (!t.IsEnabled || !bundle.IsEnabled) {
			continue
		}
		items = append(items, spec.ToolListItem{
			BundleID:       bdi.ID,
			BundleSlug:     bdi.Slug,
			ToolSlug:       t.Slug,
			ToolVersion:    t.Version,
			IsBuiltIn:      false,
			ToolDefinition: t,
		})
		if len(items) >= pageSize {
			break
		}
	}

	if len(items) < pageSize {
		next = ""
	}

	return &spec.SearchToolsResponse{
		Body: &spec.SearchToolsResponseBody{
			ToolListItems: items,
			NextPageToken: nullableStr(next),
		},
	}, nil
}

// findTool locates (slug, version) inside the given bundle directory.
func (ts *ToolStore) findTool(
	bdi bundleitemutils.BundleDirInfo,
	slug bundleitemutils.ItemSlug,
	version bundleitemutils.ItemVersion,
) (bundleitemutils.FileInfo, string, error) {
	if slug == "" || version == "" {
		return bundleitemutils.FileInfo{}, "", spec.ErrInvalidRequest
	}
	fi, err := bundleitemutils.BuildItemFileInfo(slug, version)
	if err != nil {
		return fi, "", err
	}
	key := bundleitemutils.GetBundlePartitionFileKey(fi.FileName, bdi.DirName)
	raw, err := ts.toolStore.GetFileData(key, false)
	if err != nil {
		return fi, "", fmt.Errorf("%w: %s", spec.ErrToolNotFound, slug)
	}
	if s, _ := raw["slug"].(string); s != string(slug) {
		return fi, "", fmt.Errorf("%w: %s", spec.ErrToolNotFound, slug)
	}
	return fi, filepath.Join(bdi.DirName, fi.FileName), nil
}

// getAnyBundle returns either a built-in or user bundle by ID.
func (ts *ToolStore) getAnyBundle(
	ctx context.Context,
	id bundleitemutils.BundleID,
) (spec.ToolBundle, bool, error) {
	if ts.builtinData != nil {
		if b, err := ts.builtinData.GetBuiltInToolBundle(ctx, id); err == nil {
			return b, true, nil
		}
	}
	b, err := ts.getUserBundle(id)
	return b, false, err
}

// getUserBundle fetches a non-soft-deleted user bundle.
func (ts *ToolStore) getUserBundle(id bundleitemutils.BundleID) (spec.ToolBundle, error) {
	all, err := ts.readAllBundles(false)
	if err != nil {
		return spec.ToolBundle{}, err
	}
	b, ok := all.Bundles[id]
	if !ok {
		return spec.ToolBundle{}, fmt.Errorf("%w: %s", spec.ErrBundleNotFound, id)
	}
	if isSoftDeletedTool(b) {
		return b, fmt.Errorf("%w: %s", spec.ErrBundleDeleting, id)
	}
	return b, nil
}

func (ts *ToolStore) startCleanupLoop() {
	ts.cleanOnce.Do(func() {
		ts.cleanKick = make(chan struct{}, 1)
		ts.cleanCtx, ts.cleanStop = context.WithCancel(context.Background())
		ts.wg.Go(func() {
			tick := time.NewTicker(cleanupIntervalTools)
			defer tick.Stop()
			defer func() {
				if r := recover(); r != nil {
					slog.Error(
						"panic in tool-bundle sweep",
						"err",
						r,
						"stack",
						string(debug.Stack()),
					)
				}
			}()

			ts.sweepSoftDeleted()

			for {
				select {
				case <-ts.cleanCtx.Done():
					return
				case <-tick.C:
				case <-ts.cleanKick:
				}
				ts.sweepSoftDeleted()
			}
		})
	})
}

// sweepSoftDeleted hard-deletes bundles whose grace period expired.
func (ts *ToolStore) sweepSoftDeleted() {
	ts.sweepMu.Lock()
	defer ts.sweepMu.Unlock()

	all, err := ts.readAllBundles(false)
	if err != nil {
		slog.Error("sweepSoftDeleted/readAllBundles", "err", err)
		return
	}
	now := time.Now().UTC()
	changed := false

	for id, b := range all.Bundles {
		if b.SoftDeletedAt == nil || b.SoftDeletedAt.IsZero() {
			continue
		}
		if now.Sub(*b.SoftDeletedAt) < softDeleteGraceTools {
			continue
		}

		dirInfo, _ := bundleitemutils.BuildBundleDir(b.ID, b.Slug)
		files, _, err := ts.toolStore.ListFiles(
			mapstore.ListingConfig{FilterPartitions: []string{dirInfo.DirName}, PageSize: 1}, "",
		)
		if err != nil || len(files) > 0 {
			slog.Warn("sweepSoftDeleted: bundle not empty", "bundleID", id)
			continue
		}

		delete(all.Bundles, id)
		changed = true
		_ = os.RemoveAll(filepath.Join(ts.baseDir, dirInfo.DirName))
		slog.Info("hard-deleted bundle", "bundleID", id)
	}

	if changed {
		if err := ts.writeAllBundles(all); err != nil {
			slog.Error("sweepSoftDeleted/writeAllBundles", "err", err)
		}
	}
}

// kickCleanupLoop triggers an immediate sweep.
func (ts *ToolStore) kickCleanupLoop() {
	select {
	case ts.cleanKick <- struct{}{}:
	default:
	}
}

func (ts *ToolStore) readAllBundles(force bool) (spec.AllBundles, error) {
	raw, err := ts.bundleStore.GetAll(force)
	if err != nil {
		return spec.AllBundles{}, err
	}
	var ab spec.AllBundles
	if err := jsonencdec.MapToStructWithJSONTags(raw, &ab); err != nil {
		return ab, err
	}
	return ab, nil
}

func (ts *ToolStore) writeAllBundles(ab spec.AllBundles) error {
	mp, _ := jsonencdec.StructWithJSONTagsToMap(ab)
	return ts.bundleStore.SetAll(mp)
}

// isSoftDeletedTool returns true if bundle is in soft-deleted state.
func isSoftDeletedTool(b spec.ToolBundle) bool {
	return b.SoftDeletedAt != nil && !b.SoftDeletedAt.IsZero()
}

// nullableStr returns &s unless s=="" in which case it returns nil.
func nullableStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
