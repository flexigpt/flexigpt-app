package ingest

import (
	"context"

	schema "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema"
	schemaModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/schema/model"
	ingestModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
)

type SourceEntryReader interface {
	ReadSourceEntry(
		ctx context.Context,
		locator spec.Locator,
	) (ingestModel.SourceContent, error)
}

type Decoder interface {
	ID() spec.DecoderID
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
