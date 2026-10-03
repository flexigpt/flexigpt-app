package model

import (
	"slices"

	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type Recognition int

const (
	RecognitionNone Recognition = iota
	RecognitionPossible
	RecognitionPreferred
)

type Candidate struct {
	SourceID            sourceModel.SourceID
	SourceKind          sourceModel.SourceKind
	Locator             spec.Locator
	SourceContentDigest cryptoutil.Digest
	Content             []byte
	RequestedDecoderIDs []spec.DecoderID
}

func (c Candidate) RequestsDecoder(
	id spec.DecoderID,
) bool {
	return slices.Contains(c.RequestedDecoderIDs, id)
}

type SourceContent struct {
	Locator spec.Locator
	Content []byte
	Digest  cryptoutil.Digest
}

type Decoded struct {
	SubresourceLocator spec.SubresourceLocator

	OriginLocator       spec.Locator
	OriginContentDigest *cryptoutil.Digest

	Definition  definitionModel.Definition
	Diagnostics []diagnostic.Diagnostic
}
