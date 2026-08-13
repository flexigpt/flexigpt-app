package schemaadapter

import (
	"context"
	"fmt"
	"io/fs"
	"slices"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/protection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/shareable"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/topology"
	"github.com/flexigpt/flexigpt-app/internal/skill/bundle"
)

type skillInstaller interface {
	ListBundles(
		ctx context.Context,
		rootID basespec.RootID,
	) ([]bundle.Bundle, error)
	ListSkills(
		ctx context.Context,
		ref collection.CollectionRef,
	) ([]artifact.Artifact, error)
	EnsureBuiltInBundleTopology(
		ctx context.Context,
		t bundle.BuiltInBundleTopology,
	) (bundle.Bundle, error)
	InstallBuiltInCollection(
		ctx context.Context,
		c bundle.BuiltInCollectionInstallRequest,
	) ([]bundle.CreateManagedSkillResponse, error)
	EnsureBuiltInBundleCurrent(
		ctx context.Context,
		ref collection.CollectionRef,
	) error
}

type InstallerDependencies struct {
	Skills                 skillInstaller
	SkillRegistry          Registry
	Packages               fs.FS
	ShareableCanonicalizer shareable.ExpectedCanonicalizer
}

type Installer struct {
	skills          skillInstaller
	builtInTopology topology.Declaration
	hydrated        HydratedRegistry
	packages        fs.FS
	packageScopes   []basespec.Locator
}

func NewInstaller(
	dependencies InstallerDependencies,
) (*Installer, error) {
	if dependencies.Skills == nil ||
		dependencies.Packages == nil ||
		dependencies.ShareableCanonicalizer == nil {
		return nil, fmt.Errorf("%w: built-in installer dependencies are incomplete", basespec.ErrInvalid)
	}
	builtInTopology := artifactbuiltin.BuiltinTopologyDeclaration()
	if err := builtInTopology.Validate(); err != nil {
		return nil, err
	}
	if err := dependencies.SkillRegistry.Validate(); err != nil {
		return nil, err
	}
	if len(builtInTopology.Sources) != 1 ||
		builtInTopology.Sources[0].Kind != artifactbuiltin.ManagedDirectorySourceKind {
		return nil, fmt.Errorf(
			"%w: built-in Source kind must be %q",
			basespec.ErrInvalid,
			artifactbuiltin.ManagedDirectorySourceKind,
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
	packageScopes, err := builtInPackageScopes(hydrated)
	if err != nil {
		return nil, err
	}
	return &Installer{
		skills:          dependencies.Skills,
		builtInTopology: builtInTopology,
		hydrated:        hydrated,
		packages:        dependencies.Packages,
		packageScopes:   packageScopes,
	}, nil
}

func (*Installer) BuiltInName() string {
	return artifactbuiltin.AgentSkillBuiltInInstallerName
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
	return append([]basespec.Locator(nil), i.packageScopes...)
}

// builtInPackageScopes returns managed-source package roots, not embedded
// source-tree locations. Bootstrap collision checks must use the locations
// that installers actually publish into the shared managed Source.
func builtInPackageScopes(
	registry HydratedRegistry,
) ([]basespec.Locator, error) {
	output := make([]basespec.Locator, 0, len(registry.Collections))
	for _, value := range registry.OrderedCollections() {
		address, err := bundle.BuiltInCollectionPackageAddress(
			basespec.LogicalName(value.Definition.LogicalName),
			basespec.LogicalVersion(value.Definition.LogicalVersion),
		)
		if err != nil {
			return nil, err
		}
		scope, err := address.Directory()
		if err != nil {
			return nil, err
		}
		output = append(output, scope)
	}
	slices.Sort(output)
	return output, nil
}

func (i *Installer) Ensure(
	ctx context.Context,
) error {
	if err := protection.RequirePrivilegedInstaller(ctx); err != nil {
		return err
	}
	return i.EnsureBuiltInArtifacts(ctx)
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
	byCollectionID := make(map[basespec.CollectionID]bundle.Bundle, len(bundles))
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
		packageAddress, err := bundle.BuiltInCollectionPackageAddress(
			basespec.LogicalName(value.Definition.LogicalName),
			basespec.LogicalVersion(value.Definition.LogicalVersion),
		)
		if err != nil {
			return err
		}
		files, err := i.packageFiles(
			ctx,
			value.EmbeddedPackageRoot,
			value.Definition,
			packageAddress,
		)
		if err != nil {
			return err
		}

		request := bundle.BuiltInCollectionInstallRequest{
			Bundle:                     current.Collection.Ref(),
			ExpectedCollectionRevision: current.Collection.Revision,
			PackageAddress:             packageAddress,
			PackageFiles:               files,
			Skills: make(
				[]bundle.BuiltInCollectionSkill,
				0,
				len(value.Artifacts),
			),
		}
		for _, skill := range value.Artifacts {
			request.Skills = append(request.Skills, bundle.BuiltInCollectionSkill{
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

func (i *Installer) EnsureBuiltInBundles(
	ctx context.Context,
) ([]bundle.Bundle, error) {
	if err := protection.RequirePrivilegedInstaller(ctx); err != nil {
		return nil, err
	}
	if err := i.rejectDynamicBuiltInBundles(ctx); err != nil {
		return nil, err
	}

	output := make([]bundle.Bundle, 0, len(i.hydrated.Collections))
	for _, value := range i.hydrated.OrderedCollections() {
		if value.Definition.Digest == nil {
			return nil, fmt.Errorf(
				"%w: hydrated built-in collection has no digest",
				basespec.ErrInvalid,
			)
		}
		packageAddress, err := bundle.BuiltInCollectionPackageAddress(
			basespec.LogicalName(value.Definition.LogicalName),
			basespec.LogicalVersion(value.Definition.LogicalVersion),
		)
		if err != nil {
			return nil, err
		}
		discoveryRoot, err := packageAddress.Directory()
		if err != nil {
			return nil, err
		}
		b, err := i.skills.EnsureBuiltInBundleTopology(
			ctx,
			bundle.BuiltInBundleTopology{
				RootID:                i.builtInTopology.Root.ID,
				CollectionID:          value.Registration.ID,
				SourceID:              i.builtInTopology.Sources[0].ID,
				LogicalName:           basespec.LogicalName(value.Definition.LogicalName),
				LogicalVersion:        basespec.LogicalVersion(value.Definition.LogicalVersion),
				DisplayName:           value.Definition.DisplayName,
				Description:           value.Definition.Description,
				Labels:                value.Definition.Labels,
				Enabled:               value.Registration.Enabled,
				DiscoveryRoot:         discoveryRoot,
				ExpectedMemberDigests: value.ExpectedMemberDigests,
			},
		)
		if err != nil {
			return nil, err
		}
		output = append(output, b)
	}
	return output, nil
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
	current bundle.Bundle,
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
	document artifactbuiltin.SkillCollectionV1,
	address source.ManagedPackageAddress,
) ([]source.ManagedPackageFile, error) {
	embeddedFiles, err := topology.ReadPackageFiles(
		ctx,
		i.packages,
		packageRoot,
	)
	if err != nil {
		return nil, err
	}

	canonicalDocument, err := artifactbuiltin.MarshalSkillCollectionV1(document)
	if err != nil {
		return nil, err
	}

	files := make([]source.ManagedPackageFile, 0, len(embeddedFiles))
	foundDocument := false
	for _, file := range embeddedFiles {
		content := append([]byte(nil), file.Content...)
		if file.Locator == artifactbuiltin.SkillCollectionFileName {
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
			artifactbuiltin.SkillCollectionFileName,
		)
	}

	publication, err := source.NormalizeManagedPackagePublication(
		source.ManagedPackagePublication{
			Address: address,
			Files:   files,
		},
	)
	if err != nil {
		return nil, err
	}
	return publication.Files, nil
}
