package runtime

import (
	"testing"

	"github.com/flexigpt/agentskills-go/provider"
)

func TestMergeCatalogViews(t *testing.T) {
	first := provider.SkillDef{
		Type:     "fs",
		Name:     "same-name",
		Location: "/one",
	}
	second := provider.SkillDef{
		Type:     "fs",
		Name:     "same-name",
		Location: "/two",
	}

	tests := []struct {
		name     string
		catalogs map[CatalogID]catalogView
		want     map[provider.SkillDef]string
		wantErr  bool
	}{
		{
			name: "different definitions with same name are retained",
			catalogs: map[CatalogID]catalogView{
				"one": {first: "v1"},
				"two": {second: "v1"},
			},
			want: map[provider.SkillDef]string{
				first:  "v1",
				second: "v1",
			},
		},
		{
			name: "same definition and revision is deduplicated",
			catalogs: map[CatalogID]catalogView{
				"one": {first: "v1"},
				"two": {first: "v1"},
			},
			want: map[provider.SkillDef]string{
				first: "v1",
			},
		},
		{
			name: "same definition with conflicting revisions is removed",
			catalogs: map[CatalogID]catalogView{
				"one": {first: "v1"},
				"two": {first: "v2"},
			},
			want:    map[provider.SkillDef]string{},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := mergeCatalogViews(test.catalogs)
			if (err != nil) != test.wantErr {
				t.Fatalf("merge error=%v, wantErr=%t", err, test.wantErr)
			}
			if len(actual) != len(test.want) {
				t.Fatalf("merged definitions=%d, want %d", len(actual), len(test.want))
			}
			for definition, revision := range test.want {
				if actual[definition] != revision {
					t.Fatalf(
						"definition %+v revision=%q, want %q",
						definition,
						actual[definition],
						revision,
					)
				}
			}
		})
	}
}
