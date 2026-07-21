package engine

import (
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/record"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
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

// DiscoveryProfile defines discovery rules for one attachment class.
//
// Artifact adapters contribute their own conventions through this type.
type DiscoveryProfile struct {
	ExplicitLocators []artifactstore.Locator
	ReadmeLocator    artifactstore.Locator
	DirectoryRoots   []discovery.DirectoryRoot
}

type DiscoveryProfiles struct {
	Primary  DiscoveryProfile
	Attached DiscoveryProfile
}

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

type Resource struct {
	Record         record.Record
	Definition     definition.Definition
	Occurrence     *catalog.Occurrence
	Source         source.Source
	CatalogCurrent bool
}

type ResourceGroup struct {
	Kind       artifactstore.ArtifactKind `json:"kind"`
	Resources  []Resource                 `json:"resources"`
	Unrecorded []catalog.Occurrence       `json:"unrecorded,omitempty"`
}

type EmptyWorkspaceRequest struct {
	DisplayName    string               `json:"displayName"`
	Description    string               `json:"description,omitempty"`
	TrustReference string               `json:"trustReference,omitempty"`
	Discovery      DiscoveryPreferences `json:"discovery"`
}

type FilesystemWorkspaceRequest struct {
	DisplayName     string                 `json:"displayName"`
	Description     string                 `json:"description,omitempty"`
	PrimarySourceID artifactstore.SourceID `json:"primarySourceID"`
	TrustReference  string                 `json:"trustReference,omitempty"`
	Discovery       DiscoveryPreferences   `json:"discovery"`
}

type UpdateRequest struct {
	RootID           artifactstore.RootID `json:"rootID"`
	ExpectedRevision uint64               `json:"expectedRevision"`
	DisplayName      string               `json:"displayName"`
	Description      string               `json:"description,omitempty"`
	Enabled          bool                 `json:"enabled"`
	TrustReference   string               `json:"trustReference,omitempty"`
	Discovery        DiscoveryPreferences `json:"discovery"`
}

type AttachRequest struct {
	RootID               artifactstore.RootID         `json:"rootID"`
	ExpectedRootRevision uint64                       `json:"expectedRootRevision"`
	SourceID             artifactstore.SourceID       `json:"sourceID"`
	Role                 artifactstore.AttachmentRole `json:"role"`
	Priority             int                          `json:"priority"`
	Enabled              bool                         `json:"enabled"`
	Data                 AttachmentData               `json:"data"`
}

type CatalogView struct {
	Workspace         Workspace
	Catalog           catalog.Snapshot
	Resources         []Resource
	Unrecorded        []catalog.Occurrence
	UnresolvedRecords []record.Record
	Groups            []ResourceGroup
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

type DefinitionObservation struct {
	Preferences DiscoveryPreferences
	SourceID    artifactstore.SourceID
	Generation  string
}

type attachmentOperation struct {
	role                                 artifactstore.AttachmentRole
	canAttach                            bool
	isPrimary                            bool
	requiredSourceKind                   artifactstore.SourceKind
	defaultPriority                      int
	defaultAuthoritative                 bool
	includeReadmeWhenRequested           bool
	appliesWorkspaceDiscoveryPreferences bool
	allowsAttachmentDiscoveryOverrides   bool
}

type ArtifactSupport struct {
	Kind      artifactstore.ArtifactKind
	SchemaID  artifactstore.SchemaID
	DecoderID artifactstore.DecoderID
}
