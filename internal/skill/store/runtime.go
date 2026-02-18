package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/flexigpt/agentskills-go"
	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/skill/spec"
)

const (
	embeddedHydrateDigestFile = ".embeddedfs.sha256"
)

func (s *SkillStore) CreateSkillSession(
	ctx context.Context,
	req *spec.CreateSkillSessionRequest,
) (*spec.CreateSkillSessionResponse, error) {
	if s.runtime == nil {
		return nil, fmt.Errorf("%w: runtime not configured", spec.ErrSkillInvalidRequest)
	}
	var (
		maxActive int
		keys      []agentskillsSpec.SkillKey
	)
	if req != nil && req.Body != nil {
		maxActive = req.Body.MaxActivePerSession
		keys = req.Body.ActiveKeys
	}
	opts := []agentskills.SessionOption{}
	if maxActive > 0 {
		opts = append(opts, agentskills.WithSessionMaxActivePerSession(maxActive))
	}
	if len(keys) > 0 {
		opts = append(opts, agentskills.WithSessionActiveKeys(keys))
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
) (*spec.CloseSkillSessionResponse, error) {
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
) (*spec.GetSkillsPromptXMLResponse, error) {
	if s.runtime == nil {
		return nil, fmt.Errorf("%w: runtime not configured", spec.ErrSkillInvalidRequest)
	}
	filter := getSkillsReqToRuntimeFilter(req)
	xml, err := s.runtime.SkillsPromptXML(ctx, filter)
	if err != nil {
		return nil, err
	}
	return &spec.GetSkillsPromptXMLResponse{Body: &spec.GetSkillsPromptXMLResponseBody{XML: xml}}, nil
}

func (s *SkillStore) ListRuntimeSkills(
	ctx context.Context,
	req *spec.ListRuntimeSkillsRequest,
) (*spec.ListRuntimeSkillsResponse, error) {
	if s.runtime == nil {
		return nil, fmt.Errorf("%w: runtime not configured", spec.ErrSkillInvalidRequest)
	}
	filter := listSkillsReqToRuntimeFilter(req)
	recs, err := s.runtime.ListSkills(ctx, filter)
	if err != nil {
		return nil, err
	}
	return &spec.ListRuntimeSkillsResponse{Body: &spec.ListRuntimeSkillsResponseBody{Skills: recs}}, nil
}

func (s *SkillStore) runtimeKeyForUserSkill(bundle spec.SkillBundle, sk spec.Skill) (agentskillsSpec.SkillKey, error) {
	if strings.TrimSpace(sk.Name) == "" || strings.TrimSpace(sk.Location) == "" {
		return agentskillsSpec.SkillKey{}, fmt.Errorf("%w: empty name/location", agentskillsSpec.ErrInvalidArgument)
	}
	return agentskillsSpec.SkillKey{
		Type: string(sk.Type), // usually "fs"
		SkillHandle: agentskillsSpec.SkillHandle{
			Name:     sk.Name,
			Location: sk.Location,
		},
	}, nil
}

func (s *SkillStore) runtimeKeyForBuiltInSkill(sk spec.Skill) (agentskillsSpec.SkillKey, error) {
	loc := sk.Location
	// Built-in embeddedfs is hydrated to disk and treated as fs in runtime.
	if !filepath.IsAbs(loc) {
		// Use forward-slash semantics for embedded locations, then map to OS path.
		loc = filepath.Join(s.embeddedHydrateDir, filepath.FromSlash(path.Clean("/" + loc))[1:])
	}
	if strings.TrimSpace(sk.Name) == "" || strings.TrimSpace(loc) == "" {
		return agentskillsSpec.SkillKey{}, fmt.Errorf("%w: empty name/location", agentskillsSpec.ErrInvalidArgument)
	}
	return agentskillsSpec.SkillKey{
		Type: "fs",
		SkillHandle: agentskillsSpec.SkillHandle{
			Name:     sk.Name,
			Location: loc,
		},
	}, nil
}

func (s *SkillStore) runtimeAddSkillLocked(ctx context.Context, bundle spec.SkillBundle, sk spec.Skill) error {
	if s.runtime == nil {
		return nil
	}
	key, err := s.runtimeKeyForUserSkill(bundle, sk)
	if err != nil {
		return err
	}
	_, err = s.runtime.AddSkill(ctx, key)
	return err
}

func (s *SkillStore) runtimeRemoveSkillLocked(ctx context.Context, bundle spec.SkillBundle, sk spec.Skill) error {
	if s.runtime == nil {
		return nil
	}
	key, err := s.runtimeKeyForUserSkill(bundle, sk)
	if err != nil {
		return err
	}
	_, err = s.runtime.RemoveSkill(ctx, key)
	return err
}

func (s *SkillStore) runtimeRemoveUserSkillByParts(ctx context.Context, name, location string) error {
	if s.runtime == nil {
		return nil
	}
	key := agentskillsSpec.SkillKey{
		Type: "fs",
		SkillHandle: agentskillsSpec.SkillHandle{
			Name:     name,
			Location: location,
		},
	}
	_, err := s.runtime.RemoveSkill(ctx, key)
	return err
}

func (s *SkillStore) runtimeAddBuiltInSkill(ctx context.Context, sk spec.Skill) error {
	key, err := s.runtimeKeyForBuiltInSkill(sk)
	if err != nil {
		return err
	}
	_, err = s.runtime.AddSkill(ctx, key)
	return err
}

func (s *SkillStore) runtimeRemoveBuiltInSkill(ctx context.Context, sk spec.Skill) error {
	key, err := s.runtimeKeyForBuiltInSkill(sk)
	if err != nil {
		return err
	}
	_, err = s.runtime.RemoveSkill(ctx, key)
	return err
}

func (s *SkillStore) hydrateBuiltInEmbeddedFS(ctx context.Context) error {
	if s.builtin == nil {
		return nil
	}
	// Compute digest of current embedded FS snapshot.
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
	prev, _ := os.ReadFile(digestPath) // ignore err (first boot)
	if strings.TrimSpace(string(prev)) == digest {
		return nil // already hydrated
	}

	// Re-hydrate (wipe then copy).
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

func (s *SkillStore) syncRuntimeFromStore(ctx context.Context) error {
	if s.runtime == nil {
		return nil
	}

	// Built-ins (overlay view).
	if s.builtin != nil {
		bundles, skills, err := s.builtin.ListBuiltInSkills(ctx)
		if err != nil {
			return err
		}
		// Deterministic iteration for logs/debug.
		bids := make([]string, 0, len(bundles))
		for bid := range bundles {
			bids = append(bids, string(bid))
		}
		sort.Strings(bids)
		for _, bidStr := range bids {
			bid := spec.SkillBundleID(bidStr)
			b := bundles[bid]
			if !b.IsEnabled {
				continue
			}
			for _, sk := range skills[bid] {
				if !sk.IsEnabled {
					continue
				}
				if err := s.runtimeAddBuiltInSkill(ctx, sk); err != nil {
					slog.Error("runtime add (builtin) failed", "bundleID", bid, "skill", sk.Slug, "err", err)
				}
			}
		}
	}

	// Users.
	s.mu.RLock()
	user, err := s.readAllUser(false)
	s.mu.RUnlock()
	if err != nil {
		return err
	}
	for bid, b := range user.Bundles {
		if isSoftDeletedSkillBundle(b) || !b.IsEnabled {
			continue
		}
		for _, sk := range user.Skills[bid] {
			if !sk.IsEnabled {
				continue
			}
			if err := s.runtimeAddSkillLocked(ctx, b, sk); err != nil {
				slog.Error("runtime add (user) failed", "bundleID", bid, "skill", sk.Slug, "err", err)
			}
		}
	}
	return nil
}

func getSkillsReqToRuntimeFilter(req *spec.GetSkillsPromptXMLRequest) *agentskills.SkillFilter {
	if req == nil || req.Body == nil || req.Body.Filter == nil {
		return nil
	}
	f := req.Body.Filter
	return &agentskills.SkillFilter{
		Types:          append([]string(nil), f.Types...),
		NamePrefix:     f.NamePrefix,
		LocationPrefix: f.LocationPrefix,
		AllowKeys:      append([]agentskillsSpec.SkillKey(nil), f.AllowKeys...),
		SessionID:      f.SessionID,
		Activity:       agentskills.SkillActivity(strings.TrimSpace(f.Activity)),
	}
}

func listSkillsReqToRuntimeFilter(req *spec.ListRuntimeSkillsRequest) *agentskills.SkillFilter {
	if req == nil || req.Body == nil || req.Body.Filter == nil {
		return nil
	}
	f := req.Body.Filter
	return &agentskills.SkillFilter{
		Types:          append([]string(nil), f.Types...),
		NamePrefix:     f.NamePrefix,
		LocationPrefix: f.LocationPrefix,
		AllowKeys:      append([]agentskillsSpec.SkillKey(nil), f.AllowKeys...),
		SessionID:      f.SessionID,
		Activity:       agentskills.SkillActivity(strings.TrimSpace(f.Activity)),
	}
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
