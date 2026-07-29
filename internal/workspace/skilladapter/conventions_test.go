package skilladapter

import (
	"errors"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/skillartifact"
	"github.com/flexigpt/flexigpt-app/internal/workspace/engine"
)

func TestConventionRegistryNormalizesAndOwnsRoots(t *testing.T) {
	t.Parallel()

	registry, err := NewConventionRegistry("z-skills", "a-skills")
	if err != nil {
		t.Fatalf("NewConventionRegistry: %v", err)
	}
	roots := registry.Roots()
	if len(roots) != 2 || roots[0].Root != "a-skills" || roots[1].Root != "z-skills" ||
		!roots[0].Recursive || !roots[1].Recursive {
		t.Fatalf("Roots=%#v", roots)
	}
	roots[0].Root = "changed"
	if fresh := registry.Roots(); fresh[0].Root != "a-skills" {
		t.Fatalf("Roots leaked mutable state: %#v", fresh)
	}

	profile := registry.DiscoveryProfile()
	if len(profile.DirectoryRoots) != 2 ||
		profile.DirectoryRoots[0].IncludePatterns[0] != skillartifact.DefinitionFileName {
		t.Fatalf("DiscoveryProfile=%#v", profile)
	}
	profile.DirectoryRoots[0].IncludePatterns[0] = "changed"
	if fresh := registry.DiscoveryProfile(); fresh.DirectoryRoots[0].IncludePatterns[0] != skillartifact.DefinitionFileName {
		t.Fatalf("DiscoveryProfile leaked mutable state: %#v", fresh)
	}

	defaults, err := NewConventionRegistry()
	if err != nil || len(defaults.Roots()) != 1 || defaults.Roots()[0].Root != DefaultWorkspaceSkillRoot {
		t.Fatalf("default registry=%#v err=%v", defaults, err)
	}

	if _, err := NewConventionRegistry("same", "same"); !errors.Is(err, engine.ErrInvalidWorkspace) {
		t.Fatalf("duplicate roots error=%v, want ErrInvalidWorkspace", err)
	}
	if _, err := NewConventionRegistry("../unsafe"); err == nil {
		t.Fatal("unsafe root was accepted")
	}
}

func TestConventionRegistryMatchAndExpectedNameBoundaries(t *testing.T) {
	t.Parallel()

	registry, err := NewConventionRegistry(".flexigpt/skills")
	if err != nil {
		t.Fatalf("NewConventionRegistry: %v", err)
	}
	for _, test := range []struct {
		locator basespec.Locator
		want    bool
		name    string
	}{
		{locator: ".flexigpt/skills/weather/SKILL.md", want: true, name: "weather"},
		{locator: ".flexigpt/skills/nested/weather/SKILL.md", want: true, name: "weather"},
		{locator: ".flexigpt/skills/SKILL.md", want: false},
		{locator: ".flexigpt/skills/weather/skill.md", want: false},
		{locator: "other/weather/SKILL.md", want: false},
	} {
		root, matched := registry.Match(test.locator)
		if matched != test.want {
			t.Errorf("Match(%q) matched=%t root=%#v, want %t", test.locator, matched, root, test.want)
		}
		name, found := registry.ExpectedName(test.locator)
		if found != test.want || (found && name != test.name) {
			t.Errorf("ExpectedName(%q)=(%q,%t), want (%q,%t)", test.locator, name, found, test.name, test.want)
		}
	}

	nonRecursive := &ConventionRegistry{roots: []SkillRootConvention{{Root: "skills", Recursive: false}}}
	if _, matched := nonRecursive.Match("skills/one/SKILL.md"); !matched {
		t.Fatal("non-recursive root did not match direct child")
	}
	if _, matched := nonRecursive.Match("skills/one/two/SKILL.md"); matched {
		t.Fatal("non-recursive root matched nested child")
	}

	rootRegistry := &ConventionRegistry{roots: []SkillRootConvention{{Root: ".", Recursive: true}}}
	if _, matched := rootRegistry.Match("one/SKILL.md"); !matched {
		t.Fatal("root convention did not match nested skill")
	}
	if _, matched := rootRegistry.Match("SKILL.md"); matched {
		t.Fatal("root convention matched root-level SKILL.md")
	}
}
