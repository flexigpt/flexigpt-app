package store

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

	"github.com/flexigpt/agentskills-go"
	"github.com/flexigpt/agentskills-go/fsskillprovider"
	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/ppipada/mapstore-go"
	"github.com/ppipada/mapstore-go/jsonencdec"
	"github.com/ppipada/mapstore-go/uuidv7filename"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/skill/spec"
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
	SchemaVersion string `json:"schemaVersion"`

	Bundles map[bundleitemutils.BundleID]spec.SkillBundle              `json:"bundles"`
	Skills  map[bundleitemutils.BundleID]map[spec.SkillSlug]spec.Skill `json:"skills"`
}

// SkillStore provides CRUD and listing for Skill bundles and Skills (single JSON file).
type SkillStore struct {
	baseDir            string
	embeddedHydrateDir string

	userStore *mapstore.MapFileStore
	builtin   *BuiltInSkills
	runtime   *agentskills.Runtime

	mu sync.RWMutex // guards userStore read-modify-write

	// Cleanup loop plumbing.
	cleanOnce sync.Once
	cleanKick chan struct{}
	cleanCtx  context.Context
	cleanStop context.CancelFunc
	wg        sync.WaitGroup

	// Sweep coordination with CRUD ops.
	sweepMu sync.RWMutex

	// Serializes runtime resync calls (best-effort reconcile).
	rtResyncMu sync.Mutex
}

type skillStoreOptions struct {
	runtime *agentskills.Runtime

	// Where embeddedfs content is hydrated to disk (so runtime can treat it like fs skills).
	embeddedHydrateDir string
}

type SkillStoreOption func(*skillStoreOptions) error

func WithRuntime(rt *agentskills.Runtime) SkillStoreOption {
	return func(o *skillStoreOptions) error {
		if rt == nil {
			return fmt.Errorf("%w: nil runtime", spec.ErrSkillInvalidRequest)
		}
		o.runtime = rt
		return nil
	}
}

func WithEmbeddedHydrateDir(dir string) SkillStoreOption {
	return func(o *skillStoreOptions) error {
		o.embeddedHydrateDir = strings.TrimSpace(dir)
		return nil
	}
}

func newDefaultRuntime() (*agentskills.Runtime, error) {
	p, err := fsskillprovider.New()
	if err != nil {
		return nil, err
	}
	return agentskills.New(
		agentskills.WithProvider(p),
		agentskills.WithLogger(slog.Default()),
	)
}

func NewSkillStore(baseDir string, opts ...SkillStoreOption) (*SkillStore, error) {
	if strings.TrimSpace(baseDir) == "" {
		return nil, fmt.Errorf("%w: baseDir is empty", spec.ErrSkillInvalidRequest)
	}

	cfg := skillStoreOptions{}
	for _, o := range opts {
		if o == nil {
			continue
		}
		if err := o(&cfg); err != nil {
			return nil, err
		}
	}

	// Runtime is REQUIRED. If caller didn't provide one, create the default (fs provider).
	if cfg.runtime == nil {
		rt, err := newDefaultRuntime()
		if err != nil {
			return nil, err
		}
		cfg.runtime = rt
	}

	s := &SkillStore{
		baseDir: filepath.Clean(baseDir),
		runtime: cfg.runtime,
	}

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

	s.embeddedHydrateDir = cfg.embeddedHydrateDir
	if s.embeddedHydrateDir == "" {
		s.embeddedHydrateDir = filepath.Join(s.baseDir, ".skills-embeddedfs-hydrated")
	}
	s.embeddedHydrateDir = filepath.Clean(s.embeddedHydrateDir)

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

	// Runtime integration:
	// - hydrate embeddedfs → disk
	// - reconcile runtime catalog from enabled bundles/skills.
	if err := s.hydrateBuiltInEmbeddedFS(ctx); err != nil {
		slog.Error("runtime init: embeddedfs hydration failed", "err", err)
	}
	s.bestEffortRuntimeResync(ctx, "init")

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
	s.mu.Lock()
	resyncRuntime := false
	defer func() {
		s.mu.Unlock()
		s.sweepMu.Unlock()
		if resyncRuntime {
			s.bestEffortRuntimeResync(ctx, "putSkillBundle")
		}
	}()

	all, err := s.readAllUser(false)
	if err != nil {
		return nil, err
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

	resyncRuntime = true
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
			s.bestEffortRuntimeResync(ctx, "patchSkillBundle(builtin)")
			slog.Info("patchSkillBundle (builtin)", "bundleID", req.BundleID, "enabled", req.Body.IsEnabled)
			return &spec.PatchSkillBundleResponse{}, nil
		}
	}

	s.sweepMu.Lock()
	s.mu.Lock()
	resyncRuntime := false
	defer func() {
		s.mu.Unlock()
		s.sweepMu.Unlock()
		if resyncRuntime {
			s.bestEffortRuntimeResync(ctx, "patchSkillBundle")
		}
	}()

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

	resyncRuntime = true
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
	if sm := all.Skills[req.BundleID]; len(sm) > 0 {
		return nil, fmt.Errorf("%w: %s", spec.ErrSkillBundleNotEmpty, req.BundleID)
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
		return nil, fmt.Errorf("%w: only skillType=%q can be created", spec.ErrSkillInvalidRequest, spec.SkillTypeFS)
	}

	// Built-in bundle IDs are read-only.
	if s.builtin != nil {
		if _, err := s.builtin.GetBuiltInSkillBundle(ctx, req.BundleID); err == nil {
			return nil, fmt.Errorf("%w: bundleID %q", spec.ErrSkillBuiltInReadOnly, req.BundleID)
		}
	}

	// Strict foreground saga under the store locks:
	// runtime check → store write → rollback runtime on store failure.
	s.sweepMu.Lock()
	s.mu.Lock()

	var (
		postRollbackReason string
		postRollbackErr    error
		postResyncReason   string
	)
	defer func() {
		s.mu.Unlock()
		s.sweepMu.Unlock()

		if postRollbackErr != nil {
			s.runtimeRollbackToStoreStrict(postRollbackReason, postRollbackErr)
			return
		}
		if postResyncReason != "" {
			s.bestEffortRuntimeResync(ctx, postResyncReason)
		}
	}()

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
	if all.Skills[req.BundleID] == nil {
		all.Skills[req.BundleID] = map[spec.SkillSlug]spec.Skill{}
	}
	if _, exists := all.Skills[req.BundleID][req.SkillSlug]; exists {
		return nil, fmt.Errorf("%w: duplicate skillSlug in bundle", spec.ErrSkillConflict)
	}

	uuid, err := uuidv7filename.NewUUIDv7String()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()

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

	def, err := runtimeDefForUserSkill(sk)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", spec.ErrSkillInvalidRequest, err)
	}

	// Build desired-count view for "disabled validation removal" correctness.
	// If this def is already desired by other enabled skills, we must NOT remove it after validation.
	desiredCounts, err := s.runtimeDesiredDefCountsForSnapshot(ctx, all)
	if err != nil {
		return nil, err
	}
	wasDesiredAlready := desiredCounts[def] > 0

	// 1) Runtime commit first (strict):
	//    - enabled: ensure present (AddSkill)
	//    - disabled: validate by AddSkill, then remove if we added it AND it wasn't already desired.
	{
		addedByUs, rtErr := s.runtimeTryAddForeground(ctx, def)
		if rtErr != nil {
			// No store change yet.
			return nil, fmt.Errorf("%w: runtime rejected skill: %w", spec.ErrSkillInvalidRequest, rtErr)
		}
		if !sk.IsEnabled && addedByUs && !wasDesiredAlready {
			if rtErr := s.runtimeRemoveForegroundStrict(ctx, def); rtErr != nil {
				postRollbackReason = "putSkill(runtime-cleanup-failed)"
				postRollbackErr = rtErr
				return nil, fmt.Errorf("%w: runtime cleanup failed: %w", spec.ErrSkillInvalidRequest, rtErr)
			}
		}
	}

	// 2) Store commit (strict).
	all.Skills[req.BundleID][req.SkillSlug] = sk
	if err := s.writeAllUser(all); err != nil {
		postRollbackReason = "putSkill(store-failed)"
		postRollbackErr = err
		return nil, err
	}

	slog.Info("putSkill", "bundleID", req.BundleID, "skillSlug", req.SkillSlug)

	postResyncReason = "putSkill"

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

	// Built-in: enable/disable only.
	if s.builtin != nil {
		if _, err := s.builtin.GetBuiltInSkillBundle(ctx, req.BundleID); err == nil {
			if req.Body.Location != nil {
				return nil, fmt.Errorf("%w: cannot modify location for built-in", spec.ErrSkillBuiltInReadOnly)
			}
			if req.Body.IsEnabled == nil {
				return nil, fmt.Errorf("%w: isEnabled required for built-in patch", spec.ErrSkillInvalidRequest)
			}

			enabled := *req.Body.IsEnabled

			// Need skill record to build runtime def.
			sk, err := s.builtin.GetBuiltInSkill(ctx, req.BundleID, req.SkillSlug)
			if err != nil {
				return nil, err
			}

			// If enabling, ensure hydration + runtime can index before persisting overlay.
			if enabled {
				if err := s.hydrateBuiltInEmbeddedFS(ctx); err != nil {
					return nil, fmt.Errorf("%w: hydration failed: %w", spec.ErrSkillInvalidRequest, err)
				}
				def, err := s.runtimeDefForBuiltInSkill(sk)
				if err != nil {
					return nil, fmt.Errorf("%w: %w", spec.ErrSkillInvalidRequest, err)
				}
				if _, err := s.runtimeTryAddForeground(ctx, def); err != nil {
					// strict: no overlay write
					return nil, fmt.Errorf("%w: runtime rejected skill: %w", spec.ErrSkillInvalidRequest, err)
				}
			} else {
				// Disabling: strict runtime remove first (best effort ignore notfound).
				def, err := s.runtimeDefForBuiltInSkill(sk)
				if err != nil {
					return nil, fmt.Errorf("%w: %w", spec.ErrSkillInvalidRequest, err)
				}
				if err := s.runtimeRemoveForegroundStrict(ctx, def); err != nil {
					return nil, fmt.Errorf("%w: runtime remove failed: %w", spec.ErrSkillInvalidRequest, err)
				}
			}

			if _, err := s.builtin.SetSkillEnabled(ctx, req.BundleID, req.SkillSlug, enabled); err != nil {
				// Overlay write failed; rollback runtime to store (builtins included) and return error.
				s.runtimeRollbackToStoreStrict("patchSkill(builtin store-failed)", err)
				return nil, err
			}

			s.bestEffortRuntimeResync(ctx, "patchSkill(builtin)")
			slog.Info("patchSkill (builtin)", "bundleID", req.BundleID, "skillSlug", req.SkillSlug, "enabled", enabled)
			return &spec.PatchSkillResponse{}, nil
		}
	}

	// User path (runtime-first, strict store commit), under locks (no revision CAS).
	s.sweepMu.Lock()
	s.mu.Lock()

	var (
		postRollbackReason string
		postRollbackErr    error
		postResyncReason   string
	)
	defer func() {
		s.mu.Unlock()
		s.sweepMu.Unlock()

		if postRollbackErr != nil {
			s.runtimeRollbackToStoreStrict(postRollbackReason, postRollbackErr)
			return
		}
		if postResyncReason != "" {
			s.bestEffortRuntimeResync(ctx, postResyncReason)
		}
	}()

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
	cur, ok := sm[req.SkillSlug]
	if !ok {
		return nil, fmt.Errorf("%w: %s", spec.ErrSkillNotFound, req.SkillSlug)
	}

	// Build target.
	target := cur
	if req.Body.IsEnabled != nil {
		target.IsEnabled = *req.Body.IsEnabled
	}
	if req.Body.Location != nil && strings.TrimSpace(*req.Body.Location) == "" {
		return nil, fmt.Errorf("%w: location cannot be empty", spec.ErrSkillInvalidRequest)
	}
	locationChanged := false
	if req.Body.Location != nil && *req.Body.Location != target.Location {
		target.Location = *req.Body.Location
		target.Presence = &spec.SkillPresence{Status: spec.SkillPresenceUnknown}
		locationChanged = true
	}
	now := time.Now().UTC()
	target.ModifiedAt = now

	if err := validateSkill(&target); err != nil {
		return nil, err
	}

	oldDef, err := runtimeDefForUserSkill(cur)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", spec.ErrSkillInvalidRequest, err)
	}
	newDef, err := runtimeDefForUserSkill(target)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", spec.ErrSkillInvalidRequest, err)
	}

	// Desired counts for duplicate-safe removal decisions.
	desiredCounts, err := s.runtimeDesiredDefCountsForSnapshot(ctx, all)
	if err != nil {
		return nil, err
	}

	// Apply delta to counts (only affects this skill).
	applyDelta := func(m map[agentskillsSpec.SkillDef]int, def agentskillsSpec.SkillDef, delta int) {
		if delta == 0 {
			return
		}
		m[def] += delta
		if m[def] <= 0 {
			delete(m, def)
		}
	}

	// In old snapshot, if cur.IsEnabled then it contributed +1 to oldDef.
	// In new state, if target.IsEnabled then it contributes +1 to newDef.
	afterCounts := make(map[agentskillsSpec.SkillDef]int, 2)
	afterCounts[oldDef] = desiredCounts[oldDef]
	afterCounts[newDef] = desiredCounts[newDef]
	if cur.IsEnabled {
		applyDelta(afterCounts, oldDef, -1)
	}
	if target.IsEnabled {
		applyDelta(afterCounts, newDef, +1)
	}

	afterOld := afterCounts[oldDef]
	afterNew := afterCounts[newDef]

	// 1) Runtime commit first (strict):
	//    - If enabling OR location change: ensure newDef indexable (AddSkill).
	//    - If final disabled and we added a brand-new newDef that isn't desired otherwise, remove it.
	//    - Remove oldDef only if it becomes undesired (afterOld == 0).
	needValidateNew := locationChanged || (target.IsEnabled && !cur.IsEnabled)

	if needValidateNew {
		addedByUs, rtErr := s.runtimeTryAddForeground(ctx, newDef)
		if rtErr != nil {
			return nil, fmt.Errorf("%w: runtime rejected skill: %w", spec.ErrSkillInvalidRequest, rtErr)
		}
		if !target.IsEnabled && addedByUs && afterNew == 0 {
			if rtErr := s.runtimeRemoveForegroundStrict(ctx, newDef); rtErr != nil {
				postRollbackReason = "patchSkill(runtime-cleanup-failed)"
				postRollbackErr = rtErr
				return nil, fmt.Errorf("%w: runtime cleanup failed: %w", spec.ErrSkillInvalidRequest, rtErr)
			}
		}
	}

	// Remove oldDef only if this patch makes it undesired.
	if cur.IsEnabled && afterOld == 0 {
		if rtErr := s.runtimeRemoveForegroundStrict(ctx, oldDef); rtErr != nil {
			// We may have already added newDef; rollback to store.
			postRollbackReason = "patchSkill(runtime-remove-old-failed)"
			postRollbackErr = rtErr
			return nil, fmt.Errorf("%w: runtime remove failed: %w", spec.ErrSkillInvalidRequest, rtErr)
		}
	}

	// 2) Store commit (strict).
	sm[req.SkillSlug] = target
	all.Skills[req.BundleID] = sm
	if err := s.writeAllUser(all); err != nil {
		postRollbackReason = "patchSkill(store-failed)"
		postRollbackErr = err
		return nil, err
	}

	slog.Info("patchSkill", "bundleID", req.BundleID, "skillSlug", req.SkillSlug, "enabled", req.Body.IsEnabled)
	postResyncReason = "patchSkill"

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

	// Strict foreground saga under locks (no revision CAS).
	s.sweepMu.Lock()
	s.mu.Lock()

	var (
		postRollbackReason string
		postRollbackErr    error
		postResyncReason   string
	)
	defer func() {
		s.mu.Unlock()
		s.sweepMu.Unlock()

		if postRollbackErr != nil {
			s.runtimeRollbackToStoreStrict(postRollbackReason, postRollbackErr)
			return
		}
		if postResyncReason != "" {
			s.bestEffortRuntimeResync(ctx, postResyncReason)
		}
	}()

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

	def, err := runtimeDefForUserSkill(sk)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", spec.ErrSkillInvalidRequest, err)
	}

	// Desired counts for duplicate-safe removal decisions.
	desiredCounts, err := s.runtimeDesiredDefCountsForSnapshot(ctx, all)
	if err != nil {
		return nil, err
	}

	// If this skill is enabled AND bundle enabled, it currently contributes +1.
	// After delete it contributes 0, so count decreases by 1.
	after := desiredCounts[def]
	if b.IsEnabled && sk.IsEnabled {
		after--
	}

	// 1) Runtime commit first (strict): remove only if it becomes undesired.
	if b.IsEnabled && sk.IsEnabled && after <= 0 {
		if rtErr := s.runtimeRemoveForegroundStrict(ctx, def); rtErr != nil {
			return nil, fmt.Errorf("%w: runtime remove failed: %w", spec.ErrSkillInvalidRequest, rtErr)
		}
	}

	// 2) Store commit (strict).
	delete(sm, req.SkillSlug)
	all.Skills[req.BundleID] = sm
	if err := s.writeAllUser(all); err != nil {
		postRollbackReason = "deleteSkill(store-failed)"
		postRollbackErr = err
		return nil, err
	}

	slog.Info("deleteSkill", "bundleID", req.BundleID, "skillSlug", req.SkillSlug)
	postResyncReason = "deleteSkill"
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

// runtimeDesiredDefCountsForSnapshot builds desired counts for enabled skills across builtin + user snapshot.
// Used for duplicate-safe runtime removals in foreground paths.
func (s *SkillStore) runtimeDesiredDefCountsForSnapshot(
	ctx context.Context,
	user skillStoreSchema,
) (map[agentskillsSpec.SkillDef]int, error) {
	out := map[agentskillsSpec.SkillDef]int{}

	// Built-ins.
	if s.builtin != nil {
		bundles, skills, err := s.builtin.ListBuiltInSkills(ctx)
		if err != nil {
			return nil, err
		}
		for bid, b := range bundles {
			if !b.IsEnabled {
				continue
			}
			for _, sk := range skills[bid] {
				if !sk.IsEnabled {
					continue
				}
				def, err := s.runtimeDefForBuiltInSkill(sk)
				if err != nil {
					continue
				}
				out[def]++
			}
		}
	}

	// Users.
	for bid, b := range user.Bundles {
		if isSoftDeletedSkillBundle(b) || !b.IsEnabled {
			continue
		}
		sm := user.Skills[bid]
		for _, sk := range sm {
			if !sk.IsEnabled {
				continue
			}
			def, err := runtimeDefForUserSkill(sk)
			if err != nil {
				continue
			}
			out[def]++
		}
	}

	return out, nil
}
