package artifactstore

import (
	"context"
	"fmt"
	"reflect"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/validate"
)

// SyncRecords synchronizes root-local records with the currently published
// root catalog. It never automatically deletes a record.
func (s *Store) SyncRecords(
	ctx context.Context,
	rootID spec.RootID,
	policy spec.RecordSyncPolicy,
) (spec.RecordSyncResult, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return spec.RecordSyncResult{}, err
	}
	defer finish()
	if policy == nil {
		return spec.RecordSyncResult{}, fmt.Errorf(
			"%w: record synchronization policy is nil",
			spec.ErrInvalidRequest,
		)
	}
	root, err := s.repository.GetRoot(ctx, rootID, false)
	if err != nil {
		return spec.RecordSyncResult{}, err
	}
	generation, err := s.repository.GetRootCatalogGeneration(ctx, rootID)
	if err != nil {
		return spec.RecordSyncResult{}, err
	}
	resources, err := s.repository.ListPublishedCatalogResourcesForRoot(ctx, rootID)
	if err != nil {
		return spec.RecordSyncResult{}, err
	}
	records, err := s.repository.ListRecordsForRoot(ctx, rootID)
	if err != nil {
		return spec.RecordSyncResult{}, err
	}

	resourcesByOccurrence := make(map[string]spec.CatalogResource, len(resources))
	for _, resource := range resources {
		resourcesByOccurrence[recordOccurrenceKey(
			resource.SourceID,
			resource.Locator,
			resource.SubresourceLocator,
		)] = resource
	}

	existingByTypedOccurrence := make(map[string]struct{}, len(records))
	publication := spec.RecordSynchronizationPublication{
		RootID:                    rootID,
		ExpectedCatalogGeneration: generation.Generation,
	}
	result := spec.RecordSyncResult{RootID: rootID}

	for _, existing := range records {
		existingByTypedOccurrence[typedRecordOccurrenceKey(
			existing.SourceID,
			existing.Locator,
			existing.SubresourceLocator,
			existing.Kind,
		)] = struct{}{}

		resource, found := resourcesByOccurrence[recordOccurrenceKey(
			existing.SourceID,
			existing.Locator,
			existing.SubresourceLocator,
		)]
		var resourcePointer *spec.CatalogResource
		if found {
			resourceCopy := resource
			resourcePointer = &resourceCopy
		}

		updated := existing
		if !applyCatalogResourceToRecord(&updated, resourcePointer, false) {
			continue
		}
		updated.ModifiedAt = s.nextModifiedAt(existing.ModifiedAt)
		var definitionPointer *spec.CanonicalDefinition
		if updated.LastResolvedDefinitionDigest != nil {
			definition, err := s.GetDefinitionByDigest(
				ctx,
				*updated.LastResolvedDefinitionDigest,
			)
			if err != nil {
				return spec.RecordSyncResult{}, err
			}
			definitionPointer = &definition
		}
		if err := s.validateRecord(ctx, &updated, resourcePointer, definitionPointer); err != nil {
			return spec.RecordSyncResult{}, fmt.Errorf(
				"validate synchronized record %q: %w",
				existing.RecordID,
				err,
			)
		}
		publication.Updates = append(publication.Updates, spec.RecordSynchronizationUpdate{
			Record:               updated,
			ExpectedModifiedAt:   existing.ModifiedAt,
			ExpectedRecordMode:   existing.RecordMode,
			ExpectedTrackingMode: existing.TrackingMode,
		})
		result.Updated = append(result.Updated, existing.RecordID)
	}

	for _, resource := range resources {
		if resource.State != spec.CatalogStateValid || resource.CurrentDefinitionDigest == nil {
			continue
		}
		typedKey := typedRecordOccurrenceKey(
			resource.SourceID,
			resource.Locator,
			resource.SubresourceLocator,
			resource.Kind,
		)
		if _, exists := existingByTypedOccurrence[typedKey]; exists {
			continue
		}

		definition, err := s.GetDefinitionByDigest(ctx, *resource.CurrentDefinitionDigest)
		if err != nil {
			return spec.RecordSyncResult{}, err
		}
		derivation, create, diagnostics := policy.DeriveRecord(ctx, root, resource, definition)
		if err := validate.ValidateDiagnostics(diagnostics); err != nil {
			return spec.RecordSyncResult{}, fmt.Errorf(
				"%w: synchronization policy diagnostics: %w",
				spec.ErrInvalidRequest,
				err,
			)
		}
		result.Diagnostics = append(result.Diagnostics, diagnostics...)
		if errorDiagnostics("record synchronization policy", diagnostics) != nil || !create {
			continue
		}

		draft := spec.ArtifactRecordDraft{
			RootID:             rootID,
			CollectionID:       derivation.CollectionID,
			Kind:               resource.Kind,
			Name:               derivation.Name,
			Version:            derivation.Version,
			SourceID:           resource.SourceID,
			Locator:            resource.Locator,
			SubresourceLocator: resource.SubresourceLocator,
			RecordMode:         spec.RecordModeLinked,
			TrackingMode:       spec.TrackingModeFollowSource,
			Enabled:            derivation.Enabled,
			DataSchemaID:       derivation.DataSchemaID,
			Data:               derivation.Data,
		}
		record, err := s.prepareRecordForResolved(
			ctx,
			draft,
			resource,
			definition,
			*resource.CurrentDefinitionDigest,
		)
		if err != nil {
			return spec.RecordSyncResult{}, err
		}
		publication.Creates = append(publication.Creates, record)
		result.Created = append(result.Created, record.RecordID)
		existingByTypedOccurrence[typedKey] = struct{}{}
	}

	if len(publication.Creates) == 0 && len(publication.Updates) == 0 {
		return result, nil
	}
	if err := s.repository.PublishRecordSynchronization(ctx, publication); err != nil {
		return spec.RecordSyncResult{}, err
	}
	return result, nil
}

func applyCatalogResourceToRecord(
	record *spec.ArtifactRecord,
	resource *spec.CatalogResource,
	explicitManualRefresh bool,
) bool {
	if record == nil {
		return false
	}
	before := *record
	before.Diagnostics = append([]spec.Diagnostic(nil), record.Diagnostics...)

	if isDetachedPinnedRecord(*record) {
		record.LastResolvedDefinitionDigest = cloneDigest(record.PinnedDefinitionDigest)
		record.State = spec.RecordStateAvailable
		record.Diagnostics = nil
		return !recordSourceStateEqual(before, *record)
	}

	if resource == nil || resource.State == spec.CatalogStateMissing {
		record.State = spec.RecordStateMissing
		record.Diagnostics = []spec.Diagnostic{{
			Severity: spec.DiagnosticSeverityWarning,
			Code:     "artifactstore.record.source-missing",
			Message:  "the source catalog occurrence is missing",
		}}
		return !recordSourceStateEqual(before, *record)
	}
	if resource.State == spec.CatalogStateInvalid || resource.CurrentDefinitionDigest == nil {
		record.State = spec.RecordStateInvalid
		record.Diagnostics = append([]spec.Diagnostic(nil), resource.Diagnostics...)
		if len(record.Diagnostics) == 0 {
			record.Diagnostics = []spec.Diagnostic{{
				Severity: spec.DiagnosticSeverityError,
				Code:     "artifactstore.record.source-invalid",
				Message:  "the source catalog occurrence is invalid",
			}}
		}
		return !recordSourceStateEqual(before, *record)
	}
	if resource.Kind != record.Kind {
		record.State = spec.RecordStateIncompatible
		record.Diagnostics = []spec.Diagnostic{{
			Severity: spec.DiagnosticSeverityError,
			Code:     "artifactstore.record.kind-incompatible",
			Message:  "the source catalog occurrence changed artifact kind",
		}}
		return !recordSourceStateEqual(before, *record)
	}

	record.Diagnostics = append([]spec.Diagnostic(nil), resource.Diagnostics...)
	switch record.TrackingMode {
	case spec.TrackingModeFollowSource:
		record.LastResolvedDefinitionDigest = cloneDigest(resource.CurrentDefinitionDigest)
		record.State = spec.RecordStateAvailable
	case spec.TrackingModePinDigest:
		record.LastResolvedDefinitionDigest = cloneDigest(record.PinnedDefinitionDigest)
		record.State = spec.RecordStateAvailable
	case spec.TrackingModeManualRefresh:
		//nolint:gocritic // Dont want switch.
		if explicitManualRefresh || record.LastResolvedDefinitionDigest == nil {
			record.LastResolvedDefinitionDigest = cloneDigest(resource.CurrentDefinitionDigest)
			record.State = spec.RecordStateAvailable
		} else if *record.LastResolvedDefinitionDigest != *resource.CurrentDefinitionDigest {
			record.State = spec.RecordStateStale
		} else {
			record.State = spec.RecordStateAvailable
		}
	}
	return !recordSourceStateEqual(before, *record)
}

func isDetachedPinnedRecord(record spec.ArtifactRecord) bool {
	if record.PinnedDefinitionDigest == nil {
		return false
	}
	switch record.RecordMode {
	case spec.RecordModeCaptured, spec.RecordModeForked, spec.RecordModeAppLocal:
		return true
	default:
		return false
	}
}

func recordSourceStateEqual(left, right spec.ArtifactRecord) bool {
	return reflect.DeepEqual(left.LastResolvedDefinitionDigest, right.LastResolvedDefinitionDigest) &&
		left.State == right.State &&
		reflect.DeepEqual(left.Diagnostics, right.Diagnostics)
}

func cloneDigest(value *spec.Digest) *spec.Digest {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func typedRecordOccurrenceKey(
	sourceID spec.SourceID,
	locator spec.SourceLocator,
	subresource spec.SubresourceLocator,
	kind spec.ArtifactKind,
) string {
	return recordOccurrenceKey(sourceID, locator, subresource) + "\x00" + string(kind)
}

func recordOccurrenceKey(
	sourceID spec.SourceID,
	locator spec.SourceLocator,
	subresource spec.SubresourceLocator,
) string {
	return string(sourceID) + "\x00" + string(locator) + "\x00" + string(subresource)
}
