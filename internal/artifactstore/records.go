package artifactstore

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
)

func (s *Store) CreateRecord(ctx context.Context, draft spec.ArtifactRecordDraft) (spec.ArtifactRecord, error) {
	if err := s.ensureOpen(); err != nil {
		return spec.ArtifactRecord{}, err
	}
	record, _, _, err := s.prepareRecord(ctx, draft)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}

	if err := s.repository.CreateRecord(ctx, record); err != nil {
		return spec.ArtifactRecord{}, err
	}
	return record, nil
}

func (s *Store) ListRecords(ctx context.Context, rootID spec.RootID) ([]spec.ArtifactRecord, error) {
	if err := s.ensureOpen(); err != nil {
		return nil, err
	}
	return s.repository.ListRecordsForRoot(ctx, rootID)
}

func (s *Store) UpdateRecord(
	ctx context.Context,
	recordID spec.RecordID,
	update spec.RecordUpdate,
) (spec.ArtifactRecord, error) {
	if err := s.ensureOpen(); err != nil {
		return spec.ArtifactRecord{}, err
	}
	record, err := s.repository.GetRecord(ctx, recordID)
	if err != nil {
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
		record.Data = normalizedJSONObject(update.Data)
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
	record.ModifiedAt = s.nowUTC()
	if err := s.validateRecord(ctx, &record, resource, definition); err != nil {
		return spec.ArtifactRecord{}, err
	}
	if err := s.repository.UpdateRecord(ctx, record); err != nil {
		return spec.ArtifactRecord{}, err
	}
	return record, nil
}

func (s *Store) RefreshRecord(ctx context.Context, recordID spec.RecordID) (spec.ArtifactRecord, error) {
	if err := s.ensureOpen(); err != nil {
		return spec.ArtifactRecord{}, err
	}
	record, err := s.repository.GetRecord(ctx, recordID)
	if err != nil {
		return spec.ArtifactRecord{}, err
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

	if !applyCatalogResourceToRecord(&record, resource, true) {
		return record, nil
	}
	record.ModifiedAt = s.nowUTC()
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
	if err := s.repository.UpdateRecord(ctx, record); err != nil {
		return spec.ArtifactRecord{}, err
	}
	return record, nil
}

func (s *Store) DeleteRecord(ctx context.Context, recordID spec.RecordID) error {
	if err := s.ensureOpen(); err != nil {
		return err
	}
	return s.repository.DeleteRecord(ctx, recordID)
}

func (s *Store) ExportRecord(ctx context.Context, recordID spec.RecordID) (spec.ExportedRecord, error) {
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
	resource, err := s.repository.GetCatalogResource(
		ctx,
		spec.CatalogResourceKey{
			SourceID:           record.SourceID,
			Locator:            record.Locator,
			SubresourceLocator: record.SubresourceLocator,
		},
	)
	if err != nil {
		return spec.ExportedRecord{}, err
	}
	closure := spec.ExportClosure{DefinitionDigests: []spec.Digest{definition.Digest}, Assets: definition.AssetManifest}
	if frontend, ok := s.frontendFor(resource.FrontendID); ok {
		candidate, diagnostics := frontend.DescribeExportClosure(ctx, definition)
		if err := errorDiagnostics("export closure", diagnostics); err != nil {
			return spec.ExportedRecord{}, err
		}
		closure = candidate
	}

	return spec.ExportedRecord{
		Record:     record,
		Definition: spec.ArtifactDefinitionFile{Format: spec.ArtifactDefinitionFileFormatV1, Definition: definition},
		Closure:    closure,
	}, nil
}

func (s *Store) ImportDefinition(
	ctx context.Context,
	request spec.ImportDefinitionRequest,
) (spec.ArtifactRecord, error) {
	if err := s.ensureOpen(); err != nil {
		return spec.ArtifactRecord{}, err
	}
	if s.portableContent == nil {
		return spec.ArtifactRecord{}, fmt.Errorf(
			"%w: portable content repository is not configured",
			spec.ErrUnsupported,
		)
	}
	definition, err := s.portableContent.PutDefinition(ctx, request.File)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}

	now := s.nowUTC()
	digest := definition.Digest
	if request.FrontendID == "" {
		request.FrontendID = "artifactstore.portable-definition"
	}
	resource := spec.CatalogResource{
		SourceID:                request.Record.SourceID,
		Locator:                 request.Record.Locator,
		SubresourceLocator:      request.Record.SubresourceLocator,
		PackageManifestLocator:  request.PackageManifestLocator,
		Kind:                    definition.Kind,
		LogicalName:             definition.LogicalName,
		LogicalVersion:          definition.LogicalVersion,
		CurrentDefinitionDigest: &digest,
		SourceContentDigest:     &digest,
		FrontendID:              request.FrontendID,
		State:                   spec.CatalogStateValid,
		FirstSeenAt:             now,
		LastSeenAt:              now,
	}

	revision := spec.CatalogResourceRevision{
		SourceID:            resource.SourceID,
		Locator:             resource.Locator,
		SubresourceLocator:  resource.SubresourceLocator,
		DefinitionDigest:    digest,
		SourceContentDigest: digest,
		Kind:                definition.Kind,
		FrontendID:          request.FrontendID,
		FirstSeenAt:         now,
		LastSeenAt:          now,
	}

	if request.Record.TrackingMode == "" {
		request.Record.TrackingMode = spec.TrackingModePinDigest
		request.Record.PinnedDefinitionDigest = &digest
	}
	if request.Record.RecordMode == "" {
		request.Record.RecordMode = spec.RecordModeCaptured
	}
	request.Record.Kind = definition.Kind
	record, err := s.prepareRecordForResolved(
		ctx,
		request.Record,
		resource,
		definition,
		digest,
	)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	provenanceID, err := s.newID()
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	provenance := spec.TransferProvenance{
		ProvenanceID:           spec.ProvenanceID(provenanceID),
		TargetRecordID:         record.RecordID,
		Operation:              spec.TransferOperationImport,
		OriginDefinitionDigest: definition.Digest,
		CreatedAt:              record.CreatedAt,
	}
	if err := s.repository.PublishRecordTransfer(ctx, spec.RecordTransferPublication{
		Resource:   resource,
		Revision:   revision,
		Record:     record,
		Provenance: provenance,
	}); err != nil {
		return spec.ArtifactRecord{}, err
	}
	return record, nil
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
	if _, err := s.repository.GetRoot(ctx, draft.RootID, false); err != nil {
		return spec.ArtifactRecord{}, err
	}
	source, err := s.repository.GetSource(ctx, draft.SourceID)
	if err != nil {
		return spec.ArtifactRecord{}, err
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
	resource, err := s.repository.GetCatalogResource(
		ctx,
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

func (s *Store) validateRecord(
	ctx context.Context,
	record *spec.ArtifactRecord,
	resource *spec.CatalogResource,
	definition *spec.CanonicalDefinition,
) error {
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
	if resource != nil && definition != nil && resource.State == spec.CatalogStateValid {
		if frontend, ok := s.frontendFor(resource.FrontendID); ok {
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
			record.Diagnostics = diagnostics
		}
	}
	if err := spec.ValidateArtifactRecord(*record); err != nil {
		return err
	}
	return nil
}

func (s *Store) CaptureRecord(ctx context.Context, recordID spec.RecordID) (spec.ArtifactRecord, error) {
	return s.DetachRecord(ctx, recordID)
}

func (s *Store) ForkRecord(ctx context.Context, recordID spec.RecordID) (spec.ArtifactRecord, error) {
	record, err := s.DetachRecord(ctx, recordID)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	record.RecordMode = spec.RecordModeForked
	record.ModifiedAt = s.nowUTC()
	if err := s.repository.UpdateRecord(ctx, record); err != nil {
		return spec.ArtifactRecord{}, err
	}
	return record, nil
}

// DetachRecord captures the currently resolved portable definition by pinning
// it and changing the local record mode. It deliberately does not write source
// files; a future source driver may materialize an editable fork separately.
func (s *Store) DetachRecord(ctx context.Context, recordID spec.RecordID) (spec.ArtifactRecord, error) {
	record, err := s.GetRecord(ctx, recordID)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	if record.LastResolvedDefinitionDigest == nil {
		return spec.ArtifactRecord{}, fmt.Errorf("%w: record has no resolved definition", spec.ErrConflict)
	}
	record, err = s.PinRecord(ctx, recordID, *record.LastResolvedDefinitionDigest)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	record.RecordMode = spec.RecordModeCaptured
	record.State = spec.RecordStateAvailable
	record.Diagnostics = nil
	record.ModifiedAt = s.nowUTC()
	if err := spec.ValidateArtifactRecord(record); err != nil {
		return spec.ArtifactRecord{}, err
	}
	if err := s.repository.UpdateRecord(ctx, record); err != nil {
		return spec.ArtifactRecord{}, err
	}
	return record, nil
}

func (s *Store) GetRecord(ctx context.Context, recordID spec.RecordID) (spec.ArtifactRecord, error) {
	if err := s.ensureOpen(); err != nil {
		return spec.ArtifactRecord{}, err
	}
	return s.repository.GetRecord(ctx, recordID)
}

func (s *Store) PinRecord(
	ctx context.Context,
	recordID spec.RecordID,
	digest spec.Digest,
) (spec.ArtifactRecord, error) {
	if err := s.ensureOpen(); err != nil {
		return spec.ArtifactRecord{}, err
	}

	definition, err := s.GetDefinitionByDigest(ctx, digest)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}

	record, err := s.repository.GetRecord(ctx, recordID)
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
	record.ModifiedAt = s.nowUTC()
	if err := spec.ValidateArtifactRecord(record); err != nil {
		return spec.ArtifactRecord{}, err
	}
	if err := s.repository.UpdateRecord(ctx, record); err != nil {
		return spec.ArtifactRecord{}, err
	}
	return record, nil
}
