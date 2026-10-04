package ingest

import (
	"context"

	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/driver"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

// Scanner is the narrow Source/Ingest capability consumed by Refresh. Scanner
// implementations remain owner-local; observations are explicit Ingest values.
type Scanner interface {
	Discover(
		ctx context.Context,
		value sourceModel.Source,
		snapshot driver.Snapshot,
	) (Result, error)

	DecoderFingerprint() (cryptoutil.Digest, error)
}

// CompiledDocumentRegistrar is the trusted generated-declaration registration
// capability. Install translates its package plans into these Ingest-owned
// source-relative witnesses before calling it.
type CompiledDocumentRegistrar interface {
	RegisterCompiledDocuments(
		ctx context.Context,
		rootID rootModel.RootID,
		sourceID sourceModel.SourceID,
		documents []CompiledDocument,
	) error
}
