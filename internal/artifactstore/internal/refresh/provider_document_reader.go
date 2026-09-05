package refreshimpl

import (
	"context"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	sourceimpl "github.com/flexigpt/flexigpt-app/internal/artifactstore/internal/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
)

type providerPlanningDocumentReader struct {
	rootID    basespec.RootID
	attached  map[basespec.SourceID]struct{}
	runtime   sourceimpl.Runtime
	documents providerapi.ExpectedCanonicalizer
}

func newProviderPlanningDocumentReader(
	input providerRefreshInput,
	runtime sourceimpl.Runtime,
	documents providerapi.ExpectedCanonicalizer,
) providerPlanningDocumentReader {
	attached := make(
		map[basespec.SourceID]struct{},
		len(input.providerAttachments),
	)
	for _, attachment := range input.providerAttachments {
		attached[attachment.SourceID] = struct{}{}
	}
	return providerPlanningDocumentReader{
		rootID:    input.collection.RootID,
		attached:  attached,
		runtime:   runtime,
		documents: documents,
	}
}

func (r providerPlanningDocumentReader) ReadCanonicalDocument(
	ctx context.Context,
	request providerapi.PlanningDocumentRequest,
) (result providerapi.PlanningDocument, returnErr error) {
	if r.runtime == nil || r.documents == nil {
		return providerapi.PlanningDocument{}, fmt.Errorf(
			"%w: provider planning document dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	if ctx == nil {
		return providerapi.PlanningDocument{}, fmt.Errorf(
			"%w: provider planning document context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return providerapi.PlanningDocument{}, err
	}
	if err := request.Validate(); err != nil {
		return providerapi.PlanningDocument{}, err
	}
	if _, attached := r.attached[request.SourceID]; !attached {
		return providerapi.PlanningDocument{}, fmt.Errorf(
			"%w: planning Source %q is not attached to the Collection",
			basespec.ErrAttachmentNotFound,
			request.SourceID,
		)
	}

	sourceValue, err := r.runtime.Get(ctx, r.rootID, request.SourceID)
	if err != nil {
		return providerapi.PlanningDocument{}, err
	}

	snapshot, err := r.runtime.Open(ctx, sourceValue)
	if err != nil {
		return providerapi.PlanningDocument{}, err
	}
	defer func() {
		returnErr = errors.Join(returnErr, snapshot.Close())
	}()

	result = providerapi.PlanningDocument{
		SourceID:   request.SourceID,
		Generation: snapshot.Generation(),
	}

	entry, err := snapshot.Stat(ctx, request.Locator)
	if errors.Is(err, basespec.ErrNotFound) {
		if err := snapshot.Confirm(ctx); err != nil {
			return providerapi.PlanningDocument{}, err
		}
		if err := result.Validate(); err != nil {
			return providerapi.PlanningDocument{}, err
		}
		return result, nil
	}
	if err != nil {
		return providerapi.PlanningDocument{}, err
	}
	if err := entry.Validate(); err != nil {
		return providerapi.PlanningDocument{}, fmt.Errorf(
			"%w: planning document entry is invalid: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if entry.Locator != request.Locator || !entry.IsRegular {
		return providerapi.PlanningDocument{}, fmt.Errorf(
			"%w: planning document %q is not a regular file",
			basespec.ErrInvalid,
			request.Locator,
		)
	}

	content, err := sourceimpl.ReadSnapshotEntry(
		ctx,
		snapshot,
		entry,
		int64(basespec.MaxDefinitionBytes),
	)
	if err != nil {
		return providerapi.PlanningDocument{}, err
	}

	document, err := r.documents.CanonicalizeExpected(
		ctx,
		request.ExpectedSchema,
		content,
	)
	if err != nil {
		return providerapi.PlanningDocument{}, fmt.Errorf(
			"canonicalize provider planning document %q: %w",
			request.Locator,
			err,
		)
	}
	if err := snapshot.Confirm(ctx); err != nil {
		return providerapi.PlanningDocument{}, err
	}

	document = document.Clone()
	result.Found = true
	result.Document = &document
	if err := result.Validate(); err != nil {
		return providerapi.PlanningDocument{}, err
	}
	return result, nil
}
