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

	"github.com/modelcontextprotocol/go-sdk/auth"
	mcpSDK "github.com/modelcontextprotocol/go-sdk/mcp"

	mcpApps "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/apps"
	mcpAuth "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/auth"
	mcpConnection "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/connection"
	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/policy"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/server"
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
	logger     *slog.Logger
	clientInfo mcpServer.ClientInfo
}

func NewFactory(
	clientInfo mcpServer.ClientInfo,
) (*Factory, error) {
	if err := clientInfo.Validate(); err != nil {
		return nil, err
	}
	return &Factory{
		logger:     slog.Default(),
		clientInfo: clientInfo,
	}, nil
}

func NewFactoryWithLogger(
	clientInfo mcpServer.ClientInfo,
	logger *slog.Logger,
) (*Factory, error) {
	if err := clientInfo.Validate(); err != nil {
		return nil, err
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Factory{
		logger:     logger,
		clientInfo: clientInfo,
	}, nil
}

func (f *Factory) Connect(
	ctx context.Context,
	cfg mcpServer.RuntimeConfig,
	resolved mcpConnection.PreparedConnection,
	events mcpConnection.ClientNotificationSink,
) (mcpConnection.ClientSession, error) {
	if err := cfg.Server.Validate(); err != nil {
		return nil, err
	}
	logger := f.log()
	emit := func(ctx context.Context, event mcpServer.ClientNotification) {
		if events != nil {
			events.OnClientNotification(ctx, event)
		}
	}
	client := mcpSDK.NewClient(
		&mcpSDK.Implementation{
			Name:    f.clientInfo.Name,
			Version: f.clientInfo.Version,
		},
		&mcpSDK.ClientOptions{
			Logger: logger,

			// Important:
			// A nil Capabilities value makes the SDK advertise its historical
			// roots default. FlexiGPT must not advertise roots, sampling, or
			// elicitation until the product has explicit UX and safety mcpServer.
			Capabilities: buildClientCapabilities(cfg.Policy.AppsPolicy),

			KeepAlive: defaultClientKeepAlive,

			ToolListChangedHandler: func(ctx context.Context, req *mcpSDK.ToolListChangedRequest) {
				logger.Info("mcp tools list changed", "sessionID", safeClientSessionID(req))
				emit(ctx, mcpServer.ClientNotification{
					Server: cfg.Server,
					Kind:   mcpServer.ClientNotificationToolListChanged,
				})
			},
			PromptListChangedHandler: func(ctx context.Context, req *mcpSDK.PromptListChangedRequest) {
				logger.Info("mcp prompts list changed", "sessionID", safeClientSessionID(req))
				emit(ctx, mcpServer.ClientNotification{
					Server: cfg.Server,
					Kind:   mcpServer.ClientNotificationPromptListChanged,
				})
			},
			ResourceListChangedHandler: func(ctx context.Context, req *mcpSDK.ResourceListChangedRequest) {
				logger.Info("mcp resources list changed", "sessionID", safeClientSessionID(req))
				emit(ctx, mcpServer.ClientNotification{
					Server: cfg.Server,
					Kind:   mcpServer.ClientNotificationResourceListChanged,
				})
			},
			ResourceUpdatedHandler: func(ctx context.Context, req *mcpSDK.ResourceUpdatedNotificationRequest) {
				uri := ""
				if req != nil && req.Params != nil {
					uri = req.Params.URI
				}
				emit(ctx, mcpServer.ClientNotification{
					Server:      cfg.Server,
					Kind:        mcpServer.ClientNotificationResourceUpdated,
					ResourceURI: uri,
				})
			},
			ProgressNotificationHandler: func(ctx context.Context, req *mcpSDK.ProgressNotificationClientRequest) {
				if req == nil || req.Params == nil {
					return
				}
				emit(ctx, mcpServer.ClientNotification{
					Server:   cfg.Server,
					Kind:     mcpServer.ClientNotificationProgress,
					Progress: req.Params.Progress,
					Total:    req.Params.Total,
					Message:  req.Params.Message,
				})
			},
		},
	)

	// Compatibility with older MCP servers:
	// suppress the SDK's new sessionless server/discover probe at the client
	// middleware layer. Do not wrap the transport connection here, because the
	// SDK relies on its concrete streamable HTTP connection to receive
	// sessionUpdated callbacks after initialize.
	preferLegacyInitializeClient(client)
	var transport mcpSDK.Transport

	switch cfg.Transport {
	case mcpServer.MCPTransportStdio:
		if cfg.Stdio == nil {
			return nil, fmt.Errorf(
				"%w: missing stdio runtime config",
				mcpServer.ErrMCPInvalidRuntimeRequest,
			)
		}

		//nolint:gosec,noctx // User-configured MCP stdio command is intentional.
		cmd := exec.Command(cfg.Stdio.Command, cfg.Stdio.Args...)
		cmd.Env = envMapToList(mergeStringMaps(
			cfg.Stdio.Env,
			resolved.Env,
		))

		redactor := mcpAuth.NewSecretRedactor(mcpConnection.PreparedConnection{
			SensitiveValues: append(
				append([]string(nil), cfg.SensitiveValues...),
				resolved.SensitiveValues...,
			),
		})
		cmd.Stderr = newSlogLineWriter(
			logger,
			string(cfg.Server),
			"mcp stdio stderr",
			redactor,
		)

		transport = &mcpSDK.CommandTransport{
			Command:           cmd,
			TerminateDuration: defaultStdioTerminateDuration,
		}

	case mcpServer.MCPTransportStreamableHTTP:
		if cfg.StreamableHTTP == nil {
			return nil, fmt.Errorf(
				"%w: missing streamable HTTP runtime config",
				mcpServer.ErrMCPInvalidRuntimeRequest,
			)
		}

		oauthHandler, err := sdkOAuthHandler(resolved.OAuthHandler)
		if err != nil {
			return nil, fmt.Errorf(
				"%w: unsupported prepared OAuth handler",
				mcpServer.ErrMCPInvalidRuntimeRequest,
			)
		}

		headers := mergeStringMaps(
			cfg.StreamableHTTP.Headers,
			resolved.Headers,
		)
		httpClient := newStreamableHTTPClient(headers)

		transport = &mcpSDK.StreamableClientTransport{
			Endpoint:             cfg.StreamableHTTP.URL,
			HTTPClient:           httpClient,
			MaxRetries:           defaultHTTPMaxRetries,
			DisableStandaloneSSE: false,
			OAuthHandler:         oauthHandler,
		}

	default:
		return nil, fmt.Errorf(
			"%w: unsupported MCP runtime transport %q",
			mcpServer.ErrMCPInvalidRuntimeRequest,
			cfg.Transport,
		)
	}

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, fmt.Errorf(
			"%w: MCP SDK returned no client session",
			mcpServer.ErrMCPRuntimeNotReady,
		)
	}

	return &Session{
		server:  cfg.Server,
		session: session,
		logger:  logger,
	}, nil
}

func mergeStringMaps(
	base map[string]string,
	overlay map[string]string,
) map[string]string {
	output := maps.Clone(base)
	if output == nil {
		output = map[string]string{}
	}
	maps.Copy(output, overlay)
	return output
}

func sdkOAuthHandler(value any) (auth.OAuthHandler, error) {
	if value == nil {
		//nolint:nilnil // Explicit.
		return nil, nil
	}
	handler, ok := value.(auth.OAuthHandler)
	if !ok {
		return nil, fmt.Errorf("prepared OAuth handler has incompatible type %T", value)
	}
	return handler, nil
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
func buildClientCapabilities(
	p mcpPolicy.MCPAppsPolicy,
) *mcpSDK.ClientCapabilities {
	c := &mcpSDK.ClientCapabilities{}
	if p.Enabled {
		c.AddExtension(mcpApps.AppExtensionID, map[string]any{
			"mimeTypes": []string{mcpApps.AppMIMEType},
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
