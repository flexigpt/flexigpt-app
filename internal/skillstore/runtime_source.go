package skillstore

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/skillstore/spec"
)

// SkillSource identifies a materialized source package. It is a storage/source
// projection, not a runtime catalog identity.
type SkillSource struct {
	Type     string
	Name     string
	Location string
}

// ResolveSkillSource resolves an installed record into an accessible source.
// Built-in embedded packages are materialized and translated to filesystem
// sources entirely inside skillstore.
func (s *SkillStore) ResolveSkillSource(
	value spec.Skill,
) (SkillSource, error) {
	if s == nil {
		return SkillSource{}, fmt.Errorf(
			"%w: nil Skill Store",
			errSkillInvalidRequest,
		)
	}

	source := SkillSource{
		Type:     string(value.Type),
		Name:     value.Name,
		Location: value.Location,
	}

	if value.IsBuiltIn {
		if value.Type != spec.SkillTypeEmbeddedFS {
			return SkillSource{}, fmt.Errorf(
				"%w: built-in Skill type must be %q",
				errSkillInvalidRequest,
				spec.SkillTypeEmbeddedFS,
			)
		}
		source.Type = string(spec.SkillTypeFS)
		if !filepath.IsAbs(source.Location) {
			relative := strings.ReplaceAll(source.Location, "\\", "/")
			relative = strings.TrimPrefix(path.Clean("/"+relative), "/")
			source.Location = filepath.Join(
				s.embeddedHydrateDir,
				filepath.FromSlash(relative),
			)
		}
	}

	if strings.TrimSpace(source.Type) == "" ||
		strings.TrimSpace(source.Name) == "" ||
		strings.TrimSpace(source.Location) == "" {
		return SkillSource{}, fmt.Errorf(
			"%w: resolved Skill source is incomplete",
			errSkillInvalidRequest,
		)
	}
	return source, nil
}
