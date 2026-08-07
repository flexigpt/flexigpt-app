package artifactbuiltin

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/protection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/topology"
	"github.com/flexigpt/flexigpt-app/internal/builtin"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const hydrationFingerprintSchemaVersion = "agent.skill.builtin-hydration/v1"

type hydrationFingerprintDocument struct {
	SchemaVersion string                `json:"schemaVersion"`
	Topology      builtin.Registry      `json:"topology"`
	Registry      Registry              `json:"registry"`
	Collections   []hydrationCollection `json:"collections"`
}

type hydrationCollection struct {
	Registration          Collection                             `json:"registration"`
	DefinitionDigest      cryptoutil.Digest                      `json:"definitionDigest"`
	SourceScope           basespec.Locator                       `json:"sourceScope"`
	ExpectedMemberDigests map[basespec.Locator]cryptoutil.Digest `json:"expectedMemberDigests"`
	Artifacts             []hydrationArtifact                    `json:"artifacts"`
	Files                 []hydrationPackageFile                 `json:"files"`
}

type hydrationArtifact struct {
	Registration     Artifact          `json:"registration"`
	DefinitionDigest cryptoutil.Digest `json:"definitionDigest"`
}

type hydrationPackageFile struct {
	Locator basespec.Locator  `json:"locator"`
	Digest  cryptoutil.Digest `json:"digest"`
	Size    int64             `json:"size"`
}

// PrepareBuiltInHydration runs before generic topology creation. A changed
// binary-owned registry or package removes the previous hydration first, so
// the normal strict create APIs only ever see an empty topology or an exact
// replay of the current binary's desired state.
func (i *Installer) PrepareBuiltInHydration(
	ctx context.Context,
) error {
	if i == nil || i.hydrator == nil {
		return basespec.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf(
			"%w: built-in hydration context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := protection.RequirePrivilegedInstaller(ctx); err != nil {
		return err
	}

	i.pendingHydration = nil
	fingerprint, err := i.desiredHydrationFingerprint(ctx)
	if err != nil {
		return err
	}
	desired := topology.Hydration{
		InstallerName: i.BuiltInName(),
		RootID:        i.builtInTopology.Root.ID,
		SourceID:      i.builtInTopology.Source.ID,
		Fingerprint:   fingerprint,
	}
	if err := desired.Validate(); err != nil {
		return err
	}

	previous, found, err := i.hydrator.GetTopologyHydration(
		ctx,
		desired.InstallerName,
	)
	if err != nil {
		return err
	}
	if found && equalHydration(previous, desired) {
		i.pendingHydration = &desired
		return nil
	}

	roots := map[basespec.RootID]struct{}{
		desired.RootID: {},
	}
	if found {
		roots[previous.RootID] = struct{}{}
	}
	orderedRoots := make([]basespec.RootID, 0, len(roots))
	for rootID := range roots {
		orderedRoots = append(orderedRoots, rootID)
	}
	slices.Sort(orderedRoots)

	for _, rootID := range orderedRoots {
		if err := i.hydrator.ResetTopologyHydration(
			ctx,
			desired.InstallerName,
			rootID,
		); err != nil {
			return fmt.Errorf(
				"reset stale built-in hydration root %q: %w",
				rootID,
				err,
			)
		}
	}

	i.pendingHydration = &desired
	return nil
}

// CommitBuiltInHydration records completion only after BootstrapRegistry has
// successfully installed every registered built-in family.
func (i *Installer) CommitBuiltInHydration(
	ctx context.Context,
) error {
	if i == nil || i.hydrator == nil {
		return basespec.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf(
			"%w: built-in hydration context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := protection.RequirePrivilegedInstaller(ctx); err != nil {
		return err
	}
	if i.pendingHydration == nil {
		return fmt.Errorf(
			"%w: built-in hydration was not prepared",
			basespec.ErrInvalid,
		)
	}
	return i.hydrator.PutTopologyHydration(ctx, *i.pendingHydration)
}

func (i *Installer) desiredHydrationFingerprint(
	ctx context.Context,
) (cryptoutil.Digest, error) {
	input := hydrationFingerprintDocument{
		SchemaVersion: hydrationFingerprintSchemaVersion,
		Topology:      i.builtInTopology,
		Registry:      i.hydrated.Registry,
		Collections:   make([]hydrationCollection, 0, len(i.hydrated.Collections)),
	}

	for _, collectionValue := range i.hydrated.OrderedCollections() {
		files, err := builtin.ReadPackageFiles(
			ctx,
			i.packages,
			collectionValue.SourceScope,
		)
		if err != nil {
			return "", err
		}

		value := hydrationCollection{
			Registration:          collectionValue.Registration,
			DefinitionDigest:      collectionValue.Definition.Digest,
			SourceScope:           collectionValue.SourceScope,
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

func equalHydration(
	left topology.Hydration,
	right topology.Hydration,
) bool {
	return left.InstallerName == right.InstallerName &&
		left.RootID == right.RootID &&
		left.SourceID == right.SourceID &&
		left.Fingerprint == right.Fingerprint
}
