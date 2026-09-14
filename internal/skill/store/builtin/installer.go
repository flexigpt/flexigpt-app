package builtin

import (
	"context"
	"fmt"
	"io/fs"
	"slices"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi/topology"
	skillConsumerAPI "github.com/flexigpt/flexigpt-app/internal/skill/store/consumerapi"
	skillDomain "github.com/flexigpt/flexigpt-app/internal/skill/store/domain"
)

type InstallerDependencies struct {
	Skills        skillConsumerAPI.BuiltinStore
	SkillRegistry Registry
	Packages      fs.FS
}

type Installer struct {
	skills          skillConsumerAPI.BuiltinStore
	builtInTopology topology.Declaration
	hydrated        HydratedRegistry
	packageScopes   []basespec.Locator
}

func NewInstaller(
	dependencies InstallerDependencies,
) (*Installer, error) {
	if dependencies.Skills == nil || dependencies.Packages == nil {
		return nil, fmt.Errorf(
			"%w: built-in Skill installer dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	if err := dependencies.SkillRegistry.Validate(); err != nil {
		return nil, err
	}

	topologyValue := artifactbuiltin.BuiltinTopologyDeclaration()
	if err := topologyValue.Validate(); err != nil {
		return nil, err
	}
	if len(topologyValue.Sources) != 1 ||
		topologyValue.Sources[0].Kind != source.SourceKindManagedDirectory {
		return nil, fmt.Errorf(
			"%w: built-in Skill Source must be managed",
			basespec.ErrInvalid,
		)
	}
	hydrated, err := dependencies.SkillRegistry.Hydrate(
		context.Background(),
		dependencies.Packages,
	)
	if err != nil {
		return nil, err
	}
	scopes, err := builtInPackageScopes(hydrated)
	if err != nil {
		return nil, err
	}

	return &Installer{
		skills:          dependencies.Skills,
		builtInTopology: topologyValue,
		hydrated:        hydrated,
		packageScopes:   scopes,
	}, nil
}

func (*Installer) BuiltInName() string {
	return skillDomain.BuiltInInstallerName
}

// BuiltInIDs - Artifact IDs are Store-owned source synchronization identities. The Skill
// installer has no static Artifact IDs to reserve in BootstrapRegistry.
func (*Installer) BuiltInIDs() []string {
	return nil
}

func (i *Installer) BuiltInPackageScopes() []basespec.Locator {
	if i == nil {
		return nil
	}
	return append([]basespec.Locator(nil), i.packageScopes...)
}

func (i *Installer) Ensure(ctx context.Context) error {
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return err
	}
	return i.EnsureBuiltInArtifacts(ctx)
}

func (i *Installer) EnsureBuiltInArtifacts(
	ctx context.Context,
) error {
	if i == nil {
		return basespec.ErrClosed
	}
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return err
	}

	for _, value := range i.hydrated.OrderedSkills() {
		record, err := i.skills.InstallBuiltInSkill(
			ctx,
			skillConsumerAPI.BuiltInSkillInstallRequest{
				RootID:              i.builtInTopology.Root.ID,
				SourceID:            i.builtInTopology.Sources[0].ID,
				PackageAddress:      value.PackageAddress,
				PackageFiles:        value.PackageFiles,
				ExpectedLogicalName: value.Definition.LogicalName,
				ExpectedDefinition:  value.Definition.Digest,
				Enabled:             value.Registration.Enabled,
			},
		)
		if err != nil {
			return fmt.Errorf(
				"install built-in Skill %q: %w",
				value.Definition.LogicalName,
				err,
			)
		}
		if err := verifyBuiltInArtifact(record, value); err != nil {
			return err
		}
	}

	members := make(
		[]basespec.LogicalName,
		0,
		len(i.hydrated.Skills),
	)
	for _, value := range i.hydrated.OrderedSkills() {
		members = append(members, value.Definition.LogicalName)
	}
	collection, err := i.skills.InstallBuiltInSkillCollection(
		ctx,
		skillConsumerAPI.BuiltInSkillCollectionInstallRequest{
			RootID:      i.builtInTopology.Root.ID,
			SourceID:    i.builtInTopology.Sources[0].ID,
			Name:        skillDomain.BuiltinSkillCollectionName,
			Description: skillDomain.BuiltinSkillCollectionDescription,
			Members:     members,
			Enabled:     true,
		},
	)
	if err != nil {
		return fmt.Errorf("install built-in Skill Collection: %w", err)
	}
	if collection.State != artifact.StateAvailable {
		return fmt.Errorf("%w: built-in Skill Collection is unavailable", basespec.ErrReferenceUnresolved)
	}

	return i.FinalizeHydration(ctx)
}

func verifyBuiltInArtifact(
	record artifact.Artifact,
	expected HydratedSkill,
) error {
	if record.Kind != skillDomain.SkillArtifactKind ||
		record.LogicalName != expected.Definition.LogicalName ||
		record.State != artifact.StateAvailable ||
		record.ResolvedDefinition == nil ||
		*record.ResolvedDefinition != expected.Definition.Digest ||
		record.Enabled != expected.Registration.Enabled {
		return fmt.Errorf(
			"%w: built-in Skill Artifact %q does not match embedded registry",
			basespec.ErrReferenceUnresolved,
			record.ID,
		)
	}
	return nil
}

func builtInPackageScopes(
	registry HydratedRegistry,
) ([]basespec.Locator, error) {
	output := make([]basespec.Locator, 0, len(registry.Skills))
	for _, value := range registry.OrderedSkills() {
		directory, err := value.PackageAddress.Directory()
		if err != nil {
			return nil, err
		}
		output = append(output, directory)
	}
	collectionAddress, err := source.NewManagedPackageAddress(
		skillDomain.BuiltinCollectionPackageKind,
		skillDomain.BuiltinSkillCollectionName,
		artifactbuiltin.UnversionedPackageVersion,
	)
	if err != nil {
		return nil, err
	}
	collectionScope, err := collectionAddress.Directory()
	if err != nil {
		return nil, err
	}
	output = append(output, collectionScope)
	slices.Sort(output)
	return output, nil
}
