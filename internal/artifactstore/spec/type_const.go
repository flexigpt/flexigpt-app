// Package spec defines Artifact Store's portable and app-local data contracts.
package spec

import "errors"

const (
	// ArtifactDefinitionFileFormatV1 is the JSON-file envelope used for portable
	// definition export, import, capture, and content-addressed definition storage.
	// It is not required for a frontend's native source format.
	ArtifactDefinitionFileFormatV1 = "artifact-definition/v1"

	// PortablePackageManifestFormatV1 is the JSON-file manifest used for a
	// portable Artifact Store package. It is an interchange format, not a
	// mandatory source envelope for frontends.
	PortablePackageManifestFormatV1 = "artifact-package/v1"
)

const (
	MaxKindBytes                  = 128
	MaxSchemaIDBytes              = 256
	MaxDisplayNameBytes           = 256
	MaxDescriptionBytes           = 16 * 1024
	MaxLogicalNameBytes           = 256
	MaxVersionBytes               = 256
	MaxSourceGenerationBytes      = 512
	MaxSourceLocatorBytes         = 4096
	MaxFilesystemPathBytes        = 16 * 1024
	MaxDiagnosticCodeBytes        = 128
	MaxDiagnosticMessageBytes     = 4096
	MaxLabelsPerDefinition        = 64
	MaxLabelValueBytes            = 256
	MaxAssetsPerDefinition        = 4096
	MaxSelectorsPerDefinition     = 1024
	MaxDiagnosticsPerEntity       = 128
	MaxConfigJSONBytes            = 1 << 20
	MaxLocalDataJSONBytes         = 1 << 20
	MaxDefinitionJSONBytes        = 4 << 20
	MaxExtensionsJSONBytes        = 1 << 20
	MaxPortablePackageDefinitions = 4096
	MaxAttachmentPriority         = 1_000_000
	MaxSlugRunes                  = 64
	MaxSourcePlansPerScan         = 1024
	MaxExplicitLocatorsPerSource  = 10_000
	MaxDirectoryRootsPerSource    = 1024
	MaxFrontendIDsPerSource       = 256
	MaxIncludePatternsPerRoot     = 256
	MaxAssetRootsPerDefinition    = 16
	DefaultMaxScanCandidates      = 10_000
	DefaultMaxScanEntries         = 100_000
	DefaultMaxTraversalDepth      = 64
	DefaultMaxScanAssetFiles      = 10_000
	MaxScanTotalBytes             = int64(512 << 20)
	DefaultMaxMaterializedEntries = 100_000
	DefaultMaxMaterializedFiles   = 50_000
	DefaultMaxMaterializedBytes   = int64(512 << 20)
	MaxTransferPayloadBytes       = int64(512 << 20)
	MaxTransferReceiptBytes       = 2 << 20
	MaxTransferMaterializedFiles  = MaxPortablePackageDefinitions + MaxAssetsPerDefinition
	MaxObservationRevision        = uint64(1<<63 - 1)
)

const (
	// ManagedArtifactDataDirectoryName and ManagedArtifactTrashDirectoryName
	// are reserved in app-managed filesystem sources. Source drivers must not
	// expose their contents as scan candidates.
	ManagedArtifactDataDirectoryName        = ".flexigpt-artifacts"
	ManagedArtifactDefinitionsDirectoryName = ".flexigpt-artifacts/definitions"
	ManagedArtifactTrashDirectoryName       = ".artifactstore-trash"
)

var ErrInvalid = errors.New("artifactstore: invalid")

type (
	RootID       string
	SourceID     string
	CollectionID string
	RecordID     string
	ProvenanceID string

	RootKind       string
	SourceKind     string
	CollectionKind string
	ArtifactKind   string
	FrontendID     string
	SchemaID       string

	AttachmentRole     string
	RecordName         string
	RecordVersion      string
	CollectionSlug     string
	LogicalName        string
	LogicalVersion     string
	SourceLocator      string
	SubresourceLocator string
	PortablePath       string
	Digest             string
	SourceGeneration   string
)

const (
	FSDirectoryConfigSchemaID         SchemaID = "artifactstore.fs-directory.config.v1"
	EmbeddedFSDirectoryConfigSchemaID SchemaID = "artifactstore.embedded-fs-directory.config.v1"
	MemoryDirectoryConfigSchemaID     SchemaID = "artifactstore.memory-directory.config.v1"

	PortableDefinitionFrontendID FrontendID = "artifactstore.portable-definition"
)

type CatalogState string

const (
	CatalogStateValid   CatalogState = "valid"
	CatalogStateInvalid CatalogState = "invalid"
	CatalogStateMissing CatalogState = "missing"
)

type RecordMode string

const (
	RecordModeLinked          RecordMode = "linked"
	RecordModeCaptured        RecordMode = "captured"
	RecordModeForked          RecordMode = "forked"
	RecordModeAppLocal        RecordMode = "app-local"
	RecordModeEmbeddedOverlay RecordMode = "embedded-overlay"
)

type TrackingMode string

const (
	TrackingModeFollowSource  TrackingMode = "follow-source"
	TrackingModePinDigest     TrackingMode = "pin-digest"
	TrackingModeManualRefresh TrackingMode = "manual-refresh"
)

type RecordState string

const (
	RecordStateAvailable    RecordState = "available"
	RecordStateStale        RecordState = "stale"
	RecordStateMissing      RecordState = "missing"
	RecordStateInvalid      RecordState = "invalid"
	RecordStateIncompatible RecordState = "incompatible"
)

type DependencyResolutionState string

const (
	DependencyResolutionStateResolved  DependencyResolutionState = "resolved"
	DependencyResolutionStateMissing   DependencyResolutionState = "missing"
	DependencyResolutionStateAmbiguous DependencyResolutionState = "ambiguous"
)

type DiagnosticSeverity string

const (
	DiagnosticSeverityError   DiagnosticSeverity = "error"
	DiagnosticSeverityWarning DiagnosticSeverity = "warning"
	DiagnosticSeverityInfo    DiagnosticSeverity = "info"
)

type TransferOperation string

const (
	TransferOperationImport  TransferOperation = "import"
	TransferOperationCapture TransferOperation = "capture"
	TransferOperationFork    TransferOperation = "fork"
)

const (
	SourceKindFSDirectory         SourceKind = "fs-directory"
	SourceKindEmbeddedFSDirectory SourceKind = "embedded-fs-directory"
	SourceKindMemoryDirectory     SourceKind = "memory-directory"
)
