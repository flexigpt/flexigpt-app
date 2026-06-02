package sdkclient

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/mcp/runtime"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
	mcpSDK "github.com/modelcontextprotocol/go-sdk/mcp"
)

var uriTemplateVariableRE = regexp.MustCompile(`\{([A-Za-z_][A-Za-z0-9_.-]*)\}`)

type Session struct {
	bundleID bundleitemutils.BundleID
	serverID spec.MCPServerID
	session  *mcpSDK.ClientSession
	logger   *slog.Logger
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

func (s *Session) Ping(ctx context.Context) error {
	if s == nil || s.session == nil {
		return fmt.Errorf("%w: nil session", spec.ErrMCPRuntimeNotReady)
	}
	return s.session.Ping(ctx, nil)
}

func (s *Session) Discover(
	ctx context.Context,
	serverID spec.MCPServerID,
	defaultPolicy spec.MCPServerPolicy,
	trustLevel spec.MCPTrustLevel,
) (spec.MCPDiscoverySnapshot, error) {
	if s == nil || s.session == nil {
		return spec.MCPDiscoverySnapshot{}, fmt.Errorf("%w: nil session", spec.ErrMCPRuntimeNotReady)
	}

	if serverID == "" {
		serverID = s.serverID
	}

	out := spec.MCPDiscoverySnapshot{
		BundleID: s.bundleID,
		ServerID: serverID,
	}

	initResult := s.session.InitializeResult()
	if initResult != nil {
		out.NegotiatedProtocolVersion = initResult.ProtocolVersion
		out.Instructions = initResult.Instructions

		if initResult.ServerInfo != nil {
			out.ServerInfo = &spec.MCPImplementationInfo{
				Name:    initResult.ServerInfo.Name,
				Version: initResult.ServerInfo.Version,
			}
		}

		if initResult.Capabilities != nil {
			out.ServerCapabilities = summarizeCapabilities(initResult.Capabilities)
		}
	}

	caps := initResultCapabilities(initResult)

	if caps == nil || caps.Tools != nil {
		tools, err := s.listAllTools(ctx, serverID, defaultPolicy, trustLevel)
		if err != nil {
			s.log().Warn("mcp tools discovery failed", "serverID", serverID, "err", err)
		} else {
			out.Tools = tools
		}
	}

	if caps == nil || caps.Resources != nil {
		resources, err := s.listAllResources(ctx, serverID)
		if err != nil {
			s.log().Warn("mcp resources discovery failed", "serverID", serverID, "err", err)
		} else {
			out.Resources = resources
		}

		templates, err := s.listAllResourceTemplates(ctx, serverID)
		if err != nil {
			s.log().Warn("mcp resource templates discovery failed", "serverID", serverID, "err", err)
		} else {
			out.ResourceTemplates = templates
		}
	}

	if caps == nil || caps.Prompts != nil {
		prompts, err := s.listAllPrompts(ctx, serverID)
		if err != nil {
			s.log().Warn("mcp prompts discovery failed", "serverID", serverID, "err", err)
		} else {
			out.Prompts = prompts
		}
	}

	return out, nil
}

func (s *Session) CallTool(
	ctx context.Context,
	toolName string,
	args map[string]any,
) (*spec.InvokeMCPToolResponseBody, error) {
	if s == nil || s.session == nil {
		return nil, fmt.Errorf("%w: nil session", spec.ErrMCPRuntimeNotReady)
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

	return &spec.InvokeMCPToolResponseBody{
		BundleID:          s.bundleID,
		ServerID:          s.serverID,
		ToolName:          toolName,
		Content:           contentSliceToSpec(res.Content),
		StructuredContent: res.StructuredContent,
		IsError:           res.IsError,
	}, nil
}

func (s *Session) ReadResource(
	ctx context.Context,
	uri string,
) (*spec.MCPReadResourceResponseBody, error) {
	if s == nil || s.session == nil {
		return nil, fmt.Errorf("%w: nil session", spec.ErrMCPRuntimeNotReady)
	}

	res, err := s.session.ReadResource(ctx, &mcpSDK.ReadResourceParams{
		URI: uri,
	})
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, fmt.Errorf("%w: resource read returned nil response", spec.ErrMCPRuntimeNotReady)
	}

	contents := make([]spec.MCPContent, 0, len(res.Contents))
	for _, rc := range res.Contents {
		if rc == nil {
			continue
		}
		contents = append(contents, contentToSpec(&mcpSDK.EmbeddedResource{Resource: rc}))
	}

	return &spec.MCPReadResourceResponseBody{
		BundleID: s.bundleID,
		ServerID: s.serverID,
		URI:      uri,
		Contents: contents,
	}, nil
}

func (s *Session) GetPrompt(
	ctx context.Context,
	name string,
	args map[string]string,
) (*spec.MCPGetPromptResponseBody, error) {
	if s == nil || s.session == nil {
		return nil, fmt.Errorf("%w: nil session", spec.ErrMCPRuntimeNotReady)
	}

	res, err := s.session.GetPrompt(ctx, &mcpSDK.GetPromptParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, fmt.Errorf("%w: prompt read returned nil response", spec.ErrMCPRuntimeNotReady)
	}
	messages := make([]spec.MCPPromptMessage, 0, len(res.Messages))
	for _, msg := range res.Messages {
		if msg == nil {
			continue
		}
		messages = append(messages, spec.MCPPromptMessage{
			Role:    string(msg.Role),
			Content: contentToSpec(msg.Content),
		})
	}

	return &spec.MCPGetPromptResponseBody{
		BundleID:    s.bundleID,
		ServerID:    s.serverID,
		PromptName:  name,
		Description: res.Description,
		Messages:    messages,
	}, nil
}

func (s *Session) Complete(
	ctx context.Context,
	req spec.MCPCompleteArgumentRequestBody,
) (*spec.MCPCompletionResult, error) {
	if s == nil || s.session == nil {
		return nil, fmt.Errorf("%w: nil session", spec.ErrMCPRuntimeNotReady)
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
		return nil, fmt.Errorf("%w: completion returned nil response", spec.ErrMCPRuntimeNotReady)
	}
	return &spec.MCPCompletionResult{
		Values:  res.Completion.Values,
		Total:   res.Completion.Total,
		HasMore: res.Completion.HasMore,
	}, nil
}

func (s *Session) listAllTools(
	ctx context.Context,
	serverID spec.MCPServerID,
	defaultPolicy spec.MCPServerPolicy,
	trustLevel spec.MCPTrustLevel,
) ([]spec.MCPToolCapability, error) {
	if defaultPolicy == (spec.MCPServerPolicy{}) {
		defaultPolicy = spec.DefaultMCPServerPolicy()
	}

	out := make([]spec.MCPToolCapability, 0)
	cursor := ""

	for {
		res, err := s.session.ListTools(ctx, &mcpSDK.ListToolsParams{
			Cursor: cursor,
		})
		if err != nil {
			return nil, err
		}

		for _, t := range res.Tools {
			if t == nil {
				continue
			}

			taskSupport := taskSupportFromMeta(t.Meta)

			out = append(out, spec.MCPToolCapability{
				BundleID:         s.bundleID,
				ServerID:         serverID,
				ToolName:         t.Name,
				ProviderToolName: runtime.ProviderToolName(serverID, t.Name),
				ChoiceID:         runtime.ChoiceID(serverID, t.Name),

				Title:       t.Title,
				DisplayName: displayNameForTool(t),
				Description: t.Description,

				InputSchema:  schemaToMap(t.InputSchema),
				OutputSchema: optionalSchemaToMap(t.OutputSchema),

				Annotations:  toolAnnotationsToSpec(t.Annotations),
				InferredRisk: inferRisk(t.Annotations, trustLevel),

				ApprovalRule:  defaultPolicy.DefaultApprovalRule,
				ExecutionMode: defaultPolicy.DefaultExecutionMode,

				TaskSupport: taskSupport,

				App: appInfoFromMeta(t.Meta),

				Digest: digestAny(t),
				// Task-required tools are surfaced as disabled because this app
				// does not support MCP task augmentation.
				Enabled: taskSupport != spec.MCPTaskSupportRequired,
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
	serverID spec.MCPServerID,
) ([]spec.MCPResourceRef, error) {
	out := make([]spec.MCPResourceRef, 0)
	cursor := ""

	for {
		res, err := s.session.ListResources(ctx, &mcpSDK.ListResourcesParams{
			Cursor: cursor,
		})
		if err != nil {
			return nil, err
		}

		for _, r := range res.Resources {
			if r == nil {
				continue
			}

			out = append(out, spec.MCPResourceRef{
				BundleID:    s.bundleID,
				ServerID:    serverID,
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
	serverID spec.MCPServerID,
) ([]spec.MCPResourceTemplateRef, error) {
	out := make([]spec.MCPResourceTemplateRef, 0)
	cursor := ""

	for {
		res, err := s.session.ListResourceTemplates(ctx, &mcpSDK.ListResourceTemplatesParams{
			Cursor: cursor,
		})
		if err != nil {
			return nil, err
		}

		for _, rt := range res.ResourceTemplates {
			if rt == nil {
				continue
			}

			out = append(out, spec.MCPResourceTemplateRef{
				BundleID:    s.bundleID,
				ServerID:    serverID,
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
	serverID spec.MCPServerID,
) ([]spec.MCPPromptRef, error) {
	out := make([]spec.MCPPromptRef, 0)
	cursor := ""

	for {
		res, err := s.session.ListPrompts(ctx, &mcpSDK.ListPromptsParams{
			Cursor: cursor,
		})
		if err != nil {
			return nil, err
		}

		for _, p := range res.Prompts {
			if p == nil {
				continue
			}

			out = append(out, spec.MCPPromptRef{
				BundleID:    s.bundleID,
				ServerID:    serverID,
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

func summarizeCapabilities(caps *mcpSDK.ServerCapabilities) *spec.MCPServerCapabilitiesSummary {
	if caps == nil {
		return nil
	}

	out := &spec.MCPServerCapabilitiesSummary{
		Tools:        caps.Tools != nil,
		Resources:    caps.Resources != nil,
		Prompts:      caps.Prompts != nil,
		Logging:      caps.Logging != nil,
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

func inferRisk(a *mcpSDK.ToolAnnotations, trustLevel spec.MCPTrustLevel) spec.MCPToolRisk {
	if a == nil {
		return spec.MCPToolRiskUnknown
	}

	if a.DestructiveHint != nil && *a.DestructiveHint {
		return spec.MCPToolRiskDestructive
	}
	if a.OpenWorldHint != nil && *a.OpenWorldHint {
		return spec.MCPToolRiskOpenWorld
	}
	// Do not let untrusted server-provided annotations lower risk.
	if trustLevel != spec.MCPTrustLevelTrusted {
		return spec.MCPToolRiskUnknown
	}
	if a.ReadOnlyHint {
		return spec.MCPToolRiskRead
	}
	if a.DestructiveHint != nil && !*a.DestructiveHint {
		return spec.MCPToolRiskWrite
	}

	return spec.MCPToolRiskUnknown
}

func appInfoFromMeta(meta mcpSDK.Meta) *spec.MCPToolAppInfo {
	if len(meta) == 0 {
		return nil
	}

	rawUI, ok := meta["ui"]
	if !ok || rawUI == nil {
		return nil
	}

	ui := anyToMap(rawUI)
	if len(ui) == 0 {
		return nil
	}

	out := &spec.MCPToolAppInfo{}

	if resourceURI, ok := ui["resourceUri"].(string); ok {
		out.ResourceURI = resourceURI
	}

	out.Visibility = stringSliceFromAny(ui["visibility"])
	if len(out.Visibility) == 0 {
		out.Visibility = []string{visibilityModel, visibilityApp}
	}

	return out
}

func taskSupportFromMeta(meta mcpSDK.Meta) spec.MCPTaskSupport {
	if len(meta) == 0 {
		return spec.MCPTaskSupportForbidden
	}

	rawExecution, ok := meta["execution"]
	if !ok || rawExecution == nil {
		return spec.MCPTaskSupportForbidden
	}

	execution := anyToMap(rawExecution)
	if len(execution) == 0 {
		return spec.MCPTaskSupportForbidden
	}

	switch strings.TrimSpace(fmt.Sprint(execution["taskSupport"])) {
	case string(spec.MCPTaskSupportRequired):
		return spec.MCPTaskSupportRequired
	case string(spec.MCPTaskSupportOptional):
		return spec.MCPTaskSupportOptional
	case string(spec.MCPTaskSupportForbidden):
		return spec.MCPTaskSupportForbidden
	default:
		return spec.MCPTaskSupportForbidden
	}
}

func completionReference(req spec.MCPCompleteArgumentRequestBody) (*mcpSDK.CompleteReference, error) {
	switch strings.TrimSpace(strings.ToLower(req.RefType)) {
	case refTypePrompt, refTypeRefPrompt:
		if strings.TrimSpace(req.Name) == "" {
			return nil, fmt.Errorf("%w: completion prompt name required", spec.ErrMCPInvalidRequest)
		}
		return &mcpSDK.CompleteReference{
			Type: refTypeRefPrompt,
			Name: req.Name,
		}, nil

	case refTypeResource, refTypeRefResource:
		if strings.TrimSpace(req.Name) == "" {
			return nil, fmt.Errorf("%w: completion resource uri required", spec.ErrMCPInvalidRequest)
		}
		return &mcpSDK.CompleteReference{
			Type: refTypeRefResource,
			URI:  req.Name,
		}, nil

	default:
		return nil, fmt.Errorf("%w: invalid completion refType %q", spec.ErrMCPInvalidRequest, req.RefType)
	}
}

func promptArgumentsToSpec(in []*mcpSDK.PromptArgument) map[string]spec.MCPArgumentDefinition {
	if len(in) == 0 {
		return nil
	}

	out := make(map[string]spec.MCPArgumentDefinition, len(in))
	for _, arg := range in {
		if arg == nil || strings.TrimSpace(arg.Name) == "" {
			continue
		}
		name := strings.TrimSpace(arg.Name)
		out[name] = spec.MCPArgumentDefinition{
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

func resourceTemplateArgumentsToSpec(uriTemplate string) map[string]spec.MCPArgumentDefinition {
	matches := uriTemplateVariableRE.FindAllStringSubmatch(uriTemplate, -1)
	if len(matches) == 0 {
		return nil
	}

	out := map[string]spec.MCPArgumentDefinition{}
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		name := strings.TrimSpace(match[1])
		if name == "" {
			continue
		}
		out[name] = spec.MCPArgumentDefinition{
			Name:     name,
			Required: true,
		}
	}

	if len(out) == 0 {
		return nil
	}
	return out
}

func contentSliceToSpec(in []mcpSDK.Content) []spec.MCPContent {
	if len(in) == 0 {
		return nil
	}
	out := make([]spec.MCPContent, 0, len(in))
	for _, c := range in {
		if c == nil {
			continue
		}
		out = append(out, contentToSpec(c))
	}
	return out
}

func contentToSpec(c mcpSDK.Content) spec.MCPContent {
	switch v := c.(type) {
	case *mcpSDK.TextContent:
		return spec.MCPContent{
			Type:        spec.MCPContentTypeText,
			Text:        v.Text,
			Meta:        cloneMap(v.Meta),
			Annotations: annotationsToMap(v.Annotations),
		}
	case *mcpSDK.ImageContent:
		return spec.MCPContent{
			Type:        spec.MCPContentTypeImage,
			Data:        append([]byte(nil), v.Data...),
			MIMEType:    v.MIMEType,
			Meta:        cloneMap(v.Meta),
			Annotations: annotationsToMap(v.Annotations),
		}
	case *mcpSDK.AudioContent:
		return spec.MCPContent{
			Type:        spec.MCPContentTypeAudio,
			Data:        append([]byte(nil), v.Data...),
			MIMEType:    v.MIMEType,
			Meta:        cloneMap(v.Meta),
			Annotations: annotationsToMap(v.Annotations),
		}
	case *mcpSDK.ResourceLink:
		return spec.MCPContent{
			Type:        spec.MCPContentTypeResourceLink,
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
		return spec.MCPContent{
			Type:        spec.MCPContentTypeResource,
			Resource:    resourceContentsToSpec(v.Resource),
			Meta:        cloneMap(v.Meta),
			Annotations: annotationsToMap(v.Annotations),
		}
	default:
		raw, err := json.Marshal(c)
		if err != nil {
			return spec.MCPContent{
				Type: spec.MCPContentTypeText,
				Text: fmt.Sprintf("%T", c),
			}
		}
		return spec.MCPContent{
			Type: spec.MCPContentTypeText,
			Text: string(raw),
		}
	}
}

func resourceContentsToSpec(rc *mcpSDK.ResourceContents) *spec.MCPResourceContents {
	if rc == nil {
		return nil
	}
	return &spec.MCPResourceContents{
		URI:      rc.URI,
		MIMEType: rc.MIMEType,
		Text:     rc.Text,
		Blob:     append([]byte(nil), rc.Blob...),
		Meta:     cloneMap(rc.Meta),
	}
}

func iconsToSpec(in []mcpSDK.Icon) []spec.MCPIcon {
	if len(in) == 0 {
		return nil
	}
	out := make([]spec.MCPIcon, 0, len(in))
	for _, icon := range in {
		out = append(out, spec.MCPIcon{
			Source:   icon.Source,
			MIMEType: icon.MIMEType,
			Sizes:    slices.Clone(icon.Sizes),
			Theme:    string(icon.Theme),
		})
	}
	return out
}

func toolAnnotationsToSpec(a *mcpSDK.ToolAnnotations) *spec.MCPToolAnnotations {
	if a == nil {
		return nil
	}
	return &spec.MCPToolAnnotations{
		DestructiveHint: a.DestructiveHint,
		IdempotentHint:  a.IdempotentHint,
		OpenWorldHint:   a.OpenWorldHint,
		ReadOnlyHint:    a.ReadOnlyHint,
		Title:           a.Title,
	}
}

func schemaToMap(v any) map[string]any {
	return schemaToMapWithFallback(v, map[string]any{"type": "object"})
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
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
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
