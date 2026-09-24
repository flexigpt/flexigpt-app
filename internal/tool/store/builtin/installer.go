package builtin

import (
	"context"
	"fmt"
	"io/fs"
	"slices"

	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi/topology"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	toolConsumerAPI "github.com/flexigpt/flexigpt-app/internal/tool/store/consumerapi"
	toolDomain "github.com/flexigpt/flexigpt-app/internal/tool/store/domain"
)

type InstallerDependencies struct {
	Tools    toolConsumerAPI.BuiltinStore
	Packages fs.FS
	GoTools  toolDomain.GoToolLocator
}

type Installer struct {
	tools         toolConsumerAPI.BuiltinStore
	rootID        root.RootID
	sourceID      source.SourceID
	prepared      []PreparedPackage
	packageScopes []basespec.Locator
	fingerprint   cryptoutil.Digest
}

func NewInstaller(
	dependencies InstallerDependencies,
) (*Installer, error) {
	if dependencies.Tools == nil ||
		dependencies.Packages == nil ||
		dependencies.GoTools == nil {
		return nil, fmt.Errorf(
			"%w: Tool built-in installer dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}

	topologyValue := documentTopology.BuiltinTopologyDeclaration()
	if err := topologyValue.Validate(); err != nil {
		return nil, err
	}
	builtinSource, err := documentTopology.BuiltinSource(
		documentTopology.BuiltinSourceRolePackages,
	)
	if err != nil {
		return nil, err
	}
	if builtinSource.Kind != source.SourceKindManagedDirectory {
		return nil, fmt.Errorf(
			"%w: Tool built-in Source must be managed",
			basespec.ErrInvalid,
		)
	}

	prepared, err := PreparePackages(
		context.Background(),
		dependencies.Packages,
		dependencies.GoTools,
	)
	if err != nil {
		return nil, err
	}

	scopes := make([]basespec.Locator, 0, len(prepared))
	for _, value := range prepared {
		scope, err := value.Address.Directory()
		if err != nil {
			return nil, err
		}
		scopes = append(scopes, scope)
	}
	slices.Sort(scopes)

	fingerprint, err := topology.HydrationFingerprint(
		toolDomain.HydrationSchemaVersion,
		topologyValue,
	)
	if err != nil {
		return nil, err
	}

	return &Installer{
		tools:         dependencies.Tools,
		rootID:        topologyValue.Root.ID,
		sourceID:      builtinSource.ID,
		prepared:      prepared,
		packageScopes: scopes,
		fingerprint:   fingerprint,
	}, nil
}

func (*Installer) BuiltInName() string {
	return toolDomain.BuiltInInstallerName
}

func (i *Installer) BuiltInPackageScopes() []basespec.Locator {
	if i == nil {
		return nil
	}
	return append([]basespec.Locator(nil), i.packageScopes...)
}

func (i *Installer) DesiredHydration(
	ctx context.Context,
) (topology.Hydration, error) {
	if i == nil {
		return topology.Hydration{}, basespec.ErrClosed
	}
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return topology.Hydration{}, err
	}

	value := topology.Hydration{
		InstallerName: i.BuiltInName(),
		RootID:        i.rootID,
		SourceID:      i.sourceID,
		Fingerprint:   i.fingerprint,
	}
	if err := value.Validate(); err != nil {
		return topology.Hydration{}, err
	}
	return value, nil
}

func (i *Installer) DesiredPackageHydrations(
	ctx context.Context,
) ([]topology.PackageHydration, error) {
	if i == nil {
		return nil, basespec.ErrClosed
	}
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return nil, err
	}

	output := make([]topology.PackageHydration, 0, len(i.prepared))
	for _, value := range i.prepared {
		scope, err := value.Address.Directory()
		if err != nil {
			return nil, err
		}
		fingerprint, err := value.Fingerprint()
		if err != nil {
			return nil, err
		}
		output = append(output, topology.PackageHydration{
			Key: topology.PackageHydrationKey{
				InstallerName: i.BuiltInName(),
				Scope:         scope,
			},
			RootID:      i.rootID,
			SourceID:    i.sourceID,
			Fingerprint: fingerprint,
		})
	}
	return topology.NormalizePackageHydrations(output)
}

func (i *Installer) Ensure(
	ctx context.Context,
) error {
	if i == nil {
		return basespec.ErrClosed
	}
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return err
	}
	if err := i.EnsureHydration(ctx, false); err != nil {
		return err
	}
	return i.FinalizeHydration(ctx)
}

func (i *Installer) EnsureHydration(
	ctx context.Context,
	_ bool,
) error {
	if i == nil {
		return basespec.ErrClosed
	}
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return err
	}
	for _, value := range i.prepared {
		if err := i.installPreparedPackage(ctx, value); err != nil {
			return err
		}
	}
	return nil
}

func (i *Installer) EnsurePackageHydration(
	ctx context.Context,
	_ bool,
	stale []topology.PackageHydration,
) error {
	if i == nil {
		return basespec.ErrClosed
	}
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return err
	}

	for _, value := range stale {
		address, err := source.ParseManagedPackageAddressDirectory(
			value.Key.Scope,
		)
		if err != nil {
			return err
		}
		if address.Kind != toolDomain.ToolPackageKind &&
			address.Kind != toolDomain.ToolCollectionPackageKind {
			continue
		}
		if err := i.tools.RemoveBuiltInPackage(
			ctx,
			value.RootID,
			value.SourceID,
			address,
		); err != nil {
			return fmt.Errorf(
				"remove stale Tool package %q: %w",
				value.Key.Scope,
				err,
			)
		}
	}

	for _, value := range i.prepared {
		if err := i.installPreparedPackage(ctx, value); err != nil {
			return err
		}
	}
	return nil
}

func (i *Installer) FinalizeHydration(
	ctx context.Context,
) error {
	if i == nil {
		return basespec.ErrClosed
	}
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return err
	}
	return i.tools.EnsureBuiltInSourceCurrent(
		ctx,
		i.rootID,
		i.sourceID,
	)
}

func (i *Installer) installPreparedPackage(
	ctx context.Context,
	value PreparedPackage,
) error {
	return i.tools.InstallBuiltInPackage(
		ctx,
		toolConsumerAPI.BuiltInPackageInstallRequest{
			RootID:              i.rootID,
			SourceID:            i.sourceID,
			Package:             value.Address,
			DocumentFile:        value.DocumentFile,
			PackageFiles:        value.PackageFiles,
			ExpectedKind:        value.ExpectedKind,
			ExpectedLogicalName: value.ExpectedLogicalName,
			ExpectedDefinition:  value.ExpectedDefinition,
		},
	)
}
