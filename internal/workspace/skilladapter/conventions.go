package skilladapter

import (
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/workspace/engine"
)

type SkillRootConvention struct {
	Root      artifactstore.Locator
	Recursive bool
}

type ConventionRegistry struct {
	roots []SkillRootConvention
}

func DefaultSkillRoots() []artifactstore.Locator {
	return []artifactstore.Locator{
		artifactstore.Locator(workspaceSkillsDirectory),
	}
}

func NewConventionRegistry(
	roots ...artifactstore.Locator,
) (*ConventionRegistry, error) {
	if len(roots) == 0 {
		roots = DefaultSkillRoots()
	}
	seen := make(map[artifactstore.Locator]struct{}, len(roots))
	values := make([]SkillRootConvention, 0, len(roots))
	for _, root := range roots {
		if err := artifactstore.ValidateLocator(root, true); err != nil {
			return nil, err
		}
		if _, duplicate := seen[root]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate Workspace Skill root %q",
				engine.ErrInvalidWorkspace,
				root,
			)
		}
		seen[root] = struct{}{}
		values = append(values, SkillRootConvention{
			Root:      root,
			Recursive: true,
		})
	}
	slices.SortFunc(values, func(left, right SkillRootConvention) int {
		return strings.Compare(string(left.Root), string(right.Root))
	})
	return &ConventionRegistry{roots: values}, nil
}

func (r *ConventionRegistry) Roots() []SkillRootConvention {
	if r == nil {
		return nil
	}
	return append([]SkillRootConvention(nil), r.roots...)
}

func (r *ConventionRegistry) DiscoveryProfile() engine.DiscoveryProfile {
	var output engine.DiscoveryProfile
	for _, root := range r.Roots() {
		output.DirectoryRoots = append(
			output.DirectoryRoots,
			discovery.DirectoryRoot{
				Root:      root.Root,
				Recursive: root.Recursive,
				IncludePatterns: []string{
					skillDefinitionFileName,
				},
			},
		)
	}
	return output
}

// Match accepts SKILL.md beneath any configured Skill root. The Skill can be
// nested at any depth, but SKILL.md must belong to a containing Skill
// directory and cannot sit directly at the configured root.
func (r *ConventionRegistry) Match(
	locator artifactstore.Locator,
) (SkillRootConvention, bool) {
	value := string(locator)
	if path.Base(value) != skillDefinitionFileName {
		return SkillRootConvention{}, false
	}
	for _, root := range r.Roots() {
		base := string(root.Root)
		relative := value
		if base != "." {
			prefix := base + "/"
			var found bool
			relative, found = strings.CutPrefix(value, prefix)
			if !found {
				continue
			}
		}
		if relative == skillDefinitionFileName {
			continue
		}
		parent := path.Dir(relative)
		if parent == "." || parent == "/" || parent == "" {
			continue
		}
		if !root.Recursive && strings.Contains(parent, "/") {
			continue
		}
		return root, true
	}
	return SkillRootConvention{}, false
}

func (r *ConventionRegistry) ExpectedName(
	locator artifactstore.Locator,
) (string, bool) {
	if _, found := r.Match(locator); !found {
		return "", false
	}
	return path.Base(path.Dir(string(locator))), true
}
