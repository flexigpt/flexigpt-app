package store

import (
	"context"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/flexigpt/agentskills-go"
	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/skill/spec"
)

const (
	embeddedHydrateDigestFile = ".embeddedfs.sha256"

	// Best-effort cap for runtime resync work done inline after store mutations.
	runtimeResyncTimeout = 30 * time.Second

	// Foreground validation should be quick; runtime is in-mem, but provider indexing may touch disk.
	runtimeForegroundValidateTimeout = 15 * time.Second
)

type runtimeApplyMode int

const (
	runtimeApplyBestEffort runtimeApplyMode = iota
	runtimeApplyStrict
)

func (s *SkillStore) CreateSkillSession(
	ctx context.Context,
	req *spec.CreateSkillSessionRequest,
) (*spec.CreateSkillSessionResponse, error) {
	if s.runtime == nil {
		return nil, fmt.Errorf("%w: runtime not configured", spec.ErrSkillInvalidRequest)
	}
	if req == nil || req.Body == nil {
		return nil, fmt.Errorf("%w: missing request", spec.ErrSkillInvalidRequest)
	}

	if len(req.Body.AllowSkillRefs) == 0 {
		return nil, fmt.Errorf("%w: allowSkillRefs required", spec.ErrSkillInvalidRequest)
	}

	for _, r := range req.Body.AllowSkillRefs {
		if err := validateSkillRef(r); err != nil {
			return nil, fmt.Errorf("%w: invalid allowSkillRef: %w", spec.ErrSkillInvalidRequest, err)
		}
	}
	for _, r := range req.Body.ActiveSkillRefs {
		if err := validateSkillRef(r); err != nil {
			return nil, fmt.Errorf("%w: invalid activeSkillRef: %w", spec.ErrSkillInvalidRequest, err)
		}
	}

	// Best-effort close an old session if requested (avoids session leaks on tab/conversation switches).
	if sid := strings.TrimSpace(string(req.Body.CloseSessionID)); sid != "" {
		_ = s.runtime.CloseSession(ctx, agentskillsSpec.SessionID(sid))
	}

	activeRefs := normalizeActiveRefsSubsetOfAllow(req.Body.AllowSkillRefs, req.Body.ActiveSkillRefs)

	// Resolve allowlist refs -> defs once; then resolve active refs from the allowlist mapping.
	res := s.resolveAllowSkillRefs(ctx, req.Body.AllowSkillRefs)
	if len(res.AllowDefs) == 0 {
		return nil, fmt.Errorf("%w: allowSkillRefs did not resolve to any enabled skills", spec.ErrSkillInvalidRequest)
	}

	activeDefSet := map[agentskillsSpec.SkillDef]struct{}{}
	for _, r := range activeRefs {
		if def, ok := res.RefToDef[refKey(r)]; ok {
			activeDefSet[def] = struct{}{}
		}
	}
	activeDefs := make([]agentskillsSpec.SkillDef, 0, len(activeDefSet))
	for d := range activeDefSet {
		activeDefs = append(activeDefs, d)
	}
	sortSkillDefs(activeDefs)

	opts := []agentskills.SessionOption{}
	if req.Body.MaxActivePerSession > 0 {
		opts = append(opts, agentskills.WithSessionMaxActivePerSession(req.Body.MaxActivePerSession))
	}
	if len(activeDefs) > 0 {
		opts = append(opts, agentskills.WithSessionActiveSkills(activeDefs))
	}

	sid, _, err := s.runtime.NewSession(ctx, opts...)
	if err != nil {
		return nil, err
	}

	// Query actual active skills from runtime (don’t trust input echo).
	recs, err := s.runtime.ListSkills(ctx, &agentskills.SkillListFilter{
		SessionID:   sid,
		Activity:    agentskills.SkillActivityActive,
		AllowSkills: res.AllowDefs, // constrain to allowlist to keep mapping well-defined
	})
	if err != nil {
		// If session somehow disappeared immediately, surface it.
		if errors.Is(err, agentskillsSpec.ErrSessionNotFound) {
			return nil, err
		}
		// Otherwise: return session but empty actives.
		recs = nil
	}

	actualActiveDefSet := map[agentskillsSpec.SkillDef]struct{}{}
	for _, r := range recs {
		actualActiveDefSet[r.Def] = struct{}{}
	}
	activeOutRefs := buildActiveSkillRefs(res.DefToRefs, actualActiveDefSet)

	return &spec.CreateSkillSessionResponse{
		Body: &spec.CreateSkillSessionResponseBody{
			SessionID:       sid,
			ActiveSkillRefs: activeOutRefs,
		},
	}, nil
}

func (s *SkillStore) CloseSkillSession(
	ctx context.Context,
	req *spec.CloseSkillSessionRequest,
) (resp *spec.CloseSkillSessionResponse, err error) {
	if s.runtime == nil {
		return nil, fmt.Errorf("%w: runtime not configured", spec.ErrSkillInvalidRequest)
	}
	if req == nil {
		return nil, fmt.Errorf("%w: missing request", spec.ErrSkillInvalidRequest)
	}
	if err := s.runtime.CloseSession(ctx, req.SessionID); err != nil {
		return nil, err
	}
	return &spec.CloseSkillSessionResponse{}, nil
}

func (s *SkillStore) GetSkillsPromptXML(
	ctx context.Context,
	req *spec.GetSkillsPromptXMLRequest,
) (resp *spec.GetSkillsPromptXMLResponse, err error) {
	if s.runtime == nil {
		return nil, fmt.Errorf("%w: runtime not configured", spec.ErrSkillInvalidRequest)
	}

	var filter *agentskills.SkillFilter
	if req != nil && req.Body != nil && req.Body.Filter != nil {
		f := req.Body.Filter

		var allow []agentskillsSpec.SkillDef
		if len(f.AllowSkillRefs) > 0 {
			for _, r := range f.AllowSkillRefs {
				if err := validateSkillRef(r); err != nil {
					return nil, fmt.Errorf("%w: invalid filter.allowSkillRefs: %w", spec.ErrSkillInvalidRequest, err)
				}
			}
			res := s.resolveAllowSkillRefs(ctx, f.AllowSkillRefs)
			allow = res.AllowDefs
		}

		filter = &agentskills.SkillFilter{
			Types:          append([]string(nil), f.Types...),
			LocationPrefix: f.LocationPrefix,
			AllowSkills:    allow,
			SessionID:      f.SessionID,
			Activity:       agentskills.SkillActivity(strings.TrimSpace(f.Activity)),
		}
	}
	xml, err := s.runtime.SkillsPromptXML(ctx, filter)
	if err != nil {
		return nil, err
	}
	return &spec.GetSkillsPromptXMLResponse{
		Body: &spec.GetSkillsPromptXMLResponseBody{XML: xml},
	}, nil
}

func (s *SkillStore) ListRuntimeSkills(
	ctx context.Context,
	req *spec.ListRuntimeSkillsRequest,
) (resp *spec.ListRuntimeSkillsResponse, err error) {
	if s.runtime == nil {
		return nil, fmt.Errorf("%w: runtime not configured", spec.ErrSkillInvalidRequest)
	}
	if req == nil || req.Body == nil || req.Body.Filter == nil {
		return nil, fmt.Errorf("%w: missing filter", spec.ErrSkillInvalidRequest)
	}

	f := req.Body.Filter
	if len(f.AllowSkillRefs) == 0 {
		return nil, fmt.Errorf("%w: filter.allowSkillRefs required", spec.ErrSkillInvalidRequest)
	}

	for _, r := range f.AllowSkillRefs {
		if err := validateSkillRef(r); err != nil {
			return nil, fmt.Errorf("%w: invalid filter.allowSkillRefs: %w", spec.ErrSkillInvalidRequest, err)
		}
	}

	act := strings.TrimSpace(f.Activity)
	if act == "" {
		act = "any"
	}
	if act == "active" && strings.TrimSpace(string(f.SessionID)) == "" {
		return nil, fmt.Errorf("%w: activity=active requires sessionID", spec.ErrSkillInvalidRequest)
	}

	res := s.resolveAllowSkillRefs(ctx, f.AllowSkillRefs)

	// If nothing resolves (all stale/disabled/etc), return empty list (do not error).
	if len(res.AllowDefs) == 0 {
		body := &spec.ListRuntimeSkillsResponseBody{
			Skills: nil,
		}
		return &spec.ListRuntimeSkillsResponse{Body: body}, nil
	}

	filter := &agentskills.SkillListFilter{
		Types:          append([]string(nil), f.Types...),
		LocationPrefix: f.LocationPrefix,
		AllowSkills:    res.AllowDefs,
		SessionID:      f.SessionID,
		Activity:       agentskills.SkillActivity(strings.TrimSpace(f.Activity)),
	}

	recs, err := s.runtime.ListSkills(ctx, filter)
	if err != nil {
		return nil, err
	}

	activeDefSet := map[agentskillsSpec.SkillDef]struct{}{}
	needActiveSet := strings.TrimSpace(string(f.SessionID)) != "" && act == "any"
	if needActiveSet {
		activeRecs, err := s.runtime.ListSkills(ctx, &agentskills.SkillListFilter{
			SessionID:   f.SessionID,
			Activity:    agentskills.SkillActivityActive,
			AllowSkills: res.AllowDefs,
		})
		if err != nil {
			return nil, err
		}
		for _, r := range activeRecs {
			activeDefSet[r.Def] = struct{}{}
		}

	}

	items := make([]spec.RuntimeSkillListItem, 0, len(recs))
	seenItem := map[string]struct{}{}

	for _, r := range recs {
		refs := res.DefToRefs[r.Def]
		if len(refs) == 0 {
			// Defensive: should not happen when AllowSkills was derived from SkillRefs.
			continue
		}
		for _, sr := range refs {
			k := refKey(sr)
			if _, ok := seenItem[k]; ok {
				continue
			}
			seenItem[k] = struct{}{}

			items = append(items, spec.RuntimeSkillListItem{
				SkillRef:    sr,
				Type:        r.Def.Type,
				Name:        r.Def.Name,
				Description: r.Description,
				Digest:      r.Digest,
				IsActive: func() bool {
					if act == "active" {
						return true
					}
					if act == "inactive" || strings.TrimSpace(string(f.SessionID)) == "" {
						return false
					}
					_, ok := activeDefSet[r.Def]
					return ok
				}(),
			})
		}
	}

	sort.Slice(items, func(i, j int) bool {
		a := items[i].SkillRef
		b := items[j].SkillRef
		if a.BundleID != b.BundleID {
			return a.BundleID < b.BundleID
		}
		if a.SkillSlug != b.SkillSlug {
			return a.SkillSlug < b.SkillSlug
		}
		return a.SkillID < b.SkillID
	})

	out := &spec.ListRuntimeSkillsResponseBody{
		Skills: items,
	}

	return &spec.ListRuntimeSkillsResponse{
		Body: out,
	}, nil
}

// resolveRuntimeDefForSkillRef resolves a store identity (SkillRef) to a runtime SkillDef.
// If SkillID is provided, it must match the current store value (stale-ref hardening).
func (s *SkillStore) resolveRuntimeDefForSkillRef(
	ctx context.Context,
	r spec.SkillRef,
) (agentskillsSpec.SkillDef, bool) {
	bid := strings.TrimSpace(string(r.BundleID))
	slug := strings.TrimSpace(string(r.SkillSlug))
	if bid == "" || slug == "" {
		return agentskillsSpec.SkillDef{}, false
	}

	gs, err := s.GetSkill(ctx, &spec.GetSkillRequest{
		BundleID:  bundleitemutils.BundleID(bid),
		SkillSlug: spec.SkillSlug(slug),
	})
	if err != nil || gs == nil || gs.Body == nil {
		return agentskillsSpec.SkillDef{}, false
	}
	sk := *gs.Body

	// Stale ref hardening (optional).
	if strings.TrimSpace(string(r.SkillID)) != "" && sk.ID != r.SkillID {
		return agentskillsSpec.SkillDef{}, false
	}

	def, err := s.runtimeDefForStoreSkill(sk)
	if err != nil {
		return agentskillsSpec.SkillDef{}, false
	}
	return def, true
}

type resolvedAllowSkillRefs struct {
	DefToRefs map[agentskillsSpec.SkillDef][]spec.SkillRef
	RefToDef  map[string]agentskillsSpec.SkillDef
	AllowDefs []agentskillsSpec.SkillDef // unique
}

// resolveAllowSkillRefs converts allowlist SkillRefs -> runtime SkillDefs, and builds mappings both ways.
// Invalid/unresolvable refs are silently skipped (best-effort, stale-safe).
func (s *SkillStore) resolveAllowSkillRefs(
	ctx context.Context,
	allowRefs []spec.SkillRef,
) resolvedAllowSkillRefs {
	out := resolvedAllowSkillRefs{
		DefToRefs: map[agentskillsSpec.SkillDef][]spec.SkillRef{},
		RefToDef:  map[string]agentskillsSpec.SkillDef{},
		AllowDefs: nil,
	}
	if len(allowRefs) == 0 {
		return out
	}

	seenRef := map[string]struct{}{}
	seenDef := map[agentskillsSpec.SkillDef]struct{}{}

	for _, r := range allowRefs {
		k := refKey(r)
		if k == "|" || k == "" {
			continue
		}
		if _, ok := seenRef[k]; ok {
			continue
		}
		seenRef[k] = struct{}{}

		def, ok := s.resolveRuntimeDefForSkillRef(ctx, r)
		if !ok {
			continue
		}

		out.DefToRefs[def] = append(out.DefToRefs[def], r)
		out.RefToDef[k] = def

		if _, ok := seenDef[def]; !ok {
			seenDef[def] = struct{}{}
			out.AllowDefs = append(out.AllowDefs, def)
		}
	}

	sortSkillDefs(out.AllowDefs)
	return out
}

func (s *SkillStore) runtimeDefForBuiltInSkill(sk spec.Skill) (agentskillsSpec.SkillDef, error) {
	loc := sk.Location

	// Built-in embeddedfs is hydrated to disk and treated as fs in runtime.
	if !filepath.IsAbs(loc) {
		loc = filepath.Join(s.embeddedHydrateDir, filepath.FromSlash(path.Clean("/" + loc))[1:])
	}

	if strings.TrimSpace(sk.Name) == "" || strings.TrimSpace(loc) == "" {
		return agentskillsSpec.SkillDef{}, fmt.Errorf("%w: empty name/location", agentskillsSpec.ErrInvalidArgument)
	}

	return agentskillsSpec.SkillDef{
		Type:     "fs",
		Name:     sk.Name,
		Location: loc,
	}, nil
}

func buildActiveSkillRefs(
	defToRefs map[agentskillsSpec.SkillDef][]spec.SkillRef,
	activeDefs map[agentskillsSpec.SkillDef]struct{},
) []spec.SkillRef {
	if len(activeDefs) == 0 || len(defToRefs) == 0 {
		return nil
	}

	seen := map[string]struct{}{}
	out := make([]spec.SkillRef, 0)

	for def := range activeDefs {
		for _, sr := range defToRefs[def] {
			k := refKey(sr)
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			out = append(out, sr)
		}
	}

	sortSkillRefs(out)
	return out
}

func normalizeActiveRefsSubsetOfAllow(allow, active []spec.SkillRef) []spec.SkillRef {
	if len(active) == 0 || len(allow) == 0 {
		return nil
	}
	allowSet := map[string]struct{}{}
	for _, r := range allow {
		allowSet[refKey(r)] = struct{}{}
	}
	out := make([]spec.SkillRef, 0, len(active))
	seen := map[string]struct{}{}
	for _, r := range active {
		k := refKey(r)
		if _, ok := allowSet[k]; !ok {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, r)
	}
	return out
}

func sortSkillRefs(refs []spec.SkillRef) {
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].BundleID != refs[j].BundleID {
			return refs[i].BundleID < refs[j].BundleID
		}
		if refs[i].SkillSlug != refs[j].SkillSlug {
			return refs[i].SkillSlug < refs[j].SkillSlug
		}
		return refs[i].SkillID < refs[j].SkillID
	})
}

func refKey(r spec.SkillRef) string {
	return string(r.BundleID) + "|" + string(r.SkillSlug) + "|" + string(r.SkillID)
}

func validateSkillRef(r spec.SkillRef) error {
	if strings.TrimSpace(string(r.BundleID)) == "" {
		return errors.New("bundleID is empty")
	}
	if strings.TrimSpace(string(r.SkillSlug)) == "" {
		return errors.New("skillSlug is empty")
	}
	// Require SkillID for runtime-facing refs to avoid stale-ref ambiguity.
	if strings.TrimSpace(string(r.SkillID)) == "" {
		return errors.New("skillID is empty")
	}
	return nil
}
