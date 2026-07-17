package metadatastore

import "database/sql"

// The row types below are the SQLite representation of Artifact Store
// metadata. They intentionally remain private to this adapter. Domain and
// business code only use the structs in artifactstore/spec.

type artifactRootRow struct {
	RootID        string
	Kind          string
	DisplayName   string
	Description   string
	Enabled       int
	MountRevision uint64
	DataSchemaID  string
	Data          []byte
	CreatedAt     string
	ModifiedAt    string
	SoftDeletedAt sql.NullString
}

func (r *artifactRootRow) destinations() []any {
	return []any{
		&r.RootID,
		&r.Kind,
		&r.DisplayName,
		&r.Description,
		&r.Enabled,
		&r.MountRevision,
		&r.DataSchemaID,
		&r.Data,
		&r.CreatedAt,
		&r.ModifiedAt,
		&r.SoftDeletedAt,
	}
}

type artifactSourceRow struct {
	SourceID               string
	Kind                   string
	DisplayName            string
	Enabled                int
	ConfigSchemaID         string
	Config                 []byte
	LastObservedGeneration sql.NullString
	LastScannedAt          sql.NullString
	ObservationRevision    uint64
	Diagnostics            []byte
	CreatedAt              string
	ModifiedAt             string
}

func (r *artifactSourceRow) destinations() []any {
	return []any{
		&r.SourceID,
		&r.Kind,
		&r.DisplayName,
		&r.Enabled,
		&r.ConfigSchemaID,
		&r.Config,
		&r.LastObservedGeneration,
		&r.LastScannedAt,
		&r.ObservationRevision,
		&r.Diagnostics,
		&r.CreatedAt,
		&r.ModifiedAt,
	}
}

type rootSourceAttachmentRow struct {
	RootID       string
	SourceID     string
	Role         string
	Priority     int
	Enabled      int
	DataSchemaID string
	Data         []byte
	CreatedAt    string
	ModifiedAt   string
}

func (r *rootSourceAttachmentRow) destinations() []any {
	return []any{
		&r.RootID,
		&r.SourceID,
		&r.Role,
		&r.Priority,
		&r.Enabled,
		&r.DataSchemaID,
		&r.Data,
		&r.CreatedAt,
		&r.ModifiedAt,
	}
}

type artifactPackageRow struct {
	SourceID              string
	ManifestLocator       string
	Name                  string
	Version               string
	DisplayName           string
	Description           string
	CurrentManifestDigest sql.NullString
	State                 string
	Diagnostics           []byte
	FirstSeenAt           string
	LastSeenAt            string
}

func (r *artifactPackageRow) destinations() []any {
	return []any{
		&r.SourceID,
		&r.ManifestLocator,
		&r.Name,
		&r.Version,
		&r.DisplayName,
		&r.Description,
		&r.CurrentManifestDigest,
		&r.State,
		&r.Diagnostics,
		&r.FirstSeenAt,
		&r.LastSeenAt,
	}
}

type catalogResourceRow struct {
	SourceID                string
	Locator                 string
	SubresourceLocator      string
	PackageManifestLocator  string
	Kind                    string
	LogicalName             string
	LogicalVersion          string
	CurrentDefinitionDigest sql.NullString
	SourceContentDigest     sql.NullString
	FrontendID              string
	State                   string
	FirstSeenAt             string
	LastSeenAt              string
	Diagnostics             []byte
}

func (r *catalogResourceRow) destinations() []any {
	return []any{
		&r.SourceID,
		&r.Locator,
		&r.SubresourceLocator,
		&r.PackageManifestLocator,
		&r.Kind,
		&r.LogicalName,
		&r.LogicalVersion,
		&r.CurrentDefinitionDigest,
		&r.SourceContentDigest,
		&r.FrontendID,
		&r.State,
		&r.FirstSeenAt,
		&r.LastSeenAt,
		&r.Diagnostics,
	}
}

type catalogResourceRevisionRow struct {
	SourceID            string
	Locator             string
	SubresourceLocator  string
	DefinitionDigest    string
	SourceContentDigest string
	Kind                string
	FrontendID          string
	FirstSeenAt         string
	LastSeenAt          string
}

func (r *catalogResourceRevisionRow) destinations() []any {
	return []any{
		&r.SourceID,
		&r.Locator,
		&r.SubresourceLocator,
		&r.DefinitionDigest,
		&r.SourceContentDigest,
		&r.Kind,
		&r.FrontendID,
		&r.FirstSeenAt,
		&r.LastSeenAt,
	}
}

type artifactCollectionRow struct {
	CollectionID  string
	RootID        string
	Kind          string
	Slug          string
	DisplayName   string
	Description   string
	Enabled       int
	DataSchemaID  string
	Data          []byte
	CreatedAt     string
	ModifiedAt    string
	SoftDeletedAt sql.NullString
}

func (r *artifactCollectionRow) destinations() []any {
	return []any{
		&r.CollectionID,
		&r.RootID,
		&r.Kind,
		&r.Slug,
		&r.DisplayName,
		&r.Description,
		&r.Enabled,
		&r.DataSchemaID,
		&r.Data,
		&r.CreatedAt,
		&r.ModifiedAt,
		&r.SoftDeletedAt,
	}
}

type artifactRecordRow struct {
	RecordID                     string
	RootID                       string
	CollectionID                 sql.NullString
	Kind                         string
	Name                         string
	Version                      string
	SourceID                     string
	Locator                      string
	SubresourceLocator           string
	RecordMode                   string
	TrackingMode                 string
	PinnedDefinitionDigest       sql.NullString
	LastResolvedDefinitionDigest sql.NullString
	Enabled                      int
	DataSchemaID                 string
	Data                         []byte
	State                        string
	Diagnostics                  []byte
	CreatedAt                    string
	ModifiedAt                   string
}

func (r *artifactRecordRow) destinations() []any {
	return []any{
		&r.RecordID,
		&r.RootID,
		&r.CollectionID,
		&r.Kind,
		&r.Name,
		&r.Version,
		&r.SourceID,
		&r.Locator,
		&r.SubresourceLocator,
		&r.RecordMode,
		&r.TrackingMode,
		&r.PinnedDefinitionDigest,
		&r.LastResolvedDefinitionDigest,
		&r.Enabled,
		&r.DataSchemaID,
		&r.Data,
		&r.State,
		&r.Diagnostics,
		&r.CreatedAt,
		&r.ModifiedAt,
	}
}

type transferProvenanceRow struct {
	ProvenanceID           string
	TargetRecordID         string
	Operation              string
	OriginRecordID         sql.NullString
	OriginResource         []byte
	OriginDefinitionDigest string
	CreatedAt              string
}

func (r *transferProvenanceRow) destinations() []any {
	return []any{
		&r.ProvenanceID,
		&r.TargetRecordID,
		&r.Operation,
		&r.OriginRecordID,
		&r.OriginResource,
		&r.OriginDefinitionDigest,
		&r.CreatedAt,
	}
}

type artifactDependencyRow struct {
	RootID               string
	RecordID             string
	CatalogGeneration    uint64
	RootDefinitionDigest string
	DefinitionDigest     string
	SelectorIndex        int
	Selector             []byte
	State                string
	Candidates           []byte
	Diagnostics          []byte
	ModifiedAt           string
}

func (r *artifactDependencyRow) destinations() []any {
	return []any{
		&r.RootID,
		&r.RecordID,
		&r.CatalogGeneration,
		&r.RootDefinitionDigest,
		&r.DefinitionDigest,
		&r.SelectorIndex,
		&r.Selector,
		&r.State,
		&r.Candidates,
		&r.Diagnostics,
		&r.ModifiedAt,
	}
}
