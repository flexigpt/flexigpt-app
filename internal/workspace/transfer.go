package workspace

import (
	"context"
	"fmt"

	artifactstoreSpec "github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
)

func (s *Service) ExportRecord(
	ctx context.Context,
	recordID artifactstoreSpec.RecordID,
) (artifactstoreSpec.ExportedRecord, error) {
	record, err := s.store.GetRecord(ctx, recordID)
	if err != nil {
		return artifactstoreSpec.ExportedRecord{}, err
	}
	if _, err := s.GetWorkspace(ctx, record.RootID); err != nil {
		return artifactstoreSpec.ExportedRecord{}, err
	}
	return s.store.ExportRecord(ctx, recordID)
}

func (s *Service) ImportDefinition(
	ctx context.Context,
	request artifactstoreSpec.ImportDefinitionRequest,
	discoverImmediately bool,
) (TransferResult, error) {
	destination, err := s.prepareTransferDestination(
		ctx,
		request.Destination,
		request.File.Definition,
	)
	if err != nil {
		return TransferResult{}, err
	}
	request.Destination = destination
	record, err := s.store.ImportDefinition(ctx, request)
	if err != nil {
		return TransferResult{}, err
	}
	return s.finishTransfer(ctx, record, discoverImmediately)
}

func (s *Service) CaptureRecord(
	ctx context.Context,
	request artifactstoreSpec.CaptureRecordRequest,
	discoverImmediately bool,
) (TransferResult, error) {
	origin, err := s.store.GetRecord(ctx, request.OriginRecordID)
	if err != nil {
		return TransferResult{}, err
	}
	if _, err := s.GetWorkspace(ctx, origin.RootID); err != nil {
		return TransferResult{}, err
	}
	if origin.LastResolvedDefinitionDigest == nil {
		return TransferResult{}, fmt.Errorf(
			"%w: origin record has no resolved definition",
			artifactstoreSpec.ErrConflict,
		)
	}
	definition, err := s.store.GetDefinitionByDigest(ctx, *origin.LastResolvedDefinitionDigest)
	if err != nil {
		return TransferResult{}, err
	}
	request.Destination, err = s.prepareTransferDestination(ctx, request.Destination, definition)
	if err != nil {
		return TransferResult{}, err
	}
	record, err := s.store.CaptureRecord(ctx, request)
	if err != nil {
		return TransferResult{}, err
	}
	return s.finishTransfer(ctx, record, discoverImmediately)
}

func (s *Service) ForkRecord(
	ctx context.Context,
	request artifactstoreSpec.ForkRecordRequest,
	discoverImmediately bool,
) (TransferResult, error) {
	origin, err := s.store.GetRecord(ctx, request.OriginRecordID)
	if err != nil {
		return TransferResult{}, err
	}
	if _, err := s.GetWorkspace(ctx, origin.RootID); err != nil {
		return TransferResult{}, err
	}
	if origin.LastResolvedDefinitionDigest == nil {
		return TransferResult{}, fmt.Errorf(
			"%w: origin record has no resolved definition",
			artifactstoreSpec.ErrConflict,
		)
	}
	definition, err := s.store.GetDefinitionByDigest(ctx, *origin.LastResolvedDefinitionDigest)
	if err != nil {
		return TransferResult{}, err
	}
	request.Destination, err = s.prepareTransferDestination(ctx, request.Destination, definition)
	if err != nil {
		return TransferResult{}, err
	}
	record, err := s.store.ForkRecord(ctx, request)
	if err != nil {
		return TransferResult{}, err
	}
	return s.finishTransfer(ctx, record, discoverImmediately)
}

func (s *Service) prepareTransferDestination(
	ctx context.Context,
	destination artifactstoreSpec.TransferDestination,
	definition artifactstoreSpec.CanonicalDefinition,
) (artifactstoreSpec.TransferDestination, error) {
	if err := s.validateTransferDestination(ctx, destination); err != nil {
		return artifactstoreSpec.TransferDestination{}, err
	}
	if _, known := s.descriptors[definition.Kind]; !known {
		return artifactstoreSpec.TransferDestination{}, fmt.Errorf(
			"%w: artifact kind %q is not registered",
			ErrProjectionUnavailable,
			definition.Kind,
		)
	}
	if destination.CollectionID == nil {
		collectionID, err := s.ensureCollectionForKind(ctx, destination.RootID, definition.Kind)
		if err != nil {
			return artifactstoreSpec.TransferDestination{}, err
		}
		destination.CollectionID = &collectionID
	}
	if destination.Name == "" {
		destination.Name = workspaceRecordName(
			definition.LogicalName,
			destination.SourceID,
			destination.Locator,
			destination.SubresourceLocator,
		)
	}
	if destination.Version == "" {
		destination.Version = artifactstoreSpec.RecordVersion(definition.LogicalVersion)
	}
	return destination, nil
}

func (s *Service) validateTransferDestination(
	ctx context.Context,
	destination artifactstoreSpec.TransferDestination,
) error {
	if destination.RootID == "" ||
		destination.SourceID == "" ||
		destination.Locator == "" {
		return fmt.Errorf(
			"%w: destination rootID, sourceID, and locator are required",
			ErrInvalidWorkspace,
		)
	}
	workspace, err := s.GetWorkspace(ctx, destination.RootID)
	if err != nil {
		return err
	}
	for _, attachment := range workspace.Attachments {
		if attachment.SourceID == destination.SourceID && attachment.Enabled {
			return nil
		}
	}
	return fmt.Errorf(
		"%w: destination source %q is not enabled for workspace %q",
		artifactstoreSpec.ErrSourceNotAttached,
		destination.SourceID,
		destination.RootID,
	)
}

func (s *Service) finishTransfer(
	ctx context.Context,
	record artifactstoreSpec.ArtifactRecord,
	discoverImmediately bool,
) (TransferResult, error) {
	result := TransferResult{Record: record}
	if !discoverImmediately {
		return result, nil
	}
	refreshed, err := s.Refresh(ctx, record.RootID)
	result.Refresh = &refreshed
	if err != nil {
		return result, err
	}
	return result, nil
}
