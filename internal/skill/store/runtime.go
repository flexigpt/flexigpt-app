package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/flexigpt/agentskills-go"
	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/skill/spec"
)

const (
	embeddedHydrateDigestFile = ".embeddedfs.sha256"

	// Best-effort cap for runtime resync work done inline after store mutations.
	runtimeResyncTimeout = 30 * time.Second

	// Foreground validation should be quick; runtime is in-mem, but provider indexing may touch disk.
	runtimeForegroundValidateTimeout = 15 * time.Second
)

func (s *SkillStore) CreateSkillSession(
	ctx context.Context,
	req *spec.CreateSkillSessionRequest,
) (resp *spec.CreateSkillSessionResponse, err error) {
	if s.runtime == nil {
		return nil, fmt.Errorf("%w: runtime not configured", spec.ErrSkillInvalidRequest)
	}

	var (
		maxActive    int
		activeSkills []agentskillsSpec.SkillDef
	)
	if req != nil && req.Body != nil {
		maxActive = req.Body.MaxActivePerSession
		activeSkills = req.Body.ActiveSkills
	}

	opts := []agentskills.SessionOption{}
	if maxActive > 0 {
		opts = append(opts, agentskills.WithSessionMaxActivePerSession(maxActive))
	}
	if len(activeSkills) > 0 {
		opts = append(opts, agentskills.WithSessionActiveSkills(activeSkills))
	}

	sid, active, err := s.runtime.NewSession(ctx, opts...)
	if err != nil {
		return nil, err
	}

	return &spec.CreateSkillSessionResponse{
		Body: &spec.CreateSkillSessionResponseBody{
			SessionID:    sid,
			ActiveSkills: active,
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

	filter := getSkillsReqToRuntimePromptFilter(req)
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
	defer func() {
		if r := recover(); r != nil {
			slog.Error("listRuntimeSkills: panic", "panic", r)
			resp = nil
			err = fmt.Errorf("runtime panic: %v", r)
		}
	}()

	if s.runtime == nil {
		return nil, fmt.Errorf("%w: runtime not configured", spec.ErrSkillInvalidRequest)
	}

	filter := listSkillsReqToRuntimeListFilter(req)
	recs, err := s.runtime.ListSkills(ctx, filter)
	if err != nil {
		return nil, err
	}
	return &spec.ListRuntimeSkillsResponse{
		Body: &spec.ListRuntimeSkillsResponseBody{Skills: recs},
	}, nil
}

// bestEffortRuntimeResync reconciles the runtime catalog to match the store's enabled skills.
// It must never panic and must never abort store initialization or CRUD.
//
// IMPORTANT:
//   - It does NOT persist any runtime-produced canonicalization.
//   - It uses only SkillDef (type/name/location) lifecycle selectors.
//   - It is safe to call frequently; it serializes internally.
func (s *SkillStore) bestEffortRuntimeResync(ctx context.Context, reason string) {
	if s == nil || s.runtime == nil {
		return
	}

	s.rtResyncMu.Lock()
	defer s.rtResyncMu.Unlock()

	defer func() {
		if r := recover(); r != nil {
			slog.Error("runtime resync: panic", "reason", reason, "panic", r)
		}
	}()

	ctx, cancel := context.WithTimeout(ctx, runtimeResyncTimeout)
	defer cancel()

	if err := s.runtimeResyncFromStore(ctx); err != nil {
		slog.Error("runtime resync failed", "reason", reason, "err", err)
	}
}

// runtimeResyncStrictFromStore performs a STRICT reconcile: any add/remove failure returns an error.
// This is used for rollback after store write failures.
func (s *SkillStore) runtimeResyncStrictFromStore(ctx context.Context) error {
	if s == nil || s.runtime == nil {
		return nil
	}
	desired, desiredByTypeName, err := s.runtimeDesiredSkillDefs(ctx)
	if err != nil {
		return err
	}
	return s.runtimeApplyDesiredStrict(ctx, desired, desiredByTypeName)
}

// runtimeApplyDesiredStrict reconciles runtime to match desired, failing fast on add/remove errors.
func (s *SkillStore) runtimeApplyDesiredStrict(
	ctx context.Context,
	desired map[agentskillsSpec.SkillDef]struct{},
	desiredByTypeName map[string][]agentskillsSpec.SkillDef,
) (err error) {
	if s == nil || s.runtime == nil {
		return nil
	}

	defer func() {
		if r := recover(); r != nil {
			slog.Error("runtimeApplyDesiredStrict: panic", "panic", r)
			err = fmt.Errorf("runtimeApplyDesiredStrict panic: %v", r)

		}
	}()

	currentRecs, err := s.runtime.ListSkills(ctx, nil)
	if err != nil {
		return err
	}
	currentSet := make(map[agentskillsSpec.SkillDef]struct{}, len(currentRecs))
	for _, r := range currentRecs {
		currentSet[r.Def] = struct{}{}
	}

	// Add first.
	var toAdd []agentskillsSpec.SkillDef
	for def := range desired {
		if _, ok := currentSet[def]; !ok {
			toAdd = append(toAdd, def)
		}
	}
	sortSkillDefs(toAdd)

	presentAfterAdds := make(map[agentskillsSpec.SkillDef]struct{}, len(currentSet)+len(toAdd))
	for def := range currentSet {
		presentAfterAdds[def] = struct{}{}
	}

	for _, def := range toAdd {
		if _, err := s.runtime.AddSkill(ctx, def); err != nil {
			if errors.Is(err, agentskillsSpec.ErrSkillAlreadyExists) {
				presentAfterAdds[def] = struct{}{}
				continue
			}
			return err
		}
		presentAfterAdds[def] = struct{}{}
	}

	// Replacement safety index (same as best-effort logic).
	desiredPresentTypeName := map[string]bool{}
	for def := range presentAfterAdds {
		if _, ok := desired[def]; ok {
			desiredPresentTypeName[typeNameKey(def.Type, def.Name)] = true
		}
	}

	// Remove.
	var toRemove []agentskillsSpec.SkillDef
	for def := range currentSet {
		if _, ok := desired[def]; !ok {
			toRemove = append(toRemove, def)
		}
	}
	sortSkillDefs(toRemove)

	for _, def := range toRemove {
		tn := typeNameKey(def.Type, def.Name)
		if _, hasReplacement := desiredByTypeName[tn]; hasReplacement && !desiredPresentTypeName[tn] {
			// Shouldn't happen in strict mode (add would have errored),
			// but keep safety rule.
			continue
		}

		if _, err := s.runtime.RemoveSkill(ctx, def); err != nil {
			if errors.Is(err, agentskillsSpec.ErrSkillNotFound) {
				continue
			}
			return err
		}
	}

	return nil
}

func (s *SkillStore) runtimeResyncFromStore(ctx context.Context) error {
	if s == nil || s.runtime == nil {
		return nil
	}

	desired, desiredByTypeName, err := s.runtimeDesiredSkillDefs(ctx)
	if err != nil {
		// Hard stop: do not mutate runtime if we couldn't build the desired view safely.
		return err
	}

	currentRecs, err := s.runtime.ListSkills(ctx, nil)
	if err != nil {
		return err
	}

	currentSet := make(map[agentskillsSpec.SkillDef]struct{}, len(currentRecs))
	for _, r := range currentRecs {
		currentSet[r.Def] = struct{}{}
	}

	// Determine additions first.
	var toAdd []agentskillsSpec.SkillDef
	for def := range desired {
		if _, ok := currentSet[def]; !ok {
			toAdd = append(toAdd, def)
		}
	}
	sortSkillDefs(toAdd)

	addedOK := map[agentskillsSpec.SkillDef]struct{}{}
	for _, def := range toAdd {
		if _, err := s.runtime.AddSkill(ctx, def); err != nil {
			if errors.Is(err, agentskillsSpec.ErrSkillAlreadyExists) {
				addedOK[def] = struct{}{}
				continue
			}
			slog.Error("runtime add failed", "type", def.Type, "name", def.Name, "location", def.Location, "err", err)
			continue
		}
		addedOK[def] = struct{}{}
	}

	// Build "present after adds" view for safe removals.
	presentAfterAdds := make(map[agentskillsSpec.SkillDef]struct{}, len(currentSet)+len(addedOK))
	for def := range currentSet {
		presentAfterAdds[def] = struct{}{}
	}
	for def := range addedOK {
		presentAfterAdds[def] = struct{}{}
	}

	desiredPresentTypeName := map[string]bool{}
	for def := range presentAfterAdds {
		if _, ok := desired[def]; ok {
			desiredPresentTypeName[typeNameKey(def.Type, def.Name)] = true
		}
	}

	// Determine removals.
	var toRemove []agentskillsSpec.SkillDef
	for def := range currentSet {
		if _, ok := desired[def]; !ok {
			toRemove = append(toRemove, def)
		}
	}
	sortSkillDefs(toRemove)

	for _, def := range toRemove {
		tn := typeNameKey(def.Type, def.Name)

		// Safety rule: if this looks like a "replacement" and none of the desired
		// replacements are present, skip removing the old one.
		if _, hasReplacement := desiredByTypeName[tn]; hasReplacement && !desiredPresentTypeName[tn] {
			slog.Warn(
				"runtime remove skipped (replacement missing)",
				"type", def.Type,
				"name", def.Name,
				"location", def.Location,
			)
			continue
		}

		if _, err := s.runtime.RemoveSkill(ctx, def); err != nil {
			if errors.Is(err, agentskillsSpec.ErrSkillNotFound) {
				continue
			}
			slog.Error(
				"runtime remove failed",
				"type", def.Type,
				"name", def.Name,
				"location", def.Location,
				"err", err,
			)
		}
	}

	return nil
}

// runtimeDesiredSkillDefs returns the desired enabled SkillDefs and an index by (type,name).
func (s *SkillStore) runtimeDesiredSkillDefs(
	ctx context.Context,
) (desired map[agentskillsSpec.SkillDef]struct{}, byTypeName map[string][]agentskillsSpec.SkillDef, err error) {
	desired = map[agentskillsSpec.SkillDef]struct{}{}
	byTypeName = map[string][]agentskillsSpec.SkillDef{}

	// Built-ins (overlay view).
	if s.builtin != nil {
		bundles, skills, err := s.builtin.ListBuiltInSkills(ctx)
		if err != nil {
			return nil, nil, err
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
					slog.Error("runtime desired (builtin) invalid def", "bundleID", bid, "skill", sk.Slug, "err", err)
					continue
				}
				desired[def] = struct{}{}
				k := typeNameKey(def.Type, def.Name)
				byTypeName[k] = append(byTypeName[k], def)
			}
		}
	}

	// Users.
	s.mu.RLock()
	user, err := s.readAllUser(false)
	s.mu.RUnlock()
	if err != nil {
		return nil, nil, err
	}

	for bid, b := range user.Bundles {
		if isSoftDeletedSkillBundle(b) || !b.IsEnabled {
			continue
		}
		for _, sk := range user.Skills[bid] {
			if !sk.IsEnabled {
				continue
			}
			def, err := runtimeDefForUserSkill(sk)
			if err != nil {
				slog.Error("runtime desired (user) invalid def", "bundleID", bid, "skill", sk.Slug, "err", err)
				continue
			}
			desired[def] = struct{}{}
			k := typeNameKey(def.Type, def.Name)
			byTypeName[k] = append(byTypeName[k], def)
		}
	}

	return desired, byTypeName, nil
}

func runtimeDefForUserSkill(sk spec.Skill) (agentskillsSpec.SkillDef, error) {
	if strings.TrimSpace(sk.Name) == "" || strings.TrimSpace(sk.Location) == "" {
		return agentskillsSpec.SkillDef{}, fmt.Errorf("%w: empty name/location", agentskillsSpec.ErrInvalidArgument)
	}
	return agentskillsSpec.SkillDef{
		Type:     string(sk.Type),
		Name:     sk.Name,
		Location: sk.Location,
	}, nil
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

func getSkillsReqToRuntimePromptFilter(req *spec.GetSkillsPromptXMLRequest) *agentskills.SkillFilter {
	if req == nil || req.Body == nil || req.Body.Filter == nil {
		return nil
	}
	f := req.Body.Filter
	return &agentskills.SkillFilter{
		Types:          append([]string(nil), f.Types...),
		NamePrefix:     f.NamePrefix,
		LocationPrefix: f.LocationPrefix,
		AllowSkills:    append([]agentskillsSpec.SkillDef(nil), f.AllowSkills...),
		SessionID:      f.SessionID,
		Activity:       agentskills.SkillActivity(strings.TrimSpace(f.Activity)),
	}
}

func listSkillsReqToRuntimeListFilter(req *spec.ListRuntimeSkillsRequest) *agentskills.SkillListFilter {
	if req == nil || req.Body == nil || req.Body.Filter == nil {
		return nil
	}
	f := req.Body.Filter
	return &agentskills.SkillListFilter{
		Types:          append([]string(nil), f.Types...),
		NamePrefix:     f.NamePrefix,
		LocationPrefix: f.LocationPrefix,
		AllowSkills:    append([]agentskillsSpec.SkillDef(nil), f.AllowSkills...),
		SessionID:      f.SessionID,
		Activity:       agentskills.SkillActivity(strings.TrimSpace(f.Activity)),
	}
}

func (s *SkillStore) hydrateBuiltInEmbeddedFS(ctx context.Context) error {
	if s.builtin == nil {
		return nil
	}

	defer func() {
		if r := recover(); r != nil {
			slog.Error("hydrateBuiltInEmbeddedFS: panic", "panic", r)
		}
	}()

	sub, err := resolveSkillsFS(s.builtin.skillsFS, s.builtin.skillsDir)
	if err != nil {
		return err
	}
	digest, err := fsDigestSHA256(sub)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(s.embeddedHydrateDir, 0o755); err != nil {
		return err
	}

	digestPath := filepath.Join(s.embeddedHydrateDir, embeddedHydrateDigestFile)
	prev, _ := os.ReadFile(digestPath)
	if strings.TrimSpace(string(prev)) == digest {
		return nil
	}

	if err := os.RemoveAll(s.embeddedHydrateDir); err != nil {
		return err
	}
	if err := os.MkdirAll(s.embeddedHydrateDir, 0o755); err != nil {
		return err
	}
	if err := copyFSToDir(sub, s.embeddedHydrateDir); err != nil {
		return err
	}
	if err := os.WriteFile(digestPath, []byte(digest+"\n"), 0o600); err != nil {
		return err
	}

	slog.Info("hydrated embedded skills fs", "dir", s.embeddedHydrateDir, "digest", digest)
	return nil
}

// runtimeTryAddForeground attempts to add/index a skill in runtime for strict foreground validation.
// Returns (addedByUs=true) if we successfully added; false if it already existed.
func (s *SkillStore) runtimeTryAddForeground(ctx context.Context, def agentskillsSpec.SkillDef) (bool, error) {
	if s == nil || s.runtime == nil {
		return false, fmt.Errorf("%w: runtime not configured", spec.ErrSkillInvalidRequest)
	}
	ctx, cancel := context.WithTimeout(ctx, runtimeForegroundValidateTimeout)
	defer cancel()

	if _, err := s.runtime.AddSkill(ctx, def); err != nil {
		if errors.Is(err, agentskillsSpec.ErrSkillAlreadyExists) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func sortSkillDefs(defs []agentskillsSpec.SkillDef) {
	sort.Slice(defs, func(i, j int) bool {
		if defs[i].Type != defs[j].Type {
			return defs[i].Type < defs[j].Type
		}
		if defs[i].Name != defs[j].Name {
			return defs[i].Name < defs[j].Name
		}
		return defs[i].Location < defs[j].Location
	})
}

func fsDigestSHA256(fsys fs.FS) (string, error) {
	h := sha256.New()
	var paths []string
	if err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		paths = append(paths, p)
		return nil
	}); err != nil {
		return "", err
	}

	sort.Strings(paths)
	for _, p := range paths {
		b, err := fs.ReadFile(fsys, p)
		if err != nil {
			return "", err
		}
		_, _ = io.WriteString(h, p)
		_, _ = h.Write([]byte{0})
		_, _ = h.Write(b)
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func copyFSToDir(fsys fs.FS, dest string) error {
	return fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		outPath := filepath.Join(dest, filepath.FromSlash(p))
		if d.IsDir() {
			return os.MkdirAll(outPath, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}
		in, err := fsys.Open(p)
		if err != nil {
			return err
		}
		defer in.Close()

		out, err := os.Create(outPath)
		if err != nil {
			return err
		}
		defer out.Close()

		if _, err := io.Copy(out, in); err != nil {
			return err
		}
		return nil
	})
}

func typeNameKey(typ, name string) string {
	return typ + "\x00" + name
}
