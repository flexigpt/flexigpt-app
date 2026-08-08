package artifactbuiltin

import (
	"context"
	"fmt"
	"io/fs"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/protection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/shareable"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source/managed"
	"github.com/flexigpt/flexigpt-app/internal/builtin"
	builtinSchema "github.com/flexigpt/flexigpt-app/internal/builtin/schema"

	skillBundle "github.com/flexigpt/flexigpt-app/internal/skill/bundle"
)

type skillInstaller interface {
	ListBundles(
		ctx context.Context,
		rootID basespec.RootID,
	) ([]skillBundle.Bundle, error)
	ListSkills(
		ctx context.Context,
		ref collection.CollectionRef,
	) ([]artifact.Artifact, error)
	EnsureBuiltInBundleTopology(
		ctx context.Context,
		t skillBundle.BuiltInBundleTopology,
	) (skillBundle.Bundle, error)
	InstallBuiltInCollection(
		ctx context.Context,
		c skillBundle.BuiltInCollectionInstallRequest,
	) ([]skillBundle.CreateManagedSkillResponse, error)
	EnsureBuiltInBundleCurrent(
		ctx context.Context,
		ref collection.CollectionRef,
	) error
}

type InstallerDependencies struct {
	Skills                 skillInstaller
	BuiltInTopology        builtin.Registry
	SkillRegistry          Registry
	Packages               fs.FS
	ShareableCanonicalizer shareable.Canonicalizer
}

type Installer struct {
	skills          skillInstaller
	builtInTopology builtin.Registry
	hydrated        HydratedRegistry
	packages        fs.FS
}

func NewInstaller(
	dependencies InstallerDependencies,
) (*Installer, error) {
	if dependencies.Skills == nil ||
		dependencies.Packages == nil ||
		dependencies.ShareableCanonicalizer == nil {
		return nil, fmt.Errorf("%w: built-in installer dependencies are incomplete", basespec.ErrInvalid)
	}
	if err := dependencies.BuiltInTopology.Validate(); err != nil {
		return nil, err
	}
	if err := dependencies.SkillRegistry.Validate(); err != nil {
		return nil, err
	}
	if dependencies.BuiltInTopology.Source.Kind != managed.Kind {
		return nil, fmt.Errorf(
			"%w: built-in Source kind must be %q",
			basespec.ErrInvalid,
			managed.Kind,
		)
	}
	hydrated, err := dependencies.SkillRegistry.Hydrate(
		context.Background(),
		dependencies.ShareableCanonicalizer,
		dependencies.Packages,
	)
	if err != nil {
		return nil, err
	}
	return &Installer{
		skills:          dependencies.Skills,
		builtInTopology: dependencies.BuiltInTopology,
		hydrated:        hydrated,
		packages:        dependencies.Packages,
	}, nil
}

func (*Installer) BuiltInName() string {
	return "agent.skill"
}

func (i *Installer) BuiltInIDs() []string {
	if i == nil {
		return nil
	}
	count := 0
	for _, value := range i.hydrated.Collections {
		count += 1 + len(value.Artifacts)
	}
	output := make([]string, 0, count)
	for _, value := range i.hydrated.OrderedCollections() {
		output = append(output, string(value.Registration.ID))
		for _, artifact := range value.Artifacts {
			output = append(output, string(artifact.Registration.ID))
		}
	}
	return output
}

func (i *Installer) BuiltInPackageScopes() []basespec.Locator {
	if i == nil {
		return nil
	}
	output := make([]basespec.Locator, 0, len(i.hydrated.Collections))
	for _, value := range i.hydrated.OrderedCollections() {
		output = append(output, value.SourceScope)
	}
	return output
}

func (i *Installer) EnsureBuiltInBundles(
	ctx context.Context,
) ([]skillBundle.Bundle, error) {
	if err := protection.RequirePrivilegedInstaller(ctx); err != nil {
		return nil, err
	}
	if err := i.rejectDynamicBuiltInBundles(ctx); err != nil {
		return nil, err
	}

	output := make([]skillBundle.Bundle, 0, len(i.hydrated.Collections))
	for _, value := range i.hydrated.OrderedCollections() {
		if value.Definition.Digest == nil {
			return nil, fmt.Errorf(
				"%w: hydrated built-in collection has no digest",
				basespec.ErrInvalid,
			)
		}
		portableDefinition := value.Definition.Clone()
		bundle, err := i.skills.EnsureBuiltInBundleTopology(
			ctx,
			skillBundle.BuiltInBundleTopology{
				RootID:                i.builtInTopology.Root.ID,
				CollectionID:          value.Registration.ID,
				SourceID:              i.builtInTopology.Source.ID,
				LogicalName:           basespec.LogicalName(value.Definition.LogicalName),
				LogicalVersion:        basespec.LogicalVersion(value.Definition.LogicalVersion),
				DisplayName:           value.Definition.DisplayName,
				Description:           value.Definition.Description,
				Labels:                value.Definition.Labels,
				Enabled:               value.Registration.Enabled,
				DiscoveryRoot:         value.SourceScope,
				ExpectedMemberDigests: value.ExpectedMemberDigests,
				PortableDefinition:    &portableDefinition,
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
	if err := protection.RequirePrivilegedInstaller(ctx); err != nil {
		return err
	}
	bundles, err := i.EnsureBuiltInBundles(ctx)
	if err != nil {
		return err
	}
	byCollectionID := make(map[basespec.CollectionID]skillBundle.Bundle, len(bundles))
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
		if !current.Collection.Enabled {
			continue
		}

		files, err := i.packageFiles(
			ctx,
			value.SourceScope,
			value.Definition,
		)
		if err != nil {
			return err
		}

		request := skillBundle.BuiltInCollectionInstallRequest{
			Bundle:                     current.Collection.Ref(),
			ExpectedCollectionRevision: current.Collection.Revision,
			PackageDirectory:           value.SourceScope,
			PackageFiles:               files,
			Skills: make(
				[]skillBundle.BuiltInCollectionSkill,
				0,
				len(value.Artifacts),
			),
		}
		for _, skill := range value.Artifacts {
			request.Skills = append(request.Skills, skillBundle.BuiltInCollectionSkill{
				ArtifactID: skill.Registration.ID,
				Member:     basespec.Locator(skill.Member.Locator),
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
				found = result.Artifact.RootID == i.builtInTopology.Root.ID &&
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
		if !current.Collection.Enabled {
			continue
		}
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
	if err := protection.RequirePrivilegedInstaller(ctx); err != nil {
		return err
	}
	return i.EnsureBuiltInArtifacts(ctx)
}

func (i *Installer) ensureBuiltInCatalogsCurrent(
	ctx context.Context,
) error {
	for _, value := range i.hydrated.OrderedCollections() {
		if !value.Registration.Enabled {
			continue
		}
		ref := collection.CollectionRef{
			RootID:       i.builtInTopology.Root.ID,
			CollectionID: value.Registration.ID,
		}
		if err := i.skills.EnsureBuiltInBundleCurrent(
			ctx,
			ref,
		); err != nil {
			return fmt.Errorf(
				"ensure current built-in Collection %q: %w",
				value.Registration.ID,
				err,
			)
		}
	}
	return nil
}

func (i *Installer) rejectDynamicBuiltInBundles(
	ctx context.Context,
) error {
	bundles, err := i.skills.ListBundles(ctx, i.builtInTopology.Root.ID)
	if err != nil {
		return err
	}

	declared := make(
		map[basespec.LogicalName]basespec.CollectionID,
		len(i.hydrated.Collections),
	)
	for _, value := range i.hydrated.Collections {
		declared[basespec.LogicalName(value.Definition.LogicalName)] = value.Registration.ID
	}

	for _, bundle := range bundles {
		expectedID, builtIn := declared[bundle.Data.LogicalName]
		if !builtIn {
			return fmt.Errorf(
				"%w: undeclared built-in Skill Bundle %q remains installed",
				basespec.ErrConflict,
				bundle.Data.LogicalName,
			)
		}
		if bundle.Collection.ID == expectedID {
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
	current skillBundle.Bundle,
	declaredCollection HydratedCollection,
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

func (i *Installer) packageFiles(
	ctx context.Context,
	packageRoot basespec.Locator,
	document builtinSchema.SkillCollectionV1,
) ([]source.ManagedPackageFile, error) {
	embeddedFiles, err := builtin.ReadPackageFiles(
		ctx,
		i.packages,
		packageRoot,
	)
	if err != nil {
		return nil, err
	}

	canonicalDocument, err := builtinSchema.MarshalSkillCollectionV1(document)
	if err != nil {
		return nil, err
	}

	files := make([]source.ManagedPackageFile, 0, len(embeddedFiles))
	foundDocument := false
	for _, file := range embeddedFiles {
		content := append([]byte(nil), file.Content...)
		if file.Locator == builtinSchema.SkillCollectionV1FileName {
			content = append([]byte(nil), canonicalDocument...)
			foundDocument = true
		}
		files = append(files, source.ManagedPackageFile{
			Locator: file.Locator,
			Content: content,
		})
	}
	if !foundDocument {
		return nil, fmt.Errorf(
			"%w: built-in package lacks %q",
			basespec.ErrInvalid,
			builtinSchema.SkillCollectionV1FileName,
		)
	}

	publication, err := source.NormalizeManagedPackagePublication(
		source.ManagedPackagePublication{
			Directory: packageRoot,
			Files:     files,
		},
	)
	if err != nil {
		return nil, err
	}
	return publication.Files, nil
}
