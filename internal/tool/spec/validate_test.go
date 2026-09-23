package spec

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func validBuiltInGoTool() Tool {
	now := time.Now().UTC()

	return Tool{
		SchemaVersion: SchemaVersion,
		ID:            "builtin-tool-id",
		Slug:          "builtin-tool",
		Version:       "v1",
		DisplayName:   "Built-in Tool",
		Description:   "Test built-in tool",
		UserCallable:  true,
		LLMCallable:   true,
		ArgSchema:     json.RawMessage(`{"type":"object"}`),
		LLMToolType:   ToolStoreChoiceTypeFunction,
		Type:          ToolTypeGo,
		GoImpl: &GoToolImpl{
			Func: "github.com/flexigpt/flexigpt-app/tests/BuiltInTool",
		},
		IsEnabled:  true,
		IsBuiltIn:  true,
		CreatedAt:  now,
		ModifiedAt: now,
	}
}

func TestToolValidateAcceptsBuiltInGoTool(t *testing.T) {
	tool := validBuiltInGoTool()

	if err := tool.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestToolValidateAcceptsBuiltInSDKTool(t *testing.T) {
	tool := validBuiltInGoTool()
	tool.Type = ToolTypeSDK
	tool.GoImpl = nil
	tool.SDKImpl = &SDKToolImpl{
		SDKType: "provider-api",
	}

	if err := tool.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestToolValidateRejectsNonBuiltInTool(t *testing.T) {
	tool := validBuiltInGoTool()
	tool.IsBuiltIn = false

	err := tool.Validate()
	if err == nil {
		t.Fatal("Validate() succeeded for a non-built-in tool")
	}
	if !strings.Contains(err.Error(), "only built-in tools") {
		t.Fatalf("Validate() error = %v, want built-in error", err)
	}
}

func TestToolValidateRejectsLegacyHTTPToolType(t *testing.T) {
	tool := validBuiltInGoTool()
	tool.Type = ToolImplType("http")
	tool.GoImpl = nil

	err := tool.Validate()
	if err == nil {
		t.Fatal("Validate() succeeded for legacy HTTP tool type")
	}
	if !strings.Contains(err.Error(), "only go and sdk are supported") {
		t.Fatalf("Validate() error = %v, want unsupported type error", err)
	}
}
