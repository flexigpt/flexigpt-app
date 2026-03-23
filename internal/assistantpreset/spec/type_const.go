package spec

import (
	"errors"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	modelpresetSpec "github.com/flexigpt/flexigpt-app/internal/modelpreset/spec"
	promptSpec "github.com/flexigpt/flexigpt-app/internal/prompt/spec"
	skillSpec "github.com/flexigpt/flexigpt-app/internal/skill/spec"
	toolSpec "github.com/flexigpt/flexigpt-app/internal/tool/spec"
)

const (
	AssistantPresetBundlesMetaFileName      = "assistantpresetbundles.json"
	AssistantPresetBuiltInOverlayDBFileName = "assistantpresetsbuiltin.overlay.sqlite"
	SchemaVersion                           = "2026-03-22"
	MaxPageSize                             = 256
	DefaultPageSize                         = 25
)

var (
	ErrInvalidRequest = errors.New("invalid request")
	ErrInvalidDir     = errors.New("invalid directory")
	ErrConflict       = errors.New("resource already exists")

	ErrBuiltInReadOnly       = errors.New("built-in resource is read-only")
	ErrBuiltInBundleNotFound = errors.New("built-in bundle not found")
	ErrBundleNotFound        = errors.New("bundle not found")
	ErrBundleDisabled        = errors.New("bundle is disabled")
	ErrBundleNotEmpty        = errors.New("bundle is not empty")
	ErrBundleDeleting        = errors.New("bundle is being deleted")

	ErrAssistantPresetNotFound = errors.New("assistant preset not found")
	ErrAssistantPresetDisabled = errors.New("assistant preset is disabled")
	ErrNilAssistantPreset      = errors.New("assistant preset is nil")
)

// AssistantPreset is an immutable starter configuration snapshot.
// One (slug, version) is stored as one JSON file.
type AssistantPreset struct {
	SchemaVersion string `json:"schemaVersion"`

	ID          bundleitemutils.ItemID      `json:"id"`
	Slug        bundleitemutils.ItemSlug    `json:"slug"`
	Version     bundleitemutils.ItemVersion `json:"version"`
	DisplayName string                      `json:"displayName"`
	Description string                      `json:"description,omitempty"`

	IsEnabled bool `json:"isEnabled"`
	IsBuiltIn bool `json:"isBuiltIn"`

	StartingModelPresetRef *modelpresetSpec.ModelPresetRef `json:"startingModelPresetRef,omitempty"`

	// Validation rules:
	//   - systemPrompt must be nil
	//   - capabilitiesOverride must be nil
	StartingModelPresetPatch *modelpresetSpec.ModelPresetPatch `json:"startingModelPresetPatch,omitempty"`

	// Nil means the preset does not express a preference.
	StartingIncludeModelSystemPrompt *bool `json:"startingIncludeModelSystemPrompt,omitempty"`

	// Ordered refs. Validation requires:
	//   - template exists
	//   - template enabled
	//   - kind == instructionsOnly
	//   - isResolved == true
	StartingInstructionTemplateRefs []promptSpec.PromptTemplateRef `json:"startingInstructionTemplateRefs,omitempty"`

	// Ordered tool selections.
	StartingToolSelections []toolSpec.ToolSelection `json:"startingToolSelections,omitempty"`

	// Ordered enabled skills.
	StartingEnabledSkillRefs []skillSpec.SkillRef `json:"startingEnabledSkillRefs,omitempty"`

	CreatedAt  time.Time `json:"createdAt"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

// AssistantPresetBundle is a notional grouping for assistant preset version files.
// Bundle metadata is stored in a shared meta file; actual assistant preset versions
// are stored as individual JSON files inside the bundle directory.
type AssistantPresetBundle struct {
	SchemaVersion string `json:"schemaVersion"`

	ID            bundleitemutils.BundleID   `json:"id"`
	Slug          bundleitemutils.BundleSlug `json:"slug"`
	DisplayName   string                     `json:"displayName"`
	Description   string                     `json:"description,omitempty"`
	IsEnabled     bool                       `json:"isEnabled"`
	IsBuiltIn     bool                       `json:"isBuiltIn"`
	CreatedAt     time.Time                  `json:"createdAt"`
	ModifiedAt    time.Time                  `json:"modifiedAt"`
	SoftDeletedAt *time.Time                 `json:"softDeletedAt,omitempty"`
}

type AllBundles struct {
	SchemaVersion string                                             `json:"schemaVersion"`
	Bundles       map[bundleitemutils.BundleID]AssistantPresetBundle `json:"bundles"`
}
