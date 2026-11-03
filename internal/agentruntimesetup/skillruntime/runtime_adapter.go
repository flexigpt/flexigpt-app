package skillruntime

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/flexigpt/agentskills-go/provider"
	"github.com/flexigpt/agentskills-go/provider/fs"
	agentskillsRuntimeSpec "github.com/flexigpt/agentskills-go/runtime/spec"
	skillRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/skill"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog"
	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	resourceFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/support"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/materialize"
	skillSource "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/source"
)

const artifactRootCatalogPrefix = "artifact-root:"

var ErrArtifactSkillSelectionRequired = errors.New(
	"artifact skill selection is required",
)

type ResolvedArtifactSkill struct {
	Artifact         artifactModel.ArtifactRef
	Definition       provider.SkillDef
	Version          string
	Revision         uint64
	DefinitionDigest string
	Enabled          bool
	DisplayName      string
	Description      string
	SourceLocation   string
}

// ResolvedSkills is a request-owned, verified selection. Values preserve the
// caller's ordering. No slices or maps are shared with the adapter.
type ResolvedSkills struct {
	Values                []ResolvedArtifactSkill
	Definitions           []provider.SkillDef
	ArtifactsByDefinition map[provider.SkillDef]artifactModel.ArtifactRef
}

func (s ResolvedSkills) ArtifactRefs(
	records []agentskillsRuntimeSpec.SkillRecord,
) []artifactModel.ArtifactRef {
	output := make([]artifactModel.ArtifactRef, 0, len(records))
	for _, record := range records {
		if ref, found := s.ArtifactsByDefinition[record.Def]; found {
			output = append(output, ref)
		}
	}
	sort.Slice(output, func(left, right int) bool {
		if output[left].RootID != output[right].RootID {
			return output[left].RootID < output[right].RootID
		}
		return output[left].ArtifactID < output[right].ArtifactID
	})
	return output
}

// CatalogAdapter projects source-backed Skills into Skill Runtime's
// Artifact-unaware CatalogSource contract.
type CatalogAdapter struct {
	artifacts       artifact.API
	cat             catalog.API
	resources       resourceFlow.API
	nativeResources resourceFlow.NativePathAPI
	documents       support.Documents
}

func NewCatalogAdapter(
	artifacts artifact.API,
	cat catalog.API,
	resources resourceFlow.API,
	nativeResources resourceFlow.NativePathAPI,
	documents support.Documents,
) (*CatalogAdapter, error) {
	if artifacts == nil || cat == nil ||
		resources == nil || nativeResources == nil {
		return nil, fmt.Errorf(
			"%w: Skill runtime adapter dependencies are incomplete",
			storeSpec.ErrInvalid,
		)
	}
	if err := documents.Validate(); err != nil {
		return nil, err
	}

	return &CatalogAdapter{
		artifacts:       artifacts,
		cat:             cat,
		resources:       resources,
		nativeResources: nativeResources,
		documents:       documents.Clone(),
	}, nil
}

// Skills implements the runtime-owned CatalogSource contract. Plugin
// membership does not determine runtime catalog ownership.
func (a *CatalogAdapter) Skills(
	ctx context.Context,
	catalogID skillRuntime.CatalogID,
) ([]skillRuntime.SkillRegistration, error) {
	rootID, err := rootForCatalogID(catalogID)
	if err != nil {
		return nil, err
	}

	entries, err := a.cat.ListByRoot(ctx, rootID, catalogModel.ListOptions{})
	if err != nil {
		return nil, err
	}

	refs := make([]artifactModel.ArtifactRef, 0, len(entries))
	for _, entry := range entries {
		if skillSource.IsSkillKind(entry.Kind) &&
			entry.State == artifactModel.StateAvailable &&
			entry.Enabled {
			refs = append(refs, entry.Ref())
		}
	}

	records, err := a.readRecords(ctx, refs)
	if err != nil {
		return nil, err
	}

	// Enablement or availability may have changed since catalog enumeration.
	// Such records must disappear from the desired runtime catalog.
	candidates := make([]artifactModel.Artifact, 0, len(records))
	for _, record := range records {
		if record.RootID != rootID {
			return nil, fmt.Errorf(
				"%w: Skill catalog returned an Artifact from another Root",
				storeSpec.ErrRefreshRequired,
			)
		}
		if skillSource.IsSkillKind(record.Kind) &&
			record.State == artifactModel.StateAvailable &&
			record.Enabled {
			candidates = append(candidates, record)
		}
	}

	values, err := a.materializeRecords(ctx, candidates)
	if err != nil {
		return nil, err
	}
	if _, err := selectionFromValues(values); err != nil {
		return nil, err
	}

	output := make([]skillRuntime.SkillRegistration, 0, len(values))
	for _, value := range values {
		output = append(output, skillRuntime.SkillRegistration{
			Definition: value.Definition,
			Revision:   value.Version,
		})
	}
	return output, nil
}

func (a *CatalogAdapter) readRecords(
	ctx context.Context,
	refs []artifactModel.ArtifactRef,
) ([]artifactModel.Artifact, error) {
	if len(refs) == 0 {
		return []artifactModel.Artifact{}, nil
	}
	records, err := a.artifacts.GetMany(ctx, refs)
	if err != nil {
		return nil, err
	}
	if len(records) != len(refs) {
		return nil, fmt.Errorf(
			"%w: Skill Artifact batch returned an unexpected result count",
			storeSpec.ErrInvalid,
		)
	}
	for index, record := range records {
		if record.Ref() != refs[index] {
			return nil, fmt.Errorf(
				"%w: Skill Artifact batch returned another Artifact",
				storeSpec.ErrRefreshRequired,
			)
		}
	}
	return records, nil
}

// Native paths are available only to the trusted materialization boundary.
type materializeResources struct {
	resourceFlow.API
	resourceFlow.NativePathAPI
}

func (a *CatalogAdapter) materializeRecords(
	ctx context.Context,
	records []artifactModel.Artifact,
) ([]ResolvedArtifactSkill, error) {
	if len(records) == 0 {
		return []ResolvedArtifactSkill{}, nil
	}
	for _, record := range records {
		if !skillSource.IsSkillKind(record.Kind) ||
			record.State != artifactModel.StateAvailable ||
			!record.Enabled {
			return nil, fmt.Errorf(
				"%w: Artifact %s/%s is not an available enabled Skill",
				storeSpec.ErrReferenceUnresolved,
				record.RootID,
				record.ID,
			)
		}
	}

	materials, err := materialize.ResolveAll(
		ctx,
		materializeResources{
			API:           a.resources,
			NativePathAPI: a.nativeResources,
		},
		a.documents,
		records,
	)
	if err != nil {
		return nil, err
	}
	if len(materials) != len(records) {
		return nil, fmt.Errorf(
			"%w: Skill materializer returned an unexpected result count",
			storeSpec.ErrInvalid,
		)
	}

	output := make([]ResolvedArtifactSkill, 0, len(materials))
	for index, material := range materials {
		record := records[index]
		if material.Artifact != record.Ref() {
			return nil, fmt.Errorf(
				"%w: Skill materializer resolved another Artifact",
				storeSpec.ErrRefreshRequired,
			)
		}
		if material.Document.Name == "" ||
			material.RuntimeLocation == "" ||
			material.VersionDigest == "" {
			return nil, fmt.Errorf(
				"%w: materialized Skill definition or revision is incomplete",
				storeSpec.ErrInvalid,
			)
		}
		definitionDigest := ""
		if record.ResolvedDefinition != nil {
			definitionDigest = string(*record.ResolvedDefinition)
		}

		output = append(output, ResolvedArtifactSkill{
			Artifact: record.Ref(),
			Definition: provider.SkillDef{
				Type:     fs.Type,
				Name:     material.Document.Name,
				Location: material.RuntimeLocation,
			},
			Version:          "artifact-skill:" + string(material.VersionDigest),
			Revision:         record.Revision,
			DefinitionDigest: definitionDigest,
			Enabled:          true,
			DisplayName:      record.DisplayName,
			Description:      material.Document.Description,
			SourceLocation:   string(record.Binding.Locator),
		})
	}
	return output, nil
}

func selectionFromValues(values []ResolvedArtifactSkill) (ResolvedSkills, error) {
	output := ResolvedSkills{
		Values:                values,
		Definitions:           make([]provider.SkillDef, 0, len(values)),
		ArtifactsByDefinition: make(map[provider.SkillDef]artifactModel.ArtifactRef, len(values)),
	}
	for _, value := range values {
		if previous, found := output.ArtifactsByDefinition[value.Definition]; found &&
			previous != value.Artifact {
			return ResolvedSkills{}, fmt.Errorf(
				"%w: Skill Artifacts %s/%s and %s/%s resolve to one runtime definition",
				storeSpec.ErrConflict,
				previous.RootID,
				previous.ArtifactID,
				value.Artifact.RootID,
				value.Artifact.ArtifactID,
			)
		}
		output.ArtifactsByDefinition[value.Definition] = value.Artifact
		output.Definitions = append(output.Definitions, value.Definition)
	}
	return output, nil
}

func selectionRoots(
	refs []artifactModel.ArtifactRef,
) ([]rootModel.RootID, error) {
	if len(refs) == 0 {
		return nil, ErrArtifactSkillSelectionRequired
	}

	seenRefs := make(map[artifactModel.ArtifactRef]struct{}, len(refs))
	seenRoots := make(map[rootModel.RootID]struct{})
	roots := make([]rootModel.RootID, 0)
	for _, ref := range refs {
		if err := ref.Validate(); err != nil {
			return nil, err
		}
		if _, duplicate := seenRefs[ref]; duplicate {
			return nil, fmt.Errorf("%w: duplicate Skill ArtifactRef", storeSpec.ErrInvalid)
		}
		seenRefs[ref] = struct{}{}
		if _, found := seenRoots[ref.RootID]; !found {
			seenRoots[ref.RootID] = struct{}{}
			roots = append(roots, ref.RootID)
		}
	}
	return roots, nil
}

func rootForCatalogID(
	catalogID skillRuntime.CatalogID,
) (rootModel.RootID, error) {
	raw, found := strings.CutPrefix(string(catalogID), artifactRootCatalogPrefix)
	if !found {
		return "", fmt.Errorf(
			"%w: unsupported Skill catalog ID %q",
			storeSpec.ErrInvalid,
			catalogID,
		)
	}
	rootID := rootModel.RootID(raw)
	if err := rootID.Validate(); err != nil {
		return "", err
	}
	return rootID, nil
}
