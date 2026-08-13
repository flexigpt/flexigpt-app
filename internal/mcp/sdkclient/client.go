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

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/mcp/auth"
	"github.com/flexigpt/flexigpt-app/internal/mcp/policy"
	"github.com/flexigpt/flexigpt-app/internal/mcp/runtime"
	"github.com/flexigpt/flexigpt-app/internal/mcp/server"
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
	cfg server.RuntimeConfig,
	resolved auth.ResolvedTransportAuth,
	events runtime.ClientNotificationSink,
) (runtime.ClientSession, error) {
	if err := cfg.Server.Validate(); err != nil {
		return nil, err
	}
	if cfg.Collection.RootID != cfg.Server.RootID {
		return nil, fmt.Errorf(
			"%w: MCP runtime config has mismatched server and Collection roots",
			runtime.ErrMCPInvalidRuntimeRequest,
		)
	}

	logger := f.log()
	emit := func(ctx context.Context, event runtime.ClientNotification) {
		if events != nil {
			events.OnClientNotification(ctx, event)
		}
	}
	client := mcpSDK.NewClient(
		&mcpSDK.Implementation{
			Name:    artifactbuiltin.MCPHostName,
			Version: artifactbuiltin.MCPHostVersion,
		},
		&mcpSDK.ClientOptions{
			Logger: logger,

			// Important:
			// A nil Capabilities value makes the SDK advertise its historical
			// roots default. FlexiGPT must not advertise roots, sampling, or
			// elicitation until the product has explicit UX and safety policy.
			Capabilities: buildClientCapabilities(cfg.AppsPolicy),

			KeepAlive: defaultClientKeepAlive,

			ToolListChangedHandler: func(ctx context.Context, req *mcpSDK.ToolListChangedRequest) {
				logger.Info("mcp tools list changed", "sessionID", safeClientSessionID(req))
				emit(ctx, runtime.ClientNotification{
					Server: cfg.Server,
					Kind:   runtime.ClientNotificationToolListChanged,
				})
			},
			PromptListChangedHandler: func(ctx context.Context, req *mcpSDK.PromptListChangedRequest) {
				logger.Info("mcp prompts list changed", "sessionID", safeClientSessionID(req))
				emit(ctx, runtime.ClientNotification{
					Server: cfg.Server,
					Kind:   runtime.ClientNotificationPromptListChanged,
				})
			},
			ResourceListChangedHandler: func(ctx context.Context, req *mcpSDK.ResourceListChangedRequest) {
				logger.Info("mcp resources list changed", "sessionID", safeClientSessionID(req))
				emit(ctx, runtime.ClientNotification{
					Server: cfg.Server,
					Kind:   runtime.ClientNotificationResourceListChanged,
				})
			},
			ResourceUpdatedHandler: func(ctx context.Context, req *mcpSDK.ResourceUpdatedNotificationRequest) {
				uri := ""
				if req != nil && req.Params != nil {
					uri = req.Params.URI
				}
				emit(ctx, runtime.ClientNotification{
					Server:      cfg.Server,
					Kind:        runtime.ClientNotificationResourceUpdated,
					ResourceURI: uri,
				})
			},
			ProgressNotificationHandler: func(ctx context.Context, req *mcpSDK.ProgressNotificationClientRequest) {
				if req == nil || req.Params == nil {
					return
				}
				emit(ctx, runtime.ClientNotification{
					Server:   cfg.Server,
					Kind:     runtime.ClientNotificationProgress,
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
	case server.MCPTransportStdio:
		if cfg.Stdio == nil {
			return nil, fmt.Errorf(
				"%w: missing stdio runtime config",
				runtime.ErrMCPInvalidRuntimeRequest,
			)
		}

		//nolint:gosec,noctx // User-configured MCP stdio command is intentional.
		cmd := exec.Command(cfg.Stdio.Command, cfg.Stdio.Args...)
		cmd.Env = envMapToList(mergeStringMaps(
			cfg.Stdio.Env,
			resolved.Env,
		))

		redactor := auth.NewSecretRedactor(auth.ResolvedTransportAuth{
			SensitiveValues: append(
				append([]string(nil), cfg.SensitiveValues...),
				resolved.SensitiveValues...,
			),
		})
		cmd.Stderr = newSlogLineWriter(
			logger,
			string(cfg.Server.ArtifactID),
			"mcp stdio stderr",
			redactor,
		)

		transport = &mcpSDK.CommandTransport{
			Command:           cmd,
			TerminateDuration: defaultStdioTerminateDuration,
		}

	case server.MCPTransportStreamableHTTP:
		if cfg.StreamableHTTP == nil {
			return nil, fmt.Errorf(
				"%w: missing streamable HTTP runtime config",
				runtime.ErrMCPInvalidRuntimeRequest,
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
			OAuthHandler:         resolved.OAuthHandler,
		}

	default:
		return nil, fmt.Errorf(
			"%w: unsupported MCP runtime transport %q",
			runtime.ErrMCPInvalidRuntimeRequest,
			cfg.Transport,
		)
	}

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, err
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
	p policy.MCPAppsPolicy,
) *mcpSDK.ClientCapabilities {
	c := &mcpSDK.ClientCapabilities{}
	if p.Enabled {
		c.AddExtension(runtime.AppExtensionID, map[string]any{
			"mimeTypes": []string{runtime.AppMIMEType},
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
