package sdkclient

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/mcp/policy"
	"github.com/flexigpt/flexigpt-app/internal/mcp/runtime"
	"github.com/flexigpt/flexigpt-app/internal/mcp/server"
	mcpSDK "github.com/modelcontextprotocol/go-sdk/mcp"
)

var uriTemplateVariableRE = regexp.MustCompile(`\{([A-Za-z_][A-Za-z0-9_.-]*)\}`)

type Session struct {
	server  artifact.ArtifactRef
	session *mcpSDK.ClientSession
	logger  *slog.Logger
}

func (s *Session) Close(ctx context.Context) error {
	if s == nil || s.session == nil {
		return nil
	}

	done := make(chan error, 1)
	go func() {
		done <- s.session.Close()
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Session) Discover(
	ctx context.Context,
	config server.RuntimeConfig,
) (runtime.MCPDiscoverySnapshot, error) {
	if s == nil || s.session == nil {
		return runtime.MCPDiscoverySnapshot{}, fmt.Errorf("%w: nil session", runtime.ErrMCPRuntimeNotReady)
	}

	if config.Server != s.server {
		return runtime.MCPDiscoverySnapshot{}, fmt.Errorf(
			"%w: SDK session belongs to another MCP Server Artifact",
			runtime.ErrMCPRuntimeNotReady,
		)
	}

	out := runtime.MCPDiscoverySnapshot{
		Server: s.server,
	}

	initResult := s.session.InitializeResult()
	if initResult != nil {
		out.NegotiatedProtocolVersion = initResult.ProtocolVersion
		out.Instructions = initResult.Instructions

		if initResult.ServerInfo != nil {
			out.ServerInfo = &runtime.MCPImplementationInfo{
				Name:    initResult.ServerInfo.Name,
				Version: initResult.ServerInfo.Version,
			}
		}

		if initResult.Capabilities != nil {
			out.ServerCapabilities = summarizeCapabilities(initResult.Capabilities)
		}
	}

	caps := initResultCapabilities(initResult)

	var (
		tools             []runtime.MCPToolCapability
		resources         []runtime.MCPResourceRef
		resourceTemplates []runtime.MCPResourceTemplateRef
		prompts           []runtime.MCPPromptRef

		toolsErr             error
		resourcesErr         error
		resourceTemplatesErr error
		promptsErr           error

		wait sync.WaitGroup
	)

	if caps == nil || caps.Tools != nil {
		wait.Go(func() {
			tools, toolsErr = s.listAllTools(ctx, config)
		})
	}

	if caps == nil || caps.Resources != nil {
		wait.Add(2)
		go func() {
			defer wait.Done()
			resources, resourcesErr = s.listAllResources(ctx)
		}()
		go func() {
			defer wait.Done()
			resourceTemplates, resourceTemplatesErr = s.listAllResourceTemplates(ctx)
		}()
	}

	if caps == nil || caps.Prompts != nil {
		wait.Go(func() {
			prompts, promptsErr = s.listAllPrompts(ctx)
		})
	}

	wait.Wait()

	if toolsErr != nil {
		s.log().Warn("mcp tools discovery failed", "server", s.server, "err", toolsErr)
	} else {
		out.Tools = tools
	}
	if resourcesErr != nil {
		s.log().Warn("mcp resources discovery failed", "server", s.server, "err", resourcesErr)
	} else {
		out.Resources = resources
	}
	if resourceTemplatesErr != nil {
		s.log().Warn(
			"mcp resource templates discovery failed",
			"server",
			s.server,
			"err",
			resourceTemplatesErr,
		)
	} else {
		out.ResourceTemplates = resourceTemplates
	}
	if promptsErr != nil {
		s.log().Warn("mcp prompts discovery failed", "server", s.server, "err", promptsErr)
	} else {
		out.Prompts = prompts
	}

	return out, nil
}

func (s *Session) CallTool(
	ctx context.Context,
	toolName string,
	args map[string]any,
) (*runtime.InvokeMCPToolResponseBody, error) {
	if s == nil || s.session == nil {
		return nil, fmt.Errorf("%w: nil session", runtime.ErrMCPRuntimeNotReady)
	}
	if args == nil {
		args = map[string]any{}
	}

	res, err := s.session.CallTool(ctx, &mcpSDK.CallToolParams{
		Name:      toolName,
		Arguments: args,
	})
	if err != nil {
		return nil, err
	}
	if res == nil {
		res = &mcpSDK.CallToolResult{}
	}

	return &runtime.InvokeMCPToolResponseBody{
		Server:            s.server,
		ToolName:          toolName,
		Content:           contentSliceToSpec(res.Content),
		StructuredContent: res.StructuredContent,
		IsError:           res.IsError,
	}, nil
}

func (s *Session) ReadResource(
	ctx context.Context,
	uri string,
) (*runtime.MCPReadResourceResponseBody, error) {
	if s == nil || s.session == nil {
		return nil, fmt.Errorf("%w: nil session", runtime.ErrMCPRuntimeNotReady)
	}

	res, err := s.session.ReadResource(ctx, &mcpSDK.ReadResourceParams{
		URI: uri,
	})
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, fmt.Errorf("%w: resource read returned nil response", runtime.ErrMCPRuntimeNotReady)
	}

	contents := make([]runtime.MCPContent, 0, len(res.Contents))
	for _, rc := range res.Contents {
		if rc == nil {
			continue
		}
		contents = append(contents, contentToSpec(&mcpSDK.EmbeddedResource{Resource: rc}))
	}

	return &runtime.MCPReadResourceResponseBody{
		Server:   s.server,
		URI:      uri,
		Contents: contents,
	}, nil
}

func (s *Session) GetPrompt(
	ctx context.Context,
	name string,
	args map[string]string,
) (*runtime.MCPGetPromptResponseBody, error) {
	if s == nil || s.session == nil {
		return nil, fmt.Errorf("%w: nil session", runtime.ErrMCPRuntimeNotReady)
	}

	res, err := s.session.GetPrompt(ctx, &mcpSDK.GetPromptParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, fmt.Errorf("%w: prompt read returned nil response", runtime.ErrMCPRuntimeNotReady)
	}
	messages := make([]runtime.MCPPromptMessage, 0, len(res.Messages))
	for _, msg := range res.Messages {
		if msg == nil {
			continue
		}
		messages = append(messages, runtime.MCPPromptMessage{
			Role:    string(msg.Role),
			Content: contentToSpec(msg.Content),
		})
	}

	return &runtime.MCPGetPromptResponseBody{
		Server:      s.server,
		PromptName:  name,
		Description: res.Description,
		Messages:    messages,
	}, nil
}

func (s *Session) Complete(
	ctx context.Context,
	req runtime.MCPCompleteArgumentRequestBody,
) (*runtime.MCPCompletionResult, error) {
	if s == nil || s.session == nil {
		return nil, fmt.Errorf("%w: nil session", runtime.ErrMCPRuntimeNotReady)
	}

	ref, err := completionReference(req)
	if err != nil {
		return nil, err
	}

	res, err := s.session.Complete(ctx, &mcpSDK.CompleteParams{
		Argument: mcpSDK.CompleteParamsArgument{
			Name:  req.ArgumentName,
			Value: req.ArgumentValue,
		},
		Context: &mcpSDK.CompleteContext{
			Arguments: req.Context,
		},
		Ref: ref,
	})
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, fmt.Errorf("%w: completion returned nil response", runtime.ErrMCPRuntimeNotReady)
	}
	return &runtime.MCPCompletionResult{
		Values:  res.Completion.Values,
		Total:   res.Completion.Total,
		HasMore: res.Completion.HasMore,
	}, nil
}

func (s *Session) listAllTools(
	ctx context.Context,
	config server.RuntimeConfig,
) ([]runtime.MCPToolCapability, error) {
	out := make([]runtime.MCPToolCapability, 0)
	cursor := ""

	for {
		res, err := s.session.ListTools(ctx, &mcpSDK.ListToolsParams{
			Cursor: cursor,
		})
		if err != nil {
			return nil, err
		}
		if res == nil {
			return nil, fmt.Errorf("%w: tools/list returned nil response", runtime.ErrMCPRuntimeNotReady)
		}

		for _, t := range res.Tools {
			if t == nil {
				continue
			}

			taskSupport := taskSupportFromMeta(t.Meta)
			approvalRule, executionMode, override, hasOverride := effectiveToolPolicy(config, t.Name)
			toolDigest := digestAny(t)
			enabled := taskSupport != runtime.MCPTaskSupportRequired &&
				approvalRule != policy.MCPApprovalRuleDeny
			if hasOverride &&
				override.ExpectedDigest != "" &&
				override.ExpectedDigest != toolDigest &&
				!override.AllowStaleDigest {
				enabled = false
			}

			out = append(out, runtime.MCPToolCapability{
				Server:           s.server,
				ToolName:         t.Name,
				ProviderToolName: providerToolName(s.server, t.Name),
				ChoiceID:         choiceID(s.server, t.Name),

				Title:       t.Title,
				DisplayName: displayNameForTool(t),
				Description: t.Description,

				InputSchema:  schemaToMap(t.InputSchema),
				OutputSchema: optionalSchemaToMap(t.OutputSchema),

				Annotations:  toolAnnotationsToSpec(t.Annotations),
				InferredRisk: inferRisk(t.Annotations, config.TrustLevel),

				ApprovalRule:  approvalRule,
				ExecutionMode: executionMode,

				TaskSupport: taskSupport,

				App: appInfoFromMeta(t.Meta),

				Digest:  toolDigest,
				Enabled: enabled,
			})
		}

		if res.NextCursor == "" {
			break
		}
		cursor = res.NextCursor
	}

	return out, nil
}

func (s *Session) listAllResources(
	ctx context.Context,
) ([]runtime.MCPResourceRef, error) {
	out := make([]runtime.MCPResourceRef, 0)
	cursor := ""

	for {
		res, err := s.session.ListResources(ctx, &mcpSDK.ListResourcesParams{
			Cursor: cursor,
		})
		if err != nil {
			return nil, err
		}
		if res == nil {
			return nil, fmt.Errorf("%w: resources/list returned nil response", runtime.ErrMCPRuntimeNotReady)
		}
		for _, r := range res.Resources {
			if r == nil {
				continue
			}

			out = append(out, runtime.MCPResourceRef{
				Server:      s.server,
				URI:         r.URI,
				Name:        r.Name,
				Title:       r.Title,
				DisplayName: displayNameFirstNonEmpty(r.Title, r.Name, r.URI),
				Description: r.Description,
				MimeType:    r.MIMEType,
				Size:        r.Size,
				Annotations: annotationsToMap(r.Annotations),
				Digest:      digestAny(r),
			})
		}

		if res.NextCursor == "" {
			break
		}
		cursor = res.NextCursor
	}

	return out, nil
}

func (s *Session) listAllResourceTemplates(
	ctx context.Context,
) ([]runtime.MCPResourceTemplateRef, error) {
	out := make([]runtime.MCPResourceTemplateRef, 0)
	cursor := ""

	for {
		res, err := s.session.ListResourceTemplates(ctx, &mcpSDK.ListResourceTemplatesParams{
			Cursor: cursor,
		})
		if err != nil {
			return nil, err
		}
		if res == nil {
			return nil, fmt.Errorf("%w: resources/templates/list returned nil response", runtime.ErrMCPRuntimeNotReady)
		}
		for _, rt := range res.ResourceTemplates {
			if rt == nil {
				continue
			}

			out = append(out, runtime.MCPResourceTemplateRef{
				Server:      s.server,
				URITemplate: rt.URITemplate,
				Name:        rt.Name,
				Title:       rt.Title,
				DisplayName: displayNameFirstNonEmpty(rt.Title, rt.Name, rt.URITemplate),
				Description: rt.Description,
				MimeType:    rt.MIMEType,
				Arguments:   resourceTemplateArgumentsToSpec(rt.URITemplate),

				Annotations: annotationsToMap(rt.Annotations),
				Digest:      digestAny(rt),
			})
		}

		if res.NextCursor == "" {
			break
		}
		cursor = res.NextCursor
	}

	return out, nil
}

func (s *Session) listAllPrompts(
	ctx context.Context,
) ([]runtime.MCPPromptRef, error) {
	out := make([]runtime.MCPPromptRef, 0)
	cursor := ""

	for {
		res, err := s.session.ListPrompts(ctx, &mcpSDK.ListPromptsParams{
			Cursor: cursor,
		})
		if err != nil {
			return nil, err
		}
		if res == nil {
			return nil, fmt.Errorf("%w: prompts/list returned nil response", runtime.ErrMCPRuntimeNotReady)
		}
		for _, p := range res.Prompts {
			if p == nil {
				continue
			}

			out = append(out, runtime.MCPPromptRef{
				Server:      s.server,
				PromptName:  p.Name,
				Title:       p.Title,
				DisplayName: displayNameFirstNonEmpty(p.Title, p.Name),
				Description: p.Description,
				Arguments:   promptArgumentsToSpec(p.Arguments),
				Digest:      digestAny(p),
			})
		}

		if res.NextCursor == "" {
			break
		}
		cursor = res.NextCursor
	}

	return out, nil
}

func (s *Session) log() *slog.Logger {
	if s != nil && s.logger != nil {
		return s.logger
	}
	return slog.Default()
}

func initResultCapabilities(initResult *mcpSDK.InitializeResult) *mcpSDK.ServerCapabilities {
	if initResult == nil {
		return nil
	}
	return initResult.Capabilities
}

func summarizeCapabilities(caps *mcpSDK.ServerCapabilities) *runtime.MCPServerCapabilitiesSummary {
	if caps == nil {
		return nil
	}

	out := &runtime.MCPServerCapabilitiesSummary{
		Tools:        caps.Tools != nil,
		Resources:    caps.Resources != nil,
		Prompts:      caps.Prompts != nil,
		Completions:  caps.Completions != nil,
		Experimental: cloneMap(caps.Experimental),
		Extensions:   cloneMap(caps.Extensions),
	}

	if caps.Tools != nil {
		out.ToolsListChanged = caps.Tools.ListChanged
	}
	if caps.Resources != nil {
		out.ResourcesSubscribe = caps.Resources.Subscribe
		out.ResourcesListChanged = caps.Resources.ListChanged
	}
	if caps.Prompts != nil {
		out.PromptsListChanged = caps.Prompts.ListChanged
	}

	return out
}

func effectiveToolPolicy(
	config server.RuntimeConfig,
	toolName string,
) (
	policy.MCPApprovalRule,
	policy.MCPExecutionMode,
	policy.MCPToolPolicyOverride,
	bool,
) {
	defaults := policy.DefaultMCPServerPolicy()
	approvalRule := config.DefaultPolicy.DefaultApprovalRule
	executionMode := config.DefaultPolicy.DefaultExecutionMode
	if approvalRule == "" {
		approvalRule = defaults.DefaultApprovalRule
	}
	if executionMode == "" {
		executionMode = defaults.DefaultExecutionMode
	}

	override, found := config.ToolPolicies[toolName]
	if found {
		if override.ApprovalRule != nil {
			approvalRule = *override.ApprovalRule
		}
		if override.ExecutionMode != nil {
			executionMode = *override.ExecutionMode
		}
	}
	return approvalRule, executionMode, override, found
}

func inferRisk(a *mcpSDK.ToolAnnotations, trustLevel policy.MCPTrustLevel) runtime.MCPToolRisk {
	if a == nil {
		return runtime.MCPToolRiskUnknown
	}

	if a.DestructiveHint != nil && *a.DestructiveHint {
		return runtime.MCPToolRiskDestructive
	}
	if a.OpenWorldHint != nil && *a.OpenWorldHint {
		return runtime.MCPToolRiskOpenWorld
	}
	// Do not let untrusted server-provided annotations lower risk.
	if trustLevel != policy.MCPTrustLevelTrusted {
		return runtime.MCPToolRiskUnknown
	}
	if a.ReadOnlyHint {
		return runtime.MCPToolRiskRead
	}
	if a.DestructiveHint != nil && !*a.DestructiveHint {
		return runtime.MCPToolRiskWrite
	}

	return runtime.MCPToolRiskUnknown
}

func appInfoFromMeta(meta mcpSDK.Meta) *runtime.MCPToolAppInfo {
	if len(meta) == 0 {
		return nil
	}

	ui := map[string]any{}
	// The MCP Apps descriptor may live under "_meta.ui" or under the
	// advertised extension id key (io.modelcontextprotocol/ui).
	for _, key := range []string{"ui", runtime.AppExtensionID} {
		if raw, ok := meta[key]; ok && raw != nil {
			for k, v := range anyToMap(raw) {
				if _, exists := ui[k]; !exists {
					ui[k] = v
				}
			}
		}
	}
	// Deprecated MCP Apps shape. Keep accepting it during migration.
	if flatResourceURI, ok := meta["ui/resourceUri"].(string); ok && strings.TrimSpace(flatResourceURI) != "" {
		if _, exists := ui["resourceUri"]; !exists {
			ui["resourceUri"] = flatResourceURI
		}
	}

	if len(ui) == 0 {
		return nil
	}

	out := &runtime.MCPToolAppInfo{}

	if resourceURI, ok := ui["resourceUri"].(string); ok {
		resourceURI = strings.TrimSpace(resourceURI)
		if strings.HasPrefix(resourceURI, "ui://") {
			out.ResourceURI = resourceURI
		}
	}

	out.Visibility = normalizeAppVisibility(stringSliceFromAny(ui["visibility"]))

	hasExplicitVisibility := len(out.Visibility) > 0

	// Only treat this as an MCP App tool when it actually carries a UI
	// resource or an explicit visibility constraint. Never fabricate
	// visibility for ordinary tools.
	if out.ResourceURI == "" && !hasExplicitVisibility {
		return nil
	}
	if len(out.Visibility) == 0 {
		out.Visibility = []string{runtime.VisibilityModel, runtime.VisibilityApp}
	}
	return out
}

func normalizeAppVisibility(in []string) []string {
	if len(in) == 0 {
		return nil
	}

	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, raw := range in {
		v := strings.ToLower(strings.TrimSpace(raw))
		switch v {
		case runtime.VisibilityModel, runtime.VisibilityApp:
		default:
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func taskSupportFromMeta(meta mcpSDK.Meta) runtime.MCPTaskSupport {
	if len(meta) == 0 {
		return runtime.MCPTaskSupportForbidden
	}

	rawExecution, ok := meta["execution"]
	if !ok || rawExecution == nil {
		return runtime.MCPTaskSupportForbidden
	}

	execution := anyToMap(rawExecution)
	if len(execution) == 0 {
		return runtime.MCPTaskSupportForbidden
	}

	switch strings.TrimSpace(fmt.Sprint(execution["taskSupport"])) {
	case string(runtime.MCPTaskSupportRequired):
		return runtime.MCPTaskSupportRequired
	case string(runtime.MCPTaskSupportOptional):
		return runtime.MCPTaskSupportOptional
	case string(runtime.MCPTaskSupportForbidden):
		return runtime.MCPTaskSupportForbidden
	default:
		return runtime.MCPTaskSupportForbidden
	}
}

func completionReference(req runtime.MCPCompleteArgumentRequestBody) (*mcpSDK.CompleteReference, error) {
	switch strings.TrimSpace(strings.ToLower(req.RefType)) {
	case refTypePrompt, refTypeRefPrompt:
		if strings.TrimSpace(req.Name) == "" {
			return nil, fmt.Errorf("%w: completion prompt name required", runtime.ErrMCPInvalidRuntimeRequest)
		}
		return &mcpSDK.CompleteReference{
			Type: refTypeRefPrompt,
			Name: req.Name,
		}, nil

	case refTypeResource, refTypeRefResource:
		if strings.TrimSpace(req.Name) == "" {
			return nil, fmt.Errorf("%w: completion resource uri required", runtime.ErrMCPInvalidRuntimeRequest)
		}
		return &mcpSDK.CompleteReference{
			Type: refTypeRefResource,
			URI:  req.Name,
		}, nil

	default:
		return nil, fmt.Errorf("%w: invalid completion refType %q", runtime.ErrMCPInvalidRuntimeRequest, req.RefType)
	}
}

func promptArgumentsToSpec(in []*mcpSDK.PromptArgument) map[string]runtime.MCPArgumentDefinition {
	if len(in) == 0 {
		return nil
	}

	out := make(map[string]runtime.MCPArgumentDefinition, len(in))
	for _, arg := range in {
		if arg == nil || strings.TrimSpace(arg.Name) == "" {
			continue
		}
		name := strings.TrimSpace(arg.Name)
		out[name] = runtime.MCPArgumentDefinition{
			Name:        name,
			Description: arg.Description,
			Required:    arg.Required,
		}
	}

	if len(out) == 0 {
		return nil
	}
	return out
}

func resourceTemplateArgumentsToSpec(uriTemplate string) map[string]runtime.MCPArgumentDefinition {
	matches := uriTemplateVariableRE.FindAllStringSubmatch(uriTemplate, -1)
	if len(matches) == 0 {
		return nil
	}

	out := map[string]runtime.MCPArgumentDefinition{}
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		name := strings.TrimSpace(match[1])
		if name == "" {
			continue
		}
		out[name] = runtime.MCPArgumentDefinition{
			Name:     name,
			Required: true,
		}
	}

	if len(out) == 0 {
		return nil
	}
	return out
}

func contentSliceToSpec(in []mcpSDK.Content) []runtime.MCPContent {
	if len(in) == 0 {
		return nil
	}
	out := make([]runtime.MCPContent, 0, len(in))
	for _, c := range in {
		if c == nil {
			continue
		}
		out = append(out, contentToSpec(c))
	}
	return out
}

func contentToSpec(c mcpSDK.Content) runtime.MCPContent {
	switch v := c.(type) {
	case *mcpSDK.TextContent:
		return runtime.MCPContent{
			Type:        runtime.MCPContentTypeText,
			Text:        v.Text,
			Meta:        cloneMap(v.Meta),
			Annotations: annotationsToMap(v.Annotations),
		}
	case *mcpSDK.ImageContent:
		return runtime.MCPContent{
			Type:        runtime.MCPContentTypeImage,
			Data:        append([]byte(nil), v.Data...),
			MIMEType:    v.MIMEType,
			Meta:        cloneMap(v.Meta),
			Annotations: annotationsToMap(v.Annotations),
		}
	case *mcpSDK.AudioContent:
		return runtime.MCPContent{
			Type:        runtime.MCPContentTypeAudio,
			Data:        append([]byte(nil), v.Data...),
			MIMEType:    v.MIMEType,
			Meta:        cloneMap(v.Meta),
			Annotations: annotationsToMap(v.Annotations),
		}
	case *mcpSDK.ResourceLink:
		return runtime.MCPContent{
			Type:        runtime.MCPContentTypeResourceLink,
			URI:         v.URI,
			Name:        v.Name,
			Title:       v.Title,
			Description: v.Description,
			MIMEType:    v.MIMEType,
			Size:        v.Size,
			Meta:        cloneMap(v.Meta),
			Annotations: annotationsToMap(v.Annotations),
			Icons:       iconsToSpec(v.Icons),
		}
	case *mcpSDK.EmbeddedResource:
		return runtime.MCPContent{
			Type:        runtime.MCPContentTypeResource,
			Resource:    resourceContentsToSpec(v.Resource),
			Meta:        cloneMap(v.Meta),
			Annotations: annotationsToMap(v.Annotations),
		}
	default:
		raw, err := json.Marshal(c)
		if err != nil {
			return runtime.MCPContent{
				Type: runtime.MCPContentTypeText,
				Text: fmt.Sprintf("%T", c),
			}
		}
		return runtime.MCPContent{
			Type: runtime.MCPContentTypeText,
			Text: string(raw),
		}
	}
}

func resourceContentsToSpec(rc *mcpSDK.ResourceContents) *runtime.MCPResourceContents {
	if rc == nil {
		return nil
	}
	return &runtime.MCPResourceContents{
		URI:      rc.URI,
		MIMEType: rc.MIMEType,
		Text:     rc.Text,
		Blob:     append([]byte(nil), rc.Blob...),
		Meta:     cloneMap(rc.Meta),
	}
}

func iconsToSpec(in []mcpSDK.Icon) []runtime.MCPIcon {
	if len(in) == 0 {
		return nil
	}
	out := make([]runtime.MCPIcon, 0, len(in))
	for _, icon := range in {
		out = append(out, runtime.MCPIcon{
			Source:   icon.Source,
			MIMEType: icon.MIMEType,
			Sizes:    slices.Clone(icon.Sizes),
			Theme:    string(icon.Theme),
		})
	}
	return out
}

func toolAnnotationsToSpec(a *mcpSDK.ToolAnnotations) *runtime.MCPToolAnnotations {
	if a == nil {
		return nil
	}
	return &runtime.MCPToolAnnotations{
		DestructiveHint: a.DestructiveHint,
		IdempotentHint:  a.IdempotentHint,
		OpenWorldHint:   a.OpenWorldHint,
		ReadOnlyHint:    a.ReadOnlyHint,
		Title:           a.Title,
	}
}

func schemaToMap(v any) map[string]any {
	return schemaToMapWithFallback(v, getEmptySchema())
}

func optionalSchemaToMap(v any) map[string]any {
	return schemaToMapWithFallback(v, nil)
}

func schemaToMapWithFallback(v any, fallback map[string]any) map[string]any {
	if v == nil {
		return cloneMap(fallback)
	}

	var out map[string]any
	raw, err := json.Marshal(v)
	if err != nil {
		return cloneMap(fallback)
	}
	if err := json.Unmarshal(raw, &out); err != nil || out == nil {
		return cloneMap(fallback)
	}
	return out
}

func annotationsToMap(a *mcpSDK.Annotations) map[string]any {
	if a == nil {
		return nil
	}
	return anyToMap(a)
}

func anyToMap(v any) map[string]any {
	if v == nil {
		return nil
	}

	if m, ok := v.(map[string]any); ok {
		return cloneMap(m)
	}

	raw, err := json.Marshal(v)
	if err != nil {
		return nil
	}

	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

func stringSliceFromAny(v any) []string {
	switch x := v.(type) {
	case []string:
		return append([]string(nil), x...)

	case []any:
		out := make([]string, 0, len(x))
		for _, item := range x {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out

	default:
		return nil
	}
}

func digestAny(v any) string {
	raw, _ := json.Marshal(v)
	return string(cryptoutil.DigestBytes(raw))
}

func displayNameForTool(t *mcpSDK.Tool) string {
	if t == nil {
		return ""
	}
	if strings.TrimSpace(t.Title) != "" {
		return t.Title
	}
	if t.Annotations != nil && strings.TrimSpace(t.Annotations.Title) != "" {
		return t.Annotations.Title
	}
	return t.Name
}

func displayNameFirstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func cloneMap(m map[string]any) map[string]any {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]any, len(m))
	maps.Copy(out, m)
	return out
}

func getEmptySchema() map[string]any {
	return map[string]any{"type": "object"}
}
