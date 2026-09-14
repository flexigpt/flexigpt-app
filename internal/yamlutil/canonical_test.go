package yamlutil

import "testing"

func TestCanonicalObjectJSON(t *testing.T) {
	raw := []byte(`
type: model
name: reasoning
parameters:
  maxTokens: 1e2
  temperature: 0
`)

	got, err := CanonicalObjectJSON(raw, 1024)
	if err != nil {
		t.Fatalf("CanonicalObjectJSON() error = %v", err)
	}

	const want = `{"name":"reasoning","parameters":{"maxTokens":100,"temperature":0},"type":"model"}`
	if string(got) != want {
		t.Fatalf(
			"CanonicalObjectJSON() = %s, want %s",
			got,
			want,
		)
	}
}

func TestCanonicalObjectJSONPreservesLargeIntegers(t *testing.T) {
	raw := []byte(`
type: model
name: exact
parameters:
  tokenBudget: 9007199254740993
`)

	got, err := CanonicalObjectJSON(raw, 1024)
	if err != nil {
		t.Fatalf("CanonicalObjectJSON() error = %v", err)
	}

	const want = `{"name":"exact","parameters":{"tokenBudget":9007199254740993},"type":"model"}`
	if string(got) != want {
		t.Fatalf(
			"CanonicalObjectJSON() = %s, want %s",
			got,
			want,
		)
	}
}

func TestCanonicalObjectJSONRejectsUnsafeOrAmbiguousInput(t *testing.T) {
	cases := map[string][]byte{
		"duplicate key": []byte(`
type: tool
type: model
`),
		"multiple documents": []byte(`
type: tool
---
type: model
`),
		"non-string key": []byte(`
1: tool
`),
		"yaml-only number": []byte(`
value: 0x10
`),
		"array root": []byte(`
- type: tool
`),
	}

	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := CanonicalObjectJSON(raw, 1024); err == nil {
				t.Fatal("CanonicalObjectJSON() succeeded unexpectedly")
			}
		})
	}
}

func TestCanonicalObjectJSONRejectsOversizeInput(t *testing.T) {
	raw := []byte("type: tool\nname: search\n")

	if _, err := CanonicalObjectJSON(raw, len(raw)-1); err == nil {
		t.Fatal("CanonicalObjectJSON() succeeded unexpectedly")
	}
}
