package store

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/builtin"
	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/overlay"
	"github.com/flexigpt/flexigpt-app/internal/skill/spec"
)

type builtInSkillBundleID bundleitemutils.BundleID

func (builtInSkillBundleID) Group() overlay.GroupID { return "bundles" }
func (k builtInSkillBundleID) ID() overlay.KeyID    { return overlay.KeyID(k) }

type builtInSkillKey string

func (builtInSkillKey) Group() overlay.GroupID { return "skills" }
func (k builtInSkillKey) ID() overlay.KeyID    { return overlay.KeyID(k) }

type BuiltInSkills struct {
	// Immutable base.
	bundles map[bundleitemutils.BundleID]spec.SkillBundle
	skills  map[bundleitemutils.BundleID]map[spec.SkillSlug]spec.Skill

	// Overlay view.
	mu          sync.RWMutex
	viewBundles map[bundleitemutils.BundleID]spec.SkillBundle
	viewSkills  map[bundleitemutils.BundleID]map[spec.SkillSlug]spec.Skill

	// IO.
	skillsFS       fs.FS
	skillsDir      string
	overlayBaseDir string
	store          *overlay.Store
	bundleFlags    *overlay.TypedGroup[builtInSkillBundleID, bool]
	skillFlags     *overlay.TypedGroup[builtInSkillKey, bool]

	rebuilder *builtin.AsyncRebuilder
}

type BuiltInSkillsOption func(*BuiltInSkills)

func WithBuiltInSkillsFS(fsys fs.FS, root string) BuiltInSkillsOption {
	return func(b *BuiltInSkills) {
		b.skillsFS = fsys
		b.skillsDir = root
	}
}

func NewBuiltInSkills(
	ctx context.Context,
	overlayBaseDir string,
	maxSnapshotAge time.Duration,
	opts ...BuiltInSkillsOption,
) (*BuiltInSkills, error) {
	if overlayBaseDir == "" {
		return nil, fmt.Errorf("%w: overlayBaseDir", spec.ErrSkillInvalidRequest)
	}
	if maxSnapshotAge <= 0 {
		maxSnapshotAge = time.Hour
	}
	if err := os.MkdirAll(overlayBaseDir, 0o755); err != nil {
		return nil, err
	}

	store, err := overlay.NewOverlayStore(
		ctx,
		filepath.Join(overlayBaseDir, spec.SkillBuiltInOverlayDBFileName),
		overlay.WithKeyType[builtInSkillBundleID](),
		overlay.WithKeyType[builtInSkillKey](),
	)
	if err != nil {
		return nil, err
	}

	bundleFlags, err := overlay.NewTypedGroup[builtInSkillBundleID, bool](ctx, store)
	if err != nil {
		return nil, err
	}
	skillFlags, err := overlay.NewTypedGroup[builtInSkillKey, bool](ctx, store)
	if err != nil {
		return nil, err
	}

	b := &BuiltInSkills{
		skillsFS:       builtin.BuiltInSkillBundlesFS,
		skillsDir:      builtin.BuiltInSkillBundlesRootDir,
		overlayBaseDir: overlayBaseDir,
		store:          store,
		bundleFlags:    bundleFlags,
		skillFlags:     skillFlags,
	}
	for _, o := range opts {
		o(b)
	}

	if err := b.loadFromFS(ctx); err != nil {
		return nil, err
	}

	b.rebuilder = builtin.NewAsyncRebuilder(
		maxSnapshotAge,
		func() error { //nolint:contextcheck // Cannot pass app context to async builder.
			b.mu.Lock()
			defer b.mu.Unlock()
			return b.rebuildSnapshot(context.Background())
		},
	)
	b.rebuilder.MarkFresh()
	return b, nil
}

func (b *BuiltInSkills) ListBuiltInSkills(ctx context.Context) (
	bundles map[bundleitemutils.BundleID]spec.SkillBundle,
	skills map[bundleitemutils.BundleID]map[spec.SkillSlug]spec.Skill,
	err error,
) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	outBundles := make(map[bundleitemutils.BundleID]spec.SkillBundle, len(b.viewBundles))
	for id, sb := range b.viewBundles {
		outBundles[id] = cloneBundle(sb)
	}
	outSkills := make(map[bundleitemutils.BundleID]map[spec.SkillSlug]spec.Skill, len(b.viewSkills))
	for bid, inner := range b.viewSkills {
		m := make(map[spec.SkillSlug]spec.Skill, len(inner))
		for slug, sk := range inner {
			m[slug] = cloneSkill(sk)
		}
		outSkills[bid] = m
	}
	return outBundles, outSkills, nil
}

func (b *BuiltInSkills) GetBuiltInSkillBundle(
	ctx context.Context,
	id bundleitemutils.BundleID,
) (spec.SkillBundle, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	sb, ok := b.viewBundles[id]
	if !ok {
		return spec.SkillBundle{}, spec.ErrSkillBundleNotFound
	}
	return cloneBundle(sb), nil
}

func (b *BuiltInSkills) GetBuiltInSkill(
	ctx context.Context,
	bundleID bundleitemutils.BundleID,
	slug spec.SkillSlug,
) (spec.Skill, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	sm, ok := b.viewSkills[bundleID]
	if !ok {
		return spec.Skill{}, spec.ErrSkillBundleNotFound
	}
	sk, ok := sm[slug]
	if !ok {
		return spec.Skill{}, spec.ErrSkillNotFound
	}
	return cloneSkill(sk), nil
}

func (b *BuiltInSkills) SetSkillBundleEnabled(
	ctx context.Context,
	id bundleitemutils.BundleID,
	enabled bool,
) (spec.SkillBundle, error) {
	if _, ok := b.bundles[id]; !ok {
		return spec.SkillBundle{}, spec.ErrSkillBundleNotFound
	}

	flag, err := b.bundleFlags.SetFlag(ctx, builtInSkillBundleID(id), enabled)
	if err != nil {
		return spec.SkillBundle{}, err
	}

	b.mu.Lock()

	sb := b.viewBundles[id]
	sb.IsEnabled = enabled
	sb.ModifiedAt = flag.ModifiedAt
	b.viewBundles[id] = sb

	b.mu.Unlock()

	b.rebuilder.Trigger()
	return cloneBundle(sb), nil
}

func (b *BuiltInSkills) SetSkillEnabled(
	ctx context.Context,
	bundleID bundleitemutils.BundleID,
	slug spec.SkillSlug,
	enabled bool,
) (spec.Skill, error) {
	// Validate existence on base.
	if _, ok := b.skills[bundleID]; !ok {
		return spec.Skill{}, spec.ErrSkillBundleNotFound
	}
	if _, ok := b.skills[bundleID][slug]; !ok {
		return spec.Skill{}, spec.ErrSkillNotFound
	}

	flag, err := b.skillFlags.SetFlag(ctx, getBuiltInSkillKey(bundleID, slug), enabled)
	if err != nil {
		return spec.Skill{}, err
	}

	b.mu.Lock()

	sk := b.viewSkills[bundleID][slug]
	sk.IsEnabled = enabled
	sk.ModifiedAt = flag.ModifiedAt
	b.viewSkills[bundleID][slug] = sk

	b.mu.Unlock()

	b.rebuilder.Trigger()
	return cloneSkill(sk), nil
}

func (b *BuiltInSkills) loadFromFS(ctx context.Context) error {
	sub, err := resolveSkillsFS(b.skillsFS, b.skillsDir)
	if err != nil {
		return err
	}

	raw, err := fs.ReadFile(sub, builtin.BuiltInSkillBundlesJSON)
	if err != nil {
		return err
	}

	var schema skillStoreSchema
	if err := json.Unmarshal(raw, &schema); err != nil {
		return err
	}
	if schema.SchemaVersion != spec.SkillSchemaVersion {
		return fmt.Errorf("schemaVersion %q not equal to %q", schema.SchemaVersion, spec.SkillSchemaVersion)
	}
	if len(schema.Bundles) == 0 {
		return fmt.Errorf("%s contains no bundles", builtin.BuiltInSkillBundlesJSON)
	}

	// Normalize + validate + mark built-in.
	bundles := make(map[bundleitemutils.BundleID]spec.SkillBundle, len(schema.Bundles))
	skills := make(map[bundleitemutils.BundleID]map[spec.SkillSlug]spec.Skill, len(schema.Skills))

	for bid, sb := range schema.Bundles {
		sb.IsBuiltIn = true
		sb.SchemaVersion = spec.SkillSchemaVersion
		if err := validateSkillBundle(&sb); err != nil {
			return fmt.Errorf("builtin bundle %s: %w", bid, err)
		}
		bundles[bid] = sb
	}

	for bid, sm := range schema.Skills {
		if _, ok := bundles[bid]; !ok {
			return fmt.Errorf("builtin skills: bundle %s not present in bundles", bid)
		}
		subm := make(map[spec.SkillSlug]spec.Skill, len(sm))
		for slug, sk := range sm {
			sk.IsBuiltIn = true
			sk.SchemaVersion = spec.SkillSchemaVersion
			// Strongly enforce that built-ins are embeddedfs.
			if sk.Type != spec.SkillTypeEmbeddedFS {
				return fmt.Errorf("builtin skill %s/%s: type must be %q", bid, slug, spec.SkillTypeEmbeddedFS)
			}
			if err := validateSkill(&sk); err != nil {
				return fmt.Errorf("builtin skill %s/%s: %w", bid, slug, err)
			}
			// Ensure JSON slug matches map key (hardening).
			if sk.Slug != slug {
				return fmt.Errorf("builtin skill %s: map key slug %q != skill.slug %q", bid, slug, sk.Slug)
			}
			subm[slug] = sk
		}
		skills[bid] = subm
	}

	b.bundles = bundles
	b.skills = skills

	b.mu.Lock()
	defer b.mu.Unlock()
	return b.rebuildSnapshot(ctx)
}

// rebuildSnapshot applies overlay flags onto the immutable base sets.
// Caller must hold b.mu (write).
func (b *BuiltInSkills) rebuildSnapshot(ctx context.Context) error {
	newBundles := make(map[bundleitemutils.BundleID]spec.SkillBundle, len(b.bundles))
	newSkills := make(map[bundleitemutils.BundleID]map[spec.SkillSlug]spec.Skill, len(b.skills))

	for bid, sb := range b.bundles {
		if flag, ok, err := b.bundleFlags.GetFlag(ctx, builtInSkillBundleID(bid)); err != nil {
			return err
		} else if ok {
			sb.IsEnabled = flag.Value
			sb.ModifiedAt = flag.ModifiedAt
		}
		newBundles[bid] = sb
	}

	for bid, sm := range b.skills {
		subm := make(map[spec.SkillSlug]spec.Skill, len(sm))
		for slug, sk := range sm {
			if flag, ok, err := b.skillFlags.GetFlag(ctx, getBuiltInSkillKey(bid, slug)); err != nil {
				return err
			} else if ok {
				sk.IsEnabled = flag.Value
				sk.ModifiedAt = flag.ModifiedAt
			}
			subm[slug] = sk
		}
		newSkills[bid] = subm
	}

	b.viewBundles = newBundles
	b.viewSkills = newSkills
	return nil
}

func resolveSkillsFS(fsys fs.FS, dir string) (fs.FS, error) {
	if dir == "" || dir == "." {
		return fsys, nil
	}

	// Validate dir exists and is a directory.
	fi, err := fs.Stat(fsys, dir)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("%q is not a directory", dir)
	}

	return fs.Sub(fsys, dir)
}

func getBuiltInSkillKey(bundleID bundleitemutils.BundleID, slug spec.SkillSlug) builtInSkillKey {
	return builtInSkillKey(fmt.Sprintf("%s::%s", bundleID, slug))
}
