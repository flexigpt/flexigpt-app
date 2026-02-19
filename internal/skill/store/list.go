package store

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/skill/spec"
)

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
