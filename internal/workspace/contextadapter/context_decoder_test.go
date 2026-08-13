package contextadapter

import (
	"encoding/json"
	"maps"
	"strings"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
)

func TestContextDecoderRecognitionAndDecode(t *testing.T) {
	t.Parallel()

	decoder := NewContextDecoder()
	if decoder.ID() != contextDecoderID || decoder.Revision() != workspaceContextSchemaVersionV1 {
		t.Fatalf("decoder identity=%q/%q", decoder.ID(), decoder.Revision())
	}

	if got := decoder.Recognize(
		t.Context(),
		discovery.Candidate{
			Locator: artifactbuiltin.WorkspaceAgentsFileName,
		},
	); got != discovery.RecognitionPreferred {
		t.Fatalf("AGENTS recognition=%v", got)
	}
	if got := decoder.Recognize(t.Context(), discovery.Candidate{
		Locator:             "docs/notes.md",
		RequestedDecoderIDs: []basespec.DecoderID{contextDecoderID},
	}); got != discovery.RecognitionPossible {
		t.Fatalf("requested markdown recognition=%v", got)
	}
	if got := decoder.Recognize(t.Context(), discovery.Candidate{
		Locator:             "docs/notes.txt",
		RequestedDecoderIDs: []basespec.DecoderID{contextDecoderID},
	}); got != discovery.RecognitionNone {
		t.Fatalf("requested text recognition=%v", got)
	}

	decoded, diagnostics := decoder.Decode(t.Context(), discovery.Candidate{
		Locator: artifactbuiltin.WorkspaceAgentsFileName,
		Content: []byte("first\r\nsecond\rthird\n"),
	})
	if len(diagnostics) != 0 || len(decoded) != 1 {
		t.Fatalf("decoded=%#v diagnostics=%#v", decoded, diagnostics)
	}
	value := decoded[0].Definition
	if value.Kind != contextKind || value.SchemaID != contextSchemaID ||
		value.SchemaVersion != workspaceContextSchemaVersionV1 ||
		value.LogicalName != "agents" ||
		value.DisplayName != string(artifactbuiltin.WorkspaceAgentsFileName) ||
		value.Labels[contextRoleLabelKey] != string(artifactbuiltin.WorkspaceContextRoleAgentInstructions) {
		t.Fatalf("definition=%#v", value)
	}
	var body contextDefinition
	if err := json.Unmarshal(value.Body, &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Content != "first\nsecond\nthird\n" ||
		body.Role != string(artifactbuiltin.WorkspaceContextRoleAgentInstructions) ||
		body.MediaType != string(artifactbuiltin.WorkspaceContextMediaTypeMarkdown) {
		t.Fatalf("body=%#v", body)
	}
	if err := ValidateContextDefinition(value); err != nil {
		t.Fatalf("ValidateContextDefinition(decoded): %v", err)
	}

	decoded, diagnostics = decoder.Decode(t.Context(), discovery.Candidate{
		Locator:             "docs/notes.md",
		RequestedDecoderIDs: []basespec.DecoderID{contextDecoderID},
		Content:             []byte("project notes"),
	})
	if len(diagnostics) != 0 || len(decoded) != 1 {
		t.Fatalf("requested decode=%#v diagnostics=%#v", decoded, diagnostics)
	}
	if err := json.Unmarshal(decoded[0].Definition.Body, &body); err != nil {
		t.Fatalf("decode requested body: %v", err)
	}
	if body.Role != string(artifactbuiltin.WorkspaceContextRoleProjectContext) ||
		body.Name != "notes.md" {
		t.Fatalf("requested body=%#v", body)
	}
}

func TestContextDecoderRejectsUnsafeContentAndProfileIsStable(t *testing.T) {
	decoder := NewContextDecoder()
	for _, test := range []struct {
		name string
		data []byte
		code string
	}{
		{name: "invalid UTF-8", data: []byte{0xff}, code: engineDiagnosticInvalidUTF8()},
		{name: "nul", data: []byte("before\x00after"), code: engineDiagnosticInvalidContent()},
	} {
		t.Run(test.name, func(t *testing.T) {
			decoded, diagnostics := decoder.Decode(t.Context(), discovery.Candidate{
				Locator: artifactbuiltin.WorkspaceAgentsFileName,
				Content: test.data,
			})
			if len(decoded) != 0 || len(diagnostics) != 1 || diagnostics[0].Code != test.code {
				t.Fatalf("decoded=%#v diagnostics=%#v", decoded, diagnostics)
			}
		})
	}

	profile := DiscoveryProfile()
	if len(profile.ExplicitLocators) != 2 ||
		profile.ExplicitLocators[0] != artifactbuiltin.WorkspaceAgentsFileName ||
		profile.ExplicitLocators[1] != artifactbuiltin.WorkspaceClaudeFileName ||
		profile.ReadmeLocator != artifactbuiltin.WorkspaceReadmeFileName {
		t.Fatalf("DiscoveryProfile=%#v", profile)
	}
	profile.ExplicitLocators[0] = "changed.md"
	fresh := DiscoveryProfile()
	if fresh.ExplicitLocators[0] != artifactbuiltin.WorkspaceAgentsFileName {
		t.Fatalf("DiscoveryProfile leaked mutable backing storage: %#v", fresh)
	}
}

func TestValidateContextDefinitionAndLogicalNames(t *testing.T) {
	valid := makeContextDefinition(t, contextDefinition{
		Name:      string(artifactbuiltin.WorkspaceAgentsFileName),
		Role:      string(artifactbuiltin.WorkspaceContextRoleAgentInstructions),
		MediaType: string(artifactbuiltin.WorkspaceContextMediaTypeMarkdown),
		Content:   "instructions",
	})
	if err := ValidateContextDefinition(valid); err != nil {
		t.Fatalf("valid definition: %v", err)
	}

	cases := []struct {
		name   string
		mutate func(*definition.Definition)
	}{
		{name: "wrong kind", mutate: func(value *definition.Definition) { value.Kind = "other.kind" }},
		{name: "wrong schema", mutate: func(value *definition.Definition) { value.SchemaID = "other.schema" }},
		{name: "wrong version", mutate: func(value *definition.Definition) { value.SchemaVersion = "v2" }},
		{
			name:   "dependencies",
			mutate: func(value *definition.Definition) { value.Dependencies = []definition.Selector{{Kind: "other.kind"}} },
		},
		{name: "unknown field", mutate: func(value *definition.Definition) {
			value.Body = []byte(
				`{"name":"AGENTS.md","role":"agent-instructions","mediaType":"text/markdown","content":"x","extra":true}`,
			)
		}},
		{name: "empty content", mutate: func(value *definition.Definition) {
			value.Body = contextBodyJSON(
				t,
				contextDefinition{
					Name:      string(artifactbuiltin.WorkspaceAgentsFileName),
					Role:      string(artifactbuiltin.WorkspaceContextRoleAgentInstructions),
					MediaType: string(artifactbuiltin.WorkspaceContextMediaTypeMarkdown),
					Content:   " \t",
				},
			)
		}},
		{name: "invalid role", mutate: func(value *definition.Definition) {
			value.Body = contextBodyJSON(
				t,
				contextDefinition{
					Name:      string(artifactbuiltin.WorkspaceAgentsFileName),
					Role:      "other",
					MediaType: string(artifactbuiltin.WorkspaceContextMediaTypeMarkdown),
					Content:   "x",
				},
			)
		}},
		{name: "mismatched name", mutate: func(value *definition.Definition) { value.DisplayName = "other.md" }},
		{name: "mismatched logical name", mutate: func(value *definition.Definition) { value.LogicalName = "other" }},
		{
			name: "mismatched role label",
			mutate: func(value *definition.Definition) {
				value.Labels[contextRoleLabelKey] = string(artifactbuiltin.WorkspaceContextRoleProjectReadme)
			},
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			value := valid
			value.Labels = map[string]string{}
			maps.Copy(value.Labels, valid.Labels)
			test.mutate(&value)
			if err := ValidateContextDefinition(value); err == nil {
				t.Fatal("ValidateContextDefinition succeeded")
			}
		})
	}

	for _, test := range []struct {
		input string
		want  basespec.LogicalName
	}{
		{input: "AGENTS.md", want: "agents"},
		{input: "123 My Thing.md", want: "context-123-my-thing"},
		{input: "---.md", want: "context"},
		{input: strings.Repeat("a", basespec.MaxLogicalNameBytes+32) + ".md", want: basespec.LogicalName(strings.Repeat("a", basespec.MaxLogicalNameBytes))},
	} {
		if got := contextLogicalName(test.input); got != test.want {
			t.Errorf("contextLogicalName(%q)=%q, want %q", test.input, got, test.want)
		}
	}
}

func makeContextDefinition(t *testing.T, body contextDefinition) definition.Definition {
	t.Helper()
	return definition.Definition{
		Kind:          contextKind,
		SchemaID:      contextSchemaID,
		SchemaVersion: workspaceContextSchemaVersionV1,
		LogicalName:   contextLogicalName(body.Name),
		DisplayName:   body.Name,
		Labels:        map[string]string{contextRoleLabelKey: body.Role},
		Body:          contextBodyJSON(t, body),
	}
}

func contextBodyJSON(t *testing.T, body contextDefinition) []byte {
	t.Helper()
	value, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal context body: %v", err)
	}
	return value
}

func engineDiagnosticInvalidUTF8() string {
	return "workspace.context.invalid-utf8"
}

func engineDiagnosticInvalidContent() string {
	return "workspace.context.invalid-content"
}
