package spec

import (
	"time"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	modelpresetSpec "github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
	promptSpec "github.com/flexigpt/flexigpt-app/internal/prompt/spec"
	skillSpec "github.com/flexigpt/flexigpt-app/internal/skill/spec"
	toolSpec "github.com/flexigpt/flexigpt-app/internal/tool/spec"
)

type PutAssistantPresetBundleRequestBody struct {
	Slug        bundleitemutils.BundleSlug `json:"slug"                  required:"true"`
	DisplayName string                     `json:"displayName"           required:"true"`
	Description string                     `json:"description,omitempty"`
	IsEnabled   bool                       `json:"isEnabled"             required:"true"`
}

type PutAssistantPresetBundleRequest struct {
	BundleID bundleitemutils.BundleID `path:"bundleID" required:"true"`
	Body     *PutAssistantPresetBundleRequestBody
}

type PutAssistantPresetBundleResponse struct{}

type PatchAssistantPresetBundleRequestBody struct {
	IsEnabled bool `json:"isEnabled" required:"true"`
}

type PatchAssistantPresetBundleRequest struct {
	BundleID bundleitemutils.BundleID `path:"bundleID" required:"true"`
	Body     *PatchAssistantPresetBundleRequestBody
}

type PatchAssistantPresetBundleResponse struct{}

type DeleteAssistantPresetBundleRequest struct {
	BundleID bundleitemutils.BundleID `path:"bundleID" required:"true"`
}

type DeleteAssistantPresetBundleResponse struct{}

type BundlePageToken struct {
	BundleIDs       []bundleitemutils.BundleID `json:"ids,omitempty"` //nolint:tagliatelle // Page token encoding.
	IncludeDisabled bool                       `json:"d,omitempty"`   //nolint:tagliatelle // Page token encoding.
	PageSize        int                        `json:"s,omitempty"`   //nolint:tagliatelle // Page token encoding.
	CursorMod       string                     `json:"t,omitempty"`   //nolint:tagliatelle // RFC3339Nano.
	CursorID        bundleitemutils.BundleID   `json:"id,omitempty"`  //nolint:tagliatelle // Page token encoding.
}

type ListAssistantPresetBundlesRequest struct {
	BundleIDs       []bundleitemutils.BundleID `query:"bundleIDs"`
	IncludeDisabled bool                       `query:"includeDisabled"`
	PageSize        int                        `query:"pageSize"`
	PageToken       string                     `query:"pageToken"`
}

type ListAssistantPresetBundlesResponseBody struct {
	AssistantPresetBundles []AssistantPresetBundle `json:"assistantPresetBundles"`
	NextPageToken          *string                 `json:"nextPageToken,omitempty"`
}

type ListAssistantPresetBundlesResponse struct {
	Body *ListAssistantPresetBundlesResponseBody
}

type PutAssistantPresetRequestBody struct {
	DisplayName string `json:"displayName"           required:"true"`
	Description string `json:"description,omitempty"`
	IsEnabled   bool   `json:"isEnabled"             required:"true"`

	StartingModelPresetRef *modelpresetSpec.ModelPresetRef `json:"startingModelPresetRef,omitempty"`

	// Validation rules:
	//   - systemPrompt must be nil
	//   - capabilitiesOverride must be nil
	StartingModelPresetPatch *modelpresetSpec.ModelPresetPatch `json:"startingModelPresetPatch,omitempty"`

	StartingIncludeModelSystemPrompt *bool                          `json:"startingIncludeModelSystemPrompt,omitempty"`
	StartingInstructionTemplateRefs  []promptSpec.PromptTemplateRef `json:"startingInstructionTemplateRefs,omitempty"`
	StartingToolSelections           []toolSpec.ToolSelection       `json:"startingToolSelections,omitempty"`
	StartingSkillSelections          []skillSpec.SkillSelection     `json:"startingSkillSelections,omitempty"`
}

type PutAssistantPresetRequest struct {
	BundleID            bundleitemutils.BundleID    `path:"bundleID"            required:"true"`
	AssistantPresetSlug bundleitemutils.ItemSlug    `path:"assistantPresetSlug" required:"true"`
	Version             bundleitemutils.ItemVersion `path:"version"             required:"true"`
	Body                *PutAssistantPresetRequestBody
}

type PutAssistantPresetResponse struct{}

type PatchAssistantPresetRequestBody struct {
	IsEnabled bool `json:"isEnabled" required:"true"`
}

type PatchAssistantPresetRequest struct {
	BundleID            bundleitemutils.BundleID    `path:"bundleID"            required:"true"`
	AssistantPresetSlug bundleitemutils.ItemSlug    `path:"assistantPresetSlug" required:"true"`
	Version             bundleitemutils.ItemVersion `path:"version"             required:"true"`
	Body                *PatchAssistantPresetRequestBody
}

type PatchAssistantPresetResponse struct{}

type DeleteAssistantPresetRequest struct {
	BundleID            bundleitemutils.BundleID    `path:"bundleID"            required:"true"`
	AssistantPresetSlug bundleitemutils.ItemSlug    `path:"assistantPresetSlug" required:"true"`
	Version             bundleitemutils.ItemVersion `path:"version"             required:"true"`
}

type DeleteAssistantPresetResponse struct{}

type GetAssistantPresetRequest struct {
	BundleID            bundleitemutils.BundleID    `path:"bundleID"            required:"true"`
	AssistantPresetSlug bundleitemutils.ItemSlug    `path:"assistantPresetSlug" required:"true"`
	Version             bundleitemutils.ItemVersion `path:"version"             required:"true"`
}

type GetAssistantPresetResponse struct {
	Body *AssistantPreset
}

type AssistantPresetPageToken struct {
	RecommendedPageSize int                        `json:"s,omitempty"`   //nolint:tagliatelle // Page token encoding.
	IncludeDisabled     bool                       `json:"d,omitempty"`   //nolint:tagliatelle // Page token encoding.
	BundleIDs           []bundleitemutils.BundleID `json:"ids,omitempty"` //nolint:tagliatelle // Page token encoding.
	DirTok              string                     `json:"dt,omitempty"`  //nolint:tagliatelle // Directory scan token.
	BuiltInDone         bool                       `json:"bi,omitempty"`  //nolint:tagliatelle // Page token encoding.
}

type ListAssistantPresetsRequest struct {
	BundleIDs           []bundleitemutils.BundleID `query:"bundleIDs"`
	IncludeDisabled     bool                       `query:"includeDisabled"`
	RecommendedPageSize int                        `query:"recommendedPageSize"`
	PageToken           string                     `query:"pageToken"`
}

type AssistantPresetListItem struct {
	BundleID               bundleitemutils.BundleID    `json:"bundleID"`
	BundleSlug             bundleitemutils.BundleSlug  `json:"bundleSlug"`
	AssistantPresetSlug    bundleitemutils.ItemSlug    `json:"assistantPresetSlug"`
	AssistantPresetVersion bundleitemutils.ItemVersion `json:"assistantPresetVersion"`
	DisplayName            string                      `json:"displayName"`
	Description            string                      `json:"description,omitempty"`
	IsEnabled              bool                        `json:"isEnabled"`
	IsBuiltIn              bool                        `json:"isBuiltIn"`
	ModifiedAt             *time.Time                  `json:"modifiedAt,omitempty"`
}

type ListAssistantPresetsResponseBody struct {
	AssistantPresetListItems []AssistantPresetListItem `json:"assistantPresetListItems"`
	NextPageToken            *string                   `json:"nextPageToken,omitempty"`
}

type ListAssistantPresetsResponse struct {
	Body *ListAssistantPresetsResponseBody
}
