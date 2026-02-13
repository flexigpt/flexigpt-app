package store

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/tool/goregistry"
	"github.com/flexigpt/flexigpt-app/internal/tool/spec"
	"github.com/flexigpt/flexigpt-app/internal/tool/storehelper"
	"github.com/flexigpt/llmtools-go/fstool"
	llmtoolsgoSpec "github.com/flexigpt/llmtools-go/spec"
	"github.com/ppipada/mapstore-go/jsonencdec"
)

func TestInvokeTool(t *testing.T) {
	t.Parallel()

	type (
		handlerFn func(w http.ResponseWriter, r *http.Request)
		mkToolFn  func(baseURL string) spec.HTTPToolImpl
		verifyFn  func(t *testing.T, resp *spec.InvokeToolResponse, err error)
	)

	// Common minimal tool builder: GET with query/header templating.
	defaultTool := func(baseURL, pathSuffix string) spec.HTTPToolImpl {
		return spec.HTTPToolImpl{
			Request: spec.HTTPRequest{
				Method:      "GET",
				URLTemplate: baseURL + pathSuffix,
				Query: map[string]string{
					"q":   "${msg}",
					"sec": "${SECRET}",
				},
				Headers: map[string]string{
					"X-Auth": "token ${SECRET}",
				},
			},
			Response: spec.HTTPResponse{},
		}
	}

	tests := []struct {
		name          string
		handler       handlerFn
		mkTool        mkToolFn
		args          string
		httpOptions   *spec.InvokeHTTPOptions
		disableBundle bool
		disableTool   bool
		wantErrIs     error
		verify        verifyFn
	}{
		{
			name: "http_get_success_with_templating_and_meta",
			handler: func(w http.ResponseWriter, r *http.Request) {
				// Echo query/header into JSON.
				q := r.URL.Query().Get("q")
				sec := r.URL.Query().Get("sec")
				auth := r.Header.Get("X-Auth")
				extra := r.Header.Get("X-Extra")
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{
					"q":     q,
					"sec":   sec,
					"auth":  auth,
					"extra": extra,
				})
			},
			mkTool: func(baseURL string) spec.HTTPToolImpl {
				impl := defaultTool(baseURL, "/echo")
				return impl
			},
			args: `{"msg":"hello"}`,
			httpOptions: &spec.InvokeHTTPOptions{
				Secrets:      map[string]string{"SECRET": "s3cr3t"},
				ExtraHeaders: map[string]string{"X-Extra": "1"},
			},
			verify: func(t *testing.T, resp *spec.InvokeToolResponse, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("InvokeTool error: %v", err)
				}
				if resp == nil || resp.Body == nil {
					t.Fatalf("nil response/body")
				}
				if resp.Body.IsBuiltIn {
					t.Fatalf("expected IsBuiltIn=false for user tool")
				}
				if resp.Body.IsError {
					t.Fatalf("expected IsError=false, got true (ErrorMessage=%q)", resp.Body.ErrorMessage)
				}
				if resp.Body.ErrorMessage != "" {
					t.Fatalf("expected empty ErrorMessage for success, got %q", resp.Body.ErrorMessage)
				}
				if resp.Body.Meta == nil {
					t.Fatalf("expected meta, got nil")
				}
				if typ, _ := resp.Body.Meta["type"].(string); typ != string(spec.ToolTypeHTTP) {
					t.Fatalf("meta.type = %v, want http", typ)
				}
				if _, ok := resp.Body.Meta["status"]; !ok {
					t.Fatalf("meta.status missing")
				}

				var got map[string]any
				textOut := getOneTextOutput(t, resp.Body)
				if err := json.Unmarshal([]byte(textOut), &got); err != nil {
					t.Fatalf("unmarshal output: %v; raw: %q", err, textOut)
				}
				if got["q"] != "hello" {
					t.Fatalf("q = %v, want hello", got["q"])
				}
				if got["sec"] != "s3cr3t" {
					t.Fatalf("sec = %v, want s3cr3t", got["sec"])
				}
				if got["auth"] != "token s3cr3t" {
					t.Fatalf("auth = %v, want 'token s3cr3t'", got["auth"])
				}
				if got["extra"] != "1" {
					t.Fatalf("extra = %v, want 1", got["extra"])
				}
			},
		},
		{
			name: "bundle_disabled",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
			},
			mkTool:        func(baseURL string) spec.HTTPToolImpl { return defaultTool(baseURL, "/ok") },
			args:          `{"msg":"x"}`,
			disableBundle: true,
			wantErrIs:     spec.ErrBundleDisabled,
		},
		{
			name: "tool_disabled",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
			},
			mkTool:      func(baseURL string) spec.HTTPToolImpl { return defaultTool(baseURL, "/ok") },
			args:        `{"msg":"x"}`,
			disableTool: true,
			wantErrIs:   spec.ErrToolDisabled,
		},
		{
			name: "non_json_text_response_treated_as_text",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				_, _ = w.Write([]byte("ok"))
			},
			mkTool: func(baseURL string) spec.HTTPToolImpl { return defaultTool(baseURL, "/plain") },
			args:   `{}`,
			verify: func(t *testing.T, resp *spec.InvokeToolResponse, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("InvokeTool error: %v", err)
				}
				if resp == nil || resp.Body == nil {
					t.Fatalf("nil response/body")
				}
				if resp.Body.IsError {
					t.Fatalf("expected IsError=false for text/plain response, got true: %q", resp.Body.ErrorMessage)
				}
				textOut := getOneTextOutput(t, resp.Body)
				if textOut != "ok" {
					t.Fatalf("text output = %q, want %q", textOut, "ok")
				}
			},
		},
		{
			name: "invalid_json_body_sets_is_error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte("{invalid"))
			},
			mkTool: func(baseURL string) spec.HTTPToolImpl { return defaultTool(baseURL, "/badjson") },
			args:   `{}`,
			verify: func(t *testing.T, resp *spec.InvokeToolResponse, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("InvokeTool error: %v", err)
				}
				if resp == nil || resp.Body == nil {
					t.Fatalf("nil response/body")
				}
				if !resp.Body.IsError {
					t.Fatalf("expected IsError=true for invalid JSON body")
				}
				if !strings.Contains(resp.Body.ErrorMessage, "response body is not valid JSON") {
					t.Fatalf(
						"ErrorMessage = %q, want contains %q",
						resp.Body.ErrorMessage,
						"response body is not valid JSON",
					)
				}
			},
		},
		{
			name: "no_content_204_returns_no_outputs",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			},
			mkTool: func(baseURL string) spec.HTTPToolImpl { return defaultTool(baseURL, "/nocontent") },
			args:   `{}`,
			verify: func(t *testing.T, resp *spec.InvokeToolResponse, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("InvokeTool error: %v", err)
				}
				if resp == nil || resp.Body == nil {
					t.Fatalf("nil response/body")
				}
				if resp.Body.IsError {
					t.Fatalf(
						"expected IsError=false for 204 success, got true (ErrorMessage=%q)",
						resp.Body.ErrorMessage,
					)
				}
				if len(resp.Body.Outputs) != 0 {
					t.Fatalf("expected no outputs for 204, got %#v", resp.Body.Outputs)
				}
			},
		},
		{
			name: "custom_success_codes_mismatch_sets_is_error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				// Return 200 while the tool requires 201.
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
			},
			mkTool: func(baseURL string) spec.HTTPToolImpl {
				impl := defaultTool(baseURL, "/status")
				impl.Response.SuccessCodes = []int{http.StatusCreated}
				return impl
			},
			args: `{}`,
			verify: func(t *testing.T, resp *spec.InvokeToolResponse, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("InvokeTool error: %v", err)
				}
				if resp == nil || resp.Body == nil {
					t.Fatalf("nil response/body")
				}
				if !resp.Body.IsError {
					t.Fatalf("expected IsError=true for success-code mismatch")
				}
				if !strings.Contains(resp.Body.ErrorMessage, "not in success set") {
					t.Fatalf("ErrorMessage = %q, want contains %q", resp.Body.ErrorMessage, "not in success set")
				}
			},
		},
		{
			name: "error_mode_empty_suppresses_error_and_returns_empty_outputs",
			handler: func(w http.ResponseWriter, r *http.Request) {
				// Return 404; errorMode: empty should suppress errors and yield empty outputs.
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]any{"err": "nope"})
			},
			mkTool: func(baseURL string) spec.HTTPToolImpl {
				impl := defaultTool(baseURL, "/404")
				impl.Response.ErrorMode = "empty"
				return impl
			},
			args: `{}`,
			verify: func(t *testing.T, resp *spec.InvokeToolResponse, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("InvokeTool error: %v", err)
				}
				if resp == nil || resp.Body == nil {
					t.Fatalf("nil response/body")
				}
				if resp.Body.IsError {
					t.Fatalf(
						"expected IsError=false when errorMode=empty, got true (ErrorMessage=%q)",
						resp.Body.ErrorMessage,
					)
				}
				if len(resp.Body.Outputs) != 0 {
					t.Fatalf("expected no outputs when errorMode=empty, got %#v", resp.Body.Outputs)
				}
				if resp.Body.Meta == nil {
					t.Fatalf("meta should be present")
				}
				if st, ok := resp.Body.Meta["status"].(int); !ok || st != http.StatusNotFound {
					t.Fatalf(
						"meta.status = %v, want %d",
						resp.Body.Meta["status"],
						http.StatusNotFound,
					)
				}
			},
		},
		{
			name: "timeout_override_sets_is_error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				// Sleep long enough to exceed override timeout.
				time.Sleep(150 * time.Millisecond)
				_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
			},
			mkTool: func(baseURL string) spec.HTTPToolImpl {
				return defaultTool(baseURL, "/slow")
			},
			args: `{}`,
			httpOptions: &spec.InvokeHTTPOptions{
				TimeoutMS: 20,
			},
			verify: func(t *testing.T, resp *spec.InvokeToolResponse, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("InvokeTool error: %v", err)
				}
				if resp == nil || resp.Body == nil {
					t.Fatalf("nil response/body")
				}
				if !resp.Body.IsError {
					t.Fatalf("expected IsError=true for timeout")
				}
				// Generic marker; actual error text may vary but should mention the underlying http.Do.
				if !strings.Contains(resp.Body.ErrorMessage, "http.Do") {
					t.Fatalf("ErrorMessage = %q, want contains %q", resp.Body.ErrorMessage, "http.Do")
				}
			},
		},
		{
			name: "body_output_mode_text_on_binary",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/octet-stream")
				_, _ = w.Write([]byte("BINARY"))
			},
			mkTool: func(baseURL string) spec.HTTPToolImpl {
				impl := defaultTool(baseURL, "/bintext")
				impl.Response.BodyOutputMode = spec.HTTPBodyOutputModeText
				return impl
			},
			args: `{}`,
			verify: func(t *testing.T, resp *spec.InvokeToolResponse, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("InvokeTool error: %v", err)
				}
				if resp == nil || resp.Body == nil {
					t.Fatalf("nil resp/body")
				}
				if resp.Body.IsError {
					t.Fatalf("unexpected IsError=true: %q", resp.Body.ErrorMessage)
				}
				textOut := getOneTextOutput(t, resp.Body)
				if textOut != "BINARY" {
					t.Fatalf("text output = %q, want %q", textOut, "BINARY")
				}
			},
		},
		{
			name: "body_output_mode_file_on_binary",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/octet-stream")
				_, _ = w.Write([]byte{0x01, 0x02, 0x03, 0x04})
			},
			mkTool: func(baseURL string) spec.HTTPToolImpl {
				impl := defaultTool(baseURL, "/bin.dat")
				impl.Response.BodyOutputMode = spec.HTTPBodyOutputModeFile
				return impl
			},
			args: `{}`,
			verify: func(t *testing.T, resp *spec.InvokeToolResponse, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("InvokeTool error: %v", err)
				}
				if resp == nil || resp.Body == nil {
					t.Fatalf("nil resp/body")
				}
				if resp.Body.IsError {
					t.Fatalf("unexpected IsError=true: %q", resp.Body.ErrorMessage)
				}
				f := getOneFileOutput(t, resp.Body)
				if f.FileName != "bin.dat" {
					t.Fatalf("FileName = %q, want %q", f.FileName, "bin.dat")
				}
				if f.FileMIME != "application/octet-stream" {
					t.Fatalf("FileMIME = %q, want %q", f.FileMIME, "application/octet-stream")
				}
				data, err := base64.StdEncoding.DecodeString(f.FileData)
				if err != nil {
					t.Fatalf("DecodeString: %v", err)
				}
				want := []byte{0x01, 0x02, 0x03, 0x04}
				if len(data) != len(want) {
					t.Fatalf("decoded length = %d, want %d", len(data), len(want))
				}
				for i := range want {
					if data[i] != want[i] {
						t.Fatalf("decoded[%d] = %d, want %d", i, data[i], want[i])
					}
				}
			},
		},
		{
			name: "body_output_mode_auto_image",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "image/png")
				_, _ = w.Write([]byte{0x89, 'P', 'N', 'G'})
			},
			mkTool: func(baseURL string) spec.HTTPToolImpl {
				impl := defaultTool(baseURL, "/img.png")
				// BodyOutputMode default (auto).
				return impl
			},
			args: `{}`,
			verify: func(t *testing.T, resp *spec.InvokeToolResponse, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("InvokeTool error: %v", err)
				}
				if resp == nil || resp.Body == nil {
					t.Fatalf("nil resp/body")
				}
				if resp.Body.IsError {
					t.Fatalf("unexpected IsError=true: %q", resp.Body.ErrorMessage)
				}
				img := getOneImageOutput(t, resp.Body)
				if img.ImageName != "img.png" {
					t.Fatalf("ImageName = %q, want %q", img.ImageName, "img.png")
				}
				if img.ImageMIME != "image/png" {
					t.Fatalf("ImageMIME = %q, want %q", img.ImageMIME, "image/png")
				}
				data, err := base64.StdEncoding.DecodeString(img.ImageData)
				if err != nil {
					t.Fatalf("DecodeString: %v", err)
				}
				want := []byte{0x89, 'P', 'N', 'G'}
				if len(data) != len(want) {
					t.Fatalf("decoded length = %d, want %d", len(data), len(want))
				}
				for i := range want {
					if data[i] != want[i] {
						t.Fatalf("decoded[%d] = %d, want %d", i, data[i], want[i])
					}
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Test HTTP server for this case.
			srv := httptest.NewServer(http.HandlerFunc(tc.handler))
			defer srv.Close()

			baseDir := t.TempDir()
			ts, err := NewToolStore(baseDir)
			if err != nil {
				t.Fatalf("NewToolStore: %v", err)
			}
			defer ts.Close()

			// Create bundle and tool.
			const (
				bundleID   = bundleitemutils.BundleID("bundle-1")
				bundleSlug = bundleitemutils.BundleSlug("bundle1")
				toolSlug   = bundleitemutils.ItemSlug("tool1")
				version    = bundleitemutils.ItemVersion("v1")
			)

			if _, err := ts.PutToolBundle(t.Context(), &spec.PutToolBundleRequest{
				BundleID: bundleID,
				Body: &spec.PutToolBundleRequestBody{
					Slug:        bundleSlug,
					DisplayName: "Bundle 1",
					IsEnabled:   true,
					Description: "test bundle",
				},
			}); err != nil {
				t.Fatalf("PutToolBundle: %v", err)
			}

			// Tool (HTTP-only custom tools).
			impl := tc.mkTool(srv.URL)
			if _, err := ts.PutTool(t.Context(), &spec.PutToolRequest{
				BundleID: bundleID,
				ToolSlug: toolSlug,
				Version:  version,
				Body: &spec.PutToolRequestBody{
					DisplayName:  "Tool 1",
					Description:  "test tool",
					UserCallable: true,
					LLMCallable:  true,

					Tags:      []string{"t"},
					IsEnabled: true,
					ArgSchema: "{}",
					Type:      spec.ToolTypeHTTP,
					HTTPImpl:  &impl,
				},
			}); err != nil {
				t.Fatalf("PutTool: %v", err)
			}

			// Optional: disable bundle or tool prior to invocation.
			if tc.disableBundle {
				if _, err := ts.PatchToolBundle(t.Context(), &spec.PatchToolBundleRequest{
					BundleID: bundleID,
					Body:     &spec.PatchToolBundleRequestBody{IsEnabled: false},
				}); err != nil {
					t.Fatalf("PatchToolBundle: %v", err)
				}
			}
			if tc.disableTool {
				if _, err := ts.PatchTool(t.Context(), &spec.PatchToolRequest{
					BundleID: bundleID,
					ToolSlug: toolSlug,
					Version:  version,
					Body:     &spec.PatchToolRequestBody{IsEnabled: false},
				}); err != nil {
					t.Fatalf("PatchTool: %v", err)
				}
			}

			resp, err := ts.InvokeTool(t.Context(), &spec.InvokeToolRequest{
				BundleID: bundleID,
				ToolSlug: toolSlug,
				Version:  version,
				Body: &spec.InvokeToolRequestBody{
					Args:        tc.args,
					HTTPOptions: tc.httpOptions,
				},
			})

			if tc.wantErrIs != nil {
				if !errors.Is(err, tc.wantErrIs) {
					t.Fatalf("InvokeTool error = %v, want errors.Is(..., %v)", err, tc.wantErrIs)
				}
				return
			}

			if tc.verify != nil {
				tc.verify(t, resp, err)
				return
			}

			if err != nil {
				t.Fatalf("InvokeTool error: %v", err)
			}
			if resp == nil || resp.Body == nil {
				t.Fatalf("nil response/body")
			}
			if resp.Body.IsError {
				t.Fatalf("expected IsError=false, got true (ErrorMessage=%q)", resp.Body.ErrorMessage)
			}
			if resp.Body.Meta == nil {
				t.Fatalf("meta is nil")
			}
			if typ, _ := resp.Body.Meta["type"].(string); typ != string(spec.ToolTypeHTTP) {
				t.Fatalf("meta.type = %v, want http", typ)
			}
		})
	}
}

func TestInvokeTool_InvalidRequest(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	ts, err := NewToolStore(baseDir)
	if err != nil {
		t.Fatalf("NewToolStore: %v", err)
	}
	defer ts.Close()

	// Missing required fields.
	_, err = ts.InvokeTool(t.Context(), &spec.InvokeToolRequest{
		BundleID: "",
		ToolSlug: "",
		Version:  "",
		Body:     &spec.InvokeToolRequestBody{Args: "{}"},
	})
	if !errors.Is(err, spec.ErrInvalidRequest) {
		t.Fatalf("err = %v, want ErrInvalidRequest", err)
	}
}

func TestInvokeTool_RequestBodyTemplating_PathQueryHeaderAuth(t *testing.T) {
	t.Parallel()

	// Handler asserts templated values for path, query, header, and Authorization.
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Path element is last segment.
		gotID := path.Base(r.URL.Path)
		if gotID == "" {
			t.Fatalf("empty path base")
		}
		qv := r.URL.Query().Get("id")
		if qv == "" {
			t.Fatalf("missing query 'id'")
		}
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			t.Fatalf("Authorization header unexpected: %q", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"pathID":  gotID,
			"queryID": qv,
			"auth":    auth,
		})
	}

	srv := httptest.NewServer(http.HandlerFunc(handler))
	defer srv.Close()

	baseDir := t.TempDir()
	ts, err := NewToolStore(baseDir)
	if err != nil {
		t.Fatalf("NewToolStore: %v", err)
	}
	defer ts.Close()

	const (
		bundleID   = bundleitemutils.BundleID("bundle-2")
		bundleSlug = bundleitemutils.BundleSlug("bundle2")
		toolSlug   = bundleitemutils.ItemSlug("tool2")
		version    = bundleitemutils.ItemVersion("v1")
	)

	_, err = ts.PutToolBundle(t.Context(), &spec.PutToolBundleRequest{
		BundleID: bundleID,
		Body: &spec.PutToolBundleRequestBody{
			Slug:        bundleSlug,
			DisplayName: "Bundle 2",
			IsEnabled:   true,
		},
	})
	if err != nil {
		t.Fatalf("PutToolBundle: %v", err)
	}

	impl := spec.HTTPToolImpl{
		Request: spec.HTTPRequest{
			Method:      "GET",
			URLTemplate: srv.URL + "/items/${id}",
			Query:       map[string]string{"id": "${id}"},
			Headers:     map[string]string{"X-Trace": "${id}"},
			Auth: &spec.HTTPAuth{
				Type:          "bearer",
				ValueTemplate: "${SECRET}",
			},
		},
		Response: spec.HTTPResponse{},
	}

	_, err = ts.PutTool(t.Context(), &spec.PutToolRequest{
		BundleID: bundleID,
		ToolSlug: toolSlug,
		Version:  version,
		Body: &spec.PutToolRequestBody{
			DisplayName:  "Tool 2",
			IsEnabled:    true,
			UserCallable: true,
			LLMCallable:  true,

			ArgSchema: "{}",
			Type:      spec.ToolTypeHTTP,
			HTTPImpl:  &impl,
		},
	})
	if err != nil {
		t.Fatalf("PutTool: %v", err)
	}

	resp, err := ts.InvokeTool(t.Context(), &spec.InvokeToolRequest{
		BundleID: bundleID,
		ToolSlug: toolSlug,
		Version:  version,
		Body: &spec.InvokeToolRequestBody{
			Args:        `{"id":"42"}`,
			HTTPOptions: &spec.InvokeHTTPOptions{Secrets: map[string]string{"SECRET": "abc123"}},
		},
	})
	if err != nil {
		t.Fatalf("InvokeTool: %v", err)
	}
	if resp == nil || resp.Body == nil {
		t.Fatalf("nil response/body")
	}
	if resp.Body.IsError {
		t.Fatalf("expected IsError=false, got true (ErrorMessage=%q)", resp.Body.ErrorMessage)
	}

	var got map[string]any
	textOut := getOneTextOutput(t, resp.Body)
	if err := json.Unmarshal([]byte(textOut), &got); err != nil {
		t.Fatalf("unmarshal output: %v; raw: %q", err, textOut)
	}
	if got["pathID"] != "42" || got["queryID"] != "42" {
		t.Fatalf("templating mismatch: got=%v", got)
	}
	if a, _ := got["auth"].(string); a != "Bearer abc123" {
		t.Fatalf("auth = %v, want 'Bearer abc123'", got["auth"])
	}
}

// TestInvokeTool_Go_CustomRegistered covers invoking user-created Go tools
// by directly inserting Tool records (type=go) into the directory-store.
// We bypass PutTool because it only accepts custom HTTP tools.
func TestInvokeGoCustomRegistered(t *testing.T) {
	t.Parallel()

	type verifyFn func(t *testing.T, resp *spec.InvokeToolResponse, err error)

	tests := []struct {
		name          string
		register      func(t *testing.T) string // returns funcName registered in the default registry
		funcName      string                    // if register is nil, allows unknown func-name case
		args          string
		disableBundle bool
		disableTool   bool
		wantErrIs     error
		verify        verifyFn
	}{
		{
			name: "echo_success",
			register: func(t *testing.T) string {
				t.Helper()
				type Args struct {
					Name  string `json:"name"`
					Times int    `json:"times"`
				}
				type Out struct {
					Msg string `json:"msg"`
				}
				fn := func(ctx context.Context, a Args) (Out, error) {
					if a.Times <= 0 {
						a.Times = 1
					}
					var b strings.Builder
					for i := 0; i < a.Times; i++ {
						b.WriteString(a.Name)
					}
					return Out{Msg: b.String()}, nil
				}
				argSchema := llmtoolsgoSpec.JSONSchema(`{
					"$schema": "http://json-schema.org/draft-07/schema#",
					"type": "object",
					"properties": {
						"name": {
								"type": "string",
								"description": "name"
						},
						"times": {
							"type": "integer",
							"description": "times",
							"default": 100
						}
					},
					"required": ["name"],
					"additionalProperties": false
				}`)

				return registerTypedAsTextInDefault(t, t.Name(), argSchema, fn)
			},
			args: `{"name":"ab","times":3}`,
			verify: func(t *testing.T, resp *spec.InvokeToolResponse, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("InvokeTool error: %v", err)
				}
				if resp == nil || resp.Body == nil {
					t.Fatalf("nil response/body")
				}
				if resp.Body.IsError {
					t.Fatalf("expected IsError=false, got true (ErrorMessage=%q)", resp.Body.ErrorMessage)
				}
				if resp.Body.Meta == nil || resp.Body.Meta["type"] != "go" {
					t.Fatalf("expected go meta")
				}
				type Out struct {
					Msg string `json:"msg"`
				}
				var out Out
				textOut := getOneTextOutput(t, resp.Body)
				if err := json.Unmarshal([]byte(textOut), &out); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				if out.Msg != "ababab" {
					t.Fatalf("msg = %q, want %q", out.Msg, "ababab")
				}
			},
		},
		{
			name: "multi_output_success",
			register: func(t *testing.T) string {
				t.Helper()
				type Args struct {
					N int `json:"n"`
				}
				fn := func(ctx context.Context, a Args) ([]llmtoolsgoSpec.ToolStoreOutputUnion, error) {
					if a.N <= 0 {
						a.N = 1
					}
					var outs []llmtoolsgoSpec.ToolStoreOutputUnion
					for i := 0; i < a.N; i++ {
						outs = append(outs, llmtoolsgoSpec.ToolStoreOutputUnion{
							Kind: llmtoolsgoSpec.ToolStoreOutputKindText,
							TextItem: &llmtoolsgoSpec.ToolStoreOutputText{
								Text: fmt.Sprintf("item-%d", i),
							},
						})
					}
					return outs, nil
				}
				argSchema := llmtoolsgoSpec.JSONSchema(`{
					"$schema": "http://json-schema.org/draft-07/schema#",
					"type": "object",
					"properties": {
						"n": {
								"type": "integer",
								"description": "no"
						}
					},
					"required": ["n"],
					"additionalProperties": false
				}`)
				return registerOutputsToolInDefault(t, t.Name(), argSchema, fn)
			},
			args: `{"n":2}`,
			verify: func(t *testing.T, resp *spec.InvokeToolResponse, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("InvokeTool error: %v", err)
				}
				if resp == nil || resp.Body == nil {
					t.Fatalf("nil resp/body")
				}
				if resp.Body.IsError {
					t.Fatalf("IsError=true: %q", resp.Body.ErrorMessage)
				}
				if resp.Body.Meta == nil || resp.Body.Meta["type"] != "go" {
					t.Fatalf("expected go meta")
				}
				if len(resp.Body.Outputs) != 2 {
					t.Fatalf("len(outputs)=%d, want 2", len(resp.Body.Outputs))
				}
				if resp.Body.Outputs[0].TextItem == nil || resp.Body.Outputs[0].TextItem.Text != "item-0" {
					t.Fatalf("first output = %#v, want text 'item-0'", resp.Body.Outputs[0])
				}
				if resp.Body.Outputs[1].TextItem == nil || resp.Body.Outputs[1].TextItem.Text != "item-1" {
					t.Fatalf("second output = %#v, want text 'item-1'", resp.Body.Outputs[1])
				}
			},
		},
		{
			name: "invalid_input_type_sets_is_error",
			register: func(t *testing.T) string {
				t.Helper()
				type Args struct {
					Times int `json:"times"`
				}
				type Out struct {
					Ok bool `json:"ok"`
				}
				fn := func(ctx context.Context, a Args) (Out, error) {
					return Out{Ok: a.Times > 0}, nil
				}
				argSchema := llmtoolsgoSpec.JSONSchema(`{
					"$schema": "http://json-schema.org/draft-07/schema#",
					"type": "object",
					"properties": {
						"times": {
							"type": "integer",
							"description": "times."
						}
					},
					"required": ["times"],
					"additionalProperties": false
				}`)
				return registerTypedAsTextInDefault(t, t.Name(), argSchema, fn)
			},
			args: `{"times":"oops"}`, // wrong type for int -> strict decode should fail
			verify: func(t *testing.T, resp *spec.InvokeToolResponse, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("InvokeTool error: %v", err)
				}
				if resp == nil || resp.Body == nil {
					t.Fatalf("nil response/body")
				}
				if !resp.Body.IsError {
					t.Fatalf("expected IsError=true for invalid input")
				}
				if !strings.Contains(resp.Body.ErrorMessage, "invalid input") {
					t.Fatalf("ErrorMessage = %q, want contains %q", resp.Body.ErrorMessage, "invalid input")
				}
				if resp.Body.Meta == nil || resp.Body.Meta["type"] != "go" {
					t.Fatalf("expected go meta")
				}
			},
		},
		{
			name: "function_returns_error_sets_is_error",
			register: func(t *testing.T) string {
				t.Helper()

				type Args struct {
					Fail bool `json:"fail"`
				}
				type Out struct{}
				fn := func(ctx context.Context, a Args) (Out, error) {
					if a.Fail {
						return Out{}, errors.New("boom")
					}
					return Out{}, nil
				}
				argSchema := llmtoolsgoSpec.JSONSchema(`{
					"$schema": "http://json-schema.org/draft-07/schema#",
					"type": "object",
					"properties": {
						"fail": {
							"type": "boolean",
							"description": "fail."
						}
					},
					"required": ["fail"],
					"additionalProperties": false
				}`)

				return registerTypedAsTextInDefault(t, t.Name(), argSchema, fn)
			},
			args: `{"fail":true}`,
			verify: func(t *testing.T, resp *spec.InvokeToolResponse, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("InvokeTool error: %v", err)
				}
				if resp == nil || resp.Body == nil {
					t.Fatalf("nil response/body")
				}
				if !resp.Body.IsError {
					t.Fatalf("expected IsError=true when function returns error")
				}
				if !strings.Contains(resp.Body.ErrorMessage, "boom") {
					t.Fatalf("ErrorMessage = %q, want contains %q", resp.Body.ErrorMessage, "boom")
				}
				if resp.Body.Meta == nil || resp.Body.Meta["type"] != "go" {
					t.Fatalf("expected go meta")
				}
			},
		},
		{
			name:     "sets_is_error",
			register: nil,
			funcName: "github.com/flexigpt/flexigpt-app/tests/unknownFunc",
			args:     `{}`,
			verify: func(t *testing.T, resp *spec.InvokeToolResponse, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("InvokeTool error: %v", err)
				}
				if resp == nil || resp.Body == nil {
					t.Fatalf("nil response/body")
				}
				if !resp.Body.IsError {
					t.Fatalf("expected IsError=true for unknown function")
				}
				if !strings.Contains(resp.Body.ErrorMessage, "unknown tool") {
					t.Fatalf("ErrorMessage = %q, want contains %q", resp.Body.ErrorMessage, "unknown tool")
				}
				if resp.Body.Meta == nil || resp.Body.Meta["type"] != "go" {
					t.Fatalf("expected go meta")
				}
			},
		},
		{
			name: "respects_context_timeout",
			register: func(t *testing.T) string {
				t.Helper()
				type Args struct {
					SleepMS int `json:"sleepMS"`
				}
				type Out struct {
					Ok bool `json:"ok"`
				}
				fn := func(ctx context.Context, a Args) (Out, error) {
					select {
					case <-ctx.Done():
						return Out{}, ctx.Err()
					case <-time.After(time.Duration(a.SleepMS) * time.Millisecond):
						return Out{Ok: true}, nil
					}
				}
				argSchema := llmtoolsgoSpec.JSONSchema(`{
					"$schema": "http://json-schema.org/draft-07/schema#",
					"type": "object",
					"properties": {
						"sleepMs": {
							"type": "integer",
							"description": "Sleep."
						}
					},
					"required": ["sleepMs"],
					"additionalProperties": false
				}`)

				return registerTypedAsTextInDefault(t, t.Name(), argSchema, fn)
			},
			args: `{"sleepMs":200}`,
			verify: func(t *testing.T, resp *spec.InvokeToolResponse, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("InvokeTool error: %v", err)
				}
				if resp == nil || resp.Body == nil {
					t.Fatalf("nil response/body")
				}
				if !resp.Body.IsError {
					t.Fatalf("expected IsError=true for context timeout")
				}
				if !strings.Contains(resp.Body.ErrorMessage, "context deadline exceeded") {
					t.Fatalf("ErrorMessage = %q, want contains %q", resp.Body.ErrorMessage, "context deadline exceeded")
				}
				if resp.Body.Meta == nil || resp.Body.Meta["type"] != "go" {
					t.Fatalf("expected go meta")
				}
			},
		},
		{
			name: "bundle_disabled",
			register: func(t *testing.T) string {
				t.Helper()
				type Args struct{}
				type Out struct{}
				fn := func(ctx context.Context, a Args) (Out, error) { return Out{}, nil }
				argSchema := llmtoolsgoSpec.JSONSchema(`{
						"$schema": "http://json-schema.org/draft-07/schema#",
						"type": "object",
						"properties": {},
						"required": [],
						"additionalProperties": false
				}`)
				return registerTypedAsTextInDefault(t, t.Name(), argSchema, fn)
			},
			args:          `{}`,
			disableBundle: true,
			wantErrIs:     spec.ErrBundleDisabled,
		},
		{
			name: "tool_disabled",
			register: func(t *testing.T) string {
				t.Helper()
				type Args struct{}
				type Out struct{}
				fn := func(ctx context.Context, a Args) (Out, error) { return Out{}, nil }
				argSchema := llmtoolsgoSpec.JSONSchema(`{
						"$schema": "http://json-schema.org/draft-07/schema#",
						"type": "object",
						"properties": {},
						"required": [],
						"additionalProperties": false
				}`)
				return registerTypedAsTextInDefault(t, t.Name(), argSchema, fn)
			},
			args:        `{}`,
			disableTool: true,
			wantErrIs:   spec.ErrToolDisabled,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			baseDir := t.TempDir()
			ts, err := NewToolStore(baseDir)
			if err != nil {
				t.Fatalf("NewToolStore: %v", err)
			}
			defer ts.Close()

			// Create a dedicated bundle per sub-test.
			bundleID := bundleitemutils.BundleID("b-" + sanitizeID(t.Name()))
			bundleSlug := bundleitemutils.BundleSlug("bundle-" + sanitizeID(t.Name()))

			putBundle(t, ts, bundleID, bundleSlug, true)

			// Register or use provided function name.
			funcName := tc.funcName
			if tc.register != nil {
				funcName = tc.register(t)
			}

			// Insert a Go tool (bypass PutTool).
			toolSlug := bundleitemutils.ItemSlug("go-" + sanitizeID(t.Name()))
			version := bundleitemutils.ItemVersion("v1")
			if err := addGoToolFile(t, ts, bundleID, bundleSlug, toolSlug, version, funcName, true); err != nil {
				t.Fatalf("addGoToolFile: %v", err)
			}

			// Optional disable bundle/tool.
			if tc.disableBundle {
				if _, err := ts.PatchToolBundle(t.Context(), &spec.PatchToolBundleRequest{
					BundleID: bundleID,
					Body:     &spec.PatchToolBundleRequestBody{IsEnabled: false},
				}); err != nil {
					t.Fatalf("PatchToolBundle: %v", err)
				}
			}
			if tc.disableTool {
				if _, err := ts.PatchTool(t.Context(), &spec.PatchToolRequest{
					BundleID: bundleID,
					ToolSlug: toolSlug,
					Version:  version,
					Body:     &spec.PatchToolRequestBody{IsEnabled: false},
				}); err != nil {
					t.Fatalf("PatchTool: %v", err)
				}
			}

			// Build invocation context (custom timeout for the timeout scenario).
			ctx := t.Context()
			if strings.Contains(tc.name, "timeout") {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 50*time.Millisecond)
				defer cancel()
			}

			resp, err := ts.InvokeTool(ctx, &spec.InvokeToolRequest{
				BundleID: bundleID,
				ToolSlug: toolSlug,
				Version:  version,
				Body: &spec.InvokeToolRequestBody{
					Args: tc.args,
					// Note: GoOptions currently not used by InvokeTool; context controls timeout.
				},
			})

			if tc.wantErrIs != nil {
				if !errors.Is(err, tc.wantErrIs) {
					t.Fatalf("err = %v, want errors.Is(..., %v)", err, tc.wantErrIs)
				}
				return
			}
			if tc.verify != nil {
				tc.verify(t, resp, err)
				return
			}
			if err != nil {
				t.Fatalf("InvokeTool error: %v", err)
			}
			if resp == nil || resp.Body == nil {
				t.Fatalf("nil response/body")
			}
			if resp.Body.IsError {
				t.Fatalf("expected IsError=false, got true (ErrorMessage=%q)", resp.Body.ErrorMessage)
			}
			if resp.Body.Meta == nil || resp.Body.Meta["type"] != "go" {
				t.Fatalf("unexpected response/meta: %+v", resp)
			}
		})
	}
}

// TestInvokeTool_Go_BuiltIns exercises the inbuilt Go tools you provided:
// - ReadFile
// - ListDirectory
// - SearchFiles
// We synthesize user Tool records that reference those registered function IDs.
func TestInvokeTool_Go_BuiltIns(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()

	// Create some files for read/list/search.
	fileA := filepath.Join(tmp, "a.txt")
	fileB := filepath.Join(tmp, "b.md")
	fileImg := filepath.Join(tmp, "img.png")
	if err := os.WriteFile(fileA, []byte("hello world"), 0o600); err != nil {
		t.Fatalf("write a.txt: %v", err)
	}
	if err := os.WriteFile(fileB, []byte("lorem ipsum"), 0o600); err != nil {
		t.Fatalf("write b.md: %v", err)
	}
	if err := os.WriteFile(fileImg, []byte{0x01, 0x02, 0x03}, 0o600); err != nil {
		t.Fatalf("write img.png: %v", err)
	}

	ft, err := fstool.NewFSTool()
	if err != nil {
		t.Fatalf("new file tool error")
	}

	tests := []struct {
		name     string
		funcName string
		args     string
		verify   func(t *testing.T, body *spec.InvokeToolResponseBody)
	}{
		{
			name:     "ReadFile_text_success",
			funcName: string(ft.ReadFileTool().GoImpl.FuncID),
			args:     fmt.Sprintf(`{"path":%q,"encoding":"text"}`, fileA),
			verify: func(t *testing.T, body *spec.InvokeToolResponseBody) {
				t.Helper()
				text := getOneTextOutput(t, body)
				if text != "hello world" {
					t.Fatalf("content=%q, want %q", text, "hello world")
				}
			},
		},
		{
			name:     "ReadFile_binary_file_output",
			funcName: string(ft.ReadFileTool().GoImpl.FuncID),
			args:     fmt.Sprintf(`{"path":%q,"encoding":"binary"}`, fileB),
			verify: func(t *testing.T, body *spec.InvokeToolResponseBody) {
				t.Helper()
				f := getOneFileOutput(t, body)
				if f.FileName != "b.md" {
					t.Fatalf("FileName=%q, want %q", f.FileName, "b.md")
				}
				if f.FileMIME == "" {
					t.Fatalf("FileMIME should not be empty")
				}
				data, err := base64.StdEncoding.DecodeString(f.FileData)
				if err != nil {
					t.Fatalf("DecodeString: %v", err)
				}
				if string(data) != "lorem ipsum" {
					t.Fatalf("decoded content=%q, want %q", string(data), "lorem ipsum")
				}
			},
		},
		{
			name:     "ReadFile_binary_image_output",
			funcName: string(ft.ReadFileTool().GoImpl.FuncID),
			args:     fmt.Sprintf(`{"path":%q,"encoding":"binary"}`, fileImg),
			verify: func(t *testing.T, body *spec.InvokeToolResponseBody) {
				t.Helper()
				img := getOneImageOutput(t, body)
				if img.ImageName != "img.png" {
					t.Fatalf("ImageName=%q, want %q", img.ImageName, "img.png")
				}
				if !strings.HasPrefix(img.ImageMIME, "image/") {
					t.Fatalf("ImageMIME=%q, want prefix %q", img.ImageMIME, "image/")
				}
				data, err := base64.StdEncoding.DecodeString(img.ImageData)
				if err != nil {
					t.Fatalf("DecodeString: %v", err)
				}
				want := []byte{0x01, 0x02, 0x03}
				if len(data) != len(want) {
					t.Fatalf("decoded length = %d, want %d", len(data), len(want))
				}
				for i := range want {
					if data[i] != want[i] {
						t.Fatalf("decoded[%d]=%d, want %d", i, data[i], want[i])
					}
				}
			},
		},
		{
			name:     "ListDirectory_glob",
			funcName: string(ft.ListDirectoryTool().GoImpl.FuncID),
			args:     fmt.Sprintf(`{"path":%q,"pattern":"*.txt"}`, tmp),
			verify: func(t *testing.T, body *spec.InvokeToolResponseBody) {
				t.Helper()
				text := getOneTextOutput(t, body)
				var o fstool.ListDirectoryOut
				if err := json.Unmarshal([]byte(text), &o); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				has := func(name string) bool {
					return slices.Contains(o.Entries, name)
				}
				if !has("a.txt") || has("b.md") {
					t.Fatalf("entries=%v, expected only a.txt with pattern", o.Entries)
				}
			},
		},
		{
			name:     "SearchFiles_regex",
			funcName: string(ft.SearchFilesTool().GoImpl.FuncID),
			args:     fmt.Sprintf(`{"root":%q,"pattern":"hello"}`, tmp),
			verify: func(t *testing.T, body *spec.InvokeToolResponseBody) {
				t.Helper()
				text := getOneTextOutput(t, body)
				var o fstool.SearchFilesOut
				if err := json.Unmarshal([]byte(text), &o); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				found := slices.Contains(o.Matches, fileA)
				if !found {
					t.Fatalf("expected to find %q in matches; got %v", fileA, o.Matches)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			baseDir := t.TempDir()
			ts, err := NewToolStore(baseDir)
			if err != nil {
				t.Fatalf("NewToolStore: %v", err)
			}
			defer ts.Close()

			// Bundle per case.
			bundleID := bundleitemutils.BundleID("b-" + sanitizeID(t.Name()))
			bundleSlug := bundleitemutils.BundleSlug("bundle-" + sanitizeID(t.Name()))
			putBundle(t, ts, bundleID, bundleSlug, true)

			// Write the Go tool referencing the built-in function name.
			toolSlug := bundleitemutils.ItemSlug("go-" + sanitizeID(t.Name()))
			version := bundleitemutils.ItemVersion("v1")
			if err := addGoToolFile(t, ts, bundleID, bundleSlug, toolSlug, version, tc.funcName, true); err != nil {
				t.Fatalf("addGoToolFile: %v", err)
			}

			resp, err := ts.InvokeTool(t.Context(), &spec.InvokeToolRequest{
				BundleID: bundleID,
				ToolSlug: toolSlug,
				Version:  version,
				Body: &spec.InvokeToolRequestBody{
					Args: tc.args,
				},
			})
			if err != nil {
				t.Fatalf("InvokeTool: %v", err)
			}
			if resp == nil || resp.Body == nil {
				t.Fatalf("nil response/body")
			}
			if resp.Body.IsError {
				t.Fatalf("unexpected IsError=true for built-in tool: %q", resp.Body.ErrorMessage)
			}
			tc.verify(t, resp.Body)
		})
	}
}

// addGoToolFile inserts a Go-type Tool record directly into the directory store.
// This bypasses PutTool (which only accepts HTTP tools) so we can exercise the Go path.
func addGoToolFile(
	t *testing.T,
	ts *ToolStore,
	bundleID bundleitemutils.BundleID,
	bundleSlug bundleitemutils.BundleSlug,
	toolSlug bundleitemutils.ItemSlug,
	version bundleitemutils.ItemVersion,
	funcName string,
	enabled bool,
) error {
	t.Helper()

	dirInfo, err := bundleitemutils.BuildBundleDir(bundleID, bundleSlug)
	if err != nil {
		return err
	}
	finf, err := bundleitemutils.BuildItemFileInfo(toolSlug, version)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	tool := spec.Tool{
		SchemaVersion: spec.SchemaVersion,
		ID:            bundleitemutils.ItemID("go-" + string(toolSlug) + "-" + string(version)),
		Slug:          toolSlug,
		Version:       version,
		DisplayName:   "Go Tool " + string(toolSlug),
		Description:   "test go tool",
		Tags:          []string{"go"},
		UserCallable:  true,
		LLMCallable:   true,
		ArgSchema:     json.RawMessage(`{}`),
		LLMToolType:   spec.ToolStoreChoiceTypeFunction,
		Type:          spec.ToolTypeGo,
		GoImpl:        &spec.GoToolImpl{Func: funcName},
		HTTPImpl:      nil,
		IsEnabled:     enabled,
		IsBuiltIn:     false,
		CreatedAt:     now,
		ModifiedAt:    now,
	}
	if err := storehelper.ValidateTool(&tool); err != nil {
		return fmt.Errorf("ValidateTool: %w", err)
	}

	mp, _ := jsonencdec.StructWithJSONTagsToMap(tool)
	return ts.toolStore.SetFileData(
		bundleitemutils.GetBundlePartitionFileKey(finf.FileName, dirInfo.DirName),
		mp,
	)
}

func putBundle(
	t *testing.T,
	ts *ToolStore,
	id bundleitemutils.BundleID,
	slug bundleitemutils.BundleSlug,
	enabled bool,
) {
	t.Helper()
	if _, err := ts.PutToolBundle(t.Context(), &spec.PutToolBundleRequest{
		BundleID: id,
		Body: &spec.PutToolBundleRequestBody{
			Slug:        slug,
			DisplayName: "bundle " + string(slug),
			IsEnabled:   enabled,
		},
	}); err != nil {
		t.Fatalf("PutToolBundle: %v", err)
	}
}

func getOneTextOutput(t *testing.T, respBody *spec.InvokeToolResponseBody) string {
	t.Helper()
	outputs := respBody.Outputs
	if len(outputs) != 1 || outputs[0].TextItem == nil {
		t.Fatalf("expected exactly one text output, got %#v", outputs)
	}
	return outputs[0].TextItem.Text
}

func getOneFileOutput(t *testing.T, respBody *spec.InvokeToolResponseBody) *spec.ToolStoreOutputFile {
	t.Helper()
	outputs := respBody.Outputs
	if len(outputs) != 1 || outputs[0].FileItem == nil {
		t.Fatalf("expected exactly one file output, got %#v", outputs)
	}
	return outputs[0].FileItem
}

func getOneImageOutput(t *testing.T, respBody *spec.InvokeToolResponseBody) *spec.ToolStoreOutputImage {
	t.Helper()
	outputs := respBody.Outputs
	if len(outputs) != 1 || outputs[0].ImageItem == nil {
		t.Fatalf("expected exactly one image output, got %#v", outputs)
	}
	return outputs[0].ImageItem
}

func registerTypedAsTextInDefault[T, R any](
	t *testing.T, nameSuffix string, argSchema llmtoolsgoSpec.JSONSchema,
	fn func(context.Context, T) (R, error),
) string {
	t.Helper()
	slug := sanitizeID(nameSuffix)
	funcName := "github.com/flexigpt/flexigpt-app/tests/" + slug

	llmTool := llmtoolsgoSpec.Tool{
		SchemaVersion: llmtoolsgoSpec.SchemaVersion,
		ID:            "018fe0f4-b8cd-7e55-82d5-9df0bd70e4ba",
		Slug:          nameSuffix,
		Version:       "v1.0.0",
		DisplayName:   "Read file",
		Description:   "Read a local file from disk and return its contents (text or base64).",
		Tags:          []string{"fs", "read"},

		ArgSchema: argSchema,
		GoImpl:    llmtoolsgoSpec.GoToolImpl{FuncID: llmtoolsgoSpec.FuncID(funcName)},

		CreatedAt:  llmtoolsgoSpec.SchemaStartTime,
		ModifiedAt: llmtoolsgoSpec.SchemaStartTime,
	}
	if err := goregistry.RegisterTypedAsTextToolUsingDefaultGoRegistry(llmTool, fn); err != nil {
		t.Fatalf("RegisterTypedAsText: %v", err)
	}
	return funcName
}

func registerOutputsToolInDefault[T any](
	t *testing.T, nameSuffix string, argSchema llmtoolsgoSpec.JSONSchema,
	fn func(context.Context, T) ([]llmtoolsgoSpec.ToolStoreOutputUnion, error),
) string {
	t.Helper()
	slug := sanitizeID(nameSuffix)
	funcName := "github.com/flexigpt/flexigpt-app/tests/" + slug

	llmTool := llmtoolsgoSpec.Tool{
		SchemaVersion: llmtoolsgoSpec.SchemaVersion,
		ID:            "018fe0f4-b8cd-7e55-82d5-9df0bd70e4ba",
		Slug:          nameSuffix,
		Version:       "v1.0.0",
		DisplayName:   "Read file",
		Description:   "Read a local file from disk and return its contents (text or base64).",
		Tags:          []string{"fs", "read"},

		ArgSchema: argSchema,
		GoImpl:    llmtoolsgoSpec.GoToolImpl{FuncID: llmtoolsgoSpec.FuncID(funcName)},

		CreatedAt:  llmtoolsgoSpec.SchemaStartTime,
		ModifiedAt: llmtoolsgoSpec.SchemaStartTime,
	}
	if err := goregistry.RegisterOutputsToolUsingDefaultGoRegistry(llmTool, fn); err != nil {
		t.Fatalf("RegisterTypedAsText: %v", err)
	}
	return funcName
}

// sanitizeID produces a slug-ish identifier from a test name for use in IDs, slugs and func names.
func sanitizeID(s string) string {
	s = strings.ToLower(s)
	if len(s) > 48 {
		s = s[:48]
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}
