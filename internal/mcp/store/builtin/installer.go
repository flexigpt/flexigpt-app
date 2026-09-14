package builtin

import (
	"context"
	"fmt"
	"io/fs"
	"slices"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi/topology"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	mcpConsumerAPI "github.com/flexigpt/flexigpt-app/internal/mcp/store/consumerapi"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain"
	mcpOverlay "github.com/flexigpt/flexigpt-app/internal/mcp/store/overlay"
)

type InstallerDependencies struct {
	MCP      mcpConsumerAPI.BuiltinStore
	Registry Registry
	Packages fs.FS
	Overlays mcpOverlay.RootPurger
}

type Installer struct {
	mcp             mcpConsumerAPI.BuiltinStore
	builtInTopology topology.Declaration
	overlays        mcpOverlay.RootPurger
	prepared        []PreparedPackage
	fingerprint     cryptoutil.Digest
	packageScopes   []basespec.Locator
}

func NewInstaller(
	dependencies InstallerDependencies,
) (*Installer, error) {
	if dependencies.MCP == nil || dependencies.Packages == nil {
		return nil, fmt.Errorf(
			"%w: MCP built-in installer dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	topologyValue := builtin.BuiltinTopologyDeclaration()
	if err := topologyValue.Validate(); err != nil {
		return nil, err
	}
	if err := dependencies.Registry.Validate(); err != nil {
		return nil, err
	}
	prepared, err := PrepareCollections(
		context.Background(),
		dependencies.Registry,
		dependencies.Packages,
	)
	if err != nil {
		return nil, err
	}
	fingerprint, err := hydrationFingerprint(topologyValue, prepared)
	if err != nil {
		return nil, err
	}
	scopes, err := packageScopes(prepared)
	if err != nil {
		return nil, err
	}
	return &Installer{
		mcp:             dependencies.MCP,
		builtInTopology: topologyValue,
		overlays:        dependencies.Overlays,
		prepared:        prepared,
		fingerprint:     fingerprint,
		packageScopes:   scopes,
	}, nil
}

func (*Installer) BuiltInName() string {
	return mcpDomain.BuiltInInstallerName
}

func (*Installer) BuiltInIDs() []string {
	return nil
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
	if err := ctx.Err(); err != nil {
		return topology.Hydration{}, err
	}
	return topology.Hydration{
		InstallerName: i.BuiltInName(),
		RootID:        i.builtInTopology.Root.ID,
		SourceID:      i.builtInTopology.Sources[0].ID,
		Fingerprint:   i.fingerprint,
	}, nil
}

func (i *Installer) EnsureHydration(
	ctx context.Context,
	current bool,
) error {
	if i == nil {
		return basespec.ErrClosed
	}
	if err := installerapi.RequirePrivileged(ctx); err != nil {
		return err
	}
	if !current && i.overlays != nil {
		if err := i.overlays.PurgeRoot(
			ctx,
			i.builtInTopology.Root.ID,
		); err != nil {
			return err
		}
	}
	if current {
		return nil
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
		i.builtInTopology.Root.ID,
		i.builtInTopology.Sources[0].ID,
	)
}

func (i *Installer) ensurePackages(
	ctx context.Context,
) error {
	for _, value := range i.prepared {
		if _, err := i.mcp.InstallBuiltInPackage(
			ctx,
			mcpConsumerAPI.BuiltInPackageInstallRequest{
				RootID:         i.builtInTopology.Root.ID,
				SourceID:       i.builtInTopology.Sources[0].ID,
				PackageAddress: value.PackageAddress,
				DocumentFile:   value.DocumentFile,
				PackageFiles:   value.PackageFiles,
				Expectations:   value.Expectations,
			},
		); err != nil {
			return fmt.Errorf(
				"install built-in MCP package %q: %w",
				value.Registration.EmbeddedPackageRoot,
				err,
			)
		}
	}
	return nil
}

func hydrationFingerprint(
	topologyValue topology.Declaration,
	prepared []PreparedPackage,
) (cryptoutil.Digest, error) {
	type packageFingerprint struct {
		Root   basespec.Locator  `json:"root"`
		Digest cryptoutil.Digest `json:"digest"`
	}
	values := make(
		[]packageFingerprint,
		0,
		len(prepared),
	)
	for _, value := range prepared {
		digest, err := PackageFingerprint(value)
		if err != nil {
			return "", err
		}
		values = append(values, packageFingerprint{
			Root:   value.Registration.EmbeddedPackageRoot,
			Digest: digest,
		})
	}
	sort.Slice(values, func(left, right int) bool {
		return values[left].Root < values[right].Root
	})
	return cryptoutil.CanonicalDigest(struct {
		SchemaVersion string               `json:"schemaVersion"`
		Topology      topology.Declaration `json:"topology"`
		Collections   []packageFingerprint `json:"collections"`
	}{
		SchemaVersion: mcpDomain.HydrationSchemaVersion,
		Topology:      topologyValue,
		Collections:   values,
	})
}

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
