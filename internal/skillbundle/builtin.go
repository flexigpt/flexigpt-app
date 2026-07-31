package skillbundle

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/builtin"
	"github.com/flexigpt/flexigpt-app/internal/fsutil"
	"github.com/flexigpt/flexigpt-app/internal/skillartifact"
)

const (
	builtInBootstrapKey      = "builtin-agent-skills-seed"
	builtInBundleLogicalName = "built-in-agent-skills"
	builtInBundleVersion     = "v1"
	builtInBundleName        = "Built-in Skills"
	builtInDescription       = "Application-provided Agent Skills."
)

// EnsureEmbeddedBuiltInsForRoot converges the built-in Collection for one
// existing Root. It is an idempotent Artifact Store workflow, not an observer
// or a second in-memory source of membership state.
func (a *API) EnsureEmbeddedBuiltInsForRoot(
	ctx context.Context,
	rootID basespec.RootID,
) (Bundle, error) {
	if err := a.Ready(); err != nil {
		return Bundle{}, err
	}
	if err := basespec.ValidateRootID(rootID); err != nil {
		return Bundle{}, err
	}

	skills, err := embeddedBuiltInSkills(ctx)
	if err != nil {
		return Bundle{}, err
	}
	return a.bootstrapEmbeddedBuiltInsForRoot(ctx, rootID, skills)
}

func (a *API) bootstrapEmbeddedBuiltInsForRoot(
	ctx context.Context,
	rootID basespec.RootID,
	skills []BuiltInSkill,
) (Bundle, error) {
	if len(skills) == 0 {
		return Bundle{}, nil
	}
	return a.BootstrapBuiltInBundle(ctx, BootstrapBundleRequest{
		RootID:         rootID,
		BootstrapKey:   builtInBootstrapKey,
		LogicalName:    builtInBundleLogicalName,
		LogicalVersion: builtInBundleVersion,
		DisplayName:    builtInBundleName,
		Description:    builtInDescription,
		Skills:         cloneBuiltInSkills(skills),
	})
}

// BootstrapEmbeddedBuiltIns imports built-in packages through the normal
// managed Source, Collection, Artifact, publication, and refresh path.
//
// The bootstrap key is an idempotency key only. It is deliberately unrelated
// to RootID, CollectionID, ArtifactID, package publication keys, or durable
// Skill selection.
func (a *API) BootstrapEmbeddedBuiltIns(
	ctx context.Context,
) ([]Bundle, error) {
	if err := a.Ready(); err != nil {
		return nil, err
	}

	skills, err := embeddedBuiltInSkills(ctx)
	if err != nil {
		return nil, err
	}
	if len(skills) == 0 {
		return []Bundle{}, nil
	}

	roots, err := a.dependencies.Roots.List(ctx)
	if err != nil {
		return nil, err
	}

	output := make([]Bundle, 0, len(roots))
	for _, root := range roots {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		value, err := a.bootstrapEmbeddedBuiltInsForRoot(
			ctx,
			root.ID,
			skills,
		)
		if err != nil {
			return nil, err
		}
		output = append(output, value)
	}

	sort.Slice(output, func(left, right int) bool {
		if output[left].Collection.RootID != output[right].Collection.RootID {
			return output[left].Collection.RootID <
				output[right].Collection.RootID
		}
		return output[left].Collection.ID < output[right].Collection.ID
	})
	return output, nil
}

func embeddedBuiltInSkills(
	ctx context.Context,
) ([]BuiltInSkill, error) {
	root, err := fsutil.ResolveFS(
		builtin.BuiltInSkillBundlesFS,
		builtin.BuiltInSkillBundlesRootDir,
	)
	if err != nil {
		return nil, err
	}

	packageRoots := make([]string, 0)
	if err := fs.WalkDir(root, ".", func(
		location string,
		entry fs.DirEntry,
		walkErr error,
	) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() || path.Base(location) != skillartifact.DefinitionFileName {
			return nil
		}
		if path.Dir(location) == "." {
			return fmt.Errorf(
				"%w: built-in %q must be inside a package directory",
				basespec.ErrInvalid,
				skillartifact.DefinitionFileName,
			)
		}
		packageRoots = append(packageRoots, path.Dir(location))
		return nil
	}); err != nil {
		return nil, err
	}

	sort.Strings(packageRoots)
	seenNames := make(map[string]struct{}, len(packageRoots))
	output := make([]BuiltInSkill, 0, len(packageRoots))
	for _, packageRoot := range packageRoots {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		name := path.Base(packageRoot)
		if err := basespec.ValidateLogicalName(
			basespec.LogicalName(name),
		); err != nil {
			return nil, fmt.Errorf(
				"built-in package %q: %w",
				packageRoot,
				err,
			)
		}
		if _, exists := seenNames[name]; exists {
			return nil, fmt.Errorf(
				"%w: duplicate built-in Skill name %q",
				basespec.ErrConflict,
				name,
			)
		}
		seenNames[name] = struct{}{}

		files, err := readEmbeddedPackage(ctx, root, packageRoot)
		if err != nil {
			return nil, err
		}

		var skillMD []byte
		for _, file := range files {
			if file.Locator == skillartifact.DefinitionFileName {
				skillMD = append([]byte(nil), file.Content...)
				break
			}
		}
		if len(skillMD) == 0 {
			return nil, fmt.Errorf(
				"%w: built-in package %q does not contain %q",
				basespec.ErrInvalid,
				packageRoot,
				skillartifact.DefinitionFileName,
			)
		}

		output = append(output, BuiltInSkill{
			Name:    name,
			SKILLMD: skillMD,
			Files:   files,
			Enabled: true,
		})
	}
	return output, nil
}

func readEmbeddedPackage(
	ctx context.Context,
	root fs.FS,
	packageRoot string,
) ([]source.ManagedPackageFile, error) {
	files := make([]source.ManagedPackageFile, 0)
	err := fs.WalkDir(root, packageRoot, func(
		location string,
		entry fs.DirEntry,
		walkErr error,
	) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}

		relative, found := strings.CutPrefix(location, packageRoot+"/")
		if !found || relative == "" {
			return fmt.Errorf(
				"%w: invalid built-in package member %q",
				basespec.ErrInvalid,
				location,
			)
		}
		if err := basespec.ValidatePortableLocator(
			basespec.Locator(relative),
			false,
		); err != nil {
			return err
		}

		content, err := fs.ReadFile(root, location)
		if err != nil {
			return err
		}
		files = append(files, source.ManagedPackageFile{
			Locator: basespec.Locator(relative),
			Content: append([]byte(nil), content...),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(files, func(left, right int) bool {
		return files[left].Locator < files[right].Locator
	})
	return files, nil
}

func cloneBuiltInSkills(input []BuiltInSkill) []BuiltInSkill {
	output := make([]BuiltInSkill, len(input))
	for index, value := range input {
		output[index] = value
		output[index].SKILLMD = append([]byte(nil), value.SKILLMD...)
		output[index].Files = make(
			[]source.ManagedPackageFile,
			len(value.Files),
		)
		for fileIndex, file := range value.Files {
			output[index].Files[fileIndex] = source.ManagedPackageFile{
				Locator: file.Locator,
				Content: append([]byte(nil), file.Content...),
			}
		}
	}
	return output
}
