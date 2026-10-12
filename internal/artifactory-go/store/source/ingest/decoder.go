package ingest

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema"
	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	ingestModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
)

type SourceEntryReader interface {
	ReadSourceEntry(
		ctx context.Context,
		locator storeSpec.Locator,
	) (ingestModel.SourceContent, error)
}

type Decoder interface {
	ID() storeSpec.DecoderID
	Revision() string

	Recognize(
		ctx context.Context,
		candidate ingestModel.Candidate,
	) ingestModel.Recognition

	Decode(
		ctx context.Context,
		candidate ingestModel.Candidate,
	) ([]ingestModel.Decoded, []diagnostic.Diagnostic)
}

type SourceAwareDecoder interface {
	Decoder

	DecodeWithSource(
		ctx context.Context,
		candidate ingestModel.Candidate,
		reader SourceEntryReader,
	) ([]ingestModel.Decoded, []diagnostic.Diagnostic)
}

type SchemaCanonicalizerBinder interface {
	RequiredSchemaKeys() []schemaModel.Key

	BindExpectedCanonicalizer(
		schemas schema.Catalog,
	) error
}
