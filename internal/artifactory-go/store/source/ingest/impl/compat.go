// Package ingestimpl is a temporary compatibility facade.
//
// Deprecated: Source/Ingest public contracts and composition capabilities
// will replace this package after decoder and schema contracts are extracted.
package ingestimpl

import "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest/internal"

type (
	DecoderRegistry  = internal.DecoderRegistry
	Result           = internal.Result
	Engine           = internal.Engine
	ObservationState = internal.ObservationState
	Observation      = internal.Observation
)

const (
	DiagnosticCodeCandidateTooLarge         = internal.DiagnosticCodeCandidateTooLarge
	DiagnosticCodeContentDigestMismatch     = internal.DiagnosticCodeContentDigestMismatch
	DiagnosticCodeDecoderAmbiguous          = internal.DiagnosticCodeDecoderAmbiguous
	DiagnosticCodeDecoderInvalidRecognition = internal.DiagnosticCodeDecoderInvalidRecognition
	DiagnosticCodeDefinitionInvalid         = internal.DiagnosticCodeDefinitionInvalid
	DiagnosticCodeSubresourceDuplicate      = internal.DiagnosticCodeSubresourceDuplicate
	DiagnosticCodeOriginConflict            = internal.DiagnosticCodeOriginConflict

	ObservationValid   = internal.ObservationValid
	ObservationInvalid = internal.ObservationInvalid
)

var (
	NewDecoderRegistry = internal.NewDecoderRegistry
	NewEngine          = internal.NewEngine
)
