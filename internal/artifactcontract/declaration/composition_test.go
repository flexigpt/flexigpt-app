package declaration_test

import (
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
)

func TestMemberForm(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		value map[string]any
		want  declaration.MemberForm
	}{
		{
			name: "locator skill reference",
			value: map[string]any{
				"type":    "skill",
				"name":    "code-review",
				"locator": "./skills/code-review",
			},
			want: declaration.MemberNamed,
		},
		{
			name: "selected MCP reference",
			value: map[string]any{
				"type":    "mcp",
				"name":    "github",
				"locator": "./.mcp.json",
				"server":  "github",
			},
			want: declaration.MemberNamed,
		},
		{
			name: "inline MCP",
			value: map[string]any{
				"type": "mcp",
				"name": "local-files",
				"parameters": map[string]any{
					"transport": "stdio",
					"command":   "npx",
				},
			},
			want: declaration.MemberContained,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			entry := mustEntry(t, testCase.value)
			got, err := entry.MemberForm()
			if err != nil {
				t.Fatalf("MemberForm() error = %v", err)
			}
			if got != testCase.want {
				t.Fatalf(
					"MemberForm() = %q, want %q",
					got,
					testCase.want,
				)
			}
		})
	}
}

func TestMemberFormRejectsNonLocalSelectorBase(t *testing.T) {
	t.Parallel()

	entry := mustEntry(t, map[string]any{
		"type": "skill",
		"base": map[string]any{
			"kind":       "git",
			"repository": "https://example.com/skills.git",
			"revision":   "main",
		},
	})

	if _, err := entry.MemberForm(); err == nil {
		t.Fatal("MemberForm() error = nil, want local selector base error")
	}
}

func TestMemberFormRejectsForeignMCPSelector(t *testing.T) {
	t.Parallel()

	entry := mustEntry(t, map[string]any{
		"type":   "skill",
		"name":   "code-review",
		"server": "github",
	})
	if _, err := entry.MemberForm(); err == nil {
		t.Fatal("MemberForm() error = nil, want invalid selector error")
	}
}

func TestWalkNamedEntriesSkipsExternalMembers(t *testing.T) {
	t.Parallel()

	root := mustEntry(t, map[string]any{
		"type": "plugin",
		"name": "developer-tools",
		"members": []any{
			map[string]any{
				"type":    "skill",
				"name":    "code-review",
				"locator": "./skills/code-review",
			},
			map[string]any{
				"type": "mcp",
				"name": "local-files",
				"parameters": map[string]any{
					"transport": "stdio",
					"command":   "npx",
				},
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

func TestWalkNamedEntriesUsesTextInsertionIdentity(t *testing.T) {
	t.Parallel()

	root := mustEntry(t, map[string]any{
		"type": "workspace",
		"name": "repository",
		"members": []any{
			map[string]any{
				"type":   "text",
				"name":   "rules",
				"insert": "instructions",
				"parameters": map[string]any{
					"content": "Keep changes focused.",
				},
			},
		},
	})

	entries, err := declaration.WalkNamedEntries(root)
	if err != nil {
		t.Fatalf("WalkNamedEntries() error = %v", err)
	}
	if got := entries[1].SubresourceLocator; got !=
		"members/text/instructions/rules" {
		t.Fatalf("contained Text subresource = %q", got)
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
