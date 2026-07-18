package spec

import (
	"encoding/json"
	"time"
)

// SourceCatalogPublication is the driver-independent result of observing one
// source. A repository publishes it atomically to app-local metadata.
type SourceCatalogPublication struct {
	SourceID                    SourceID
	ExpectedObservationRevision uint64
	AdvanceObservationRevision  bool
	ObservedGeneration          SourceGeneration
	ObservedAt                  time.Time
	Diagnostics                 []Diagnostic
	Resources                   []CatalogResource
	Revisions                   []CatalogResourceRevision
}

// RootCatalogPublication records a durable root-level catalog generation after
// one or more source publications have completed.
type RootCatalogPublication struct {
	RootID         RootID
	RootRevision   uint64
	SourceVersions map[SourceID]SourceCatalogVersion
	ScanPlanDigest Digest
	CatalogDigest  Digest
	CreatedAt      time.Time
	Diagnostics    []Diagnostic
}

// RootScanSourceExpectation protects source configuration, enabled state, and
// mutable source observation state used to construct a root catalog snapshot.
type RootScanSourceExpectation struct {
	SourceID            SourceID
	ObservationRevision uint64
	Enabled             bool
}

// RootScanPublication atomically publishes all source observations and the
// resulting root generation. CatalogResources is the exact root-local snapshot
// calculated by the business layer. The repository verifies it against the
// post-publication source catalog before committing.
type RootScanPublication struct {
	RootCatalog          RootCatalogPublication
	ExpectedRootRevision uint64
	Sources              []RootScanSourceExpectation
	SourceCatalogs       []SourceCatalogPublication
	CatalogResources     []CatalogResource
}

// DependencySnapshotPublication replaces one record's resolution snapshot for
// one root definition and catalog generation.
type DependencySnapshotPublication struct {
	RootID                   RootID
	RecordID                 RecordID
	RootDefinitionDigest     Digest
	CatalogGeneration        uint64
	ExpectedRecordModifiedAt time.Time
	Snapshots                []ArtifactDependencySnapshot
}

// RecordUpdate replaces mutable app-local record fields. ClearCollection is
// explicit because a nil CollectionID otherwise cannot distinguish clear from
// leave-unchanged semantics.
type RecordUpdate struct {
	ExpectedModifiedAt time.Time
	CollectionID       *CollectionID
	ClearCollection    bool
	Enabled            *bool
	DataSchemaID       *SchemaID
	Data               *json.RawMessage
}

// RecordSynchronizationUpdate is an optimistic source-state update. Expected
// values prevent synchronization from overwriting a concurrent local record
// mutation such as pinning, detaching, or moving a record.
type RecordSynchronizationUpdate struct {
	Record               ArtifactRecord
	ExpectedModifiedAt   time.Time
	ExpectedRecordMode   RecordMode
	ExpectedTrackingMode TrackingMode
}

// RecordSynchronizationPublication commits one root synchronization as a
// single app-metadata transaction. Portable definition content is already
// content-addressed and is not part of this transaction.
type RecordSynchronizationPublication struct {
	RootID                    RootID
	ExpectedCatalogGeneration uint64
	Creates                   []ArtifactRecord
	Updates                   []RecordSynchronizationUpdate
}

// RecordTransferPublication atomically publishes the app-local metadata for an
// imported, captured, or forked record. The portable definition is immutable
// content and may be persisted before this transaction without compromising
// metadata consistency.
type RecordTransferPublication struct {
	Resource                          CatalogResource
	Revision                          CatalogResourceRevision
	Record                            ArtifactRecord
	Provenance                        TransferProvenance
	ExpectedSourceObservationRevision uint64
	ExpectedRootRevision              uint64
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
	Resource   CatalogResource     `json:"resource"`
	Definition CanonicalDefinition `json:"definition"`
}

// DependencyExplanation preserves all candidates because Artifact Store does
// not choose feature-specific precedence rules.
type DependencyExplanation struct {
	Selector    ArtifactSelector      `json:"selector"`
	Candidates  []DependencyCandidate `json:"candidates"`
	Selected    *DependencyCandidate  `json:"selected,omitempty"`
	Diagnostics []Diagnostic          `json:"diagnostics,omitempty"`
}

// DependencyGraph is a diagnostic graph rooted at one record definition.
type DependencyGraph struct {
	RootRecordID RecordID                           `json:"rootRecordID"`
	Nodes        map[Digest]CanonicalDefinition     `json:"nodes"`
	Edges        map[Digest][]DependencyExplanation `json:"edges"`
	Diagnostics  []Diagnostic                       `json:"diagnostics,omitempty"`
}
