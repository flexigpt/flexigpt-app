package spec

import (
	"context"
	"encoding/json"
	"io"
)

// MaterializeSourceRequest requests a stable real-directory projection of a
// source. PublicationKey is app-local and must not be placed in portable data.
type MaterializeSourceRequest struct {
	SourceID       SourceID
	Root           SourceLocator
	PublicationKey string
	MaxEntries     int
	MaxFiles       int
	MaxBytes       int64
}

// SourceMaterializationInput is passed to the injected source materializer.
// Driver remains the sole source transport implementation.
type SourceMaterializationInput struct {
	Source         ArtifactSource
	Driver         SourceDriver
	Root           SourceLocator
	PublicationKey string
	MaxEntries     int
	MaxFiles       int
	MaxBytes       int64
}

// MaterializedSource is an app-local runtime projection. RootPath must never be
// written into portable definitions or package manifests.
type MaterializedSource struct {
	PublicationKey string
	RootPath       string
	Generation     SourceGeneration
	Entries        int
	Files          int
	Bytes          int64
}

// SourceMaterializer copies a SourceDriver snapshot into a real directory
// publication. Implementations must not use direct operating-system filesystem
// APIs from Artifact Store business logic.
type SourceMaterializer interface {
	Materialize(
		ctx context.Context,
		input SourceMaterializationInput,
	) (MaterializedSource, error)
}

// DirectoryPublisher is the write-side storage port used by the generic source
// materializer. A production implementation should be backed by verified
// LLMTools create, write, move, and delete operations.
type DirectoryPublisher interface {
	BeginDirectoryPublication(
		ctx context.Context,
		publicationKey string,
		generation SourceGeneration,
	) (DirectoryPublication, error)
}

// DirectoryPublication represents an isolated staging directory. Commit must
// publish the staging tree atomically from a consumer's perspective. Abort must
// remove only staging state owned by this publication.
type DirectoryPublication interface {
	MakeDirectory(
		ctx context.Context,
		relativePath PortablePath,
		mode uint32,
	) error
	WriteFile(
		ctx context.Context,
		relativePath PortablePath,
		mode uint32,
		content io.Reader,
	) error
	Commit(ctx context.Context) (publishedRootPath string, err error)
	Abort(ctx context.Context) error
}

type TransferMaterializationMode string

const (
	TransferMaterializeDefinitionOnly TransferMaterializationMode = "definition-only"
	TransferMaterializeExportClosure  TransferMaterializationMode = "export-closure"
)

// TransferDestination is the complete app-local destination for import,
// capture, and fork. Artifact Store derives kind and pinned digest from the
// portable definition.
type TransferDestination struct {
	RootID       RootID
	CollectionID *CollectionID

	SourceID           SourceID
	Locator            SourceLocator
	SubresourceLocator SubresourceLocator

	Name    RecordName
	Version RecordVersion
	Enabled bool

	DataSchemaID SchemaID
	Data         json.RawMessage

	FrontendID             FrontendID
	PackageManifestLocator SourceLocator
}

type PortableAssetContent struct {
	Manifest AssetManifestEntry
	Content  []byte
}

// DefinitionTransferPayload contains shareable files only. It contains no root,
// source, collection, record, local path, secret, or runtime identity.
type DefinitionTransferPayload struct {
	RootDefinitionDigest Digest
	Definitions          []ArtifactDefinitionFile
	Assets               []PortableAssetContent
}

type DefinitionMaterializationRequest struct {
	Source      ArtifactSource
	Destination TransferDestination
	Payload     DefinitionTransferPayload
	Exclusive   bool
}

// DefinitionMaterialization is returned only after the destination publication
// is durable and visible. Receipt is opaque and scoped to safe compensation.
type DefinitionMaterialization struct {
	SourceContentDigest Digest
	Receipt             string
}

// DefinitionMaterializer owns source-kind-specific writes for transfer
// operations. MaterializeDefinition must honor Exclusive without overwriting an
// existing destination. DiscardDefinition must only remove data represented by
// the supplied receipt.
type DefinitionMaterializer interface {
	Kind() SourceKind
	MaterializeDefinition(
		ctx context.Context,
		request DefinitionMaterializationRequest,
	) (DefinitionMaterialization, error)
	DiscardDefinition(
		ctx context.Context,
		source ArtifactSource,
		receipt string,
	) error
}

type ImportDefinitionRequest struct {
	File        ArtifactDefinitionFile
	Destination TransferDestination
}

type CaptureRecordRequest struct {
	OriginRecordID      RecordID
	Destination         TransferDestination
	MaterializationMode TransferMaterializationMode
}

type ForkRecordRequest struct {
	OriginRecordID      RecordID
	Destination         TransferDestination
	MaterializationMode TransferMaterializationMode
}
