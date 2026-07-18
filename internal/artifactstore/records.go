package artifactstore

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/validate"
)

func (s *Store) CreateRecord(ctx context.Context, draft spec.ArtifactRecordDraft) (spec.ArtifactRecord, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	defer finish()
	generation, err := s.repository.GetRootCatalogGeneration(ctx, draft.RootID)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	if err := s.ensureRootCatalogCurrent(ctx, draft.RootID, generation); err != nil {
		return spec.ArtifactRecord{}, err
	}
	record, _, _, err := s.prepareRecord(ctx, draft)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}

	if err := s.repository.PublishRecordSynchronization(
		ctx,
		spec.RecordSynchronizationPublication{
			RootID:                    draft.RootID,
			ExpectedCatalogGeneration: generation.Generation,
			Creates:                   []spec.ArtifactRecord{record},
		},
	); err != nil {
		return spec.ArtifactRecord{}, err
	}
	return record, nil
}

func (s *Store) ListRecords(ctx context.Context, rootID spec.RootID) ([]spec.ArtifactRecord, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return nil, err
	}
	defer finish()
	if _, err := s.repository.GetRoot(ctx, rootID, false); err != nil {
		return nil, err
	}
	return s.repository.ListRecordsForRoot(ctx, rootID)
}

func (s *Store) UpdateRecord(
	ctx context.Context,
	recordID spec.RecordID,
	update spec.RecordUpdate,
) (spec.ArtifactRecord, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	defer finish()
	if update.ClearCollection && update.CollectionID != nil {
		return spec.ArtifactRecord{}, fmt.Errorf(
			"%w: clearCollection and collectionID are mutually exclusive",
			spec.ErrInvalidRequest,
		)
	}
	record, err := s.GetRecord(ctx, recordID)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	original := record
	if err := requireExpectedModifiedAt(
		"record "+string(recordID),
		record.ModifiedAt,
		update.ExpectedModifiedAt,
	); err != nil {
		return spec.ArtifactRecord{}, err
	}
	if update.ClearCollection {
		record.CollectionID = nil
	} else if update.CollectionID != nil {
		record.CollectionID = update.CollectionID
	}
	if update.Enabled != nil {
		record.Enabled = *update.Enabled
	}
	if update.DataSchemaID != nil {
		record.DataSchemaID = *update.DataSchemaID
	}
	if update.Data != nil {
		record.Data = normalizedJSONObject(*update.Data)
	}
	if reflect.DeepEqual(original.CollectionID, record.CollectionID) &&
		original.Enabled == record.Enabled &&
		original.DataSchemaID == record.DataSchemaID &&
		equivalentJSONObjects(original.Data, record.Data) {
		return original, nil
	}

	var resource *spec.CatalogResource
	resourceValue, err := s.repository.GetCatalogResource(
		ctx,
		spec.CatalogResourceKey{
			SourceID:           record.SourceID,
			Locator:            record.Locator,
			SubresourceLocator: record.SubresourceLocator,
		},
	)
	if err == nil {
		resource = &resourceValue
	} else if !isNotFound(err) {
		return spec.ArtifactRecord{}, err
	}
	var definition *spec.CanonicalDefinition
	if record.LastResolvedDefinitionDigest != nil {
		definitionValue, err := s.GetDefinitionByDigest(ctx, *record.LastResolvedDefinitionDigest)
		if err != nil {
			return spec.ArtifactRecord{}, err
		}
		definition = &definitionValue
	}
	record.ModifiedAt = s.nextModifiedAt(record.ModifiedAt)
	if err := s.validateRecord(ctx, &record, resource, definition); err != nil {
		return spec.ArtifactRecord{}, err
	}
	if err := s.repository.UpdateRecord(ctx, record, update.ExpectedModifiedAt); err != nil {
		return spec.ArtifactRecord{}, err
	}
	return record, nil
}

func (s *Store) RefreshRecord(ctx context.Context, recordID spec.RecordID) (spec.ArtifactRecord, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	defer finish()

	record, err := s.GetRecord(ctx, recordID)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	var resource *spec.CatalogResource
	if !isDetachedPinnedRecord(record) {
		resourceValue, err := s.publishedCatalogResource(
			ctx,
			record.RootID,
			spec.CatalogResourceKey{
				SourceID:           record.SourceID,
				Locator:            record.Locator,
				SubresourceLocator: record.SubresourceLocator,
			},
		)
		if err == nil {
			resource = &resourceValue
		} else if !isNotFound(err) {
			return spec.ArtifactRecord{}, err
		}
	}

	if !applyCatalogResourceToRecord(&record, resource, true) {
		return record, nil
	}
	expectedModifiedAt := record.ModifiedAt
	record.ModifiedAt = s.nextModifiedAt(record.ModifiedAt)
	var definition *spec.CanonicalDefinition
	if record.LastResolvedDefinitionDigest != nil {
		definitionValue, err := s.GetDefinitionByDigest(ctx, *record.LastResolvedDefinitionDigest)
		if err != nil {
			return spec.ArtifactRecord{}, err
		}
		definition = &definitionValue
	}
	if err := s.validateRecord(ctx, &record, resource, definition); err != nil {
		return spec.ArtifactRecord{}, err
	}
	if err := s.repository.UpdateRecord(ctx, record, expectedModifiedAt); err != nil {
		return spec.ArtifactRecord{}, err
	}
	return record, nil
}

func (s *Store) DeleteRecord(
	ctx context.Context,
	recordID spec.RecordID,
	expectedModifiedAt time.Time,
) error {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return err
	}
	defer finish()
	record, err := s.GetRecord(ctx, recordID)
	if err != nil {
		return err
	}
	if err := requireExpectedModifiedAt("record "+string(recordID), record.ModifiedAt, expectedModifiedAt); err != nil {
		return err
	}
	return s.repository.DeleteRecord(ctx, recordID, expectedModifiedAt)
}

func (s *Store) ExportRecord(ctx context.Context, recordID spec.RecordID) (spec.ExportedRecord, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return spec.ExportedRecord{}, err
	}
	defer finish()
	return s.exportRecord(ctx, recordID)
}

func (s *Store) exportRecord(ctx context.Context, recordID spec.RecordID) (spec.ExportedRecord, error) {
	record, err := s.GetRecord(ctx, recordID)
	if err != nil {
		return spec.ExportedRecord{}, err
	}
	if record.LastResolvedDefinitionDigest == nil {
		return spec.ExportedRecord{}, fmt.Errorf("%w: record has no resolved definition", spec.ErrConflict)
	}
	definition, err := s.GetDefinitionByDigest(ctx, *record.LastResolvedDefinitionDigest)
	if err != nil {
		return spec.ExportedRecord{}, err
	}
	var resource *spec.CatalogResource
	resourceValue, resourceErr := s.repository.GetCatalogResource(
		ctx,
		spec.CatalogResourceKey{
			SourceID:           record.SourceID,
			Locator:            record.Locator,
			SubresourceLocator: record.SubresourceLocator,
		},
	)
	if resourceErr != nil && !isNotFound(resourceErr) {
		return spec.ExportedRecord{}, resourceErr
	}
	if resourceErr == nil {
		resource = &resourceValue
	}
	closure := spec.ExportClosure{DefinitionDigests: []spec.Digest{definition.Digest}, Assets: definition.AssetManifest}
	frontend, found, err := s.recordFrontendForDefinition(ctx, record, resource, &definition)
	if err != nil {
		return spec.ExportedRecord{}, err
	}
	if found {
		candidate, diagnostics := frontend.DescribeExportClosure(ctx, definition)
		if err := errorDiagnostics("export closure", diagnostics); err != nil {
			return spec.ExportedRecord{}, err
		}
		closure = candidate
	} else if resource != nil && resource.FrontendID != "" {
		return spec.ExportedRecord{}, fmt.Errorf(
			"%w: frontend %q required to export record %q",
			spec.ErrFrontendUnavailable,
			resource.FrontendID,
			record.RecordID,
		)
	}
	if err := validate.ValidateExportClosure(definition, closure); err != nil {
		return spec.ExportedRecord{}, fmt.Errorf(
			"%w: export closure: %w",
			spec.ErrInvalidRequest,
			err,
		)
	}
	slices.Sort(closure.DefinitionDigests)
	sort.Slice(closure.Assets, func(left, right int) bool {
		return closure.Assets[left].Path < closure.Assets[right].Path
	})

	return spec.ExportedRecord{
		Record:     record,
		Definition: spec.ArtifactDefinitionFile{Format: spec.ArtifactDefinitionFileFormatV1, Definition: definition},
		Closure:    closure,
	}, nil
}

func (s *Store) prepareRecord(
	ctx context.Context,
	draft spec.ArtifactRecordDraft,
) (spec.ArtifactRecord, spec.CatalogResource, spec.CanonicalDefinition, error) {
	resource, definition, digest, err := s.resolveRecordTarget(ctx, draft)
	if err != nil {
		return spec.ArtifactRecord{}, spec.CatalogResource{}, spec.CanonicalDefinition{}, err
	}
	record, err := s.prepareRecordForResolved(ctx, draft, resource, definition, digest)
	if err != nil {
		return spec.ArtifactRecord{}, spec.CatalogResource{}, spec.CanonicalDefinition{}, err
	}
	return record, resource, definition, nil
}

func (s *Store) prepareRecordForResolved(
	ctx context.Context,
	draft spec.ArtifactRecordDraft,
	resource spec.CatalogResource,
	definition spec.CanonicalDefinition,
	digest spec.Digest,
) (spec.ArtifactRecord, error) {
	root, err := s.repository.GetRoot(ctx, draft.RootID, false)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	if !root.Enabled {
		return spec.ArtifactRecord{}, fmt.Errorf(
			"%w: root %q is disabled",
			spec.ErrConflict,
			draft.RootID,
		)
	}
	source, err := s.repository.GetSource(ctx, draft.SourceID)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	if draft.RecordMode == spec.RecordModeEmbeddedOverlay &&
		source.Kind != spec.SourceKindEmbeddedFSDirectory {
		return spec.ArtifactRecord{}, fmt.Errorf(
			"%w: embedded-overlay record requires an embedded filesystem source",
			spec.ErrInvalidRequest,
		)
	}
	attachment, err := s.repository.GetRootSourceAttachment(ctx, draft.RootID, draft.SourceID)
	if err != nil {
		if isNotFound(err) {
			return spec.ArtifactRecord{}, fmt.Errorf(
				"%w: source %q is not attached to root %q",
				spec.ErrSourceNotAttached,
				draft.SourceID,
				draft.RootID,
			)
		}
		return spec.ArtifactRecord{}, err
	}
	if !source.Enabled || !attachment.Enabled {
		return spec.ArtifactRecord{}, fmt.Errorf(
			"%w: source %q is disabled for root %q",
			spec.ErrConflict,
			draft.SourceID,
			draft.RootID,
		)
	}
	if resource.SourceID != draft.SourceID ||
		resource.Locator != draft.Locator ||
		resource.SubresourceLocator != draft.SubresourceLocator ||
		resource.Kind != draft.Kind ||
		definition.Kind != draft.Kind ||
		definition.Digest != digest {
		return spec.ArtifactRecord{}, fmt.Errorf(
			"%w: resolved definition does not match record target",
			spec.ErrInvalidRequest,
		)
	}

	id, err := s.newID()
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	now := s.nowUTC()
	resolvedDigest := digest
	record := spec.ArtifactRecord{
		RecordID:                     spec.RecordID(id),
		RootID:                       draft.RootID,
		CollectionID:                 draft.CollectionID,
		Kind:                         draft.Kind,
		Name:                         draft.Name,
		Version:                      draft.Version,
		SourceID:                     draft.SourceID,
		Locator:                      draft.Locator,
		SubresourceLocator:           draft.SubresourceLocator,
		RecordMode:                   draft.RecordMode,
		TrackingMode:                 draft.TrackingMode,
		PinnedDefinitionDigest:       cloneDigest(draft.PinnedDefinitionDigest),
		LastResolvedDefinitionDigest: &resolvedDigest,
		Enabled:                      draft.Enabled,
		DataSchemaID:                 draft.DataSchemaID,
		Data:                         normalizedJSONObject(draft.Data),
		State:                        spec.RecordStateAvailable,
		CreatedAt:                    now,
		ModifiedAt:                   now,
	}
	if err := s.validateRecord(ctx, &record, &resource, &definition); err != nil {
		return spec.ArtifactRecord{}, err
	}
	return record, nil
}

func (s *Store) resolveRecordTarget(
	ctx context.Context,
	draft spec.ArtifactRecordDraft,
) (spec.CatalogResource, spec.CanonicalDefinition, spec.Digest, error) {
	resource, err := s.publishedCatalogResource(
		ctx,
		draft.RootID,
		spec.CatalogResourceKey{
			SourceID:           draft.SourceID,
			Locator:            draft.Locator,
			SubresourceLocator: draft.SubresourceLocator,
		},
	)
	if err != nil {
		return spec.CatalogResource{}, spec.CanonicalDefinition{}, "", err
	}
	if resource.State != spec.CatalogStateValid || resource.CurrentDefinitionDigest == nil ||
		resource.Kind != draft.Kind {
		return spec.CatalogResource{}, spec.CanonicalDefinition{}, "", fmt.Errorf(
			"%w: catalog resource is not a valid matching record target",
			spec.ErrConflict,
		)
	}
	digest := *resource.CurrentDefinitionDigest
	if draft.TrackingMode == spec.TrackingModePinDigest {
		if draft.PinnedDefinitionDigest == nil {
			return spec.CatalogResource{}, spec.CanonicalDefinition{}, "", fmt.Errorf(
				"%w: pinned digest required",
				spec.ErrInvalidRequest,
			)
		}
		digest = *draft.PinnedDefinitionDigest
	}
	definition, err := s.GetDefinitionByDigest(ctx, digest)
	if err != nil {
		return spec.CatalogResource{}, spec.CanonicalDefinition{}, "", err
	}
	if definition.Kind != draft.Kind {
		return spec.CatalogResource{}, spec.CanonicalDefinition{}, "", fmt.Errorf(
			"%w: definition kind %q does not match record kind %q",
			spec.ErrInvalidRequest,
			definition.Kind,
			draft.Kind,
		)
	}
	return resource, definition, digest, nil
}

func (s *Store) PinRecord(
	ctx context.Context,
	recordID spec.RecordID,
	digest spec.Digest,
	expectedModifiedAt time.Time,
) (spec.ArtifactRecord, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	defer finish()

	record, err := s.GetRecord(ctx, recordID)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	if err := requireExpectedModifiedAt("record "+string(recordID), record.ModifiedAt, expectedModifiedAt); err != nil {
		return spec.ArtifactRecord{}, err
	}
	belongs, err := s.definitionBelongsToRecordOccurrence(ctx, record, digest)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	if !belongs {
		return spec.ArtifactRecord{}, fmt.Errorf(
			"%w: definition %q is not a revision of record occurrence %q",
			spec.ErrInvalidRequest,
			digest,
			recordID,
		)
	}
	definition, err := s.GetDefinitionByDigest(ctx, digest)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	if definition.Kind != record.Kind {
		return spec.ArtifactRecord{}, fmt.Errorf(
			"%w: pinned definition kind %q does not match record kind %q",
			spec.ErrInvalidRequest,
			definition.Kind,
			record.Kind,
		)
	}

	record.TrackingMode = spec.TrackingModePinDigest
	record.PinnedDefinitionDigest = &digest
	record.LastResolvedDefinitionDigest = &digest
	if record.State != spec.RecordStateMissing || isDetachedPinnedRecord(record) {
		record.State = spec.RecordStateAvailable
	}
	record.ModifiedAt = s.nextModifiedAt(record.ModifiedAt)
	var resource *spec.CatalogResource
	resourceValue, resourceErr := s.repository.GetCatalogResource(
		ctx,
		spec.CatalogResourceKey{
			SourceID:           record.SourceID,
			Locator:            record.Locator,
			SubresourceLocator: record.SubresourceLocator,
		},
	)
	if resourceErr == nil {
		resource = &resourceValue
	} else if !isNotFound(resourceErr) {
		return spec.ArtifactRecord{}, resourceErr
	}
	if err := s.validateRecord(ctx, &record, resource, &definition); err != nil {
		return spec.ArtifactRecord{}, err
	}
	if err := s.repository.UpdateRecord(ctx, record, expectedModifiedAt); err != nil {
		return spec.ArtifactRecord{}, err
	}
	return record, nil
}

// DetachRecord pins the existing occurrence in place. It is deliberately
// distinct from CaptureRecord and ForkRecord, which create new occurrences.
func (s *Store) DetachRecord(
	ctx context.Context,
	recordID spec.RecordID,
	expectedModifiedAt time.Time,
) (spec.ArtifactRecord, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	defer finish()

	record, err := s.GetRecord(ctx, recordID)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	if err := requireExpectedModifiedAt("record "+string(recordID), record.ModifiedAt, expectedModifiedAt); err != nil {
		return spec.ArtifactRecord{}, err
	}
	if isDetachedPinnedRecord(record) {
		return record, nil
	}
	if record.LastResolvedDefinitionDigest == nil {
		return spec.ArtifactRecord{}, fmt.Errorf(
			"%w: record has no resolved definition",
			spec.ErrConflict,
		)
	}
	digest := *record.LastResolvedDefinitionDigest
	definition, err := s.GetDefinitionByDigest(ctx, digest)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	record.RecordMode = spec.RecordModeCaptured
	record.TrackingMode = spec.TrackingModePinDigest
	record.PinnedDefinitionDigest = &digest
	record.LastResolvedDefinitionDigest = &digest
	record.State = spec.RecordStateAvailable
	record.Diagnostics = nil
	record.ModifiedAt = s.nextModifiedAt(record.ModifiedAt)
	if err := s.validateRecord(ctx, &record, nil, &definition); err != nil {
		return spec.ArtifactRecord{}, err
	}
	if err := s.repository.UpdateRecord(ctx, record, expectedModifiedAt); err != nil {
		return spec.ArtifactRecord{}, err
	}
	return record, nil
}

func (s *Store) GetRecord(ctx context.Context, recordID spec.RecordID) (spec.ArtifactRecord, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	defer finish()
	record, err := s.repository.GetRecord(ctx, recordID)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	if _, err := s.repository.GetRoot(ctx, record.RootID, false); err != nil {
		return spec.ArtifactRecord{}, err
	}
	return record, nil
}

func (s *Store) definitionBelongsToRecordOccurrence(
	ctx context.Context,
	record spec.ArtifactRecord,
	digest spec.Digest,
) (bool, error) {
	key := spec.CatalogResourceKey{
		SourceID:           record.SourceID,
		Locator:            record.Locator,
		SubresourceLocator: record.SubresourceLocator,
	}
	resource, err := s.repository.GetCatalogResource(ctx, key)
	if err == nil && resource.CurrentDefinitionDigest != nil &&
		*resource.CurrentDefinitionDigest == digest {
		return true, nil
	}
	if err != nil && !isNotFound(err) {
		return false, err
	}
	revisions, err := s.repository.ListCatalogResourceRevisions(ctx, key)
	if err != nil {
		return false, err
	}
	for _, revision := range revisions {
		if revision.DefinitionDigest == digest {
			return true, nil
		}
	}
	return false, nil
}

func (s *Store) validateRecord(
	ctx context.Context,
	record *spec.ArtifactRecord,
	resource *spec.CatalogResource,
	definition *spec.CanonicalDefinition,
) error {
	if record == nil {
		return fmt.Errorf("%w: record is nil", spec.ErrInvalidRequest)
	}
	if definition != nil && definition.Kind != record.Kind {
		return fmt.Errorf(
			"%w: definition kind %q does not match record kind %q",
			spec.ErrInvalidRequest,
			definition.Kind,
			record.Kind,
		)
	}
	if definition != nil &&
		resource != nil &&
		resource.State == spec.CatalogStateValid &&
		resource.CurrentDefinitionDigest != nil &&
		*resource.CurrentDefinitionDigest == definition.Digest &&
		resource.Kind == definition.Kind {
		record.Diagnostics = append([]spec.Diagnostic(nil), resource.Diagnostics...)
	} else if definition != nil && record.State == spec.RecordStateAvailable {
		record.Diagnostics = nil
	}
	if err := validate.ValidateArtifactRecord(*record); err != nil {
		return fmt.Errorf("%w: record: %w", spec.ErrInvalidRequest, err)
	}
	if record.CollectionID != nil {
		collection, err := s.repository.GetCollection(ctx, *record.CollectionID, false)
		if err != nil {
			return err
		}
		if collection.RootID != record.RootID {
			return fmt.Errorf("%w: record collection belongs to another root", spec.ErrInvalidRequest)
		}
		if hook, ok := s.collectionHookFor(collection.Kind); ok {
			if err := errorDiagnostics(
				"record placement",
				hook.ValidateRecordPlacement(ctx, collection, *record),
			); err != nil {
				return err
			}
		}
	}
	if definition != nil {
		frontend, found, err := s.recordFrontendForDefinition(ctx, *record, resource, definition)
		if err != nil {
			return err
		}
		if !found {
			return fmt.Errorf(
				"%w: frontend ownership for record %q definition %q is unavailable",
				spec.ErrFrontendUnavailable,
				record.RecordID,
				definition.Digest,
			)
		}
		diagnostics := frontend.ValidateRecordData(
			ctx,
			*definition,
			spec.ArtifactRecordDraft{
				RootID:                 record.RootID,
				CollectionID:           record.CollectionID,
				Kind:                   record.Kind,
				Name:                   record.Name,
				Version:                record.Version,
				SourceID:               record.SourceID,
				Locator:                record.Locator,
				SubresourceLocator:     record.SubresourceLocator,
				RecordMode:             record.RecordMode,
				TrackingMode:           record.TrackingMode,
				PinnedDefinitionDigest: record.PinnedDefinitionDigest,
				Enabled:                record.Enabled,
				DataSchemaID:           record.DataSchemaID,
				Data:                   record.Data,
			},
		)
		if err := errorDiagnostics("record data", diagnostics); err != nil {
			return err
		}
		record.Diagnostics = appendBoundedDiagnostics(record.Diagnostics, diagnostics...)
	}
	if err := validate.ValidateArtifactRecord(*record); err != nil {
		return fmt.Errorf("%w: record: %w", spec.ErrInvalidRequest, err)
	}
	return nil
}

func (s *Store) recordFrontendForDefinition(
	ctx context.Context,
	record spec.ArtifactRecord,
	resource *spec.CatalogResource,
	definition *spec.CanonicalDefinition,
) (spec.ArtifactFrontend, bool, error) {
	if definition == nil {
		return nil, false, nil
	}
	resolve := func(id spec.FrontendID) (spec.ArtifactFrontend, bool, error) {
		if id == "" {
			return nil, false, nil
		}
		frontend, ok := s.frontendFor(id)
		if !ok {
			return nil, false, fmt.Errorf("%w: frontend %q", spec.ErrFrontendUnavailable, id)
		}
		return frontend, true, nil
	}
	if resource != nil &&
		resource.Kind == definition.Kind &&
		resource.CurrentDefinitionDigest != nil &&
		*resource.CurrentDefinitionDigest == definition.Digest {
		return resolve(resource.FrontendID)
	}
	revisions, err := s.repository.ListCatalogResourceRevisions(ctx, spec.CatalogResourceKey{
		SourceID:           record.SourceID,
		Locator:            record.Locator,
		SubresourceLocator: record.SubresourceLocator,
	})
	if err != nil {
		return nil, false, err
	}
	for _, revision := range revisions {
		if revision.DefinitionDigest == definition.Digest &&
			revision.Kind == definition.Kind {
			return resolve(revision.FrontendID)
		}
	}
	return nil, false, nil
}
