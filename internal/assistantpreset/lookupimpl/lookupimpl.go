package lookupimpl

import (
	"context"
	"errors"
	"fmt"
	"strings"

	assistantpresetStore "github.com/flexigpt/flexigpt-app/internal/assistantpreset/store"
	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	mcpSpec "github.com/flexigpt/flexigpt-app/internal/mcp/spec"
	modelpresetSpec "github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
	modelpresetStore "github.com/flexigpt/flexigpt-app/internal/modelpreset/store"
	skillSpec "github.com/flexigpt/flexigpt-app/internal/skill/spec"
	skillStore "github.com/flexigpt/flexigpt-app/internal/skill/store"
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
		if errors.Is(err, toolSpec.ErrToolDisabled) || errors.Is(err, toolSpec.ErrBundleDisabled) {
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
	store *skillStore.SkillStore
}

func (a *skillLookupAdapter) GetSkillSummaryForSelection(
	ctx context.Context,
	selection skillSpec.SkillSelection,
) (assistantpresetStore.SkillSummary, error) {
	if a == nil || a.store == nil {
		return assistantpresetStore.SkillSummary{}, errors.New("skill lookup adapter is not configured")
	}

	if selection.SkillRef.BundleID == "" || selection.SkillRef.SkillSlug == "" {
		return assistantpresetStore.SkillSummary{}, errors.New("skill selection skillRef is incomplete")
	}

	bundleEnabled, err := getSkillBundleEnabled(ctx, a.store, selection.SkillRef.BundleID)
	if err != nil {
		return assistantpresetStore.SkillSummary{}, err
	}
	resp, err := a.store.GetSkill(ctx, &skillSpec.GetSkillRequest{
		BundleID:        selection.SkillRef.BundleID,
		SkillSlug:       selection.SkillRef.SkillSlug,
		IncludeDisabled: true,
	})
	if err != nil {
		if errors.Is(err, skillSpec.ErrSkillDisabled) || errors.Is(err, skillSpec.ErrSkillBundleDisabled) {
			return assistantpresetStore.SkillSummary{IsEnabled: false}, nil
		}
		return assistantpresetStore.SkillSummary{}, err
	}
	if resp == nil || resp.Body == nil {
		return assistantpresetStore.SkillSummary{}, errors.New("empty skill response")
	}
	if selection.SkillRef.SkillID != "" && resp.Body.ID != selection.SkillRef.SkillID {
		return assistantpresetStore.SkillSummary{}, fmt.Errorf(
			"skill ref id mismatch: got %q, expected %q",
			resp.Body.ID,
			selection.SkillRef.SkillID,
		)
	}

	return assistantpresetStore.SkillSummary{
		IsEnabled:    bundleEnabled && resp.Body.IsEnabled,
		Insert:       resp.Body.Insert,
		HasArguments: len(resp.Body.Arguments) > 0,
		HasResources: resp.Body.Resources.HasResources,
	}, nil
}

type MCPServerConfigStore interface {
	GetMCPServer(
		ctx context.Context,
		req *mcpSpec.GetMCPServerRequest,
	) (*mcpSpec.GetMCPServerResponse, error)
}

type MCPDiscoveryLookup interface {
	ListTools(
		ctx context.Context,
		req *mcpSpec.ListMCPServerToolsRequest,
	) (*mcpSpec.ListMCPServerToolsResponse, error)

	ListResources(
		ctx context.Context,
		req *mcpSpec.ListMCPServerResourcesRequest,
	) (*mcpSpec.ListMCPServerResourcesResponse, error)

	ListResourceTemplates(
		ctx context.Context,
		req *mcpSpec.ListMCPServerResourceTemplatesRequest,
	) (*mcpSpec.ListMCPServerResourceTemplatesResponse, error)

	ListPrompts(
		ctx context.Context,
		req *mcpSpec.ListMCPServerPromptsRequest,
	) (*mcpSpec.ListMCPServerPromptsResponse, error)
}

type mcpServerKey struct {
	BundleID bundleitemutils.BundleID
	ServerID mcpSpec.MCPServerID
}

type mcpContextLookupAdapter struct {
	serverStore MCPServerConfigStore
	discovery   MCPDiscoveryLookup
}

func NewMCPContextLookup(
	serverStore MCPServerConfigStore,
	discovery MCPDiscoveryLookup,
) assistantpresetStore.MCPContextLookup {
	return &mcpContextLookupAdapter{
		serverStore: serverStore,
		discovery:   discovery,
	}
}

func (a *mcpContextLookupAdapter) ValidateMCPConversationContext(
	ctx context.Context,
	mcpContext mcpSpec.MCPConversationContext,
) error {
	if a == nil || a.serverStore == nil {
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
	mcpContext mcpSpec.MCPConversationContext,
) error {
	keys := collectMCPServerKeys(mcpContext)
	for _, key := range keys {
		resp, err := a.serverStore.GetMCPServer(ctx, &mcpSpec.GetMCPServerRequest{
			BundleID: key.BundleID,
			ServerID: key.ServerID,
		})
		if err != nil {
			return fmt.Errorf("server %s/%s: %w", key.BundleID, key.ServerID, err)
		}
		if resp == nil || resp.Body == nil {
			return fmt.Errorf("server %s/%s: empty mcp server response", key.BundleID, key.ServerID)
		}
		if !resp.Body.Enabled {
			return fmt.Errorf("server %s/%s: referenced MCP server is disabled", key.BundleID, key.ServerID)
		}
	}
	return nil
}

func (a *mcpContextLookupAdapter) validateSelectedMCPTools(
	ctx context.Context,
	mcpContext mcpSpec.MCPConversationContext,
) error {
	for i, server := range mcpContext.Servers {
		if server.ToolExposure != mcpSpec.MCPToolExposureSelected {
			continue
		}

		tools, ok, err := a.listAllMCPTools(ctx, server.BundleID, server.ServerID)
		if err != nil {
			return fmt.Errorf("servers[%d]: %w", i, err)
		}
		if !ok {
			continue
		}

		byName := make(map[string]mcpSpec.MCPToolCapability, len(tools))
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
			if !current.Enabled {
				return fmt.Errorf(
					"servers[%d].selectedTools[%d]: MCP tool %q is disabled",
					i,
					j,
					selected.ToolName,
				)
			}
		}
	}
	return nil
}

func (a *mcpContextLookupAdapter) validateSelectedMCPResources(
	ctx context.Context,
	mcpContext mcpSpec.MCPConversationContext,
) error {
	cache := map[mcpServerKey]map[string]struct{}{}

	for i, selected := range mcpContext.Resources {
		key := mcpServerKey{BundleID: selected.BundleID, ServerID: selected.ServerID}
		byURI, exists := cache[key]
		if !exists {
			resources, ok, err := a.listAllMCPResources(ctx, key.BundleID, key.ServerID)
			if err != nil {
				return fmt.Errorf("resources[%d]: %w", i, err)
			}
			if !ok {
				continue
			}

			byURI = make(map[string]struct{}, len(resources))
			for _, resource := range resources {
				if resource.URI != "" {
					byURI[resource.URI] = struct{}{}
				}
			}
			cache[key] = byURI
		}

		if _, ok := byURI[selected.URI]; !ok {
			return fmt.Errorf(
				"resources[%d]: MCP resource %q was not found in current discovery",
				i,
				selected.URI,
			)
		}
	}
	return nil
}

func (a *mcpContextLookupAdapter) validateSelectedMCPResourceTemplates(
	ctx context.Context,
	mcpContext mcpSpec.MCPConversationContext,
) error {
	cache := map[mcpServerKey]map[string]mcpSpec.MCPResourceTemplateRef{}

	for i, selected := range mcpContext.ResourceTemplates {
		ref := selected.MCPResourceTemplateRef
		key := mcpServerKey{BundleID: ref.BundleID, ServerID: ref.ServerID}
		byTemplate, exists := cache[key]
		if !exists {
			templates, ok, err := a.listAllMCPResourceTemplates(ctx, key.BundleID, key.ServerID)
			if err != nil {
				return fmt.Errorf("resourceTemplates[%d]: %w", i, err)
			}
			if !ok {
				continue
			}

			byTemplate = make(map[string]mcpSpec.MCPResourceTemplateRef, len(templates))
			for _, tmpl := range templates {
				if tmpl.URITemplate != "" {
					byTemplate[tmpl.URITemplate] = tmpl
				}
			}
			cache[key] = byTemplate
		}

		current, ok := byTemplate[ref.URITemplate]
		if !ok {
			return fmt.Errorf(
				"resourceTemplates[%d]: MCP resource template %q was not found in current discovery",
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
	mcpContext mcpSpec.MCPConversationContext,
) error {
	cache := map[mcpServerKey]map[string]mcpSpec.MCPPromptRef{}

	for i, selected := range mcpContext.Prompts {
		key := mcpServerKey{BundleID: selected.BundleID, ServerID: selected.ServerID}
		byName, exists := cache[key]
		if !exists {
			prompts, ok, err := a.listAllMCPPrompts(ctx, key.BundleID, key.ServerID)
			if err != nil {
				return fmt.Errorf("prompts[%d]: %w", i, err)
			}
			if !ok {
				continue
			}

			byName = make(map[string]mcpSpec.MCPPromptRef, len(prompts))
			for _, prompt := range prompts {
				if prompt.PromptName != "" {
					byName[prompt.PromptName] = prompt
				}
			}
			cache[key] = byName
		}

		current, ok := byName[selected.PromptName]
		if !ok {
			return fmt.Errorf(
				"prompts[%d]: MCP prompt %q was not found in current discovery",
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

func (a *mcpContextLookupAdapter) listAllMCPTools(
	ctx context.Context,
	bundleID bundleitemutils.BundleID,
	serverID mcpSpec.MCPServerID,
) ([]mcpSpec.MCPToolCapability, bool, error) {
	out := make([]mcpSpec.MCPToolCapability, 0)
	pageToken := ""
	seenTokens := map[string]struct{}{}

	for {
		resp, err := a.discovery.ListTools(ctx, &mcpSpec.ListMCPServerToolsRequest{
			BundleID:  bundleID,
			ServerID:  serverID,
			PageSize:  mcpSpec.MaxMCPServerPageSize,
			PageToken: pageToken,
		})
		if err != nil {
			if isOptionalMCPDiscoveryError(err) {
				return nil, false, nil
			}
			return nil, false, err
		}
		if resp == nil || resp.Body == nil {
			return nil, false, errors.New("empty mcp tools response")
		}
		out = append(out, resp.Body.Tools...)
		if resp.Body.NextPageToken == nil || *resp.Body.NextPageToken == "" {
			break
		}
		pageToken = *resp.Body.NextPageToken
		if _, seen := seenTokens[pageToken]; seen {
			return nil, false, errors.New("mcp tools pagination returned a repeated pageToken")
		}
		seenTokens[pageToken] = struct{}{}
	}
	return out, true, nil
}

func (a *mcpContextLookupAdapter) listAllMCPResources(
	ctx context.Context,
	bundleID bundleitemutils.BundleID,
	serverID mcpSpec.MCPServerID,
) ([]mcpSpec.MCPResourceRef, bool, error) {
	out := make([]mcpSpec.MCPResourceRef, 0)
	pageToken := ""
	seenTokens := map[string]struct{}{}

	for {
		resp, err := a.discovery.ListResources(ctx, &mcpSpec.ListMCPServerResourcesRequest{
			BundleID:  bundleID,
			ServerID:  serverID,
			PageSize:  mcpSpec.MaxMCPServerPageSize,
			PageToken: pageToken,
		})
		if err != nil {
			if isOptionalMCPDiscoveryError(err) {
				return nil, false, nil
			}
			return nil, false, err
		}
		if resp == nil || resp.Body == nil {
			return nil, false, errors.New("empty mcp resources response")
		}
		out = append(out, resp.Body.Resources...)
		if resp.Body.NextPageToken == nil || *resp.Body.NextPageToken == "" {
			break
		}
		pageToken = *resp.Body.NextPageToken
		if _, seen := seenTokens[pageToken]; seen {
			return nil, false, errors.New("mcp resources pagination returned a repeated pageToken")
		}
		seenTokens[pageToken] = struct{}{}
	}
	return out, true, nil
}

func (a *mcpContextLookupAdapter) listAllMCPResourceTemplates(
	ctx context.Context,
	bundleID bundleitemutils.BundleID,
	serverID mcpSpec.MCPServerID,
) ([]mcpSpec.MCPResourceTemplateRef, bool, error) {
	out := make([]mcpSpec.MCPResourceTemplateRef, 0)
	pageToken := ""
	seenTokens := map[string]struct{}{}

	for {
		resp, err := a.discovery.ListResourceTemplates(ctx, &mcpSpec.ListMCPServerResourceTemplatesRequest{
			BundleID:  bundleID,
			ServerID:  serverID,
			PageSize:  mcpSpec.MaxMCPServerPageSize,
			PageToken: pageToken,
		})
		if err != nil {
			if isOptionalMCPDiscoveryError(err) {
				return nil, false, nil
			}
			return nil, false, err
		}
		if resp == nil || resp.Body == nil {
			return nil, false, errors.New("empty mcp resource templates response")
		}
		out = append(out, resp.Body.ResourceTemplates...)
		if resp.Body.NextPageToken == nil || *resp.Body.NextPageToken == "" {
			break
		}
		pageToken = *resp.Body.NextPageToken
		if _, seen := seenTokens[pageToken]; seen {
			return nil, false, errors.New("mcp resource templates pagination returned a repeated pageToken")
		}
		seenTokens[pageToken] = struct{}{}
	}
	return out, true, nil
}

func (a *mcpContextLookupAdapter) listAllMCPPrompts(
	ctx context.Context,
	bundleID bundleitemutils.BundleID,
	serverID mcpSpec.MCPServerID,
) ([]mcpSpec.MCPPromptRef, bool, error) {
	out := make([]mcpSpec.MCPPromptRef, 0)
	pageToken := ""
	seenTokens := map[string]struct{}{}

	for {
		resp, err := a.discovery.ListPrompts(ctx, &mcpSpec.ListMCPServerPromptsRequest{
			BundleID:  bundleID,
			ServerID:  serverID,
			PageSize:  mcpSpec.MaxMCPServerPageSize,
			PageToken: pageToken,
		})
		if err != nil {
			if isOptionalMCPDiscoveryError(err) {
				return nil, false, nil
			}
			return nil, false, err
		}
		if resp == nil || resp.Body == nil {
			return nil, false, errors.New("empty mcp prompts response")
		}
		out = append(out, resp.Body.Prompts...)
		if resp.Body.NextPageToken == nil || *resp.Body.NextPageToken == "" {
			break
		}
		pageToken = *resp.Body.NextPageToken
		if _, seen := seenTokens[pageToken]; seen {
			return nil, false, errors.New("mcp prompts pagination returned a repeated pageToken")
		}
		seenTokens[pageToken] = struct{}{}
	}
	return out, true, nil
}

func collectMCPServerKeys(mcpContext mcpSpec.MCPConversationContext) []mcpServerKey {
	seen := map[mcpServerKey]struct{}{}
	keys := make([]mcpServerKey, 0, len(mcpContext.Servers))

	add := func(bundleID bundleitemutils.BundleID, serverID mcpSpec.MCPServerID) {
		if bundleID == "" || serverID == "" {
			return
		}
		key := mcpServerKey{BundleID: bundleID, ServerID: serverID}
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}

	for _, server := range mcpContext.Servers {
		add(server.BundleID, server.ServerID)
	}
	for _, resource := range mcpContext.Resources {
		add(resource.BundleID, resource.ServerID)
	}
	for _, tmpl := range mcpContext.ResourceTemplates {
		add(tmpl.BundleID, tmpl.ServerID)
	}
	for _, prompt := range mcpContext.Prompts {
		add(prompt.BundleID, prompt.ServerID)
	}

	return keys
}

func validateRequiredMCPArgumentsForLookup(
	defs map[string]mcpSpec.MCPArgumentDefinition,
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
	return errors.Is(err, mcpSpec.ErrMCPRuntimeNotReady) ||
		errors.Is(err, mcpSpec.ErrMCPAuthRequired) ||
		errors.Is(err, mcpSpec.ErrMCPServerDisabled)
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
		return false, toolSpec.ErrBundleNotFound
	}
	return resp.Body.ToolBundles[0].IsEnabled, nil
}

func getSkillBundleEnabled(
	ctx context.Context,
	store *skillStore.SkillStore,
	bundleID bundleitemutils.BundleID,
) (bool, error) {
	resp, err := store.ListSkillBundles(ctx, &skillSpec.ListSkillBundlesRequest{
		BundleIDs:       []bundleitemutils.BundleID{bundleID},
		IncludeDisabled: true,
		PageSize:        1,
	})
	if err != nil {
		return false, err
	}
	if resp == nil || resp.Body == nil || len(resp.Body.SkillBundles) == 0 {
		return false, skillSpec.ErrSkillBundleNotFound
	}
	return resp.Body.SkillBundles[0].IsEnabled, nil
}

func NewAssistantPresetReferenceLookups(
	modelPresetSt *modelpresetStore.ModelPresetStore,
	toolSt *toolStore.ToolStore,
	skillSt *skillStore.SkillStore,
	mcpServerStore MCPServerConfigStore,
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
			store: skillSt,
		},
	}
	if mcpServerStore != nil {
		lookups.MCPContext = NewMCPContextLookup(mcpServerStore, mcpDiscovery)
	}
	return lookups
}
