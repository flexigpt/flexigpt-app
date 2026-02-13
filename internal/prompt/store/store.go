// Package store implements the prompt template storage and management logic.
// It provides CRUD operations for prompt bundles and templates, supports soft deletion, and background cleanup.
package store

import (
	"context"
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
	"github.com/flexigpt/flexigpt-app/internal/prompt/spec"
	"github.com/ppipada/mapstore-go"

	"github.com/ppipada/mapstore-go/jsonencdec"
	"github.com/ppipada/mapstore-go/uuidv7filename"
)

const (
	fetchBatch            = 512            // IO chunk for internal dir scans
	maxPageSize           = 256            // Maximum allowed page size for listing.
	defaultPageSize       = 25             // Default page size for listing.
	softDeleteGrace       = 48 * time.Hour // Grace period before hard-deleting a soft-deleted bundle.
	cleanupInterval       = 24 * time.Hour // Interval for periodic cleanup sweep.
	builtInSnapshotMaxAge = time.Hour
)

// PromptTemplateStore is the main store for prompt bundles and templates.
// It manages bundle and template CRUD, soft deletion, background cleanup.
type PromptTemplateStore struct {
	baseDir       string
	builtinData   *BuiltInData
	bundleStore   *mapstore.MapFileStore
	templateStore *mapstore.MapDirectoryStore
	pp            mapstore.PartitionProvider

	cleanOnce sync.Once
	cleanKick chan struct{}
	cleanCtx  context.Context
	cleanStop context.CancelFunc
	wg        sync.WaitGroup

	slugLock *slugLocks

	// Mutex to coordinate sweep operations with bundle modifications.
	sweepMu sync.RWMutex
}

// Option is a functional option for PromptTemplateStore.
type Option func(*PromptTemplateStore) error

func NewPromptTemplateStore(baseDir string, opts ...Option) (*PromptTemplateStore, error) {
	s := &PromptTemplateStore{
		baseDir: filepath.Clean(baseDir),
		pp:      &bundleitemutils.BundlePartitionProvider{},
	}
	for _, o := range opts {
		if err := o(s); err != nil {
			return nil, err
		}
	}
	ctx := context.Background()

	builtIn, err := NewBuiltInData(ctx, s.baseDir, builtInSnapshotMaxAge)
	if err != nil {
		return nil, err
	}
	s.builtinData = builtIn
	// Initialize bundle meta store (single JSON file).
	def, err := jsonencdec.StructWithJSONTagsToMap(
		spec.AllBundles{Bundles: map[bundleitemutils.BundleID]spec.PromptBundle{}},
	)
	if err != nil {
		return nil, err
	}
	s.bundleStore, err = mapstore.NewMapFileStore(
		filepath.Join(s.baseDir, spec.PromptBundlesMetaFileName),
		def,
		jsonencdec.JSONEncoderDecoder{},
		mapstore.WithCreateIfNotExists(true),
		mapstore.WithFileAutoFlush(true),
		mapstore.WithFileLogger(slog.Default()),
	)
	if err != nil {
		return nil, err
	}

	// Initialize template directory store (per-bundle folder).
	dirOpts := []mapstore.DirOption{mapstore.WithDirLogger(slog.Default())}
	s.templateStore, err = mapstore.NewMapDirectoryStore(
		s.baseDir,
		true,
		s.pp,
		jsonencdec.JSONEncoderDecoder{},
		dirOpts...)
	if err != nil {
		return nil, err
	}

	s.slugLock = newSlugLocks()
	s.startCleanupLoop()
	slog.Info("prompt-store ready", "baseDir", s.baseDir)
	return s, nil
}

// Close gracefully terminates the background cleanup goroutine.
func (s *PromptTemplateStore) Close() {
	if s.cleanStop != nil {
		s.cleanStop()
	}
	s.wg.Wait()
}

// PutPromptBundle creates or replaces a prompt bundle.
func (s *PromptTemplateStore) PutPromptBundle(
	ctx context.Context, req *spec.PutPromptBundleRequest,
) (*spec.PutPromptBundleResponse, error) {
	if req == nil || req.Body == nil ||
		req.BundleID == "" || req.Body.Slug == "" || req.Body.DisplayName == "" {
		return nil, fmt.Errorf("%w: id, slug & displayName are required", spec.ErrInvalidRequest)
	}
	if err := bundleitemutils.ValidateBundleSlug(req.Body.Slug); err != nil {
		return nil, err
	}
	if s.builtinData != nil {
		_, err := s.builtinData.GetBuiltInBundle(ctx, req.BundleID)
		if err == nil {
			return nil, fmt.Errorf("%w: bundleID: %q", spec.ErrBuiltInReadOnly, req.BundleID)
		}
		// We allow same slug with different IDs in bundle.
	}

	// Coordinate with sweep operations.
	s.sweepMu.Lock()
	defer s.sweepMu.Unlock()

	all, err := s.readAllBundles(false)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	createdAt := now

	if existing, ok := all.Bundles[req.BundleID]; ok {
		if !existing.CreatedAt.IsZero() {
			createdAt = existing.CreatedAt
		}
	}

	b := spec.PromptBundle{
		SchemaVersion: spec.SchemaVersion,
		ID:            req.BundleID,
		Slug:          req.Body.Slug,
		DisplayName:   req.Body.DisplayName,
		Description:   req.Body.Description,
		IsEnabled:     req.Body.IsEnabled,
		CreatedAt:     createdAt,
		ModifiedAt:    now,
		IsBuiltIn:     false,
		SoftDeletedAt: nil,
	}
	all.Bundles[req.BundleID] = b
	if err := s.writeAllBundles(all); err != nil {
		return nil, err
	}
	slog.Info("putPromptBundle", "id", req.BundleID)
	return &spec.PutPromptBundleResponse{}, nil
}

// PatchPromptBundle toggles the enabled flag of a bundle.
func (s *PromptTemplateStore) PatchPromptBundle(
	ctx context.Context, req *spec.PatchPromptBundleRequest,
) (*spec.PatchPromptBundleResponse, error) {
	if req == nil || req.Body == nil || req.BundleID == "" {
		return nil, fmt.Errorf("%w: bundleID required", spec.ErrInvalidRequest)
	}

	if s.builtinData != nil {
		if _, err := s.builtinData.GetBuiltInBundle(ctx, req.BundleID); err == nil {
			if _, err := s.builtinData.SetBundleEnabled(ctx, req.BundleID, req.Body.IsEnabled); err != nil {
				return nil, err
			}
			slog.Info("patchPromptBundle", "id", req.BundleID, "enabled", req.Body.IsEnabled)
			return &spec.PatchPromptBundleResponse{}, nil
		}
	}

	// Coordinate with sweep operations.
	s.sweepMu.Lock()
	defer s.sweepMu.Unlock()

	all, err := s.readAllBundles(false)
	if err != nil {
		return nil, err
	}
	bundle, ok := all.Bundles[req.BundleID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", spec.ErrBundleNotFound, req.BundleID)
	}
	bundle.IsEnabled = req.Body.IsEnabled
	bundle.ModifiedAt = time.Now().UTC()
	all.Bundles[req.BundleID] = bundle

	if err := s.writeAllBundles(all); err != nil {
		return nil, err
	}
	slog.Info("patchPromptBundle", "id", req.BundleID, "enabled", req.Body.IsEnabled)
	return &spec.PatchPromptBundleResponse{}, nil
}

// DeletePromptBundle soft-deletes a bundle if it is empty.
func (s *PromptTemplateStore) DeletePromptBundle(
	ctx context.Context, req *spec.DeletePromptBundleRequest,
) (*spec.DeletePromptBundleResponse, error) {
	if req == nil || req.BundleID == "" {
		return nil, fmt.Errorf("%w: bundleID required", spec.ErrInvalidRequest)
	}

	if s.builtinData != nil {
		_, err := s.builtinData.GetBuiltInBundle(ctx, req.BundleID)
		if err == nil {
			return nil, fmt.Errorf("%w: bundleID: %q", spec.ErrBuiltInReadOnly, req.BundleID)
		}
	}

	// Coordinate with sweep operations.
	s.sweepMu.Lock()
	defer s.sweepMu.Unlock()

	all, err := s.readAllBundles(false)
	if err != nil {
		return nil, err
	}
	bundle, ok := all.Bundles[req.BundleID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", spec.ErrBundleNotFound, req.BundleID)
	}
	if isSoftDeleted(bundle) {
		return nil, fmt.Errorf("%w: %s", spec.ErrBundleDeleting, req.BundleID)
	}

	dirInfo, derr := bundleitemutils.BuildBundleDir(bundle.ID, bundle.Slug)
	if derr != nil {
		return nil, derr
	}

	fileEntries, _, err := s.templateStore.ListFiles(
		mapstore.ListingConfig{
			FilterPartitions: []string{dirInfo.DirName},
			PageSize:         1,
		}, "",
	)
	if err != nil {
		return nil, err
	}
	if len(fileEntries) != 0 {
		return nil, fmt.Errorf("%w: %s", spec.ErrBundleNotEmpty, req.BundleID)
	}

	now := time.Now().UTC()
	bundle.IsEnabled = false
	bundle.SoftDeletedAt = &now
	all.Bundles[req.BundleID] = bundle

	if err := s.writeAllBundles(all); err != nil {
		return nil, err
	}
	slog.Info("deletePromptBundle request", "id", req.BundleID)
	s.kickCleanupLoop()
	return &spec.DeletePromptBundleResponse{}, nil
}

// ListPromptBundles lists prompt bundles with optional filtering and pagination.
// If PageToken is provided its embedded parameters override the request body.
func (s *PromptTemplateStore) ListPromptBundles(
	ctx context.Context,
	req *spec.ListPromptBundlesRequest,
) (*spec.ListPromptBundlesResponse, error) {
	// Resolve paging / filter parameters.
	var (
		pageSize        = defaultPageSize
		wantIDs         = map[bundleitemutils.BundleID]struct{}{}
		includeDisabled bool
		cursorMod       time.Time
		cursorID        bundleitemutils.BundleID
	)

	// Token present? -> authoritative.
	if req != nil && req.PageToken != "" {
		if tok, err := jsonutil.Base64JSONDecode[spec.BundlePageToken](req.PageToken); err == nil {
			pageSize = tok.PageSize
			if pageSize <= 0 || pageSize > maxPageSize {
				pageSize = defaultPageSize
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
	} else {
		// No token, fall back to request fields.
		if req != nil {
			if req.PageSize > 0 && req.PageSize <= maxPageSize {
				pageSize = req.PageSize
			}
			includeDisabled = req.IncludeDisabled
			for _, id := range req.BundleIDs {
				wantIDs[id] = struct{}{}
			}
		}
	}

	// Collect all bundles (built-in + user).
	allBundles := make([]spec.PromptBundle, 0)

	if s.builtinData != nil {
		bi, _, _ := s.builtinData.ListBuiltInData(ctx)
		for _, b := range bi {
			allBundles = append(allBundles, b)
		}
	}

	user, err := s.readAllBundles(false)
	if err != nil {
		return nil, err
	}
	for _, b := range user.Bundles {
		if isSoftDeleted(b) {
			continue
		}
		allBundles = append(allBundles, b)
	}

	// Apply filters.
	filtered := make([]spec.PromptBundle, 0, len(allBundles))
	for _, b := range allBundles {
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

	// Deterministic order.
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].ModifiedAt.Equal(filtered[j].ModifiedAt) {
			return filtered[i].ID < filtered[j].ID
		}
		return filtered[i].ModifiedAt.After(filtered[j].ModifiedAt)
	})

	// Guard against stale, malicious index.
	startIdx := 0
	if !cursorMod.IsZero() || cursorID != "" {
		for i, b := range filtered {
			if b.ModifiedAt.Before(cursorMod) ||
				(b.ModifiedAt.Equal(cursorMod) && b.ID == cursorID) {
				startIdx = i + 1
				break
			}
		}
		// If cursorMod is newer than every element, startIdx will be len(filtered).
	}

	end := min(startIdx+pageSize, len(filtered))
	nextToken := ""
	if end < len(filtered) {
		ids := make([]bundleitemutils.BundleID, 0, len(wantIDs))
		for id := range wantIDs {
			ids = append(ids, id)
		}
		slices.Sort(ids)

		nextToken = jsonutil.Base64JSONEncode(spec.BundlePageToken{
			CursorMod:       filtered[end-1].ModifiedAt.Format(time.RFC3339Nano),
			CursorID:        filtered[end-1].ID,
			BundleIDs:       ids,
			IncludeDisabled: includeDisabled,
			PageSize:        pageSize,
		})
	}

	return &spec.ListPromptBundlesResponse{
		Body: &spec.ListPromptBundlesResponseBody{
			PromptBundles: filtered[startIdx:end],
			NextPageToken: nullableStr(nextToken),
		},
	}, nil
}

// PutPromptTemplate creates a new prompt template version.
// Returns an error if the (slug, version) already exists.
func (s *PromptTemplateStore) PutPromptTemplate(
	ctx context.Context, req *spec.PutPromptTemplateRequest,
) (*spec.PutPromptTemplateResponse, error) {
	if req == nil || req.Body == nil {
		return nil, fmt.Errorf("%w: nil request/body", spec.ErrInvalidRequest)
	}
	if req.BundleID == "" || req.TemplateSlug == "" || req.Version == "" {
		return nil, fmt.Errorf(
			"%w: bundleID, templateSlug, version required",
			spec.ErrInvalidRequest,
		)
	}
	if err := bundleitemutils.ValidateItemSlug(req.TemplateSlug); err != nil {
		return nil, err
	}
	if err := bundleitemutils.ValidateItemVersion(req.Version); err != nil {
		return nil, err
	}
	if len(req.Body.Blocks) == 0 || req.Body.DisplayName == "" {
		return nil, fmt.Errorf("%w: displayName & ≥1 block required", spec.ErrInvalidRequest)
	}

	bundle, isBuiltIn, err := s.getAnyBundle(ctx, req.BundleID)
	if err != nil {
		return nil, err
	}
	if isBuiltIn {
		return nil, fmt.Errorf("%w: bundleID: %q", spec.ErrBuiltInReadOnly, req.BundleID)
	}
	if !bundle.IsEnabled {
		return nil, fmt.Errorf("%w: %s", spec.ErrBundleDisabled, req.BundleID)
	}
	dirInfo, derr := bundleitemutils.BuildBundleDir(bundle.ID, bundle.Slug)
	if derr != nil {
		return nil, derr
	}

	// Per-slug lock.
	lk := s.slugLock.lockKey(bundle.ID, req.TemplateSlug)
	lk.Lock()
	defer lk.Unlock()

	targetFN, ferr := bundleitemutils.BuildItemFileInfo(req.TemplateSlug, req.Version)
	if ferr != nil {
		return nil, ferr
	}
	existsList, _, _ := s.templateStore.ListFiles(
		mapstore.ListingConfig{
			FilterPartitions: []string{dirInfo.DirName},
			FilenamePrefix:   targetFN.FileName, PageSize: 10,
		}, "")

	// Check for exact filename match.
	for _, existing := range existsList {
		if filepath.Base(existing.BaseRelativePath) == targetFN.FileName {
			return nil, fmt.Errorf("%w: slug+version already exists", spec.ErrConflict)
		}
	}

	// Build object.
	now := time.Now().UTC()
	u, err := uuidv7filename.NewUUIDv7String()
	if err != nil {
		return nil, fmt.Errorf("uuid not available. err: %w", err)
	}

	tpl := spec.PromptTemplate{
		SchemaVersion: spec.SchemaVersion,
		ID:            bundleitemutils.ItemID(u),
		DisplayName:   req.Body.DisplayName,
		Slug:          req.TemplateSlug,
		Description:   req.Body.Description,
		IsEnabled:     req.Body.IsEnabled,
		Tags:          req.Body.Tags,
		Blocks:        req.Body.Blocks,
		Variables:     req.Body.Variables,
		Version:       req.Version,
		CreatedAt:     now,
		ModifiedAt:    now,
		IsBuiltIn:     false,
	}

	// Validate template structure.
	if err := validateTemplate(&tpl); err != nil {
		return nil, fmt.Errorf("template validation failed: %w", err)
	}

	mp, _ := jsonencdec.StructWithJSONTagsToMap(tpl)

	if err := s.templateStore.SetFileData(
		bundleitemutils.GetBundlePartitionFileKey(targetFN.FileName, dirInfo.DirName),
		mp,
	); err != nil {
		return nil, err
	}
	slog.Debug(
		"putPromptTemplate",
		"bundleId",
		req.BundleID,
		"slug",
		req.TemplateSlug,
		"version",
		req.Version,
	)
	return &spec.PutPromptTemplateResponse{}, nil
}

// DeletePromptTemplate hard-deletes a template version.
func (s *PromptTemplateStore) DeletePromptTemplate(
	ctx context.Context, req *spec.DeletePromptTemplateRequest,
) (*spec.DeletePromptTemplateResponse, error) {
	if req == nil || req.TemplateSlug == "" || req.BundleID == "" || req.Version == "" {
		return nil, fmt.Errorf(
			"%w: bundleID, templateSlug, version required",
			spec.ErrInvalidRequest,
		)
	}
	if err := bundleitemutils.ValidateItemSlug(req.TemplateSlug); err != nil {
		return nil, err
	}
	if err := bundleitemutils.ValidateItemVersion(req.Version); err != nil {
		return nil, err
	}
	bundle, isBuiltIn, err := s.getAnyBundle(ctx, req.BundleID)
	if err != nil {
		return nil, err
	}
	if isBuiltIn {
		return nil, fmt.Errorf("%w: bundleID: %q", spec.ErrBuiltInReadOnly, req.BundleID)
	}

	dirInfo, derr := bundleitemutils.BuildBundleDir(bundle.ID, bundle.Slug)
	if derr != nil {
		return nil, derr
	}

	// Per-slug lock.
	lk := s.slugLock.lockKey(bundle.ID, req.TemplateSlug)
	lk.Lock()
	defer lk.Unlock()

	targetFN, ferr := bundleitemutils.BuildItemFileInfo(req.TemplateSlug, req.Version)
	if ferr != nil {
		return nil, ferr
	}
	if err := s.templateStore.DeleteFile(
		bundleitemutils.GetBundlePartitionFileKey(targetFN.FileName, dirInfo.DirName),
	); err != nil {
		return nil, err
	}
	slog.Info(
		"deletePromptTemplate",
		"bundleId",
		req.BundleID,
		"slug",
		req.TemplateSlug,
		"version",
		req.Version,
	)
	return &spec.DeletePromptTemplateResponse{}, nil
}

// PatchPromptTemplate enables or disables a template version.
func (s *PromptTemplateStore) PatchPromptTemplate(
	ctx context.Context, req *spec.PatchPromptTemplateRequest,
) (*spec.PatchPromptTemplateResponse, error) {
	if req == nil || req.Body == nil || req.BundleID == "" || req.TemplateSlug == "" ||
		req.Version == "" {
		return nil, fmt.Errorf(
			"%w: bundleID, templateSlug, version required",
			spec.ErrInvalidRequest,
		)
	}
	if err := bundleitemutils.ValidateItemSlug(req.TemplateSlug); err != nil {
		return nil, err
	}
	if err := bundleitemutils.ValidateItemVersion(req.Version); err != nil {
		return nil, err
	}
	bundle, isBuiltIn, err := s.getAnyBundle(ctx, req.BundleID)
	if err != nil {
		return nil, err
	}
	if isBuiltIn {
		_, err := s.builtinData.SetTemplateEnabled(
			ctx,
			bundle.ID,
			req.TemplateSlug,
			req.Version,
			req.Body.IsEnabled,
		)
		if err != nil {
			return nil, err
		}

		slog.Info(
			"patchPromptTemplate",
			"bundleId",
			req.BundleID,
			"slug",
			req.TemplateSlug,
			"version",
			req.Version,
			"enabled",
			req.Body.IsEnabled,
		)
		return &spec.PatchPromptTemplateResponse{}, nil
	}

	if !bundle.IsEnabled {
		return nil, fmt.Errorf("%w: %s", spec.ErrBundleDisabled, req.BundleID)
	}
	dirInfo, derr := bundleitemutils.BuildBundleDir(bundle.ID, bundle.Slug)
	if derr != nil {
		return nil, derr
	}

	// Per-slug lock.
	lk := s.slugLock.lockKey(bundle.ID, req.TemplateSlug)
	lk.Lock()
	defer lk.Unlock()

	targetFN, ferr := bundleitemutils.BuildItemFileInfo(req.TemplateSlug, req.Version)
	if ferr != nil {
		return nil, ferr
	}
	key := bundleitemutils.GetBundlePartitionFileKey(targetFN.FileName, dirInfo.DirName)
	raw, err := s.templateStore.GetFileData(key, false)
	if err != nil {
		return nil, err
	}
	var tpl spec.PromptTemplate
	if err := jsonencdec.MapToStructWithJSONTags(raw, &tpl); err != nil {
		return nil, err
	}
	tpl.IsEnabled = req.Body.IsEnabled
	tpl.ModifiedAt = time.Now()

	mp, _ := jsonencdec.StructWithJSONTagsToMap(tpl)
	if err := s.templateStore.SetFileData(key, mp); err != nil {
		return nil, err
	}
	slog.Info(
		"patchPromptTemplate",
		"bundleId",
		req.BundleID,
		"slug",
		req.TemplateSlug,
		"version",
		req.Version,
		"enabled",
		req.Body.IsEnabled,
	)
	return &spec.PatchPromptTemplateResponse{}, nil
}

// GetPromptTemplate returns a template version or the active version if version is omitted.
func (s *PromptTemplateStore) GetPromptTemplate(
	ctx context.Context, req *spec.GetPromptTemplateRequest,
) (*spec.GetPromptTemplateResponse, error) {
	if req == nil || req.BundleID == "" || req.TemplateSlug == "" || req.Version == "" {
		return nil, fmt.Errorf(
			"%w: bundleID, templateSlug, version required",
			spec.ErrInvalidRequest,
		)
	}
	if err := bundleitemutils.ValidateItemSlug(req.TemplateSlug); err != nil {
		return nil, err
	}

	bundle, isBuiltIn, err := s.getAnyBundle(ctx, req.BundleID)
	if err != nil {
		return nil, err
	}
	if isBuiltIn {
		tmpl, err := s.builtinData.GetBuiltInTemplate(
			ctx,
			req.BundleID,
			req.TemplateSlug,
			req.Version,
		)
		if err != nil {
			return nil, err
		}
		return &spec.GetPromptTemplateResponse{Body: &tmpl}, nil
	}

	dirInfo, derr := bundleitemutils.BuildBundleDir(bundle.ID, bundle.Slug)
	if derr != nil {
		return nil, derr
	}

	// Per-slug lock.
	lk := s.slugLock.lockKey(bundle.ID, req.TemplateSlug)
	lk.RLock()
	defer lk.RUnlock()

	var finf bundleitemutils.FileInfo

	finf, _, err = s.findTemplate(dirInfo, req.TemplateSlug, req.Version)
	if err != nil {
		return nil, err
	}
	raw, err := s.templateStore.GetFileData(
		bundleitemutils.GetBundlePartitionFileKey(finf.FileName, dirInfo.DirName),
		false,
	)
	if err != nil {
		return nil, err
	}
	var tpl spec.PromptTemplate
	if err := jsonencdec.MapToStructWithJSONTags(raw, &tpl); err != nil {
		return nil, err
	}
	return &spec.GetPromptTemplateResponse{Body: &tpl}, nil
}

// ListPromptTemplates streams every stored version.
// Server-side filters are bundle-IDs, tags and include disabled.
func (s *PromptTemplateStore) ListPromptTemplates(
	ctx context.Context,
	req *spec.ListPromptTemplatesRequest,
) (*spec.ListPromptTemplatesResponse, error) {
	//  Restore / initialise paging state.
	tok := spec.TemplatePageToken{}
	if req != nil && req.PageToken != "" {
		// Second and later calls.
		_ = func() error {
			t, err := jsonutil.Base64JSONDecode[spec.TemplatePageToken](req.PageToken)
			if err == nil {
				tok = t
			}
			return err
		}() // Decode error is treated as start from scratch.
	}

	// First call, use request body.
	if req != nil && req.PageToken == "" {
		tok.RecommendedPageSize = req.RecommendedPageSize
		tok.IncludeDisabled = req.IncludeDisabled
		tok.BundleIDs = slices.Clone(req.BundleIDs)
		slices.Sort(tok.BundleIDs)
		tok.Tags = slices.Clone(req.Tags)
		sort.Strings(tok.Tags)
	}

	pageHint := tok.RecommendedPageSize
	if pageHint <= 0 || pageHint > maxPageSize {
		pageHint = defaultPageSize
	}

	//  Prepare helpers / constant filters
	bundleFilter := map[bundleitemutils.BundleID]struct{}{}
	for _, id := range tok.BundleIDs {
		bundleFilter[id] = struct{}{}
	}
	tagFilter := map[string]struct{}{}
	for _, tg := range tok.Tags {
		tagFilter[tg] = struct{}{}
	}

	addAllowed := func(bid bundleitemutils.BundleID, tpl *spec.PromptTemplate) bool {
		if len(bundleFilter) != 0 {
			if _, ok := bundleFilter[bid]; !ok {
				return false
			}
		}
		if !tpl.IsEnabled && !tok.IncludeDisabled {
			return false
		}
		if len(tagFilter) != 0 {
			hit := false
			for _, t := range tpl.Tags {
				if _, ok := tagFilter[t]; ok {
					hit = true
					break
				}
			}
			if !hit {
				return false
			}
		}
		return true
	}

	var out []spec.PromptTemplateListItem
	scannedUsers := false
	//  Emit built-ins once (can exceed pageHint).
	if s.builtinData == nil {
		tok.BuiltInDone = true
	} else if !tok.BuiltInDone {
		biBundles, biTpls, _ := s.builtinData.ListBuiltInData(ctx)

		// Deterministic ordering.
		bidList := make([]bundleitemutils.BundleID, 0, len(biBundles))
		for bid := range biBundles {
			bidList = append(bidList, bid)
		}
		slices.Sort(bidList)

		for _, bid := range bidList {
			bslug := biBundles[bid].Slug

			tidList := make([]bundleitemutils.ItemID, 0, len(biTpls[bid]))
			for tid := range biTpls[bid] {
				tidList = append(tidList, tid)
			}
			slices.SortFunc(tidList, func(a, b bundleitemutils.ItemID) int {
				return strings.Compare(string(a), string(b))
			})

			for _, templateID := range tidList {
				pt := biTpls[bid][templateID]
				if addAllowed(bid, &pt) {
					out = append(out, spec.PromptTemplateListItem{
						BundleID:        bid,
						BundleSlug:      bslug,
						TemplateSlug:    pt.Slug,
						TemplateVersion: pt.Version,
						IsBuiltIn:       true,
					})
				}
			}
		}
		tok.BuiltInDone = true
	}

	// Stream user templates until we reach pageHint.
	// Always consume whole directory pages so that no object can be skipped.
	for len(out) < pageHint {
		fileEntries, nxt, err := s.templateStore.ListFiles(
			mapstore.ListingConfig{
				PageSize:  fetchBatch,
				SortOrder: mapstore.SortOrderDescending,
			}, tok.DirTok)
		if err != nil {
			return nil, err
		}

		for _, p := range fileEntries {
			fn := filepath.Base(p.BaseRelativePath)
			dir := filepath.Base(filepath.Dir(p.BaseRelativePath))

			// Quick reject on malformed names.
			if _, err := bundleitemutils.ParseItemFileName(fn); err != nil {
				continue
			}
			bdi, err := bundleitemutils.ParseBundleDir(dir)
			if err != nil {
				continue
			}

			raw, err := s.templateStore.GetFileData(
				bundleitemutils.GetBundlePartitionFileKey(fn, dir), false)
			if err != nil {
				continue
			}
			var pt spec.PromptTemplate
			if err := jsonencdec.MapToStructWithJSONTags(raw, &pt); err != nil {
				continue
			}
			if !addAllowed(bdi.ID, &pt) {
				continue
			}

			out = append(out, spec.PromptTemplateListItem{
				BundleID:        bdi.ID,
				BundleSlug:      bdi.Slug,
				TemplateSlug:    pt.Slug,
				TemplateVersion: pt.Version,
				IsBuiltIn:       false,
			})
		}

		tok.DirTok = nxt
		scannedUsers = true
		if tok.DirTok == "" { // fully consumed
			break
		}
	}

	//  Next-page token (or terminate stream).
	var next *string
	if needMorePages(tok.DirTok, scannedUsers) {
		s := jsonutil.Base64JSONEncode(tok)
		next = &s
	}

	return &spec.ListPromptTemplatesResponse{
		Body: &spec.ListPromptTemplatesResponseBody{
			PromptTemplateListItems: out,
			NextPageToken:           next,
		},
	}, nil
}

func needMorePages(dirTok string, scannedUsers bool) bool {
	// More user files waiting        -> token.
	if dirTok != "" {
		return true
	}
	// We never scanned user files yet (built-ins overflow) -> token.
	if !scannedUsers {
		return true
	}
	// Otherwise we are done.
	return false
}

// findTemplate locates a template file by bundle-dir, slug **and** version.
// Both slug and version are required; wantVersion must NOT be empty.
func (s *PromptTemplateStore) findTemplate(
	bdi bundleitemutils.BundleDirInfo,
	slug bundleitemutils.ItemSlug,
	wantVersion bundleitemutils.ItemVersion,
) (fi bundleitemutils.FileInfo, fullPath string, err error) {
	// Input must be complete.
	if slug == "" || wantVersion == "" {
		return fi, fullPath, spec.ErrInvalidRequest
	}

	// Build the canonical filename and access it directly.
	fi, err = bundleitemutils.BuildItemFileInfo(slug, wantVersion)
	if err != nil {
		return fi, fullPath, err
	}

	key := bundleitemutils.GetBundlePartitionFileKey(fi.FileName, bdi.DirName)
	raw, err := s.templateStore.GetFileData(key, false)
	if err != nil {
		// File does not exist.
		return fi, fullPath, fmt.Errorf(
			"%w: bundle slug-%s, item slug-%s",
			spec.ErrTemplateNotFound,
			bdi.Slug,
			slug,
		)
	}

	// Minimal integrity check: the JSON must contain the same slug.
	if sVal, _ := raw["slug"].(string); sVal != string(slug) {
		return fi, fullPath, fmt.Errorf(
			"%w: bundle slug-%s, item slug-%s",
			spec.ErrTemplateNotFound,
			bdi.Slug,
			slug,
		)
	}

	// Success - return relative path (same format the old code used).
	return fi, filepath.Join(bdi.DirName, fi.FileName), nil
}

func (s *PromptTemplateStore) getAnyBundle(
	ctx context.Context,
	id bundleitemutils.BundleID,
) (bundle spec.PromptBundle, isBuiltIn bool, err error) {
	// Built-in?
	if s.builtinData != nil {
		if bundle, err = s.builtinData.GetBuiltInBundle(ctx, id); err == nil {
			return bundle, true, nil
		}
	}
	// User bundle?
	if bundle, err = s.getUserBundle(id); err == nil {
		return bundle, false, nil
	}

	return spec.PromptBundle{}, false, fmt.Errorf("%w: %s", spec.ErrBundleNotFound, id)
}

// getUserBundle returns an active bundle (i.e., not soft-deleted) by ID.
func (s *PromptTemplateStore) getUserBundle(
	id bundleitemutils.BundleID,
) (spec.PromptBundle, error) {
	all, err := s.readAllBundles(false)
	if err != nil {
		return spec.PromptBundle{}, err
	}

	b, ok := all.Bundles[id]
	if !ok {
		return spec.PromptBundle{}, fmt.Errorf("%w: %s", spec.ErrBundleNotFound, id)
	}

	if isSoftDeleted(b) {
		return b, fmt.Errorf("%w: %s", spec.ErrBundleDeleting, id)
	}
	return b, nil
}

// startCleanupLoop starts the background cleanup goroutine for hard-deleting bundles after the grace period.
func (s *PromptTemplateStore) startCleanupLoop() {
	s.cleanOnce.Do(func() {
		s.cleanKick = make(chan struct{}, 1)
		s.cleanCtx, s.cleanStop = context.WithCancel(context.Background())
		s.wg.Go(func() {
			ticker := time.NewTicker(cleanupInterval)
			defer ticker.Stop()
			defer func() {
				if r := recover(); r != nil {
					slog.Error("panic in bundle cleanup loop",
						"err", r,
						"stack", string(debug.Stack()))
				}
			}()

			// Run once at start-up to clean up bundles left after a crash.
			s.sweepSoftDeleted()

			for {
				select {
				case <-s.cleanCtx.Done():
					return
				case <-ticker.C:
					// Periodic sweep.
				case <-s.cleanKick:
					// Explicit sweep triggered by DeletePromptBundle.
				}
				s.sweepSoftDeleted()
			}
		})
	})
}

// sweepSoftDeleted performs hard deletion of bundles that have been soft-deleted and are past the grace period.
// Uses sweepMu to coordinate with bundle operations and prevent race conditions.
func (s *PromptTemplateStore) sweepSoftDeleted() {
	s.sweepMu.Lock()
	defer s.sweepMu.Unlock()

	all, err := s.readAllBundles(false)
	if err != nil {
		slog.Error("sweep - readAllBundles", "err", err)
		return
	}
	now := time.Now().UTC()
	changed := false

	for id, b := range all.Bundles {
		if b.SoftDeletedAt == nil || b.SoftDeletedAt.IsZero() {
			continue
		}
		if now.Sub(*b.SoftDeletedAt) < softDeleteGrace {
			continue
		}

		dirInfo, derr := bundleitemutils.BuildBundleDir(b.ID, b.Slug)
		if derr != nil {
			slog.Error(
				"sweep - bundleitemutils.BuildBundleDir failed.",
				"bundleId",
				id,
				"err",
				derr,
			)
			continue
		}

		fileEntries, _, err := s.templateStore.ListFiles(
			mapstore.ListingConfig{
				FilterPartitions: []string{dirInfo.DirName},
				PageSize:         1,
			}, "",
		)
		if err != nil || len(fileEntries) != 0 {
			slog.Warn("sweep - bundle still contains templates, skipping", "bundleId", id)
			continue
		}

		delete(all.Bundles, id)
		changed = true

		_ = os.RemoveAll(filepath.Join(s.baseDir, dirInfo.DirName))
		slog.Info("hard-deleted bundle", "bundleId", id)
	}
	if changed {
		if err := s.writeAllBundles(all); err != nil {
			slog.Error("sweep - writeAllBundles failed", "err", err)
		}
	}
}

// kickCleanupLoop signals the cleanup goroutine to run immediately.
func (s *PromptTemplateStore) kickCleanupLoop() {
	select {
	case s.cleanKick <- struct{}{}:
	default:
		// Queue already has a signal - that is good enough.
	}
}

// readAllBundles loads and decodes the meta-file.
func (s *PromptTemplateStore) readAllBundles(forceFetch bool) (spec.AllBundles, error) {
	raw, err := s.bundleStore.GetAll(forceFetch)
	if err != nil {
		return spec.AllBundles{}, err
	}
	var ab spec.AllBundles
	if err := jsonencdec.MapToStructWithJSONTags(raw, &ab); err != nil {
		return ab, err
	}
	return ab, nil
}

// writeAllBundles encodes and writes the strongly-typed value.
func (s *PromptTemplateStore) writeAllBundles(ab spec.AllBundles) error {
	mp, _ := jsonencdec.StructWithJSONTagsToMap(ab)
	return s.bundleStore.SetAll(mp)
}

// isSoftDeleted returns true if the bundle is soft-deleted.
func isSoftDeleted(b spec.PromptBundle) bool {
	return b.SoftDeletedAt != nil && !b.SoftDeletedAt.IsZero()
}

// nullableStr returns a pointer to the string, or nil if the string is empty.
func nullableStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
