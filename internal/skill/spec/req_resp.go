package spec

import "github.com/flexigpt/flexigpt-app/internal/bundleitemutils"

type PutSkillBundleRequestBody struct {
	Slug        bundleitemutils.BundleSlug `json:"slug"                  required:"true"`
	DisplayName string                     `json:"displayName"           required:"true"`
	IsEnabled   bool                       `json:"isEnabled"             required:"true"`
	Description string                     `json:"description,omitempty"`
}

type PutSkillBundleRequest struct {
	BundleID bundleitemutils.BundleID `path:"bundleID" required:"true"`
	Body     *PutSkillBundleRequestBody
}

type PutSkillBundleResponse struct{}

type DeleteSkillBundleRequest struct {
	BundleID bundleitemutils.BundleID `path:"bundleID" required:"true"`
}
type DeleteSkillBundleResponse struct{}

type PatchSkillBundleRequestBody struct {
	IsEnabled bool `json:"isEnabled" required:"true"`
}

type PatchSkillBundleRequest struct {
	BundleID bundleitemutils.BundleID `path:"bundleID" required:"true"`
	Body     *PatchSkillBundleRequestBody
}

type PatchSkillBundleResponse struct{}

// SkillBundlePageToken is a stable cursor for bundle listing.
// Same style as BundlePageToken in tools.
type SkillBundlePageToken struct {
	BundleIDs       []bundleitemutils.BundleID `json:"ids,omitempty"` //nolint:tagliatelle // Page Token specific. // optional filter
	IncludeDisabled bool                       `json:"d,omitempty"`   //nolint:tagliatelle // Page Token specific.
	PageSize        int                        `json:"s"`             //nolint:tagliatelle // Page Token specific.
	CursorMod       string                     `json:"t,omitempty"`   //nolint:tagliatelle // Page Token specific.// RFC-3339-nano modifiedAt
	CursorID        bundleitemutils.BundleID   `json:"id,omitempty"`  //nolint:tagliatelle // Page Token specific.// tie-breaker
}

type ListSkillBundlesRequest struct {
	BundleIDs       []bundleitemutils.BundleID `query:"bundleIDs"`
	IncludeDisabled bool                       `query:"includeDisabled"`
	PageSize        int                        `query:"pageSize"`
	PageToken       string                     `query:"pageToken"`
}

type ListSkillBundlesResponseBody struct {
	SkillBundles  []SkillBundle `json:"skillBundles"`
	NextPageToken *string       `json:"nextPageToken,omitempty"`
}

type ListSkillBundlesResponse struct {
	Body *ListSkillBundlesResponseBody
}

type PutSkillRequestBody struct {
	SkillType SkillType `json:"skillType" required:"true"`
	Location  string    `json:"location"  required:"true"`
	Name      string    `json:"name"      required:"true"`
	IsEnabled bool      `json:"isEnabled" required:"true"`

	DisplayName string   `json:"displayName,omitempty"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

type PutSkillRequest struct {
	BundleID  bundleitemutils.BundleID `path:"bundleID"  required:"true"`
	SkillSlug SkillSlug                `path:"skillSlug" required:"true"`
	Body      *PutSkillRequestBody
}

type PutSkillResponse struct{}

type DeleteSkillRequest struct {
	BundleID  bundleitemutils.BundleID `path:"bundleID"  required:"true"`
	SkillSlug SkillSlug                `path:"skillSlug" required:"true"`
}
type DeleteSkillResponse struct{}

type PatchSkillRequestBody struct {
	IsEnabled *bool   `json:"isEnabled,omitempty"`
	Location  *string `json:"location,omitempty"`
}

type PatchSkillRequest struct {
	BundleID  bundleitemutils.BundleID `path:"bundleID"  required:"true"`
	SkillSlug SkillSlug                `path:"skillSlug" required:"true"`
	Body      *PatchSkillRequestBody
}

type PatchSkillResponse struct{}

type GetSkillRequest struct {
	BundleID  bundleitemutils.BundleID `path:"bundleID"  required:"true"`
	SkillSlug SkillSlug                `path:"skillSlug" required:"true"`
}
type GetSkillResponse struct{ Body *Skill }

type ListSkillPhase string

const (
	ListSkillPhaseBuiltIn ListSkillPhase = "builtin"
	ListSkillPhaseUser    ListSkillPhase = "user"
)

// SkillPageToken for paging skills across bundles.
// Mirrors ToolPageToken but without versioning.
type SkillPageToken struct {
	RecommendedPageSize int                        `json:"ps,omitempty"`  //nolint:tagliatelle // Page token specific.
	IncludeDisabled     bool                       `json:"d,omitempty"`   //nolint:tagliatelle // Page token specific.
	IncludeMissing      bool                       `json:"m,omitempty"`   //nolint:tagliatelle // Page token specific. // include presence=missing?
	BundleIDs           []bundleitemutils.BundleID `json:"ids,omitempty"` //nolint:tagliatelle // Page token specific. // optional filter
	Types               []SkillType                `json:"ty,omitempty"`  //nolint:tagliatelle // Page token specific. // optional filter
	Phase               ListSkillPhase             `json:"ph,omitempty"`  //nolint:tagliatelle //nolint:tagliatelle // Page token specific.
	BuiltInCursor       string                     `json:"bc,omitempty"`  //nolint:tagliatelle // opaque: last (bundleID|skillSlug)
	DirTok              string                     `json:"dt,omitempty"`  //nolint:tagliatelle // user cursor
}

type ListSkillsRequest struct {
	BundleIDs           []bundleitemutils.BundleID `query:"bundleIDs"`
	Types               []SkillType                `query:"types"`
	IncludeDisabled     bool                       `query:"includeDisabled"`
	IncludeMissing      bool                       `query:"includeMissing"`
	RecommendedPageSize int                        `query:"recommendedPageSize"`
	PageToken           string                     `query:"pageToken"`
}

type SkillListItem struct {
	BundleID   bundleitemutils.BundleID   `json:"bundleID"`
	BundleSlug bundleitemutils.BundleSlug `json:"bundleSlug"`

	SkillSlug SkillSlug `json:"skillSlug"`
	IsBuiltIn bool      `json:"isBuiltIn"`

	SkillDefinition Skill `json:"skillDefinition"`
}

type ListSkillsResponseBody struct {
	SkillListItems []SkillListItem `json:"skillListItems"`
	NextPageToken  *string         `json:"nextPageToken,omitempty"`
}
type ListSkillsResponse struct {
	Body *ListSkillsResponseBody
}
