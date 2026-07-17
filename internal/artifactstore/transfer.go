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
	definition, err := baseutils.CanonicalizeDefinition(request.File.Definition)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	file := spec.ArtifactDefinitionFile{
		Format:     request.File.Format,
		Definition: definition,
	}
	if err := validate.ValidateArtifactDefinitionFile(file); err != nil {
		return spec.ArtifactRecord{}, fmt.Errorf(
			"%w: imported definition: %w",
			spec.ErrInvalidRequest,
			err,
		)
	}
	assets, err := validateImportedAssets(definition.AssetManifest, request.Assets)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	for _, asset := range assets {
		digest, size, err := s.portableContent.PutAsset(ctx, asset.Content)
		if err != nil {
			return spec.ArtifactRecord{}, err
		}
		if digest != asset.Manifest.Digest || size != asset.Manifest.SizeBytes {
			return spec.ArtifactRecord{}, fmt.Errorf(
				"%w: imported asset %q changed while being persisted",
				spec.ErrDigestMismatch,
				asset.Manifest.Path,
			)
		}
	}
	definition, err = s.portableContent.PutDefinition(ctx, file)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	file.Definition = definition
	payload := spec.DefinitionTransferPayload{
		RootDefinitionDigest: definition.Digest,
		Definitions:          []spec.ArtifactDefinitionFile{file},
		Assets:               assets,
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

func validateImportedAssets(
	manifest []spec.AssetManifestEntry,
	assets []spec.PortableAssetContent,
) ([]spec.PortableAssetContent, error) {
	if len(manifest) != len(assets) {
		return nil, fmt.Errorf(
			"%w: imported definition declares %d assets but request contains %d",
			spec.ErrInvalidRequest,
			len(manifest),
			len(assets),
		)
	}
	expected := make(map[spec.PortablePath]spec.AssetManifestEntry, len(manifest))
	for _, entry := range manifest {
		expected[entry.Path] = entry
	}
	out := make([]spec.PortableAssetContent, 0, len(assets))
	seen := make(map[spec.PortablePath]struct{}, len(assets))
	var totalBytes int64
	for _, asset := range assets {
		if err := validate.ValidateAssetManifestEntry(asset.Manifest); err != nil {
			return nil, fmt.Errorf("%w: imported asset: %w", spec.ErrInvalidRequest, err)
		}
		if _, duplicate := seen[asset.Manifest.Path]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate imported asset path %q",
				spec.ErrInvalidRequest,
				asset.Manifest.Path,
			)
		}
		seen[asset.Manifest.Path] = struct{}{}
		declared, ok := expected[asset.Manifest.Path]
		if !ok || declared != asset.Manifest {
			return nil, fmt.Errorf(
				"%w: imported asset %q does not match the definition manifest",
				spec.ErrInvalidRequest,
				asset.Manifest.Path,
			)
		}
		if int64(len(asset.Content)) != asset.Manifest.SizeBytes ||
			baseutils.DigestBytes(asset.Content) != asset.Manifest.Digest {
			return nil, fmt.Errorf(
				"%w: imported asset %q content does not match its manifest",
				spec.ErrDigestMismatch,
				asset.Manifest.Path,
			)
		}
		if int64(len(asset.Content)) > spec.MaxTransferPayloadBytes-totalBytes {
			return nil, fmt.Errorf(
				"%w: imported assets exceed %d bytes",
				spec.ErrInvalidRequest,
				spec.MaxTransferPayloadBytes,
			)
		}
		totalBytes += int64(len(asset.Content))
		out = append(out, spec.PortableAssetContent{
			Manifest: asset.Manifest,
			Content:  append([]byte(nil), asset.Content...),
		})
	}
	return out, nil
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
		if len(exported.Definition.Definition.AssetManifest) != 0 {
			return spec.DefinitionTransferPayload{}, fmt.Errorf(
				"%w: definition-only transfer cannot omit declared assets",
				spec.ErrInvalidRequest,
			)
		}
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
	expectedAssets := make(map[spec.PortablePath]spec.AssetManifestEntry)
	for _, digest := range digests {
		definition, err := s.GetDefinitionByDigest(ctx, digest)
		if err != nil {
			return spec.DefinitionTransferPayload{}, err
		}
		payload.Definitions = append(payload.Definitions, spec.ArtifactDefinitionFile{
			Format:     spec.ArtifactDefinitionFileFormatV1,
			Definition: definition,
		})
		for _, asset := range definition.AssetManifest {
			if previous, exists := expectedAssets[asset.Path]; exists && previous != asset {
				return spec.DefinitionTransferPayload{}, fmt.Errorf(
					"%w: definitions declare conflicting asset metadata for %q",
					spec.ErrInvalidRequest,
					asset.Path,
				)
			}
			expectedAssets[asset.Path] = asset
		}
	}
	if len(expectedAssets) != len(exported.Closure.Assets) {
		return spec.DefinitionTransferPayload{}, fmt.Errorf(
			"%w: export closure does not contain the exact declared asset set",
			spec.ErrInvalidRequest,
		)
	}

	seenPaths := make(map[spec.PortablePath]struct{}, len(exported.Closure.Assets))
	var totalBytes int64
	for _, manifest := range exported.Closure.Assets {
		if err := validate.ValidateAssetManifestEntry(manifest); err != nil {
			return spec.DefinitionTransferPayload{}, err
		}
		expected, exists := expectedAssets[manifest.Path]
		if !exists || expected != manifest {
			return spec.DefinitionTransferPayload{}, fmt.Errorf(
				"%w: export asset %q is not declared by the transferred definitions",
				spec.ErrInvalidRequest,
				manifest.Path,
			)
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
	s.scanMu.Lock()
	defer s.scanMu.Unlock()

	if err := validateDefinitionTransferPayload(payload, definition.Digest); err != nil {
		return spec.ArtifactRecord{}, err
	}
	root, err := s.repository.GetRoot(ctx, destination.RootID, false)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	if !root.Enabled {
		return spec.ArtifactRecord{}, fmt.Errorf(
			"%w: destination root %q is disabled",
			spec.ErrConflict,
			destination.RootID,
		)
	}
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
	attachment, err := s.repository.GetRootSourceAttachment(
		ctx,
		destination.RootID,
		destination.SourceID,
	)
	if err != nil {
		return spec.ArtifactRecord{}, err
	}
	if !attachment.Enabled {
		return spec.ArtifactRecord{}, fmt.Errorf(
			"%w: destination source %q is disabled for root %q",
			spec.ErrConflict,
			source.SourceID,
			destination.RootID,
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
	if _, ok := s.frontendFor(frontendID); !ok {
		return spec.ArtifactRecord{}, fmt.Errorf(
			"%w: destination frontend %q",
			spec.ErrFrontendUnavailable,
			frontendID,
		)
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
		Resource:                          resource,
		Revision:                          revision,
		Record:                            record,
		Provenance:                        provenance,
		ExpectedSourceModifiedAt:          source.ModifiedAt,
		ExpectedSourceObservationRevision: source.ObservationRevision,
		ExpectedAttachmentModifiedAt:      attachment.ModifiedAt,
		ExpectedRootRevision:              root.MountRevision,
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

func validateDefinitionTransferPayload(
	payload spec.DefinitionTransferPayload,
	rootDigest spec.Digest,
) error {
	if payload.RootDefinitionDigest != rootDigest {
		return fmt.Errorf(
			"%w: transfer root digest does not match the target definition",
			spec.ErrDigestMismatch,
		)
	}
	if len(payload.Definitions) == 0 ||
		len(payload.Definitions) > spec.MaxPortablePackageDefinitions {
		return fmt.Errorf(
			"%w: transfer definition count is invalid",
			spec.ErrInvalidRequest,
		)
	}
	seenDefinitions := make(map[spec.Digest]struct{}, len(payload.Definitions))
	expectedAssets := make(map[spec.PortablePath]spec.AssetManifestEntry)
	rootSeen := false
	for _, file := range payload.Definitions {
		if err := validate.ValidateArtifactDefinitionFile(file); err != nil {
			return fmt.Errorf("%w: transfer definition: %w", spec.ErrInvalidRequest, err)
		}
		canonical, err := baseutils.CanonicalizeDefinition(file.Definition)
		if err != nil {
			return err
		}
		if _, duplicate := seenDefinitions[canonical.Digest]; duplicate {
			return fmt.Errorf(
				"%w: duplicate transfer definition %q",
				spec.ErrInvalidRequest,
				canonical.Digest,
			)
		}
		seenDefinitions[canonical.Digest] = struct{}{}
		rootSeen = rootSeen || canonical.Digest == rootDigest
		for _, asset := range canonical.AssetManifest {
			if previous, exists := expectedAssets[asset.Path]; exists && previous != asset {
				return fmt.Errorf(
					"%w: transferred definitions declare conflicting asset metadata for %q",
					spec.ErrInvalidRequest,
					asset.Path,
				)
			}
			expectedAssets[asset.Path] = asset
		}
	}
	if !rootSeen {
		return fmt.Errorf(
			"%w: transfer payload does not contain its root definition",
			spec.ErrInvalidRequest,
		)
	}
	var totalBytes int64
	seenAssets := make(map[spec.PortablePath]struct{}, len(payload.Assets))
	if len(expectedAssets) != len(payload.Assets) {
		return fmt.Errorf(
			"%w: transfer payload does not contain the exact declared asset set",
			spec.ErrInvalidRequest,
		)
	}
	for _, asset := range payload.Assets {
		if err := validate.ValidateAssetManifestEntry(asset.Manifest); err != nil {
			return fmt.Errorf("%w: transfer asset: %w", spec.ErrInvalidRequest, err)
		}
		if _, duplicate := seenAssets[asset.Manifest.Path]; duplicate {
			return fmt.Errorf(
				"%w: duplicate transfer asset path %q",
				spec.ErrInvalidRequest,
				asset.Manifest.Path,
			)
		}
		seenAssets[asset.Manifest.Path] = struct{}{}
		expected, exists := expectedAssets[asset.Manifest.Path]
		if !exists || expected != asset.Manifest {
			return fmt.Errorf(
				"%w: transfer asset %q is not declared by a transferred definition",
				spec.ErrInvalidRequest,
				asset.Manifest.Path,
			)
		}
		if int64(len(asset.Content)) != asset.Manifest.SizeBytes ||
			baseutils.DigestBytes(asset.Content) != asset.Manifest.Digest {
			return fmt.Errorf(
				"%w: transfer asset %q does not match its manifest",
				spec.ErrDigestMismatch,
				asset.Manifest.Path,
			)
		}
		if int64(len(asset.Content)) > spec.MaxTransferPayloadBytes-totalBytes {
			return fmt.Errorf(
				"%w: transfer payload exceeds %d bytes",
				spec.ErrInvalidRequest,
				spec.MaxTransferPayloadBytes,
			)
		}
		totalBytes += int64(len(asset.Content))
	}
	return nil
}
