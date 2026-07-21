package workspace

import (
	"context"
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/record"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

const (
	RootKind artifactstore.RootKind = "workspace.root"

	RolePrimary         artifactstore.AttachmentRole = "primary"
	RoleBuiltIn         artifactstore.AttachmentRole = "built-in"
	RoleLibrary         artifactstore.AttachmentRole = "library"
	RoleAttachedPackage artifactstore.AttachmentRole = "attached-package"
	RoleOverlay         artifactstore.AttachmentRole = "overlay"

	FilesystemSourceKind artifactstore.SourceKind = "fs-directory"

	DefinitionKind      artifactstore.ArtifactKind = "workspace.definition"
	DefinitionSchemaID  artifactstore.SchemaID     = "workspace.definition.v1"
	DefinitionDecoderID artifactstore.DecoderID    = "workspace.definition-json"

	CapabilityProfileVersion = "1"
	PrimaryPriority          = 1_000_000
)

var (
	ErrInvalidWorkspace           = errors.New("workspace: invalid")
	ErrNotWorkspace               = errors.New("workspace: root is not a Workspace")
	ErrPrimarySourceRequired      = errors.New("workspace: primary source is required")
	ErrPrimarySourceImmutable     = errors.New("workspace: primary source is immutable")
	ErrReferenceUnresolved        = errors.New("workspace: reference unresolved")
	ErrReferenceAmbiguous         = errors.New("workspace: reference ambiguous")
	ErrWorkspaceDefinitionInvalid = errors.New("workspace: definition invalid")
)

type Mode string

const (
	ModeEmpty      Mode = "empty"
	ModeFilesystem Mode = "filesystem"
)

type DiscoveryRoot struct {
	Root            artifactstore.Locator `json:"root"`
	Recursive       bool                  `json:"recursive"`
	IncludePatterns []string              `json:"includePatterns,omitempty"`
}

type DiscoveryPreferences struct {
	AdditionalLocators []artifactstore.Locator `json:"additionalLocators,omitempty"`
	AdditionalRoots    []DiscoveryRoot         `json:"additionalRoots,omitempty"`
	IncludeReadme      bool                    `json:"includeReadme,omitempty"`
}

type RootData struct {
	Mode                     Mode                   `json:"mode"`
	PrimarySourceID          artifactstore.SourceID `json:"primarySourceID,omitempty"`
	TrustReference           string                 `json:"trustReference,omitempty"`
	Discovery                DiscoveryPreferences   `json:"discovery"`
	CapabilityProfileVersion string                 `json:"capabilityProfileVersion"`
}

type AttachmentData struct {
	Recursive     *bool `json:"recursive,omitempty"`
	Authoritative *bool `json:"authoritative,omitempty"`
}

type Workspace struct {
	Root        catalog.Root         `json:"root"`
	Data        RootData             `json:"data"`
	Attachments []catalog.Attachment `json:"attachments"`
	Sources     []source.Source      `json:"sources"`
}

type EmptyWorkspaceRequest struct {
	DisplayName    string
	Description    string
	TrustReference string
	Discovery      DiscoveryPreferences
}

type FilesystemWorkspaceRequest struct {
	DisplayName     string
	Description     string
	PrimarySourceID artifactstore.SourceID
	TrustReference  string
	Discovery       DiscoveryPreferences
}

type UpdateRequest struct {
	RootID           artifactstore.RootID
	ExpectedRevision uint64
	DisplayName      string
	Description      string
	Enabled          bool
	TrustReference   string
	Discovery        DiscoveryPreferences
}

type AttachRequest struct {
	RootID               artifactstore.RootID
	ExpectedRootRevision uint64
	SourceID             artifactstore.SourceID
	Role                 artifactstore.AttachmentRole
	Priority             int
	Enabled              bool
	Data                 AttachmentData
}

type Descriptor struct {
	Kind     artifactstore.ArtifactKind
	SchemaID artifactstore.SchemaID
}

type Resource struct {
	Record         record.Record
	Definition     definition.Definition
	Occurrence     *catalog.Occurrence
	Source         source.Source
	CatalogCurrent bool
}

type CatalogView struct {
	Workspace         Workspace
	Catalog           catalog.Snapshot
	Resources         []Resource
	Unrecorded        []catalog.Occurrence
	UnresolvedRecords []record.Record
}

type Reference struct {
	RecordID *artifactstore.RecordID
	Selector *definition.Selector
}

type LoadPlanItem struct {
	Record     record.Record
	Definition definition.Definition
	Source     source.Source
}

type LoadPlan struct {
	RootID          artifactstore.RootID
	CatalogRevision uint64
	Items           []LoadPlanItem
	Diagnostics     []artifactstore.Diagnostic
}

type DefinitionDocument struct {
	Discovery DiscoveryPreferences `json:"discovery"`
}

type rootManager interface {
	CreateRoot(
		ctx context.Context,
		draft catalog.RootDraft,
		attachments []catalog.AttachmentDraft,
	) (catalog.Root, []catalog.Attachment, error)

	GetRoot(
		ctx context.Context,
		id artifactstore.RootID,
	) (catalog.Root, error)

	ListRoots(
		ctx context.Context,
		includeDeleted bool,
	) ([]catalog.Root, error)

	UpdateRoot(
		ctx context.Context,
		id artifactstore.RootID,
		update catalog.RootUpdate,
	) (catalog.Root, error)

	DeleteRoot(
		ctx context.Context,
		id artifactstore.RootID,
		expectedRevision uint64,
	) (catalog.Root, error)

	Attach(
		ctx context.Context,
		rootID artifactstore.RootID,
		expectedRootRevision uint64,
		draft catalog.AttachmentDraft,
	) (catalog.Root, catalog.Attachment, error)

	GetAttachment(
		ctx context.Context,
		rootID artifactstore.RootID,
		sourceID artifactstore.SourceID,
	) (catalog.Attachment, error)

	ListAttachments(
		ctx context.Context,
		rootID artifactstore.RootID,
	) ([]catalog.Attachment, error)

	Detach(
		ctx context.Context,
		rootID artifactstore.RootID,
		sourceID artifactstore.SourceID,
		expectedRootRevision uint64,
		expectedAttachmentRevision uint64,
	) (catalog.Root, error)

	Current(
		ctx context.Context,
		rootID artifactstore.RootID,
	) (catalog.Snapshot, error)
}

type sourceReader interface {
	Get(
		ctx context.Context,
		id artifactstore.SourceID,
	) (source.Source, error)
}

type recordReader interface {
	Get(
		ctx context.Context,
		id artifactstore.RecordID,
	) (record.Record, error)

	ListByRoot(
		ctx context.Context,
		rootID artifactstore.RootID,
	) ([]record.Record, error)
}

type definitionReader interface {
	Get(
		ctx context.Context,
		digest artifactstore.Digest,
	) (definition.Definition, error)
}
