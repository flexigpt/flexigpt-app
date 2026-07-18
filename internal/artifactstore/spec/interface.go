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

// SourceAssetRoot declaratively requests portable assets beneath a candidate's
// directory. The frontend declares scope only. Artifact Store retains source
// transport, traversal, limits, digesting, and persistence ownership.
type SourceAssetRoot struct {
	Root            SourceLocator
	PortablePrefix  PortablePath
	IncludePatterns []string
	Recursive       bool
}

// DecodedArtifact maps a portable definition to one source-local subresource.
// The frontend leaves Digest empty; Artifact Store canonicalizes and calculates
// it before storage.
type DecodedArtifact struct {
	SubresourceLocator SubresourceLocator
	Definition         CanonicalDefinition
	AssetRoots         []SourceAssetRoot
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

// RootAttachmentSourceHook is an optional RootKindHook companion for root
// kinds that need to validate an attachment against its registered source
// kind or normalized source configuration.
//
// Artifact Store owns source retrieval. Consumers receive the source only for
// typed attachment validation and must not perform source transport directly.
type RootAttachmentSourceHook interface {
	ValidateSourceAttachmentSource(
		ctx context.Context,
		root ArtifactRoot,
		attachment RootSourceAttachment,
		source ArtifactSource,
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

// DependencyResolver selects at most one candidate using root-kind-specific
// precedence. Artifact Store still discovers candidates, validates the result,
// builds the graph, detects cycles, and persists dependency snapshots.
type DependencyResolver interface {
	RootKind() RootKind
	ResolveDependency(
		ctx context.Context,
		root ArtifactRoot,
		attachments []RootSourceAttachment,
		selector ArtifactSelector,
		candidates []DependencyCandidate,
	) (*DependencyCandidate, []Diagnostic)
}

// DirectoryScanRoot selects regular files beneath a source-relative directory.
type DirectoryScanRoot struct {
	// IncludePatterns use path.Match syntax. A pattern without a slash is also
	// matched against each entry's basename, allowing recursive selection such
	// as "*.json" or "SKILL.md".
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
	MaxTotalBytes       int64               `json:"maxTotalBytes,omitempty"`
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

// SourceScanFilter optionally hides transport-owned or app-managed entries
// from candidate and asset discovery. A false result for a directory also
// prevents traversal beneath that directory.
type SourceScanFilter interface {
	IncludeSourceEntry(ctx context.Context, source ArtifactSource, entry SourceEntry) (bool, error)
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
