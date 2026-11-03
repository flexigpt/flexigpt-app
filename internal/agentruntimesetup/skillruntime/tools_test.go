package skillruntime

import (
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

func TestDecodeToolArgSchemaUnwrapsWailsJSONString(t *testing.T) {
	raw := jsonutil.JSONRawString(
		`"{\"type\":\"object\",\"properties\":{\"path\":{\"type\":\"string\"}}}"`,
	)

	schema, err := decodeToolArgSchema(raw)
	if err != nil {
		t.Fatalf("decodeToolArgSchema: %v", err)
	}
	if schema["type"] != "object" {
		t.Fatalf("schema.type=%#v, want object", schema["type"])
	}
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("schema.properties=%#v, want object", schema["properties"])
	}
	if _, found := properties["path"]; !found {
		t.Fatalf("path property missing: %#v", properties)
	}
}

func TestDecodeToolArgSchemaAcceptsDirectJSON(t *testing.T) {
	schema, err := decodeToolArgSchema(
		jsonutil.JSONRawString(`{"type":"object"}`),
	)
	if err != nil {
		t.Fatalf("decodeToolArgSchema: %v", err)
	}
	if schema["type"] != "object" {
		t.Fatalf("schema.type=%#v, want object", schema["type"])
	}
}

func TestDecodeToolArgSchemaUsesEmptyObjectForBlankInput(t *testing.T) {
	schema, err := decodeToolArgSchema("")
	if err != nil {
		t.Fatalf("decodeToolArgSchema: %v", err)
	}
	if schema["type"] != "object" {
		t.Fatalf("schema.type=%#v, want object", schema["type"])
	}
}

func TestSkillToolChoicesFollowPolicy(t *testing.T) {
	tests := []struct {
		name       string
		all        bool
		runScripts bool
		want       []string
	}{
		{
			name: "inactive session",
			want: []string{"skills-load"},
		},
		{
			name:       "inactive session never exposes scripts",
			runScripts: true,
			want:       []string{"skills-load"},
		},
		{
			name: "active session without scripts",
			all:  true,
			want: []string{
				"skills-load",
				"skills-unload",
				"skills-readresource",
			},
		},
		{
			name:       "active session with scripts",
			all:        true,
			runScripts: true,
			want: []string{
				"skills-load",
				"skills-unload",
				"skills-readresource",
				"skills-runscript",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			choices, err := buildSkillToolChoices(test.all, test.runScripts)
			if err != nil {
				t.Fatalf("buildSkillToolChoices: %v", err)
			}
			if len(choices) != len(test.want) {
				t.Fatalf("choices=%d, want %d", len(choices), len(test.want))
			}
			for index, choice := range choices {
				if choice.Name != test.want[index] ||
					choice.ID != "builtin."+test.want[index] {
					t.Fatalf("unexpected choice %d: %+v", index, choice)
				}
			}
		})
	}
}
