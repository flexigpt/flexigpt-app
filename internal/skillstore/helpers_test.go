package skillstore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/skillstore/spec"
)

func newTestSkillStore(t *testing.T) *SkillStore {
	t.Helper()

	s, err := NewSkillStore(t.TempDir())
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

func writeAllUserLocked(t *testing.T, s *SkillStore, sc skillStoreSchema) {
	t.Helper()
	s.writeMu.Lock()
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.writeMu.Unlock()

	if err := s.writeAllUser(sc); err != nil {
		t.Fatalf("writeAllUser: %v", err)
	}
}

func setUserStoreAllLocked(t *testing.T, s *SkillStore, mp map[string]any) {
	t.Helper()
	s.writeMu.Lock()
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.writeMu.Unlock()

	if err := s.userStore.SetAll(mp); err != nil {
		t.Fatalf("userStore.SetAll: %v", err)
	}
}

func readAllUserLocked(t *testing.T, s *SkillStore, force bool) (skillStoreSchema, error) {
	t.Helper()

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.readAllUser(force)
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
