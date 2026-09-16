package declaration_test

import (
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
)

func TestCompositionForm(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		value map[string]any
		want  declaration.CompositionEntryForm
	}{
		{
			name: "locator skill reference",
			value: map[string]any{
				"type":        "skill",
				"name":        "code-review",
				"description": "Membership-local display text.",
				"locator":     "./skills/code-review",
			},
			want: declaration.CompositionEntryReference,
		},
		{
			name: "selected MCP reference",
			value: map[string]any{
				"type":    "mcp",
				"name":    "github",
				"locator": "./.mcp.json",
				"server":  "github",
			},
			want: declaration.CompositionEntryReference,
		},
		{
			name: "inline MCP",
			value: map[string]any{
				"type":      "mcp",
				"name":      "local-files",
				"transport": "stdio",
				"command":   "npx",
			},
			want: declaration.CompositionEntryContained,
		},
		{
			name: "command located MCP",
			value: map[string]any{
				"type": "mcp",
				"name": "local-files",
				"locator": map[string]any{
					"kind":    "command",
					"command": "npx",
				},
			},
			want: declaration.CompositionEntryContained,
		},
		{
			name: "empty collection declaration",
			value: map[string]any{
				"type":    "collection",
				"name":    "review",
				"members": []any{},
			},
			want: declaration.CompositionEntryContained,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			entry := mustEntry(t, testCase.value)
			got, err := entry.CompositionForm()
			if err != nil {
				t.Fatalf("CompositionForm() error = %v", err)
			}
			if got != testCase.want {
				t.Fatalf(
					"CompositionForm() = %q, want %q",
					got,
					testCase.want,
				)
			}
		})
	}
}

func TestCompositionFormRejectsForeignMCPSelector(t *testing.T) {
	t.Parallel()

	entry := mustEntry(t, map[string]any{
		"type":   "skill",
		"name":   "code-review",
		"server": "github",
	})
	if _, err := entry.CompositionForm(); err == nil {
		t.Fatal("CompositionForm() error = nil, want invalid selector error")
	}
}

func TestWalkNamedEntriesSkipsExternalMembers(t *testing.T) {
	t.Parallel()

	root := mustEntry(t, map[string]any{
		"type": "collection",
		"name": "developer-tools",
		"members": []any{
			map[string]any{
				"type":    "skill",
				"name":    "code-review",
				"locator": "./skills/code-review",
			},
			map[string]any{
				"type":      "mcp",
				"name":      "local-files",
				"transport": "stdio",
				"command":   "npx",
			},
		},
	})

	entries, err := declaration.WalkNamedEntries(root)
	if err != nil {
		t.Fatalf("WalkNamedEntries() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("WalkNamedEntries() returned %d entries, want 2", len(entries))
	}
	if got := entries[1].SubresourceLocator; got != "members/mcp/local-files" {
		t.Fatalf("contained MCP subresource = %q", got)
	}
}

func mustEntry(t *testing.T, value any) declaration.Entry {
	t.Helper()

	entry, err := declaration.NewEntry(value)
	if err != nil {
		t.Fatalf("NewEntry() error = %v", err)
	}
	return entry
}
