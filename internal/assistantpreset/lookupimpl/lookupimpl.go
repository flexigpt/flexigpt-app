package lookupimpl

import (
	"context"
	"errors"
	"fmt"
	"strings"

	assistantpresetSpec "github.com/flexigpt/flexigpt-app/internal/assistantpreset/spec"
	assistantpresetStore "github.com/flexigpt/flexigpt-app/internal/assistantpreset/store"
	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/policy"
	mcpRuntime "github.com/flexigpt/flexigpt-app/internal/mcp/runtime"
	mcpAuth "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/auth"
	mcpSpec "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/spec"
	mcpStoreServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/server"
	modelpresetSpec "github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
	modelpresetStore "github.com/flexigpt/flexigpt-app/internal/modelpreset/store"
	skillAggregate "github.com/flexigpt/flexigpt-app/internal/skill/aggregate"
	toolSpec "github.com/flexigpt/flexigpt-app/internal/tool/spec"
	toolStore "github.com/flexigpt/flexigpt-app/internal/tool/store"
)

type modelPresetLookupAdapter struct {
	store *modelpresetStore.ModelPresetStore
}

func (a *modelPresetLookupAdapter) GetModelPresetSummary(
	ctx context.Context,
	ref modelpresetSpec.ModelPresetRef,
) (assistantpresetStore.ModelPresetSummary, error) {
	if a == nil || a.store == nil {
		return assistantpresetStore.ModelPresetSummary{}, errors.New("model preset lookup adapter is not configured")
	}

	if ref.IsZero() {
		return assistantpresetStore.ModelPresetSummary{}, errors.New("model preset ref is zero")
	}

	resp, err := a.store.GetModelPreset(ctx, &modelpresetSpec.GetModelPresetRequest{
		ProviderName:    ref.ProviderName,
		ModelPresetID:   ref.ModelPresetID,
		IncludeDisabled: true,
	})
	if err != nil {
		return assistantpresetStore.ModelPresetSummary{}, err
	}
	if resp == nil || resp.Body == nil {
		return assistantpresetStore.ModelPresetSummary{}, errors.New("empty model preset response")
	}

	return assistantpresetStore.ModelPresetSummary{
		IsEnabled: resp.Body.Provider.IsEnabled && resp.Body.Model.IsEnabled,
	}, nil
}

type toolSelectionLookupAdapter struct {
	store *toolStore.ToolStore
}

func (a *toolSelectionLookupAdapter) GetToolSummaryForSelection(
	ctx context.Context,
	selection toolSpec.ToolSelection,
) (assistantpresetStore.ToolSummary, error) {
	if a == nil || a.store == nil {
		return assistantpresetStore.ToolSummary{}, errors.New("tool selection lookup adapter is not configured")
	}
	if selection.ToolRef.BundleID == "" || selection.ToolRef.ToolSlug == "" || selection.ToolRef.ToolVersion == "" {
		return assistantpresetStore.ToolSummary{}, errors.New("tool selection toolRef is incomplete")
	}

	bundleEnabled, err := getToolBundleEnabled(ctx, a.store, selection.ToolRef.BundleID)
	if err != nil {
		return assistantpresetStore.ToolSummary{}, err
	}

	resp, err := a.store.GetTool(ctx, &toolSpec.GetToolRequest{
		BundleID: selection.ToolRef.BundleID,
		ToolSlug: selection.ToolRef.ToolSlug,
		Version:  selection.ToolRef.ToolVersion,
	})
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "disabled") {
			return assistantpresetStore.ToolSummary{IsEnabled: false}, nil
		}
		return assistantpresetStore.ToolSummary{}, err
	}
	if resp == nil || resp.Body == nil {
		return assistantpresetStore.ToolSummary{}, errors.New("empty tool response")
	}

	return assistantpresetStore.ToolSummary{
		IsEnabled: bundleEnabled && resp.Body.IsEnabled,
	}, nil
}

type skillLookupAdapter struct {
	runtime *skillAggregate.Service
}

func (a *skillLookupAdapter) GetSkillSummaryForSelection(
	ctx context.Context,
	selection assistantpresetSpec.ArtifactSkillSelection,
) (assistantpresetStore.SkillSummary, error) {
	if a == nil || a.runtime == nil {
		return assistantpresetStore.SkillSummary{}, errors.New("skill lookup adapter is not configured")
	}
	if err := selection.Artifact.Validate(); err != nil {
		return assistantpresetStore.SkillSummary{}, err
	}

	summary, err := a.runtime.DescribeArtifactSkill(ctx, selection.Artifact)
	if err != nil {
		return assistantpresetStore.SkillSummary{}, fmt.Errorf(
			"artifact skill %q: %w",
			selection.Artifact.ArtifactID,
			err,
		)
	}

	return assistantpresetStore.SkillSummary{
		IsEnabled:    summary.IsEnabled,
		Insert:       summary.Insert,
		HasArguments: summary.HasArguments,
		HasResources: summary.HasResources,
	}, nil
}

type MCPServerResolver interface {
	ResolveMCPServer(
		ctx context.Context,
		ref mcpSpec.ServerID,
	) (mcpStoreServer.Resolved, error)
}

type MCPDiscoveryLookup interface {
	ListTools(
		ctx context.Context,
		server mcpSpec.ServerID,
	) ([]mcpRuntime.MCPToolCapability, error)

	ListResources(
		ctx context.Context,
		server mcpSpec.ServerID,
	) ([]mcpRuntime.MCPResourceRef, error)

	ListResourceTemplates(
		ctx context.Context,
		server mcpSpec.ServerID,
	) ([]mcpRuntime.MCPResourceTemplateRef, error)

	ListPrompts(
		ctx context.Context,
		server mcpSpec.ServerID,
	) ([]mcpRuntime.MCPPromptRef, error)
}

type mcpContextLookupAdapter struct {
	resolver  MCPServerResolver
	discovery MCPDiscoveryLookup
}

func NewAssistantPresetReferenceLookups(
	modelPresetSt *modelpresetStore.ModelPresetStore,
	toolSt *toolStore.ToolStore,
	skillRt *skillAggregate.Service,
	mcpResolver MCPServerResolver,
	mcpDiscovery MCPDiscoveryLookup,
) assistantpresetStore.ReferenceLookups {
	lookups := assistantpresetStore.ReferenceLookups{
		ModelPresets: &modelPresetLookupAdapter{
			store: modelPresetSt,
		},
		ToolSelections: &toolSelectionLookupAdapter{
			store: toolSt,
		},
		Skills: &skillLookupAdapter{
			runtime: skillRt,
		},
	}
	if mcpResolver != nil {
		lookups.MCPContext = NewMCPContextLookup(mcpResolver, mcpDiscovery)
	}
	return lookups
}

func NewMCPContextLookup(
	resolver MCPServerResolver,
	discovery MCPDiscoveryLookup,
) assistantpresetStore.MCPContextLookup {
	return &mcpContextLookupAdapter{
		resolver:  resolver,
		discovery: discovery,
	}
}

func (a *mcpContextLookupAdapter) ValidateMCPConversationContext(
	ctx context.Context,
	mcpContext mcpRuntime.MCPConversationContext,
) error {
	if a == nil || a.resolver == nil {
		return errors.New("mcp context lookup adapter is not configured")
	}

	if err := a.validateMCPServers(ctx, mcpContext); err != nil {
		return err
	}

	if a.discovery == nil {
		return nil
	}

	if err := a.validateSelectedMCPTools(ctx, mcpContext); err != nil {
		return err
	}
	if err := a.validateSelectedMCPResources(ctx, mcpContext); err != nil {
		return err
	}
	if err := a.validateSelectedMCPResourceTemplates(ctx, mcpContext); err != nil {
		return err
	}
	if err := a.validateSelectedMCPPrompts(ctx, mcpContext); err != nil {
		return err
	}

	return nil
}

func (a *mcpContextLookupAdapter) validateMCPServers(
	ctx context.Context,
	mcpContext mcpRuntime.MCPConversationContext,
) error {
	for _, ref := range collectMCPServerRefs(mcpContext) {
		resolved, err := a.resolver.ResolveMCPServer(ctx, ref)
		if err != nil {
			return fmt.Errorf("server %q: %w", ref, err)
		}
		if !resolved.RuntimeEnabled {
			return fmt.Errorf(
				"server %q: referenced MCP server is disabled for this installation",
				ref,
			)
		}
	}
	return nil
}

func (a *mcpContextLookupAdapter) validateSelectedMCPTools(
	ctx context.Context,
	mcpContext mcpRuntime.MCPConversationContext,
) error {
	for i, server := range mcpContext.Servers {
		if server.ToolExposure != mcpRuntime.MCPToolExposureSelected {
			continue
		}

		tools, err := a.discovery.ListTools(ctx, server.Server)
		if err != nil {
			if isOptionalMCPDiscoveryError(err) {
				continue
			}
			return fmt.Errorf("servers[%d]: %w", i, err)
		}

		byName := make(map[string]mcpRuntime.MCPToolCapability, len(tools))
		for _, tool := range tools {
			if tool.ToolName != "" {
				byName[tool.ToolName] = tool
			}
		}

		for j, selected := range server.SelectedTools {
			current, exists := byName[selected.ToolName]
			if !exists {
				return fmt.Errorf(
					"servers[%d].selectedTools[%d]: MCP tool %q was not found in current discovery",
					i,
					j,
					selected.ToolName,
				)
			}
			if selected.Digest != "" &&
				selected.Digest != current.Digest {
				return fmt.Errorf(
					"servers[%d].selectedTools[%d]: MCP tool %q digest changed",
					i,
					j,
					selected.ToolName,
				)
			}
			if selected.ProviderToolName != "" &&
				selected.ProviderToolName != current.ProviderToolName {
				return fmt.Errorf(
					"servers[%d].selectedTools[%d]: MCP tool %q provider identity changed",
					i,
					j,
					selected.ToolName,
				)
			}
			if selected.ChoiceID != "" &&
				selected.ChoiceID != current.ChoiceID {
				return fmt.Errorf(
					"servers[%d].selectedTools[%d]: MCP tool %q choice identity changed",
					i,
					j,
					selected.ToolName,
				)
			}
			if !current.Enabled {
				return fmt.Errorf(
					"servers[%d].selectedTools[%d]: MCP tool %q is disabled",
					i,
					j,
					selected.ToolName,
				)
			}
			if selected.ApprovalRule != nil &&
				mcpPolicy.ApprovalRuleRank(*selected.ApprovalRule) < mcpPolicy.ApprovalRuleRank(current.ApprovalRule) {
				return fmt.Errorf(
					"servers[%d].selectedTools[%d]: approval override weakens policy",
					i,
					j,
				)
			}
			if selected.ExecutionMode != nil &&
				current.ExecutionMode == mcpPolicy.MCPExecutionModeManual &&
				*selected.ExecutionMode == mcpPolicy.MCPExecutionModeAuto {
				return fmt.Errorf(
					"servers[%d].selectedTools[%d]: execution override weakens policy",
					i,
					j,
				)
			}
		}
	}
	return nil
}

func (a *mcpContextLookupAdapter) validateSelectedMCPResources(
	ctx context.Context,
	mcpContext mcpRuntime.MCPConversationContext,
) error {
	cache := map[mcpSpec.ServerID]map[string]mcpRuntime.MCPResourceRef{}

	for i, selected := range mcpContext.Resources {
		byURI, exists := cache[selected.Server]
		if !exists {
			resources, err := a.discovery.ListResources(ctx, selected.Server)
			if err != nil {
				if isOptionalMCPDiscoveryError(err) {
					continue
				}
				return fmt.Errorf("resources[%d]: %w", i, err)
			}
			byURI = make(map[string]mcpRuntime.MCPResourceRef, len(resources))

			for _, resource := range resources {
				if resource.URI != "" {
					byURI[resource.URI] = resource
				}
			}
			cache[selected.Server] = byURI
		}

		current, ok := byURI[selected.URI]
		if !ok {
			return fmt.Errorf(
				"resources[%d]: MCP resource %q was not found in current discovery",
				i,
				selected.URI,
			)
		}
		if selected.Digest != "" && selected.Digest != current.Digest {
			return fmt.Errorf(
				"resources[%d]: MCP resource %q digest changed",
				i,
				selected.URI,
			)
		}
	}
	return nil
}

func (a *mcpContextLookupAdapter) validateSelectedMCPResourceTemplates(
	ctx context.Context,
	mcpContext mcpRuntime.MCPConversationContext,
) error {
	cache := map[mcpSpec.ServerID]map[string]mcpRuntime.MCPResourceTemplateRef{}

	for i, selected := range mcpContext.ResourceTemplates {
		ref := selected.MCPResourceTemplateRef
		byTemplate, exists := cache[ref.Server]
		if !exists {
			templates, err := a.discovery.ListResourceTemplates(ctx, ref.Server)
			if err != nil {
				if isOptionalMCPDiscoveryError(err) {
					continue
				}
				return fmt.Errorf("resourceTemplates[%d]: %w", i, err)
			}
			byTemplate = make(map[string]mcpRuntime.MCPResourceTemplateRef, len(templates))
			for _, tmpl := range templates {
				if tmpl.URITemplate != "" {
					byTemplate[tmpl.URITemplate] = tmpl
				}
			}
			cache[ref.Server] = byTemplate
		}

		current, ok := byTemplate[ref.URITemplate]
		if !ok {
			return fmt.Errorf(
				"resourceTemplates[%d]: MCP resource template %q was not found in current discovery",
				i,
				ref.URITemplate,
			)
		}
		if ref.Digest != "" && ref.Digest != current.Digest {
			return fmt.Errorf(
				"resourceTemplates[%d]: MCP resource template %q digest changed",
				i,
				ref.URITemplate,
			)
		}
		if err := validateRequiredMCPArgumentsForLookup(current.Arguments, selected.ArgumentValues); err != nil {
			return fmt.Errorf("resourceTemplates[%d].argumentValues: %w", i, err)
		}
	}
	return nil
}

func (a *mcpContextLookupAdapter) validateSelectedMCPPrompts(
	ctx context.Context,
	mcpContext mcpRuntime.MCPConversationContext,
) error {
	cache := map[mcpSpec.ServerID]map[string]mcpRuntime.MCPPromptRef{}

	for i, selected := range mcpContext.Prompts {
		byName, exists := cache[selected.Server]
		if !exists {
			prompts, err := a.discovery.ListPrompts(ctx, selected.Server)
			if err != nil {
				if isOptionalMCPDiscoveryError(err) {
					continue
				}
				return fmt.Errorf("prompts[%d]: %w", i, err)
			}
			byName = make(map[string]mcpRuntime.MCPPromptRef, len(prompts))
			for _, prompt := range prompts {
				if prompt.PromptName != "" {
					byName[prompt.PromptName] = prompt
				}
			}
			cache[selected.Server] = byName
		}

		current, ok := byName[selected.PromptName]
		if !ok {
			return fmt.Errorf(
				"prompts[%d]: MCP prompt %q was not found in current discovery",
				i,
				selected.PromptName,
			)
		}
		if selected.Digest != "" && selected.Digest != current.Digest {
			return fmt.Errorf(
				"prompts[%d]: MCP prompt %q digest changed",
				i,
				selected.PromptName,
			)
		}
		if err := validateRequiredMCPArgumentsForLookup(current.Arguments, selected.ArgumentValues); err != nil {
			return fmt.Errorf("prompts[%d].argumentValues: %w", i, err)
		}
	}
	return nil
}

func collectMCPServerRefs(
	mcpContext mcpRuntime.MCPConversationContext,
) []mcpSpec.ServerID {
	seen := map[mcpSpec.ServerID]struct{}{}
	refs := make([]mcpSpec.ServerID, 0, len(mcpContext.Servers))
	add := func(ref mcpSpec.ServerID) {
		if err := ref.Validate(); err != nil {
			return
		}
		if _, exists := seen[ref]; exists {
			return
		}
		seen[ref] = struct{}{}
		refs = append(refs, ref)
	}

	for _, server := range mcpContext.Servers {
		add(server.Server)
	}
	for _, resource := range mcpContext.Resources {
		add(resource.Server)
	}
	for _, tmpl := range mcpContext.ResourceTemplates {
		add(tmpl.Server)
	}
	for _, prompt := range mcpContext.Prompts {
		add(prompt.Server)
	}

	return refs
}

func validateRequiredMCPArgumentsForLookup(
	defs map[string]mcpRuntime.MCPArgumentDefinition,
	values map[string]string,
) error {
	for name, def := range defs {
		if !def.Required {
			continue
		}
		argName := strings.TrimSpace(def.Name)
		if argName == "" {
			argName = strings.TrimSpace(name)
		}
		if argName == "" {
			continue
		}
		if strings.TrimSpace(values[argName]) == "" {
			return fmt.Errorf("missing required argument %q", argName)
		}
	}
	return nil
}

func isOptionalMCPDiscoveryError(err error) bool {
	return errors.Is(err, mcpRuntime.ErrMCPRuntimeNotReady) ||
		errors.Is(err, mcpAuth.ErrMCPAuthRequired)
}

func getToolBundleEnabled(
	ctx context.Context,
	store *toolStore.ToolStore,
	bundleID bundleitemutils.BundleID,
) (bool, error) {
	resp, err := store.ListToolBundles(ctx, &toolSpec.ListToolBundlesRequest{
		BundleIDs:       []bundleitemutils.BundleID{bundleID},
		IncludeDisabled: true,
		PageSize:        1,
	})
	if err != nil {
		return false, err
	}
	if resp == nil || resp.Body == nil || len(resp.Body.ToolBundles) == 0 {
		return false, errors.New("bundle not found")
	}
	return resp.Body.ToolBundles[0].IsEnabled, nil
}
