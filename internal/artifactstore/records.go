package artifactstore

import (
	"context"
	"fmt"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
)

func (s *Store) CreateRecord(ctx context.Context, draft spec.ArtifactRecordDraft) (spec.ArtifactRecord, error) {
	if err := s.ensureOpen(); err != nil {
		return spec.ArtifactRecord{}, err
	}
	resource, definition, digest, err := s.resolveRecordTarget(ctx, draft)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	id, err := s.newID()
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	now := s.nowUTC()
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
		PinnedDefinitionDigest:       draft.PinnedDefinitionDigest,
		LastResolvedDefinitionDigest: &digest,
		Enabled:                      draft.Enabled,
		DataSchemaID:                 draft.DataSchemaID,
		Data:                         normalizedJSONObject(draft.Data),
		State:                        spec.RecordStateAvailable,
		CreatedAt:                    now,
		ModifiedAt:                   now,
	}
	if err := s.validateRecord(ctx, &record, resource, definition); err != nil {
		return spec.ArtifactRecord{}, err
	}
	if err := s.repository.CreateRecord(ctx, record); err != nil {
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
	resource, err := s.repository.GetCatalogResource(
		ctx,
		spec.CatalogResourceKey{
			SourceID:           record.SourceID,
			Locator:            record.Locator,
			SubresourceLocator: record.SubresourceLocator,
		},
	)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	definition, err := s.GetDefinitionByDigest(ctx, *record.LastResolvedDefinitionDigest)
	if err != nil {
		return spec.ArtifactRecord{}, err
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
	resource, err := s.repository.GetCatalogResource(
		ctx,
		spec.CatalogResourceKey{
			SourceID:           record.SourceID,
			Locator:            record.Locator,
			SubresourceLocator: record.SubresourceLocator,
		},
	)
	//nolint:gocritic // Don't want switch.
	if err != nil {
		record.State = spec.RecordStateMissing
	} else if resource.State == spec.CatalogStateMissing {
		record.State = spec.RecordStateMissing
	} else if resource.State != spec.CatalogStateValid {
		record.State = spec.RecordStateInvalid
	} else if resource.Kind != record.Kind {
		record.State = spec.RecordStateIncompatible
	} else {
		switch record.TrackingMode {
		case spec.TrackingModeFollowSource:
			record.LastResolvedDefinitionDigest = resource.CurrentDefinitionDigest
			record.State = spec.RecordStateAvailable
		case spec.TrackingModePinDigest:
			record.LastResolvedDefinitionDigest = record.PinnedDefinitionDigest
			record.State = spec.RecordStateAvailable
		case spec.TrackingModeManualRefresh:
			if resource.CurrentDefinitionDigest != nil && record.LastResolvedDefinitionDigest != nil &&
				*resource.CurrentDefinitionDigest != *record.LastResolvedDefinitionDigest {
				record.State = spec.RecordStateStale
			} else {
				record.State = spec.RecordStateAvailable
			}
		}
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

func (s *Store) PinRecord(
	ctx context.Context,
	recordID spec.RecordID,
	digest spec.Digest,
) (spec.ArtifactRecord, error) {
	if err := s.ensureOpen(); err != nil {
		return spec.ArtifactRecord{}, err
	}
	if _, err := s.GetDefinitionByDigest(ctx, digest); err != nil {
		return spec.ArtifactRecord{}, err
	}
	record, err := s.repository.GetRecord(ctx, recordID)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	record.TrackingMode = spec.TrackingModePinDigest
	record.PinnedDefinitionDigest = &digest
	record.LastResolvedDefinitionDigest = &digest
	if record.State != spec.RecordStateMissing {
		record.State = spec.RecordStateAvailable
	}
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
	record.ModifiedAt = s.nowUTC()
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
	if _, err := s.repository.GetSource(ctx, request.Record.SourceID); err != nil {
		return spec.ArtifactRecord{}, err
	}
	now := s.nowUTC()
	digest := definition.Digest
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
	if err := s.repository.UpsertCatalogResource(ctx, resource); err != nil {
		return spec.ArtifactRecord{}, err
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
	if err := s.repository.UpsertCatalogResourceRevision(ctx, revision); err != nil {
		return spec.ArtifactRecord{}, err
	}
	if request.Record.TrackingMode == "" {
		request.Record.TrackingMode = spec.TrackingModePinDigest
		request.Record.PinnedDefinitionDigest = &digest
	}
	request.Record.Kind = definition.Kind
	record, err := s.CreateRecord(ctx, request.Record)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	provenanceID, err := s.newID()
	if err == nil {
		_ = s.repository.CreateTransferProvenance(
			ctx,
			spec.TransferProvenance{
				ProvenanceID:           spec.ProvenanceID(provenanceID),
				TargetRecordID:         record.RecordID,
				Operation:              spec.TransferOperationImport,
				OriginDefinitionDigest: definition.Digest,
				CreatedAt:              now,
			},
		)
	}
	return record, nil
}

func (s *Store) resolveRecordTarget(
	ctx context.Context,
	draft spec.ArtifactRecordDraft,
) (spec.CatalogResource, spec.CanonicalDefinition, spec.Digest, error) {
	if _, err := s.repository.GetRoot(ctx, draft.RootID, false); err != nil {
		return spec.CatalogResource{}, spec.CanonicalDefinition{}, "", err
	}
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
	return resource, definition, digest, nil
}

func (s *Store) validateRecord(
	ctx context.Context,
	record *spec.ArtifactRecord,
	resource spec.CatalogResource,
	definition spec.CanonicalDefinition,
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
	if frontend, ok := s.frontendFor(resource.FrontendID); ok {
		diagnostics := frontend.ValidateRecordData(
			ctx,
			definition,
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
	record.ModifiedAt = time.Now().UTC()
	if err := s.repository.UpdateRecord(ctx, record); err != nil {
		return spec.ArtifactRecord{}, err
	}
	return record, nil
}
