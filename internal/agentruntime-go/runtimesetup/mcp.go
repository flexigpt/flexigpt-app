package runtimesetup

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"slices"
	"sort"
	"strings"

	inferenceSpec "github.com/flexigpt/inference-go/spec"

	mcpApps "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/apps"
	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/invocation"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/server"
	sessionRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/session"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/mcpruntime"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/mcppolicy"
)

func (c *coordinator) prepareMCP(
	ctx context.Context,
	selections []sessionRuntime.MCPServerSelection,
	uses map[string]*sessionRuntime.ResolvedArtifact,
	owners map[string]sessionRuntime.ConfigurationOwner,
	summary *sessionRuntime.OperationResolutionSummary,
) ([]inferenceSpec.ToolChoice, preparedBindings, []string, error) {
	choices := make([]inferenceSpec.ToolChoice, 0)
	bindings := preparedBindings{
		byChoice: map[string]sessionRuntime.ToolCallProposal{},
		byName:   map[string]sessionRuntime.ToolCallProposal{},
	}
	contexts := make([]string, 0)

	for _, selection := range selections {
		key := selection.CanonicalKey()
		used := uses[key]
		owner := owners[key]
		if owner == "" {
			owner = sessionRuntime.ConfigurationOwnerSession
		}

		ref, err := artifactRef(selection.Server)
		if err != nil {
			return nil, preparedBindings{}, nil, err
		}

		serverID, err := mcpruntime.ServerIDForArtifact(ref)
		if err != nil {
			return nil, preparedBindings{}, nil, err
		}

		status, err := c.mcpRuntime.runtime.Status(ctx, serverID)
		if err != nil ||
			status == nil ||
			status.Status != mcpServer.MCPServerStatusReady {
			message := "MCP server is not connected"
			if err != nil {
				message = err.Error()
			}

			summary.MCP = append(
				summary.MCP,
				sessionRuntime.ResolutionItem{
					Key:      key,
					Kind:     resolutionItemKindMCP,
					Owner:    owner,
					State:    sessionRuntime.ResolutionStateUnavailable,
					Selected: &selection.Server,
					Used:     used,
					Reason:   message,
				},
			)
			summary.Diagnostics = append(
				summary.Diagnostics,
				sessionRuntime.ResolutionDiagnostic{
					Severity: sessionRuntime.DiagnosticSeverityWarning,
					Code:     "mcp_unavailable",
					Message:  message,
					Key:      key,
				},
			)
			continue
		}

		summary.MCP = append(
			summary.MCP,
			sessionRuntime.ResolutionItem{
				Key:      key,
				Kind:     resolutionItemKindMCP,
				Owner:    owner,
				State:    sessionRuntime.ResolutionStateIncluded,
				Selected: &selection.Server,
				Used:     used,
			},
		)

		if selection.IncludeServerInstructions &&
			strings.TrimSpace(status.Instructions) != "" {
			contexts = append(
				contexts,
				strings.Join([]string{
					"### MCP server instructions",
					"Server: " + string(serverID),
					status.Instructions,
				}, "\n\n"),
			)
		}

		serverChoices, serverBindings, toolDiagnostics, err := c.prepareMCPTools(
			ctx,
			serverID,
			selection,
		)
		summary.Diagnostics = append(summary.Diagnostics, toolDiagnostics...)
		if err != nil {
			return nil, preparedBindings{}, nil, err
		}

		choices, bindings, err = mergeToolBindings(
			choices,
			bindings,
			serverChoices,
			serverBindings,
		)
		if err != nil {
			return nil, preparedBindings{}, nil, err
		}

		resourceContexts, resourceDiagnostics, err := c.prepareMCPResources(
			ctx,
			serverID,
			selection,
		)
		summary.Diagnostics = append(summary.Diagnostics, resourceDiagnostics...)
		if err != nil {
			summary.Diagnostics = append(
				summary.Diagnostics,
				sessionRuntime.ResolutionDiagnostic{
					Severity: sessionRuntime.DiagnosticSeverityWarning,
					Code:     "mcp_context_unavailable",
					Message:  err.Error(),
					Key:      selection.CanonicalKey(),
				},
			)
			continue
		}
		contexts = append(contexts, resourceContexts...)
	}

	return choices, bindings, contexts, nil
}

func (c *coordinator) prepareMCPTools(
	ctx context.Context,
	serverID mcpServer.ServerID,
	selection sessionRuntime.MCPServerSelection,
) ([]inferenceSpec.ToolChoice, preparedBindings,
	[]sessionRuntime.ResolutionDiagnostic, error,
) {
	diagnostics := make([]sessionRuntime.ResolutionDiagnostic, 0)
	if selection.ToolExposure == sessionRuntime.MCPToolExposureNone {
		return nil, preparedBindings{
			byChoice: map[string]sessionRuntime.ToolCallProposal{},
			byName:   map[string]sessionRuntime.ToolCallProposal{},
		}, diagnostics, nil
	}

	tools, err := c.mcpRuntime.runtime.ListTools(ctx, serverID)
	if err != nil {
		return nil, preparedBindings{}, diagnostics, err
	}

	selected := make([]mcpServer.MCPToolCapability, 0, len(tools))
	switch selection.ToolExposure {
	case sessionRuntime.MCPToolExposureAll:
		for _, tool := range tools {
			if !tool.Enabled ||
				tool.TaskSupport == mcpServer.MCPTaskSupportRequired ||
				!mcpApps.ToolVisibleToModel(tool.App) {
				continue
			}
			selected = append(selected, tool)
		}

	case sessionRuntime.MCPToolExposureSelected:
		byName := make(
			map[string]mcpServer.MCPToolCapability,
			len(tools),
		)
		for _, tool := range tools {
			byName[tool.ToolName] = tool
		}

		for _, expected := range selection.SelectedTools {
			tool, found := byName[expected.ToolName]
			if !found {
				diagnostics = append(diagnostics, mcpSelectionDiagnostic(
					selection,
					"tool:"+expected.ToolName,
					"mcp_tool_unavailable",
					"selected MCP Tool is no longer available",
				))
				continue
			}
			if !tool.Enabled ||
				tool.TaskSupport == mcpServer.MCPTaskSupportRequired ||
				!mcpApps.ToolVisibleToModel(tool.App) {
				diagnostics = append(diagnostics, mcpSelectionDiagnostic(
					selection,
					"tool:"+expected.ToolName,
					"mcp_tool_unavailable",
					"selected MCP Tool is unavailable under current server policy",
				))
				continue
			}
			if expected.ExpectedDigest != "" &&
				expected.ExpectedDigest != tool.Digest {
				diagnostics = append(diagnostics, mcpSelectionDiagnostic(
					selection,
					"tool:"+expected.ToolName,
					"mcp_tool_stale",
					"selected MCP Tool changed since selection",
				))
				continue
			}

			constrained, err := constrainMCPTool(tool, expected)
			if err != nil {
				return nil, preparedBindings{}, diagnostics, err
			}
			selected = append(selected, constrained)
		}

	default:
		return nil, preparedBindings{}, diagnostics, errors.New(
			"unsupported MCP Tool exposure",
		)
	}

	sort.Slice(selected, func(left, right int) bool {
		return selected[left].ProviderToolName <
			selected[right].ProviderToolName
	})

	choices := make(
		[]inferenceSpec.ToolChoice,
		0,
		len(selected),
	)
	bindings := preparedBindings{
		byChoice: make(
			map[string]sessionRuntime.ToolCallProposal,
			len(selected),
		),
		byName: make(
			map[string]sessionRuntime.ToolCallProposal,
			len(selected),
		),
	}

	for _, tool := range selected {
		description := strings.TrimSpace(tool.Description)
		if description == "" {
			description = firstNonEmpty(
				tool.DisplayName,
				tool.ToolName,
			)
		}

		schema := tool.InputSchema
		if len(schema) == 0 {
			schema = map[string]any{typeStr: jsonSchemaTypeObject}
		}

		choice := inferenceSpec.ToolChoice{
			Type:        inferenceSpec.ToolTypeFunction,
			ID:          tool.ChoiceID,
			Name:        tool.ProviderToolName,
			Description: description,
			Arguments:   schema,
		}

		bindingData, err := json.Marshal(
			invocation.MappedToolBinding{
				Server:           tool.Server,
				ProviderToolName: tool.ProviderToolName,
				ChoiceID:         tool.ChoiceID,
				ToolName:         tool.ToolName,
				ToolDigest:       tool.Digest,
				ApprovalRule:     tool.ApprovalRule,
				ExecutionMode:    tool.ExecutionMode,
				AppResourceURI:   appResourceURI(tool.App),
				Visibility:       appVisibility(tool.App),
			},
		)
		if err != nil {
			return nil, preparedBindings{}, diagnostics, err
		}

		proposal := sessionRuntime.ToolCallProposal{
			Name:      choice.Name,
			ChoiceID:  choice.ID,
			Type:      choice.Type,
			Required:  true,
			Automatic: tool.ExecutionMode == mcppolicy.MCPExecutionModeAuto,
			Binding: sessionRuntime.ToolBinding{
				Kind: sessionRuntime.ToolBindingKindMCP,
				Data: bindingData,
			},
		}

		choices = append(choices, choice)
		bindings.byChoice[choice.ID] = proposal
		bindings.byName[choice.Name] = proposal
	}

	return choices, bindings, diagnostics, nil
}

func constrainMCPTool(
	tool mcpServer.MCPToolCapability,
	selection sessionRuntime.MCPToolSelection,
) (mcpServer.MCPToolCapability, error) {
	output := tool

	if selection.ApprovalRule != nil {
		value, err := selectedMCPApprovalRule(*selection.ApprovalRule)
		if err != nil {
			return mcpServer.MCPToolCapability{}, err
		}
		if mcppolicy.ApprovalRuleRank(value) <
			mcppolicy.ApprovalRuleRank(
				mcppolicy.NormalizedApprovalRule(tool.ApprovalRule),
			) {
			return mcpServer.MCPToolCapability{}, errors.New(
				"MCP approval override weakens the effective server policy",
			)
		}
		output.ApprovalRule = value
	}

	if selection.ExecutionMode != nil {
		value, err := selectedMCPExecutionMode(*selection.ExecutionMode)
		if err != nil {
			return mcpServer.MCPToolCapability{}, err
		}
		if mcppolicy.ExecutionModeRank(value) <
			mcppolicy.ExecutionModeRank(
				mcppolicy.NormalizedExecutionMode(tool.ExecutionMode),
			) {
			return mcpServer.MCPToolCapability{}, errors.New(
				"MCP execution override weakens the effective server policy",
			)
		}
		output.ExecutionMode = value
	}

	return output, nil
}

func selectedMCPApprovalRule(
	value string,
) (mcppolicy.MCPApprovalRule, error) {
	if strings.TrimSpace(value) != value {
		return "", errors.New("MCP approval override is not normalized")
	}

	rule := mcppolicy.MCPApprovalRule(value)
	switch rule {
	case mcppolicy.MCPApprovalRuleAllow,
		mcppolicy.MCPApprovalRuleAsk,
		mcppolicy.MCPApprovalRuleDeny:
		return rule, nil
	default:
		return "", errors.New("unsupported MCP approval override")
	}
}

func selectedMCPExecutionMode(
	value string,
) (mcppolicy.MCPExecutionMode, error) {
	if strings.TrimSpace(value) != value {
		return "", errors.New("MCP execution override is not normalized")
	}

	mode := mcppolicy.MCPExecutionMode(value)
	switch mode {
	case mcppolicy.MCPExecutionModeAuto,
		mcppolicy.MCPExecutionModeManual:
		return mode, nil
	default:
		return "", errors.New("unsupported MCP execution override")
	}
}

func (c *coordinator) prepareMCPResources(
	ctx context.Context,
	serverID mcpServer.ServerID,
	selection sessionRuntime.MCPServerSelection,
) ([]string, []sessionRuntime.ResolutionDiagnostic, error) {
	sections := make([]string, 0)
	diagnostics := make([]sessionRuntime.ResolutionDiagnostic, 0)

	if len(selection.Resources) != 0 {
		available, err := c.mcpRuntime.runtime.ListResources(
			ctx,
			serverID,
		)
		if err != nil {
			return nil, diagnostics, err
		}

		byURI := make(
			map[string]mcpServer.MCPResourceRef,
			len(available),
		)
		for _, resource := range available {
			byURI[resource.URI] = resource
		}

		for _, expected := range selection.Resources {
			resource, found := byURI[expected.URI]
			if !found {
				diagnostics = append(diagnostics, mcpSelectionDiagnostic(
					selection,
					"resource:"+expected.URI,
					"mcp_resource_unavailable",
					"selected MCP resource is no longer available",
				))
				continue
			}
			if expected.ExpectedDigest != "" &&
				expected.ExpectedDigest != resource.Digest {
				diagnostics = append(diagnostics, mcpSelectionDiagnostic(
					selection,
					"resource:"+expected.URI,
					"mcp_resource_stale",
					"selected MCP resource changed since selection",
				))
				continue
			}

			response, err := c.mcpRuntime.runtime.ReadResource(
				ctx,
				serverID,
				expected.URI,
			)
			if err != nil || response == nil {
				diagnostics = append(diagnostics, mcpSelectionDiagnostic(
					selection,
					"resource:"+expected.URI,
					"mcp_resource_unavailable",
					"selected MCP resource could not be read",
				))
				continue
			}

			if text := mcpContentsText(response.Contents); text != "" {
				sections = append(
					sections,
					strings.Join([]string{
						"### MCP resource",
						"Server: " + string(serverID),
						"URI: " + expected.URI,
						text,
					}, "\n\n"),
				)
			} else {
				diagnostics = append(diagnostics, mcpSelectionDiagnostic(
					selection,
					"resource:"+expected.URI,
					"mcp_context_omitted",
					"selected MCP resource returned no textual context",
				))
			}
		}
	}

	if len(selection.ResourceTemplates) != 0 {
		available, err := c.mcpRuntime.runtime.ListResourceTemplates(
			ctx,
			serverID,
		)
		if err != nil {
			return nil, diagnostics, err
		}

		byTemplate := make(
			map[string]mcpServer.MCPResourceTemplateRef,
			len(available),
		)
		for _, template := range available {
			byTemplate[template.URITemplate] = template
		}

		for _, expected := range selection.ResourceTemplates {
			template, found := byTemplate[expected.URITemplate]
			if !found {
				diagnostics = append(diagnostics, mcpSelectionDiagnostic(
					selection,
					"resourceTemplate:"+expected.URITemplate,
					"mcp_resource_template_unavailable",
					"selected MCP resource template is no longer available",
				))
				continue
			}
			if expected.ExpectedDigest != "" &&
				expected.ExpectedDigest != template.Digest {
				diagnostics = append(diagnostics, mcpSelectionDiagnostic(
					selection,
					"resourceTemplate:"+expected.URITemplate,
					"mcp_resource_template_stale",
					"selected MCP resource template changed since selection",
				))
				continue
			}

			uri, err := resolveResourceTemplate(
				expected.URITemplate,
				expected.Arguments,
			)
			if err != nil {
				diagnostics = append(diagnostics, mcpSelectionDiagnostic(
					selection,
					"resourceTemplate:"+expected.URITemplate,
					"mcp_resource_template_invalid",
					"selected MCP resource template arguments are invalid",
				))
				continue
			}

			response, err := c.mcpRuntime.runtime.ReadResource(
				ctx,
				serverID,
				uri,
			)
			if err != nil || response == nil {
				diagnostics = append(diagnostics, mcpSelectionDiagnostic(
					selection,
					"resourceTemplate:"+expected.URITemplate,
					"mcp_resource_template_unavailable",
					"selected MCP resource template could not be read",
				))
				continue
			}

			if text := mcpContentsText(response.Contents); text != "" {
				sections = append(
					sections,
					strings.Join([]string{
						"### MCP resource template",
						"Server: " + string(serverID),
						"URI: " + uri,
						text,
					}, "\n\n"),
				)
			} else {
				diagnostics = append(diagnostics, mcpSelectionDiagnostic(
					selection,
					"resourceTemplate:"+expected.URITemplate,
					"mcp_context_omitted",
					"selected MCP resource template returned no textual context",
				))
			}
		}
	}

	if len(selection.Prompts) != 0 {
		available, err := c.mcpRuntime.runtime.ListPrompts(
			ctx,
			serverID,
		)
		if err != nil {
			return nil, diagnostics, err
		}

		byName := make(
			map[string]mcpServer.MCPPromptRef,
			len(available),
		)
		for _, prompt := range available {
			byName[prompt.PromptName] = prompt
		}

		for _, expected := range selection.Prompts {
			prompt, found := byName[expected.PromptName]
			if !found {
				diagnostics = append(diagnostics, mcpSelectionDiagnostic(
					selection,
					"prompt:"+expected.PromptName,
					"mcp_prompt_unavailable",
					"selected MCP prompt is no longer available",
				))
				continue
			}
			if expected.ExpectedDigest != "" &&
				expected.ExpectedDigest != prompt.Digest {
				diagnostics = append(diagnostics, mcpSelectionDiagnostic(
					selection,
					"prompt:"+expected.PromptName,
					"mcp_prompt_stale",
					"selected MCP prompt changed since selection",
				))
				continue
			}

			response, err := c.mcpRuntime.runtime.GetPrompt(
				ctx,
				serverID,
				expected.PromptName,
				expected.Arguments,
			)
			if err != nil || response == nil {
				diagnostics = append(diagnostics, mcpSelectionDiagnostic(
					selection,
					"prompt:"+expected.PromptName,
					"mcp_prompt_unavailable",
					"selected MCP prompt could not be read",
				))
				continue
			}

			parts := make(
				[]string,
				0,
				len(response.Messages),
			)
			for _, message := range response.Messages {
				if text := mcpContentText(message.Content); text != "" {
					parts = append(
						parts,
						message.Role+":\n"+text,
					)
				}
			}

			if len(parts) != 0 {
				sections = append(
					sections,
					strings.Join([]string{
						"### MCP prompt",
						"Server: " + string(serverID),
						"Name: " + expected.PromptName,
						strings.Join(parts, "\n\n"),
					}, "\n\n"),
				)
			} else {
				diagnostics = append(diagnostics, mcpSelectionDiagnostic(
					selection,
					"prompt:"+expected.PromptName,
					"mcp_context_omitted",
					"selected MCP prompt returned no textual context",
				))
			}
		}
	}

	return sections, diagnostics, nil
}

func mcpSelectionDiagnostic(
	selection sessionRuntime.MCPServerSelection,
	item string,
	code string,
	message string,
) sessionRuntime.ResolutionDiagnostic {
	key := selection.CanonicalKey()
	if strings.TrimSpace(item) != "" {
		key += ":" + item
	}
	return sessionRuntime.ResolutionDiagnostic{
		Severity: sessionRuntime.DiagnosticSeverityWarning,
		Code:     code,
		Message:  message,
		Key:      key,
	}
}

func (c *coordinator) PrepareAppTool(
	ctx context.Context,
	request sessionRuntime.AppToolPreparationRequest,
) (sessionRuntime.AppToolPreparation, error) {
	if request.App.Route == nil {
		return sessionRuntime.AppToolPreparation{}, errors.New(
			"app runtime route is unavailable",
		)
	}

	var origin invocation.MappedToolBinding
	if err := json.Unmarshal(
		request.App.Route.Binding.Data,
		&origin,
	); err != nil {
		return sessionRuntime.AppToolPreparation{}, err
	}
	if err := origin.Validate(); err != nil {
		return sessionRuntime.AppToolPreparation{}, err
	}
	if request.App.ServerID != string(origin.Server) {
		return sessionRuntime.AppToolPreparation{}, errors.New(
			"app route belongs to another MCP server",
		)
	}

	tools, err := c.mcpRuntime.runtime.ListTools(ctx, origin.Server)
	if err != nil {
		return sessionRuntime.AppToolPreparation{}, err
	}

	var selected *mcpServer.MCPToolCapability
	for index := range tools {
		if tools[index].ToolName == request.ToolName {
			selected = &tools[index]
			break
		}
	}
	if selected == nil {
		return sessionRuntime.AppToolPreparation{}, errors.New(
			"app requested an unavailable MCP Tool",
		)
	}
	if !selected.Enabled ||
		!mcpApps.ToolVisibleToApp(selected.App) {
		return sessionRuntime.AppToolPreparation{}, errors.New(
			"app requested an MCP Tool that is not App-visible",
		)
	}

	bindingData, err := json.Marshal(
		invocation.MappedToolBinding{
			Server:           selected.Server,
			ProviderToolName: selected.ProviderToolName,
			ChoiceID:         selected.ChoiceID,
			ToolName:         selected.ToolName,
			ToolDigest:       selected.Digest,
			ApprovalRule:     selected.ApprovalRule,
			ExecutionMode:    selected.ExecutionMode,
			AppResourceURI:   appResourceURI(selected.App),
			Visibility:       appVisibility(selected.App),
		},
	)
	if err != nil {
		return sessionRuntime.AppToolPreparation{}, err
	}

	arguments := request.Arguments
	if len(arguments) == 0 {
		arguments = json.RawMessage(`{}`)
	}
	if !json.Valid(arguments) {
		return sessionRuntime.AppToolPreparation{}, errors.New(
			"app Tool arguments are invalid JSON",
		)
	}

	return sessionRuntime.AppToolPreparation{
		Proposal: sessionRuntime.ToolCallProposal{
			Name:      selected.ProviderToolName,
			ChoiceID:  selected.ChoiceID,
			Type:      inferenceSpec.ToolTypeFunction,
			Arguments: append(json.RawMessage(nil), arguments...),
			Required:  false,
			Binding: sessionRuntime.ToolBinding{
				Kind: sessionRuntime.ToolBindingKindMCP,
				Data: bindingData,
			},
		},
	}, nil
}

func (c *coordinator) ReadAppResource(
	ctx context.Context,
	request sessionRuntime.AppResourceReadRequest,
) (sessionRuntime.AppResourceReadResult, error) {
	if request.App.Route == nil {
		return sessionRuntime.AppResourceReadResult{}, errors.New(
			"app runtime route is unavailable",
		)
	}

	var binding invocation.MappedToolBinding
	if err := json.Unmarshal(
		request.App.Route.Binding.Data,
		&binding,
	); err != nil {
		return sessionRuntime.AppResourceReadResult{}, err
	}
	if err := binding.Validate(); err != nil {
		return sessionRuntime.AppResourceReadResult{}, err
	}
	if request.App.ServerID != string(binding.Server) {
		return sessionRuntime.AppResourceReadResult{}, errors.New(
			"app route belongs to another MCP server",
		)
	}

	uri := strings.TrimSpace(request.ResourceURI)
	if uri == "" {
		uri = strings.TrimSpace(request.App.ResourceURI)
	}
	if uri == "" {
		return sessionRuntime.AppResourceReadResult{}, errors.New(
			"app resource URI is required",
		)
	}

	resources, err := c.mcpRuntime.runtime.ListResources(
		ctx,
		binding.Server,
	)
	if err != nil {
		return sessionRuntime.AppResourceReadResult{}, err
	}
	if !slices.ContainsFunc(
		resources,
		func(value mcpServer.MCPResourceRef) bool {
			return value.URI == uri
		},
	) {
		return sessionRuntime.AppResourceReadResult{}, errors.New(
			"app resource is not available on the originating MCP server",
		)
	}

	response, err := c.mcpRuntime.runtime.ReadResource(
		ctx,
		binding.Server,
		uri,
	)
	if err != nil {
		return sessionRuntime.AppResourceReadResult{}, err
	}

	raw, err := json.Marshal(response)
	if err != nil {
		return sessionRuntime.AppResourceReadResult{}, err
	}
	return sessionRuntime.AppResourceReadResult{
		Content: raw,
	}, nil
}

func (c *coordinator) ValidateAppContext(
	_ context.Context,
	request sessionRuntime.AppContextStagingRequest,
) (sessionRuntime.AppContextUpdate, error) {
	if request.App.Route == nil {
		return sessionRuntime.AppContextUpdate{}, errors.New(
			"app runtime route is unavailable",
		)
	}
	if request.Update.AppInstanceID != "" &&
		request.Update.AppInstanceID != request.App.ID {
		return sessionRuntime.AppContextUpdate{}, errors.New(
			"app context update belongs to another App instance",
		)
	}
	if request.Update.ViewID != "" &&
		request.Update.ViewID != request.ViewID {
		return sessionRuntime.AppContextUpdate{}, errors.New(
			"app context update belongs to another App view",
		)
	}
	if request.Update.ServerID != "" &&
		request.Update.ServerID != request.App.ServerID {
		return sessionRuntime.AppContextUpdate{}, errors.New(
			"app context update belongs to another MCP server",
		)
	}

	update := request.Update
	update.AppInstanceID = request.App.ID
	update.ViewID = request.ViewID
	update.ServerID = request.App.ServerID
	if update.ResourceURI == "" {
		update.ResourceURI = request.App.ResourceURI
	}
	if len(update.StructuredContent) != 0 {
		canonical, err := jsonutil.Canonicalize(
			update.StructuredContent,
		)
		if err != nil {
			return sessionRuntime.AppContextUpdate{}, err
		}
		update.StructuredContent = canonical
	}
	return update, nil
}

func resolveResourceTemplate(
	template string,
	arguments map[string]string,
) (string, error) {
	parsed, err := url.Parse(template)
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New(
			"MCP resource template must be an absolute URI",
		)
	}

	for name, value := range arguments {
		template = strings.ReplaceAll(
			template,
			"{"+name+"}",
			url.PathEscape(value),
		)
	}
	if strings.Contains(template, "{") ||
		strings.Contains(template, "}") {
		return "", errors.New(
			"MCP resource template arguments are incomplete",
		)
	}
	return template, nil
}

func mcpContentsText(contents []mcpServer.MCPContent) string {
	parts := make([]string, 0, len(contents))
	for _, content := range contents {
		if text := mcpContentText(content); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n\n")
}

func mcpContentText(content mcpServer.MCPContent) string {
	switch content.Type {
	case mcpServer.MCPContentTypeText:
		return content.Text

	case mcpServer.MCPContentTypeResource:
		if content.Resource != nil &&
			content.Resource.Text != "" {
			return content.Resource.Text
		}
		if content.Resource != nil {
			return "[MCP resource omitted: " +
				content.Resource.URI + "]"
		}

	case mcpServer.MCPContentTypeResourceLink:
		return strings.TrimSpace(content.Name + " " + content.URI)

	case mcpServer.MCPContentTypeImage:
		return "[MCP image content omitted]"

	case mcpServer.MCPContentTypeAudio:
		return "[MCP audio content omitted]"
	}
	return ""
}

func appResourceURI(info *mcpServer.MCPToolAppInfo) string {
	if info == nil {
		return ""
	}
	return info.ResourceURI
}

func appVisibility(info *mcpServer.MCPToolAppInfo) []string {
	if info == nil {
		return nil
	}
	return slices.Clone(info.Visibility)
}
