package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
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

func mustGetSkillRef(t *testing.T, s *SkillStore, bid bundleitemutils.BundleID, slug spec.SkillSlug) spec.SkillRef {
	t.Helper()

	gs, err := s.GetSkill(t.Context(), &spec.GetSkillRequest{
		BundleID:  bid,
		SkillSlug: slug,
	})
	if err != nil {
		t.Fatalf("GetSkill(%s/%s): %v", bid, slug, err)
	}
	if gs == nil || gs.Body == nil {
		t.Fatalf("GetSkill(%s/%s): nil response", bid, slug)
	}
	return spec.SkillRef{
		BundleID:  bid,
		SkillSlug: slug,
		SkillID:   gs.Body.ID,
	}
}

func mustHaveRuntimeItemBySlug(
	t *testing.T,
	items []spec.RuntimeSkillListItem,
	slug spec.SkillSlug,
) spec.RuntimeSkillListItem {
	t.Helper()
	for _, it := range items {
		if it.SkillRef.SkillSlug == slug {
			return it
		}
	}
	t.Fatalf("expected runtime list to contain slug=%q; got=%v", slug, runtimeSlugs(items))
	return spec.RuntimeSkillListItem{}
}

func mustNotHaveRuntimeItemBySlug(t *testing.T, items []spec.RuntimeSkillListItem, slug spec.SkillSlug) {
	t.Helper()
	for _, it := range items {
		if it.SkillRef.SkillSlug == slug {
			t.Fatalf("expected runtime list to NOT contain slug=%q; got=%v", slug, runtimeSlugs(items))
		}
	}
}

func runtimeSlugs(items []spec.RuntimeSkillListItem) []spec.SkillSlug {
	out := make([]spec.SkillSlug, 0, len(items))
	for _, it := range items {
		out = append(out, it.SkillRef.SkillSlug)
	}
	slices.Sort(out)
	return out
}

func listRuntimeSkillsAllow(t *testing.T, s *SkillStore, allow []spec.SkillRef) *spec.ListRuntimeSkillsResponse {
	t.Helper()

	resp, err := s.ListRuntimeSkills(t.Context(), &spec.ListRuntimeSkillsRequest{
		Body: &spec.ListRuntimeSkillsRequestBody{
			Filter: &spec.RuntimeSkillFilter{
				AllowSkillRefs: allow,
				Activity:       "any",
			},
		},
	})
	if err != nil {
		t.Fatalf("ListRuntimeSkills: %v", err)
	}
	if resp == nil || resp.Body == nil {
		t.Fatalf("ListRuntimeSkills: nil response")
	}
	return resp
}

func listRuntimeSkillsAllowFiltered(
	t *testing.T,
	s *SkillStore,
	f *spec.RuntimeSkillFilter,
) *spec.ListRuntimeSkillsResponse {
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
	return resp
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

	// Runtime endpoint assertions (store-identity based).
	biHelloRef := mustGetSkillRef(t, s, bundle.ID, biHello.Slug)
	biDisabledRef := spec.SkillRef{
		BundleID:  bundle.ID,
		SkillSlug: biDisabled.Slug,
		SkillID:   biDisabled.ID,
	} // disabled => won't resolve yet

	{
		resp := listRuntimeSkillsAllow(t, s, []spec.SkillRef{biHelloRef, biDisabledRef})
		items := resp.Body.Skills

		// Only enabled built-in skills should be present initially.
		mustHaveRuntimeItemBySlug(t, items, biHello.Slug)
		mustNotHaveRuntimeItemBySlug(t, items, biDisabled.Slug)

		// Ensure runtime record fields come from provider indexing (SKILL.md frontmatter),
		// not from store JSON metadata.
		it := mustHaveRuntimeItemBySlug(t, items, biHello.Slug)
		if got, want := it.Description, biHello.FMDesc; got != want {
			t.Fatalf("builtin description mismatch: got=%q want=%q", got, want)
		}
		if !strings.HasPrefix(it.Digest, "sha256:") {
			t.Fatalf("expected digest sha256:..., got %q", it.Digest)
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

	userHelloRef := mustGetSkillRef(t, s, userBundleID, spec.SkillSlug("user-hello"))
	userOtherRef := mustGetSkillRef(t, s, userBundleID, spec.SkillSlug("user-other"))

	allow := []spec.SkillRef{biHelloRef, userHelloRef, userOtherRef}

	{
		resp := listRuntimeSkillsAllow(t, s, allow)
		items := resp.Body.Skills

		mustHaveRuntimeItemBySlug(t, items, biHello.Slug)
		mustHaveRuntimeItemBySlug(t, items, spec.SkillSlug("user-hello"))
		mustHaveRuntimeItemBySlug(t, items, spec.SkillSlug("user-other"))

		it := mustHaveRuntimeItemBySlug(t, items, spec.SkillSlug("user-hello"))
		if got, want := it.Description, "USER hello frontmatter description"; got != want {
			t.Fatalf("user-hello description mismatch: got=%q want=%q", got, want)
		}
	}

	// Create session with initial active skills (built-in + one user) using SkillRefs only.
	createResp, err := s.CreateSkillSession(ctx, &spec.CreateSkillSessionRequest{
		Body: &spec.CreateSkillSessionRequestBody{
			AllowSkillRefs:  allow,
			ActiveSkillRefs: []spec.SkillRef{biHelloRef, userHelloRef},
		},
	})
	if err != nil {
		t.Fatalf("CreateSkillSession: %v", err)
	}
	sid := createResp.Body.SessionID
	if sid == "" {
		t.Fatalf("expected non-empty sessionID")
	}
	if len(createResp.Body.ActiveSkillRefs) == 0 {
		t.Fatalf("expected non-empty activeSkillRefs")
	}

	// List runtime skills with activity filters scoped to the session (and combined active refs).
	{
		activeResp := listRuntimeSkillsAllowFiltered(t, s, &spec.RuntimeSkillFilter{
			AllowSkillRefs: allow,
			SessionID:      sid,
			Activity:       "active",
		})
		active := activeResp.Body.Skills
		mustHaveRuntimeItemBySlug(t, active, biHello.Slug)
		mustHaveRuntimeItemBySlug(t, active, spec.SkillSlug("user-hello"))
		mustNotHaveRuntimeItemBySlug(t, active, spec.SkillSlug("user-other"))

		activeCount := 0
		for _, s := range activeResp.Body.Skills {
			if s.IsActive == true {
				activeCount++
			}
		}

		if activeCount == 0 {
			t.Fatalf("expected activeSkillRefs to be returned when sessionID is provided")
		}

		inactiveResp := listRuntimeSkillsAllowFiltered(t, s, &spec.RuntimeSkillFilter{
			AllowSkillRefs: allow,
			SessionID:      sid,
			Activity:       "inactive",
		})
		inactive := inactiveResp.Body.Skills
		mustHaveRuntimeItemBySlug(t, inactive, spec.SkillSlug("user-other"))
		mustNotHaveRuntimeItemBySlug(t, inactive, biHello.Slug)
		mustNotHaveRuntimeItemBySlug(t, inactive, spec.SkillSlug("user-hello"))
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

	{
		resp := listRuntimeSkillsAllow(t, s, allow)
		items := resp.Body.Skills
		mustHaveRuntimeItemBySlug(t, items, biHello.Slug)
		mustNotHaveRuntimeItemBySlug(t, items, spec.SkillSlug("user-hello"))
		mustNotHaveRuntimeItemBySlug(t, items, spec.SkillSlug("user-other"))
	}

	activeAfterBundleDisable := listRuntimeSkillsAllowFiltered(t, s, &spec.RuntimeSkillFilter{
		AllowSkillRefs: allow,
		SessionID:      sid,
		Activity:       "active",
	}).Body.Skills
	mustHaveRuntimeItemBySlug(t, activeAfterBundleDisable, biHello.Slug)
	mustNotHaveRuntimeItemBySlug(t, activeAfterBundleDisable, spec.SkillSlug("user-hello"))

	// Re-enable user bundle -> runtime should add them back.
	if _, err := s.PatchSkillBundle(ctx, &spec.PatchSkillBundleRequest{
		BundleID: userBundleID,
		Body: &spec.PatchSkillBundleRequestBody{
			IsEnabled: true,
		},
	}); err != nil {
		t.Fatalf("PatchSkillBundle(user enable): %v", err)
	}
	{
		resp := listRuntimeSkillsAllow(t, s, allow)
		items := resp.Body.Skills
		mustHaveRuntimeItemBySlug(t, items, spec.SkillSlug("user-hello"))
		mustHaveRuntimeItemBySlug(t, items, spec.SkillSlug("user-other"))
	}

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
	{
		resp := listRuntimeSkillsAllow(t, s, allow)
		items := resp.Body.Skills
		mustNotHaveRuntimeItemBySlug(t, items, spec.SkillSlug("user-hello"))
	}

	activeAfterUserDisable := listRuntimeSkillsAllowFiltered(t, s, &spec.RuntimeSkillFilter{
		AllowSkillRefs: allow,
		SessionID:      sid,
		Activity:       "active",
	}).Body.Skills
	mustHaveRuntimeItemBySlug(t, activeAfterUserDisable, biHello.Slug)
	mustNotHaveRuntimeItemBySlug(t, activeAfterUserDisable, spec.SkillSlug("user-hello"))

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

	// Now disabled built-in is resolvable; include it in allowlist and ensure it appears.
	biDisabledRef = mustGetSkillRef(t, s, bundle.ID, biDisabled.Slug)
	allow2 := append(slices.Clone(allow), biDisabledRef)

	{
		resp := listRuntimeSkillsAllow(t, s, allow2)
		items := resp.Body.Skills
		mustHaveRuntimeItemBySlug(t, items, biDisabled.Slug)
	}

	// Disable entire built-in bundle -> runtime should remove both built-in skills and prune from session.
	if _, err := s.PatchSkillBundle(ctx, &spec.PatchSkillBundleRequest{
		BundleID: bundle.ID,
		Body: &spec.PatchSkillBundleRequestBody{
			IsEnabled: false,
		},
	}); err != nil {
		t.Fatalf("PatchSkillBundle(builtin disable bundle): %v", err)
	}

	{
		resp := listRuntimeSkillsAllow(t, s, allow2)
		items := resp.Body.Skills
		mustNotHaveRuntimeItemBySlug(t, items, biHello.Slug)
		mustNotHaveRuntimeItemBySlug(t, items, biDisabled.Slug)
	}

	activeAfterBIDisable := listRuntimeSkillsAllowFiltered(t, s, &spec.RuntimeSkillFilter{
		AllowSkillRefs: allow2,
		SessionID:      sid,
		Activity:       "active",
	}).Body.Skills
	mustNotHaveRuntimeItemBySlug(t, activeAfterBIDisable, biHello.Slug)
	mustNotHaveRuntimeItemBySlug(t, activeAfterBIDisable, biDisabled.Slug)

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

	// Internal assertion: replacement safety is a runtime-catalog property (SkillDef includes location).
	recs, err := s.runtime.ListSkills(ctx, nil)
	if err != nil {
		t.Fatalf("runtime.ListSkills: %v", err)
	}
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

	recs, err = s.runtime.ListSkills(ctx, nil)
	if err != nil {
		t.Fatalf("runtime.ListSkills: %v", err)
	}
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

	recs, err = s.runtime.ListSkills(ctx, nil)
	if err != nil {
		t.Fatalf("runtime.ListSkills: %v", err)
	}
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

	recs, err := s.runtime.ListSkills(ctx, nil)
	if err != nil {
		t.Fatalf("runtime.ListSkills: %v", err)
	}
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

	recs, err = s.runtime.ListSkills(ctx, nil)
	if err != nil {
		t.Fatalf("runtime.ListSkills: %v", err)
	}
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

	// Create one valid user skill so ListRuntimeSkills can reach runtime validation paths.
	userSkillsRoot := filepath.Join(baseDir, "user-skills")
	loc := writeSkillPackage(t, userSkillsRoot, "err-skill", "Err skill", "ERR_SKILL_BODY")

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
	if _, err := s.PutSkill(ctx, &spec.PutSkillRequest{
		BundleID:  userBundleID,
		SkillSlug: spec.SkillSlug("err-skill"),
		Body: &spec.PutSkillRequestBody{
			SkillType: spec.SkillTypeFS,
			Location:  loc,
			Name:      "err-skill",
			IsEnabled: true,
		},
	}); err != nil {
		t.Fatalf("PutSkill: %v", err)
	}
	allow := []spec.SkillRef{mustGetSkillRef(t, s, userBundleID, spec.SkillSlug("err-skill"))}

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
					&spec.CreateSkillSessionRequest{Body: &spec.CreateSkillSessionRequestBody{AllowSkillRefs: allow}},
				)
				return err
			},
			wantIs: spec.ErrSkillInvalidRequest,
		},
		{
			name: "CreateSkillSession allowSkillRefs required",
			run: func(t *testing.T) error {
				t.Helper()
				s.runtime = rt
				_, err := s.CreateSkillSession(ctx, &spec.CreateSkillSessionRequest{
					Body: &spec.CreateSkillSessionRequestBody{},
				})
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
			name: "ListRuntimeSkills missing filter.allowSkillRefs",
			run: func(t *testing.T) error {
				t.Helper()
				s.runtime = rt
				_, err := s.ListRuntimeSkills(ctx, &spec.ListRuntimeSkillsRequest{
					Body: &spec.ListRuntimeSkillsRequestBody{
						Filter: &spec.RuntimeSkillFilter{},
					},
				})
				return err
			},
			wantIs: spec.ErrSkillInvalidRequest,
		},
		{
			name: "ListRuntimeSkills activity=active requires sessionID",
			run: func(t *testing.T) error {
				t.Helper()
				s.runtime = rt
				_, err := s.ListRuntimeSkills(ctx, &spec.ListRuntimeSkillsRequest{
					Body: &spec.ListRuntimeSkillsRequestBody{
						Filter: &spec.RuntimeSkillFilter{
							AllowSkillRefs: allow,
							Activity:       "active",
						},
					},
				})
				return err
			},
			wantIs: spec.ErrSkillInvalidRequest,
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

func TestFSDigestSHA256_StableAndSensitive(t *testing.T) {
	t.Parallel()

	fsys1 := fstest.MapFS{
		"b.txt": &fstest.MapFile{Data: []byte("B")},
		"a.txt": &fstest.MapFile{Data: []byte("A")},
	}
	d1, err := fsDigestSHA256(fsys1)
	if err != nil {
		t.Fatalf("fsDigestSHA256: %v", err)
	}

	// Same content => same digest.
	d2, err := fsDigestSHA256(fsys1)
	if err != nil {
		t.Fatalf("fsDigestSHA256: %v", err)
	}
	if d1 != d2 {
		t.Fatalf("digest not stable: %q != %q", d1, d2)
	}

	// Different content => different digest.
	fsys2 := fstest.MapFS{
		"b.txt": &fstest.MapFile{Data: []byte("B_CHANGED")},
		"a.txt": &fstest.MapFile{Data: []byte("A")},
	}
	d3, err := fsDigestSHA256(fsys2)
	if err != nil {
		t.Fatalf("fsDigestSHA256: %v", err)
	}
	if d1 == d3 {
		t.Fatalf("expected digest to change when content changes")
	}
}

func TestCopyFSToDir_CopiesNestedFiles(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"dir/note.txt": &fstest.MapFile{Data: []byte("hello\n")},
	}
	dest := t.TempDir()

	if err := copyFSToDir(fsys, dest); err != nil {
		t.Fatalf("copyFSToDir: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dest, "dir", "note.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "hello\n" {
		t.Fatalf("content mismatch: %q", string(got))
	}
}

func TestSkillStore_HydrateBuiltInEmbeddedFS_DigestMismatchWipes(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	hydrateDir := filepath.Join(baseDir, "hydrate")

	s, err := NewSkillStore(baseDir, WithEmbeddedHydrateDir(hydrateDir))
	if err != nil {
		t.Fatalf("NewSkillStore: %v", err)
	}
	t.Cleanup(s.Close)

	// Minimal FS: hydration only cares about digest + copying.
	fsys1 := fstest.MapFS{
		builtin.BuiltInSkillBundlesJSON: &fstest.MapFile{
			Data: []byte(`{"schemaVersion":"` + spec.SkillSchemaVersion + `","bundles":{},"skills":{}}`),
		},
		"x.txt": &fstest.MapFile{Data: []byte("v1")},
	}
	s.builtin.skillsFS = fsys1
	s.builtin.skillsDir = "."

	if err := s.hydrateBuiltInEmbeddedFS(t.Context()); err != nil {
		t.Fatalf("hydrateBuiltInEmbeddedFS(v1): %v", err)
	}

	sentinel := filepath.Join(hydrateDir, "SENTINEL")
	if err := os.WriteFile(sentinel, []byte("keep"), 0o600); err != nil {
		t.Fatalf("WriteFile sentinel: %v", err)
	}

	// Change FS => digest mismatch => hydrate wipes dir => sentinel should disappear.
	fsys2 := fstest.MapFS{
		builtin.BuiltInSkillBundlesJSON: &fstest.MapFile{
			Data: []byte(`{"schemaVersion":"` + spec.SkillSchemaVersion + `","bundles":{},"skills":{}}`),
		},
		"x.txt": &fstest.MapFile{Data: []byte("v2")},
	}
	s.builtin.skillsFS = fsys2
	s.builtin.skillsDir = "."

	if err := s.hydrateBuiltInEmbeddedFS(t.Context()); err != nil {
		t.Fatalf("hydrateBuiltInEmbeddedFS(v2): %v", err)
	}

	if _, err := os.Stat(sentinel); err == nil {
		t.Fatalf("expected sentinel to be removed on digest mismatch wipe")
	}
}
