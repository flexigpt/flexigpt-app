package schemaadapter

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/protection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/topology"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/skill/store/bundle"
)

type hydrationArtifact struct {
	Registration     Artifact          `json:"registration"`
	DefinitionDigest cryptoutil.Digest `json:"definitionDigest"`
}

type hydrationPackageFile struct {
	Locator basespec.Locator  `json:"locator"`
	Digest  cryptoutil.Digest `json:"digest"`
	Size    int64             `json:"size"`
}
type hydrationCollection struct {
	Registration          Collection                             `json:"registration"`
	DefinitionDigest      cryptoutil.Digest                      `json:"definitionDigest"`
	ManagedPackageRoot    basespec.Locator                       `json:"managedPackageRoot"`
	ExpectedMemberDigests map[basespec.Locator]cryptoutil.Digest `json:"expectedMemberDigests"`
	Artifacts             []hydrationArtifact                    `json:"artifacts"`
	Files                 []hydrationPackageFile                 `json:"files"`
}
type hydrationFingerprintDocument struct {
	SchemaVersion string                `json:"schemaVersion"`
	Topology      topology.Declaration  `json:"topology"`
	Registry      Registry              `json:"registry"`
	Collections   []hydrationCollection `json:"collections"`
}

func (i *Installer) DesiredHydration(
	ctx context.Context,
) (topology.Hydration, error) {
	if i == nil {
		return topology.Hydration{}, basespec.ErrClosed
	}
	if ctx == nil {
		return topology.Hydration{}, fmt.Errorf(
			"%w: built-in hydration context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return topology.Hydration{}, err
	}
	if err := protection.RequirePrivilegedInstaller(ctx); err != nil {
		return topology.Hydration{}, err
	}

	fingerprint, err := i.desiredHydrationFingerprint(ctx)
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

func (i *Installer) EnsureHydration(
	ctx context.Context,
	current bool,
) error {
	if err := protection.RequirePrivilegedInstaller(ctx); err != nil {
		return err
	}
	if current {
		if err := i.FinalizeHydration(ctx); err == nil {
			return nil
		} else if ctx.Err() != nil {
			return err
		}
	}
	return i.EnsureBuiltInArtifacts(ctx)
}

// FinalizeHydration refreshes built-in Skill catalogs after all artifact
// families sharing the protected managed Source have finished publication.
//
// It does not publish packages. Bootstrap invokes this after the primary
// installer pass so Skill catalogs record the final shared Source revision.
func (i *Installer) FinalizeHydration(
	ctx context.Context,
) error {
	if i == nil {
		return basespec.ErrClosed
	}
	if err := protection.RequirePrivilegedInstaller(ctx); err != nil {
		return err
	}
	return i.ensureBuiltInCatalogsCurrent(ctx)
}

func (i *Installer) desiredHydrationFingerprint(
	ctx context.Context,
) (cryptoutil.Digest, error) {
	input := hydrationFingerprintDocument{
		SchemaVersion: artifactbuiltin.AgentSkillHydrationFingerprintSchemaVersion,
		Topology:      i.builtInTopology,
		Registry:      i.hydrated.Registry,
		Collections:   make([]hydrationCollection, 0, len(i.hydrated.Collections)),
	}

	for _, collectionValue := range i.hydrated.OrderedCollections() {
		files, err := topology.ReadPackageFiles(
			ctx,
			i.packages,
			collectionValue.EmbeddedPackageRoot,
		)
		if err != nil {
			return "", err
		}
		packageAddress, err := bundle.BuiltInCollectionPackageAddress(
			basespec.LogicalName(collectionValue.Definition.LogicalName),
			basespec.LogicalVersion(collectionValue.Definition.LogicalVersion),
		)
		if err != nil {
			return "", err
		}
		managedPackageRoot, err := packageAddress.Directory()
		if err != nil {
			return "", err
		}
		if collectionValue.Definition.Digest == nil {
			return "", fmt.Errorf(
				"%w: hydrated built-in collection has no digest",
				basespec.ErrInvalid,
			)
		}
		definitionDigest := cryptoutil.Digest(
			*collectionValue.Definition.Digest,
		)
		if err := cryptoutil.ValidateDigest(definitionDigest); err != nil {
			return "", err
		}
		value := hydrationCollection{
			Registration:          collectionValue.Registration,
			DefinitionDigest:      definitionDigest,
			ManagedPackageRoot:    managedPackageRoot,
			ExpectedMemberDigests: collectionValue.ExpectedMemberDigests,
			Artifacts:             make([]hydrationArtifact, 0, len(collectionValue.Artifacts)),
			Files:                 make([]hydrationPackageFile, 0, len(files)),
		}
		for _, artifactValue := range collectionValue.Artifacts {
			value.Artifacts = append(value.Artifacts, hydrationArtifact{
				Registration:     artifactValue.Registration,
				DefinitionDigest: artifactValue.SkillDefinition.Digest,
			})
		}
		sort.Slice(value.Artifacts, func(left, right int) bool {
			return value.Artifacts[left].Registration.ID <
				value.Artifacts[right].Registration.ID
		})
		for _, file := range files {
			value.Files = append(value.Files, hydrationPackageFile{
				Locator: file.Locator,
				Digest:  cryptoutil.DigestBytes(file.Content),
				Size:    int64(len(file.Content)),
			})
		}
		input.Collections = append(input.Collections, value)
	}
	raw, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	canonical, err := jsonutil.Canonicalize(raw)
	if err != nil {
		return "", err
	}
	return cryptoutil.DigestBytes(canonical), nil
}
