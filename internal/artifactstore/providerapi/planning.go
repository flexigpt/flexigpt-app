package providerapi

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
)

// PlanningDocumentReader is the only source-read capability available to a
// document-aware collection planner.
//
// Artifact Store validates that the requested Source is attached to the
// Collection, opens and confirms the snapshot, bounds the read, dispatches the
// expected schema codec, and closes the snapshot. Providers never receive
// source configuration, source paths, a Snapshot, or raw filesystem access.
type PlanningDocumentReader interface {
	ReadCanonicalDocument(
		ctx context.Context,
		request PlanningDocumentRequest,
	) (PlanningDocument, error)
}

// PlanningDocumentRequest identifies one source-relative portable document
// needed to derive a provider discovery plan.
type PlanningDocumentRequest struct {
	SourceID       basespec.SourceID
	Locator        basespec.Locator
	ExpectedSchema SchemaKey
}

func (r PlanningDocumentRequest) Validate() error {
	if err := basespec.ValidateSourceID(r.SourceID); err != nil {
		return err
	}
	if err := basespec.ValidateLocator(r.Locator, false); err != nil {
		return err
	}
	return r.ExpectedSchema.Validate()
}

// PlanningDocument is a Store-verified result of a planning document read.
//
// A missing document is represented by Found=false and Document=nil. The
// confirmed Source generation is returned in both the found and missing cases
// so the provider can place it into the resulting SourcePlan as an optimistic
// precondition.
type PlanningDocument struct {
	SourceID   basespec.SourceID
	Generation string
	Found      bool
	Document   *ParsedDocument
}

func (d PlanningDocument) Clone() PlanningDocument {
	output := d
	if d.Document != nil {
		value := d.Document.Clone()
		output.Document = &value
	}
	return output
}

func (d PlanningDocument) Validate() error {
	if err := basespec.ValidateSourceID(d.SourceID); err != nil {
		return err
	}
	if err := basespec.ValidateSourceGeneration(d.Generation); err != nil {
		return err
	}
	if !d.Found {
		if d.Document != nil {
			return fmt.Errorf(
				"%w: missing planning document cannot contain a document",
				basespec.ErrInvalid,
			)
		}
		return nil
	}
	if d.Document == nil {
		return fmt.Errorf(
			"%w: found planning document is missing its document",
			basespec.ErrInvalid,
		)
	}
	return d.Document.Validate()
}
