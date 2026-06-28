package sdkclient

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/mcp/apps"
	"github.com/flexigpt/flexigpt-app/internal/mcp/auth"
	"github.com/flexigpt/flexigpt-app/internal/mcp/runtime"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
	mcpSDK "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	defaultStdioTerminateDuration = 5 * time.Second
	defaultHTTPMaxRetries         = 5
	defaultClientKeepAlive        = 60 * time.Second
	refTypePrompt                 = "prompt"
	refTypeRefPrompt              = "ref/prompt"
	refTypeResource               = "resource"
	refTypeRefResource            = "ref/resource"
)

type Factory struct {
	logger *slog.Logger
}

func NewFactory() *Factory {
	return &Factory{logger: slog.Default()}
}

func NewFactoryWithLogger(logger *slog.Logger) *Factory {
	if logger == nil {
		logger = slog.Default()
	}
	return &Factory{logger: logger}
}

func (f *Factory) Connect(
	ctx context.Context,
	cfg spec.MCPServerConfig,
	resolved auth.ResolvedTransportAuth,
	events runtime.ClientNotificationSink,
) (runtime.ClientSession, error) {
	if cfg.BundleID == "" {
		return nil, fmt.Errorf("%w: bundleID required", spec.ErrMCPInvalidRequest)
	}
	if cfg.ID == "" {
		return nil, fmt.Errorf("%w: serverID required", spec.ErrMCPInvalidRequest)
	}

	logger := f.log()
	emit := func(ctx context.Context, event runtime.ClientNotification) {
		if events != nil {
			events.OnClientNotification(ctx, event)
		}
	}
	client := mcpSDK.NewClient(
		&mcpSDK.Implementation{
			Name:    spec.MCPHostName,
			Version: spec.MCPHostVersion,
		},
		&mcpSDK.ClientOptions{
			Logger: logger,

			// Important:
			// A nil Capabilities value makes the SDK advertise its historical
			// roots default. FlexiGPT must not advertise roots, sampling, or
			// elicitation until the product has explicit UX and safety policy.
			Capabilities: buildClientCapabilities(cfg),

			KeepAlive: defaultClientKeepAlive,

			ToolListChangedHandler: func(ctx context.Context, req *mcpSDK.ToolListChangedRequest) {
				logger.Info("mcp tools list changed", "sessionID", safeClientSessionID(req))
				emit(ctx, runtime.ClientNotification{
					BundleID: cfg.BundleID,
					ServerID: cfg.ID,
					Kind:     runtime.ClientNotificationToolListChanged,
				})
			},
			PromptListChangedHandler: func(ctx context.Context, req *mcpSDK.PromptListChangedRequest) {
				logger.Info("mcp prompts list changed", "sessionID", safeClientSessionID(req))
				emit(ctx, runtime.ClientNotification{
					BundleID: cfg.BundleID,
					ServerID: cfg.ID,
					Kind:     runtime.ClientNotificationPromptListChanged,
				})
			},
			ResourceListChangedHandler: func(ctx context.Context, req *mcpSDK.ResourceListChangedRequest) {
				logger.Info("mcp resources list changed", "sessionID", safeClientSessionID(req))
				emit(ctx, runtime.ClientNotification{
					BundleID: cfg.BundleID,
					ServerID: cfg.ID,
					Kind:     runtime.ClientNotificationResourceListChanged,
				})
			},
			ResourceUpdatedHandler: func(ctx context.Context, req *mcpSDK.ResourceUpdatedNotificationRequest) {
				uri := ""
				if req != nil && req.Params != nil {
					uri = req.Params.URI
				}
				emit(ctx, runtime.ClientNotification{
					BundleID:    cfg.BundleID,
					ServerID:    cfg.ID,
					Kind:        runtime.ClientNotificationResourceUpdated,
					ResourceURI: uri,
				})
			},
			ProgressNotificationHandler: func(ctx context.Context, req *mcpSDK.ProgressNotificationClientRequest) {
				if req == nil || req.Params == nil {
					return
				}
				emit(ctx, runtime.ClientNotification{
					BundleID: cfg.BundleID,
					ServerID: cfg.ID,
					Kind:     runtime.ClientNotificationProgress,
					Progress: req.Params.Progress,
					Total:    req.Params.Total,
					Message:  req.Params.Message,
				})
			},
		},
	)

	var transport mcpSDK.Transport

	switch cfg.Transport {
	case spec.MCPTransportStdio:
		if cfg.Stdio == nil {
			return nil, fmt.Errorf("%w: missing stdio config", spec.ErrMCPInvalidRequest)
		}

		// Do not use exec.CommandContext here.
		//
		// The connect context is normally a startup timeout context. It is
		// canceled immediately after Connect returns. If we used CommandContext,
		// that cancellation would kill the long-lived MCP stdio server right after
		// a successful connection.
		//
		// Process shutdown is instead owned by mcp.CommandTransport.Close.
		//nolint:gosec,noctx // User-configured MCP stdio command is intentional.
		cmd := exec.Command(cfg.Stdio.Command, cfg.Stdio.Args...)
		if strings.TrimSpace(cfg.Stdio.WorkingDir) != "" {
			cmd.Dir = cfg.Stdio.WorkingDir
		}

		// Do not inherit the full environment. The store/auth layer already
		// resolved explicit env and secret env refs into resolved.Env.
		cmd.Env = envMapToList(resolved.Env)

		cmd.Stderr = newSlogLineWriter(
			logger,
			string(cfg.ID),
			"mcp stdio stderr",
			auth.NewSecretRedactor(resolved),
		)

		transport = &mcpSDK.CommandTransport{
			Command:           cmd,
			TerminateDuration: defaultStdioTerminateDuration,
		}

	case spec.MCPTransportStreamableHTTP:
		if cfg.StreamableHTTP == nil {
			return nil, fmt.Errorf("%w: missing streamableHttp config", spec.ErrMCPInvalidRequest)
		}

		// Do not set http.Client.Timeout here. Streamable HTTP may keep a
		// standalone SSE GET open for the life of the session. Use per-operation
		// contexts in the runtime layer for request bounds.
		httpClient := newStreamableHTTPClient(resolved.Headers)

		transport = &mcpSDK.StreamableClientTransport{
			Endpoint:   cfg.StreamableHTTP.URL,
			HTTPClient: httpClient,
			MaxRetries: defaultHTTPMaxRetries,
			// This needs to be false for receiving server notifications.
			DisableStandaloneSSE: false,
			OAuthHandler:         resolved.OAuthHandler,
		}

	default:
		return nil, fmt.Errorf("%w: unsupported transport %s", spec.ErrMCPInvalidRequest, cfg.Transport)
	}

	// App-only compatibility workaround:
	// the upstream SDK currently probes the sessionless server/discover path
	// first and has no exported option to force legacy initialize. Wrap the
	// transport so the probe gets a local method-not-found response and the SDK
	// falls back to legacy initialize immediately.
	transport = preferLegacyInitializeTransport(transport)

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, err
	}

	return &Session{
		bundleID: cfg.BundleID,
		serverID: cfg.ID,
		session:  session,
		logger:   logger,
	}, nil
}

type headerRoundTripper struct {
	base    http.RoundTripper
	headers map[string]string
}

func (t *headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	if len(t.headers) == 0 {
		return base.RoundTrip(req)
	}
	cloned := req.Clone(req.Context())
	cloned.Header = req.Header.Clone()
	for key, value := range t.headers {
		cloned.Header.Set(key, value)
	}
	return base.RoundTrip(cloned)
}

func (f *Factory) log() *slog.Logger {
	if f != nil && f.logger != nil {
		return f.logger
	}
	return slog.Default()
}

func newStreamableHTTPClient(headers map[string]string) *http.Client {
	if len(headers) == 0 {
		return &http.Client{}
	}
	return &http.Client{
		Transport: &headerRoundTripper{base: http.DefaultTransport, headers: maps.Clone(headers)},
	}
}

// buildClientCapabilities returns the client capability set advertised on
// MCP initialize. FlexiGPT does not advertise roots, sampling, or elicitation.
// It advertises the MCP Apps extension only when Apps are enabled for this
// server config.
func buildClientCapabilities(cfg spec.MCPServerConfig) *mcpSDK.ClientCapabilities {
	c := &mcpSDK.ClientCapabilities{}
	if apps.EffectiveAppsPolicy(cfg).Enabled {
		c.AddExtension(apps.AppExtensionID, map[string]any{
			"mimeTypes": []string{apps.AppMIMEType},
			"host": map[string]any{
				"platform": "desktop",
			},
		})
	}

	return c
}

func envMapToList(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for _, k := range slices.Sorted(maps.Keys(m)) {

		if strings.TrimSpace(k) == "" {
			continue
		}
		out = append(out, k+"="+m[k])

	}
	return out
}

func safeClientSessionID(req any) string {
	if req == nil {
		return ""
	}

	type getSession interface {
		GetSession() mcpSDK.Session
	}

	if r, ok := req.(getSession); ok {
		if sess := r.GetSession(); sess != nil {
			return sess.ID()
		}
	}

	return ""
}
