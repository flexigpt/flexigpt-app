package runtime

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/flexigpt/llmtools-go/fstool"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/tool/runtime/spec"
	toolSpec "github.com/flexigpt/flexigpt-app/internal/tool/spec"
	toolStore "github.com/flexigpt/flexigpt-app/internal/tool/store"
)

func TestInvokeToolUnwrapsWailsJSONArguments(t *testing.T) {
	store := newRuntimeToolStore(t)
	target := findBuiltInReadFileTool(t, store)
	enableBuiltInToolForTest(t, store, target)

	filePath := t.TempDir() + "/quoted-args.txt"
	const content = "tool runtime quoted Wails JSON test"
	if err := os.WriteFile(filePath, []byte(content), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	runtime := NewToolRuntime(store)
	response, err := runtime.InvokeTool(t.Context(), &spec.InvokeToolRequest{
		BundleID: target.BundleID,
		ToolSlug: target.ToolSlug,
		Version:  target.ToolVersion,
		Body: &spec.InvokeToolRequestBody{
			Args: wailsJSONArgs(t, map[string]string{
				"path":     filePath,
				"encoding": "text",
			}),
		},
	})
	if err != nil {
		t.Fatalf("InvokeTool() error = %v", err)
	}
	if response == nil || response.Body == nil {
		t.Fatal("InvokeTool() returned an empty body")
	}
	if !response.Body.IsBuiltIn {
		t.Fatal("InvokeTool() reported IsBuiltIn=false")
	}
	if response.Body.IsError {
		t.Fatalf(
			"InvokeTool() returned tool error: %s",
			response.Body.ErrorMessage,
		)
	}
	if response.Body.Meta == nil || response.Body.Meta["type"] != "go" {
		t.Fatalf("InvokeTool() metadata = %#v, want Go metadata", response.Body.Meta)
	}
	if len(response.Body.Outputs) != 1 ||
		response.Body.Outputs[0].TextItem == nil {
		t.Fatalf("InvokeTool() outputs = %#v, want one text output", response.Body.Outputs)
	}
	if got := response.Body.Outputs[0].TextItem.Text; got != content {
		t.Fatalf("InvokeTool() text output = %q, want %q", got, content)
	}
}

func TestInvokeToolRejectsProviderAPITool(t *testing.T) {
	store := newRuntimeToolStore(t)
	target, found := findBuiltInSDKTool(t, store)
	if !found {
		t.Skip("no built-in provider API tool is registered")
	}
	enableBuiltInToolForTest(t, store, target)

	runtime := NewToolRuntime(store)
	_, err := runtime.InvokeTool(t.Context(), &spec.InvokeToolRequest{
		BundleID: target.BundleID,
		ToolSlug: target.ToolSlug,
		Version:  target.ToolVersion,
		Body: &spec.InvokeToolRequestBody{
			Args: jsonutil.JSONRawString(`{}`),
		},
	})
	if err == nil {
		t.Fatal("InvokeTool() succeeded for a provider API tool")
	}
	if !strings.Contains(err.Error(), "API-backed tools") {
		t.Fatalf("InvokeTool() error = %v, want provider API rejection", err)
	}
}

func TestInvokeToolRejectsUnknownBundle(t *testing.T) {
	store := newRuntimeToolStore(t)
	runtime := NewToolRuntime(store)

	_, err := runtime.InvokeTool(t.Context(), &spec.InvokeToolRequest{
		BundleID: "not-a-built-in-bundle",
		ToolSlug: "tool",
		Version:  "v1",
		Body: &spec.InvokeToolRequestBody{
			Args: jsonutil.JSONRawString(`{}`),
		},
	})
	if err == nil {
		t.Fatal("InvokeTool() succeeded for an unknown bundle")
	}
}

func newRuntimeToolStore(t *testing.T) *toolStore.ToolStore {
	t.Helper()

	store, err := toolStore.NewToolStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewToolStore() error = %v", err)
	}
	t.Cleanup(store.Close)

	return store
}

func findBuiltInReadFileTool(
	t *testing.T,
	store *toolStore.ToolStore,
) toolSpec.ToolListItem {
	t.Helper()

	fileTools, err := fstool.NewFSTool()
	if err != nil {
		t.Fatalf("fstool.NewFSTool() error = %v", err)
	}
	functionID := string(fileTools.ReadFileTool().GoImpl.FuncID)

	items, err := store.ListBuiltInTools(t.Context())
	if err != nil {
		t.Fatalf("ListBuiltInTools() error = %v", err)
	}
	for _, item := range items {
		tool := item.ToolDefinition
		if tool.Type != toolSpec.ToolTypeGo ||
			tool.GoImpl == nil ||
			tool.GoImpl.Func != functionID {
			continue
		}
		return item
	}

	t.Fatalf("built-in ReadFile tool with function %q was not registered", functionID)
	return toolSpec.ToolListItem{}
}

func findBuiltInSDKTool(
	t *testing.T,
	store *toolStore.ToolStore,
) (toolSpec.ToolListItem, bool) {
	t.Helper()

	items, err := store.ListBuiltInTools(t.Context())
	if err != nil {
		t.Fatalf("ListBuiltInTools() error = %v", err)
	}
	for _, item := range items {
		if item.ToolDefinition.Type == toolSpec.ToolTypeSDK {
			return item, true
		}
	}
	return toolSpec.ToolListItem{}, false
}

func enableBuiltInToolForTest(
	t *testing.T,
	store *toolStore.ToolStore,
	item toolSpec.ToolListItem,
) {
	t.Helper()

	ctx := t.Context()
	bundle, isBuiltIn, err := store.GetAnyToolBundle(ctx, item.BundleID)
	if err != nil {
		t.Fatalf("GetAnyToolBundle() error = %v", err)
	}
	if !isBuiltIn || !bundle.IsBuiltIn {
		t.Fatal("test target is not a built-in tool bundle")
	}

	originalBundleEnabled := bundle.IsEnabled
	originalToolEnabled := item.ToolDefinition.IsEnabled
	t.Cleanup(func() {
		_, _ = store.PatchTool(ctx, &toolSpec.PatchToolRequest{
			BundleID: item.BundleID,
			ToolSlug: item.ToolSlug,
			Version:  item.ToolVersion,
			Body: &toolSpec.PatchToolRequestBody{
				IsEnabled: originalToolEnabled,
			},
		})
		_, _ = store.PatchToolBundle(ctx, &toolSpec.PatchToolBundleRequest{
			BundleID: item.BundleID,
			Body: &toolSpec.PatchToolBundleRequestBody{
				IsEnabled: originalBundleEnabled,
			},
		})
	})

	if !bundle.IsEnabled {
		_, err = store.PatchToolBundle(ctx, &toolSpec.PatchToolBundleRequest{
			BundleID: item.BundleID,
			Body: &toolSpec.PatchToolBundleRequestBody{
				IsEnabled: true,
			},
		})
		if err != nil {
			t.Fatalf("PatchToolBundle() error = %v", err)
		}
	}
	if !item.ToolDefinition.IsEnabled {
		_, err = store.PatchTool(ctx, &toolSpec.PatchToolRequest{
			BundleID: item.BundleID,
			ToolSlug: item.ToolSlug,
			Version:  item.ToolVersion,
			Body: &toolSpec.PatchToolRequestBody{
				IsEnabled: true,
			},
		})
		if err != nil {
			t.Fatalf("PatchTool() error = %v", err)
		}
	}
}

func wailsJSONArgs(t *testing.T, value any) jsonutil.JSONRawString {
	t.Helper()

	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal(raw args) error = %v", err)
	}

	wire, err := json.Marshal(string(raw))
	if err != nil {
		t.Fatalf("json.Marshal(Wails args) error = %v", err)
	}

	return jsonutil.JSONRawString(string(wire))
}
