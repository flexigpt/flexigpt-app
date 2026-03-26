package spec

import (
	"errors"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
)

const (
	PromptBundlesMetaFileName      = "prompts.bundles.json"
	PromptDBFileName               = "prompts.fts.sqlite"
	PromptBuiltInOverlayDBFileName = "promptsbuiltin.overlay.sqlite"

	// BasePromptBundleID Reserved writable bundle that acts as the default scratch-pad for user prompts.
	BasePromptBundleID          bundleitemutils.BundleID   = "019d2a53-964e-7700-b8b7-24573c722142"
	BasePromptBundleSlug        bundleitemutils.BundleSlug = "base"
	BasePromptBundleDisplayName                            = "Base"
	BasePromptBundleDescription                            = "Editable starter bundle for custom prompts."

	// SchemaVersion is the current on-disk schema version.
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
	ErrBundleNotEmpty        = errors.New("bundle still contains templates")
	ErrTemplateNotFound      = errors.New("template not found")

	ErrReservedBundleReadOnly = errors.New("reserved bundle metadata is read-only")
	ErrBuiltInReadOnly        = errors.New("built-in resource is read-only")

	ErrFTSDisabled = errors.New("FTS is disabled")
)

type MessageBlockID string

type PromptRoleEnum string

const (
	System    PromptRoleEnum = "system"
	Developer PromptRoleEnum = "developer"
	User      PromptRoleEnum = "user"
)

type PromptTemplateKind string

const (
	PromptTemplateKindInstructionsOnly PromptTemplateKind = "instructionsOnly"
	PromptTemplateKindGeneric          PromptTemplateKind = "generic"
)

type VarType string

const (
	VarString  VarType = "string"
	VarNumber  VarType = "number"
	VarBoolean VarType = "boolean"
	VarEnum    VarType = "enum"
	VarDate    VarType = "date"
)

type VarSource string

const (
	// SourceUser: Ask UI / CLI.
	SourceUser VarSource = "user"
	// SourceStatic: Fixed literal.
	SourceStatic VarSource = "static"
)

// MessageBlock - One role-tagged chunk of text.
type MessageBlock struct {
	ID      MessageBlockID `json:"id"`
	Role    PromptRoleEnum `json:"role"`
	Content string         `json:"content"`
}

type PromptVariable struct {
	Name        string    `json:"name"`
	Type        VarType   `json:"type"`
	Required    bool      `json:"required"`
	Source      VarSource `json:"source"`
	Description string    `json:"description,omitempty"`

	// SourceStatic.
	StaticVal string `json:"staticVal,omitempty"`
	// VarEnum.
	EnumValues []string `json:"enumValues,omitempty"`

	// Optional default for the var.
	Default *string `json:"default,omitempty"`
}

type PromptTemplateRef struct {
	BundleID        bundleitemutils.BundleID    `json:"bundleID"`
	TemplateSlug    bundleitemutils.ItemSlug    `json:"templateSlug"`
	TemplateVersion bundleitemutils.ItemVersion `json:"templateVersion"`
}

type PromptTemplate struct {
	SchemaVersion string             `json:"schemaVersion"`
	Kind          PromptTemplateKind `json:"kind"`

	ID        bundleitemutils.ItemID   `json:"id"`
	Slug      bundleitemutils.ItemSlug `json:"slug"`
	IsEnabled bool                     `json:"isEnabled"`

	DisplayName string   `json:"displayName"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`

	// Ordered list of blocks that form the final prompt.
	Blocks []MessageBlock `json:"blocks"`
	// Declared placeholders.
	Variables []PromptVariable `json:"variables,omitempty"`

	// IsResolved is a backend-computed persisted property.
	//
	// True means all placeholders used by this template can be resolved using template-local variable information only,
	// with no external caller/user input required.
	//
	// Current resolution rules:
	//   - SourceStatic => resolved iff StaticVal is non-empty
	//   - SourceUser   => resolved iff Default is non-nil
	//
	// This field is computed and enforced by backend validation on put/read flows.
	IsResolved bool `json:"isResolved"`

	Version    bundleitemutils.ItemVersion `json:"version"`
	CreatedAt  time.Time                   `json:"createdAt"`
	ModifiedAt time.Time                   `json:"modifiedAt"`
	IsBuiltIn  bool                        `json:"isBuiltIn"`
}

// PromptBundle is a hard grouping & distribution unit.
type PromptBundle struct {
	SchemaVersion string                     `json:"schemaVersion"`
	ID            bundleitemutils.BundleID   `json:"id"`
	Slug          bundleitemutils.BundleSlug `json:"slug"`

	DisplayName string `json:"displayName,omitempty"`
	Description string `json:"description,omitempty"`

	IsEnabled     bool       `json:"isEnabled"`
	CreatedAt     time.Time  `json:"createdAt"`
	ModifiedAt    time.Time  `json:"modifiedAt"`
	IsBuiltIn     bool       `json:"isBuiltIn"`
	SoftDeletedAt *time.Time `json:"softDeletedAt,omitempty"`
}

type AllBundles struct {
	Bundles map[bundleitemutils.BundleID]PromptBundle `json:"bundles"`
}
