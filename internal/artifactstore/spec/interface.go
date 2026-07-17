package spec

import (
	"context"
	"encoding/json"
	"io"
	"time"
)

// SourceEntry is a driver-provided, source-relative directory entry.
type SourceEntry struct {
	Locator     SourceLocator
	Name        string
	Mode        uint32
	SizeBytes   int64
	ModifiedAt  time.Time
	IsDirectory bool
	IsRegular   bool
	IsSymlink   bool
}

// Recognition ranks a frontend's confidence that it owns a candidate.
type Recognition int

const (
	RecognitionNone Recognition = iota
	RecognitionPossible
	RecognitionPreferred
)

// ArtifactCandidate is bounded source content supplied to an artifact frontend.
type ArtifactCandidate struct {
	Source                 ArtifactSource
	Locator                SourceLocator
	SourceContentDigest    Digest
	Content                []byte
	PackageManifestLocator SourceLocator
}

// DecodedArtifact maps a portable definition to one source-local subresource.
// The frontend leaves Digest empty; Artifact Store canonicalizes and calculates
// it before storage.
type DecodedArtifact struct {
	SubresourceLocator SubresourceLocator
	Definition         CanonicalDefinition
}

// ExportClosure describes portable definition and asset content required to
// export a definition without relying on app-local source registrations.
type ExportClosure struct {
	DefinitionDigests []Digest
	Assets            []AssetManifestEntry
}

// ArtifactRecordDraft is the caller-provided portion of a new app-local record.
// Artifact Store assigns RecordID, state, resolved digest, and timestamps.
type ArtifactRecordDraft struct {
	RootID       RootID
	CollectionID *CollectionID

	Kind    ArtifactKind
	Name    RecordName
	Version RecordVersion

	SourceID           SourceID
	Locator            SourceLocator
	SubresourceLocator SubresourceLocator

	RecordMode             RecordMode
	TrackingMode           TrackingMode
	PinnedDefinitionDigest *Digest
	Enabled                bool
	DataSchemaID           SchemaID
	Data                   json.RawMessage
}

// ArtifactFrontend owns domain-specific recognition, decoding, and validation.
// It must not implement transport behavior or depend on app-local source paths.
type ArtifactFrontend interface {
	ID() FrontendID
	Recognizes(ctx context.Context, candidate ArtifactCandidate) Recognition
	Decode(ctx context.Context, candidate ArtifactCandidate) ([]DecodedArtifact, []Diagnostic)
	ValidateStructure(ctx context.Context, definition CanonicalDefinition) []Diagnostic
	ValidateSemantic(ctx context.Context, definition CanonicalDefinition) []Diagnostic
	ExtractDependencies(
		ctx context.Context,
		definition CanonicalDefinition,
	) ([]ArtifactSelector, []Diagnostic)
	ValidateRecordData(
		ctx context.Context,
		definition CanonicalDefinition,
		record ArtifactRecordDraft,
	) []Diagnostic
	DescribeExportClosure(ctx context.Context, definition CanonicalDefinition) (ExportClosure, []Diagnostic)
}

// ArtifactVersionMatcher owns version-constraint syntax for one artifact kind.
// Artifact Store deliberately does not interpret semver, date versions, tags,
// ranges, or domain-specific version aliases.
type ArtifactVersionMatcher interface {
	Kind() ArtifactKind
	MatchesVersionConstraint(
		ctx context.Context,
		constraint string,
		version LogicalVersion,
	) (bool, error)
}

// FrontendVersionMatcher lets a frontend own version syntax for definitions it
// emitted. It takes precedence over a kind-wide registered matcher.
type FrontendVersionMatcher interface {
	MatchesVersionConstraint(
		ctx context.Context,
		constraint string,
		version LogicalVersion,
	) (bool, error)
}

// RootKindHook validates app-local typed root data and source attachment rules.
type RootKindHook interface {
	Kind() RootKind
	ValidateRootData(ctx context.Context, root ArtifactRoot) []Diagnostic
	ValidateSourceAttachment(
		ctx context.Context,
		root ArtifactRoot,
		attachment RootSourceAttachment,
	) []Diagnostic
}

// RootAttachmentSetHook is an optional RootKindHook companion for root kinds
// whose typed data constrains the complete source-attachment set. Artifact
// Store invokes it against the proposed set before attachment mutations and
// root-data updates are published.
type RootAttachmentSetHook interface {
	ValidateSourceAttachments(
		ctx context.Context,
		root ArtifactRoot,
		attachments []RootSourceAttachment,
	) []Diagnostic
}

// CollectionKindHook validates app-local typed collection data and record
// placement without interpreting the record's runtime behavior.
type CollectionKindHook interface {
	Kind() CollectionKind
	ValidateCollectionData(ctx context.Context, collection ArtifactCollection) []Diagnostic
	ValidateRecordPlacement(
		ctx context.Context,
		collection ArtifactCollection,
		record ArtifactRecord,
	) []Diagnostic
}

// RecordDerivation is a policy-owned set of local values for a record created
// by synchronization.
type RecordDerivation struct {
	CollectionID *CollectionID
	Name         RecordName
	Version      RecordVersion
	Enabled      bool
	DataSchemaID SchemaID
	Data         json.RawMessage
}

// RecordSyncPolicy chooses whether a catalog resource should create a local
// linked record and, if so, its app-local placement and data.
type RecordSyncPolicy interface {
	DeriveRecord(
		ctx context.Context,
		root ArtifactRoot,
		resource CatalogResource,
		definition CanonicalDefinition,
	) (RecordDerivation, bool, []Diagnostic)
}

// DirectoryScanRoot selects regular files beneath a source-relative directory.
type DirectoryScanRoot struct {
	Root            SourceLocator `json:"root"`
	IncludePatterns []string      `json:"includePatterns"`
	Recursive       bool          `json:"recursive"`
}

// SourceScanPlan controls candidate discovery for one attached source.
// An authoritative plan marks previously known resources inside its declared
// locator, directory, and allowed-frontend scope as missing when they are no
// longer observed. Resources outside that scope remain unchanged.
type SourceScanPlan struct {
	SourceID            SourceID            `json:"sourceID"`
	ExplicitLocators    []SourceLocator     `json:"explicitLocators,omitempty"`
	DirectoryRoots      []DirectoryScanRoot `json:"directoryRoots,omitempty"`
	AllowedFrontendIDs  []FrontendID        `json:"allowedFrontendIDs,omitempty"`
	MaxFileBytes        int64               `json:"maxFileBytes,omitempty"`
	MaxCandidates       int                 `json:"maxCandidates,omitempty"`
	MaxTraversalEntries int                 `json:"maxTraversalEntries,omitempty"`
	MaxTraversalDepth   int                 `json:"maxTraversalDepth,omitempty"`
	Authoritative       bool                `json:"authoritative,omitempty"`
}

// ScanPlan is supplied by a consumer feature. With no source plans, ScanRoot
// performs an authoritative recursive scan of every enabled attachment.
type ScanPlan struct {
	SourcePlans []SourceScanPlan `json:"sourcePlans"`
}

// SourceScanResult summarizes one source observation in a root scan.
type SourceScanResult struct {
	SourceID         SourceID
	Generation       SourceGeneration
	Candidates       int
	ValidResources   int
	InvalidResources int
	Diagnostics      []Diagnostic
}

// ScanResult describes a published root catalog generation.
type ScanResult struct {
	RootID      RootID
	Generation  RootCatalogGeneration
	Sources     []SourceScanResult
	Diagnostics []Diagnostic
}

// RecordSyncResult describes record synchronization after catalog publication.
type RecordSyncResult struct {
	RootID      RootID
	Created     []RecordID
	Updated     []RecordID
	Diagnostics []Diagnostic
}

// RootDraft is the caller-provided portion of a newly created root.
type RootDraft struct {
	Kind         RootKind
	DisplayName  string
	Description  string
	Enabled      bool
	DataSchemaID SchemaID
	Data         json.RawMessage
}

// SourceDraft is the caller-provided portion of a newly created source.
type SourceDraft struct {
	Kind           SourceKind
	DisplayName    string
	Enabled        bool
	ConfigSchemaID SchemaID
	Config         json.RawMessage
}

// CollectionDraft is the caller-provided portion of a newly created collection.
type CollectionDraft struct {
	RootID       RootID
	Kind         CollectionKind
	Slug         CollectionSlug
	DisplayName  string
	Description  string
	Enabled      bool
	DataSchemaID SchemaID
	Data         json.RawMessage
}

// WalkFunc receives one source-relative entry during SourceDriver.Walk.
type WalkFunc func(context.Context, SourceEntry) error

// SourceConfigNormalizer optionally canonicalizes source-kind-specific
// configuration before it is validated and persisted. Normalization must be
// deterministic and must not access source content.
type SourceConfigNormalizer interface {
	NormalizeConfig(
		ctx context.Context,
		config json.RawMessage,
	) (json.RawMessage, []Diagnostic)
}

// SourceDriver owns source transport, traversal safety, and generation
// calculation for one Artifact Store-owned source kind.
type SourceDriver interface {
	Kind() SourceKind

	ValidateConfig(ctx context.Context, config json.RawMessage) []Diagnostic
	Snapshot(ctx context.Context, source ArtifactSource) (SourceGeneration, error)
	Open(ctx context.Context, source ArtifactSource, locator SourceLocator) (io.ReadCloser, error)
	Stat(ctx context.Context, source ArtifactSource, locator SourceLocator) (SourceEntry, error)
	ReadDir(ctx context.Context, source ArtifactSource, locator SourceLocator) ([]SourceEntry, error)
	Walk(ctx context.Context, source ArtifactSource, root SourceLocator, walk WalkFunc) error
}
