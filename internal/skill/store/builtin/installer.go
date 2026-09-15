package builtin

import (
	"context"
	"fmt"
	"io/fs"
	"slices"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi/topology"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	skillConsumerAPI "github.com/flexigpt/flexigpt-app/internal/skill/store/consumerapi"
	skillDomain "github.com/flexigpt/flexigpt-app/internal/skill/store/domain"
)

type InstallerDependencies struct {
	Skills   skillConsumerAPI.BuiltinStore
	Packages fs.FS
}

type Installer struct {
	skills          skillConsumerAPI.BuiltinStore
	builtInTopology topology.Declaration
	prepared        []PreparedPackage
	packageScopes   []basespec.Locator
	fingerprint     cryptoutil.Digest
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

	topologyValue := builtin.BuiltinTopologyDeclaration()
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

	prepared, err := PreparePackages(
		context.Background(),
		dependencies.Packages,
	)
	if err != nil {
		return nil, err
	}
	scopes, err := builtInPackageScopes(prepared)
	if err != nil {
		return nil, err
	}
	fingerprint, err := hydrationFingerprint(
		topologyValue,
		prepared,
	)
	if err != nil {
		return nil, err
	}

	return &Installer{
		skills:          dependencies.Skills,
		builtInTopology: topologyValue,
		prepared:        prepared,
		packageScopes:   scopes,
		fingerprint:     fingerprint,
	}, nil
}

func (*Installer) BuiltInName() string {
	return skillDomain.BuiltInInstallerName
}

func (i *Installer) BuiltInPackageScopes() []basespec.Locator {
	if i == nil {
		return nil
	}
	return append([]basespec.Locator(nil), i.packageScopes...)
}

func (i *Installer) Ensure(ctx context.Context) error {
	if i == nil {
		return basespec.ErrClosed
	}
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return err
	}
	if err := i.EnsureBuiltInArtifacts(ctx); err != nil {
		return err
	}
	return i.FinalizeHydration(ctx)
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

	for _, value := range i.prepared {
		if _, err := i.skills.InstallBuiltInSkillPackage(
			ctx,
			skillConsumerAPI.BuiltInSkillPackageInstallRequest{
				RootID:         i.builtInTopology.Root.ID,
				SourceID:       i.builtInTopology.Sources[0].ID,
				PackageAddress: value.PackageAddress,
				DocumentFile:   value.DocumentFile,
				PackageFiles:   value.PackageFiles,
				Expectations:   value.Expectations,
			},
		); err != nil {
			return fmt.Errorf(
				"install built-in Skill package %q: %w",
				value.EmbeddedPackageRoot,
				err,
			)
		}
	}

	return nil
}

func builtInPackageScopes(
	prepared []PreparedPackage,
) ([]basespec.Locator, error) {
	output := make([]basespec.Locator, 0, len(prepared))
	for _, value := range prepared {
		directory, err := value.PackageAddress.Directory()
		if err != nil {
			return nil, err
		}
		output = append(output, directory)
	}
	slices.Sort(output)
	return output, nil
}
