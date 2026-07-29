package engine

import (
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/workspace/engine/portable"
)

var (
	ErrInvalidWorkspace           = errors.New("workspace: invalid")
	ErrNotWorkspace               = errors.New("workspace: collection is not a Workspace")
	ErrPrimarySourceRequired      = errors.New("workspace: primary source is required")
	ErrPrimarySourceImmutable     = errors.New("workspace: primary source is immutable")
	ErrReferenceUnresolved        = errors.New("workspace: reference unresolved")
	ErrReferenceAmbiguous         = errors.New("workspace: reference ambiguous")
	ErrWorkspaceDefinitionInvalid = errors.New("workspace: descriptor invalid")
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
	ExplicitLocators []basespec.Locator
	ReadmeLocator    basespec.Locator
	DirectoryRoots   []discovery.DirectoryRoot
}

type DiscoveryProfiles struct {
	Primary  DiscoveryProfile
	Attached DiscoveryProfile
}

type DiscoveryRoot struct {
	Root            basespec.Locator `json:"root"`
	Recursive       bool             `json:"recursive"`
	IncludePatterns []string         `json:"includePatterns,omitempty"`
}

type DiscoveryPreferences struct {
	AdditionalLocators []basespec.Locator `json:"additionalLocators,omitempty"`
	AdditionalRoots    []DiscoveryRoot    `json:"additionalRoots,omitempty"`
	IncludeReadme      bool               `json:"includeReadme,omitempty"`
}

// CollectionData contains local Workspace policy only. The Workspace mode and
// primary Source are derived from the current collection attachments.
//
// DiscoveryPolicyRevision is deliberately local. It is not portable descriptor
// content and it gives the running Workspace implementation an explicit way to
// invalidate a catalog after planner behavior changes.
type CollectionData struct {
	DiscoveryPolicyRevision string               `json:"discoveryPolicyRevision"`
	Discovery               DiscoveryPreferences `json:"discovery"`
}

type AttachmentData struct {
	Recursive     *bool `json:"recursive,omitempty"`
	Authoritative *bool `json:"authoritative,omitempty"`
}

type ArtifactData struct {
	RuntimeDisabled bool `json:"runtimeDisabled,omitempty"`
}

type WorkspaceRef = collection.CollectionRef

// Workspace is an internal privileged aggregate. API packages project it into
// explicit view models and must not serialize collection local data, attachment
// local data, or Source configuration.
type Workspace struct {
	Collection      collection.Collection   `json:"-"`
	Data            CollectionData          `json:"-"`
	Mode            Mode                    `json:"-"`
	PrimarySourceID basespec.SourceID       `json:"-"`
	Attachments     []collection.Attachment `json:"-"`
	Sources         []source.Summary        `json:"-"`
}

type Resource struct {
	Artifact        artifact.Artifact       `json:"-"`
	Definition      definition.Definition   `json:"-"`
	Occurrence      *catalog.Occurrence     `json:"-"`
	Source          source.Summary          `json:"-"`
	CatalogCurrent  bool                    `json:"-"`
	ProjectionValid bool                    `json:"-"`
	Diagnostics     []diagnostic.Diagnostic `json:"-"`
}

type ResourceGroup struct {
	Kind       basespec.ArtifactKind `json:"-"`
	Resources  []Resource            `json:"-"`
	Unrecorded []catalog.Occurrence  `json:"-"`
}

type EmptyWorkspaceRequest struct {
	RootID      basespec.RootID      `json:"rootID"`
	DisplayName string               `json:"displayName"`
	Description string               `json:"description,omitempty"`
	Discovery   DiscoveryPreferences `json:"discovery"`
}

type FilesystemWorkspaceRequest struct {
	RootID          basespec.RootID      `json:"rootID"`
	DisplayName     string               `json:"displayName"`
	Description     string               `json:"description,omitempty"`
	PrimarySourceID basespec.SourceID    `json:"primarySourceID"`
	Discovery       DiscoveryPreferences `json:"discovery"`
}

type UpdateRequest struct {
	Workspace        WorkspaceRef         `json:"workspace"`
	ExpectedRevision uint64               `json:"expectedRevision"`
	DisplayName      string               `json:"displayName"`
	Description      string               `json:"description,omitempty"`
	Enabled          bool                 `json:"enabled"`
	Discovery        DiscoveryPreferences `json:"discovery"`
}

type AttachRequest struct {
	Workspace                  WorkspaceRef            `json:"workspace"`
	ExpectedCollectionRevision uint64                  `json:"expectedCollectionRevision"`
	SourceID                   basespec.SourceID       `json:"sourceID"`
	Role                       basespec.AttachmentRole `json:"role"`
	Enabled                    bool                    `json:"enabled"`
	Data                       AttachmentData          `json:"data"`
}

type UpdateAttachmentRequest struct {
	Workspace                  WorkspaceRef
	SourceID                   basespec.SourceID
	ExpectedCollectionRevision uint64
	ExpectedAttachmentRevision uint64
	Role                       basespec.AttachmentRole
	Enabled                    bool
	Data                       AttachmentData
}

type ReplacePrimaryRequest struct {
	Workspace                  WorkspaceRef
	ExpectedCollectionRevision uint64
	PreviousSourceID           basespec.SourceID
	PreviousAttachmentRevision uint64
	SourceID                   basespec.SourceID
}

type SetPrimaryRequest struct {
	Workspace                  WorkspaceRef
	ExpectedCollectionRevision uint64
	PreviousSourceID           basespec.SourceID
	PreviousAttachmentRevision uint64
	SourceID                   basespec.SourceID
	Clear                      bool
}

type CatalogView struct {
	Workspace            Workspace               `json:"-"`
	Catalog              catalog.Snapshot        `json:"-"`
	Resources            []Resource              `json:"-"`
	Unrecorded           []catalog.Occurrence    `json:"-"`
	UnresolvedArtifacts  []artifact.Artifact     `json:"-"`
	Groups               []ResourceGroup         `json:"-"`
	CatalogCurrent       bool                    `json:"-"`
	FreshnessDiagnostics []diagnostic.Diagnostic `json:"-"`
}

type Reference struct {
	Artifact *artifact.ArtifactRef `json:"-"`
	Selector *definition.Selector  `json:"-"`
}

// LoadPlanItem contains privileged materialized source state. It must be
// projected into an explicit adapter response before crossing an API boundary.
type LoadPlanItem struct {
	Artifact                   artifact.Artifact     `json:"-"`
	Definition                 definition.Definition `json:"-"`
	Source                     source.Summary        `json:"-"`
	CatalogCurrent             bool                  `json:"-"`
	OccurrenceDefinitionDigest cryptoutil.Digest     `json:"-"`
	SourceContentDigest        cryptoutil.Digest     `json:"-"`
	SourceGeneration           string                `json:"-"`
}

type LoadPlan struct {
	Workspace       WorkspaceRef            `json:"-"`
	CatalogRevision uint64                  `json:"-"`
	Items           []LoadPlanItem          `json:"-"`
	Diagnostics     []diagnostic.Diagnostic `json:"-"`
}

// Descriptor is the portable Collection Definition stored at
// .flexigpt/workspace.json. Its domain body contains Workspace discovery
// policy while its generic Members field contains relative or external member
// references.
type Descriptor = portable.CollectionDefinition

type DescriptorObservation struct {
	Preferences            DiscoveryPreferences
	SourceID               basespec.SourceID
	Generation             string
	ExpectedContentDigests map[basespec.Locator]cryptoutil.Digest
}

type attachmentOperation struct {
	role                                 basespec.AttachmentRole
	canAttach                            bool
	isPrimary                            bool
	requiredSourceKind                   basespec.SourceKind
	defaultAuthoritative                 bool
	includeReadmeWhenRequested           bool
	appliesWorkspaceDiscoveryPreferences bool
	allowsAttachmentDiscoveryOverrides   bool
}

type DefinitionValidator func(definition.Definition) error

type ArtifactSupport struct {
	Kind      basespec.ArtifactKind
	SchemaID  basespec.SchemaID
	DecoderID basespec.DecoderID
	Validator DefinitionValidator
}
