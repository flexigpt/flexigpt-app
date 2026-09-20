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

	spec := WorkspaceDiscoverySpec()
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

func containsPattern(values []string, expected string) bool {
	return slices.Contains(values, expected)
}
