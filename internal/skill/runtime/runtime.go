package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/flexigpt/agentskills-go"
	"github.com/flexigpt/agentskills-go/fsskillprovider"
	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/llmtoolsutil"
)

var errSkillInvalidRequest = errors.New("invalid request")

const (
	runtimeResyncTimeout             = 30 * time.Second
	runtimeForegroundValidateTimeout = 15 * time.Second
)

// SkillRuntime owns the in-memory Agent Skills catalog, provider lifecycle,
// sessions, prompt generation, rendering, and tool invocation.
type SkillRuntime struct {
	resolver          *ArtifactRouter
	runtime           *agentskills.Runtime
	runScriptsEnabled bool

	rtResyncMu sync.Mutex

	lifecycleMu sync.RWMutex
	closed      bool

	// These maps are only an ephemeral inventory of provider registrations.
	// Artifact Store remains the source of truth for every Artifact decision.
	managedCollections map[collection.CollectionRef]runtimeDesiredView
	managedRuntime     map[agentskillsSpec.SkillDef]string
}

type skillRuntimeOptions struct {
	runtime              *agentskills.Runtime
	resolver             *ArtifactRouter
	runScriptsEnabled    bool
	runScriptsConfigured bool
}

type SkillRuntimeOption func(*skillRuntimeOptions) error

func WithRuntime(value *agentskills.Runtime) SkillRuntimeOption {
	return func(options *skillRuntimeOptions) error {
		if value == nil {
			return errors.New("skill runtime is nil")
		}
		options.runtime = value
		return nil
	}
}

func WithArtifactResolver(
	value *ArtifactRouter,
) SkillRuntimeOption {
	return func(options *skillRuntimeOptions) error {
		if value == nil {
			return errors.New("artifact skill resolver is nil")
		}
		options.resolver = value
		return nil
	}
}

// WithRunScripts configures the shared filesystem-provider execution policy.
// Workspace filesystem skills and installed filesystem skills always use this
// same provider and therefore this same policy.
func WithRunScripts(enabled bool) SkillRuntimeOption {
	return func(options *skillRuntimeOptions) error {
		options.runScriptsEnabled = enabled
		options.runScriptsConfigured = true
		return nil
	}
}

// NewSkillRuntime creates an Artifact-backed Agent Skills runtime. Durable
// Skill identity is always artifact.ArtifactRef; no standalone Skill Store,
// bundle slug, legacy Skill ID, or source location enters this package.
func NewSkillRuntime(
	opts ...SkillRuntimeOption,
) (*SkillRuntime, error) {
	options := skillRuntimeOptions{}
	for _, option := range opts {
		if option != nil {
			if err := option(&options); err != nil {
				return nil, err
			}
		}
	}
	if options.resolver == nil {
		return nil, errors.New("artifact skill resolver is required")
	}
	if options.runtime == nil {
		runScriptsEnabled := false
		if options.runScriptsConfigured {
			runScriptsEnabled = options.runScriptsEnabled
		}
		filesystemProvider, err := fsskillprovider.New(
			fsskillprovider.WithRunScripts(runScriptsEnabled),
		)
		if err != nil {
			return nil, err
		}
		options.runtime, err = agentskills.New(
			agentskills.WithProvider(filesystemProvider),
			agentskills.WithLogger(slog.Default()),
		)
		if err != nil {
			return nil, err
		}
		options.runScriptsEnabled = runScriptsEnabled
	} else if !options.runScriptsConfigured {
		options.runScriptsEnabled = false
	}

	value := &SkillRuntime{
		resolver:           options.resolver,
		runtime:            options.runtime,
		runScriptsEnabled:  options.runScriptsEnabled,
		managedCollections: map[collection.CollectionRef]runtimeDesiredView{},
		managedRuntime:     map[agentskillsSpec.SkillDef]string{},
	}
	return value, nil
}

func (s *SkillRuntime) AgentSkillsRuntime() *agentskills.Runtime {
	if s == nil || s.isClosed() {
		return nil
	}
	return s.runtime
}

// RunScriptsEnabled reports the effective shared filesystem-provider policy.
// Inference composition uses this value only to decide whether
// skills-runscript is advertised to the model.
func (s *SkillRuntime) RunScriptsEnabled() bool {
	if s == nil || s.isClosed() {
		return false
	}
	return s.runScriptsEnabled
}

// ManagedCollectionRefs returns derived runtime partitions. Collection kind is
// deliberately not encoded in a Skill reference; application feature
// synchronizers use their own typed Collection discovery to decide removal.
func (s *SkillRuntime) ManagedCollectionRefs() []collection.CollectionRef {
	if s == nil {
		return nil
	}
	s.rtResyncMu.Lock()
	defer s.rtResyncMu.Unlock()

	output := make([]collection.CollectionRef, 0, len(s.managedCollections))
	for ref := range s.managedCollections {
		output = append(output, ref)
	}
	sort.Slice(output, func(left, right int) bool {
		if output[left].RootID != output[right].RootID {
			return output[left].RootID < output[right].RootID
		}
		return output[left].CollectionID < output[right].CollectionID
	})
	return output
}

func (s *SkillRuntime) Close() {
	if s == nil {
		return
	}

	s.lifecycleMu.Lock()
	if s.closed {
		s.lifecycleMu.Unlock()
		return
	}
	s.closed = true
	s.lifecycleMu.Unlock()

	s.rtResyncMu.Lock()
	if s.runtime != nil && len(s.managedRuntime) != 0 {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			runtimeResyncTimeout,
		)
		remaining, err := s.runtimeApplyDesired(
			ctx,
			s.managedRuntime,
			newRuntimeDesiredView(),
			runtimeApplyBestEffort,
		)
		cancel()
		if err != nil || len(remaining) != 0 {
			slog.Error(
				"remove managed Skill runtime registrations during close",
				"remaining",
				len(remaining),
				"error",
				err,
			)
		}
	}
	s.managedCollections = map[collection.CollectionRef]runtimeDesiredView{}
	s.managedRuntime = map[agentskillsSpec.SkillDef]string{}
	s.rtResyncMu.Unlock()
}

func (s *SkillRuntime) CreateSkillSession(
	ctx context.Context,
	req *CreateSkillSessionRequest,
) (*CreateSkillSessionResponse, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, fmt.Errorf("%w: %w", errSkillInvalidRequest, err)
	}
	if req == nil || req.Body == nil {
		return nil, fmt.Errorf("%w: missing request", errSkillInvalidRequest)
	}
	if len(req.Body.AllowArtifacts) == 0 {
		return nil, fmt.Errorf("%w: allowArtifacts required", errSkillInvalidRequest)
	}
	if err := validateArtifactRefs(req.Body.AllowArtifacts); err != nil {
		return nil, fmt.Errorf("%w: invalid allowArtifacts: %w", errSkillInvalidRequest, err)
	}
	if err := validateArtifactRefs(req.Body.ActiveArtifacts); err != nil {
		return nil, fmt.Errorf("%w: invalid activeArtifacts: %w", errSkillInvalidRequest, err)
	}

	previousSessionID := strings.TrimSpace(string(req.Body.CloseSessionID))

	resolved, err := s.resolveAllowArtifacts(ctx, req.Body.AllowArtifacts)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errSkillInvalidRequest, err)
	}
	activeRefs := subsetArtifacts(
		req.Body.AllowArtifacts,
		req.Body.ActiveArtifacts,
	)
	activeDefinitions := map[agentskillsSpec.SkillDef]struct{}{}
	for _, ref := range activeRefs {
		if definition, found := resolved.RefToDef[artifactRefKey(ref)]; found {
			activeDefinitions[definition] = struct{}{}
		}
	}

	activeDefs := make([]agentskillsSpec.SkillDef, 0, len(activeDefinitions))
	for definition := range activeDefinitions {
		activeDefs = append(activeDefs, definition)
	}
	sortSkillDefs(activeDefs)

	options := []agentskills.SessionOption{}
	if req.Body.MaxActivePerSession > 0 {
		options = append(
			options,
			agentskills.WithSessionMaxActivePerSession(
				req.Body.MaxActivePerSession,
			),
		)
	}
	if len(activeDefs) > 0 {
		options = append(
			options,
			agentskills.WithSessionActiveSkills(activeDefs),
		)
	}

	sessionID, _, err := s.runtime.NewSession(ctx, options...)
	if err != nil {
		return nil, err
	}

	records, err := s.runtime.ListSkills(ctx, &agentskills.SkillListFilter{
		SessionID:   sessionID,
		Activity:    agentskillsSpec.SkillActivityActive,
		AllowSkills: resolved.AllowDefs,
	})
	if err != nil {
		closeErr := s.runtime.CloseSession(
			context.WithoutCancel(ctx),
			sessionID,
		)
		return nil, errors.Join(err, closeErr)
	}

	active := map[agentskillsSpec.SkillDef]struct{}{}
	for _, record := range records {
		active[record.Def] = struct{}{}
	}

	// Preserve the old session until the replacement has been created and
	// inspected successfully. Closing the old session is explicitly best
	// effort and must not invalidate a successful new session response.
	if previousSessionID != "" {
		_ = s.runtime.CloseSession(context.WithoutCancel(ctx), agentskillsSpec.SessionID(previousSessionID))
	}

	return &CreateSkillSessionResponse{
		Body: &CreateSkillSessionResponseBody{
			SessionID:       sessionID,
			ActiveArtifacts: buildActiveArtifacts(resolved.DefToArtifacts, active),
		},
	}, nil
}

func (s *SkillRuntime) CloseSkillSession(
	ctx context.Context,
	req *CloseSkillSessionRequest,
) (*CloseSkillSessionResponse, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, fmt.Errorf("%w: %w", errSkillInvalidRequest, err)
	}
	if req == nil {
		return nil, fmt.Errorf("%w: missing request", errSkillInvalidRequest)
	}
	if err := s.runtime.CloseSession(ctx, req.SessionID); err != nil {
		return nil, err
	}
	return &CloseSkillSessionResponse{}, nil
}

func (s *SkillRuntime) GetSkillsPrompt(
	ctx context.Context,
	req *GetSkillsPromptRequest,
) (*GetSkillsPromptResponse, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, fmt.Errorf("%w: %w", errSkillInvalidRequest, err)
	}
	if req == nil || req.Body == nil || req.Body.Filter == nil {
		return nil, fmt.Errorf("%w: Skill prompt filter is required", errSkillInvalidRequest)
	}

	filterRequest := req.Body.Filter
	if len(filterRequest.AllowArtifacts) == 0 {
		return &GetSkillsPromptResponse{
			Body: &GetSkillsPromptResponseBody{},
		}, nil
	}
	if err := validateArtifactRefs(filterRequest.AllowArtifacts); err != nil {
		return nil, fmt.Errorf("%w: invalid allowArtifacts: %w", errSkillInvalidRequest, err)
	}
	if len(filterRequest.Inserts) > 0 &&
		!containsInstructionInsert(filterRequest.Inserts) {
		return &GetSkillsPromptResponse{
			Body: &GetSkillsPromptResponseBody{},
		}, nil
	}

	resolved, err := s.resolveAllowArtifacts(ctx, filterRequest.AllowArtifacts)
	if err != nil {
		return nil, err
	}
	if len(resolved.AllowDefs) == 0 {
		return &GetSkillsPromptResponse{
			Body: &GetSkillsPromptResponseBody{},
		}, nil
	}

	prompt, err := s.runtime.SkillsPrompt(ctx, &agentskills.SkillFilter{
		Types:          append([]string(nil), filterRequest.Types...),
		LocationPrefix: filterRequest.LocationPrefix,
		AllowSkills:    resolved.AllowDefs,
		SessionID:      filterRequest.SessionID,
		Activity:       filterRequest.Activity,
	})
	if err != nil {
		return nil, err
	}
	return &GetSkillsPromptResponse{
		Body: &GetSkillsPromptResponseBody{Prompt: prompt},
	}, nil
}

func (s *SkillRuntime) ListRuntimeSkills(
	ctx context.Context,
	req *ListRuntimeSkillsRequest,
) (*ListRuntimeSkillsResponse, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, fmt.Errorf("%w: %w", errSkillInvalidRequest, err)
	}
	if req == nil || req.Body == nil || req.Body.Filter == nil {
		return nil, fmt.Errorf("%w: missing filter", errSkillInvalidRequest)
	}

	filterRequest := req.Body.Filter
	if len(filterRequest.AllowArtifacts) == 0 {
		return &ListRuntimeSkillsResponse{
			Body: &ListRuntimeSkillsResponseBody{
				Skills: []RuntimeSkillListItem{},
			},
		}, nil
	}
	if err := validateArtifactRefs(filterRequest.AllowArtifacts); err != nil {
		return nil, fmt.Errorf("%w: invalid allowArtifacts: %w", errSkillInvalidRequest, err)
	}

	activity := filterRequest.Activity
	if activity == "" {
		activity = agentskillsSpec.SkillActivityAny
	}
	if activity == agentskillsSpec.SkillActivityActive &&
		strings.TrimSpace(string(filterRequest.SessionID)) == "" {
		return nil, fmt.Errorf(
			"%w: activity=active requires sessionID",
			errSkillInvalidRequest,
		)
	}

	resolved, err := s.resolveAllowArtifacts(ctx, filterRequest.AllowArtifacts)
	if err != nil {
		return nil, err
	}
	if len(resolved.AllowDefs) == 0 {
		return &ListRuntimeSkillsResponse{
			Body: &ListRuntimeSkillsResponseBody{
				Skills: []RuntimeSkillListItem{},
			},
		}, nil
	}

	records, err := s.runtime.ListSkills(ctx, &agentskills.SkillListFilter{
		Types:          append([]string(nil), filterRequest.Types...),
		LocationPrefix: filterRequest.LocationPrefix,
		AllowSkills:    resolved.AllowDefs,
		Inserts:        append([]agentskillsSpec.SkillInsert(nil), filterRequest.Inserts...),
		SessionID:      filterRequest.SessionID,
		Activity:       activity,
	})
	if err != nil {
		return nil, err
	}

	active := map[agentskillsSpec.SkillDef]struct{}{}
	if filterRequest.SessionID != "" &&
		activity == agentskillsSpec.SkillActivityAny {
		current, err := s.runtime.ListSkills(ctx, &agentskills.SkillListFilter{
			SessionID:   filterRequest.SessionID,
			Activity:    agentskillsSpec.SkillActivityActive,
			AllowSkills: resolved.AllowDefs,
		})
		if err != nil {
			return nil, err
		}
		for _, record := range current {
			active[record.Def] = struct{}{}
		}
	}

	items := make([]RuntimeSkillListItem, 0, len(records))
	for _, record := range records {
		ref, found := resolved.DefToArtifacts[record.Def]
		if !found {
			continue
		}
		_, isActive := active[record.Def]
		items = append(items, RuntimeSkillListItem{
			SkillRef:       ref,
			Type:           record.Def.Type,
			Name:           record.Def.Name,
			DisplayName:    record.DisplayName,
			Description:    record.Description,
			Digest:         record.Digest,
			Insert:         record.Insert,
			Arguments:      append([]agentskillsSpec.SkillArgument(nil), record.Arguments...),
			SourceTags:     append([]string(nil), record.Tags...),
			Resources:      cloneSkillResourceInfo(record.Resources),
			RawFrontmatter: cloneAnyMap(record.RawFrontmatter),
			Warnings:       append([]string(nil), record.Warnings...),
			IsActive: activity == agentskillsSpec.SkillActivityActive ||
				(activity == agentskillsSpec.SkillActivityAny && isActive),
		})
	}
	sort.Slice(items, func(left, right int) bool {
		return artifactRefKey(items[left].SkillRef) <
			artifactRefKey(items[right].SkillRef)
	})
	return &ListRuntimeSkillsResponse{
		Body: &ListRuntimeSkillsResponseBody{Skills: items},
	}, nil
}

func (s *SkillRuntime) RenderSkill(
	ctx context.Context,
	req *RenderSkillRequest,
) (*RenderSkillResponse, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, fmt.Errorf("%w: %w", errSkillInvalidRequest, err)
	}
	if req == nil || req.Body == nil {
		return nil, fmt.Errorf("%w: missing request", errSkillInvalidRequest)
	}
	if err := req.Body.Artifact.Validate(); err != nil {
		return nil, fmt.Errorf("%w: invalid artifact: %w", errSkillInvalidRequest, err)
	}

	resolved, found := s.resolveArtifactSkill(ctx, req.Body.Artifact)
	if !found {
		return nil, ErrSkillNotFound
	}
	out, err := s.runtime.RenderSkill(ctx, agentskills.RenderSkillParams{
		Def:       resolved.Definition,
		Arguments: req.Body.Arguments,
	})
	if err != nil {
		return nil, err
	}
	return &RenderSkillResponse{
		Body: &RenderSkillResponseBody{
			Text:             out.Text,
			Insert:           out.Insert,
			Name:             out.Name,
			Description:      out.Description,
			DisplayName:      out.DisplayName,
			SourceTags:       append([]string(nil), out.Tags...),
			Resources:        cloneSkillResourceInfo(out.Resources),
			Arguments:        append([]agentskillsSpec.SkillArgument(nil), out.Arguments...),
			AppliedArguments: cloneStringMap(out.AppliedArguments),
			RawFrontmatter:   cloneAnyMap(out.RawFrontmatter),
			Warnings:         append([]string(nil), out.Warnings...),
		},
	}, nil
}

// DescribeArtifactSkill resolves and indexes a selected ArtifactRef through
// the ownership router before reading Agent Skills metadata. The Artifact
// record and its Collection membership, rather than reference shape, decide
// the owning feature adapter.
func (s *SkillRuntime) DescribeArtifactSkill(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (ArtifactSkillSummary, error) {
	if err := s.ensureConfigured(); err != nil {
		return ArtifactSkillSummary{}, err
	}
	if err := ref.Validate(); err != nil {
		return ArtifactSkillSummary{}, err
	}

	resolved, found := s.resolveArtifactSkill(ctx, ref)
	if !found {
		return ArtifactSkillSummary{}, fmt.Errorf(
			"%w: skill Artifact %q is unavailable",
			basespec.ErrReferenceUnresolved,
			ref.ArtifactID,
		)
	}

	records, err := s.runtime.ListSkills(ctx, &agentskills.SkillListFilter{
		AllowSkills: []agentskillsSpec.SkillDef{resolved.Definition},
		Activity:    agentskillsSpec.SkillActivityAny,
	})
	if err != nil {
		return ArtifactSkillSummary{}, err
	}

	for _, record := range records {
		if record.Def != resolved.Definition {
			continue
		}
		return ArtifactSkillSummary{
			Artifact:     ref,
			IsEnabled:    true,
			Insert:       record.Insert,
			HasArguments: len(record.Arguments) != 0,
			HasResources: record.Resources.HasResources,
		}, nil
	}

	return ArtifactSkillSummary{}, fmt.Errorf(
		"%w: runtime did not index skill Artifact %q",
		basespec.ErrReferenceUnresolved,
		ref.ArtifactID,
	)
}

func (s *SkillRuntime) InvokeSkillTool(
	ctx context.Context,
	req *InvokeSkillToolRequest,
) (*InvokeSkillToolResponse, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, fmt.Errorf("%w: %w", errSkillInvalidRequest, err)
	}
	if req == nil || req.Body == nil {
		return nil, fmt.Errorf("%w: missing request", errSkillInvalidRequest)
	}
	sessionID := strings.TrimSpace(string(req.Body.SessionID))
	if sessionID == "" {
		return nil, fmt.Errorf("%w: sessionID required", errSkillInvalidRequest)
	}
	toolName := strings.TrimSpace(req.Body.ToolName)
	if toolName == "" {
		return nil, fmt.Errorf("%w: toolName required", errSkillInvalidRequest)
	}
	arguments := strings.TrimSpace(req.Body.Args)
	if arguments == "" {
		arguments = "{}"
	}
	if len(arguments) > maxSkillToolArgsBytes {
		return nil, fmt.Errorf("%w: args too large", errSkillInvalidRequest)
	}
	if !json.Valid([]byte(arguments)) {
		return nil, fmt.Errorf("%w: args must be valid JSON", errSkillInvalidRequest)
	}
	if arguments[0] != '{' {
		return nil, fmt.Errorf("%w: args must be a JSON object", errSkillInvalidRequest)
	}

	registry, err := s.runtime.NewSessionRegistry(ctx, agentskillsSpec.SessionID(sessionID))
	if err != nil {
		return nil, err
	}
	var functionID string
	switch toolName {
	case "skills-load":
		functionID = string(agentskillsSpec.FuncIDSkillsLoad)
	case "skills-unload":
		functionID = string(agentskillsSpec.FuncIDSkillsUnload)
	case "skills-readresource":
		functionID = string(agentskillsSpec.FuncIDSkillsReadResource)
	case "skills-runscript":
		if !s.runScriptsEnabled {
			return nil, fmt.Errorf(
				"%w: skills-runscript is disabled by runtime policy",
				errSkillInvalidRequest,
			)
		}
		functionID = string(agentskillsSpec.FuncIDSkillsRunScript)
	default:
		return nil, fmt.Errorf("%w: unknown toolName %q", errSkillInvalidRequest, toolName)
	}

	outputs, callErr := llmtoolsutil.CallUsingRegistry(ctx, registry, functionID, json.RawMessage(arguments))
	response := &InvokeSkillToolResponse{Body: &InvokeSkillToolResponseBody{
		Outputs:   outputs,
		Meta:      map[string]any{"toolName": toolName},
		IsBuiltIn: true,
	}}
	if callErr != nil {
		response.Body.IsError = true
		response.Body.ErrorMessage = callErr.Error()
	}
	return response, nil
}

func (s *SkillRuntime) ensureConfigured() error {
	if s == nil {
		return errors.New("skill runtime is not configured")
	}
	s.lifecycleMu.RLock()
	closed := s.closed
	configured := s.resolver != nil && s.runtime != nil
	s.lifecycleMu.RUnlock()
	if closed || !configured {
		return errors.New("skill runtime is not configured")
	}
	return nil
}

func (s *SkillRuntime) isClosed() bool {
	s.lifecycleMu.RLock()
	defer s.lifecycleMu.RUnlock()
	return s.closed
}

type resolvedAllowArtifacts struct {
	DefToArtifacts map[agentskillsSpec.SkillDef]artifact.ArtifactRef
	RefToDef       map[string]agentskillsSpec.SkillDef
	AllowDefs      []agentskillsSpec.SkillDef
}

func (s *SkillRuntime) resolveAllowArtifacts(
	ctx context.Context,
	refs []artifact.ArtifactRef,
) (resolvedAllowArtifacts, error) {
	output := resolvedAllowArtifacts{
		DefToArtifacts: map[agentskillsSpec.SkillDef]artifact.ArtifactRef{},
		RefToDef:       map[string]agentskillsSpec.SkillDef{},
	}

	seenRefs := map[string]struct{}{}
	byName := map[string][]ResolvedArtifactSkill{}
	resynced := map[collection.CollectionRef]error{}
	unavailable := make([]artifact.ArtifactRef, 0)
	for _, ref := range refs {
		if err := ref.Validate(); err != nil {
			continue
		}
		key := artifactRefKey(ref)
		if _, exists := seenRefs[key]; exists {
			continue
		}
		seenRefs[key] = struct{}{}

		value, found := s.resolveArtifactSkillWithResync(
			ctx,
			ref,
			resynced,
		)
		if !found {
			unavailable = append(unavailable, ref)
			continue
		}
		byName[value.Definition.Name] = append(
			byName[value.Definition.Name],
			value,
		)
	}
	if len(unavailable) != 0 {
		return output, unavailableArtifactSkillsError(unavailable)
	}

	collisions := make([]string, 0)
	for name, values := range byName {
		if len(values) != 1 {
			collisions = append(collisions, name)
			continue
		}
		value := values[0]
		output.DefToArtifacts[value.Definition] = value.Artifact
		output.RefToDef[artifactRefKey(value.Artifact)] = value.Definition
		output.AllowDefs = append(output.AllowDefs, value.Definition)
	}
	if len(collisions) != 0 {
		sort.Strings(collisions)
		return output, fmt.Errorf(
			"%w: ambiguous runtime Skill names: %s",
			basespec.ErrConflict,
			strings.Join(collisions, ", "),
		)
	}
	sortSkillDefs(output.AllowDefs)
	return output, nil
}

func (s *SkillRuntime) resolveArtifactSkill(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (ResolvedArtifactSkill, bool) {
	return s.resolveArtifactSkillWithResync(ctx, ref, nil)
}

func (s *SkillRuntime) registrationMatches(
	value ResolvedArtifactSkill,
) bool {
	s.rtResyncMu.Lock()
	defer s.rtResyncMu.Unlock()

	version, found := s.managedRuntime[value.Definition]
	return found && version == value.Version
}

// resolveArtifactSkillWithResync keeps request-local synchronization state
// only. It is not a cache of Artifact Store state: every Artifact is resolved
// again after its owning Collection has been reconciled.
func (s *SkillRuntime) resolveArtifactSkillWithResync(
	ctx context.Context,
	ref artifact.ArtifactRef,
	resynced map[collection.CollectionRef]error,
) (ResolvedArtifactSkill, bool) {
	value, err := s.resolver.ResolveArtifactSkill(ctx, ref)
	if err != nil {
		return ResolvedArtifactSkill{}, false
	}

	if resynced == nil {
		if err := s.ResyncCollection(ctx, value.Collection); err != nil {
			return ResolvedArtifactSkill{}, false
		}
	} else if previous, found := resynced[value.Collection]; found {
		if previous != nil {
			return ResolvedArtifactSkill{}, false
		}
	} else {
		err := s.ResyncCollection(ctx, value.Collection)
		resynced[value.Collection] = err
		if err != nil {
			return ResolvedArtifactSkill{}, false
		}
	}

	if s.registrationMatches(value) {
		return value, true
	}

	refreshed, err := s.resolver.ResolveArtifactSkill(ctx, ref)
	if err != nil || refreshed.Collection != value.Collection {
		return ResolvedArtifactSkill{}, false
	}
	if !s.registrationMatches(refreshed) {
		return ResolvedArtifactSkill{}, false
	}
	return refreshed, true
}

func unavailableArtifactSkillsError(
	refs []artifact.ArtifactRef,
) error {
	sort.Slice(refs, func(left, right int) bool {
		return artifactRefKey(refs[left]) < artifactRefKey(refs[right])
	})
	values := make([]string, 0, len(refs))
	for _, ref := range refs {
		values = append(values, artifactRefKey(ref))
	}
	return fmt.Errorf(
		"%w: unavailable Artifact Skills: %s",
		basespec.ErrReferenceUnresolved,
		strings.Join(values, ", "),
	)
}

func validateArtifactRefs(values []artifact.ArtifactRef) error {
	seen := map[string]struct{}{}
	for _, value := range values {
		if err := value.Validate(); err != nil {
			return err
		}
		key := artifactRefKey(value)
		if _, duplicate := seen[key]; duplicate {
			return errors.New("duplicate ArtifactRef")
		}
		seen[key] = struct{}{}
	}
	return nil
}

func subsetArtifacts(
	allow []artifact.ArtifactRef,
	active []artifact.ArtifactRef,
) []artifact.ArtifactRef {
	allowed := map[string]struct{}{}
	for _, ref := range allow {
		allowed[artifactRefKey(ref)] = struct{}{}
	}

	seen := map[string]struct{}{}
	output := make([]artifact.ArtifactRef, 0, len(active))
	for _, ref := range active {
		key := artifactRefKey(ref)
		if _, exists := allowed[key]; !exists {
			continue
		}
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		output = append(output, ref)
	}
	return output
}

func buildActiveArtifacts(
	definitions map[agentskillsSpec.SkillDef]artifact.ArtifactRef,
	active map[agentskillsSpec.SkillDef]struct{},
) []artifact.ArtifactRef {
	output := make([]artifact.ArtifactRef, 0)
	for definition := range active {
		ref, found := definitions[definition]
		if found {
			output = append(output, ref)
		}
	}
	sort.Slice(output, func(left, right int) bool {
		return artifactRefKey(output[left]) < artifactRefKey(output[right])
	})
	return output
}

func containsInstructionInsert(
	values []agentskillsSpec.SkillInsert,
) bool {
	return slices.Contains(values, agentskillsSpec.SkillInsertInstructions)
}

func cloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	output := make(map[string]string, len(input))
	maps.Copy(output, input)
	return output
}

func cloneAnyMap(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	output := make(map[string]any, len(input))
	maps.Copy(output, input)
	return output
}

func cloneSkillResourceInfo(
	input agentskillsSpec.SkillResourceInfo,
) agentskillsSpec.SkillResourceInfo {
	input.Locations = append([]string(nil), input.Locations...)
	return input
}
