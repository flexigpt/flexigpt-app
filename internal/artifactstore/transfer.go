package artifactstore

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/baseutils"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/validate"
)

func (s *Store) ImportDefinition(
	ctx context.Context,
	request spec.ImportDefinitionRequest,
) (spec.ArtifactRecord, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	defer finish()

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
	file := spec.ArtifactDefinitionFile{
		Format:     spec.ArtifactDefinitionFileFormatV1,
		Definition: definition,
	}
	payload := spec.DefinitionTransferPayload{
		RootDefinitionDigest: definition.Digest,
		Definitions:          []spec.ArtifactDefinitionFile{file},
	}
	return s.publishTransferredRecord(
		ctx,
		spec.TransferOperationImport,
		nil,
		request.Destination,
		payload,
		definition,
	)
}

func (s *Store) CaptureRecord(
	ctx context.Context,
	request spec.CaptureRecordRequest,
) (spec.ArtifactRecord, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	defer finish()

	exported, err := s.exportRecord(ctx, request.OriginRecordID)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	mode := request.MaterializationMode
	if mode == "" {
		mode = spec.TransferMaterializeExportClosure
	}
	payload, err := s.buildTransferPayload(ctx, exported, mode)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	return s.publishTransferredRecord(
		ctx,
		spec.TransferOperationCapture,
		&exported.Record,
		request.Destination,
		payload,
		exported.Definition.Definition,
	)
}

func (s *Store) ForkRecord(
	ctx context.Context,
	request spec.ForkRecordRequest,
) (spec.ArtifactRecord, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	defer finish()

	exported, err := s.exportRecord(ctx, request.OriginRecordID)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	mode := request.MaterializationMode
	if mode == "" {
		mode = spec.TransferMaterializeExportClosure
	}
	payload, err := s.buildTransferPayload(ctx, exported, mode)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	return s.publishTransferredRecord(
		ctx,
		spec.TransferOperationFork,
		&exported.Record,
		request.Destination,
		payload,
		exported.Definition.Definition,
	)
}

func (s *Store) buildTransferPayload(
	ctx context.Context,
	exported spec.ExportedRecord,
	mode spec.TransferMaterializationMode,
) (spec.DefinitionTransferPayload, error) {
	switch mode {
	case spec.TransferMaterializeDefinitionOnly:
		return spec.DefinitionTransferPayload{
			RootDefinitionDigest: exported.Definition.Definition.Digest,
			Definitions:          []spec.ArtifactDefinitionFile{exported.Definition},
		}, nil
	case spec.TransferMaterializeExportClosure:
	default:
		return spec.DefinitionTransferPayload{}, fmt.Errorf(
			"%w: unsupported transfer materialization mode %q",
			spec.ErrInvalidRequest,
			mode,
		)
	}

	rootDigest := exported.Definition.Definition.Digest
	digestSet := map[spec.Digest]struct{}{rootDigest: {}}
	for _, digest := range exported.Closure.DefinitionDigests {
		digestSet[digest] = struct{}{}
	}
	digests := make([]spec.Digest, 0, len(digestSet))
	for digest := range digestSet {
		digests = append(digests, digest)
	}
	slices.Sort(digests)

	payload := spec.DefinitionTransferPayload{
		RootDefinitionDigest: rootDigest,
		Definitions:          make([]spec.ArtifactDefinitionFile, 0, len(digests)),
	}
	for _, digest := range digests {
		definition, err := s.GetDefinitionByDigest(ctx, digest)
		if err != nil {
			return spec.DefinitionTransferPayload{}, err
		}
		payload.Definitions = append(payload.Definitions, spec.ArtifactDefinitionFile{
			Format:     spec.ArtifactDefinitionFileFormatV1,
			Definition: definition,
		})
	}

	seenPaths := make(map[spec.PortablePath]struct{}, len(exported.Closure.Assets))
	var totalBytes int64
	for _, manifest := range exported.Closure.Assets {
		if err := validate.ValidateAssetManifestEntry(manifest); err != nil {
			return spec.DefinitionTransferPayload{}, err
		}
		if _, exists := seenPaths[manifest.Path]; exists {
			return spec.DefinitionTransferPayload{}, fmt.Errorf(
				"%w: export closure contains duplicate asset path %q",
				spec.ErrInvalidRequest,
				manifest.Path,
			)
		}
		seenPaths[manifest.Path] = struct{}{}
		content, err := s.portableContent.GetAsset(ctx, manifest.Digest)
		if err != nil {
			return spec.DefinitionTransferPayload{}, err
		}
		if int64(len(content)) != manifest.SizeBytes ||
			baseutils.DigestBytes(content) != manifest.Digest {
			return spec.DefinitionTransferPayload{}, fmt.Errorf(
				"%w: export asset %q does not match its manifest",
				spec.ErrDigestMismatch,
				manifest.Path,
			)
		}
		totalBytes += int64(len(content))
		if totalBytes > spec.MaxTransferPayloadBytes {
			return spec.DefinitionTransferPayload{}, fmt.Errorf(
				"%w: transfer payload exceeds %d bytes",
				spec.ErrInvalidRequest,
				spec.MaxTransferPayloadBytes,
			)
		}
		payload.Assets = append(payload.Assets, spec.PortableAssetContent{
			Manifest: manifest,
			Content:  append([]byte(nil), content...),
		})
	}
	return payload, nil
}

func (s *Store) publishTransferredRecord(
	ctx context.Context,
	operation spec.TransferOperation,
	origin *spec.ArtifactRecord,
	destination spec.TransferDestination,
	payload spec.DefinitionTransferPayload,
	definition spec.CanonicalDefinition,
) (spec.ArtifactRecord, error) {
	source, err := s.repository.GetSource(ctx, destination.SourceID)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	if !source.Enabled {
		return spec.ArtifactRecord{}, fmt.Errorf(
			"%w: destination source %q is disabled",
			spec.ErrConflict,
			source.SourceID,
		)
	}
	materializer, ok := s.definitionMaterializerFor(source.Kind)
	if !ok {
		return spec.ArtifactRecord{}, fmt.Errorf(
			"%w: no definition materializer for source kind %q",
			spec.ErrMaterializerUnavailable,
			source.Kind,
		)
	}

	frontendID := destination.FrontendID
	if frontendID == "" {
		frontendID = spec.PortableDefinitionFrontendID
	}
	if frontendID == spec.PortableDefinitionFrontendID &&
		destination.SubresourceLocator != "" {
		return spec.ArtifactRecord{}, fmt.Errorf(
			"%w: portable definition destinations cannot declare a subresource locator",
			spec.ErrInvalidRequest,
		)
	}

	key := spec.CatalogResourceKey{
		SourceID:           destination.SourceID,
		Locator:            destination.Locator,
		SubresourceLocator: destination.SubresourceLocator,
	}
	if _, err := s.repository.GetCatalogResource(ctx, key); err == nil {
		return spec.ArtifactRecord{}, fmt.Errorf(
			"%w: destination catalog occurrence already exists",
			spec.ErrConflict,
		)
	} else if !isNotFound(err) {
		return spec.ArtifactRecord{}, err
	}

	now := s.nowUTC()
	definitionDigest := definition.Digest
	provisionalSourceDigest := definition.Digest
	resource := spec.CatalogResource{
		SourceID:                destination.SourceID,
		Locator:                 destination.Locator,
		SubresourceLocator:      destination.SubresourceLocator,
		PackageManifestLocator:  destination.PackageManifestLocator,
		Kind:                    definition.Kind,
		LogicalName:             definition.LogicalName,
		LogicalVersion:          definition.LogicalVersion,
		CurrentDefinitionDigest: &definitionDigest,
		SourceContentDigest:     &provisionalSourceDigest,
		FrontendID:              frontendID,
		State:                   spec.CatalogStateValid,
		FirstSeenAt:             now,
		LastSeenAt:              now,
	}

	recordMode := spec.RecordModeCaptured
	if operation == spec.TransferOperationFork {
		recordMode = spec.RecordModeForked
	}
	record, err := s.prepareRecordForResolved(
		ctx,
		spec.ArtifactRecordDraft{
			RootID:                 destination.RootID,
			CollectionID:           destination.CollectionID,
			Kind:                   definition.Kind,
			Name:                   destination.Name,
			Version:                destination.Version,
			SourceID:               destination.SourceID,
			Locator:                destination.Locator,
			SubresourceLocator:     destination.SubresourceLocator,
			RecordMode:             recordMode,
			TrackingMode:           spec.TrackingModePinDigest,
			PinnedDefinitionDigest: &definitionDigest,
			Enabled:                destination.Enabled,
			DataSchemaID:           destination.DataSchemaID,
			Data:                   destination.Data,
		},
		resource,
		definition,
		definitionDigest,
	)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}

	materialized, err := materializer.MaterializeDefinition(
		ctx,
		spec.DefinitionMaterializationRequest{
			Source:      source,
			Destination: destination,
			Payload:     payload,
			Exclusive:   true,
		},
	)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	if strings.TrimSpace(materialized.Receipt) == "" {
		return spec.ArtifactRecord{}, fmt.Errorf(
			"%w: definition materializer returned an empty receipt",
			spec.ErrMaterializerUnavailable,
		)
	}
	sourceDigest := materialized.SourceContentDigest
	resource.SourceContentDigest = &sourceDigest
	if err := validate.ValidateCatalogResource(resource); err != nil {
		discardErr := materializer.DiscardDefinition(
			context.WithoutCancel(ctx),
			source,
			materialized.Receipt,
		)
		return spec.ArtifactRecord{}, errors.Join(err, discardErr)
	}

	revision := spec.CatalogResourceRevision{
		SourceID:            resource.SourceID,
		Locator:             resource.Locator,
		SubresourceLocator:  resource.SubresourceLocator,
		DefinitionDigest:    definitionDigest,
		SourceContentDigest: sourceDigest,
		Kind:                definition.Kind,
		FrontendID:          frontendID,
		FirstSeenAt:         now,
		LastSeenAt:          now,
	}
	provenanceID, err := s.newID()
	if err != nil {
		discardErr := materializer.DiscardDefinition(
			context.WithoutCancel(ctx),
			source,
			materialized.Receipt,
		)
		return spec.ArtifactRecord{}, errors.Join(err, discardErr)
	}
	provenance := spec.TransferProvenance{
		ProvenanceID:           spec.ProvenanceID(provenanceID),
		TargetRecordID:         record.RecordID,
		Operation:              operation,
		OriginDefinitionDigest: definitionDigest,
		CreatedAt:              record.CreatedAt,
	}
	if origin != nil {
		originRecordID := origin.RecordID
		provenance.OriginRecordID = &originRecordID
		provenance.OriginResource = &spec.CatalogResourceKey{
			SourceID:           origin.SourceID,
			Locator:            origin.Locator,
			SubresourceLocator: origin.SubresourceLocator,
		}
	}

	if err := s.repository.PublishRecordTransfer(ctx, spec.RecordTransferPublication{
		Resource:   resource,
		Revision:   revision,
		Record:     record,
		Provenance: provenance,
	}); err != nil {
		discardErr := materializer.DiscardDefinition(
			context.WithoutCancel(ctx),
			source,
			materialized.Receipt,
		)
		if discardErr != nil {
			return spec.ArtifactRecord{}, errors.Join(
				err,
				fmt.Errorf("discard failed transfer publication: %w", discardErr),
			)
		}
		return spec.ArtifactRecord{}, err
	}
	return record, nil
}
