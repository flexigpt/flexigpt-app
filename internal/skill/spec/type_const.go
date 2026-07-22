package spec

import (
	"errors"
	"time"

	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
)

const (
	SkillSchemaVersion = "2026-02-10"

	SkillBundlesMetaFileName      = "skills.bundles.json"
	SkillBuiltInOverlayDBFileName = "skillsbuiltin.overlay.sqlite" // optional: built-in overlay index

	// BaseSkillBundleID is the default writable bundle for user-created skill artifacts.
	BaseSkillBundleID          bundleitemutils.BundleID   = "019d3150-6a12-7a6b-a34e-d9032342bc31"
	BaseSkillBundleSlug        bundleitemutils.BundleSlug = "base"
	BaseSkillBundleDisplayName                            = "Base"
	BaseSkillBundleDescription                            = "Editable starter bundle for custom skills and prompt-like skill templates."
)

var (
	ErrSkillInvalidRequest = errors.New("invalid request")
	ErrSkillConflict       = errors.New("resource already exists")

	ErrSkillBuiltInReadOnly = errors.New("built-in resource is read-only")

	ErrSkillBundleNotFound = errors.New("bundle not found")
	ErrSkillBundleDisabled = errors.New("bundle is disabled")
	ErrSkillBundleDeleting = errors.New("bundle is being deleted")
	ErrSkillBundleNotEmpty = errors.New("bundle still contains skills")

	ErrSkillNotFound  = errors.New("skill not found")
	ErrSkillDisabled  = errors.New("skill is disabled")
	ErrSkillIsMissing = errors.New("skill is marked missing (cannot delete)")
)

// SkillType describes *where/how* the skill content is sourced.
// Only SkillTypeFS can be created by users (custom skills).
type SkillType string

const (
	SkillTypeFS         SkillType = "fs"         // filesystem skill package (custom)
	SkillTypeEmbeddedFS SkillType = "embeddedfs" // built-in embedded FS (read-only except enable/disable)
)

// SkillPresenceStatus tracks whether the skill was observed to exist at its location.
type SkillPresenceStatus string

const (
	SkillPresenceUnknown SkillPresenceStatus = "unknown" // never checked yet
	SkillPresencePresent SkillPresenceStatus = "present" // last check: exists
	SkillPresenceMissing SkillPresenceStatus = "missing" // last check: not found at location
	SkillPresenceError   SkillPresenceStatus = "error"   // check attempt failed (IO error, perms, etc.)
)

// SkillPresence contains the minimal history needed for storage consistency decisions.
type SkillPresence struct {
	Status SkillPresenceStatus `json:"status"`

	// LastCheckedAt is when we last attempted to validate presence.
	LastCheckedAt *time.Time `json:"lastCheckedAt,omitempty"`

	// LastSeenAt is when the location was last confirmed present.
	LastSeenAt *time.Time `json:"lastSeenAt,omitempty"`

	// MissingSince is set when we transition into "missing".
	MissingSince *time.Time `json:"missingSince,omitempty"`

	// LastCheckError is only meaningful when Status == "error".
	LastCheckError string `json:"lastCheckError,omitempty"`
}

type (
	SkillBundleID   = bundleitemutils.BundleID
	SkillBundleSlug = bundleitemutils.BundleSlug

	// SkillSlug is the user-facing identifier (no versioning).
	SkillSlug = bundleitemutils.ItemSlug

	// SkillID is a stable internal identifier (UUID-v7 recommended).
	SkillID = bundleitemutils.ItemID
)

// SkillRef is the store identity used by the UI for selection/persistence.
// It intentionally avoids runtime identity (type/name/location).
type SkillRef struct {
	BundleID  SkillBundleID `json:"bundleID"`
	SkillSlug SkillSlug     `json:"skillSlug"`
	SkillID   SkillID       `json:"skillID"`
}

type SkillSelection struct {
	SkillRef          SkillRef `json:"skillRef"`
	PreLoadAsActive   bool     `json:"preLoadAsActive"`
	UseAsInstructions bool     `json:"useAsInstructions"`
}

type (
	SkillInsert       = agentskillsSpec.SkillInsert
	SkillArgument     = agentskillsSpec.SkillArgument
	SkillResourceInfo = agentskillsSpec.SkillResourceInfo
)

const (
	SkillInsertInstructions = agentskillsSpec.SkillInsertInstructions
	SkillInsertUserMessage  = agentskillsSpec.SkillInsertUserMessage

	MaxSkillResourceLocations = agentskillsSpec.MaxSkillResourceLocations
)

// Skill is the storage + management record.
// It intentionally includes fields that are useful for JSON persistence, indexing, and listing/paging.
type Skill struct {
	SchemaVersion string    `json:"schemaVersion"`
	ID            SkillID   `json:"id"`   // UUID-v7
	Slug          SkillSlug `json:"slug"` // unique slug identifier

	Type     SkillType `json:"type"`
	Location string    `json:"location"` // opaque provider/app location; user-created fs skills usually use an absolute base dir.
	Name     string    `json:"name"`     // name of the skill inside SKILL.md.

	DisplayName string `json:"displayName,omitempty"`
	Description string `json:"description,omitempty"`

	// Tags are application-managed tags used for app filtering and
	// organization. Source tags from SKILL.md are exposed through runtime
	// projections rather than overwriting this field during indexing.
	Tags []string `json:"tags,omitempty"`

	// Parsed from SKILL.md frontmatter field "insert".
	// Missing/empty defaults to "instructions".
	Insert SkillInsert `json:"insert,omitempty"`

	// Parsed from SKILL.md frontmatter field "arguments".
	Arguments []SkillArgument `json:"arguments,omitempty"`

	// Runtime/provider-indexed resource metadata. Locations are provider-defined
	// values intended for use with skills-readresource.resourceLocation.
	Resources SkillResourceInfo `json:"resources"`

	// Full parsed YAML frontmatter. FlexiGPT only gives semantics to name,
	// description, insert, and arguments; other fields are preserved for callers.
	RawFrontmatter map[string]any `json:"rawFrontmatter,omitempty"`

	// Runtime/provider indexing metadata.
	RuntimeWarnings []string `json:"runtimeWarnings,omitempty"`
	Digest          string   `json:"digest,omitempty"`

	Presence *SkillPresence `json:"presence,omitempty"`

	IsEnabled bool `json:"isEnabled"`
	IsBuiltIn bool `json:"isBuiltIn"`

	CreatedAt  time.Time `json:"createdAt"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

type SkillBundle struct {
	SchemaVersion string          `json:"schemaVersion"`
	ID            SkillBundleID   `json:"id"` // UUID-v7
	Slug          SkillBundleSlug `json:"slug"`

	DisplayName string `json:"displayName,omitempty"`
	Description string `json:"description,omitempty"`

	IsEnabled  bool      `json:"isEnabled"`
	IsBuiltIn  bool      `json:"isBuiltIn"`
	CreatedAt  time.Time `json:"createdAt"`
	ModifiedAt time.Time `json:"modifiedAt"`

	SoftDeletedAt *time.Time `json:"softDeletedAt,omitempty"`
}

type AllSkillBundles struct {
	Bundles map[bundleitemutils.BundleID]SkillBundle `json:"bundles"`
}
