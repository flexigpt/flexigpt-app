package skillstore

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/flexigpt/mapstore-go"
	"github.com/flexigpt/mapstore-go/jsonencdec"
	"github.com/flexigpt/mapstore-go/uuidv7filename"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/skillstore/spec"
)

const (
	skillsMaxPageSize     = 256
	skillsDefaultPageSize = 25

	softDeleteGraceSkills = 48 * time.Hour
	cleanupIntervalSkills = 24 * time.Hour

	builtInSnapshotMaxAgeSkills = time.Hour
)

// skillStoreSchema is the single-file persisted structure for user-managed
// bundles and skills. Built-ins are stored and overlaid separately.
type skillStoreSchema struct {
	SchemaVersion string `json:"schemaVersion"`

	Bundles map[bundleitemutils.BundleID]spec.SkillBundle              `json:"bundles"`
	Skills  map[bundleitemutils.BundleID]map[spec.SkillSlug]spec.Skill `json:"skills"`
}

// SkillStore owns durable Skill management state. It has no session, prompt,
// catalog, provider, or Agent Skills runtime lifecycle dependency.
type SkillStore struct {
	baseDir            string
	embeddedHydrateDir string

	userStore *mapstore.MapFileStore
	builtin   *BuiltInSkills

	writeMu               sync.Mutex
	mu                    sync.RWMutex
	embeddedMaterializeMu sync.Mutex

	cleanOnce sync.Once
	cleanKick chan struct{}
	cleanCtx  context.Context
	cleanStop context.CancelFunc
	wg        sync.WaitGroup
}

type skillStoreOptions struct {
	// EmbeddedHydrateDir is the store-managed materialization location for
	// immutable built-in Skill packages.
	embeddedHydrateDir string
}

type SkillStoreOption func(*skillStoreOptions) error

func WithEmbeddedHydrateDir(dir string) SkillStoreOption {
	return func(options *skillStoreOptions) error {
		options.embeddedHydrateDir = strings.TrimSpace(dir)
		return nil
	}
}

func NewSkillStore(baseDir string, opts ...SkillStoreOption) (*SkillStore, error) {
	if strings.TrimSpace(baseDir) == "" {
		return nil, fmt.Errorf("%w: baseDir is empty", errSkillInvalidRequest)
	}

	options := skillStoreOptions{}
	for _, option := range opts {
		if option == nil {
			continue
		}
		if err := option(&options); err != nil {
			return nil, err
		}
	}

	store := &SkillStore{baseDir: filepath.Clean(baseDir)}
	if err := os.MkdirAll(store.baseDir, 0o755); err != nil {
		return nil, err
	}

	ctx := context.Background()
	builtinSkills, err := NewBuiltInSkills(
		ctx,
		store.baseDir,
		builtInSnapshotMaxAgeSkills,
	)
	if err != nil {
		return nil, err
	}
	store.builtin = builtinSkills

	store.embeddedHydrateDir = options.embeddedHydrateDir
	if store.embeddedHydrateDir == "" {
		store.embeddedHydrateDir = filepath.Join(
			store.baseDir,
			"skills-embeddedfs-hydrated",
		)
	}
	store.embeddedHydrateDir = filepath.Clean(store.embeddedHydrateDir)

	if err := store.materializeBuiltInEmbeddedFS(ctx); err != nil {
		_ = store.builtin.Close()
		return nil, err
	}

	defaults, err := jsonencdec.StructWithJSONTagsToMap(skillStoreSchema{
		SchemaVersion: spec.SkillSchemaVersion,
		Bundles:       map[bundleitemutils.BundleID]spec.SkillBundle{},
		Skills:        map[bundleitemutils.BundleID]map[spec.SkillSlug]spec.Skill{},
	})
	if err != nil {
		_ = store.builtin.Close()
		return nil, err
	}

	store.userStore, err = mapstore.NewMapFileStore(
		filepath.Join(store.baseDir, spec.SkillBundlesMetaFileName),
		defaults,
		jsonencdec.JSONEncoderDecoder{},
		mapstore.WithCreateIfNotExists(true),
		mapstore.WithFileAutoFlush(true),
		mapstore.WithFileLogger(slog.Default()),
	)
	if err != nil {
		_ = store.builtin.Close()
		return nil, err
	}

	if err := store.ensureBaseSkillBundleHydrated(); err != nil {
		_ = store.userStore.Close()
		_ = store.builtin.Close()
		return nil, err
	}
	store.startCleanupLoop()

	slog.Info("skill-store ready", "baseDir", store.baseDir)
	return store, nil
}

func (s *SkillStore) Close() {
	if s == nil {
		return
	}
	if s.cleanStop != nil {
		s.cleanStop()
	}
	s.wg.Wait()
	if s.builtin != nil {
		_ = s.builtin.Close()
	}
	if s.userStore != nil {
		_ = s.userStore.Close()
	}
}

func (s *SkillStore) PutSkillBundle(
	ctx context.Context,
	req *spec.PutSkillBundleRequest,
) (*spec.PutSkillBundleResponse, error) {
	if req == nil || req.Body == nil || req.BundleID == "" {
		return nil, fmt.Errorf("%w: bundleID and body required", errSkillInvalidRequest)
	}
	if req.Body.Slug == "" || req.Body.DisplayName == "" {
		return nil, fmt.Errorf("%w: slug and displayName required", errSkillInvalidRequest)
	}
	if err := bundleitemutils.ValidateBundleSlug(req.Body.Slug); err != nil {
		return nil, err
	}
	if err := validateManagedPathSegment(string(req.BundleID), "bundleID"); err != nil {
		return nil, fmt.Errorf("%w: %w", errSkillInvalidRequest, err)
	}
	if s.builtin != nil {
		if _, err := s.builtin.GetBuiltInSkillBundle(ctx, req.BundleID); err == nil {
			return nil, fmt.Errorf("%w: bundleID %q", errSkillBuiltInReadOnly, req.BundleID)
		}
	}

	if err := s.withUserWrite(
		ctx,
		"putSkillBundle",
		func(snapshot *skillStoreSchema) error {
			now := time.Now().UTC()
			createdAt := now
			if existing, ok := snapshot.Bundles[req.BundleID]; ok {
				if isSoftDeletedSkillBundle(existing) {
					return fmt.Errorf("%w: %s", errSkillBundleDeleting, req.BundleID)
				}
				if !existing.CreatedAt.IsZero() {
					createdAt = existing.CreatedAt
				}
			}

			bundle := spec.SkillBundle{
				SchemaVersion: spec.SkillSchemaVersion,
				ID:            req.BundleID,
				Slug:          req.Body.Slug,
				DisplayName:   req.Body.DisplayName,
				Description:   req.Body.Description,
				IsEnabled:     req.Body.IsEnabled,
				IsBuiltIn:     false,
				CreatedAt:     createdAt,
				ModifiedAt:    now,
			}
			if err := validateSkillBundle(&bundle); err != nil {
				return err
			}
			snapshot.Bundles[req.BundleID] = bundle
			if snapshot.Skills[req.BundleID] == nil {
				snapshot.Skills[req.BundleID] = map[spec.SkillSlug]spec.Skill{}
			}
			return nil
		},
	); err != nil {
		return nil, err
	}

	slog.Info("putSkillBundle", "bundleID", req.BundleID)
	return &spec.PutSkillBundleResponse{}, nil
}

func (s *SkillStore) PatchSkillBundle(
	ctx context.Context,
	req *spec.PatchSkillBundleRequest,
) (*spec.PatchSkillBundleResponse, error) {
	if req == nil || req.Body == nil || req.BundleID == "" {
		return nil, fmt.Errorf("%w: bundleID and body required", errSkillInvalidRequest)
	}

	if s.builtin != nil {
		if _, err := s.builtin.GetBuiltInSkillBundle(ctx, req.BundleID); err == nil {
			s.writeMu.Lock()
			defer s.writeMu.Unlock()
			if _, err := s.builtin.SetSkillBundleEnabled(ctx, req.BundleID, req.Body.IsEnabled); err != nil {
				return nil, err
			}
			return &spec.PatchSkillBundleResponse{}, nil
		}
	}

	if err := s.withUserWrite(
		ctx,
		"patchSkillBundle",
		func(snapshot *skillStoreSchema) error {
			bundle, ok := snapshot.Bundles[req.BundleID]
			if !ok {
				return fmt.Errorf("%w: %s", errSkillBundleNotFound, req.BundleID)
			}
			if isSoftDeletedSkillBundle(bundle) {
				return fmt.Errorf("%w: %s", errSkillBundleDeleting, req.BundleID)
			}
			bundle.IsEnabled = req.Body.IsEnabled
			bundle.ModifiedAt = time.Now().UTC()
			snapshot.Bundles[req.BundleID] = bundle
			return nil
		},
	); err != nil {
		return nil, err
	}

	slog.Info("patchSkillBundle", "bundleID", req.BundleID, "enabled", req.Body.IsEnabled)
	return &spec.PatchSkillBundleResponse{}, nil
}

func (s *SkillStore) DeleteSkillBundle(
	ctx context.Context,
	req *spec.DeleteSkillBundleRequest,
) (*spec.DeleteSkillBundleResponse, error) {
	if req == nil || req.BundleID == "" {
		return nil, fmt.Errorf("%w: bundleID required", errSkillInvalidRequest)
	}
	if s.builtin != nil {
		if _, err := s.builtin.GetBuiltInSkillBundle(ctx, req.BundleID); err == nil {
			return nil, fmt.Errorf("%w: bundleID %q", errSkillBuiltInReadOnly, req.BundleID)
		}
	}

	if err := s.withUserWrite(
		ctx,
		"deleteSkillBundle",
		func(snapshot *skillStoreSchema) error {
			bundle, ok := snapshot.Bundles[req.BundleID]
			if !ok {
				return fmt.Errorf("%w: %s", errSkillBundleNotFound, req.BundleID)
			}
			if isSoftDeletedSkillBundle(bundle) {
				return fmt.Errorf("%w: %s", errSkillBundleDeleting, req.BundleID)
			}
			if len(snapshot.Skills[req.BundleID]) > 0 {
				return fmt.Errorf("%w: %s", errSkillBundleNotEmpty, req.BundleID)
			}
			now := time.Now().UTC()
			bundle.IsEnabled = false
			bundle.SoftDeletedAt = &now
			bundle.ModifiedAt = now
			snapshot.Bundles[req.BundleID] = bundle
			return nil
		},
	); err != nil {
		return nil, err
	}

	s.kickCleanupLoop()
	slog.Info("deleteSkillBundle", "bundleID", req.BundleID)
	return &spec.DeleteSkillBundleResponse{}, nil
}

func (s *SkillStore) ListSkillBundles(
	ctx context.Context,
	req *spec.ListSkillBundlesRequest,
) (*spec.ListSkillBundlesResponse, error) {
	var (
		pageSize        = skillsDefaultPageSize
		includeDisabled bool
		wantIDs         = map[bundleitemutils.BundleID]struct{}{}
		cursorMod       time.Time
		cursorID        bundleitemutils.BundleID
	)
	if req != nil && req.PageToken != "" {
		token, err := jsonutil.Base64JSONDecode[spec.SkillBundlePageToken](req.PageToken)
		if err != nil {
			return nil, fmt.Errorf("%w: bad pageToken", errSkillInvalidRequest)
		}
		pageSize = token.PageSize
		if pageSize <= 0 || pageSize > skillsMaxPageSize {
			pageSize = skillsDefaultPageSize
		}
		includeDisabled = token.IncludeDisabled
		if token.CursorMod != "" {
			parsed, err := time.Parse(time.RFC3339Nano, token.CursorMod)
			if err != nil {
				return nil, fmt.Errorf("%w: bad cursor time", errSkillInvalidRequest)
			}
			cursorMod = parsed
			cursorID = token.CursorID
		}
		for _, id := range token.BundleIDs {
			wantIDs[id] = struct{}{}
		}
	} else if req != nil {
		if req.PageSize > 0 && req.PageSize <= skillsMaxPageSize {
			pageSize = req.PageSize
		}
		includeDisabled = req.IncludeDisabled
		for _, id := range req.BundleIDs {
			wantIDs[id] = struct{}{}
		}
	}

	allBundles := make([]spec.SkillBundle, 0)
	if s.builtin != nil {
		bundles, _, err := s.builtin.ListBuiltInSkills(ctx)
		if err != nil {
			return nil, err
		}
		for _, bundle := range bundles {
			allBundles = append(allBundles, bundle)
		}
	}
	s.mu.RLock()
	user, err := s.readAllUser(false)
	s.mu.RUnlock()
	if err != nil {
		return nil, err
	}
	for _, bundle := range user.Bundles {
		if !isSoftDeletedSkillBundle(bundle) {
			allBundles = append(allBundles, bundle)
		}
	}

	filtered := make([]spec.SkillBundle, 0, len(allBundles))
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
	sort.Slice(filtered, func(left, right int) bool {
		if filtered[left].ModifiedAt.Equal(filtered[right].ModifiedAt) {
			return filtered[left].ID < filtered[right].ID
		}
		return filtered[left].ModifiedAt.After(filtered[right].ModifiedAt)
	})

	start := 0
	if !cursorMod.IsZero() {
		start = sort.Search(len(filtered), func(index int) bool {
			bundle := filtered[index]
			if bundle.ModifiedAt.Before(cursorMod) {
				return true
			}
			return bundle.ModifiedAt.Equal(cursorMod) && bundle.ID > cursorID
		})
	}
	end := min(start+pageSize, len(filtered))

	var nextToken *string
	if end < len(filtered) {
		ids := make([]bundleitemutils.BundleID, 0, len(wantIDs))
		for id := range wantIDs {
			ids = append(ids, id)
		}
		slices.Sort(ids)
		encoded := jsonutil.Base64JSONEncode(spec.SkillBundlePageToken{
			BundleIDs:       ids,
			IncludeDisabled: includeDisabled,
			PageSize:        pageSize,
			CursorMod:       filtered[end-1].ModifiedAt.Format(time.RFC3339Nano),
			CursorID:        filtered[end-1].ID,
		})
		nextToken = &encoded
	}

	return &spec.ListSkillBundlesResponse{Body: &spec.ListSkillBundlesResponseBody{
		SkillBundles:  filtered[start:end],
		NextPageToken: nextToken,
	}}, nil
}

func (s *SkillStore) PutSkill(
	ctx context.Context,
	req *spec.PutSkillRequest,
) (*spec.PutSkillResponse, error) {
	if req == nil || req.Body == nil || req.BundleID == "" || req.SkillSlug == "" {
		return nil, fmt.Errorf("%w: bundleID, skillSlug and body required", errSkillInvalidRequest)
	}
	if err := bundleitemutils.ValidateItemSlug(req.SkillSlug); err != nil {
		return nil, fmt.Errorf("%w: invalid skillSlug", errSkillInvalidRequest)
	}
	if req.Body.SkillType != spec.SkillTypeFS {
		return nil, fmt.Errorf("%w: only skillType=%q can be created", errSkillInvalidRequest, spec.SkillTypeFS)
	}
	if s.builtin != nil {
		if _, err := s.builtin.GetBuiltInSkillBundle(ctx, req.BundleID); err == nil {
			return nil, fmt.Errorf("%w: bundleID %q", errSkillBuiltInReadOnly, req.BundleID)
		}
	}

	if err := s.withUserWrite(ctx, "putSkill", func(snapshot *skillStoreSchema) error {
		bundle, ok := snapshot.Bundles[req.BundleID]
		if !ok {
			return fmt.Errorf("%w: %s", errSkillBundleNotFound, req.BundleID)
		}
		if isSoftDeletedSkillBundle(bundle) {
			return fmt.Errorf("%w: %s", errSkillBundleDeleting, req.BundleID)
		}
		if snapshot.Skills[req.BundleID] == nil {
			snapshot.Skills[req.BundleID] = map[spec.SkillSlug]spec.Skill{}
		}
		if _, exists := snapshot.Skills[req.BundleID][req.SkillSlug]; exists {
			return fmt.Errorf("%w: duplicate skillSlug in bundle", errSkillConflict)
		}
		id, err := uuidv7filename.NewUUIDv7String()
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		skill := spec.Skill{
			SchemaVersion: spec.SkillSchemaVersion,
			ID:            bundleitemutils.ItemID(id),
			Slug:          req.SkillSlug,
			Type:          req.Body.SkillType,
			Location:      req.Body.Location,
			Name:          req.Body.Name,
			DisplayName:   req.Body.DisplayName,
			Description:   req.Body.Description,
			Tags:          slices.Clone(req.Body.Tags),
			Presence:      &spec.SkillPresence{Status: spec.SkillPresenceUnknown},
			IsEnabled:     req.Body.IsEnabled,
			IsBuiltIn:     false,
			CreatedAt:     now,
			ModifiedAt:    now,
		}
		if err := validateSkill(&skill); err != nil {
			return err
		}
		snapshot.Skills[req.BundleID][req.SkillSlug] = skill
		return nil
	}); err != nil {
		return nil, err
	}

	slog.Info("putSkill", "bundleID", req.BundleID, "skillSlug", req.SkillSlug)
	return &spec.PutSkillResponse{}, nil
}

func (s *SkillStore) PatchSkill(
	ctx context.Context,
	req *spec.PatchSkillRequest,
) (*spec.PatchSkillResponse, error) {
	if req == nil || req.Body == nil || req.BundleID == "" || req.SkillSlug == "" {
		return nil, fmt.Errorf("%w: bundleID, skillSlug and body required", errSkillInvalidRequest)
	}
	if req.Body.IsEnabled == nil && req.Body.Location == nil && req.Body.DisplayName == nil &&
		req.Body.Description == nil &&
		req.Body.Tags == nil {
		return nil, fmt.Errorf("%w: empty patch", errSkillInvalidRequest)
	}
	if err := bundleitemutils.ValidateItemSlug(req.SkillSlug); err != nil {
		return nil, fmt.Errorf("%w: invalid skillSlug", errSkillInvalidRequest)
	}

	if s.builtin != nil {
		if _, err := s.builtin.GetBuiltInSkillBundle(ctx, req.BundleID); err == nil {
			if req.Body.Location != nil || req.Body.DisplayName != nil || req.Body.Description != nil ||
				req.Body.Tags != nil {
				return nil, fmt.Errorf("%w: cannot modify metadata for built-in", errSkillBuiltInReadOnly)
			}
			if req.Body.IsEnabled == nil {
				return nil, fmt.Errorf("%w: isEnabled required for built-in patch", errSkillInvalidRequest)
			}
			s.writeMu.Lock()
			defer s.writeMu.Unlock()
			if _, err := s.builtin.SetSkillEnabled(ctx, req.BundleID, req.SkillSlug, *req.Body.IsEnabled); err != nil {
				return nil, err
			}
			return &spec.PatchSkillResponse{}, nil
		}
	}

	if err := s.withUserWrite(ctx, "patchSkill", func(snapshot *skillStoreSchema) error {
		bundle, ok := snapshot.Bundles[req.BundleID]
		if !ok {
			return fmt.Errorf("%w: %s", errSkillBundleNotFound, req.BundleID)
		}
		if isSoftDeletedSkillBundle(bundle) {
			return fmt.Errorf("%w: %s", errSkillBundleDeleting, req.BundleID)
		}
		values := snapshot.Skills[req.BundleID]
		current, ok := values[req.SkillSlug]
		if !ok {
			return fmt.Errorf("%w: %s", errSkillNotFound, req.SkillSlug)
		}

		target := current
		if req.Body.IsEnabled != nil {
			target.IsEnabled = *req.Body.IsEnabled
		}
		if req.Body.Location != nil {
			if strings.TrimSpace(*req.Body.Location) == "" {
				return fmt.Errorf("%w: location cannot be empty", errSkillInvalidRequest)
			}
			if target.Location != *req.Body.Location {
				target.Location = *req.Body.Location
				target.Presence = &spec.SkillPresence{Status: spec.SkillPresenceUnknown}
			}
		}
		if req.Body.DisplayName != nil {
			target.DisplayName = *req.Body.DisplayName
		}
		if req.Body.Description != nil {
			target.Description = *req.Body.Description
		}
		if req.Body.Tags != nil {
			target.Tags = slices.Clone(*req.Body.Tags)
		}
		target.ModifiedAt = time.Now().UTC()
		if err := validateSkill(&target); err != nil {
			return err
		}
		values[req.SkillSlug] = target
		snapshot.Skills[req.BundleID] = values
		return nil
	}); err != nil {
		return nil, err
	}

	slog.Info("patchSkill", "bundleID", req.BundleID, "skillSlug", req.SkillSlug)
	return &spec.PatchSkillResponse{}, nil
}

func (s *SkillStore) DeleteSkill(
	ctx context.Context,
	req *spec.DeleteSkillRequest,
) (*spec.DeleteSkillResponse, error) {
	if req == nil || req.BundleID == "" || req.SkillSlug == "" {
		return nil, fmt.Errorf("%w: bundleID and skillSlug required", errSkillInvalidRequest)
	}
	if err := bundleitemutils.ValidateItemSlug(req.SkillSlug); err != nil {
		return nil, err
	}
	if s.builtin != nil {
		if _, err := s.builtin.GetBuiltInSkillBundle(ctx, req.BundleID); err == nil {
			return nil, fmt.Errorf("%w: built-in", errSkillBuiltInReadOnly)
		}
	}

	var deleted spec.Skill
	if err := s.withUserWrite(ctx, "deleteSkill", func(snapshot *skillStoreSchema) error {
		bundle, ok := snapshot.Bundles[req.BundleID]
		if !ok {
			return fmt.Errorf("%w: %s", errSkillBundleNotFound, req.BundleID)
		}
		if isSoftDeletedSkillBundle(bundle) {
			return fmt.Errorf("%w: %s", errSkillBundleDeleting, req.BundleID)
		}
		values := snapshot.Skills[req.BundleID]
		skill, ok := values[req.SkillSlug]
		if !ok {
			return fmt.Errorf("%w: %s", errSkillNotFound, req.SkillSlug)
		}
		deleted = skill
		delete(values, req.SkillSlug)
		snapshot.Skills[req.BundleID] = values
		return nil
	}); err != nil {
		return nil, err
	}

	if isManagedSkillPackageLocation(
		s.baseDir,
		string(req.BundleID),
		deleted.Name,
		deleted.Location,
	) {
		if err := os.RemoveAll(deleted.Location); err != nil {
			slog.Error(
				"delete managed Skill package failed",
				"location", deleted.Location,
				"error", err,
			)
		}
	}
	slog.Info("deleteSkill", "bundleID", req.BundleID, "skillSlug", req.SkillSlug)
	return &spec.DeleteSkillResponse{}, nil
}

func (s *SkillStore) GetSkill(
	ctx context.Context,
	req *spec.GetSkillRequest,
) (*spec.GetSkillResponse, error) {
	if req == nil || req.BundleID == "" || req.SkillSlug == "" {
		return nil, fmt.Errorf("%w: bundleID and skillSlug required", errSkillInvalidRequest)
	}
	if err := bundleitemutils.ValidateItemSlug(req.SkillSlug); err != nil {
		return nil, err
	}

	bundle, builtIn, err := s.getAnyBundle(ctx, req.BundleID)
	if err != nil {
		return nil, err
	}
	if !req.IncludeDisabled && !bundle.IsEnabled {
		return nil, fmt.Errorf("%w: %s", errSkillBundleDisabled, req.BundleID)
	}
	if builtIn {
		skill, err := s.builtin.GetBuiltInSkill(ctx, req.BundleID, req.SkillSlug)
		if err != nil {
			return nil, err
		}
		if !req.IncludeDisabled && !skill.IsEnabled {
			return nil, fmt.Errorf("%w: %s", errSkillDisabled, req.SkillSlug)
		}
		return &spec.GetSkillResponse{Body: &skill}, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	user, err := s.readAllUser(false)
	if err != nil {
		return nil, err
	}
	skill, ok := user.Skills[req.BundleID][req.SkillSlug]
	if !ok {
		return nil, fmt.Errorf("%w: %s", errSkillNotFound, req.SkillSlug)
	}
	if !req.IncludeDisabled && !skill.IsEnabled {
		return nil, fmt.Errorf("%w: %s", errSkillDisabled, req.SkillSlug)
	}
	return &spec.GetSkillResponse{Body: &skill}, nil
}
