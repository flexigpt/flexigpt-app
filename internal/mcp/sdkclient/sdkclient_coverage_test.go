package sdkclient

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	mcpSDK "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/flexigpt/flexigpt-app/internal/mcp/apps"
	"github.com/flexigpt/flexigpt-app/internal/mcp/auth"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

type sessionCarrier struct{ session mcpSDK.Session }

func (r sessionCarrier) GetSession() mcpSDK.Session { return r.session }

func TestFactoryConnectValidationBranches(t *testing.T) {
	f := NewFactory()

	tests := []struct {
		name string
		cfg  spec.MCPServerConfig
		want string
	}{
		{
			name: "missing bundleID",
			cfg: spec.MCPServerConfig{
				ID:        "server",
				Transport: spec.MCPTransportStreamableHTTP,
				StreamableHTTP: &spec.MCPStreamableHTTPConfig{
					URL:      "http://127.0.0.1:1234/mcp",
					AuthMode: spec.MCPHTTPAuthNone,
				},
			},
			want: "bundleID required",
		},
		{
			name: "missing serverID",
			cfg: spec.MCPServerConfig{
				BundleID:  "bundle",
				Transport: spec.MCPTransportStreamableHTTP,
				StreamableHTTP: &spec.MCPStreamableHTTPConfig{
					URL:      "http://127.0.0.1:1234/mcp",
					AuthMode: spec.MCPHTTPAuthNone,
				},
			},
			want: "serverID required",
		},
		{
			name: "missing stdio config",
			cfg: spec.MCPServerConfig{
				BundleID:  "bundle",
				ID:        "server",
				Transport: spec.MCPTransportStdio,
			},
			want: "missing stdio config",
		},
		{
			name: "missing streamableHttp config",
			cfg: spec.MCPServerConfig{
				BundleID:  "bundle",
				ID:        "server",
				Transport: spec.MCPTransportStreamableHTTP,
			},
			want: "missing streamableHttp config",
		},
		{
			name: "unsupported transport",
			cfg: spec.MCPServerConfig{
				BundleID:  "bundle",
				ID:        "server",
				Transport: "bogus",
			},
			want: "unsupported transport",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := f.Connect(t.Context(), tt.cfg, auth.ResolvedTransportAuth{}, nil)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestSessionNilBranches(t *testing.T) {
	s := &Session{}
	if err := s.Close(t.Context()); err != nil {
		t.Fatalf("Close(nil session): %v", err)
	}

	tests := []struct {
		name string
		err  func() error
	}{
		{
			name: "Ping",
			err: func() error {
				return s.Ping(t.Context())
			},
		},
		{
			name: "Discover",
			err: func() error {
				_, err := s.Discover(t.Context(), "server", spec.MCPServerPolicy{}, spec.MCPTrustLevelTrusted)
				return err
			},
		},
		{
			name: "CallTool",
			err: func() error {
				_, err := s.CallTool(t.Context(), "tool", nil)
				return err
			},
		},
		{
			name: "ReadResource",
			err: func() error {
				_, err := s.ReadResource(t.Context(), "file:///demo")
				return err
			},
		},
		{
			name: "GetPrompt",
			err: func() error {
				_, err := s.GetPrompt(t.Context(), "greet", nil)
				return err
			},
		},
		{
			name: "Complete",
			err: func() error {
				_, err := s.Complete(t.Context(), spec.MCPCompleteArgumentRequestBody{RefType: "prompt"})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.err()
			if err == nil || !errors.Is(err, spec.ErrMCPRuntimeNotReady) ||
				!strings.Contains(err.Error(), "nil session") {
				t.Fatalf("err = %v, want nil session", err)
			}
		})
	}
}

func TestSDKClientHelperBranches(t *testing.T) {
	t.Run("streamableHTTP client and header round tripper", func(t *testing.T) {
		if client := newStreamableHTTPClient(nil); client.Transport != nil {
			t.Fatalf("newStreamableHTTPClient(nil) transport = %#v, want nil", client.Transport)
		}

		seen := make(chan http.Header, 2)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			seen <- r.Header.Clone()
			w.WriteHeader(http.StatusNoContent)
		}))
		defer srv.Close()

		rt := &headerRoundTripper{headers: map[string]string{"X-Test": "value"}}
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL, http.NoBody)
		if err != nil {
			t.Fatalf("http.NewRequest: %v", err)
		}
		resp, err := rt.RoundTrip(req)
		if err != nil {
			t.Fatalf("RoundTrip: %v", err)
		}
		_ = resp.Body.Close()

		got := <-seen
		if got.Get("X-Test") != "value" {
			t.Fatalf("RoundTrip header = %q, want %q", got.Get("X-Test"), "value")
		}
		if req.Header.Get("X-Test") != "" {
			t.Fatalf("original request header was mutated: %#v", req.Header)
		}

		client := newStreamableHTTPClient(map[string]string{"X-Client": "value"})
		req2, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL, http.NoBody)
		if err != nil {
			t.Fatalf("http.NewRequest: %v", err)
		}
		resp2, err := client.Do(req2)
		if err != nil {
			t.Fatalf("client.Do: %v", err)
		}
		_ = resp2.Body.Close()

		got2 := <-seen
		if got2.Get("X-Client") != "value" {
			t.Fatalf("client.Do header = %q, want %q", got2.Get("X-Client"), "value")
		}
	})

	t.Run("safeClientSessionID and deprecated app metadata", func(t *testing.T) {
		if got := safeClientSessionID(nil); got != "" {
			t.Fatalf("safeClientSessionID(nil) = %q, want empty", got)
		}
		if got := safeClientSessionID(sessionCarrier{session: nil}); got != "" {
			t.Fatalf("safeClientSessionID(nil session) = %q, want empty", got)
		}

		info := appInfoFromMeta(mcpSDK.Meta{
			"ui/resourceUri": "ui://legacy",
		})
		if info == nil {
			t.Fatalf("appInfoFromMeta returned nil")
		}
		if info.ResourceURI != "ui://legacy" {
			t.Fatalf("ResourceURI = %q, want %q", info.ResourceURI, "ui://legacy")
		}
		if !slices.Equal(info.Visibility, []string{apps.VisibilityModel, apps.VisibilityApp}) {
			t.Fatalf("Visibility = %#v, want [model app]", info.Visibility)
		}
	})

	t.Run("completion and argument helpers", func(t *testing.T) {
		ref, err := completionReference(spec.MCPCompleteArgumentRequestBody{
			RefType: "resource",
			Name:    "file:///demo",
		})
		if err != nil {
			t.Fatalf("completionReference(resource): %v", err)
		}
		if ref.Type != refTypeRefResource || ref.URI != "file:///demo" {
			t.Fatalf("completionReference(resource) = %#v", ref)
		}

		promptArgs := promptArgumentsToSpec([]*mcpSDK.PromptArgument{
			{Name: "name", Description: "the name", Required: true},
			nil,
			{Name: "   ", Description: "skip"},
		})
		if len(promptArgs) != 1 {
			t.Fatalf("promptArgumentsToSpec len = %d, want 1", len(promptArgs))
		}
		if got := promptArgs["name"]; got.Name != "name" || got.Description != "the name" || !got.Required {
			t.Fatalf("promptArgumentsToSpec[name] = %#v", got)
		}

		tmplArgs := resourceTemplateArgumentsToSpec("file:///items/{id}/{slug}")
		if len(tmplArgs) != 2 || !tmplArgs["id"].Required || !tmplArgs["slug"].Required {
			t.Fatalf("resourceTemplateArgumentsToSpec = %#v", tmplArgs)
		}
	})
}
