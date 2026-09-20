package topology

import (
	"slices"
	"testing"
)

func TestConfiguredDocumentAliases(t *testing.T) {
	t.Parallel()

	if !IsCollectionDocumentFile("plugin.yaml") {
		t.Fatal("plugin.yaml is not a configured collection document")
	}
	if !IsCollectionDocumentFile("collection.yaml") {
		t.Fatal("collection.yaml is not a configured collection document")
	}
	if IsCollectionDocumentFile("nested/collection.yaml") {
		t.Fatal("nested collection.yaml is not a package-root document")
	}

	if !IsMCPConfigDocument(".mcp.json") {
		t.Fatal(".mcp.json is not a configured MCP config document")
	}
	if !IsMCPConfigDocument("configs/mcp.json") {
		t.Fatal("nested mcp.json is not a configured MCP config document")
	}

	if !IsCanonicalYAMLDocument("nested/collection.yaml") {
		t.Fatal("collection.yaml is not offered to the YAML decoder")
	}
	if !IsCanonicalJSONDocument("nested/plugin.json") {
		t.Fatal("plugin.json is not offered to the JSON decoder")
	}
}

func TestWorkspaceDiscoveryIncludesHiddenMCPConfig(t *testing.T) {
	t.Parallel()

	spec, err := DiscoverySpec("workspace")
	if err != nil {
		t.Fatalf("DiscoverySpec(workspace) error = %v", err)
	}

	patterns := spec.DirectoryRoots[0].IncludePatterns
	for _, expected := range []string{
		".mcp.json",
		"**/.mcp.json",
		"collection.yaml",
		"**/collection.yaml",
	} {
		if containsPattern(patterns, expected) {
			continue
		}
		t.Fatalf("Workspace discovery lacks %q", expected)
	}
}

func TestSkillDiscoveryProfileUsesSkillDocument(t *testing.T) {
	t.Parallel()

	document, err := SkillPackageDocumentFile()
	if err != nil {
		t.Fatalf("SkillPackageDocumentFile() error = %v", err)
	}
	if document != "SKILL.md" {
		t.Fatalf(
			"SkillPackageDocumentFile() = %q, want SKILL.md",
			document,
		)
	}

	spec, err := DiscoverySpecAt("skill", "nested-skills")
	if err != nil {
		t.Fatalf("DiscoverySpecAt(skill) error = %v", err)
	}
	if len(spec.DirectoryRoots) != 1 {
		t.Fatalf(
			"skill directory root count = %d, want 1",
			len(spec.DirectoryRoots),
		)
	}
	if got := spec.DirectoryRoots[0].Root; got != "nested-skills" {
		t.Fatalf(
			"skill discovery root = %q, want nested-skills",
			got,
		)
	}
}

func containsPattern(values []string, expected string) bool {
	return slices.Contains(values, expected)
}
