package spec

import (
	"encoding/json"
	"time"
)

// CatalogResourceKey is the natural identity of a source-local occurrence.
type CatalogResourceKey struct {
	SourceID           SourceID           `json:"sourceID"`
	Locator            SourceLocator      `json:"locator"`
	SubresourceLocator SubresourceLocator `json:"subresourceLocator,omitempty"`
}

// ArtifactSelector is a portable dependency declaration. It intentionally
// contains no root, record, source, path, secret, or runtime identity.
type ArtifactSelector struct {
	Kind              ArtifactKind      `json:"kind"`
	LogicalName       LogicalName       `json:"logicalName,omitempty"`
	VersionConstraint string            `json:"versionConstraint,omitempty"`
	Labels            map[string]string `json:"labels,omitempty"`
}

// AssetManifestEntry describes portable content that belongs to a canonical
// definition. Path is relative to the package or export root.
type AssetManifestEntry struct {
	Path      PortablePath `json:"path"`
	Digest    Digest       `json:"digest"`
	MediaType string       `json:"mediaType,omitempty"`
	SizeBytes int64        `json:"sizeBytes"`
}

// CanonicalDefinition is immutable and portable. It is persisted as a
// content-addressed JSON file or included in a portable export, never as the
// authoritative definition body in SQLite metadata.
//
// The Digest is calculated from the canonical portable fields. It excludes
// app-local observations such as source IDs, records, roots, timestamps, and
// diagnostics.
type CanonicalDefinition struct {
	Digest        Digest       `json:"digest"`
	Kind          ArtifactKind `json:"kind"`
	SchemaID      SchemaID     `json:"schemaID"`
	SchemaVersion string       `json:"schemaVersion"`

	LogicalName    LogicalName    `json:"logicalName"`
	LogicalVersion LogicalVersion `json:"logicalVersion,omitempty"`
	DisplayName    string         `json:"displayName,omitempty"`
	Description    string         `json:"description,omitempty"`

	Labels              map[string]string    `json:"labels,omitempty"`
	Extensions          json.RawMessage      `json:"extensions"`
	DefinitionJSON      json.RawMessage      `json:"definitionJSON"`
	DependencySelectors []ArtifactSelector   `json:"dependencySelectors,omitempty"`
	AssetManifest       []AssetManifestEntry `json:"assetManifest,omitempty"`
}

// ArtifactDefinitionFile is the portable JSON envelope for an exported,
// imported, captured, or content-addressed canonical definition. Frontends
// remain free to use their own native source formats.
type ArtifactDefinitionFile struct {
	Format     string              `json:"format"`
	Definition CanonicalDefinition `json:"definition"`
}

// PortablePackageDefinitionRef maps a definition digest to a JSON file inside
// a portable package directory or archive.
type PortablePackageDefinitionRef struct {
	Digest Digest       `json:"digest"`
	File   PortablePath `json:"file"`
}

// PortablePackageManifest is the shareable JSON package manifest. It contains
// no source IDs, root IDs, record IDs, local paths, app state, or secrets.
type PortablePackageManifest struct {
	Format      string         `json:"format"`
	Name        LogicalName    `json:"name"`
	Version     LogicalVersion `json:"version"`
	DisplayName string         `json:"displayName,omitempty"`
	Description string         `json:"description,omitempty"`

	Definitions []PortablePackageDefinitionRef `json:"definitions,omitempty"`
	Assets      []AssetManifestEntry           `json:"assets,omitempty"`
	Extensions  json.RawMessage                `json:"extensions"`
}

// ArtifactRoot is app-local metadata. It is never part of a portable package.
type ArtifactRoot struct {
	RootID        RootID   `json:"rootID"`
	Kind          RootKind `json:"kind"`
	DisplayName   string   `json:"displayName"`
	Description   string   `json:"description,omitempty"`
	Enabled       bool     `json:"enabled"`
	MountRevision uint64   `json:"mountRevision"`

	DataSchemaID SchemaID        `json:"dataSchemaID,omitempty"`
	Data         json.RawMessage `json:"data"`

	CreatedAt     time.Time  `json:"createdAt"`
	ModifiedAt    time.Time  `json:"modifiedAt"`
	SoftDeletedAt *time.Time `json:"softDeletedAt,omitempty"`
}

// ArtifactSource is an app-local source registration. In particular,
// fs-directory configuration may contain an absolute local path and must not
// be exported as portable artifact content.
type ArtifactSource struct {
	SourceID    SourceID   `json:"sourceID"`
	Kind        SourceKind `json:"kind"`
	DisplayName string     `json:"displayName"`
	Enabled     bool       `json:"enabled"`

	ConfigSchemaID SchemaID        `json:"configSchemaID"`
	Config         json.RawMessage `json:"config"`

	LastObservedGeneration *SourceGeneration `json:"lastObservedGeneration,omitempty"`
	LastScannedAt          *time.Time        `json:"lastScannedAt,omitempty"`
	ObservationRevision    uint64            `json:"observationRevision"`
	Diagnostics            []Diagnostic      `json:"diagnostics,omitempty"`

	CreatedAt  time.Time `json:"createdAt"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

// RootSourceAttachment is app-local metadata with the natural key
// RootID + SourceID.
type RootSourceAttachment struct {
	RootID   RootID         `json:"rootID"`
	SourceID SourceID       `json:"sourceID"`
	Role     AttachmentRole `json:"role"`
	Priority int            `json:"priority"`
	Enabled  bool           `json:"enabled"`

	DataSchemaID SchemaID        `json:"dataSchemaID,omitempty"`
	Data         json.RawMessage `json:"data"`

	CreatedAt  time.Time `json:"createdAt"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

// ArtifactPackage is app-local catalog metadata for one occurrence of a
// portable package manifest. The portable manifest itself is a JSON file and
// is represented by PortablePackageManifest.
type ArtifactPackage struct {
	SourceID        SourceID      `json:"sourceID"`
	ManifestLocator SourceLocator `json:"manifestLocator"`

	Name        LogicalName    `json:"name,omitempty"`
	Version     LogicalVersion `json:"version,omitempty"`
	DisplayName string         `json:"displayName,omitempty"`
	Description string         `json:"description,omitempty"`

	CurrentManifestDigest *Digest      `json:"currentManifestDigest,omitempty"`
	State                 CatalogState `json:"state"`
	Diagnostics           []Diagnostic `json:"diagnostics,omitempty"`

	FirstSeenAt time.Time `json:"firstSeenAt"`
	LastSeenAt  time.Time `json:"lastSeenAt"`
}

// CatalogResource is app-local metadata for one discovered source occurrence.
// Its natural key is SourceID + Locator + SubresourceLocator.
type CatalogResource struct {
	SourceID           SourceID           `json:"sourceID"`
	Locator            SourceLocator      `json:"locator"`
	SubresourceLocator SubresourceLocator `json:"subresourceLocator,omitempty"`

	PackageManifestLocator SourceLocator  `json:"packageManifestLocator,omitempty"`
	Kind                   ArtifactKind   `json:"kind,omitempty"`
	LogicalName            LogicalName    `json:"logicalName,omitempty"`
	LogicalVersion         LogicalVersion `json:"logicalVersion,omitempty"`

	CurrentDefinitionDigest *Digest      `json:"currentDefinitionDigest,omitempty"`
	SourceContentDigest     *Digest      `json:"sourceContentDigest,omitempty"`
	FrontendID              FrontendID   `json:"frontendID,omitempty"`
	State                   CatalogState `json:"state"`

	FirstSeenAt time.Time    `json:"firstSeenAt"`
	LastSeenAt  time.Time    `json:"lastSeenAt"`
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
}

// CatalogResourceRevision retains durable source-occurrence history. Its
// natural key is SourceID + Locator + SubresourceLocator + DefinitionDigest.
// It makes ListDefinitionHistory possible without assigning a revision ID.
type CatalogResourceRevision struct {
	SourceID           SourceID           `json:"sourceID"`
	Locator            SourceLocator      `json:"locator"`
	SubresourceLocator SubresourceLocator `json:"subresourceLocator,omitempty"`

	DefinitionDigest    Digest       `json:"definitionDigest"`
	SourceContentDigest Digest       `json:"sourceContentDigest"`
	Kind                ArtifactKind `json:"kind"`
	FrontendID          FrontendID   `json:"frontendID"`

	FirstSeenAt time.Time `json:"firstSeenAt"`
	LastSeenAt  time.Time `json:"lastSeenAt"`
}

// ArtifactRecord is the only generic app-side artifact item. Its identity and
// local data are app-local, while its resolved definition remains portable.
type ArtifactRecord struct {
	RecordID     RecordID      `json:"recordID"`
	RootID       RootID        `json:"rootID"`
	CollectionID *CollectionID `json:"collectionID,omitempty"`

	Kind    ArtifactKind  `json:"kind"`
	Name    RecordName    `json:"name"`
	Version RecordVersion `json:"version,omitempty"`

	SourceID           SourceID           `json:"sourceID"`
	Locator            SourceLocator      `json:"locator"`
	SubresourceLocator SubresourceLocator `json:"subresourceLocator,omitempty"`

	RecordMode                   RecordMode   `json:"recordMode"`
	TrackingMode                 TrackingMode `json:"trackingMode"`
	PinnedDefinitionDigest       *Digest      `json:"pinnedDefinitionDigest,omitempty"`
	LastResolvedDefinitionDigest *Digest      `json:"lastResolvedDefinitionDigest,omitempty"`

	Enabled      bool            `json:"enabled"`
	DataSchemaID SchemaID        `json:"dataSchemaID,omitempty"`
	Data         json.RawMessage `json:"data"`
	State        RecordState     `json:"state"`
	Diagnostics  []Diagnostic    `json:"diagnostics,omitempty"`

	CreatedAt  time.Time `json:"createdAt"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

// ArtifactCollection is an app-local grouping of records. It maps to current
// bundle semantics but is not a portable package.
type ArtifactCollection struct {
	CollectionID CollectionID   `json:"collectionID"`
	RootID       RootID         `json:"rootID"`
	Kind         CollectionKind `json:"kind"`
	Slug         CollectionSlug `json:"slug"`
	DisplayName  string         `json:"displayName"`
	Description  string         `json:"description,omitempty"`
	Enabled      bool           `json:"enabled"`

	DataSchemaID SchemaID        `json:"dataSchemaID,omitempty"`
	Data         json.RawMessage `json:"data"`

	CreatedAt     time.Time  `json:"createdAt"`
	ModifiedAt    time.Time  `json:"modifiedAt"`
	SoftDeletedAt *time.Time `json:"softDeletedAt,omitempty"`
}

// SourceCatalogVersion is the freshness identity of one source catalog.
// Generation identifies source content while ObservationRevision identifies
// the exact catalog observation published for that content.
type SourceCatalogVersion struct {
	Generation          SourceGeneration `json:"generation"`
	ObservationRevision uint64           `json:"observationRevision"`
}

// RootCatalogGeneration is app-local durable scan-publication metadata. Its
// natural key is RootID + Generation.
type RootCatalogGeneration struct {
	RootID         RootID                            `json:"rootID"`
	Generation     uint64                            `json:"generation"`
	RootRevision   uint64                            `json:"rootRevision"`
	SourceVersions map[SourceID]SourceCatalogVersion `json:"sourceVersions"`
	ScanPlanDigest Digest                            `json:"scanPlanDigest"`
	CatalogDigest  Digest                            `json:"catalogDigest"`
	CreatedAt      time.Time                         `json:"createdAt"`
	Diagnostics    []Diagnostic                      `json:"diagnostics,omitempty"`
}

// DependencyCandidateRef is the durable, app-local representation of a
// dependency candidate. Canonical definition bodies are deliberately excluded.
type DependencyCandidateRef struct {
	Resource         CatalogResourceKey `json:"resource"`
	DefinitionDigest Digest             `json:"definitionDigest"`
}

// ArtifactDependencySnapshot records one selector resolution against one
// published root catalog generation.
type ArtifactDependencySnapshot struct {
	RootID               RootID                    `json:"rootID"`
	RecordID             RecordID                  `json:"recordID"`
	CatalogGeneration    uint64                    `json:"catalogGeneration"`
	RootDefinitionDigest Digest                    `json:"rootDefinitionDigest"`
	DefinitionDigest     Digest                    `json:"definitionDigest"`
	SelectorIndex        int                       `json:"selectorIndex"`
	Selector             ArtifactSelector          `json:"selector"`
	State                DependencyResolutionState `json:"state"`
	Candidates           []DependencyCandidateRef  `json:"candidates"`
	Diagnostics          []Diagnostic              `json:"diagnostics,omitempty"`
	ModifiedAt           time.Time                 `json:"modifiedAt"`
}

// TransferProvenance is app-local audit metadata for a record created through
// import, capture, or fork. It does not alter portable definition content.
type TransferProvenance struct {
	ProvenanceID   ProvenanceID      `json:"provenanceID"`
	TargetRecordID RecordID          `json:"targetRecordID"`
	Operation      TransferOperation `json:"operation"`

	OriginRecordID         *RecordID           `json:"originRecordID,omitempty"`
	OriginResource         *CatalogResourceKey `json:"originResource,omitempty"`
	OriginDefinitionDigest Digest              `json:"originDefinitionDigest"`

	CreatedAt time.Time `json:"createdAt"`
}
