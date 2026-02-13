package spec

import (
	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
)

type PutToolBundleRequestBody struct {
	Slug        bundleitemutils.BundleSlug `json:"slug"                  required:"true"`
	DisplayName string                     `json:"displayName"           required:"true"`
	IsEnabled   bool                       `json:"isEnabled"             required:"true"`
	Description string                     `json:"description,omitempty"`
}

type PutToolBundleRequest struct {
	BundleID bundleitemutils.BundleID `path:"bundleID" required:"true"`
	Body     *PutToolBundleRequestBody
}

type PutToolBundleResponse struct{}

type DeleteToolBundleRequest struct {
	BundleID bundleitemutils.BundleID `path:"bundleID" required:"true"`
}
type DeleteToolBundleResponse struct{}

type PatchToolBundleRequestBody struct {
	IsEnabled bool `json:"isEnabled" required:"true"`
}

type PatchToolBundleRequest struct {
	BundleID bundleitemutils.BundleID `path:"bundleID" required:"true"`
	Body     *PatchToolBundleRequestBody
}

type PatchToolBundleResponse struct{}

// BundlePageToken is the opaque cursor used when paging over tool-bundles.
type BundlePageToken struct {
	BundleIDs       []bundleitemutils.BundleID `json:"ids,omitempty"` //nolint:tagliatelle // Pagttoken Specific. // Optional bundle-ID filter.
	IncludeDisabled bool                       `json:"d,omitempty"`   //nolint:tagliatelle // Pagttoken Specific. // Include disabled bundles?
	PageSize        int                        `json:"s"`             //nolint:tagliatelle // Pagttoken Specific. // Requested page-size.
	CursorMod       string                     `json:"t,omitempty"`   //nolint:tagliatelle // Pagttoken Specific. // RFC-3339-nano modification timestamp.
	CursorID        bundleitemutils.BundleID   `json:"id,omitempty"`  //nolint:tagliatelle // Pagttoken Specific. // Tie-breaker for equal timestamps.
}

type ListToolBundlesRequest struct {
	BundleIDs       []bundleitemutils.BundleID `query:"bundleIDs"`
	IncludeDisabled bool                       `query:"includeDisabled"`
	PageSize        int                        `query:"pageSize"`
	PageToken       string                     `query:"pageToken"`
}

type ListToolBundlesResponseBody struct {
	ToolBundles   []ToolBundle `json:"toolBundles"`
	NextPageToken *string      `json:"nextPageToken,omitempty"`
}

type ListToolBundlesResponse struct {
	Body *ListToolBundlesResponseBody
}

type PutToolRequestBody struct {
	DisplayName string   `json:"displayName"           required:"true"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	IsEnabled   bool     `json:"isEnabled"             required:"true"`

	UserCallable bool `json:"userCallable" required:"true"`
	LLMCallable  bool `json:"llmCallable"  required:"true"`
	AutoExecReco bool `json:"autoExecReco" required:"true"`

	// Take inputs as strings that we can then validate as a json object and put a tool.
	ArgSchema JSONRawString `json:"argSchema" required:"true"`

	Type     ToolImplType  `json:"type"               required:"true"`
	HTTPImpl *HTTPToolImpl `json:"httpImpl,omitempty" required:"true"`
}

type PutToolRequest struct {
	BundleID bundleitemutils.BundleID    `path:"bundleID" required:"true"`
	ToolSlug bundleitemutils.ItemSlug    `path:"toolSlug" required:"true"`
	Version  bundleitemutils.ItemVersion `path:"version"  required:"true"`
	Body     *PutToolRequestBody
}

type PutToolResponse struct{}

type DeleteToolRequest struct {
	BundleID bundleitemutils.BundleID    `path:"bundleID" required:"true"`
	ToolSlug bundleitemutils.ItemSlug    `path:"toolSlug" required:"true"`
	Version  bundleitemutils.ItemVersion `path:"version"  required:"true"`
}
type DeleteToolResponse struct{}

type PatchToolRequestBody struct {
	IsEnabled bool `json:"isEnabled" required:"true"`
}

type PatchToolRequest struct {
	BundleID bundleitemutils.BundleID    `path:"bundleID" required:"true"`
	ToolSlug bundleitemutils.ItemSlug    `path:"toolSlug" required:"true"`
	Version  bundleitemutils.ItemVersion `path:"version"  required:"true"`

	Body *PatchToolRequestBody
}

type PatchToolResponse struct{}

type GetToolRequest struct {
	BundleID bundleitemutils.BundleID    `path:"bundleID" required:"true"`
	ToolSlug bundleitemutils.ItemSlug    `path:"toolSlug" required:"true"`
	Version  bundleitemutils.ItemVersion `path:"version"  required:"true"`
}
type GetToolResponse struct{ Body *Tool }

// ToolPageToken is the opaque cursor used when paging over individual tool
// versions (ListTools API).
type ToolPageToken struct {
	RecommendedPageSize int                        `json:"ps,omitempty"`   //nolint:tagliatelle // PageToken specific.
	IncludeDisabled     bool                       `json:"d,omitempty"`    //nolint:tagliatelle // PageToken specific.
	BundleIDs           []bundleitemutils.BundleID `json:"ids,omitempty"`  //nolint:tagliatelle // PageToken specific.
	Tags                []string                   `json:"tags,omitempty"` //nolint:tagliatelle // PageToken specific.
	BuiltInDone         bool                       `json:"bd,omitempty"`   //nolint:tagliatelle // PageToken specific. // Built-ins already emitted?
	DirTok              string                     `json:"dt,omitempty"`   //nolint:tagliatelle // PageToken specific. // Directory-store cursor.
}

type ListToolsRequest struct {
	BundleIDs           []bundleitemutils.BundleID `query:"bundleIDs"`
	Tags                []string                   `query:"tags"`
	IncludeDisabled     bool                       `query:"includeDisabled"`
	RecommendedPageSize int                        `query:"recommendedPageSize"`
	PageToken           string                     `query:"pageToken"`
}

type ToolListItem struct {
	BundleID       bundleitemutils.BundleID    `json:"bundleID"`
	BundleSlug     bundleitemutils.BundleSlug  `json:"bundleSlug"`
	ToolSlug       bundleitemutils.ItemSlug    `json:"toolSlug"`
	ToolVersion    bundleitemutils.ItemVersion `json:"toolVersion"`
	IsBuiltIn      bool                        `json:"isBuiltIn"`
	ToolDefinition Tool                        `json:"toolDefinition"`
}

type ListToolsResponseBody struct {
	ToolListItems []ToolListItem `json:"toolListItems"`
	NextPageToken *string        `json:"nextPageToken,omitempty"`
}
type ListToolsResponse struct {
	Body *ListToolsResponseBody
}
