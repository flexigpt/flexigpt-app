package store

import (
	"context"
	"errors"
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

	"github.com/flexigpt/flexigpt-app/internal/assistantpreset/spec"
	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/ppipada/mapstore-go"
	"github.com/ppipada/mapstore-go/jsonencdec"
	"github.com/ppipada/mapstore-go/uuidv7filename"
)

const (
	fetchBatchAssistantPresets            = 512
	maxPageSizeAssistantPresets           = 256
	defaultPageSizeAssistantPresets       = 25
	softDeleteGraceAssistantPresetBundles = 48 * time.Hour
	cleanupIntervalAssistantPresetBundles = 24 * time.Hour
	builtInAssistantPresetSnapshotMaxAge  = time.Hour
)

type AssistantPresetStore struct {
	baseDir string

	builtinData *BuiltInData
	builtInOpts []BuiltInDataOption

	bundleStore *mapstore.MapFileStore
	presetStore *mapstore.MapDirectoryStore
	pp          mapstore.PartitionProvider

	lookups ReferenceLookups

	slugLock *slugLocks

	cleanOnce sync.Once
	cleanKick chan struct{}
	cleanCtx  context.Context
	cleanStop context.CancelFunc
	wg        sync.WaitGroup

	sweepMu sync.RWMutex
}

type Option func(*AssistantPresetStore) error

func WithReferenceLookups(lookups ReferenceLookups) Option {
	return func(s *AssistantPresetStore) error {
		s.lookups = lookups
		return nil
	}
}

func WithModelPresetLookup(lookup ModelPresetLookup) Option {
	return func(s *AssistantPresetStore) error {
		s.lookups.ModelPresets = lookup
		return nil
	}
}

func WithPromptTemplateLookup(lookup PromptTemplateLookup) Option {
	return func(s *AssistantPresetStore) error {
		s.lookups.PromptTemplates = lookup
		return nil
	}
}

func WithToolSelectionLookup(lookup ToolSelectionLookup) Option {
	return func(s *AssistantPresetStore) error {
		s.lookups.ToolSelections = lookup
		return nil
	}
}

func WithSkillLookup(lookup SkillLookup) Option {
	return func(s *AssistantPresetStore) error {
		s.lookups.Skills = lookup
		return nil
	}
}

func WithBuiltInDataOptions(opts ...BuiltInDataOption) Option {
	return func(s *AssistantPresetStore) error {
		s.builtInOpts = append(s.builtInOpts, opts...)
		return nil
	}
}

func NewAssistantPresetStore(baseDir string, opts ...Option) (*AssistantPresetStore, error) {
	s := &AssistantPresetStore{
		baseDir: filepath.Clean(baseDir),
		pp:      &bundleitemutils.BundlePartitionProvider{},
	}

	for _, opt := range opts {
		if err := opt(s); err != nil {
			return nil, err
		}
	}

	ctx := context.Background()

	builtinData, err := NewBuiltInData(
		ctx,
		s.baseDir,
		builtInAssistantPresetSnapshotMaxAge,
		s.lookups,
		s.builtInOpts...,
	)
	if err != nil {
		return nil, err
	}
	s.builtinData = builtinData

	def, err := jsonencdec.StructWithJSONTagsToMap(
		spec.AllBundles{
			SchemaVersion: spec.SchemaVersion,
			Bundles:       map[bundleitemutils.BundleID]spec.AssistantPresetBundle{},
		},
	)
	if err != nil {
		return nil, err
	}

	s.bundleStore, err = mapstore.NewMapFileStore(
		filepath.Join(s.baseDir, spec.AssistantPresetBundlesMetaFileName),
		def,
		jsonencdec.JSONEncoderDecoder{},
		mapstore.WithCreateIfNotExists(true),
		mapstore.WithFileAutoFlush(true),
		mapstore.WithFileLogger(slog.Default()),
	)
	if err != nil {
		_ = s.builtinData.Close()
		return nil, err
	}

	dirOpts := []mapstore.DirOption{mapstore.WithDirLogger(slog.Default())}
	s.presetStore, err = mapstore.NewMapDirectoryStore(
		s.baseDir,
		true,
		s.pp,
		jsonencdec.JSONEncoderDecoder{},
		dirOpts...,
	)
	if err != nil {
		_ = s.bundleStore.Close()
		_ = s.builtinData.Close()
		return nil, err
	}

	s.slugLock = newSlugLocks()
	s.startCleanupLoop()

	slog.Info("assistant-preset-store ready", "baseDir", s.baseDir)
	return s, nil
}

func (s *AssistantPresetStore) Close() error {
	if s == nil {
		return nil
	}

	if s.cleanStop != nil {
		s.cleanStop()
	}
	s.wg.Wait()

	var firstErr error

	if s.builtinData != nil {
		if err := s.builtinData.Close(); err != nil {
			firstErr = err
		}
		s.builtinData = nil
	}

	if s.bundleStore != nil {
		if err := s.bundleStore.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		s.bundleStore = nil
	}

	if s.presetStore != nil {
		if err := s.presetStore.CloseAll(); err != nil && firstErr == nil {
			firstErr = err
		}
		s.presetStore = nil
	}

	return firstErr
}

// PutAssistantPresetBundle creates or replaces a bundle.
// Bundle slug is intentionally immutable once the bundle exists so directory
// addressing for existing version files cannot be orphaned.
func (s *AssistantPresetStore) PutAssistantPresetBundle(
	ctx context.Context,
	req *spec.PutAssistantPresetBundleRequest,
) (*spec.PutAssistantPresetBundleResponse, error) {
	if req == nil || req.Body == nil || req.BundleID == "" {
		return nil, fmt.Errorf("%w: bundleID and body required", spec.ErrInvalidRequest)
	}
	if strings.TrimSpace(req.Body.DisplayName) == "" || req.Body.Slug == "" {
		return nil, fmt.Errorf("%w: slug and displayName required", spec.ErrInvalidRequest)
	}
	if err := bundleitemutils.ValidateBundleSlug(req.Body.Slug); err != nil {
		return nil, err
	}

	if s.builtinData != nil {
		if _, err := s.builtinData.GetBuiltInBundle(ctx, req.BundleID); err == nil {
			return nil, fmt.Errorf("%w: bundleID=%q", spec.ErrBuiltInReadOnly, req.BundleID)
		}
	}

	s.sweepMu.Lock()
	defer s.sweepMu.Unlock()

	all, err := s.readAllBundles(false)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	createdAt := now

	if existing, ok := all.Bundles[req.BundleID]; ok {
		if existing.Slug != "" && existing.Slug != req.Body.Slug {
			return nil, fmt.Errorf(
				"%w: bundle slug is immutable once created",
				spec.ErrInvalidRequest,
			)
		}
		if !existing.CreatedAt.IsZero() {
			createdAt = existing.CreatedAt
		}
	}

	bundle := spec.AssistantPresetBundle{
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
	if err := validateAssistantPresetBundle(&bundle); err != nil {
		return nil, err
	}

	all.Bundles[req.BundleID] = bundle
	if err := s.writeAllBundles(all); err != nil {
		return nil, err
	}

	slog.Info("putAssistantPresetBundle", "bundleID", req.BundleID)
	return &spec.PutAssistantPresetBundleResponse{}, nil
}

func (s *AssistantPresetStore) PatchAssistantPresetBundle(
	ctx context.Context,
	req *spec.PatchAssistantPresetBundleRequest,
) (*spec.PatchAssistantPresetBundleResponse, error) {
	if req == nil || req.Body == nil || req.BundleID == "" {
		return nil, fmt.Errorf("%w: bundleID required", spec.ErrInvalidRequest)
	}

	if s.builtinData != nil {
		if _, err := s.builtinData.GetBuiltInBundle(ctx, req.BundleID); err == nil {
			if _, err := s.builtinData.SetAssistantPresetBundleEnabled(
				ctx,
				req.BundleID,
				req.Body.IsEnabled,
			); err != nil {
				return nil, err
			}
			slog.Info(
				"patchAssistantPresetBundle",
				"bundleID",
				req.BundleID,
				"enabled",
				req.Body.IsEnabled,
				"builtIn",
				true,
			)
			return &spec.PatchAssistantPresetBundleResponse{}, nil
		}
	}

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
	if isSoftDeletedAssistantPresetBundle(bundle) {
		return nil, fmt.Errorf("%w: %s", spec.ErrBundleDeleting, req.BundleID)
	}

	bundle.IsEnabled = req.Body.IsEnabled
	bundle.ModifiedAt = time.Now().UTC()

	if err := validateAssistantPresetBundle(&bundle); err != nil {
		return nil, err
	}

	all.Bundles[req.BundleID] = bundle
	if err := s.writeAllBundles(all); err != nil {
		return nil, err
	}

	slog.Info(
		"patchAssistantPresetBundle",
		"bundleID",
		req.BundleID,
		"enabled",
		req.Body.IsEnabled,
	)
	return &spec.PatchAssistantPresetBundleResponse{}, nil
}

func (s *AssistantPresetStore) DeleteAssistantPresetBundle(
	ctx context.Context,
	req *spec.DeleteAssistantPresetBundleRequest,
) (*spec.DeleteAssistantPresetBundleResponse, error) {
	if req == nil || req.BundleID == "" {
		return nil, fmt.Errorf("%w: bundleID required", spec.ErrInvalidRequest)
	}

	if s.builtinData != nil {
		if _, err := s.builtinData.GetBuiltInBundle(ctx, req.BundleID); err == nil {
			return nil, fmt.Errorf("%w: bundleID=%q", spec.ErrBuiltInReadOnly, req.BundleID)
		}
	}

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
	if isSoftDeletedAssistantPresetBundle(bundle) {
		return nil, fmt.Errorf("%w: %s", spec.ErrBundleDeleting, req.BundleID)
	}

	dirInfo, err := bundleitemutils.BuildBundleDir(bundle.ID, bundle.Slug)
	if err != nil {
		return nil, err
	}

	files, _, err := s.presetStore.ListFiles(
		mapstore.ListingConfig{
			FilterPartitions: []string{dirInfo.DirName},
			PageSize:         1,
		},
		"",
	)
	if err != nil {
		return nil, err
	}
	if len(files) != 0 {
		return nil, fmt.Errorf("%w: %s", spec.ErrBundleNotEmpty, req.BundleID)
	}

	now := time.Now().UTC()
	bundle.IsEnabled = false
	bundle.ModifiedAt = now
	bundle.SoftDeletedAt = &now

	if err := validateAssistantPresetBundle(&bundle); err != nil {
		return nil, err
	}

	all.Bundles[req.BundleID] = bundle
	if err := s.writeAllBundles(all); err != nil {
		return nil, err
	}

	s.kickCleanupLoop()
	slog.Info("deleteAssistantPresetBundle", "bundleID", req.BundleID)
	return &spec.DeleteAssistantPresetBundleResponse{}, nil
}

func (s *AssistantPresetStore) ListAssistantPresetBundles(
	ctx context.Context,
	req *spec.ListAssistantPresetBundlesRequest,
) (*spec.ListAssistantPresetBundlesResponse, error) {
	var (
		pageSize        = defaultPageSizeAssistantPresets
		includeDisabled bool
		wantIDs         = map[bundleitemutils.BundleID]struct{}{}
		cursorMod       time.Time
		cursorID        bundleitemutils.BundleID
	)

	if req != nil && req.PageToken != "" {
		if tok, err := jsonutil.Base64JSONDecode[spec.BundlePageToken](req.PageToken); err == nil {
			pageSize = tok.PageSize
			if pageSize <= 0 || pageSize > maxPageSizeAssistantPresets {
				pageSize = defaultPageSizeAssistantPresets
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
	} else if req != nil {
		if req.PageSize > 0 && req.PageSize <= maxPageSizeAssistantPresets {
			pageSize = req.PageSize
		}
		includeDisabled = req.IncludeDisabled
		for _, id := range req.BundleIDs {
			wantIDs[id] = struct{}{}
		}
	}

	allBundles := make([]spec.AssistantPresetBundle, 0)

	if s.builtinData != nil {
		builtInBundles, _, _ := s.builtinData.ListBuiltInData(ctx)
		for _, bundle := range builtInBundles {
			allBundles = append(allBundles, bundle)
		}
	}

	userBundles, err := s.readAllBundles(false)
	if err != nil {
		return nil, err
	}
	for _, bundle := range userBundles.Bundles {
		if isSoftDeletedAssistantPresetBundle(bundle) {
			continue
		}
		allBundles = append(allBundles, bundle)
	}

	filtered := make([]spec.AssistantPresetBundle, 0, len(allBundles))
	for _, bundle := range allBundles {
		if len(wantIDs) > 0 {
			if _, ok := wantIDs[bundle.ID]; !ok {
				continue
			}
		}
		if !includeDisabled && !bundle.IsEnabled {
			continue
		}
		filtered = append(filtered, bundle)
	}

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].ModifiedAt.Equal(filtered[j].ModifiedAt) {
			return filtered[i].ID < filtered[j].ID
		}
		return filtered[i].ModifiedAt.After(filtered[j].ModifiedAt)
	})

	start := 0
	if !cursorMod.IsZero() || cursorID != "" {
		start = len(filtered)
		for i, bundle := range filtered {
			if bundle.ModifiedAt.Before(cursorMod) ||
				(bundle.ModifiedAt.Equal(cursorMod) && bundle.ID > cursorID) {
				start = i
				break
			}
		}
	}

	end := min(start+pageSize, len(filtered))

	var next *string
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
		next = &encoded
	}

	return &spec.ListAssistantPresetBundlesResponse{
		Body: &spec.ListAssistantPresetBundlesResponseBody{
			AssistantPresetBundles: filtered[start:end],
			NextPageToken:          next,
		},
	}, nil
}

// PutAssistantPreset creates a new immutable assistant preset version.
func (s *AssistantPresetStore) PutAssistantPreset(
	ctx context.Context,
	req *spec.PutAssistantPresetRequest,
) (*spec.PutAssistantPresetResponse, error) {
	if req == nil || req.Body == nil {
		return nil, fmt.Errorf("%w: nil request/body", spec.ErrInvalidRequest)
	}
	if req.BundleID == "" || req.AssistantPresetSlug == "" || req.Version == "" {
		return nil, fmt.Errorf(
			"%w: bundleID, assistantPresetSlug and version required",
			spec.ErrInvalidRequest,
		)
	}
	if strings.TrimSpace(req.Body.DisplayName) == "" {
		return nil, fmt.Errorf("%w: displayName required", spec.ErrInvalidRequest)
	}
	if err := bundleitemutils.ValidateItemSlug(req.AssistantPresetSlug); err != nil {
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
		return nil, fmt.Errorf("%w: bundleID=%q", spec.ErrBuiltInReadOnly, req.BundleID)
	}
	if !bundle.IsEnabled {
		return nil, fmt.Errorf("%w: %s", spec.ErrBundleDisabled, req.BundleID)
	}

	dirInfo, err := bundleitemutils.BuildBundleDir(bundle.ID, bundle.Slug)
	if err != nil {
		return nil, err
	}

	lock := s.slugLock.lockKey(bundle.ID, req.AssistantPresetSlug)
	lock.Lock()
	defer lock.Unlock()

	targetFile, err := bundleitemutils.BuildItemFileInfo(req.AssistantPresetSlug, req.Version)
	if err != nil {
		return nil, err
	}

	existing, _, _ := s.presetStore.ListFiles(
		mapstore.ListingConfig{
			FilterPartitions: []string{dirInfo.DirName},
			FilenamePrefix:   targetFile.FileName,
			PageSize:         10,
		},
		"",
	)
	for _, ex := range existing {
		if filepath.Base(ex.BaseRelativePath) == targetFile.FileName {
			return nil, fmt.Errorf("%w: slug+version already exists", spec.ErrConflict)
		}
	}

	now := time.Now().UTC()
	uuid, err := uuidv7filename.NewUUIDv7String()
	if err != nil {
		return nil, fmt.Errorf("uuid not available: %w", err)
	}

	preset := spec.AssistantPreset{
		SchemaVersion:                    spec.SchemaVersion,
		ID:                               bundleitemutils.ItemID(uuid),
		Slug:                             req.AssistantPresetSlug,
		Version:                          req.Version,
		DisplayName:                      req.Body.DisplayName,
		Description:                      req.Body.Description,
		IsEnabled:                        req.Body.IsEnabled,
		IsBuiltIn:                        false,
		StartingModelPresetRef:           cloneJSONValue(req.Body.StartingModelPresetRef),
		StartingModelPresetPatch:         cloneJSONValue(req.Body.StartingModelPresetPatch),
		StartingIncludeModelSystemPrompt: cloneJSONValue(req.Body.StartingIncludeModelSystemPrompt),
		StartingInstructionTemplateRefs:  cloneJSONValue(req.Body.StartingInstructionTemplateRefs),
		StartingToolSelections:           cloneJSONValue(req.Body.StartingToolSelections),
		StartingEnabledSkillRefs:         cloneJSONValue(req.Body.StartingEnabledSkillRefs),
		CreatedAt:                        now,
		ModifiedAt:                       now,
	}

	if err := validateAssistantPreset(ctx, &preset, s.lookups); err != nil {
		return nil, fmt.Errorf("assistant preset validation failed: %w", err)
	}

	mp, err := jsonencdec.StructWithJSONTagsToMap(preset)
	if err != nil {
		return nil, err
	}

	if err := s.presetStore.SetFileData(
		bundleitemutils.GetBundlePartitionFileKey(targetFile.FileName, dirInfo.DirName),
		mp,
	); err != nil {
		return nil, err
	}

	slog.Info(
		"putAssistantPreset",
		"bundleID",
		req.BundleID,
		"slug",
		req.AssistantPresetSlug,
		"version",
		req.Version,
	)
	return &spec.PutAssistantPresetResponse{}, nil
}

func (s *AssistantPresetStore) PatchAssistantPreset(
	ctx context.Context,
	req *spec.PatchAssistantPresetRequest,
) (*spec.PatchAssistantPresetResponse, error) {
	if req == nil || req.Body == nil {
		return nil, fmt.Errorf("%w: nil request/body", spec.ErrInvalidRequest)
	}
	if req.BundleID == "" || req.AssistantPresetSlug == "" || req.Version == "" {
		return nil, fmt.Errorf(
			"%w: bundleID, assistantPresetSlug and version required",
			spec.ErrInvalidRequest,
		)
	}
	if err := bundleitemutils.ValidateItemSlug(req.AssistantPresetSlug); err != nil {
		return nil, err
	}
	if err := bundleitemutils.ValidateItemVersion(req.Version); err != nil {
		return nil, err
	}

	bundle, isBuiltIn, err := s.getAnyBundle(ctx, req.BundleID)
	if err != nil {
		return nil, err
	}
	if !bundle.IsEnabled {
		return nil, fmt.Errorf("%w: %s", spec.ErrBundleDisabled, req.BundleID)
	}

	if isBuiltIn {
		if _, err := s.builtinData.SetAssistantPresetEnabled(
			ctx,
			bundle.ID,
			req.AssistantPresetSlug,
			req.Version,
			req.Body.IsEnabled,
		); err != nil {
			return nil, err
		}
		slog.Info(
			"patchAssistantPreset",
			"bundleID",
			req.BundleID,
			"slug",
			req.AssistantPresetSlug,
			"version",
			req.Version,
			"enabled",
			req.Body.IsEnabled,
			"builtIn",
			true,
		)
		return &spec.PatchAssistantPresetResponse{}, nil
	}

	dirInfo, err := bundleitemutils.BuildBundleDir(bundle.ID, bundle.Slug)
	if err != nil {
		return nil, err
	}

	lock := s.slugLock.lockKey(bundle.ID, req.AssistantPresetSlug)
	lock.Lock()
	defer lock.Unlock()

	fileInfo, _, err := s.findAssistantPreset(dirInfo, req.AssistantPresetSlug, req.Version)
	if err != nil {
		return nil, err
	}

	key := bundleitemutils.GetBundlePartitionFileKey(fileInfo.FileName, dirInfo.DirName)
	raw, err := s.presetStore.GetFileData(key, false)
	if err != nil {
		return nil, err
	}

	var preset spec.AssistantPreset
	if err := jsonencdec.MapToStructWithJSONTags(raw, &preset); err != nil {
		return nil, err
	}
	if err := validateAssistantPresetStructure(&preset); err != nil {
		return nil, err
	}

	preset.IsEnabled = req.Body.IsEnabled
	preset.ModifiedAt = time.Now().UTC()

	mp, err := jsonencdec.StructWithJSONTagsToMap(preset)
	if err != nil {
		return nil, err
	}
	if err := s.presetStore.SetFileData(key, mp); err != nil {
		return nil, err
	}

	slog.Info(
		"patchAssistantPreset",
		"bundleID",
		req.BundleID,
		"slug",
		req.AssistantPresetSlug,
		"version",
		req.Version,
		"enabled",
		req.Body.IsEnabled,
	)
	return &spec.PatchAssistantPresetResponse{}, nil
}

func (s *AssistantPresetStore) DeleteAssistantPreset(
	ctx context.Context,
	req *spec.DeleteAssistantPresetRequest,
) (*spec.DeleteAssistantPresetResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", spec.ErrInvalidRequest)
	}
	if req.BundleID == "" || req.AssistantPresetSlug == "" || req.Version == "" {
		return nil, fmt.Errorf(
			"%w: bundleID, assistantPresetSlug and version required",
			spec.ErrInvalidRequest,
		)
	}
	if err := bundleitemutils.ValidateItemSlug(req.AssistantPresetSlug); err != nil {
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
		return nil, fmt.Errorf("%w: bundleID=%q", spec.ErrBuiltInReadOnly, req.BundleID)
	}

	dirInfo, err := bundleitemutils.BuildBundleDir(bundle.ID, bundle.Slug)
	if err != nil {
		return nil, err
	}

	lock := s.slugLock.lockKey(bundle.ID, req.AssistantPresetSlug)
	lock.Lock()
	defer lock.Unlock()

	fileInfo, _, err := s.findAssistantPreset(dirInfo, req.AssistantPresetSlug, req.Version)
	if err != nil {
		return nil, err
	}

	if err := s.presetStore.DeleteFile(
		bundleitemutils.GetBundlePartitionFileKey(fileInfo.FileName, dirInfo.DirName),
	); err != nil {
		return nil, err
	}

	slog.Info(
		"deleteAssistantPreset",
		"bundleID",
		req.BundleID,
		"slug",
		req.AssistantPresetSlug,
		"version",
		req.Version,
	)
	return &spec.DeleteAssistantPresetResponse{}, nil
}

func (s *AssistantPresetStore) GetAssistantPreset(
	ctx context.Context,
	req *spec.GetAssistantPresetRequest,
) (*spec.GetAssistantPresetResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", spec.ErrInvalidRequest)
	}
	if req.BundleID == "" || req.AssistantPresetSlug == "" || req.Version == "" {
		return nil, fmt.Errorf(
			"%w: bundleID, assistantPresetSlug and version required",
			spec.ErrInvalidRequest,
		)
	}
	if err := bundleitemutils.ValidateItemSlug(req.AssistantPresetSlug); err != nil {
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
		preset, err := s.builtinData.GetBuiltInAssistantPreset(
			ctx,
			bundle.ID,
			req.AssistantPresetSlug,
			req.Version,
		)
		if err != nil {
			return nil, err
		}
		return &spec.GetAssistantPresetResponse{Body: &preset}, nil
	}

	dirInfo, err := bundleitemutils.BuildBundleDir(bundle.ID, bundle.Slug)
	if err != nil {
		return nil, err
	}

	lock := s.slugLock.lockKey(bundle.ID, req.AssistantPresetSlug)
	lock.RLock()
	defer lock.RUnlock()

	fileInfo, _, err := s.findAssistantPreset(dirInfo, req.AssistantPresetSlug, req.Version)
	if err != nil {
		return nil, err
	}

	raw, err := s.presetStore.GetFileData(
		bundleitemutils.GetBundlePartitionFileKey(fileInfo.FileName, dirInfo.DirName),
		false,
	)
	if err != nil {
		return nil, err
	}

	var preset spec.AssistantPreset
	if err := jsonencdec.MapToStructWithJSONTags(raw, &preset); err != nil {
		return nil, err
	}
	if err := validateAssistantPresetStructure(&preset); err != nil {
		return nil, err
	}

	return &spec.GetAssistantPresetResponse{Body: &preset}, nil
}

func (s *AssistantPresetStore) ListAssistantPresets(
	ctx context.Context,
	req *spec.ListAssistantPresetsRequest,
) (*spec.ListAssistantPresetsResponse, error) {
	tok := spec.AssistantPresetPageToken{}
	if req != nil && req.PageToken != "" {
		_ = func() error {
			t, err := jsonutil.Base64JSONDecode[spec.AssistantPresetPageToken](req.PageToken)
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
	}

	pageHint := tok.RecommendedPageSize
	if pageHint <= 0 || pageHint > maxPageSizeAssistantPresets {
		pageHint = defaultPageSizeAssistantPresets
	}

	bundleFilter := make(map[bundleitemutils.BundleID]struct{}, len(tok.BundleIDs))
	for _, id := range tok.BundleIDs {
		bundleFilter[id] = struct{}{}
	}

	var out []spec.AssistantPresetListItem
	scannedUsers := false

	if s.builtinData == nil {
		tok.BuiltInDone = true
	} else if !tok.BuiltInDone {
		builtInBundles, builtInPresets, _ := s.builtinData.ListBuiltInData(ctx)

		bundleIDs := make([]bundleitemutils.BundleID, 0, len(builtInBundles))
		for bundleID := range builtInBundles {
			bundleIDs = append(bundleIDs, bundleID)
		}
		slices.Sort(bundleIDs)

		for _, bundleID := range bundleIDs {
			bundle := builtInBundles[bundleID]

			if len(bundleFilter) > 0 {
				if _, ok := bundleFilter[bundleID]; !ok {
					continue
				}
			}
			if !tok.IncludeDisabled && !bundle.IsEnabled {
				continue
			}

			presetIDs := make([]bundleitemutils.ItemID, 0, len(builtInPresets[bundleID]))
			for presetID := range builtInPresets[bundleID] {
				presetIDs = append(presetIDs, presetID)
			}
			slices.SortFunc(presetIDs, func(a, b bundleitemutils.ItemID) int {
				return strings.Compare(string(a), string(b))
			})

			for _, presetID := range presetIDs {
				preset := builtInPresets[bundleID][presetID]
				if !tok.IncludeDisabled && !preset.IsEnabled {
					continue
				}
				out = append(out, toAssistantPresetListItem(bundleID, bundle.Slug, preset))
			}
		}

		tok.BuiltInDone = true
	}

	allUserBundles, err := s.readAllBundles(false)
	if err != nil {
		return nil, err
	}
	userBundles := allUserBundles.Bundles

	for len(out) < pageHint {
		files, next, err := s.presetStore.ListFiles(
			mapstore.ListingConfig{
				PageSize:  fetchBatchAssistantPresets,
				SortOrder: mapstore.SortOrderDescending,
			},
			tok.DirTok,
		)
		if err != nil {
			return nil, err
		}

		for _, file := range files {
			fn := filepath.Base(file.BaseRelativePath)
			dir := filepath.Base(filepath.Dir(file.BaseRelativePath))

			itemInfo, err := bundleitemutils.ParseItemFileName(fn)
			if err != nil {
				continue
			}
			dirInfo, err := bundleitemutils.ParseBundleDir(dir)
			if err != nil {
				continue
			}

			bundle, ok := userBundles[dirInfo.ID]
			if !ok || isSoftDeletedAssistantPresetBundle(bundle) {
				continue
			}
			if bundle.Slug != dirInfo.Slug {
				continue
			}

			if len(bundleFilter) > 0 {
				if _, ok := bundleFilter[dirInfo.ID]; !ok {
					continue
				}
			}
			if !tok.IncludeDisabled && !bundle.IsEnabled {
				continue
			}

			raw, err := s.presetStore.GetFileData(
				bundleitemutils.GetBundlePartitionFileKey(fn, dir),
				false,
			)
			if err != nil {
				continue
			}

			var preset spec.AssistantPreset
			if err := jsonencdec.MapToStructWithJSONTags(raw, &preset); err != nil {
				continue
			}
			if err := validateAssistantPresetStructure(&preset); err != nil {
				continue
			}
			if preset.Slug != itemInfo.Slug || preset.Version != itemInfo.Version {
				continue
			}
			if !tok.IncludeDisabled && !preset.IsEnabled {
				continue
			}

			out = append(out, toAssistantPresetListItem(dirInfo.ID, bundle.Slug, preset))

		}

		tok.DirTok = next
		scannedUsers = true
		if tok.DirTok == "" {
			break
		}
	}

	var nextTok *string
	if tok.DirTok != "" || !scannedUsers {
		encoded := jsonutil.Base64JSONEncode(tok)
		nextTok = &encoded
	}

	return &spec.ListAssistantPresetsResponse{
		Body: &spec.ListAssistantPresetsResponseBody{
			AssistantPresetListItems: out,
			NextPageToken:            nextTok,
		},
	}, nil
}

func (s *AssistantPresetStore) findAssistantPreset(
	dirInfo bundleitemutils.BundleDirInfo,
	slug bundleitemutils.ItemSlug,
	version bundleitemutils.ItemVersion,
) (bundleitemutils.FileInfo, string, error) {
	if slug == "" || version == "" {
		return bundleitemutils.FileInfo{}, "", spec.ErrInvalidRequest
	}

	fileInfo, err := bundleitemutils.BuildItemFileInfo(slug, version)
	if err != nil {
		return fileInfo, "", err
	}

	key := bundleitemutils.GetBundlePartitionFileKey(fileInfo.FileName, dirInfo.DirName)
	raw, err := s.presetStore.GetFileData(key, false)
	if err != nil {
		return fileInfo, "", fmt.Errorf(
			"%w: bundleSlug=%s, assistantPresetSlug=%s, version=%s",
			spec.ErrAssistantPresetNotFound,
			dirInfo.Slug,
			slug,
			version,
		)
	}

	if gotSlug, _ := raw["slug"].(string); gotSlug != string(slug) {
		return fileInfo, "", fmt.Errorf(
			"%w: bundleSlug=%s, assistantPresetSlug=%s, version=%s",
			spec.ErrAssistantPresetNotFound,
			dirInfo.Slug,
			slug,
			version,
		)
	}
	if gotVersion, _ := raw["version"].(string); gotVersion != string(version) {
		return fileInfo, "", fmt.Errorf(
			"%w: bundleSlug=%s, assistantPresetSlug=%s, version=%s",
			spec.ErrAssistantPresetNotFound,
			dirInfo.Slug,
			slug,
			version,
		)
	}

	return fileInfo, filepath.Join(dirInfo.DirName, fileInfo.FileName), nil
}

func (s *AssistantPresetStore) getAnyBundle(
	ctx context.Context,
	id bundleitemutils.BundleID,
) (bundle spec.AssistantPresetBundle, isBuiltIn bool, err error) {
	if s.builtinData != nil {
		if bundle, err = s.builtinData.GetBuiltInBundle(ctx, id); err == nil {
			return bundle, true, nil
		}
	}

	if bundle, err = s.getUserBundle(id); err == nil {
		return bundle, false, nil
	} else if !errors.Is(err, spec.ErrBundleNotFound) {
		return bundle, false, err
	}

	return spec.AssistantPresetBundle{}, false, fmt.Errorf("%w: %s", spec.ErrBundleNotFound, id)
}

func (s *AssistantPresetStore) getUserBundle(
	id bundleitemutils.BundleID,
) (spec.AssistantPresetBundle, error) {
	all, err := s.readAllBundles(false)
	if err != nil {
		return spec.AssistantPresetBundle{}, err
	}

	bundle, ok := all.Bundles[id]
	if !ok {
		return spec.AssistantPresetBundle{}, fmt.Errorf("%w: %s", spec.ErrBundleNotFound, id)
	}
	if isSoftDeletedAssistantPresetBundle(bundle) {
		return bundle, fmt.Errorf("%w: %s", spec.ErrBundleDeleting, id)
	}
	return bundle, nil
}

func (s *AssistantPresetStore) startCleanupLoop() {
	s.cleanOnce.Do(func() {
		s.cleanKick = make(chan struct{}, 1)
		s.cleanCtx, s.cleanStop = context.WithCancel(context.Background())

		s.wg.Go(func() {
			ticker := time.NewTicker(cleanupIntervalAssistantPresetBundles)
			defer ticker.Stop()

			defer func() {
				if r := recover(); r != nil {
					slog.Error(
						"panic in assistant preset bundle cleanup loop",
						"err",
						r,
						"stack",
						string(debug.Stack()),
					)
				}
			}()

			s.sweepSoftDeleted()

			for {
				select {
				case <-s.cleanCtx.Done():
					return
				case <-ticker.C:
				case <-s.cleanKick:
				}
				s.sweepSoftDeleted()
			}
		})
	})
}

func (s *AssistantPresetStore) sweepSoftDeleted() {
	s.sweepMu.Lock()
	defer s.sweepMu.Unlock()

	all, err := s.readAllBundles(false)
	if err != nil {
		slog.Error("assistant preset sweep readAllBundles failed", "err", err)
		return
	}

	now := time.Now().UTC()
	changed := false

	for id, bundle := range all.Bundles {
		if bundle.SoftDeletedAt == nil || bundle.SoftDeletedAt.IsZero() {
			continue
		}
		if now.Sub(*bundle.SoftDeletedAt) < softDeleteGraceAssistantPresetBundles {
			continue
		}

		dirInfo, err := bundleitemutils.BuildBundleDir(bundle.ID, bundle.Slug)
		if err != nil {
			slog.Error(
				"assistant preset sweep BuildBundleDir failed",
				"bundleID",
				id,
				"err",
				err,
			)
			continue
		}

		files, _, err := s.presetStore.ListFiles(
			mapstore.ListingConfig{
				FilterPartitions: []string{dirInfo.DirName},
				PageSize:         1,
			},
			"",
		)
		if err != nil || len(files) != 0 {
			slog.Warn(
				"assistant preset sweep skipped non-empty bundle",
				"bundleID",
				id,
				"err",
				err,
			)
			continue
		}

		delete(all.Bundles, id)
		changed = true
		_ = os.RemoveAll(filepath.Join(s.baseDir, dirInfo.DirName))

		slog.Info("hard-deleted assistant preset bundle", "bundleID", id)
	}

	if changed {
		if err := s.writeAllBundles(all); err != nil {
			slog.Error("assistant preset sweep writeAllBundles failed", "err", err)
		}
	}
}

func (s *AssistantPresetStore) kickCleanupLoop() {
	select {
	case s.cleanKick <- struct{}{}:
	default:
	}
}

func (s *AssistantPresetStore) readAllBundles(forceFetch bool) (spec.AllBundles, error) {
	raw, err := s.bundleStore.GetAll(forceFetch)
	if err != nil {
		return spec.AllBundles{}, err
	}

	var all spec.AllBundles
	if err := jsonencdec.MapToStructWithJSONTags(raw, &all); err != nil {
		return all, err
	}

	if all.SchemaVersion == "" {
		all.SchemaVersion = spec.SchemaVersion
	}
	if all.Bundles == nil {
		all.Bundles = map[bundleitemutils.BundleID]spec.AssistantPresetBundle{}
	}

	return all, nil
}

func (s *AssistantPresetStore) writeAllBundles(all spec.AllBundles) error {
	all.SchemaVersion = spec.SchemaVersion
	if all.Bundles == nil {
		all.Bundles = map[bundleitemutils.BundleID]spec.AssistantPresetBundle{}
	}

	mp, err := jsonencdec.StructWithJSONTagsToMap(all)
	if err != nil {
		return err
	}
	return s.bundleStore.SetAll(mp)
}

// SetPreparedData is only for tests/runtime preparation helpers.
func SetPreparedData(
	s *AssistantPresetStore,
	fileName string,
	dirName string,
	data map[string]any,
) error {
	return s.presetStore.SetFileData(
		bundleitemutils.GetBundlePartitionFileKey(fileName, dirName),
		data,
	)
}

func isSoftDeletedAssistantPresetBundle(bundle spec.AssistantPresetBundle) bool {
	return bundle.SoftDeletedAt != nil && !bundle.SoftDeletedAt.IsZero()
}

func toAssistantPresetListItem(
	bundleID bundleitemutils.BundleID,
	bundleSlug bundleitemutils.BundleSlug,
	preset spec.AssistantPreset,
) spec.AssistantPresetListItem {
	return spec.AssistantPresetListItem{
		BundleID:               bundleID,
		BundleSlug:             bundleSlug,
		AssistantPresetSlug:    preset.Slug,
		AssistantPresetVersion: preset.Version,
		DisplayName:            preset.DisplayName,
		Description:            preset.Description,
		IsEnabled:              preset.IsEnabled,
		IsBuiltIn:              preset.IsBuiltIn,
		ModifiedAt:             timePtr(preset.ModifiedAt),
	}
}

func timePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	v := t
	return &v
}
