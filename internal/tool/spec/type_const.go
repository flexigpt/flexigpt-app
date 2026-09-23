package spec

import (
	"encoding/json"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
)

const (
	ToolBuiltInOverlayDBFileName = "toolsbuiltin.overlay.sqlite"

	// SchemaVersion  - Current on-disk schema version.
	SchemaVersion = "2025-07-01"
)

type ToolImplType string

const (
	ToolTypeGo  ToolImplType = "go"
	ToolTypeSDK ToolImplType = "sdk"
)

// GoToolImpl - Register-by-name pattern for Go tools.
type GoToolImpl struct {
	// Fully-qualified registration key, e.g.
	//   "github.com/acme/flexigpt/tools.Weather"
	Func string `json:"func" validate:"required"`
}

// SDKToolImpl describes how this tool is implemented using a provider SDK (e.g., OpenAI Responses, Anthropic Messages).
// It does not encode semantic kind (function/webSearch/etc.); that lives in Tool.LLMToolType.
type SDKToolImpl struct {
	// SDKType can be ProviderSDKType.
	SDKType string `json:"sdkType"`
}

type ToolRef struct {
	BundleID    bundleitemutils.BundleID    `json:"bundleID"`
	ToolSlug    bundleitemutils.ItemSlug    `json:"toolSlug"`
	ToolVersion bundleitemutils.ItemVersion `json:"toolVersion"`
}

type Tool struct {
	SchemaVersion string                      `json:"schemaVersion"`
	ID            bundleitemutils.ItemID      `json:"id"` // UUID-v7
	Slug          bundleitemutils.ItemSlug    `json:"slug"`
	Version       bundleitemutils.ItemVersion `json:"version"` // opaque

	DisplayName string   `json:"displayName"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`

	// UserCallable indicates whether the tool can be invoked directly by the user
	// (e.g. from the composer UI before sending a message).
	UserCallable bool `json:"userCallable"`
	// LLMCallable indicates whether the model may call this tool as a function.
	LLMCallable bool `json:"llmCallable"`
	// AutoExecReco indicates whether the host/UI should consider it safe enough
	// to auto-execute this tool without additional confirmation. Default: false.
	AutoExecReco bool `json:"autoExecute"`

	// ArgSchema describes the JSON arguments that are passed when the tool is invoked (by the LLM or via InvokeTool).
	// This is primarily used for Go tools.
	ArgSchema json.RawMessage `json:"argSchema"`

	// UserArgSchema, if present, describes an additional per-choice configuration object that the UI may collect when
	// enabling the tool for a model.
	// For SDK-backed API tools this typically encodes provider-specific options.
	UserArgSchema json.RawMessage `json:"userArgSchema,omitempty"`

	// LLMToolType captures the semantic kind of this tool from the model's point of view, e.g. "function", "custom",
	// "webSearch".
	// This value should always be one of ToolStoreChoiceType values.
	LLMToolType ToolStoreChoiceType `json:"llmToolType"`

	Type    ToolImplType `json:"type"`
	GoImpl  *GoToolImpl  `json:"goImpl,omitempty"`
	SDKImpl *SDKToolImpl `json:"sdkImpl,omitempty"`

	IsEnabled  bool      `json:"isEnabled"`
	IsBuiltIn  bool      `json:"isBuiltIn"`
	CreatedAt  time.Time `json:"createdAt"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

type ToolBundle struct {
	SchemaVersion string                     `json:"schemaVersion"`
	ID            bundleitemutils.BundleID   `json:"id"` // UUID-v7
	Slug          bundleitemutils.BundleSlug `json:"slug"`

	DisplayName string `json:"displayName,omitempty"`
	Description string `json:"description,omitempty"`

	IsEnabled  bool      `json:"isEnabled"`
	IsBuiltIn  bool      `json:"isBuiltIn"`
	CreatedAt  time.Time `json:"createdAt"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

type AllBundles struct {
	Bundles map[bundleitemutils.BundleID]ToolBundle `json:"bundles"`
}
