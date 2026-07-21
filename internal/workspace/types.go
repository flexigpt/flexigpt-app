package workspace

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
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

type ContextDefinition struct {
	Name      string `json:"name"`
	Role      string `json:"role"`
	MediaType string `json:"mediaType"`
	Content   string `json:"content"`
}

type SkillArgumentDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Default     string `json:"default,omitempty"`
}

type SkillDefinition struct {
	Name           string                    `json:"name"`
	DisplayName    string                    `json:"displayName,omitempty"`
	Description    string                    `json:"description"`
	Insert         string                    `json:"insert"`
	Arguments      []SkillArgumentDefinition `json:"arguments,omitempty"`
	Tags           []string                  `json:"tags,omitempty"`
	MarkdownBody   string                    `json:"markdownBody"`
	RawFrontmatter map[string]any            `json:"rawFrontmatter,omitempty"`
}

func decodeDefinitionBody[T any](
	raw json.RawMessage,
) (T, error) {
	var output T
	if err := json.Unmarshal(raw, &output); err != nil {
		return output, err
	}
	return output, nil
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
