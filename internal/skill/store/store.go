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

	// "writeMu" serializes ALL write operations (CRUD + sweeper). This lets us run potentially-slow runtime
	// validation/mutations without holding mu.
	writeMu sync.Mutex

	// "mu" guards userStore I/O and in-memory schema normalization during read/write.
	mu sync.RWMutex

	// Cleanup loop plumbing.
	cleanOnce sync.Once
	cleanKick chan struct{}
	cleanCtx  context.Context
	cleanStop context.CancelFunc
	wg        sync.WaitGroup

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

	if err := s.withUserWriteSaga(ctx, "putSkillBundle", func(sc *skillStoreSchema) (userWriteSagaOutcome, error) {
		now := time.Now().UTC()

		var (
			oldEnabled bool
			createdAt  = now
		)
		if ex, ok := sc.Bundles[req.BundleID]; ok {
			if isSoftDeletedSkillBundle(ex) {
				return userWriteSagaOutcome{}, fmt.Errorf("%w: %s", spec.ErrSkillBundleDeleting, req.BundleID)
			}
			oldEnabled = ex.IsEnabled
			if !ex.CreatedAt.IsZero() {
				createdAt = ex.CreatedAt
			}
		}

		newEnabled := req.Body.IsEnabled
		enabledChanged := oldEnabled != newEnabled

		// If enabled state changes, we must strictly apply the runtime delta BEFORE committing store.
		//
		// Note: this does not "add skills to the bundle"; it only ensures runtime reflects the bundle gate.
		if enabledChanged {
			if err := s.runtimeApplyUserBundleEnabledDelta(ctx, sc, req.BundleID, oldEnabled, newEnabled); err != nil {
				return userWriteSagaOutcome{RollbackReason: "putSkillBundle(runtime-enabled-delta-failed)"},
					fmt.Errorf("%w: %w", spec.ErrSkillInvalidRequest, err)
			}
		}

		// Commit bundle record (store).
		b := spec.SkillBundle{
			SchemaVersion: spec.SkillSchemaVersion,
			ID:            req.BundleID,
			Slug:          req.Body.Slug,
			DisplayName:   req.Body.DisplayName,
			Description:   req.Body.Description,
			IsEnabled:     newEnabled,
			IsBuiltIn:     false,
			CreatedAt:     createdAt,
			ModifiedAt:    now,
			SoftDeletedAt: nil,
		}
		if err := validateSkillBundle(&b); err != nil {
			return userWriteSagaOutcome{}, err
		}

		sc.Bundles[req.BundleID] = b
		if _, ok := sc.Skills[req.BundleID]; !ok {
			sc.Skills[req.BundleID] = map[spec.SkillSlug]spec.Skill{}
		}

		// Runtime resync is only needed if enabled state changed.
		// (Metadata-only updates don't affect runtime desired set.)
		if enabledChanged {
			return userWriteSagaOutcome{ResyncReason: "putSkillBundle"}, nil
		}
		return userWriteSagaOutcome{}, nil
	}); err != nil {
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
			s.bestEffortRuntimeResync(ctx, "patchSkillBundle(builtin)")
			slog.Info("patchSkillBundle (builtin)", "bundleID", req.BundleID, "enabled", req.Body.IsEnabled)
			return &spec.PatchSkillBundleResponse{}, nil
		}
	}
	if err := s.withUserWriteSaga(ctx, "patchSkillBundle", func(sc *skillStoreSchema) (userWriteSagaOutcome, error) {
		b, ok := sc.Bundles[req.BundleID]
		if !ok {
			return userWriteSagaOutcome{}, fmt.Errorf("%w: %s", spec.ErrSkillBundleNotFound, req.BundleID)
		}
		if isSoftDeletedSkillBundle(b) {
			return userWriteSagaOutcome{}, fmt.Errorf("%w: %s", spec.ErrSkillBundleDeleting, req.BundleID)
		}

		oldEnabled := b.IsEnabled
		newEnabled := req.Body.IsEnabled
		enabledChanged := oldEnabled != newEnabled

		if enabledChanged {
			if err := s.runtimeApplyUserBundleEnabledDelta(ctx, sc, req.BundleID, oldEnabled, newEnabled); err != nil {
				return userWriteSagaOutcome{RollbackReason: "patchSkillBundle(runtime-enabled-delta-failed)"},
					fmt.Errorf("%w: %w", spec.ErrSkillInvalidRequest, err)
			}
		}

		b.IsEnabled = newEnabled
		b.ModifiedAt = time.Now().UTC()
		sc.Bundles[req.BundleID] = b

		if enabledChanged {
			return userWriteSagaOutcome{ResyncReason: "patchSkillBundle"}, nil
		}
		return userWriteSagaOutcome{}, nil
	}); err != nil {
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

	if err := s.withUserWriteSaga(ctx, "deleteSkillBundle", func(sc *skillStoreSchema) (userWriteSagaOutcome, error) {
		b, ok := sc.Bundles[req.BundleID]
		if !ok {
			return userWriteSagaOutcome{}, fmt.Errorf("%w: %s", spec.ErrSkillBundleNotFound, req.BundleID)
		}
		if isSoftDeletedSkillBundle(b) {
			return userWriteSagaOutcome{}, fmt.Errorf("%w: %s", spec.ErrSkillBundleDeleting, req.BundleID)
		}

		// Must be empty.
		if sm := sc.Skills[req.BundleID]; len(sm) > 0 {
			return userWriteSagaOutcome{}, fmt.Errorf("%w: %s", spec.ErrSkillBundleNotEmpty, req.BundleID)
		}

		now := time.Now().UTC()
		b.IsEnabled = false
		b.SoftDeletedAt = &now
		b.ModifiedAt = now
		sc.Bundles[req.BundleID] = b

		// No runtime resync needed: bundle is empty.
		return userWriteSagaOutcome{}, nil
	}); err != nil {
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

	var created spec.Skill
	if err := s.withUserWriteSaga(ctx, "putSkill", func(sc *skillStoreSchema) (userWriteSagaOutcome, error) {
		b, ok := sc.Bundles[req.BundleID]
		if !ok {
			return userWriteSagaOutcome{}, fmt.Errorf("%w: %s", spec.ErrSkillBundleNotFound, req.BundleID)
		}
		if isSoftDeletedSkillBundle(b) {
			return userWriteSagaOutcome{}, fmt.Errorf("%w: %s", spec.ErrSkillBundleDeleting, req.BundleID)
		}
		if !b.IsEnabled {
			return userWriteSagaOutcome{}, fmt.Errorf("%w: %s", spec.ErrSkillBundleDisabled, req.BundleID)
		}
		if sc.Skills[req.BundleID] == nil {
			sc.Skills[req.BundleID] = map[spec.SkillSlug]spec.Skill{}
		}
		if _, exists := sc.Skills[req.BundleID][req.SkillSlug]; exists {
			return userWriteSagaOutcome{}, fmt.Errorf("%w: duplicate skillSlug in bundle", spec.ErrSkillConflict)
		}

		uuid, err := uuidv7filename.NewUUIDv7String()
		if err != nil {
			return userWriteSagaOutcome{}, err
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
			return userWriteSagaOutcome{}, err
		}

		// Runtime check first (strict). For disabled skills we validate via AddSkill and let resync converge later.
		def, err := runtimeDefForUserSkill(sk)
		if err != nil {
			return userWriteSagaOutcome{}, fmt.Errorf("%w: %w", spec.ErrSkillInvalidRequest, err)
		}
		if _, rtErr := s.runtimeTryAddForeground(ctx, def); rtErr != nil {
			return userWriteSagaOutcome{}, fmt.Errorf(
				"%w: runtime rejected skill: %w",
				spec.ErrSkillInvalidRequest,
				rtErr,
			)
		}

		// Store commit (strict via saga helper).
		sc.Skills[req.BundleID][req.SkillSlug] = sk
		created = sk
		return userWriteSagaOutcome{ResyncReason: "putSkill"}, nil
	}); err != nil {
		return nil, err
	}

	slog.Info("putSkill", "bundleID", req.BundleID, "skillSlug", created.Slug)
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

	var finalEnabled *bool
	if req.Body.IsEnabled != nil {
		v := *req.Body.IsEnabled
		finalEnabled = &v
	}

	if err := s.withUserWriteSaga(ctx, "patchSkill", func(sc *skillStoreSchema) (userWriteSagaOutcome, error) {
		b, ok := sc.Bundles[req.BundleID]
		if !ok {
			return userWriteSagaOutcome{}, fmt.Errorf("%w: %s", spec.ErrSkillBundleNotFound, req.BundleID)
		}
		if isSoftDeletedSkillBundle(b) {
			return userWriteSagaOutcome{}, fmt.Errorf("%w: %s", spec.ErrSkillBundleDeleting, req.BundleID)
		}
		if !b.IsEnabled {
			return userWriteSagaOutcome{}, fmt.Errorf("%w: %s", spec.ErrSkillBundleDisabled, req.BundleID)
		}

		sm := sc.Skills[req.BundleID]
		if sm == nil {
			return userWriteSagaOutcome{}, fmt.Errorf("%w: %s", spec.ErrSkillNotFound, req.SkillSlug)
		}
		cur, ok := sm[req.SkillSlug]
		if !ok {
			return userWriteSagaOutcome{}, fmt.Errorf("%w: %s", spec.ErrSkillNotFound, req.SkillSlug)
		}

		// Build target.
		target := cur
		if req.Body.IsEnabled != nil {
			target.IsEnabled = *req.Body.IsEnabled
		}
		if req.Body.Location != nil && strings.TrimSpace(*req.Body.Location) == "" {
			return userWriteSagaOutcome{}, fmt.Errorf("%w: location cannot be empty", spec.ErrSkillInvalidRequest)
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
			return userWriteSagaOutcome{}, err
		}

		oldDef, err := runtimeDefForUserSkill(cur)
		if err != nil {
			return userWriteSagaOutcome{}, fmt.Errorf("%w: %w", spec.ErrSkillInvalidRequest, err)
		}
		newDef, err := runtimeDefForUserSkill(target)
		if err != nil {
			return userWriteSagaOutcome{}, fmt.Errorf("%w: %w", spec.ErrSkillInvalidRequest, err)
		}

		// 1) Runtime check/mutation first (strict):
		//    - If enabling OR location change: ensure newDef is indexable (AddSkill).
		//    - No immediate "disabled cleanup"; resync after commit converges runtime to enabled set.
		needValidateNew := locationChanged || (target.IsEnabled && !cur.IsEnabled)
		if needValidateNew {
			if _, rtErr := s.runtimeTryAddForeground(ctx, newDef); rtErr != nil {
				return userWriteSagaOutcome{}, fmt.Errorf(
					"%w: runtime rejected skill: %w",
					spec.ErrSkillInvalidRequest,
					rtErr,
				)
			}
		}

		// Remove oldDef only if this patch makes it undesired (duplicate-safe).
		desiredCounts, err := s.runtimeDesiredDefCountsForSnapshot(ctx, *sc)
		if err != nil {
			return userWriteSagaOutcome{}, err
		}
		afterOld := desiredCounts[oldDef]
		if cur.IsEnabled {
			afterOld--
		}
		if target.IsEnabled && oldDef == newDef {
			afterOld++
		}
		if cur.IsEnabled && afterOld <= 0 {
			if rtErr := s.runtimeRemoveForegroundStrict(ctx, oldDef); rtErr != nil {
				// Runtime may already contain newDef; rollback runtime back to store.
				return userWriteSagaOutcome{
					RollbackReason: "patchSkill(runtime-remove-old-failed)",
				}, fmt.Errorf("%w: runtime remove failed: %w", spec.ErrSkillInvalidRequest, rtErr)
			}
		}

		// 2) Store commit (strict via saga helper).
		sm[req.SkillSlug] = target
		sc.Skills[req.BundleID] = sm
		return userWriteSagaOutcome{ResyncReason: "patchSkill"}, nil
	}); err != nil {
		return nil, err
	}

	slog.Info("patchSkill", "bundleID", req.BundleID, "skillSlug", req.SkillSlug, "enabled", finalEnabled)
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

	if err := s.withUserWriteSaga(ctx, "deleteSkill", func(sc *skillStoreSchema) (userWriteSagaOutcome, error) {
		b, ok := sc.Bundles[req.BundleID]
		if !ok {
			return userWriteSagaOutcome{}, fmt.Errorf("%w: %s", spec.ErrSkillBundleNotFound, req.BundleID)
		}
		if isSoftDeletedSkillBundle(b) {
			return userWriteSagaOutcome{}, fmt.Errorf("%w: %s", spec.ErrSkillBundleDeleting, req.BundleID)
		}

		sm := sc.Skills[req.BundleID]
		if sm == nil {
			return userWriteSagaOutcome{}, fmt.Errorf("%w: %s", spec.ErrSkillNotFound, req.SkillSlug)
		}
		sk, ok := sm[req.SkillSlug]
		if !ok {
			return userWriteSagaOutcome{}, fmt.Errorf("%w: %s", spec.ErrSkillNotFound, req.SkillSlug)
		}
		if sk.Presence != nil && sk.Presence.Status == spec.SkillPresenceMissing {
			return userWriteSagaOutcome{}, fmt.Errorf("%w: %s", spec.ErrSkillIsMissing, req.SkillSlug)
		}

		def, err := runtimeDefForUserSkill(sk)
		if err != nil {
			return userWriteSagaOutcome{}, fmt.Errorf("%w: %w", spec.ErrSkillInvalidRequest, err)
		}

		// Duplicate-safe removal decision.
		desiredCounts, err := s.runtimeDesiredDefCountsForSnapshot(ctx, *sc)
		if err != nil {
			return userWriteSagaOutcome{}, err
		}
		after := desiredCounts[def]
		if b.IsEnabled && sk.IsEnabled {
			after--
		}

		// 1) Runtime commit first (strict): remove only if it becomes undesired.
		if b.IsEnabled && sk.IsEnabled && after <= 0 {
			if rtErr := s.runtimeRemoveForegroundStrict(ctx, def); rtErr != nil {
				return userWriteSagaOutcome{}, fmt.Errorf(
					"%w: runtime remove failed: %w",
					spec.ErrSkillInvalidRequest,
					rtErr,
				)
			}
		}

		// 2) Store commit (strict via saga helper).
		delete(sm, req.SkillSlug)
		sc.Skills[req.BundleID] = sm
		return userWriteSagaOutcome{ResyncReason: "deleteSkill"}, nil
	}); err != nil {
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
