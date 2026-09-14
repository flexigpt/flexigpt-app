package builtin

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi/topology"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	skillDomain "github.com/flexigpt/flexigpt-app/internal/skill/store/domain"
)

type hydrationPackageFile struct {
	Locator basespec.Locator  `json:"locator"`
	Digest  cryptoutil.Digest `json:"digest"`
	Size    int64             `json:"size"`
}

type hydrationSkill struct {
	Registration     Skill                  `json:"registration"`
	DefinitionDigest cryptoutil.Digest      `json:"definitionDigest"`
	PackageAddress   string                 `json:"packageAddress"`
	Files            []hydrationPackageFile `json:"files"`
}

type hydrationFingerprintDocument struct {
	SchemaVersion         string               `json:"schemaVersion"`
	Topology              topology.Declaration `json:"topology"`
	Registry              Registry             `json:"registry"`
	Skills                []hydrationSkill     `json:"skills"`
	CollectionName        basespec.LogicalName `json:"collectionName"`
	CollectionDescription string               `json:"collectionDescription"`
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

	fingerprint, err := i.desiredHydrationFingerprint()
	if err != nil {
		return topology.Hydration{}, err
	}
	value := topology.Hydration{
		InstallerName: i.BuiltInName(),
		RootID:        i.builtInTopology.Root.ID,
		SourceID:      i.builtInTopology.Sources[0].ID,
		Fingerprint:   fingerprint,
	}
	if err := value.Validate(); err != nil {
		return topology.Hydration{}, err
	}
	return value, nil
}

func (i *Installer) desiredHydrationFingerprint() (
	cryptoutil.Digest,
	error,
) {
	input := hydrationFingerprintDocument{
		SchemaVersion:         skillDomain.HydrationSchemaVersion,
		Topology:              i.builtInTopology,
		Registry:              i.hydrated.Registry,
		Skills:                make([]hydrationSkill, 0, len(i.hydrated.Skills)),
		CollectionName:        skillDomain.BuiltinSkillCollectionName,
		CollectionDescription: skillDomain.BuiltinSkillCollectionDescription,
	}
	for _, value := range i.hydrated.OrderedSkills() {
		files := make(
			[]hydrationPackageFile,
			0,
			len(value.PackageFiles),
		)
		for _, file := range value.PackageFiles {
			files = append(files, hydrationPackageFile{
				Locator: file.Locator,
				Digest:  cryptoutil.DigestBytes(file.Content),
				Size:    int64(len(file.Content)),
			})
		}
		directory, err := value.PackageAddress.Directory()
		if err != nil {
			return "", err
		}
		input.Skills = append(input.Skills, hydrationSkill{
			Registration:     value.Registration,
			DefinitionDigest: value.Definition.Digest,
			PackageAddress:   string(directory),
			Files:            files,
		})
	}
	return cryptoutil.CanonicalDigest(input)
}

func (i *Installer) EnsureHydration(
	ctx context.Context,
	_ bool,
) error {
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
