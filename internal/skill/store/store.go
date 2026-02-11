package store

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/skill/spec"
	"github.com/ppipada/mapstore-go"
	"github.com/ppipada/mapstore-go/jsonencdec"
	"github.com/ppipada/mapstore-go/uuidv7filename"
)

const (
	skillsMaxPageSize     = 256
	skillsDefaultPageSize = 25

	softDeleteGraceSkills = 48 * time.Hour
	cleanupIntervalSkills = 24 * time.Hour

	builtInSnapshotMaxAgeSkills = time.Hour
)

// skillStoreSchema is the single-file persisted structure (bundles + skills).
type skillStoreSchema struct {
	SchemaVersion string                                                     `json:"schemaVersion"`
	Bundles       map[bundleitemutils.BundleID]spec.SkillBundle              `json:"bundles"`
	Skills        map[bundleitemutils.BundleID]map[spec.SkillSlug]spec.Skill `json:"skills"`
}

// SkillStore provides CRUD and listing for Skill bundles and Skills (single JSON file).
type SkillStore struct {
	baseDir string

	userStore *mapstore.MapFileStore
	builtin   *BuiltInSkills

	mu sync.RWMutex // guards userStore read-modify-write

	// Cleanup loop plumbing.
	cleanOnce sync.Once
	cleanKick chan struct{}
	cleanCtx  context.Context
	cleanStop context.CancelFunc
	wg        sync.WaitGroup

	// Sweep coordination with CRUD ops.
	sweepMu sync.RWMutex
}

func NewSkillStore(baseDir string) (*SkillStore, error) {
	if strings.TrimSpace(baseDir) == "" {
		return nil, fmt.Errorf("%w: baseDir is empty", spec.ErrSkillInvalidRequest)
	}
	s := &SkillStore{baseDir: filepath.Clean(baseDir)}
	if err := os.MkdirAll(s.baseDir, 0o755); err != nil {
		return nil, err
	}

	ctx := context.Background()

	// Built-in overlay (optional but expected).
	bi, err := NewBuiltInSkills(ctx, s.baseDir, builtInSnapshotMaxAgeSkills)
	if err != nil {
		return nil, err
	}
	s.builtin = bi

	def, err := jsonencdec.StructWithJSONTagsToMap(skillStoreSchema{
		SchemaVersion: spec.SkillSchemaVersion,
		Bundles:       map[bundleitemutils.BundleID]spec.SkillBundle{},
		Skills:        map[bundleitemutils.BundleID]map[spec.SkillSlug]spec.Skill{},
	})
	if err != nil {
		return nil, err
	}

	s.userStore, err = mapstore.NewMapFileStore(
		filepath.Join(s.baseDir, spec.SkillBundlesMetaFileName),
		def,
		jsonencdec.JSONEncoderDecoder{},
		mapstore.WithCreateIfNotExists(true),
		mapstore.WithFileAutoFlush(true),
		mapstore.WithFileLogger(slog.Default()),
	)
	if err != nil {
		return nil, err
	}

	s.startCleanupLoop()
	slog.Info("skill-store ready", "baseDir", s.baseDir)
	return s, nil
}

func (s *SkillStore) Close() {
	if s.cleanStop != nil {
		s.cleanStop()
	}
	s.wg.Wait()
}

func (s *SkillStore) PutSkillBundle(
	ctx context.Context,
	req *spec.PutSkillBundleRequest,
) (*spec.PutSkillBundleResponse, error) {
	if req == nil || req.Body == nil || req.BundleID == "" {
		return nil, fmt.Errorf("%w: bundleID and body required", spec.ErrSkillInvalidRequest)
	}
	if req.Body.Slug == "" || req.Body.DisplayName == "" {
		return nil, fmt.Errorf("%w: slug and displayName required", spec.ErrSkillInvalidRequest)
	}
	if err := bundleitemutils.ValidateBundleSlug(req.Body.Slug); err != nil {
		return nil, err
	}

	// Built-ins are immutable.
	if s.builtin != nil {
		if _, err := s.builtin.GetBuiltInSkillBundle(ctx, req.BundleID); err == nil {
			return nil, fmt.Errorf("%w: bundleID %q", spec.ErrSkillBuiltInReadOnly, req.BundleID)
		}
	}

	s.sweepMu.Lock()
	defer s.sweepMu.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	all, err := s.readAllUser(false)
	if err != nil {
		return nil, err
	}
	if all.Bundles == nil {
		all.Bundles = map[bundleitemutils.BundleID]spec.SkillBundle{}
	}
	if all.Skills == nil {
		all.Skills = map[bundleitemutils.BundleID]map[spec.SkillSlug]spec.Skill{}
	}

	now := time.Now().UTC()
	createdAt := now
	if ex, ok := all.Bundles[req.BundleID]; ok {
		if isSoftDeletedSkillBundle(ex) {
			return nil, fmt.Errorf("%w: %s", spec.ErrSkillBundleDeleting, req.BundleID)
		}
		if !ex.CreatedAt.IsZero() {
			createdAt = ex.CreatedAt
		}
	}

	b := spec.SkillBundle{
		SchemaVersion: spec.SkillSchemaVersion,
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

	if err := validateSkillBundle(&b); err != nil {
		return nil, err
	}

	all.Bundles[req.BundleID] = b
	if _, ok := all.Skills[req.BundleID]; !ok {
		all.Skills[req.BundleID] = map[spec.SkillSlug]spec.Skill{}
	}

	if err := s.writeAllUser(all); err != nil {
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
		return nil, fmt.Errorf("%w: bundleID and body required", spec.ErrSkillInvalidRequest)
	}

	// Built-in path: overlay toggle.
	if s.builtin != nil {
		if _, err := s.builtin.GetBuiltInSkillBundle(ctx, req.BundleID); err == nil {
			if _, err := s.builtin.SetSkillBundleEnabled(ctx, req.BundleID, req.Body.IsEnabled); err != nil {
				return nil, err
			}
			slog.Info("patchSkillBundle (builtin)", "bundleID", req.BundleID, "enabled", req.Body.IsEnabled)
			return &spec.PatchSkillBundleResponse{}, nil
		}
	}

	s.sweepMu.Lock()
	defer s.sweepMu.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	all, err := s.readAllUser(false)
	if err != nil {
		return nil, err
	}
	b, ok := all.Bundles[req.BundleID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", spec.ErrSkillBundleNotFound, req.BundleID)
	}
	if isSoftDeletedSkillBundle(b) {
		return nil, fmt.Errorf("%w: %s", spec.ErrSkillBundleDeleting, req.BundleID)
	}

	b.IsEnabled = req.Body.IsEnabled
	b.ModifiedAt = time.Now().UTC()
	all.Bundles[req.BundleID] = b

	if err := s.writeAllUser(all); err != nil {
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
		return nil, fmt.Errorf("%w: bundleID required", spec.ErrSkillInvalidRequest)
	}

	// Built-ins are immutable.
	if s.builtin != nil {
		if _, err := s.builtin.GetBuiltInSkillBundle(ctx, req.BundleID); err == nil {
			return nil, fmt.Errorf("%w: bundleID %q", spec.ErrSkillBuiltInReadOnly, req.BundleID)
		}
	}

	s.sweepMu.Lock()
	defer s.sweepMu.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	all, err := s.readAllUser(false)
	if err != nil {
		return nil, err
	}
	b, ok := all.Bundles[req.BundleID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", spec.ErrSkillBundleNotFound, req.BundleID)
	}
	if isSoftDeletedSkillBundle(b) {
		return nil, fmt.Errorf("%w: %s", spec.ErrSkillBundleDeleting, req.BundleID)
	}

	// Must be empty.
	if all.Skills != nil {
		if sm := all.Skills[req.BundleID]; len(sm) > 0 {
			return nil, fmt.Errorf("%w: %s", spec.ErrSkillBundleNotEmpty, req.BundleID)
		}
	}

	now := time.Now().UTC()
	b.IsEnabled = false
	b.SoftDeletedAt = &now
	b.ModifiedAt = now
	all.Bundles[req.BundleID] = b

	if err := s.writeAllUser(all); err != nil {
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

	// Token overrides params.
	if req != nil && req.PageToken != "" {
		tok, err := jsonutil.Base64JSONDecode[spec.SkillBundlePageToken](req.PageToken)
		if err != nil {
			return nil, fmt.Errorf("%w: bad pageToken", spec.ErrSkillInvalidRequest)
		}
		pageSize = tok.PageSize
		if pageSize <= 0 || pageSize > skillsMaxPageSize {
			pageSize = skillsDefaultPageSize
		}
		includeDisabled = tok.IncludeDisabled
		if tok.CursorMod != "" {
			parsed, err := time.Parse(time.RFC3339Nano, tok.CursorMod)
			if err != nil {
				return nil, fmt.Errorf("%w: bad cursor time", spec.ErrSkillInvalidRequest)
			}
			cursorMod = parsed
			cursorID = tok.CursorID
		}
		for _, id := range tok.BundleIDs {
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

	// Collect built-in + user.
	allBundles := make([]spec.SkillBundle, 0)

	if s.builtin != nil {
		biBundles, _, err := s.builtin.ListBuiltInSkills(ctx)
		if err != nil {
			return nil, err
		}
		for _, b := range biBundles {
			allBundles = append(allBundles, b)
		}
	}

	s.mu.RLock()
	user, err := s.readAllUser(false)
	s.mu.RUnlock()
	if err != nil {
		return nil, err
	}
	for _, b := range user.Bundles {
		if isSoftDeletedSkillBundle(b) {
			continue
		}
		allBundles = append(allBundles, b)
	}

	// Filter.
	filtered := make([]spec.SkillBundle, 0, len(allBundles))
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

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].ModifiedAt.Equal(filtered[j].ModifiedAt) {
			return filtered[i].ID < filtered[j].ID
		}
		return filtered[i].ModifiedAt.After(filtered[j].ModifiedAt)
	})

	// Cursor.
	start := 0
	if !cursorMod.IsZero() {
		start = sort.Search(len(filtered), func(i int) bool {
			b := filtered[i]
			if b.ModifiedAt.Before(cursorMod) {
				return true
			}
			// Same timestamp: IDs are ascending, so resume at first ID > cursorID.
			return b.ModifiedAt.Equal(cursorMod) && b.ID > cursorID
		})
	}
	end := min(start+pageSize, len(filtered))

	var nextTok *string
	if end < len(filtered) {
		ids := make([]bundleitemutils.BundleID, 0, len(wantIDs))
		for id := range wantIDs {
			ids = append(ids, id)
		}
		slices.Sort(ids)

		next := jsonutil.Base64JSONEncode(spec.SkillBundlePageToken{
			BundleIDs:       ids,
			IncludeDisabled: includeDisabled,
			PageSize:        pageSize,
			CursorMod:       filtered[end-1].ModifiedAt.Format(time.RFC3339Nano),
			CursorID:        filtered[end-1].ID,
		})
		nextTok = &next
	}

	return &spec.ListSkillBundlesResponse{
		Body: &spec.ListSkillBundlesResponseBody{
			SkillBundles:  filtered[start:end],
			NextPageToken: nextTok,
		},
	}, nil
}

func (s *SkillStore) PutSkill(ctx context.Context, req *spec.PutSkillRequest) (*spec.PutSkillResponse, error) {
	if req == nil || req.Body == nil || req.BundleID == "" || req.SkillSlug == "" {
		return nil, fmt.Errorf("%w: bundleID, skillSlug and body required", spec.ErrSkillInvalidRequest)
	}
	if err := bundleitemutils.ValidateItemSlug(req.SkillSlug); err != nil {
		return nil, fmt.Errorf("%w: invalid skillSlug", spec.ErrSkillInvalidRequest)
	}

	if req.Body.SkillType != spec.SkillTypeFS {
		// User can only create fs skills.
		return nil, fmt.Errorf("%w: only skillType=%q can be created", spec.ErrSkillInvalidRequest, spec.SkillTypeFS)
	}

	// Built-in bundle IDs are read-only.
	if s.builtin != nil {
		if _, err := s.builtin.GetBuiltInSkillBundle(ctx, req.BundleID); err == nil {
			return nil, fmt.Errorf("%w: bundleID %q", spec.ErrSkillBuiltInReadOnly, req.BundleID)
		}
	}

	now := time.Now().UTC()
	uuid, err := uuidv7filename.NewUUIDv7String()
	if err != nil {
		return nil, err
	}

	s.sweepMu.Lock()
	defer s.sweepMu.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	all, err := s.readAllUser(false)
	if err != nil {
		return nil, err
	}

	// Re-check bundle state under lock (prevents TOCTOU).
	b, ok := all.Bundles[req.BundleID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", spec.ErrSkillBundleNotFound, req.BundleID)
	}
	if isSoftDeletedSkillBundle(b) {
		return nil, fmt.Errorf("%w: %s", spec.ErrSkillBundleDeleting, req.BundleID)
	}
	if !b.IsEnabled {
		return nil, fmt.Errorf("%w: %s", spec.ErrSkillBundleDisabled, req.BundleID)
	}

	if all.Skills == nil {
		all.Skills = map[bundleitemutils.BundleID]map[spec.SkillSlug]spec.Skill{}
	}
	if all.Skills[req.BundleID] == nil {
		all.Skills[req.BundleID] = map[spec.SkillSlug]spec.Skill{}
	}
	if _, exists := all.Skills[req.BundleID][req.SkillSlug]; exists {
		return nil, fmt.Errorf("%w: duplicate skillSlug in bundle", spec.ErrSkillConflict)
	}

	sk := spec.Skill{
		SchemaVersion: spec.SkillSchemaVersion,
		ID:            bundleitemutils.ItemID(uuid),
		Slug:          req.SkillSlug,

		Type:     req.Body.SkillType,
		Location: req.Body.Location,
		Name:     req.Body.Name,

		DisplayName: req.Body.DisplayName,
		Description: req.Body.Description,
		Tags:        req.Body.Tags,

		Presence: &spec.SkillPresence{Status: spec.SkillPresenceUnknown},

		IsEnabled: req.Body.IsEnabled,
		IsBuiltIn: false,

		CreatedAt:  now,
		ModifiedAt: now,
	}

	if err := validateSkill(&sk); err != nil {
		return nil, err
	}

	all.Skills[req.BundleID][req.SkillSlug] = sk
	if err := s.writeAllUser(all); err != nil {
		return nil, err
	}

	slog.Info("putSkill", "bundleID", req.BundleID, "skillSlug", req.SkillSlug)
	return &spec.PutSkillResponse{}, nil
}

func (s *SkillStore) PatchSkill(ctx context.Context, req *spec.PatchSkillRequest) (*spec.PatchSkillResponse, error) {
	if req == nil || req.Body == nil || req.BundleID == "" || req.SkillSlug == "" {
		return nil, fmt.Errorf("%w: bundleID, skillSlug and body required", spec.ErrSkillInvalidRequest)
	}
	if req.Body.IsEnabled == nil && req.Body.Location == nil {
		return nil, fmt.Errorf("%w: empty patch", spec.ErrSkillInvalidRequest)
	}

	if err := bundleitemutils.ValidateItemSlug(req.SkillSlug); err != nil {
		return nil, fmt.Errorf("%w: invalid skillSlug", spec.ErrSkillInvalidRequest)
	}

	// Built-in: allow enable/disable only (location is read-only).
	if s.builtin != nil {
		if _, err := s.builtin.GetBuiltInSkillBundle(ctx, req.BundleID); err == nil {
			// Any attempt to patch location on a built-in skill is forbidden (even empty string).
			if req.Body.Location != nil {
				return nil, fmt.Errorf("%w: cannot modify location for built-in", spec.ErrSkillBuiltInReadOnly)
			}
			// Built-in patch must explicitly include isEnabled.
			if req.Body.IsEnabled == nil {
				return nil, fmt.Errorf("%w: isEnabled required for built-in patch", spec.ErrSkillInvalidRequest)
			}

			enabled := *req.Body.IsEnabled
			if _, err := s.builtin.SetSkillEnabled(ctx, req.BundleID, req.SkillSlug, enabled); err != nil {
				return nil, err
			}
			slog.Info(
				"patchSkill (builtin)",
				"bundleID", req.BundleID,
				"skillSlug", req.SkillSlug,
				"enabled", enabled,
			)
			return &spec.PatchSkillResponse{}, nil
		}
	}

	// User path.
	s.sweepMu.Lock()
	defer s.sweepMu.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	all, err := s.readAllUser(false)
	if err != nil {
		return nil, err
	}
	b, ok := all.Bundles[req.BundleID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", spec.ErrSkillBundleNotFound, req.BundleID)
	}
	if isSoftDeletedSkillBundle(b) {
		return nil, fmt.Errorf("%w: %s", spec.ErrSkillBundleDeleting, req.BundleID)
	}
	if !b.IsEnabled {
		return nil, fmt.Errorf("%w: %s", spec.ErrSkillBundleDisabled, req.BundleID)
	}

	sm := all.Skills[req.BundleID]
	if sm == nil {
		return nil, fmt.Errorf("%w: %s", spec.ErrSkillNotFound, req.SkillSlug)
	}
	sk, ok := sm[req.SkillSlug]
	if !ok {
		return nil, fmt.Errorf("%w: %s", spec.ErrSkillNotFound, req.SkillSlug)
	}

	if req.Body.IsEnabled != nil {
		sk.IsEnabled = *req.Body.IsEnabled
	}

	// If client provided location, require it to be non-empty after trimming.
	if req.Body.Location != nil && strings.TrimSpace(*req.Body.Location) == "" {
		return nil, fmt.Errorf("%w: location cannot be empty", spec.ErrSkillInvalidRequest)
	}

	if req.Body.Location != nil && *req.Body.Location != sk.Location {
		sk.Location = *req.Body.Location
		// Invalidate presence on location change.
		sk.Presence = &spec.SkillPresence{Status: spec.SkillPresenceUnknown}
	}

	sk.ModifiedAt = time.Now().UTC()
	if err := validateSkill(&sk); err != nil {
		return nil, err
	}

	sm[req.SkillSlug] = sk
	all.Skills[req.BundleID] = sm

	if err := s.writeAllUser(all); err != nil {
		return nil, err
	}

	slog.Info("patchSkill", "bundleID", req.BundleID, "skillSlug", req.SkillSlug, "enabled", req.Body.IsEnabled)
	return &spec.PatchSkillResponse{}, nil
}

func (s *SkillStore) DeleteSkill(ctx context.Context, req *spec.DeleteSkillRequest) (*spec.DeleteSkillResponse, error) {
	if req == nil || req.BundleID == "" || req.SkillSlug == "" {
		return nil, fmt.Errorf("%w: bundleID and skillSlug required", spec.ErrSkillInvalidRequest)
	}
	if err := bundleitemutils.ValidateItemSlug(req.SkillSlug); err != nil {
		return nil, err
	}

	// Built-ins are immutable.
	if s.builtin != nil {
		if _, err := s.builtin.GetBuiltInSkillBundle(ctx, req.BundleID); err == nil {
			return nil, fmt.Errorf("%w: built-in", spec.ErrSkillBuiltInReadOnly)
		}
	}

	s.sweepMu.Lock()
	defer s.sweepMu.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	all, err := s.readAllUser(false)
	if err != nil {
		return nil, err
	}
	b, ok := all.Bundles[req.BundleID]

	if !ok {
		return nil, fmt.Errorf("%w: %s", spec.ErrSkillBundleNotFound, req.BundleID)
	}
	if isSoftDeletedSkillBundle(b) {
		return nil, fmt.Errorf("%w: %s", spec.ErrSkillBundleDeleting, req.BundleID)
	}
	sm := all.Skills[req.BundleID]
	if sm == nil {
		return nil, fmt.Errorf("%w: %s", spec.ErrSkillNotFound, req.SkillSlug)
	}
	sk, ok := sm[req.SkillSlug]
	if !ok {
		return nil, fmt.Errorf("%w: %s", spec.ErrSkillNotFound, req.SkillSlug)
	}

	if sk.Presence != nil && sk.Presence.Status == spec.SkillPresenceMissing {
		return nil, fmt.Errorf("%w: %s", spec.ErrSkillIsMissing, req.SkillSlug)
	}

	delete(sm, req.SkillSlug)
	all.Skills[req.BundleID] = sm

	if err := s.writeAllUser(all); err != nil {
		return nil, err
	}

	slog.Info("deleteSkill", "bundleID", req.BundleID, "skillSlug", req.SkillSlug)
	return &spec.DeleteSkillResponse{}, nil
}

func (s *SkillStore) GetSkill(ctx context.Context, req *spec.GetSkillRequest) (*spec.GetSkillResponse, error) {
	if req == nil || req.BundleID == "" || req.SkillSlug == "" {
		return nil, fmt.Errorf("%w: bundleID and skillSlug required", spec.ErrSkillInvalidRequest)
	}
	if err := bundleitemutils.ValidateItemSlug(req.SkillSlug); err != nil {
		return nil, err
	}

	b, isBI, err := s.getAnyBundle(ctx, req.BundleID)
	if err != nil {
		return nil, err
	}

	// Enforce disabled checks for Get (spec has ErrSkillBundleDisabled / ErrSkillDisabled).
	// (If you want Get to return disabled resources, remove these guards.)
	if !b.IsEnabled {
		return nil, fmt.Errorf("%w: %s", spec.ErrSkillBundleDisabled, req.BundleID)
	}

	if isBI {
		sk, err := s.builtin.GetBuiltInSkill(ctx, req.BundleID, req.SkillSlug)
		if err != nil {
			return nil, err
		}
		if !sk.IsEnabled {
			return nil, fmt.Errorf("%w: %s", spec.ErrSkillDisabled, req.SkillSlug)
		}
		return &spec.GetSkillResponse{Body: &sk}, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	all, err := s.readAllUser(false)
	if err != nil {
		return nil, err
	}
	sm := all.Skills[req.BundleID]
	if sm == nil {
		return nil, fmt.Errorf("%w: %s", spec.ErrSkillNotFound, req.SkillSlug)
	}
	sk, ok := sm[req.SkillSlug]
	if !ok {
		return nil, fmt.Errorf("%w: %s", spec.ErrSkillNotFound, req.SkillSlug)
	}
	if !sk.IsEnabled {
		return nil, fmt.Errorf("%w: %s", spec.ErrSkillDisabled, req.SkillSlug)
	}
	return &spec.GetSkillResponse{Body: &sk}, nil
}

func (s *SkillStore) ListSkills(ctx context.Context, req *spec.ListSkillsRequest) (*spec.ListSkillsResponse, error) {
	// Resume / init token.
	tok := spec.SkillPageToken{}
	if req != nil && req.PageToken != "" {
		t, err := jsonutil.Base64JSONDecode[spec.SkillPageToken](req.PageToken)
		if err != nil {
			return nil, fmt.Errorf("%w: bad pageToken", spec.ErrSkillInvalidRequest)
		}
		tok = t
	} else if req != nil {
		tok.RecommendedPageSize = req.RecommendedPageSize
		tok.IncludeDisabled = req.IncludeDisabled
		tok.IncludeMissing = req.IncludeMissing
		tok.BundleIDs = slices.Clone(req.BundleIDs)
		slices.Sort(tok.BundleIDs)
		tok.Types = slices.Clone(req.Types)
		slices.Sort(tok.Types)
	}

	if tok.Phase == "" {
		if s.builtin != nil {
			tok.Phase = spec.ListSkillPhaseBuiltIn
		} else {
			tok.Phase = spec.ListSkillPhaseUser
		}
	}
	if tok.Phase != spec.ListSkillPhaseBuiltIn && tok.Phase != spec.ListSkillPhaseUser {
		return nil, fmt.Errorf("%w: invalid phase", spec.ErrSkillInvalidRequest)
	}

	pageSize := tok.RecommendedPageSize
	if pageSize <= 0 || pageSize > skillsMaxPageSize {
		pageSize = skillsDefaultPageSize
	}

	bFilter := map[bundleitemutils.BundleID]struct{}{}
	for _, id := range tok.BundleIDs {
		bFilter[id] = struct{}{}
	}
	tFilter := map[spec.SkillType]struct{}{}
	for _, ty := range tok.Types {
		tFilter[ty] = struct{}{}
	}

	include := func(bundle spec.SkillBundle, sk spec.Skill) bool {
		if len(bFilter) > 0 {
			if _, ok := bFilter[bundle.ID]; !ok {
				return false
			}
		}
		if len(tFilter) > 0 {
			if _, ok := tFilter[sk.Type]; !ok {
				return false
			}
		}
		if !tok.IncludeDisabled && (!bundle.IsEnabled || !sk.IsEnabled) {
			return false
		}
		if !tok.IncludeMissing && sk.Presence != nil && sk.Presence.Status == spec.SkillPresenceMissing {
			return false
		}
		return true
	}

	out := make([]spec.SkillListItem, 0, pageSize)

	// Built-ins (PAGED).
	if tok.Phase == spec.ListSkillPhaseBuiltIn && s.builtin != nil && len(out) < pageSize {
		biBundles, biSkills, err := s.builtin.ListBuiltInSkills(ctx)
		if err != nil {
			return nil, err
		}

		// Cursor is bundleID|skillSlug (lexicographic in this ordering).
		var curBid bundleitemutils.BundleID
		var curSlug spec.SkillSlug
		if tok.BuiltInCursor != "" {
			parts := strings.Split(tok.BuiltInCursor, "|")
			if len(parts) != 2 {
				return nil, fmt.Errorf("%w: bad built-in cursor", spec.ErrSkillInvalidRequest)
			}
			curBid = bundleitemutils.BundleID(parts[0])
			curSlug = spec.SkillSlug(parts[1])
		}

		bids := make([]bundleitemutils.BundleID, 0, len(biBundles))
		for bid := range biBundles {
			bids = append(bids, bid)
		}
		slices.Sort(bids)

		moreBuiltins := false
		var lastBuiltInCursor string

	emitBuiltins:
		for _, bid := range bids {
			b := biBundles[bid]
			if len(bFilter) > 0 {
				if _, ok := bFilter[bid]; !ok {
					continue
				}
			}
			if !tok.IncludeDisabled && !b.IsEnabled {
				continue
			}

			sm := biSkills[bid]
			slugs := make([]spec.SkillSlug, 0, len(sm))
			for slug := range sm {
				slugs = append(slugs, slug)
			}
			slices.Sort(slugs)

			for _, slug := range slugs {
				// Seek “strictly after” the cursor in (bid asc, slug asc).
				if tok.BuiltInCursor != "" {
					if bid < curBid || (bid == curBid && slug <= curSlug) {
						continue
					}
				}

				sk := sm[slug]
				if include(b, sk) {
					out = append(out, spec.SkillListItem{
						BundleID:        b.ID,
						BundleSlug:      b.Slug,
						SkillSlug:       sk.Slug,
						IsBuiltIn:       true,
						SkillDefinition: cloneSkill(sk),
					})
					lastBuiltInCursor = string(bid) + "|" + string(slug)
				}
				if len(out) >= pageSize {
					// Determine if there are more built-ins after this point.
					moreBuiltins = true
					break emitBuiltins
				}
			}
		}

		if moreBuiltins {
			tok.Phase = spec.ListSkillPhaseBuiltIn
			tok.BuiltInCursor = lastBuiltInCursor
		} else {
			// Built-ins exhausted; move to users.
			tok.Phase = spec.ListSkillPhaseUser
			tok.BuiltInCursor = ""
			// Note: tok.DirTok is preserved (usually empty on first switch).
		}
	}

	// Users (paged) - only if we still need more items.
	if tok.Phase == spec.ListSkillPhaseUser && len(out) < pageSize {
		s.mu.RLock()
		user, err := s.readAllUser(false)
		s.mu.RUnlock()
		if err != nil {
			return nil, err
		}

		userItems := make([]spec.SkillListItem, 0)

		for bid, b := range user.Bundles {
			if isSoftDeletedSkillBundle(b) {
				continue
			}
			if len(bFilter) > 0 {
				if _, ok := bFilter[bid]; !ok {
					continue
				}
			}
			if !tok.IncludeDisabled && !b.IsEnabled {
				continue
			}

			sm := user.Skills[bid]
			for _, sk := range sm {
				if include(b, sk) {
					userItems = append(userItems, spec.SkillListItem{
						BundleID:        b.ID,
						BundleSlug:      b.Slug,
						SkillSlug:       sk.Slug,
						IsBuiltIn:       false,
						SkillDefinition: sk,
					})
				}
			}
		}

		sort.Slice(userItems, func(i, j int) bool {
			a := userItems[i].SkillDefinition
			b := userItems[j].SkillDefinition
			if a.ModifiedAt.Equal(b.ModifiedAt) {
				if userItems[i].BundleID == userItems[j].BundleID {
					return userItems[i].SkillSlug < userItems[j].SkillSlug
				}
				return userItems[i].BundleID < userItems[j].BundleID
			}
			return a.ModifiedAt.After(b.ModifiedAt)
		})

		start := 0
		if tok.DirTok != "" {
			c, err := parseSkillCursor(tok.DirTok)
			if err != nil {
				return nil, fmt.Errorf("%w: bad cursor", spec.ErrSkillInvalidRequest)
			}
			// Seek strictly after cursor in ordering:
			// (ModifiedAt desc, BundleID asc, SkillSlug asc).
			start = sort.Search(len(userItems), func(i int) bool {
				it := userItems[i]
				mt := it.SkillDefinition.ModifiedAt
				if mt.Before(c.ModTime) {
					return true
				}
				if mt.Equal(c.ModTime) {
					if it.BundleID > c.BundleID {
						return true
					}
					return it.BundleID == c.BundleID && it.SkillSlug > c.SkillSlug
				}
				return false
			})
		}

		need := pageSize - len(out)
		end := min(start+need, len(userItems))

		for i := start; i < end; i++ {
			// Ensure deep clone of nested pointers/slices.
			userItems[i].SkillDefinition = cloneSkill(userItems[i].SkillDefinition)
			out = append(out, userItems[i])
		}

		if end < len(userItems) {
			last := userItems[end-1]
			tok.DirTok = buildSkillCursor(last.BundleID, last.SkillSlug, last.SkillDefinition.ModifiedAt)
		} else {
			tok.DirTok = ""
		}
	}

	var nextTok *string
	// More pages exist if:
	// - still in builtin phase (BuiltInCursor set), or
	// - in user phase and DirTok set (more users), or
	// - we just switched phases but didn't yet scan that phase fully.
	if tok.Phase == spec.ListSkillPhaseBuiltIn || (tok.Phase == spec.ListSkillPhaseUser && tok.DirTok != "") {
		s := jsonutil.Base64JSONEncode(tok)
		nextTok = &s
	}

	return &spec.ListSkillsResponse{
		Body: &spec.ListSkillsResponseBody{
			SkillListItems: out,
			NextPageToken:  nextTok,
		},
	}, nil
}

func (s *SkillStore) startCleanupLoop() {
	s.cleanOnce.Do(func() {
		s.cleanKick = make(chan struct{}, 1)
		s.cleanCtx, s.cleanStop = context.WithCancel(context.Background())

		s.wg.Go(func() {
			tick := time.NewTicker(cleanupIntervalSkills)
			defer tick.Stop()

			s.sweepSoftDeleted()

			for {
				select {
				case <-s.cleanCtx.Done():
					return
				case <-tick.C:
				case <-s.cleanKick:
				}
				s.sweepSoftDeleted()
			}
		})
	})
}

func (s *SkillStore) kickCleanupLoop() {
	if s.cleanKick == nil {
		return
	}
	select {
	case s.cleanKick <- struct{}{}:
	default:
	}
}

func (s *SkillStore) sweepSoftDeleted() {
	s.sweepMu.Lock()
	defer s.sweepMu.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	all, err := s.readAllUser(false)
	if err != nil {
		slog.Error("sweepSoftDeleted/readAllUser", "err", err)
		return
	}

	now := time.Now().UTC()
	changed := false

	for bid, b := range all.Bundles {
		if b.SoftDeletedAt == nil || b.SoftDeletedAt.IsZero() {
			continue
		}
		if now.Sub(*b.SoftDeletedAt) < softDeleteGraceSkills {
			continue
		}

		// Only hard-delete if still empty.
		if len(all.Skills[bid]) > 0 {
			slog.Warn("sweepSoftDeleted: bundle not empty", "bundleID", bid)
			continue
		}

		delete(all.Bundles, bid)
		delete(all.Skills, bid)
		changed = true
		slog.Info("hard-deleted skill bundle", "bundleID", bid)
	}

	if changed {
		if err := s.writeAllUser(all); err != nil {
			slog.Error("sweepSoftDeleted/writeAllUser", "err", err)
		}
	}
}

func (s *SkillStore) getAnyBundle(ctx context.Context, id bundleitemutils.BundleID) (spec.SkillBundle, bool, error) {
	if s.builtin != nil {
		if b, err := s.builtin.GetBuiltInSkillBundle(ctx, id); err == nil {
			return b, true, nil
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	all, err := s.readAllUser(false)
	if err != nil {
		return spec.SkillBundle{}, false, err
	}
	b, ok := all.Bundles[id]
	if !ok {
		return spec.SkillBundle{}, false, fmt.Errorf("%w: %s", spec.ErrSkillBundleNotFound, id)
	}
	if isSoftDeletedSkillBundle(b) {
		return b, false, fmt.Errorf("%w: %s", spec.ErrSkillBundleDeleting, id)
	}
	return b, false, nil
}

func (s *SkillStore) writeAllUser(sc skillStoreSchema) error {
	sc.SchemaVersion = spec.SkillSchemaVersion
	mp, err := jsonencdec.StructWithJSONTagsToMap(sc)
	if err != nil {
		return err
	}
	return s.userStore.SetAll(mp)
}

func (s *SkillStore) readAllUser(force bool) (skillStoreSchema, error) {
	raw, err := s.userStore.GetAll(force)
	if err != nil {
		return skillStoreSchema{}, err
	}
	var sc skillStoreSchema
	if err := jsonencdec.MapToStructWithJSONTags(raw, &sc); err != nil {
		return sc, err
	}
	if sc.SchemaVersion == "" {
		sc.SchemaVersion = spec.SkillSchemaVersion
	} else if sc.SchemaVersion != spec.SkillSchemaVersion {
		return skillStoreSchema{}, fmt.Errorf(
			"skill store schemaVersion %q != %q",
			sc.SchemaVersion,
			spec.SkillSchemaVersion,
		)
	}
	if sc.Bundles == nil {
		sc.Bundles = map[bundleitemutils.BundleID]spec.SkillBundle{}
	}
	if sc.Skills == nil {
		sc.Skills = map[bundleitemutils.BundleID]map[spec.SkillSlug]spec.Skill{}
	}
	// Validate + normalize (hardening against file corruption).
	for bid, b := range sc.Bundles {
		// Normalize fields that must be user-owned.
		b.IsBuiltIn = false
		sc.Bundles[bid] = b
		if b.ID != bid {
			return skillStoreSchema{}, fmt.Errorf("bundle key %q != bundle.id %q", bid, b.ID)
		}
		if err := validateSkillBundle(&b); err != nil {
			return skillStoreSchema{}, fmt.Errorf("invalid bundle %q: %w", bid, err)
		}
	}
	for bid, sm := range sc.Skills {
		if sm == nil {
			sc.Skills[bid] = map[spec.SkillSlug]spec.Skill{}
			continue
		}
		if _, ok := sc.Bundles[bid]; !ok {
			return skillStoreSchema{}, fmt.Errorf("skills reference missing bundle %q", bid)
		}
		for slug, sk := range sm {
			sk.IsBuiltIn = false
			sm[slug] = sk
			if sk.Slug != slug {
				return skillStoreSchema{}, fmt.Errorf("skill key %q != skill.slug %q (bundle %q)", slug, sk.Slug, bid)
			}
			if err := validateSkill(&sk); err != nil {
				return skillStoreSchema{}, fmt.Errorf("invalid skill %q/%q: %w", bid, slug, err)
			}
		}
	}
	return sc, nil
}

func isSoftDeletedSkillBundle(b spec.SkillBundle) bool {
	return b.SoftDeletedAt != nil && !b.SoftDeletedAt.IsZero()
}

type skillCursor struct {
	ModTime   time.Time
	BundleID  bundleitemutils.BundleID
	SkillSlug spec.SkillSlug
}

func buildSkillCursor(bid bundleitemutils.BundleID, slug spec.SkillSlug, t time.Time) string {
	// Stable, opaque-ish cursor encoded as a plain string inside tok.DirTok.
	// DirTok itself is wrapped in Base64 JSON token anyway.
	return fmt.Sprintf("%s|%s|%s", t.Format(time.RFC3339Nano), bid, slug)
}

func parseSkillCursor(s string) (skillCursor, error) {
	parts := strings.Split(s, "|")
	if len(parts) != 3 {
		return skillCursor{}, errors.New("bad cursor")
	}
	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return skillCursor{}, err
	}
	return skillCursor{
		ModTime:   t,
		BundleID:  bundleitemutils.BundleID(parts[1]),
		SkillSlug: spec.SkillSlug(parts[2]),
	}, nil
}

func cloneSkill(sk spec.Skill) spec.Skill {
	c := sk
	c.Tags = slices.Clone(sk.Tags)
	c.Presence = clonePresence(sk.Presence)
	return c
}

func clonePresence(p *spec.SkillPresence) *spec.SkillPresence {
	if p == nil {
		return nil
	}
	cp := *p
	cp.LastCheckedAt = cloneTimePtr(p.LastCheckedAt)
	cp.LastSeenAt = cloneTimePtr(p.LastSeenAt)
	cp.MissingSince = cloneTimePtr(p.MissingSince)
	return &cp
}

func cloneBundle(b spec.SkillBundle) spec.SkillBundle {
	c := b
	c.SoftDeletedAt = cloneTimePtr(b.SoftDeletedAt)
	return c
}

func cloneTimePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	v := *t
	return &v
}
