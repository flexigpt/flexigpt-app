package artifactbuiltin

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/protection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source/managed"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/topology"
	"github.com/flexigpt/flexigpt-app/internal/builtin/metadata"
	"github.com/flexigpt/flexigpt-app/internal/skillbundle"
)

type skillInstaller interface {
	ListBundles(
		ctx context.Context,
		rootID basespec.RootID,
	) ([]skillbundle.Bundle, error)
	ListSkills(
		ctx context.Context,
		ref collection.CollectionRef,
	) ([]artifact.Artifact, error)
	EnsureBuiltInBundleTopology(
		ctx context.Context,
		t skillbundle.BuiltInBundleTopology,
	) (skillbundle.Bundle, error)
	InstallBuiltInCollection(
		ctx context.Context,
		c skillbundle.BuiltInCollectionInstallRequest,
	) ([]skillbundle.CreateManagedSkillResponse, error)
	EnsureBuiltInBundleCurrent(
		ctx context.Context,
		ref collection.CollectionRef,
	) error
}

type InstallerDependencies struct {
	Topology topology.Ensurer
	Skills   skillInstaller
	Registry metadata.Registry
	Packages fs.FS
}

type Installer struct {
	topology topology.Ensurer
	skills   skillInstaller
	registry metadata.Registry
	hydrated metadata.HydratedRegistry
	packages fs.FS
}

func NewInstaller(
	dependencies InstallerDependencies,
) (*Installer, error) {
	if dependencies.Topology == nil ||
		dependencies.Skills == nil ||
		dependencies.Packages == nil {
		return nil, fmt.Errorf("%w: built-in installer dependencies are incomplete", basespec.ErrInvalid)
	}
	if err := dependencies.Registry.Validate(); err != nil {
		return nil, err
	}
	if dependencies.Registry.Source.Kind != managed.Kind {
		return nil, fmt.Errorf(
			"%w: built-in Source kind must be %q",
			basespec.ErrInvalid,
			managed.Kind,
		)
	}
	hydrated, err := dependencies.Registry.Hydrate(
		dependencies.Packages,
	)
	if err != nil {
		return nil, err
	}
	return &Installer{
		topology: dependencies.Topology,
		skills:   dependencies.Skills,
		registry: dependencies.Registry,
		hydrated: hydrated,
		packages: dependencies.Packages,
	}, nil
}

func (i *Installer) EnsureBuiltInSystemRoot(
	ctx context.Context,
) (root.Root, error) {
	t, err := i.ensureTopology(ctx)
	if err != nil {
		return root.Root{}, err
	}
	return t.Root, nil
}

func (i *Installer) EnsureBuiltInSource(
	ctx context.Context,
) (source.Summary, error) {
	t, err := i.ensureTopology(ctx)
	if err != nil {
		return source.Summary{}, err
	}
	if len(t.Sources) != 1 ||
		t.Sources[0].ID != i.registry.Source.ID {
		return source.Summary{}, fmt.Errorf(
			"%w: protected built-in Source declaration was not satisfied",
			basespec.ErrInvalid,
		)
	}
	return t.Sources[0], nil
}

func (i *Installer) EnsureBuiltInBundles(
	ctx context.Context,
) ([]skillbundle.Bundle, error) {
	ctx = protection.WithPrivilegedInstaller(ctx)
	if _, err := i.EnsureBuiltInSource(ctx); err != nil {
		return nil, err
	}
	if err := i.rejectDynamicBuiltInBundles(ctx); err != nil {
		return nil, err
	}

	output := make([]skillbundle.Bundle, 0, len(i.hydrated.Collections))
	for _, value := range i.hydrated.OrderedCollections() {
		bundle, err := i.skills.EnsureBuiltInBundleTopology(
			ctx,
			skillbundle.BuiltInBundleTopology{
				RootID:                   i.registry.Root.ID,
				CollectionID:             value.Registration.ID,
				SourceID:                 i.registry.Source.ID,
				LogicalName:              value.Definition.LogicalName,
				LogicalVersion:           value.Definition.LogicalVersion,
				DisplayName:              value.Definition.DisplayName,
				Description:              value.Definition.Description,
				Labels:                   value.Definition.Labels,
				Enabled:                  value.Registration.Enabled,
				DiscoveryRoot:            value.SourceScope,
				ExpectedMemberDigests:    value.ExpectedMemberDigests,
				PortableDefinitionDigest: value.Definition.Digest,
			},
		)
		if err != nil {
			return nil, err
		}
		output = append(output, bundle)
	}
	return output, nil
}

func (i *Installer) EnsureBuiltInArtifacts(
	ctx context.Context,
) error {
	ctx = protection.WithPrivilegedInstaller(ctx)
	bundles, err := i.EnsureBuiltInBundles(ctx)
	if err != nil {
		return err
	}
	byCollectionID := make(map[basespec.CollectionID]skillbundle.Bundle, len(bundles))
	for _, bundle := range bundles {
		byCollectionID[bundle.Collection.ID] = bundle
	}

	for _, value := range i.hydrated.OrderedCollections() {
		current, exists := byCollectionID[value.Registration.ID]
		if !exists {
			return fmt.Errorf(
				"%w: built-in Collection %q was not created",
				basespec.ErrInvalid,
				value.Definition.LogicalName,
			)
		}
		if err := i.rejectDynamicBuiltInArtifacts(ctx, current, value); err != nil {
			return err
		}

		files, err := i.packageFiles(ctx, value.SourceScope)
		if err != nil {
			return err
		}
		request := skillbundle.BuiltInCollectionInstallRequest{
			Bundle:                     current.Collection.Ref(),
			ExpectedCollectionRevision: current.Collection.Revision,
			PackageDirectory:           value.SourceScope,
			PackageFiles:               files,
			Skills: make(
				[]skillbundle.BuiltInCollectionSkill,
				0,
				len(value.Artifacts),
			),
		}
		for _, skill := range value.Artifacts {
			request.Skills = append(request.Skills, skillbundle.BuiltInCollectionSkill{
				ArtifactID: skill.Registration.ID,
				Member:     skill.Member.Locator,
				Enabled:    skill.Registration.Enabled,
			})
		}
		installed, err := i.skills.InstallBuiltInCollection(ctx, request)
		if err != nil {
			return err
		}
		if len(installed) != len(value.Artifacts) {
			return fmt.Errorf(
				"%w: built-in Collection %q returned %d Artifacts, expected %d",
				basespec.ErrInvalid,
				value.Definition.LogicalName,
				len(installed),
				len(value.Artifacts),
			)
		}
		for _, skill := range value.Artifacts {
			found := false
			for _, result := range installed {
				if result.Artifact.ID != skill.Registration.ID {
					continue
				}
				found = result.Artifact.RootID == i.registry.Root.ID &&
					result.Artifact.CollectionID == value.Registration.ID
				break
			}
			if !found {
				return fmt.Errorf(
					"%w: built-in Skill %q was installed with non-registry identity",
					basespec.ErrInvalid,
					skill.SkillDefinition.LogicalName,
				)
			}
		}
	}

	// Every Collection package publication advances the one shared Source.
	// Refresh every Collection only after all package writes have completed.
	for _, value := range i.hydrated.OrderedCollections() {
		current := byCollectionID[value.Registration.ID]
		if err := i.skills.EnsureBuiltInBundleCurrent(
			ctx,
			current.Collection.Ref(),
		); err != nil {
			return err
		}
	}
	return nil
}

func (i *Installer) Ensure(
	ctx context.Context,
) error {
	if _, err := i.EnsureBuiltInSystemRoot(ctx); err != nil {
		return err
	}
	if _, err := i.EnsureBuiltInSource(ctx); err != nil {
		return err
	}
	if _, err := i.EnsureBuiltInBundles(ctx); err != nil {
		return err
	}
	return i.EnsureBuiltInArtifacts(ctx)
}

func (i *Installer) rejectDynamicBuiltInBundles(
	ctx context.Context,
) error {
	bundles, err := i.skills.ListBundles(ctx, i.registry.Root.ID)
	if err != nil {
		return err
	}

	declared := make(
		map[basespec.LogicalName]basespec.CollectionID,
		len(i.hydrated.Collections),
	)
	for _, value := range i.hydrated.Collections {
		declared[value.Definition.LogicalName] = value.Registration.ID
	}

	for _, bundle := range bundles {
		expectedID, builtIn := declared[bundle.Data.LogicalName]
		if !builtIn || bundle.Collection.ID == expectedID {
			continue
		}
		return fmt.Errorf(
			"%w: dynamic Skill Bundle %q conflicts with canonical built-in bundle ID %q",
			basespec.ErrConflict,
			bundle.Data.LogicalName,
			expectedID,
		)
	}
	return nil
}

func (i *Installer) rejectDynamicBuiltInArtifacts(
	ctx context.Context,
	current skillbundle.Bundle,
	declaredCollection metadata.HydratedCollection,
) error {
	artifacts, err := i.skills.ListSkills(ctx, current.Collection.Ref())
	if err != nil {
		return err
	}

	declared := make(
		map[basespec.ArtifactID]struct{},
		len(declaredCollection.Artifacts),
	)
	for _, skill := range declaredCollection.Artifacts {
		declared[skill.Registration.ID] = struct{}{}
	}

	for _, value := range artifacts {
		if _, exists := declared[value.ID]; exists {
			continue
		}
		return fmt.Errorf(
			"%w: dynamic Skill Artifact %q is mixed into canonical built-in bundle %q",
			basespec.ErrConflict,
			value.ID,
			declaredCollection.Definition.LogicalName,
		)
	}
	return nil
}

func (i *Installer) ensureTopology(
	ctx context.Context,
) (topology.Installed, error) {
	ctx = protection.WithPrivilegedInstaller(ctx)
	return i.topology.EnsureProtectedTopology(
		ctx,
		topology.Declaration{
			Root: root.RootDraft{
				ID:          i.registry.Root.ID,
				DisplayName: i.registry.Root.DisplayName,
				Description: i.registry.Root.Description,
			},
			Sources: []source.Draft{{
				ID:          i.registry.Source.ID,
				Kind:        i.registry.Source.Kind,
				DisplayName: i.registry.Source.DisplayName,
				Enabled:     i.registry.Source.Enabled,
				Config:      json.RawMessage(`{}`),
			}},
		},
	)
}

func (i *Installer) packageFiles(
	ctx context.Context,
	packageRoot basespec.Locator,
) ([]source.ManagedPackageFile, error) {
	files := make([]source.ManagedPackageFile, 0)
	var totalBytes int64
	err := fs.WalkDir(i.packages, string(packageRoot), func(
		location string,
		entry fs.DirEntry,
		walkErr error,
	) error {
		if walkErr != nil {
			return walkErr
		}
		if entry == nil {
			return fmt.Errorf(
				"%w: built-in package walk returned no entry for %q",
				basespec.ErrInvalid,
				location,
			)
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf(
				"%w: built-in package file %q is not regular",
				basespec.ErrInvalid,
				location,
			)
		}
		if len(files) >= basespec.MaxDiscoveryEntries {
			return fmt.Errorf(
				"%w: built-in package exceeds the file count limit",
				basespec.ErrInvalid,
			)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Size() < 0 ||
			info.Size() > basespec.MaxScanBytes-totalBytes {
			return fmt.Errorf(
				"%w: built-in package exceeds the byte limit",
				basespec.ErrInvalid,
			)
		}
		relative, found := strings.CutPrefix(
			location,
			string(packageRoot)+"/",
		)
		if !found || relative == "" {
			return fmt.Errorf(
				"%w: invalid built-in package file %q",
				basespec.ErrInvalid,
				location,
			)
		}
		if err := basespec.ValidatePortableLocator(basespec.Locator(relative), false); err != nil {
			return err
		}
		content, err := fs.ReadFile(i.packages, location)
		if err != nil {
			return err
		}
		if int64(len(content)) != info.Size() {
			return fmt.Errorf(
				"%w: built-in package file %q changed while being read",
				basespec.ErrConflict,
				location,
			)
		}
		totalBytes += int64(len(content))
		files = append(files, source.ManagedPackageFile{
			Locator: basespec.Locator(relative),
			Content: append([]byte(nil), content...),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	normalized, err := source.NormalizeManagedPackagePublication(
		source.ManagedPackagePublication{
			Directory: packageRoot,
			Files:     files,
		},
	)
	if err != nil {
		return nil, err
	}
	return normalized.Files, nil
}
