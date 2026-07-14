package spec

import (
	"time"
)

// SourceCatalogPublication is the driver-independent result of observing one
// source. A repository publishes it atomically to app-local metadata.
type SourceCatalogPublication struct {
	SourceID           SourceID
	ObservedGeneration SourceGeneration
	ObservedAt         time.Time
	Diagnostics        []Diagnostic
	Resources          []CatalogResource
	Revisions          []CatalogResourceRevision
	Authoritative      bool
}

// RootCatalogPublication records a durable root-level catalog generation after
// one or more source publications have completed.
type RootCatalogPublication struct {
	RootID            RootID
	SourceGenerations map[SourceID]SourceGeneration
	ScanPlanDigest    Digest
	CatalogDigest     Digest
	CreatedAt         time.Time
	Diagnostics       []Diagnostic
}

// RecordUpdate replaces mutable app-local record fields. ClearCollection is
// explicit because a nil CollectionID otherwise cannot distinguish clear from
// leave-unchanged semantics.
type RecordUpdate struct {
	CollectionID    *CollectionID
	ClearCollection bool
	Enabled         *bool
	DataSchemaID    *SchemaID
	Data            []byte
}

// ImportDefinitionRequest imports a portable definition through the configured
// portable-content repository and materializes a catalog occurrence at the
// caller-supplied app-local source location.
type ImportDefinitionRequest struct {
	File                   ArtifactDefinitionFile
	Record                 ArtifactRecordDraft
	FrontendID             FrontendID
	PackageManifestLocator SourceLocator
}

// ExportedRecord is the portable result of exporting an app-local record. The
// caller chooses where a MapStore-backed package repository persists it.
type ExportedRecord struct {
	Record     ArtifactRecord
	Definition ArtifactDefinitionFile
	Closure    ExportClosure
}

// DependencyCandidate is one catalog definition matching a portable selector.
type DependencyCandidate struct {
	Resource   CatalogResource
	Definition CanonicalDefinition
}

// DependencyExplanation preserves all candidates because Artifact Store does
// not choose feature-specific precedence rules.
type DependencyExplanation struct {
	Selector    ArtifactSelector
	Candidates  []DependencyCandidate
	Diagnostics []Diagnostic
}

// DependencyGraph is a diagnostic graph rooted at one record definition.
type DependencyGraph struct {
	RootRecordID RecordID
	Nodes        map[Digest]CanonicalDefinition
	Edges        map[Digest][]DependencyExplanation
	Diagnostics  []Diagnostic
}
