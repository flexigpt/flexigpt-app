package skillruntime

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/flexigpt/agentskills-go/provider"
	"github.com/flexigpt/agentskills-go/provider/fs"
	agentskillsRuntime "github.com/flexigpt/agentskills-go/runtime"
	agentskillsRuntimeSpec "github.com/flexigpt/agentskills-go/runtime/spec"

	inferencewrapperSpec "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/inferencewrapper/spec"
	skillRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/skill"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog"
	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	resourceFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/support"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/materialize"
	skillSource "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/source"
)

const artifactRootCatalogPrefix = "artifact-root:"

var ErrArtifactSkillSelectionRequired = errors.New(
	"artifact skill selection is required",
)

var (
	_ skillRuntime.CatalogSource       = (*RuntimeAdapter)(nil)
	_ inferencewrapperSpec.SkillSource = (*RuntimeAdapter)(nil)
)

type ResolvedArtifactSkill struct {
	Artifact   artifactModel.ArtifactRef
	Definition provider.SkillDef
	Version    string
	Enabled    bool
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

// RuntimeAdapter projects source-backed Skills into the Artifact-unaware
// runtime and implements completion hydration. It owns no closable resources.
type RuntimeAdapter struct {
	artifacts       artifact.API
	cat             catalog.API
	resources       resourceFlow.API
	nativeResources resourceFlow.NativePathAPI
	documents       support.Documents
	runtime         *skillRuntime.Service

	syncMu  sync.Mutex
	syncing map[rootModel.RootID]*rootCatalogSync
}

type rootCatalogSync struct {
	done chan struct{}
	err  error
}

func NewRuntimeAdapter(
	artifacts artifact.API,
	cat catalog.API,
	resources resourceFlow.API,
	nativeResources resourceFlow.NativePathAPI,
	documents support.Documents,
) (*RuntimeAdapter, error) {
	if artifacts == nil || cat == nil ||
		resources == nil || nativeResources == nil {
		return nil, fmt.Errorf(
			"%w: Skill runtime adapter dependencies are incomplete",
			spec.ErrInvalid,
		)
	}
	if err := documents.Validate(); err != nil {
		return nil, err
	}

	return &RuntimeAdapter{
		artifacts:       artifacts,
		cat:             cat,
		resources:       resources,
		nativeResources: nativeResources,
		documents:       documents.Clone(),
		syncing:         make(map[rootModel.RootID]*rootCatalogSync),
	}, nil
}

// BindRuntime completes assembly after the runtime has received this adapter
// as its CatalogSource. Call once, before serving requests.
func (a *RuntimeAdapter) BindRuntime(runtime *skillRuntime.Service) error {
	if runtime == nil {
		return fmt.Errorf("%w: Skill runtime is required", spec.ErrInvalid)
	}
	if a.runtime != nil {
		return fmt.Errorf("%w: Skill runtime is already bound", spec.ErrConflict)
	}
	a.runtime = runtime
	return nil
}

func (a *RuntimeAdapter) RunScriptsEnabled() bool {
	return a != nil && a.runtime != nil && a.runtime.SupportsRunScript()
}

// Skills implements the runtime-owned CatalogSource contract. Plugin
// membership does not determine runtime catalog ownership.
func (a *RuntimeAdapter) Skills(
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
				spec.ErrRefreshRequired,
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

// ResolveSkills synchronizes each selected Root once, then verifies the
// selected current revisions against the registrations actually installed.
//
// The second materialization is intentional: a selection changed during Root
// synchronization must not be returned as though its new revision were loaded.
func (a *RuntimeAdapter) ResolveSkills(
	ctx context.Context,
	refs []artifactModel.ArtifactRef,
) (ResolvedSkills, error) {
	if a == nil || a.runtime == nil {
		return ResolvedSkills{}, spec.ErrClosed
	}

	roots, err := selectionRoots(refs)
	if err != nil {
		return ResolvedSkills{}, err
	}
	for _, rootID := range roots {
		if err := ctx.Err(); err != nil {
			return ResolvedSkills{}, err
		}
		if err := a.syncRoot(ctx, rootID); err != nil {
			return ResolvedSkills{}, fmt.Errorf(
				"%w: synchronize Skill Root %q: %w",
				spec.ErrReferenceUnresolved,
				rootID,
				err,
			)
		}
	}

	records, err := a.readRecords(ctx, refs)
	if err != nil {
		return ResolvedSkills{}, fmt.Errorf(
			"%w: read selected Skill Artifacts: %w",
			spec.ErrReferenceUnresolved,
			err,
		)
	}
	values, err := a.materializeRecords(ctx, records)
	if err != nil {
		return ResolvedSkills{}, err
	}

	for _, value := range values {
		if !a.runtime.IsRegistered(skillRuntime.SkillRegistration{
			Definition: value.Definition,
			Revision:   value.Version,
		}) {
			return ResolvedSkills{}, fmt.Errorf(
				"%w: Skill %s/%s revision %q is not registered",
				spec.ErrReferenceUnresolved,
				value.Artifact.RootID,
				value.Artifact.ArtifactID,
				value.Version,
			)
		}
	}
	return selectionFromValues(values)
}

func (a *RuntimeAdapter) SyncRootCatalog(
	ctx context.Context,
	rootID rootModel.RootID,
) error {
	if a == nil || a.runtime == nil {
		return spec.ErrClosed
	}
	if err := rootID.Validate(); err != nil {
		return err
	}
	return a.syncRoot(ctx, rootID)
}

// ResolveSkillSession prepares all completion Skill data from one verified
// selection. Errors return no partially advertised Skill capability.
func (a *RuntimeAdapter) ResolveSkillSession(
	ctx context.Context,
	request inferencewrapperSpec.SkillSessionRequest,
) (inferencewrapperSpec.SkillSession, error) {
	if strings.TrimSpace(string(request.SessionID)) == "" {
		return inferencewrapperSpec.SkillSession{}, fmt.Errorf(
			"%w: Skill session ID is required",
			skillRuntime.ErrInvalidRequest,
		)
	}

	selected, err := a.ResolveSkills(ctx, request.Artifacts)
	if err != nil {
		return inferencewrapperSpec.SkillSession{}, err
	}

	filter := agentskillsRuntime.SkillListFilter{
		AllowSkills: selected.Definitions,
		SessionID:   request.SessionID,
		Activity:    agentskillsRuntimeSpec.SkillActivityAny,
	}
	available, err := a.runtime.ListAgentSkills(ctx, &filter)
	if err != nil {
		return inferencewrapperSpec.SkillSession{}, err
	}

	filter.Activity = agentskillsRuntimeSpec.SkillActivityActive
	active, err := a.runtime.ListAgentSkills(ctx, &filter)
	if err != nil {
		return inferencewrapperSpec.SkillSession{}, err
	}

	activity := agentskillsRuntimeSpec.SkillActivityInactive
	if len(active) != 0 {
		activity = agentskillsRuntimeSpec.SkillActivityAny
	}
	prompt, err := a.runtime.SkillsPrompt(ctx, &agentskillsRuntime.SkillFilter{
		AllowSkills: selected.Definitions,
		SessionID:   request.SessionID,
		Activity:    activity,
	})
	if err != nil {
		return inferencewrapperSpec.SkillSession{}, err
	}

	output := inferencewrapperSpec.SkillSession{
		AvailableArtifacts: selected.ArtifactRefs(available),
		ActiveArtifacts:    selected.ArtifactRefs(active),
		Prompt:             strings.TrimSpace(prompt),
	}
	if output.Prompt == "" {
		return output, nil
	}

	includeAll := len(active) != 0
	includeRunScript := request.IncludeRunScript && a.RunScriptsEnabled()
	output.RulesPrompt = skillsRulesPrompt(includeAll, includeRunScript)
	output.ToolChoices, err = buildSkillToolChoices(includeAll, includeRunScript)
	if err != nil {
		return inferencewrapperSpec.SkillSession{}, err
	}
	return output, nil
}

// syncRoot coalesces overlapping observations, but does not cache successful
// observations across requests. Runtime still owns reconciliation generations.
func (a *RuntimeAdapter) syncRoot(
	ctx context.Context,
	rootID rootModel.RootID,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	a.syncMu.Lock()
	if current, found := a.syncing[rootID]; found {
		a.syncMu.Unlock()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-current.done:
			return current.err
		}
	}

	current := &rootCatalogSync{done: make(chan struct{})}
	a.syncing[rootID] = current
	a.syncMu.Unlock()

	err := a.runtime.SyncCatalog(
		ctx,
		skillRuntime.CatalogID(artifactRootCatalogPrefix+string(rootID)),
	)

	a.syncMu.Lock()
	current.err = err
	delete(a.syncing, rootID)
	close(current.done)
	a.syncMu.Unlock()

	return err
}

func (a *RuntimeAdapter) readRecords(
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
			spec.ErrInvalid,
		)
	}
	for index, record := range records {
		if record.Ref() != refs[index] {
			return nil, fmt.Errorf(
				"%w: Skill Artifact batch returned another Artifact",
				spec.ErrRefreshRequired,
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

func (a *RuntimeAdapter) materializeRecords(
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
				spec.ErrReferenceUnresolved,
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
			spec.ErrInvalid,
		)
	}

	output := make([]ResolvedArtifactSkill, 0, len(materials))
	for index, material := range materials {
		record := records[index]
		if material.Artifact != record.Ref() {
			return nil, fmt.Errorf(
				"%w: Skill materializer resolved another Artifact",
				spec.ErrRefreshRequired,
			)
		}
		if material.Document.Name == "" ||
			material.RuntimeLocation == "" ||
			material.VersionDigest == "" {
			return nil, fmt.Errorf(
				"%w: materialized Skill definition or revision is incomplete",
				spec.ErrInvalid,
			)
		}

		output = append(output, ResolvedArtifactSkill{
			Artifact: record.Ref(),
			Definition: provider.SkillDef{
				Type:     fs.Type,
				Name:     material.Document.Name,
				Location: material.RuntimeLocation,
			},
			Version: "artifact-skill:" + string(material.VersionDigest),
			Enabled: true,
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
				spec.ErrConflict,
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
			return nil, fmt.Errorf("%w: duplicate Skill ArtifactRef", spec.ErrInvalid)
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
			spec.ErrInvalid,
			catalogID,
		)
	}
	rootID := rootModel.RootID(raw)
	if err := rootID.Validate(); err != nil {
		return "", err
	}
	return rootID, nil
}
