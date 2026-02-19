package store

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flexigpt/agentskills-go"
	"github.com/flexigpt/agentskills-go/fsskillprovider"
	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/skill/spec"
)

func newTestSkillStore(t *testing.T) *SkillStore {
	t.Helper()
	p, err := fsskillprovider.New()
	if err != nil {
		t.Fatalf("fsskillprovider.New: %v", err)
	}
	rt, err := agentskills.New(agentskills.WithProvider(p))
	if err != nil {
		t.Fatalf("agentskills.New: %v", err)
	}

	s, err := NewSkillStore(t.TempDir(), WithRuntime(rt))
	if err != nil {
		t.Fatalf("NewSkillStore: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func putBundle(t *testing.T, s *SkillStore, bid, slug, displayName string, enabled bool) {
	t.Helper()
	_, err := s.PutSkillBundle(t.Context(), &spec.PutSkillBundleRequest{
		BundleID: bundleitemutils.BundleID(bid),
		Body: &spec.PutSkillBundleRequestBody{
			Slug:        bundleitemutils.BundleSlug(slug),
			DisplayName: displayName,
			IsEnabled:   enabled,
			Description: "",
		},
	})
	if err != nil {
		t.Fatalf("PutSkillBundle: %v", err)
	}
}

func putSkill(
	t *testing.T,
	s *SkillStore,
	bid, slug, skillParentDir, skillName, skillFMDesc, skillBody string,
	enabled bool,
) error {
	t.Helper()
	loc := ""
	if skillParentDir != "" {
		abs, err := filepath.Abs(skillParentDir)
		if err != nil {
			t.Fatalf("non absolute location given")
		}
		skillParentDir = abs
		loc = writeSkillPackage(t, skillParentDir, skillName, skillFMDesc, skillBody)

	}
	_, err := s.PutSkill(t.Context(), &spec.PutSkillRequest{
		BundleID:  bundleitemutils.BundleID(bid),
		SkillSlug: spec.SkillSlug(slug),
		Body: &spec.PutSkillRequestBody{
			SkillType:   spec.SkillTypeFS,
			Location:    loc,
			Name:        skillName,
			IsEnabled:   enabled,
			DisplayName: "",
			Description: "",
			Tags:        []string{"t1"},
		},
	})
	return err
}

func writeSkillPackage(t *testing.T, parentDir, name, fmDesc, body string) string {
	t.Helper()

	dir := filepath.Join(parentDir, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), buildSkillMD(name, fmDesc, body), 0o600); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	return filepath.Clean(dir)
}

func buildSkillMD(name, desc, body string) []byte {
	// Must satisfy fsskillprovider index constraints:
	// - YAML frontmatter required
	// - frontmatter.name must match directory base name
	// - frontmatter.description required.
	return []byte("" +
		"---\n" +
		"name: " + name + "\n" +
		"description: " + desc + "\n" +
		"---\n" +
		"\n" +
		body + "\n")
}

func hydrateAbsDir(hydrateDir, relLoc string) string {
	clean := path.Clean("/" + relLoc)
	if clean == "/" {
		return hydrateDir
	}
	return filepath.Join(hydrateDir, filepath.FromSlash(clean)[1:])
}

func mustHaveSkillDef(t *testing.T, recs []agentskillsSpec.SkillRecord, want agentskillsSpec.SkillDef) {
	t.Helper()
	for _, r := range recs {
		if r.Def == want {
			return
		}
	}
	t.Fatalf("expected to find skill def %+v; got defs=%v", want, defsOnly(recs))
}

func mustNotHaveSkillDef(t *testing.T, recs []agentskillsSpec.SkillRecord, want agentskillsSpec.SkillDef) {
	t.Helper()
	for _, r := range recs {
		if r.Def == want {
			t.Fatalf("expected NOT to find skill def %+v; got defs=%v", want, defsOnly(recs))
		}
	}
}

func mustNotHaveSkillName(t *testing.T, recs []agentskillsSpec.SkillRecord, name string) {
	t.Helper()
	for _, r := range recs {
		if r.Def.Name == name {
			t.Fatalf("expected NOT to find skill name %q; got defs=%v", name, defsOnly(recs))
		}
	}
}

func findByName(t *testing.T, recs []agentskillsSpec.SkillRecord, name string) agentskillsSpec.SkillRecord {
	t.Helper()
	for _, r := range recs {
		if r.Def.Name == name {
			return r
		}
	}
	t.Fatalf("skill name %q not found; got defs=%v", name, defsOnly(recs))
	return agentskillsSpec.SkillRecord{}
}

func defsOnly(recs []agentskillsSpec.SkillRecord) []agentskillsSpec.SkillDef {
	out := make([]agentskillsSpec.SkillDef, 0, len(recs))
	for _, r := range recs {
		out = append(out, r.Def)
	}
	return out
}

func mustContain(t *testing.T, s, sub string) {
	t.Helper()
	if !strings.Contains(s, sub) {
		t.Fatalf("expected output to contain %q; got:\n%s", sub, s)
	}
}

func mustNotContain(t *testing.T, s, sub string) {
	t.Helper()
	if strings.Contains(s, sub) {
		t.Fatalf("expected output to NOT contain %q; got:\n%s", sub, s)
	}
}

func boolPtr(v bool) *bool { return &v }
func strPtr(v string) *string {
	return &v
}
