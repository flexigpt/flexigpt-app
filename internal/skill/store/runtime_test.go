package store

import (
	"encoding/json"
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/flexigpt/agentskills-go"
	"github.com/flexigpt/agentskills-go/fsskillprovider"
	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/builtin"
	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/skill/spec"
)

const testBuiltins = "test-builtins"

type biSkill struct {
	Slug      spec.SkillSlug
	ID        spec.SkillID
	Name      string
	RelDir    string // embeddedfs-relative directory containing SKILL.md
	FMDesc    string // SKILL.md frontmatter description
	Body      string // SKILL.md body
	IsEnabled bool
}

type biBundle struct {
	ID          bundleitemutils.BundleID
	Slug        bundleitemutils.BundleSlug
	DisplayName string
	Description string
	IsEnabled   bool
}

func TestSkillStore_RuntimeIntegration_HydrateResync_SessionsAndFilters(t *testing.T) {
	ctx := t.Context()

	// Runtime with FS provider (the same provider used for hydrated embeddedfs content).
	p, err := fsskillprovider.New()
	if err != nil {
		t.Fatalf("fsskillprovider.New: %v", err)
	}
	rt, err := agentskills.New(agentskills.WithProvider(p))
	if err != nil {
		t.Fatalf("agentskills.New: %v", err)
	}

	baseDir := t.TempDir()
	hydrateDir := filepath.Join(baseDir, "hydrate")

	// Create store WITHOUT runtime first to avoid any init-time coupling to the real built-in FS.
	// We'll inject our temp embeddedfs into s.builtin, hydrate, then enable runtime and resync.
	s, err := NewSkillStore(baseDir, WithEmbeddedHydrateDir(hydrateDir))
	if err != nil {
		t.Fatalf("NewSkillStore: %v", err)
	}
	t.Cleanup(s.Close)

	// Inject a temp embeddedfs (MapFS) for built-ins and reload built-in snapshot from it.
	now := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
	bundle := biBundle{
		ID:          bundleitemutils.BundleID("bi-bundle-id"),
		Slug:        bundleitemutils.BundleSlug("bi-bundle"),
		DisplayName: "Built-in Bundle",
		Description: "Bundle used for runtime integration tests",
		IsEnabled:   true,
	}
	biHello := biSkill{
		Slug:      spec.SkillSlug("bi-hello"),
		ID:        spec.SkillID("bi-hello-id"),
		Name:      "bi-hello",
		RelDir:    "bundles/bi-bundle/skills/bi-hello",
		FMDesc:    "BI hello frontmatter description",
		Body:      "BI_HELLO_BODY_MARKER",
		IsEnabled: true,
	}
	biDisabled := biSkill{
		Slug:      spec.SkillSlug("bi-disabled"),
		ID:        spec.SkillID("bi-disabled-id"),
		Name:      "bi-disabled",
		RelDir:    "bundles/bi-bundle/skills/bi-disabled",
		FMDesc:    "BI disabled frontmatter description",
		Body:      "BI_DISABLED_BODY_MARKER",
		IsEnabled: false,
	}

	root := testBuiltins
	fsys := newBuiltInMapFS(t, root, now, bundle, []biSkill{biHello, biDisabled})

	// Replace the built-in FS snapshot (embeddedfs) used by BuiltInSkills, then reload.
	s.builtin.skillsFS = fsys
	s.builtin.skillsDir = root
	if err := s.builtin.loadFromFS(ctx); err != nil {
		t.Fatalf("builtin.loadFromFS: %v", err)
	}

	// Enable runtime and run the same two-step integration path:
	// 1) hydrate embeddedfs -> disk
	// 2) resync runtime catalog from enabled store skills.
	s.runtime = rt
	if err := s.hydrateBuiltInEmbeddedFS(ctx); err != nil {
		t.Fatalf("hydrateBuiltInEmbeddedFS: %v", err)
	}
	if err := s.runtimeResyncFromStore(ctx); err != nil {
		t.Fatalf("runtimeResyncFromStore: %v", err)
	}

	// Hydration assertions: digest file exists and SKILL.md exists at the hydrated path.
	if _, err := os.Stat(filepath.Join(hydrateDir, embeddedHydrateDigestFile)); err != nil {
		t.Fatalf("expected hydrate digest file to exist: %v", err)
	}
	biHelloHydratedDir := hydrateAbsDir(hydrateDir, biHello.RelDir)
	if _, err := os.Stat(filepath.Join(biHelloHydratedDir, "SKILL.md")); err != nil {
		t.Fatalf("expected hydrated SKILL.md to exist: %v", err)
	}

	// Hydration idempotency (digest match => no wipe).
	sentinel := filepath.Join(hydrateDir, "SENTINEL")
	if err := os.WriteFile(sentinel, []byte("keep"), 0o600); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}
	if err := s.hydrateBuiltInEmbeddedFS(ctx); err != nil {
		t.Fatalf("hydrateBuiltInEmbeddedFS (idempotent): %v", err)
	}
	if _, err := os.Stat(sentinel); err != nil {
		t.Fatalf("expected sentinel to remain after idempotent hydrate: %v", err)
	}

	// Runtime assertions: only enabled built-in skills should be present initially.
	recs := listRuntimeSkills(t, s)
	mustHaveSkillDef(t, recs, agentskillsSpec.SkillDef{Type: "fs", Name: biHello.Name, Location: biHelloHydratedDir})
	mustNotHaveSkillName(t, recs, biDisabled.Name)

	// Also ensure runtime record fields come from provider indexing (SKILL.md frontmatter),
	// not from store JSON metadata.
	{
		r := findByName(t, recs, biHello.Name)
		if got, want := r.Description, biHello.FMDesc; got != want {
			t.Fatalf("builtin description mismatch: got=%q want=%q", got, want)
		}
		if !strings.HasPrefix(r.Digest, "sha256:") {
			t.Fatalf("expected digest sha256:..., got %q", r.Digest)
		}
	}

	// Add user bundle + two user skills (fs).
	userSkillsRoot := filepath.Join(baseDir, "user-skills")
	userHelloDir := writeSkillPackage(
		t,
		userSkillsRoot,
		"user-hello",
		"USER hello frontmatter description",
		"USER_HELLO_BODY_MARKER",
	)
	userOtherDir := writeSkillPackage(
		t,
		userSkillsRoot,
		"user-other",
		"USER other frontmatter description",
		"USER_OTHER_BODY_MARKER",
	)

	const userBundleID = bundleitemutils.BundleID("user-bundle-id")
	if _, err := s.PutSkillBundle(ctx, &spec.PutSkillBundleRequest{
		BundleID: userBundleID,
		Body: &spec.PutSkillBundleRequestBody{
			Slug:        bundleitemutils.BundleSlug("user-bundle"),
			DisplayName: "User Bundle",
			IsEnabled:   true,
			Description: "user bundle desc",
		},
	}); err != nil {
		t.Fatalf("PutSkillBundle(user): %v", err)
	}

	// Store description != SKILL.md frontmatter description (runtime must use SKILL.md).
	if _, err := s.PutSkill(ctx, &spec.PutSkillRequest{
		BundleID:  userBundleID,
		SkillSlug: spec.SkillSlug("user-hello"),
		Body: &spec.PutSkillRequestBody{
			SkillType:   spec.SkillTypeFS,
			Location:    userHelloDir,
			Name:        "user-hello",
			IsEnabled:   true,
			DisplayName: "User Hello",
			Description: "STORE description (should not show in runtime)",
			Tags:        nil,
		},
	}); err != nil {
		t.Fatalf("PutSkill(user-hello): %v", err)
	}
	if _, err := s.PutSkill(ctx, &spec.PutSkillRequest{
		BundleID:  userBundleID,
		SkillSlug: spec.SkillSlug("user-other"),
		Body: &spec.PutSkillRequestBody{
			SkillType:   spec.SkillTypeFS,
			Location:    userOtherDir,
			Name:        "user-other",
			IsEnabled:   true,
			DisplayName: "User Other",
		},
	}); err != nil {
		t.Fatalf("PutSkill(user-other): %v", err)
	}

	recs = listRuntimeSkills(t, s)
	mustHaveSkillDef(t, recs, agentskillsSpec.SkillDef{Type: "fs", Name: "user-hello", Location: userHelloDir})
	mustHaveSkillDef(t, recs, agentskillsSpec.SkillDef{Type: "fs", Name: "user-other", Location: userOtherDir})

	{
		r := findByName(t, recs, "user-hello")
		if got, want := r.Description, "USER hello frontmatter description"; got != want {
			t.Fatalf("user-hello description mismatch: got=%q want=%q", got, want)
		}
	}

	// Create session with initial active skills (built-in + one user).
	createResp, err := s.CreateSkillSession(ctx, &spec.CreateSkillSessionRequest{
		Body: &spec.CreateSkillSessionRequestBody{
			ActiveSkills: []agentskillsSpec.SkillDef{
				{Type: "fs", Name: biHello.Name, Location: biHelloHydratedDir},
				{Type: "fs", Name: "user-hello", Location: userHelloDir},
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateSkillSession: %v", err)
	}
	sid := createResp.Body.SessionID
	if sid == "" {
		t.Fatalf("expected non-empty sessionID")
	}

	// List runtime skills with activity filters scoped to the session.
	{
		active := listRuntimeSkillsFiltered(t, s, &spec.RuntimeSkillFilter{
			SessionID: sid,
			Activity:  "active",
		})
		mustHaveSkillDef(
			t,
			active,
			agentskillsSpec.SkillDef{Type: "fs", Name: biHello.Name, Location: biHelloHydratedDir},
		)
		mustHaveSkillDef(t, active, agentskillsSpec.SkillDef{Type: "fs", Name: "user-hello", Location: userHelloDir})
		mustNotHaveSkillName(t, active, "user-other")

		inactive := listRuntimeSkillsFiltered(t, s, &spec.RuntimeSkillFilter{
			SessionID: sid,
			Activity:  "inactive",
		})
		mustHaveSkillDef(t, inactive, agentskillsSpec.SkillDef{Type: "fs", Name: "user-other", Location: userOtherDir})
		mustNotHaveSkillName(t, inactive, biHello.Name)
		mustNotHaveSkillName(t, inactive, "user-hello")
	}

	// Prompt XML: must contain active bodies and inactive metadata.
	promptResp, err := s.GetSkillsPromptXML(ctx, &spec.GetSkillsPromptXMLRequest{
		Body: &spec.GetSkillsPromptXMLRequestBody{
			Filter: &spec.RuntimeSkillFilter{
				SessionID: sid,
				Activity:  "any",
			},
		},
	})
	if err != nil {
		t.Fatalf("GetSkillsPromptXML: %v", err)
	}
	xml := promptResp.Body.XML
	mustContain(t, xml, "BI_HELLO_BODY_MARKER")
	mustContain(t, xml, "USER_HELLO_BODY_MARKER")
	mustContain(t, xml, "USER other frontmatter description")
	mustContain(t, xml, userOtherDir)

	// Ensure frontmatter is not leaked into injected skill body.
	mustNotContain(t, xml, "name: bi-hello")
	mustNotContain(t, xml, "name: user-hello")

	// Disable user bundle -> runtime should remove its skills and prune them from the session.
	if _, err := s.PatchSkillBundle(ctx, &spec.PatchSkillBundleRequest{
		BundleID: userBundleID,
		Body: &spec.PatchSkillBundleRequestBody{
			IsEnabled: false,
		},
	}); err != nil {
		t.Fatalf("PatchSkillBundle(user disable): %v", err)
	}
	recs = listRuntimeSkills(t, s)
	mustNotHaveSkillName(t, recs, "user-hello")
	mustNotHaveSkillName(t, recs, "user-other")

	activeAfterBundleDisable := listRuntimeSkillsFiltered(t, s, &spec.RuntimeSkillFilter{
		SessionID: sid,
		Activity:  "active",
	})
	mustHaveSkillDef(
		t,
		activeAfterBundleDisable,
		agentskillsSpec.SkillDef{Type: "fs", Name: biHello.Name, Location: biHelloHydratedDir},
	)
	mustNotHaveSkillName(t, activeAfterBundleDisable, "user-hello")

	// Re-enable user bundle -> runtime should add them back.
	if _, err := s.PatchSkillBundle(ctx, &spec.PatchSkillBundleRequest{
		BundleID: userBundleID,
		Body: &spec.PatchSkillBundleRequestBody{
			IsEnabled: true,
		},
	}); err != nil {
		t.Fatalf("PatchSkillBundle(user enable): %v", err)
	}
	recs = listRuntimeSkills(t, s)
	mustHaveSkillDef(t, recs, agentskillsSpec.SkillDef{Type: "fs", Name: "user-hello", Location: userHelloDir})
	mustHaveSkillDef(t, recs, agentskillsSpec.SkillDef{Type: "fs", Name: "user-other", Location: userOtherDir})

	// Disable an enabled user skill -> runtime must remove it and prune from session.
	if _, err := s.PatchSkill(ctx, &spec.PatchSkillRequest{
		BundleID:  userBundleID,
		SkillSlug: spec.SkillSlug("user-hello"),
		Body: &spec.PatchSkillRequestBody{
			IsEnabled: boolPtr(false),
		},
	}); err != nil {
		t.Fatalf("PatchSkill(user-hello disable): %v", err)
	}
	recs = listRuntimeSkills(t, s)
	mustNotHaveSkillName(t, recs, "user-hello")

	activeAfterUserDisable := listRuntimeSkillsFiltered(t, s, &spec.RuntimeSkillFilter{
		SessionID: sid,
		Activity:  "active",
	})
	mustHaveSkillDef(
		t,
		activeAfterUserDisable,
		agentskillsSpec.SkillDef{Type: "fs", Name: biHello.Name, Location: biHelloHydratedDir},
	)
	mustNotHaveSkillName(t, activeAfterUserDisable, "user-hello")

	// Enable the built-in disabled skill via built-in PatchSkill path (overlay flag + resync).
	if _, err := s.PatchSkill(ctx, &spec.PatchSkillRequest{
		BundleID:  bundle.ID,
		SkillSlug: biDisabled.Slug,
		Body: &spec.PatchSkillRequestBody{
			IsEnabled: boolPtr(true),
		},
	}); err != nil {
		t.Fatalf("PatchSkill(builtin enable disabled skill): %v", err)
	}
	recs = listRuntimeSkills(t, s)
	mustHaveSkillDef(
		t,
		recs,
		agentskillsSpec.SkillDef{
			Type:     "fs",
			Name:     biDisabled.Name,
			Location: hydrateAbsDir(hydrateDir, biDisabled.RelDir),
		},
	)

	// Disable entire built-in bundle -> runtime should remove both built-in skills and prune from session.
	if _, err := s.PatchSkillBundle(ctx, &spec.PatchSkillBundleRequest{
		BundleID: bundle.ID,
		Body: &spec.PatchSkillBundleRequestBody{
			IsEnabled: false,
		},
	}); err != nil {
		t.Fatalf("PatchSkillBundle(builtin disable bundle): %v", err)
	}
	recs = listRuntimeSkills(t, s)
	mustNotHaveSkillName(t, recs, biHello.Name)
	mustNotHaveSkillName(t, recs, biDisabled.Name)

	activeAfterBIDisable := listRuntimeSkillsFiltered(t, s, &spec.RuntimeSkillFilter{
		SessionID: sid,
		Activity:  "active",
	})
	mustNotHaveSkillName(t, activeAfterBIDisable, biHello.Name)
	mustNotHaveSkillName(t, activeAfterBIDisable, biDisabled.Name)

	// Close session and ensure session-scoped APIs error out.
	if _, err := s.CloseSkillSession(ctx, &spec.CloseSkillSessionRequest{SessionID: sid}); err != nil {
		t.Fatalf("CloseSkillSession: %v", err)
	}
	_, err = s.GetSkillsPromptXML(ctx, &spec.GetSkillsPromptXMLRequest{
		Body: &spec.GetSkillsPromptXMLRequestBody{
			Filter: &spec.RuntimeSkillFilter{
				SessionID: sid,
				Activity:  "any",
			},
		},
	})
	if err == nil || !errors.Is(err, agentskillsSpec.ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound after close; got %v", err)
	}
}

func TestSkillStore_RuntimeIntegration_ReplacementSafety_UserSkillLocationChange(t *testing.T) {
	ctx := t.Context()

	p, err := fsskillprovider.New()
	if err != nil {
		t.Fatalf("fsskillprovider.New: %v", err)
	}
	rt, err := agentskills.New(agentskills.WithProvider(p))
	if err != nil {
		t.Fatalf("agentskills.New: %v", err)
	}

	baseDir := t.TempDir()
	hydrateDir := filepath.Join(baseDir, "hydrate")
	s, err := NewSkillStore(baseDir, WithEmbeddedHydrateDir(hydrateDir))
	if err != nil {
		t.Fatalf("NewSkillStore: %v", err)
	}
	t.Cleanup(s.Close)

	// Keep built-ins minimal but valid for this test: reuse a tiny embeddedfs view.
	now := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
	bundle := biBundle{
		ID:          bundleitemutils.BundleID("bi-bundle-id"),
		Slug:        bundleitemutils.BundleSlug("bi-bundle"),
		DisplayName: "Built-in Bundle",
		Description: "Bundle used for replacement test",
		IsEnabled:   true,
	}
	biHello := biSkill{
		Slug:      spec.SkillSlug("bi-hello"),
		ID:        spec.SkillID("bi-hello-id"),
		Name:      "bi-hello",
		RelDir:    "bundles/bi-bundle/skills/bi-hello",
		FMDesc:    "BI hello",
		Body:      "BI_HELLO",
		IsEnabled: true,
	}
	root := testBuiltins
	fsys := newBuiltInMapFS(t, root, now, bundle, []biSkill{biHello})
	s.builtin.skillsFS = fsys
	s.builtin.skillsDir = root
	if err := s.builtin.loadFromFS(ctx); err != nil {
		t.Fatalf("builtin.loadFromFS: %v", err)
	}

	s.runtime = rt
	if err := s.hydrateBuiltInEmbeddedFS(ctx); err != nil {
		t.Fatalf("hydrateBuiltInEmbeddedFS: %v", err)
	}
	if err := s.runtimeResyncFromStore(ctx); err != nil {
		t.Fatalf("runtimeResyncFromStore: %v", err)
	}

	// Create a user bundle + skill with an initial valid location.
	const userBundleID = bundleitemutils.BundleID("user-bundle-id")
	if _, err := s.PutSkillBundle(ctx, &spec.PutSkillBundleRequest{
		BundleID: userBundleID,
		Body: &spec.PutSkillBundleRequestBody{
			Slug:        bundleitemutils.BundleSlug("user-bundle"),
			DisplayName: "User Bundle",
			IsEnabled:   true,
		},
	}); err != nil {
		t.Fatalf("PutSkillBundle: %v", err)
	}

	userSkillsRoot := filepath.Join(baseDir, "user-skills")
	loc1 := writeSkillPackage(t, userSkillsRoot, "replace-skill", "Replace skill v1", "REPLACE_V1_BODY")
	_, err = s.PutSkill(ctx, &spec.PutSkillRequest{
		BundleID:  userBundleID,
		SkillSlug: spec.SkillSlug("replace-skill"),
		Body: &spec.PutSkillRequestBody{
			SkillType: spec.SkillTypeFS,
			Location:  loc1,
			Name:      "replace-skill",
			IsEnabled: true,
		},
	})
	if err != nil {
		t.Fatalf("PutSkill: %v", err)
	}

	recs := listRuntimeSkills(t, s)
	mustHaveSkillDef(t, recs, agentskillsSpec.SkillDef{Type: "fs", Name: "replace-skill", Location: loc1})

	// Foreground strict behavior: patch to an invalid replacement location must FAIL
	// and must NOT persist to store.
	locBad := filepath.Join(baseDir, "does-not-exist", "replace-skill")
	_, err = s.PatchSkill(ctx, &spec.PatchSkillRequest{
		BundleID:  userBundleID,
		SkillSlug: spec.SkillSlug("replace-skill"),
		Body: &spec.PatchSkillRequestBody{
			Location: strPtr(locBad),
		},
	})
	if err == nil {
		t.Fatalf("expected PatchSkill(bad replacement location) to fail")
	}

	recs = listRuntimeSkills(t, s)
	// Old still present (not removed).
	mustHaveSkillDef(t, recs, agentskillsSpec.SkillDef{Type: "fs", Name: "replace-skill", Location: loc1})
	// Bad replacement not present.
	mustNotHaveSkillDef(t, recs, agentskillsSpec.SkillDef{Type: "fs", Name: "replace-skill", Location: locBad})

	// Store should also still be loc1.
	gs, err := s.GetSkill(
		ctx,
		&spec.GetSkillRequest{BundleID: userBundleID, SkillSlug: spec.SkillSlug("replace-skill")},
	)
	if err != nil {
		t.Fatalf("GetSkill(after failed patch): %v", err)
	}
	if got, want := gs.Body.Location, loc1; got != want {
		t.Fatalf("store location changed unexpectedly: got=%q want=%q", got, want)
	}

	// Patch to a new valid replacement location -> runtime should add new and remove old.
	loc2Parent := filepath.Join(baseDir, "user-skills-v2")
	loc2 := writeSkillPackage(t, loc2Parent, "replace-skill", "Replace skill v2", "REPLACE_V2_BODY")
	if _, err := s.PatchSkill(ctx, &spec.PatchSkillRequest{
		BundleID:  userBundleID,
		SkillSlug: spec.SkillSlug("replace-skill"),
		Body: &spec.PatchSkillRequestBody{
			Location: strPtr(loc2),
		},
	}); err != nil {
		t.Fatalf("PatchSkill(valid replacement location): %v", err)
	}

	recs = listRuntimeSkills(t, s)
	mustHaveSkillDef(t, recs, agentskillsSpec.SkillDef{Type: "fs", Name: "replace-skill", Location: loc2})
	mustNotHaveSkillDef(t, recs, agentskillsSpec.SkillDef{Type: "fs", Name: "replace-skill", Location: loc1})
}

func TestSkillStore_RuntimeIntegration_ReplacementSafety_BackgroundDrift_DoesNotRemoveLastGood(t *testing.T) {
	ctx := t.Context()

	p, err := fsskillprovider.New()
	if err != nil {
		t.Fatalf("fsskillprovider.New: %v", err)
	}
	rt, err := agentskills.New(agentskills.WithProvider(p))
	if err != nil {
		t.Fatalf("agentskills.New: %v", err)
	}

	baseDir := t.TempDir()
	hydrateDir := filepath.Join(baseDir, "hydrate")
	s, err := NewSkillStore(baseDir, WithEmbeddedHydrateDir(hydrateDir))
	if err != nil {
		t.Fatalf("NewSkillStore: %v", err)
	}
	t.Cleanup(s.Close)

	// Minimal built-ins to keep things deterministic.
	now := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
	bundle := biBundle{
		ID:          bundleitemutils.BundleID("bi-bundle-id"),
		Slug:        bundleitemutils.BundleSlug("bi-bundle"),
		DisplayName: "Built-in Bundle",
		Description: "Bundle used for drift test",
		IsEnabled:   true,
	}
	biHello := biSkill{
		Slug:      spec.SkillSlug("bi-hello"),
		ID:        spec.SkillID("bi-hello-id"),
		Name:      "bi-hello",
		RelDir:    "bundles/bi-bundle/skills/bi-hello",
		FMDesc:    "BI hello",
		Body:      "BI_HELLO",
		IsEnabled: true,
	}
	root := testBuiltins
	fsys := newBuiltInMapFS(t, root, now, bundle, []biSkill{biHello})
	s.builtin.skillsFS = fsys
	s.builtin.skillsDir = root
	if err := s.builtin.loadFromFS(ctx); err != nil {
		t.Fatalf("builtin.loadFromFS: %v", err)
	}

	s.runtime = rt
	if err := s.hydrateBuiltInEmbeddedFS(ctx); err != nil {
		t.Fatalf("hydrateBuiltInEmbeddedFS: %v", err)
	}
	if err := s.runtimeResyncFromStore(ctx); err != nil {
		t.Fatalf("runtimeResyncFromStore: %v", err)
	}

	// Create a user bundle + enabled skill at loc1 (valid).
	const userBundleID = bundleitemutils.BundleID("user-bundle-id")
	if _, err := s.PutSkillBundle(ctx, &spec.PutSkillBundleRequest{
		BundleID: userBundleID,
		Body: &spec.PutSkillBundleRequestBody{
			Slug:        bundleitemutils.BundleSlug("user-bundle"),
			DisplayName: "User Bundle",
			IsEnabled:   true,
		},
	}); err != nil {
		t.Fatalf("PutSkillBundle: %v", err)
	}

	userSkillsRoot := filepath.Join(baseDir, "user-skills")
	loc1 := writeSkillPackage(t, userSkillsRoot, "replace-skill", "Replace skill v1", "REPLACE_V1_BODY")
	if _, err := s.PutSkill(ctx, &spec.PutSkillRequest{
		BundleID:  userBundleID,
		SkillSlug: spec.SkillSlug("replace-skill"),
		Body: &spec.PutSkillRequestBody{
			SkillType: spec.SkillTypeFS,
			Location:  loc1,
			Name:      "replace-skill",
			IsEnabled: true,
		},
	}); err != nil {
		t.Fatalf("PutSkill: %v", err)
	}
	recs := listRuntimeSkills(t, s)
	mustHaveSkillDef(t, recs, agentskillsSpec.SkillDef{Type: "fs", Name: "replace-skill", Location: loc1})

	// Simulate "background drift": user manually edits the store JSON to point to a bad location.
	locBad := filepath.Join(baseDir, "does-not-exist", "replace-skill")
	metaPath := filepath.Join(baseDir, spec.SkillBundlesMetaFileName)
	raw, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("ReadFile(meta): %v", err)
	}
	var sc skillStoreSchema
	if err := json.Unmarshal(raw, &sc); err != nil {
		t.Fatalf("json.Unmarshal(meta): %v", err)
	}
	sk := sc.Skills[userBundleID][spec.SkillSlug("replace-skill")]
	sk.Location = locBad
	sc.Skills[userBundleID][spec.SkillSlug("replace-skill")] = sk
	raw2, err := json.Marshal(sc)
	if err != nil {
		t.Fatalf("json.Marshal(meta): %v", err)
	}
	if err := os.WriteFile(metaPath, raw2, 0o600); err != nil {
		t.Fatalf("WriteFile(meta): %v", err)
	}

	// Background resync: add replacement will fail; must NOT remove last-good loc1 runtime entry.
	if err := s.runtimeResyncFromStore(ctx); err != nil {
		t.Fatalf("runtimeResyncFromStore(after drift): %v", err)
	}
	recs = listRuntimeSkills(t, s)
	mustHaveSkillDef(t, recs, agentskillsSpec.SkillDef{Type: "fs", Name: "replace-skill", Location: loc1})
	mustNotHaveSkillDef(t, recs, agentskillsSpec.SkillDef{Type: "fs", Name: "replace-skill", Location: locBad})
}

func TestSkillStore_RuntimeIntegration_RuntimeEndpoints_Errors(t *testing.T) {
	ctx := t.Context()

	p, err := fsskillprovider.New()
	if err != nil {
		t.Fatalf("fsskillprovider.New: %v", err)
	}
	rt, err := agentskills.New(agentskills.WithProvider(p))
	if err != nil {
		t.Fatalf("agentskills.New: %v", err)
	}

	baseDir := t.TempDir()
	hydrateDir := filepath.Join(baseDir, "hydrate")
	s, err := NewSkillStore(baseDir, WithEmbeddedHydrateDir(hydrateDir))
	if err != nil {
		t.Fatalf("NewSkillStore: %v", err)
	}
	t.Cleanup(s.Close)

	type tc struct {
		name    string
		run     func(t *testing.T) error
		wantIs  error
		wantSub string
	}

	tests := []tc{
		{
			name: "CreateSkillSession runtime not configured",
			run: func(t *testing.T) error {
				t.Helper()
				// Ensure runtime is nil.
				s.runtime = nil
				_, err := s.CreateSkillSession(
					ctx,
					&spec.CreateSkillSessionRequest{Body: &spec.CreateSkillSessionRequestBody{}},
				)
				return err
			},
			wantIs: spec.ErrSkillInvalidRequest,
		},
		{
			name: "CloseSkillSession runtime not configured",
			run: func(t *testing.T) error {
				t.Helper()
				s.runtime = nil
				_, err := s.CloseSkillSession(
					ctx,
					&spec.CloseSkillSessionRequest{SessionID: agentskillsSpec.SessionID("x")},
				)
				return err
			},
			wantIs: spec.ErrSkillInvalidRequest,
		},
		{
			name: "CloseSkillSession missing request",
			run: func(t *testing.T) error {
				t.Helper()
				s.runtime = rt
				_, err := s.CloseSkillSession(ctx, nil)
				return err
			},
			wantIs: spec.ErrSkillInvalidRequest,
		},
		{
			name: "GetSkillsPromptXML activity=active requires sessionID",
			run: func(t *testing.T) error {
				t.Helper()
				s.runtime = rt
				_, err := s.GetSkillsPromptXML(ctx, &spec.GetSkillsPromptXMLRequest{
					Body: &spec.GetSkillsPromptXMLRequestBody{
						Filter: &spec.RuntimeSkillFilter{
							Activity: "active",
						},
					},
				})
				return err
			},
			wantIs: agentskillsSpec.ErrInvalidArgument,
		},
		{
			name: "ListRuntimeSkills activity=active requires sessionID",
			run: func(t *testing.T) error {
				t.Helper()
				s.runtime = rt
				_, err := s.ListRuntimeSkills(ctx, &spec.ListRuntimeSkillsRequest{
					Body: &spec.ListRuntimeSkillsRequestBody{
						Filter: &spec.RuntimeSkillFilter{
							Activity: "active",
						},
					},
				})
				return err
			},
			wantIs: agentskillsSpec.ErrInvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run(t)
			if err == nil {
				t.Fatalf("expected error")
			}
			if tt.wantIs != nil && !errors.Is(err, tt.wantIs) {
				t.Fatalf("expected errors.Is(err,%v)=true; got err=%v", tt.wantIs, err)
			}
			if tt.wantSub != "" && !strings.Contains(err.Error(), tt.wantSub) {
				t.Fatalf("expected error containing %q; got %v", tt.wantSub, err)
			}
		})
	}
}

func newBuiltInMapFS(t *testing.T, root string, now time.Time, b biBundle, skills []biSkill) fstest.MapFS {
	t.Helper()

	schema := skillStoreSchema{
		SchemaVersion: spec.SkillSchemaVersion,
		Bundles: map[bundleitemutils.BundleID]spec.SkillBundle{
			b.ID: {
				SchemaVersion: spec.SkillSchemaVersion,
				ID:            b.ID,
				Slug:          b.Slug,
				DisplayName:   b.DisplayName,
				Description:   b.Description,
				IsEnabled:     b.IsEnabled,
				IsBuiltIn:     true,
				CreatedAt:     now,
				ModifiedAt:    now,
				SoftDeletedAt: nil,
			},
		},
		Skills: map[bundleitemutils.BundleID]map[spec.SkillSlug]spec.Skill{
			b.ID: {},
		},
	}

	out := fstest.MapFS{}

	for _, sk := range skills {
		schema.Skills[b.ID][sk.Slug] = spec.Skill{
			SchemaVersion: spec.SkillSchemaVersion,
			ID:            sk.ID,
			Slug:          sk.Slug,
			Type:          spec.SkillTypeEmbeddedFS,
			Location:      sk.RelDir,
			Name:          sk.Name,
			DisplayName:   sk.Name,
			Description:   "STORE_SKILL_DESCRIPTION_SHOULD_NOT_AFFECT_RUNTIME",
			Tags:          nil,
			Presence:      nil,
			IsEnabled:     sk.IsEnabled,
			IsBuiltIn:     true,
			CreatedAt:     now,
			ModifiedAt:    now,
		}

		md := buildSkillMD(sk.Name, sk.FMDesc, sk.Body)
		out[path.Join(root, filepath.ToSlash(sk.RelDir), "SKILL.md")] = &fstest.MapFile{
			Data: md,
			Mode: 0o644,
		}
		// Add a small extra file so fsDigestSHA256 walks more than just SKILL.md.
		out[path.Join(root, filepath.ToSlash(sk.RelDir), "resources", "note.txt")] = &fstest.MapFile{
			Data: []byte("resource for " + sk.Name + "\n"),
			Mode: 0o644,
		}
	}

	raw, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("json.Marshal builtin schema: %v", err)
	}
	out[path.Join(root, builtin.BuiltInSkillBundlesJSON)] = &fstest.MapFile{
		Data: raw,
		Mode: 0o644,
	}

	return out
}

func listRuntimeSkills(t *testing.T, s *SkillStore) []agentskillsSpec.SkillRecord {
	t.Helper()

	resp, err := s.ListRuntimeSkills(t.Context(), nil)
	if err != nil {
		t.Fatalf("ListRuntimeSkills: %v", err)
	}
	if resp == nil || resp.Body == nil {
		t.Fatalf("ListRuntimeSkills: nil response")
	}
	return resp.Body.Skills
}

func listRuntimeSkillsFiltered(t *testing.T, s *SkillStore, f *spec.RuntimeSkillFilter) []agentskillsSpec.SkillRecord {
	t.Helper()

	resp, err := s.ListRuntimeSkills(t.Context(), &spec.ListRuntimeSkillsRequest{
		Body: &spec.ListRuntimeSkillsRequestBody{Filter: f},
	})
	if err != nil {
		t.Fatalf("ListRuntimeSkills(filtered): %v", err)
	}
	if resp == nil || resp.Body == nil {
		t.Fatalf("ListRuntimeSkills(filtered): nil response")
	}
	return resp.Body.Skills
}
