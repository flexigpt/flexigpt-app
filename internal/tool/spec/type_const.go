package spec

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
)

const (
	ToolBundlesMetaFileName      = "tools.bundles.json"
	ToolDBFileName               = "tools.fts.sqlite"
	ToolBuiltInOverlayDBFileName = "toolsbuiltin.overlay.sqlite"

	DefaultHTTPTimeoutMs = 10_000
	JSONEncoding         = "json"
	TextEncoding         = "text"
	DefaultHTTPEncoding  = JSONEncoding
	DefaultHTTPErrorMode = "fail"

	// SchemaVersion  - Current on-disk schema version.
	SchemaVersion = "2025-07-01"
)

var (
	ErrInvalidRequest = errors.New("invalid request")
	ErrInvalidDir     = errors.New("invalid directory")
	ErrConflict       = errors.New("resource already exists")

	ErrBuiltInBundleNotFound = errors.New("bundle not found in built-in data")
	ErrBundleNotFound        = errors.New("bundle not found")
	ErrBundleDisabled        = errors.New("bundle is disabled")
	ErrBundleDeleting        = errors.New("bundle is being deleted")
	ErrBundleNotEmpty        = errors.New("bundle still contains tools")

	ErrToolNotFound = errors.New("tool not found")

	ErrBuiltInReadOnly = errors.New("built-in resource is read-only")

	ErrFTSDisabled  = errors.New("FTS is disabled")
	ErrToolDisabled = errors.New("tool is disabled")
)

type (
	ToolImplType  string
	JSONRawString = string
	JSONSchema    = json.RawMessage
)

const (
	ToolTypeGo   ToolImplType = "go"
	ToolTypeHTTP ToolImplType = "http"
	ToolTypeSDK  ToolImplType = "sdk"
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

// HTTPAuth - Simple auth descriptor (can be extended later).
type HTTPAuth struct {
	Type          string `json:"type"`
	In            string `json:"in,omitempty"`   // "header" | "query"  (apiKey only)
	Name          string `json:"name,omitempty"` // header/query key
	ValueTemplate string `json:"valueTemplate"`  // may contain ${SECRET}
}
type HTTPRequest struct {
	Method      string            `json:"method,omitempty"`    // default "GET"
	URLTemplate string            `json:"urlTemplate"`         // http(s)://… may contain ${var}
	Query       map[string]string `json:"query,omitempty"`     // k:${var}
	Headers     map[string]string `json:"headers,omitempty"`   // k:${var}
	Body        string            `json:"body,omitempty"`      // raw or template
	Auth        *HTTPAuth         `json:"auth,omitempty"`      // see below
	TimeoutMs   int               `json:"timeoutMs,omitempty"` // default 10 000
}

// HTTPBodyOutputMode - how to map HTTP response body into tool outputs.
// "" / "auto" (default): infer from Content-Type.
// "text": always a single text block.
// "file": always a single file block.
// "image": always a single image block.
type HTTPBodyOutputMode string

const (
	HTTPBodyOutputModeAuto  HTTPBodyOutputMode = "auto"
	HTTPBodyOutputModeText  HTTPBodyOutputMode = "text"
	HTTPBodyOutputModeFile  HTTPBodyOutputMode = "file"
	HTTPBodyOutputModeImage HTTPBodyOutputMode = "image"
)

type HTTPResponse struct {
	SuccessCodes   []int              `json:"successCodes,omitempty"` // default: any 2xx
	ErrorMode      string             `json:"errorMode,omitempty"`    // "fail"(dflt) | "empty"
	BodyOutputMode HTTPBodyOutputMode `json:"bodyOutputMode,omitempty"`
}

type HTTPToolImpl struct {
	Request  HTTPRequest  `json:"request"`
	Response HTTPResponse `json:"response"`
}

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

type ToolStoreOutputKind string

const (
	ToolStoreOutputKindNone  ToolStoreOutputKind = "none"
	ToolStoreOutputKindText  ToolStoreOutputKind = "text"
	ToolStoreOutputKindImage ToolStoreOutputKind = "image"
	ToolStoreOutputKindFile  ToolStoreOutputKind = "file"
)

type ToolStoreOutputText struct {
	Text string `json:"text"`
}

type ImageDetail string

const (
	ImageDetailHigh ImageDetail = "high"
	ImageDetailLow  ImageDetail = "low"
	ImageDetailAuto ImageDetail = "auto"
)

type ToolStoreOutputImage struct {
	Detail    ImageDetail `json:"detail"`
	ImageName string      `json:"imageName"`
	ImageMIME string      `json:"imageMIME"`
	ImageData string      `json:"imageData"`
}

type ToolStoreOutputFile struct {
	FileName string `json:"fileName"`
	FileMIME string `json:"fileMIME"`
	FileData string `json:"fileData"`
}

type ToolStoreOutputUnion struct {
	Kind ToolStoreOutputKind `json:"kind"`

	TextItem  *ToolStoreOutputText  `json:"textItem,omitempty"`
	ImageItem *ToolStoreOutputImage `json:"imageItem,omitempty"`
	FileItem  *ToolStoreOutputFile  `json:"fileItem,omitempty"`
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
	AutoExecReco bool `json:"autoExecReco"`

	// ArgSchema describes the JSON arguments that are passed when the tool is invoked (by the LLM or via InvokeTool).
	// This is primarily used for Go/HTTP tools.
	ArgSchema JSONSchema `json:"argSchema"`

	// UserArgSchema, if present, describes an additional per-choice configuration object that the UI may collect when
	// enabling the tool for a model.
	// For SDK-backed server tools this typically encodes provider-specific options (e.g., web-search settings).
	UserArgSchema JSONSchema `json:"userArgSchema,omitempty"`

	// LLMToolType captures the semantic kind of this tool from the model's point of view, e.g. "function", "custom",
	// "webSearch".
	// This value should always be one of ToolStoreChoiceType values.
	LLMToolType ToolStoreChoiceType `json:"llmToolType"`

	Type     ToolImplType  `json:"type"`
	GoImpl   *GoToolImpl   `json:"goImpl,omitempty"`
	HTTPImpl *HTTPToolImpl `json:"httpImpl,omitempty"`
	SDKImpl  *SDKToolImpl  `json:"sdkImpl,omitempty"`

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

	IsEnabled     bool       `json:"isEnabled"`
	IsBuiltIn     bool       `json:"isBuiltIn"`
	CreatedAt     time.Time  `json:"createdAt"`
	ModifiedAt    time.Time  `json:"modifiedAt"`
	SoftDeletedAt *time.Time `json:"softDeletedAt,omitempty"`
}

type AllBundles struct {
	Bundles map[bundleitemutils.BundleID]ToolBundle `json:"bundles"`
}
