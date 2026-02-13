package spec

import "github.com/flexigpt/flexigpt-app/internal/bundleitemutils"

type ToolStoreChoiceType string

const (
	ToolStoreChoiceTypeFunction  ToolStoreChoiceType = "function"
	ToolStoreChoiceTypeCustom    ToolStoreChoiceType = "custom"
	ToolStoreChoiceTypeWebSearch ToolStoreChoiceType = "webSearch"
)

type ToolStoreChoice struct {
	ChoiceID string `json:"choiceID"`

	// BundleID, BundleSlug, ItemID, ItemSlug are string aliases.
	BundleID   bundleitemutils.BundleID   `json:"bundleID"`
	BundleSlug bundleitemutils.BundleSlug `json:"bundleSlug,omitempty"`

	ToolID      bundleitemutils.ItemID   `json:"toolID,omitempty"`
	ToolSlug    bundleitemutils.ItemSlug `json:"toolSlug"`
	ToolVersion string                   `json:"toolVersion"`

	ToolType    ToolStoreChoiceType `json:"toolType"`
	Description string              `json:"description,omitempty"`
	DisplayName string              `json:"displayName,omitempty"`

	// AutoExecute flag tells whether the tool should be automatically invoked once a llm calls this.
	AutoExecute bool `json:"autoExecute"`

	// UserArgSchemaInstance is an optional per-choice configuration object.
	//
	// For SDK-backed tools (Tool.Type == "sdk"), this typically contains provider-specific options validated against
	// Tool.UserArgSchema. The inferencewrapper interprets this JSON and maps it into the appropriate inference-go
	// ToolChoice fields (e.g. WebSearchToolChoiceItem).
	UserArgSchemaInstance JSONRawString `json:"userArgSchemaInstance,omitempty"`
}
