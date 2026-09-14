package builtin

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi/topology"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	skillDomain "github.com/flexigpt/flexigpt-app/internal/skill/store/domain"
)

// Skill is a non-portable application registration for one embedded Skill
// package. Artifact IDs are intentionally absent because the Store owns IDs
// during source synchronization.
type Skill struct {
	EmbeddedSkillLocator basespec.Locator `json:"embeddedSkillLocator"`
	Enabled              bool             `json:"enabled"`
}

// Registry is application-owned embedded Skill installation metadata. It is
// not a portable Collection declaration and does not create a Store entity.
type Registry struct {
	SchemaVersion string  `json:"schemaVersion"`
	Skills        []Skill `json:"skills"`
}

type HydratedSkill struct {
	Registration        Skill
	Definition          definition.Definition
	EmbeddedPackageRoot basespec.Locator
	PackageAddress      source.ManagedPackageAddress
	PackageFiles        []source.ManagedPackageFile
}

type HydratedRegistry struct {
	Registry Registry
	Skills   []HydratedSkill
}

func LoadRegistry() (Registry, error) {
	raw, err := artifactbuiltin.ReadEmbeddedSkillRegistry()
	if err != nil {
		return Registry{}, err
	}
	value, err := jsonutil.DecodeCanonicalObject[Registry](
		raw,
		basespec.MaxDefinitionBytes,
	)
	if err != nil {
		return Registry{}, fmt.Errorf(
			"decode built-in Skill registry: %w",
			err,
		)
	}
	if err := value.Validate(); err != nil {
		return Registry{}, err
	}
	return value, nil
}

func (r Registry) Validate() error {
	if r.SchemaVersion != skillDomain.RegistrySchemaVersion {
		return fmt.Errorf(
			"%w: unsupported built-in Skill registry schema %q",
			basespec.ErrInvalid,
			r.SchemaVersion,
		)
	}
	if len(r.Skills) == 0 {
		return fmt.Errorf(
			"%w: built-in Skill registry has no Skill registrations",
			basespec.ErrInvalid,
		)
	}
	if len(r.Skills) > basespec.MaxDefinitionDependencies {
		return fmt.Errorf(
			"%w: built-in Skill registry exceeds registration limit",
			basespec.ErrInvalid,
		)
	}

	seen := make(map[basespec.Locator]struct{}, len(r.Skills))
	for index, value := range r.Skills {
		if err := value.EmbeddedSkillLocator.ValidatePortable(false); err != nil {
			return fmt.Errorf("skills[%d]: %w", index, err)
		}
		if path.Base(string(value.EmbeddedSkillLocator)) !=
			string(skillDomain.SkillDefinitionFileName) ||
			path.Dir(string(value.EmbeddedSkillLocator)) == "." {
			return fmt.Errorf(
				"%w: skills[%d] must identify packaged %q",
				basespec.ErrInvalid,
				index,
				skillDomain.SkillDefinitionFileName,
			)
		}
		if _, duplicate := seen[value.EmbeddedSkillLocator]; duplicate {
			return fmt.Errorf(
				"%w: duplicate built-in Skill locator %q",
				basespec.ErrConflict,
				value.EmbeddedSkillLocator,
			)
		}
		seen[value.EmbeddedSkillLocator] = struct{}{}
	}
	return nil
}

func (r Registry) OrderedSkills() []Skill {
	output := append([]Skill(nil), r.Skills...)
	sort.Slice(output, func(left, right int) bool {
		return output[left].EmbeddedSkillLocator <
			output[right].EmbeddedSkillLocator
	})
	return output
}

func (r Registry) Hydrate(
	ctx context.Context,
	packages fs.FS,
) (HydratedRegistry, error) {
	if ctx == nil {
		return HydratedRegistry{}, fmt.Errorf(
			"%w: built-in Skill hydration context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return HydratedRegistry{}, err
	}
	if packages == nil {
		return HydratedRegistry{}, fmt.Errorf(
			"%w: built-in Skill package filesystem is nil",
			basespec.ErrInvalid,
		)
	}
	if err := r.Validate(); err != nil {
		return HydratedRegistry{}, err
	}

	output := HydratedRegistry{
		Registry: r,
		Skills:   make([]HydratedSkill, 0, len(r.Skills)),
	}
	for index, registration := range r.OrderedSkills() {
		value, err := hydrateSkill(ctx, packages, registration)
		if err != nil {
			return HydratedRegistry{}, fmt.Errorf(
				"skills[%d]: %w",
				index,
				err,
			)
		}
		output.Skills = append(output.Skills, value)
	}
	return output, nil
}

func (r HydratedRegistry) OrderedSkills() []HydratedSkill {
	output := append([]HydratedSkill(nil), r.Skills...)
	sort.Slice(output, func(left, right int) bool {
		if output[left].Definition.LogicalName !=
			output[right].Definition.LogicalName {
			return output[left].Definition.LogicalName <
				output[right].Definition.LogicalName
		}
		return output[left].Registration.EmbeddedSkillLocator <
			output[right].Registration.EmbeddedSkillLocator
	})
	return output
}

func hydrateSkill(
	ctx context.Context,
	packages fs.FS,
	registration Skill,
) (HydratedSkill, error) {
	if err := ctx.Err(); err != nil {
		return HydratedSkill{}, err
	}

	packageRoot := basespec.Locator(
		path.Dir(string(registration.EmbeddedSkillLocator)),
	)
	files, err := topology.ReadPackageFiles(
		ctx,
		packages,
		packageRoot,
	)
	if err != nil {
		return HydratedSkill{}, err
	}

	var skillMD []byte
	for _, file := range files {
		if file.Locator != skillDomain.SkillDefinitionFileName {
			continue
		}
		skillMD = append([]byte(nil), file.Content...)
		break
	}
	if len(skillMD) == 0 {
		return HydratedSkill{}, fmt.Errorf(
			"%w: embedded Skill package lacks %q",
			basespec.ErrInvalid,
			skillDomain.SkillDefinitionFileName,
		)
	}

	expectedName := path.Base(string(packageRoot))
	definitionValue, _, err := skillDomain.DecodeSkillDocument(
		skillMD,
		expectedName,
	)
	if err != nil {
		return HydratedSkill{}, err
	}
	address, err := skillDomain.ManagedPackageAddressForSkill(
		definitionValue.LogicalName,
		definitionValue.LogicalVersion,
	)
	if err != nil {
		return HydratedSkill{}, err
	}
	normalizedFiles, err := source.NormalizeManagedPackageFiles(files)
	if err != nil {
		return HydratedSkill{}, err
	}

	return HydratedSkill{
		Registration:        registration,
		Definition:          definitionValue,
		EmbeddedPackageRoot: packageRoot,
		PackageAddress:      address,
		PackageFiles:        normalizedFiles,
	}, nil
}

func (v HydratedSkill) PackageDigest() (
	cryptoutil.Digest,
	error,
) {
	return skillDomain.PackageDigest(v.PackageFiles)
}
