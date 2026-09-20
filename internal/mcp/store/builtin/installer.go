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
	mcpConsumerAPI "github.com/flexigpt/flexigpt-app/internal/mcp/store/consumerapi"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain"
	mcpOverlay "github.com/flexigpt/flexigpt-app/internal/mcp/store/overlay"
)

type InstallerDependencies struct {
	MCP      mcpConsumerAPI.BuiltinStore
	Packages fs.FS
	Overlays mcpOverlay.RootPurger
}

type Installer struct {
	mcp           mcpConsumerAPI.BuiltinStore
	rootID        root.RootID
	sourceID      source.SourceID
	overlays      mcpOverlay.RootPurger
	prepared      []PreparedPackage
	fingerprint   cryptoutil.Digest
	packageScopes []basespec.Locator
}

func NewInstaller(
	dependencies InstallerDependencies,
) (*Installer, error) {
	if dependencies.MCP == nil ||
		dependencies.Packages == nil ||
		dependencies.Overlays == nil {
		return nil, fmt.Errorf(
			"%w: MCP built-in installer dependencies are incomplete",
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
			"%w: built-in MCP Source must be managed",
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
	fingerprint, err := topology.HydrationFingerprint(
		mcpDomain.HydrationSchemaVersion,
		topologyValue,
	)
	if err != nil {
		return nil, err
	}
	scopes, err := packageScopes(prepared)
	if err != nil {
		return nil, err
	}
	return &Installer{
		mcp:           dependencies.MCP,
		rootID:        topologyValue.Root.ID,
		sourceID:      builtinSource.ID,
		overlays:      dependencies.Overlays,
		prepared:      prepared,
		fingerprint:   fingerprint,
		packageScopes: scopes,
	}, nil
}

func (*Installer) BuiltInName() string {
	return mcpDomain.BuiltInInstallerName
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
	if ctx == nil {
		return topology.Hydration{}, fmt.Errorf(
			"%w: built-in MCP hydration context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return topology.Hydration{}, err
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

func (i *Installer) EnsurePackageHydration(
	ctx context.Context,
	topologyCurrent bool,
	stale []topology.PackageHydration,
) error {
	if i == nil {
		return basespec.ErrClosed
	}
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return err
	}
	if !topologyCurrent {
		if err := i.overlays.PurgeRoot(
			ctx,
			i.rootID,
		); err != nil {
			return err
		}
	}
	for _, value := range stale {
		address, err := source.ParseManagedPackageAddressDirectory(value.Key.Scope)
		if err != nil {
			return err
		}
		if address.Kind != mcpDomain.MCPCollectionPackageKind {
			continue
		}
		if err := i.mcp.RemoveBuiltInPackage(
			ctx,
			value.RootID,
			value.SourceID,
			address,
		); err != nil {
			return fmt.Errorf(
				"remove stale built-in MCP package %q: %w",
				value.Key.Scope,
				err,
			)
		}
	}
	return i.ensurePackages(ctx)
}

func (i *Installer) EnsureHydration(
	ctx context.Context,
	topologyCurrent bool,
) error {
	if i == nil {
		return basespec.ErrClosed
	}
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return err
	}
	if !topologyCurrent {
		if err := i.overlays.PurgeRoot(
			ctx,
			i.rootID,
		); err != nil {
			return err
		}
	}
	return i.ensurePackages(ctx)
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
	if err := i.ensurePackages(ctx); err != nil {
		return err
	}
	return i.FinalizeHydration(ctx)
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
	return i.mcp.EnsureBuiltInSourceCurrent(
		ctx,
		i.rootID,
		i.sourceID,
	)
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
		scope, err := value.PackageAddress.Directory()
		if err != nil {
			return nil, err
		}
		digest, err := PackageFingerprint(value)
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
			Fingerprint: digest,
		})
	}
	return topology.NormalizePackageHydrations(output)
}

func (i *Installer) ensurePackages(
	ctx context.Context,
) error {
	for _, value := range i.prepared {
		if _, err := i.mcp.InstallBuiltInPackage(
			ctx,
			mcpConsumerAPI.BuiltInPackageInstallRequest{
				RootID:         i.rootID,
				SourceID:       i.sourceID,
				PackageAddress: value.PackageAddress,
				DocumentFile:   value.DocumentFile,
				PackageFiles:   value.PackageFiles,
				Expectations:   value.Expectations,
			},
		); err != nil {
			return fmt.Errorf(
				"install built-in MCP package %q: %w",
				value.EmbeddedPackageRoot,
				err,
			)
		}
	}
	return nil
}

// packageScopes returns the package roots owned by the MCP built-in installer.
func packageScopes(
	prepared []PreparedPackage,
) ([]basespec.Locator, error) {
	output := make([]basespec.Locator, 0, len(prepared))
	for _, value := range prepared {
		scope, err := value.PackageAddress.Directory()
		if err != nil {
			return nil, err
		}
		output = append(output, scope)
	}
	slices.Sort(output)
	return output, nil
}
