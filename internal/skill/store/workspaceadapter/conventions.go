package workspaceadapter

import (
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

type SkillRootConvention struct {
	Root      basespec.Locator
	Recursive bool
}

type ConventionRegistry struct {
	roots []SkillRootConvention
}

func NewConventionRegistry(
	roots ...basespec.Locator,
) (*ConventionRegistry, error) {
	if len(roots) == 0 {
		roots = artifactbuiltin.WorkspaceSkillRoots()
	}
	seen := make(map[basespec.Locator]struct{}, len(roots))
	values := make([]SkillRootConvention, 0, len(roots))
	for _, root := range roots {
		if err := basespec.ValidateLocator(root, true); err != nil {
			return nil, err
		}
		if _, duplicate := seen[root]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate Workspace Skill root %q",
				spec.ErrInvalidWorkspace,
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

func (r *ConventionRegistry) DiscoveryProfile() spec.DiscoveryProfile {
	var output spec.DiscoveryProfile
	for _, root := range r.Roots() {
		output.DirectoryRoots = append(
			output.DirectoryRoots,
			spec.DirectoryRoot{
				Root:      root.Root,
				Recursive: root.Recursive,
				IncludePatterns: []string{
					string(artifactbuiltin.AgentSkillDefinitionFileName),
				},
			},
		)
	}
	return output
}

func (r *ConventionRegistry) ExpectedName(
	locator basespec.Locator,
) (string, bool) {
	if _, found := r.Match(locator); !found {
		return "", false
	}
	return path.Base(path.Dir(string(locator))), true
}

// Match accepts SKILL.md beneath any configured Skill root. The Skill can be
// nested at any depth, but SKILL.md must belong to a containing Skill
// directory and cannot sit directly at the configured root.
func (r *ConventionRegistry) Match(
	locator basespec.Locator,
) (SkillRootConvention, bool) {
	value := string(locator)
	if path.Base(value) != string(artifactbuiltin.AgentSkillDefinitionFileName) {
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
		if relative == string(artifactbuiltin.AgentSkillDefinitionFileName) {
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

func (r *ConventionRegistry) Roots() []SkillRootConvention {
	if r == nil {
		return nil
	}
	return append([]SkillRootConvention(nil), r.roots...)
}

func DefaultSkillRoots() []basespec.Locator {
	return artifactbuiltin.WorkspaceSkillRoots()
}
