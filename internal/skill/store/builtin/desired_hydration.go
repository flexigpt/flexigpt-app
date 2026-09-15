package builtin

import (
	"context"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi/topology"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	skillDomain "github.com/flexigpt/flexigpt-app/internal/skill/store/domain"
)

type hydrationPackage struct {
	PackageRoot basespec.Locator  `json:"packageRoot"`
	Digest      cryptoutil.Digest `json:"digest"`
}

func (i *Installer) DesiredHydration(
	ctx context.Context,
) (topology.Hydration, error) {
	if i == nil {
		return topology.Hydration{}, basespec.ErrClosed
	}
	if ctx == nil {
		return topology.Hydration{}, fmt.Errorf(
			"%w: built-in Skill hydration context is nil",
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
		RootID:        i.builtInTopology.Root.ID,
		SourceID:      i.builtInTopology.Sources[0].ID,
		Fingerprint:   i.fingerprint,
	}
	if err := value.Validate(); err != nil {
		return topology.Hydration{}, err
	}
	return value, nil
}

func hydrationFingerprint(
	topologyValue topology.Declaration,
	prepared []PreparedPackage,
) (cryptoutil.Digest, error) {
	packages := make(
		[]hydrationPackage,
		0,
		len(prepared),
	)
	for _, value := range prepared {
		digest, err := PackageFingerprint(value)
		if err != nil {
			return "", err
		}
		packages = append(packages, hydrationPackage{
			PackageRoot: value.EmbeddedPackageRoot,
			Digest:      digest,
		})
	}
	sort.Slice(packages, func(left, right int) bool {
		return packages[left].PackageRoot <
			packages[right].PackageRoot
	})

	return cryptoutil.CanonicalDigest(struct {
		SchemaVersion string               `json:"schemaVersion"`
		Topology      topology.Declaration `json:"topology"`
		Packages      []hydrationPackage   `json:"packages"`
	}{
		SchemaVersion: skillDomain.HydrationSchemaVersion,
		Topology:      topologyValue,
		Packages:      packages,
	})
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
	return i.EnsureBuiltInArtifacts(ctx)
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
	return i.skills.EnsureBuiltInSkillSourceCurrent(
		ctx,
		i.builtInTopology.Root.ID,
		i.builtInTopology.Sources[0].ID,
	)
}
