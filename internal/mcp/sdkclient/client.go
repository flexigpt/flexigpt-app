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

	mcpSDK "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/flexigpt/flexigpt-app/internal/mcp/auth"
	"github.com/flexigpt/flexigpt-app/internal/mcp/runtime"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

const (
	defaultStdioTerminateDuration = 5 * time.Second
	defaultHTTPMaxRetries         = 5
	defaultClientKeepAlive        = 60 * time.Second
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
			Capabilities: &mcpSDK.ClientCapabilities{},

			KeepAlive: defaultClientKeepAlive,

			ToolListChangedHandler: func(ctx context.Context, req *mcpSDK.ToolListChangedRequest) {
				logger.Info("mcp tools list changed", "sessionID", safeClientSessionID(req))
				emit(ctx, runtime.ClientNotification{
					ServerID: cfg.ID,
					Kind:     runtime.ClientNotificationToolListChanged,
				})
			},
			PromptListChangedHandler: func(ctx context.Context, req *mcpSDK.PromptListChangedRequest) {
				logger.Info("mcp prompts list changed", "sessionID", safeClientSessionID(req))
				emit(ctx, runtime.ClientNotification{
					ServerID: cfg.ID,
					Kind:     runtime.ClientNotificationPromptListChanged,
				})
			},
			ResourceListChangedHandler: func(ctx context.Context, req *mcpSDK.ResourceListChangedRequest) {
				logger.Info("mcp resources list changed", "sessionID", safeClientSessionID(req))
				emit(ctx, runtime.ClientNotification{
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
					ServerID:    cfg.ID,
					Kind:        runtime.ClientNotificationResourceUpdated,
					ResourceURI: uri,
				})
			},
			LoggingMessageHandler: func(ctx context.Context, req *mcpSDK.LoggingMessageRequest) {
				if req == nil || req.Params == nil {
					return
				}

				emit(ctx, runtime.ClientNotification{
					ServerID:     cfg.ID,
					Kind:         runtime.ClientNotificationLoggingMessage,
					LoggerName:   req.Params.Logger,
					LoggingLevel: string(req.Params.Level),
					LogData:      req.Params.Data,
				})
			},
			ProgressNotificationHandler: func(ctx context.Context, req *mcpSDK.ProgressNotificationClientRequest) {
				if req == nil || req.Params == nil {
					return
				}
				emit(ctx, runtime.ClientNotification{
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
		httpClient := &http.Client{
			Transport: headerRoundTripper{
				base:    http.DefaultTransport,
				headers: resolved.Headers,
			},
		}

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

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, err
	}

	return &Session{
		session: session,
		logger:  logger,
	}, nil
}

func (f *Factory) log() *slog.Logger {
	if f != nil && f.logger != nil {
		return f.logger
	}
	return slog.Default()
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
