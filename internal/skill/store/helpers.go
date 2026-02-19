package store

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/skill/spec"
	"github.com/ppipada/mapstore-go/jsonencdec"
)

func (s *SkillStore) startCleanupLoop() {
	s.cleanOnce.Do(func() {
		s.cleanKick = make(chan struct{}, 1)
		s.cleanCtx, s.cleanStop = context.WithCancel(context.Background())

		s.wg.Go(func() {
			tick := time.NewTicker(cleanupIntervalSkills)
			defer tick.Stop()

			// Run once at start.
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
	defer func() {
		if r := recover(); r != nil {
			slog.Error("sweepSoftDeleted: panic", "panic", r)
		}
	}()

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
