package spec

import "github.com/flexigpt/flexigpt-app/internal/bundleitemutils"

type PutPromptBundleRequestBody struct {
	Slug        bundleitemutils.BundleSlug `json:"slug"                  required:"true"`
	DisplayName string                     `json:"displayName"           required:"true"`
	IsEnabled   bool                       `json:"isEnabled"             required:"true"`
	Description string                     `json:"description,omitempty"`
}

type PutPromptBundleRequest struct {
	BundleID bundleitemutils.BundleID `path:"bundleID" required:"true"`
	Body     *PutPromptBundleRequestBody
}

type PutPromptBundleResponse struct{}

type DeletePromptBundleRequest struct {
	BundleID bundleitemutils.BundleID `path:"bundleID" required:"true"`
}
type DeletePromptBundleResponse struct{}

type PatchPromptBundleRequestBody struct {
	IsEnabled bool `json:"isEnabled" required:"true"`
}

type PatchPromptBundleRequest struct {
	BundleID bundleitemutils.BundleID `path:"bundleID" required:"true"`
	Body     *PatchPromptBundleRequestBody
}

type PatchPromptBundleResponse struct{}

type BundlePageToken struct {
	BundleIDs       []bundleitemutils.BundleID `json:"ids,omitempty"` //nolint:tagliatelle // Pagetoken specific. optional filter
	IncludeDisabled bool                       `json:"d,omitempty"`   //nolint:tagliatelle // Pagetoken specific. include disabled bundles?
	PageSize        int                        `json:"s"`             //nolint:tagliatelle // Pagetoken specific. page size
	CursorMod       string                     `json:"t,omitempty"`   //nolint:tagliatelle // Pagetoken specific. RFC-3339-nano
	CursorID        bundleitemutils.BundleID   `json:"id,omitempty"`  //nolint:tagliatelle // Pagetoken specific.
}

type ListPromptBundlesRequest struct {
	BundleIDs       []bundleitemutils.BundleID `query:"bundleIDs"`
	IncludeDisabled bool                       `query:"includeDisabled"`
	PageSize        int                        `query:"pageSize"`
	PageToken       string                     `query:"pageToken"`
}

type ListPromptBundlesResponseBody struct {
	PromptBundles []PromptBundle `json:"promptBundles"`
	NextPageToken *string        `json:"nextPageToken,omitempty"`
}

type ListPromptBundlesResponse struct {
	Body *ListPromptBundlesResponseBody
}

type PutPromptTemplateRequestBody struct {
	// Auto populate template id internally.
	DisplayName string `json:"displayName"           required:"true"`
	IsEnabled   bool   `json:"isEnabled"             required:"true"`
	Description string `json:"description,omitempty"`

	Blocks    []MessageBlock   `json:"blocks"              required:"true"`
	Tags      []string         `json:"tags,omitempty"`
	Variables []PromptVariable `json:"variables,omitempty"`
}

type PutPromptTemplateRequest struct {
	BundleID     bundleitemutils.BundleID    `path:"bundleID"     required:"true"`
	TemplateSlug bundleitemutils.ItemSlug    `path:"templateSlug" required:"true"`
	Version      bundleitemutils.ItemVersion `path:"version"      required:"true"`
	Body         *PutPromptTemplateRequestBody
}

type PutPromptTemplateResponse struct{}

type DeletePromptTemplateRequest struct {
	BundleID     bundleitemutils.BundleID    `path:"bundleID"     required:"true"`
	TemplateSlug bundleitemutils.ItemSlug    `path:"templateSlug" required:"true"`
	Version      bundleitemutils.ItemVersion `path:"version"      required:"true"`
}
type DeletePromptTemplateResponse struct{}

type PatchPromptTemplateRequestBody struct {
	IsEnabled bool `json:"isEnabled" required:"true"`
}

type PatchPromptTemplateRequest struct {
	BundleID     bundleitemutils.BundleID    `path:"bundleID"     required:"true"`
	TemplateSlug bundleitemutils.ItemSlug    `path:"templateSlug" required:"true"`
	Version      bundleitemutils.ItemVersion `path:"version"      required:"true"`
	Body         *PatchPromptTemplateRequestBody
}

type PatchPromptTemplateResponse struct{}

type GetPromptTemplateRequest struct {
	BundleID     bundleitemutils.BundleID    `path:"bundleID"     required:"true"`
	TemplateSlug bundleitemutils.ItemSlug    `path:"templateSlug" required:"true"`
	Version      bundleitemutils.ItemVersion `path:"version"      required:"true"`
}
type GetPromptTemplateResponse struct{ Body *PromptTemplate }

type TemplatePageToken struct {
	RecommendedPageSize int                        `json:"ps,omitempty"`   //nolint:tagliatelle // PageToken specific.
	IncludeDisabled     bool                       `json:"d,omitempty"`    //nolint:tagliatelle // PageToken specific.
	BundleIDs           []bundleitemutils.BundleID `json:"ids,omitempty"`  //nolint:tagliatelle // PageToken specific.
	Tags                []string                   `json:"tags,omitempty"` //nolint:tagliatelle // PageToken specific.
	BuiltInDone         bool                       `json:"bd,omitempty"`   //nolint:tagliatelle // PageToken specific.
	DirTok              string                     `json:"dt,omitempty"`   //nolint:tagliatelle // PageToken specific.
}

type ListPromptTemplatesRequest struct {
	BundleIDs           []bundleitemutils.BundleID `query:"bundleIDs"`
	Tags                []string                   `query:"tags"`
	IncludeDisabled     bool                       `query:"includeDisabled"`
	RecommendedPageSize int                        `query:"recommendedPageSize"`
	PageToken           string                     `query:"pageToken"`
}

type PromptTemplateListItem struct {
	BundleID        bundleitemutils.BundleID    `json:"bundleID"`
	BundleSlug      bundleitemutils.BundleSlug  `json:"bundleSlug"`
	TemplateSlug    bundleitemutils.ItemSlug    `json:"templateSlug"`
	TemplateVersion bundleitemutils.ItemVersion `json:"templateVersion"`
	IsBuiltIn       bool                        `json:"isBuiltIn"`
}

type ListPromptTemplatesResponseBody struct {
	PromptTemplateListItems []PromptTemplateListItem `json:"promptTemplateListItems"`
	NextPageToken           *string                  `json:"nextPageToken,omitempty"`
}

type ListPromptTemplatesResponse struct {
	Body *ListPromptTemplatesResponseBody
}
type SearchPromptTemplatesRequest struct {
	Query           string `query:"q"               required:"true"`
	PageToken       string `query:"pageToken"`
	PageSize        int    `query:"pageSize"`
	IncludeDisabled bool   `query:"includeDisabled"`
}

type SearchPromptTemplatesResponseBody struct {
	PromptTemplateListItems []PromptTemplateListItem `json:"promptTemplateListItems"`
	NextPageToken           *string                  `json:"nextPageToken,omitempty"`
}

type SearchPromptTemplatesResponse struct {
	Body *SearchPromptTemplatesResponseBody
}
