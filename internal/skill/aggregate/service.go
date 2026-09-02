package aggregate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/flexigpt/agentskills-go/document"
	"github.com/flexigpt/agentskills-go/provider"

	"github.com/flexigpt/agentskills-go/runtime"
	agentskillsRuntimeSpec "github.com/flexigpt/agentskills-go/runtime/spec"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/llmtoolsutil"
	skillRuntime "github.com/flexigpt/flexigpt-app/internal/skill/runtime"
	skillStore "github.com/flexigpt/flexigpt-app/internal/skill/store"
)

var errSkillInvalidRequest = errors.New("invalid request")

// Service translates Artifact Store identities into runtime-owned terms.
// Agent Skills state and lifecycle are owned by runtime.Service.
type Service struct {
	resolver    *skillStore.ArtifactRouter
	runtime     *skillRuntime.Service
	lifecycleMu sync.RWMutex
	closed      bool
}

func New(
	resolver *skillStore.ArtifactRouter,
	runtimeService *skillRuntime.Service,
) (*Service, error) {
	if resolver == nil {
		return nil, errors.New("artifact skill resolver is required")
	}
	if runtimeService == nil {
		return nil, errors.New("skill runtime service is required")
	}

	return &Service{
		resolver: resolver,
		runtime:  runtimeService,
	}, nil
}

func (s *Service) AgentSkillsRuntime() *runtime.Runtime {
	if s == nil || s.isClosed() {
		return nil
	}
	return s.runtime.AgentSkillsRuntime()
}

func (s *Service) RunScriptsEnabled() bool {
	if s == nil || s.isClosed() {
		return false
	}
	return s.runtime.SupportsRunScript()
}

func (s *Service) Close() {
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
}

func (s *Service) CreateSkillSession(
	ctx context.Context,
	req *CreateSkillSessionRequest,
) (*CreateSkillSessionResponse, error) {
	if err := s.ensureConfigured(); err != nil {
		return nil, fmt.Errorf("%w: %w", errSkillInvalidRequest, err)
	}
	if req == nil || req.Body == nil {
		return nil, fmt.Errorf("%w: missing request", errSkillInvalidRequest)
	}
	allowedConfigured := req.Body.AllowArtifacts != nil
	if allowedConfigured {
		if err := validateArtifactRefs(req.Body.AllowArtifacts); err != nil {
			return nil, fmt.Errorf("%w: invalid allowArtifacts: %w", errSkillInvalidRequest, err)
		}
	}
	if err := validateArtifactRefs(req.Body.ActiveArtifacts); err != nil {
		return nil, fmt.Errorf("%w: invalid activeArtifacts: %w", errSkillInvalidRequest, err)
	}

	previousSessionID := strings.TrimSpace(string(req.Body.CloseSessionID))

	refsToResolve := req.Body.AllowArtifacts
	if !allowedConfigured {
		refsToResolve = req.Body.ActiveArtifacts
	}
	resolved, err := s.resolveAllowArtifacts(ctx, refsToResolve)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errSkillInvalidRequest, err)
	}
	activeRefs := req.Body.ActiveArtifacts
	if allowedConfigured {
		activeRefs = subsetArtifacts(
			req.Body.AllowArtifacts,
			req.Body.ActiveArtifacts,
		)
	}
	activeDefinitions := map[provider.SkillDef]struct{}{}
	for _, ref := range activeRefs {
		if definition, found := resolved.RefToDef[artifactRefKey(ref)]; found {
			activeDefinitions[definition] = struct{}{}
		}
	}

	activeDefs := make([]provider.SkillDef, 0, len(activeDefinitions))
	for definition := range activeDefinitions {
		activeDefs = append(activeDefs, definition)
	}
	sortSkillDefs(activeDefs)

	options := []runtime.SessionOption{}
	if req.Body.MaxActivePerSession > 0 {
		options = append(
			options,
			runtime.WithSessionMaxActivePerSession(
				req.Body.MaxActivePerSession,
			),
		)
	}
	if allowedConfigured {
		// The option is omitted only when the caller omitted allowArtifacts.
		// A configured empty list intentionally creates a deny-all session.
		options = append(
			options,
			runtime.WithSessionAllowedSkills(resolved.AllowDefs),
		)
	}
	if len(activeDefs) > 0 {
		options = append(
			options,
			runtime.WithSessionActiveSkills(activeDefs),
		)
	}

	sessionID, _, err := s.runtime.NewSession(ctx, options...)
	if err != nil {
		return nil, err
	}

	records, err := s.runtime.ListAgentSkills(ctx, &runtime.SkillListFilter{
		SessionID:   sessionID,
		Activity:    agentskillsRuntimeSpec.SkillActivityActive,
		AllowSkills: resolved.AllowDefs,
	})
	if err != nil {
		closeErr := s.runtime.CloseSession(
			context.WithoutCancel(ctx),
			sessionID,
		)
		return nil, errors.Join(err, closeErr)
	}

	active := map[provider.SkillDef]struct{}{}
	for _, record := range records {
		active[record.Def] = struct{}{}
	}

	// Preserve the old session until the replacement has been created and
	// inspected successfully. Closing the old session is explicitly best
	// effort and must not invalidate a successful new session response.
	if previousSessionID != "" {
		_ = s.runtime.CloseSession(context.WithoutCancel(ctx), agentskillsRuntimeSpec.SessionID(previousSessionID))
	}

	return &CreateSkillSessionResponse{
		Body: &CreateSkillSessionResponseBody{
			SessionID:       sessionID,
			ActiveArtifacts: buildActiveArtifacts(resolved.DefToArtifacts, active),
		},
	}, nil
}

func (s *Service) CloseSkillSession(
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

func (s *Service) GetSkillsPrompt(
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
	resolved := resolvedAllowArtifacts{}
	if len(filterRequest.AllowArtifacts) != 0 {
		if err := validateArtifactRefs(filterRequest.AllowArtifacts); err != nil {
			return nil, fmt.Errorf("%w: invalid allowArtifacts: %w", errSkillInvalidRequest, err)
		}
		var err error
		resolved, err = s.resolveAllowArtifacts(ctx, filterRequest.AllowArtifacts)
		if err != nil {
			return nil, err
		}
	}
	if len(filterRequest.Inserts) > 0 &&
		!containsInstructionInsert(filterRequest.Inserts) {
		return &GetSkillsPromptResponse{
			Body: &GetSkillsPromptResponseBody{},
		}, nil
	}

	prompt, err := s.runtime.SkillsPrompt(ctx, &runtime.SkillFilter{
		Types:          append([]string(nil), filterRequest.Types...),
		LocationPrefix: filterRequest.LocationPrefix,
		NamePrefix:     filterRequest.NamePrefix,
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

func (s *Service) ListRuntimeSkills(
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
		activity = agentskillsRuntimeSpec.SkillActivityAny
	}
	if activity == agentskillsRuntimeSpec.SkillActivityActive &&
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

	records, err := s.runtime.ListAgentSkills(ctx, &runtime.SkillListFilter{
		Types:          append([]string(nil), filterRequest.Types...),
		LocationPrefix: filterRequest.LocationPrefix,
		AllowSkills:    resolved.AllowDefs,
		Inserts:        append([]document.SkillInsert(nil), filterRequest.Inserts...),
		SessionID:      filterRequest.SessionID,
		Activity:       activity,
	})
	if err != nil {
		return nil, err
	}

	active := map[provider.SkillDef]struct{}{}
	if filterRequest.SessionID != "" &&
		activity == agentskillsRuntimeSpec.SkillActivityAny {
		current, err := s.runtime.ListAgentSkills(ctx, &runtime.SkillListFilter{
			SessionID:   filterRequest.SessionID,
			Activity:    agentskillsRuntimeSpec.SkillActivityActive,
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
			Arguments:      append([]document.SkillArgument(nil), record.Arguments...),
			SourceTags:     append([]string(nil), record.Tags...),
			Resources:      cloneSkillResourceInfo(record.Resources),
			RawFrontmatter: cloneAnyMap(record.RawFrontmatter),
			Warnings:       append([]string(nil), record.Warnings...),
			IsActive: activity == agentskillsRuntimeSpec.SkillActivityActive ||
				(activity == agentskillsRuntimeSpec.SkillActivityAny && isActive),
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

func (s *Service) RenderSkill(
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
	out, err := s.runtime.RenderAgentSkill(ctx, runtime.RenderSkillParams{
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
			Arguments:        append([]document.SkillArgument(nil), out.Arguments...),
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
func (s *Service) DescribeArtifactSkill(
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
		if _, resolveErr := s.resolver.ResolveArtifactSkill(ctx, ref); resolveErr != nil {
			return ArtifactSkillSummary{}, fmt.Errorf(
				"%w: skill Artifact %q is unavailable: %w",
				basespec.ErrReferenceUnresolved,
				ref.ArtifactID,
				resolveErr,
			)
		}
		return ArtifactSkillSummary{}, fmt.Errorf(
			"%w: skill Artifact %q could not be registered in the Skill runtime",
			basespec.ErrReferenceUnresolved,
			ref.ArtifactID,
		)
	}

	records, err := s.runtime.ListAgentSkills(ctx, &runtime.SkillListFilter{
		AllowSkills: []provider.SkillDef{resolved.Definition},
		Activity:    agentskillsRuntimeSpec.SkillActivityAny,
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

func (s *Service) InvokeSkillTool(
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

	registry, err := s.runtime.NewSessionRegistry(ctx, agentskillsRuntimeSpec.SessionID(sessionID))
	if err != nil {
		return nil, err
	}
	var functionID string
	switch toolName {
	case "skills-load":
		functionID = string(agentskillsRuntimeSpec.FuncIDSkillsLoad)
	case "skills-unload":
		functionID = string(agentskillsRuntimeSpec.FuncIDSkillsUnload)
	case "skills-readresource":
		functionID = string(agentskillsRuntimeSpec.FuncIDSkillsReadResource)
	case "skills-runscript":
		if !s.RunScriptsEnabled() {
			return nil, fmt.Errorf(
				"%w: skills-runscript is disabled by runtime policy",
				errSkillInvalidRequest,
			)
		}
		functionID = string(agentskillsRuntimeSpec.FuncIDSkillsRunScript)
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

func (s *Service) ensureConfigured() error {
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

func (s *Service) isClosed() bool {
	s.lifecycleMu.RLock()
	defer s.lifecycleMu.RUnlock()
	return s.closed
}

type resolvedAllowArtifacts struct {
	DefToArtifacts map[provider.SkillDef]artifact.ArtifactRef
	RefToDef       map[string]provider.SkillDef
	AllowDefs      []provider.SkillDef
}

func (s *Service) resolveAllowArtifacts(
	ctx context.Context,
	refs []artifact.ArtifactRef,
) (resolvedAllowArtifacts, error) {
	output := resolvedAllowArtifacts{
		DefToArtifacts: map[provider.SkillDef]artifact.ArtifactRef{},
		RefToDef:       map[string]provider.SkillDef{},
	}

	seenRefs := map[string]struct{}{}
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
		if previous, exists := output.DefToArtifacts[value.Definition]; exists && previous != value.Artifact {
			return output, fmt.Errorf(
				"%w: Artifact Skills %q and %q resolve to the same runtime definition",
				basespec.ErrConflict,
				previous.ArtifactID,
				value.Artifact.ArtifactID,
			)
		}

		output.DefToArtifacts[value.Definition] = value.Artifact
		output.RefToDef[artifactRefKey(value.Artifact)] = value.Definition
		output.AllowDefs = append(output.AllowDefs, value.Definition)
	}
	if len(unavailable) != 0 {
		return output, unavailableArtifactSkillsError(unavailable)
	}

	sortSkillDefs(output.AllowDefs)
	return output, nil
}

func (s *Service) resolveArtifactSkill(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (skillStore.ResolvedArtifactSkill, bool) {
	return s.resolveArtifactSkillWithResync(
		ctx,
		ref,
		map[collection.CollectionRef]error{},
	)
}

func (s *Service) resolveArtifactSkillWithResync(
	ctx context.Context,
	ref artifact.ArtifactRef,
	resynced map[collection.CollectionRef]error,
) (skillStore.ResolvedArtifactSkill, bool) {
	collectionRef, err := s.resolver.CollectionForArtifact(ctx, ref)
	if err != nil {
		return skillStore.ResolvedArtifactSkill{}, false
	}

	if previous, found := resynced[collectionRef]; found {
		if previous != nil {
			return skillStore.ResolvedArtifactSkill{}, false
		}
	} else {
		err := s.ResyncCollection(ctx, collectionRef)
		if resynced != nil {
			resynced[collectionRef] = err
		}
		if err != nil {
			return skillStore.ResolvedArtifactSkill{}, false
		}
	}

	value, err := s.resolver.ResolveArtifactSkill(ctx, ref)
	if err != nil || value.Collection != collectionRef {
		return skillStore.ResolvedArtifactSkill{}, false
	}
	if !s.registrationMatches(value) {
		return skillStore.ResolvedArtifactSkill{}, false
	}
	return value, true
}

func (s *Service) registrationMatches(
	value skillStore.ResolvedArtifactSkill,
) bool {
	return s.runtime.IsRegistered(skillRuntime.SkillRegistration{
		Definition: value.Definition,
		Revision:   value.Version,
	})
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
	definitions map[provider.SkillDef]artifact.ArtifactRef,
	active map[provider.SkillDef]struct{},
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
	values []document.SkillInsert,
) bool {
	return slices.Contains(values, document.SkillInsertInstructions)
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
	input provider.SkillResourceInfo,
) provider.SkillResourceInfo {
	input.Locations = append([]string(nil), input.Locations...)
	return input
}

func artifactRefKey(ref artifact.ArtifactRef) string {
	return string(ref.RootID) + "\x00" + string(ref.ArtifactID)
}

func sortSkillDefs(values []provider.SkillDef) {
	sort.Slice(values, func(left, right int) bool {
		if values[left].Type != values[right].Type {
			return values[left].Type < values[right].Type
		}
		if values[left].Name != values[right].Name {
			return values[left].Name < values[right].Name
		}
		return values[left].Location < values[right].Location
	})
}
