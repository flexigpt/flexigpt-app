package refresh

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type Service struct {
	collections CollectionReader
	catalogs    catalog.Reader
	sources     source.Runtime
	artifacts   ArtifactReader
	discovery   *discovery.Engine
	definitions definition.Repository
	reconciler  *artifact.Reconciler
	publisher   Publisher
	clock       basespec.Clock
}

func NewService(
	collections CollectionReader,
	catalogs catalog.Reader,
	sources source.Runtime,
	artifacts ArtifactReader,
	discoveryEngine *discovery.Engine,
	definitions definition.Repository,
	reconciler *artifact.Reconciler,
	publisher Publisher,
	clock basespec.Clock,
) (*Service, error) {
	if collections == nil ||
		catalogs == nil ||
		sources == nil ||
		artifacts == nil ||
		discoveryEngine == nil ||
		definitions == nil ||
		reconciler == nil ||
		publisher == nil ||
		clock == nil {
		return nil, fmt.Errorf(
			"%w: refresh service dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	return &Service{
		collections: collections,
		catalogs:    catalogs,
		sources:     sources,
		artifacts:   artifacts,
		discovery:   discoveryEngine,
		definitions: definitions,
		reconciler:  reconciler,
		publisher:   publisher,
		clock:       clock,
	}, nil
}

func (s *Service) Refresh(
	ctx context.Context,
	ref basespec.CollectionRef,
	plan discovery.Plan,
	policy artifact.Policy,
) (Result, error) {
	if ctx == nil {
		return Result{}, fmt.Errorf("%w: refresh context is nil", basespec.ErrInvalid)
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if policy == nil {
		return Result{}, fmt.Errorf(
			"%w: artifact adoption policy is required",
			basespec.ErrInvalid,
		)
	}
	if err := ref.Validate(); err != nil {
		return Result{}, err
	}
	if err := plan.Validate(); err != nil {
		return Result{}, err
	}

	collectionValue, err := s.collections.Get(ctx, ref)
	if err != nil {
		return Result{}, err
	}
	if err := collectionValue.Validate(); err != nil {
		return Result{}, fmt.Errorf(
			"%w: collection reader returned an invalid collection: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if collectionValue.Ref() != ref {
		return Result{}, fmt.Errorf("%w: collection reader returned another collection", basespec.ErrInvalid)
	}
	if !collectionValue.Enabled {
		return Result{}, fmt.Errorf(
			"%w: collection %q is disabled",
			basespec.ErrConflict,
			ref.CollectionID,
		)
	}

	attachments, err := s.collections.ListAttachments(ctx, ref)
	if err != nil {
		return Result{}, err
	}

	plansBySource := plan.BySource()

	var previous catalog.Snapshot
	previous, err = catalog.ReadCurrent(ctx, s.catalogs, ref)
	hasPrevious := err == nil || errors.Is(err, basespec.ErrCatalogStale)
	if !hasPrevious &&
		!errors.Is(err, basespec.ErrCatalogUnavailable) {
		return Result{}, err
	}

	previousBySource := make(
		map[basespec.SourceID][]catalog.Occurrence,
	)
	for _, occurrence := range previous.Occurrences {
		previousBySource[occurrence.Key.SourceID] = append(
			previousBySource[occurrence.Key.SourceID],
			occurrence,
		)
	}

	expectedAttachmentRevisions := make(map[basespec.SourceID]uint64)
	expectedSourceRevisions := make(map[basespec.SourceID]uint64)
	sourceGenerations := make(map[basespec.SourceID]string)
	finalOccurrences := make([]catalog.Occurrence, 0)
	allDiagnostics := make([]diagnostic.Diagnostic, 0)
	discoveredDefinitions := make(map[cryptoutil.Digest]definition.Definition)

	snapshots := make([]source.Snapshot, 0)
	candidates := 0

	defer func() {
		// Early-return cleanup must not obscure the operation failure.
		_ = closeRefreshSnapshots(snapshots)
	}()

	sort.Slice(attachments, func(left, right int) bool {
		return attachments[left].SourceID < attachments[right].SourceID
	})

	for _, attachment := range attachments {
		if err := attachment.Validate(); err != nil {
			return Result{}, fmt.Errorf(
				"%w: collection reader returned an invalid attachment: %w",
				basespec.ErrInvalid,
				err,
			)
		}
		if attachment.RootID != ref.RootID ||
			attachment.CollectionID != ref.CollectionID {
			return Result{}, fmt.Errorf("%w: attachment belongs to another collection", basespec.ErrInvalid)
		}

		if _, duplicate := expectedAttachmentRevisions[attachment.SourceID]; duplicate {
			return Result{}, fmt.Errorf(
				"%w: collection reader returned duplicate attachment source %q",
				basespec.ErrInvalid,
				attachment.SourceID,
			)
		}
		expectedAttachmentRevisions[attachment.SourceID] = attachment.Revision

		sourceValue, err := s.sources.Get(ctx, ref.RootID, attachment.SourceID)
		if err != nil {
			return Result{}, err
		}
		if sourceValue.ID != attachment.SourceID ||
			sourceValue.RootID != ref.RootID {
			return Result{}, fmt.Errorf(
				"%w: source runtime returned a source that does not match attachment %q",
				basespec.ErrInvalid,
				attachment.SourceID,
			)
		}
		if err := sourceValue.Validate(); err != nil {
			return Result{}, fmt.Errorf(
				"%w: source runtime returned an invalid source: %w",
				basespec.ErrInvalid,
				err,
			)
		}
		if _, planned := plansBySource[sourceValue.ID]; planned &&

			(!attachment.Enabled || !sourceValue.Enabled) {
			return Result{}, fmt.Errorf(
				"%w: discovery plan includes disabled source %q",
				basespec.ErrInvalid,
				sourceValue.ID,
			)
		}

		expectedSourceRevisions[sourceValue.ID] = sourceValue.Revision

		if !attachment.Enabled || !sourceValue.Enabled {
			continue
		}
		sourcePlan, exists := plansBySource[sourceValue.ID]
		if !exists {
			return Result{}, fmt.Errorf(
				"%w: enabled source %q has no discovery plan",
				basespec.ErrInvalid,
				sourceValue.ID,
			)
		}

		snapshot, err := s.sources.Open(ctx, sourceValue)
		if err != nil {
			return Result{}, err
		}
		snapshots = append(snapshots, snapshot)
		sourceGenerations[sourceValue.ID] = snapshot.Generation()

		discovered, err := s.discovery.Discover(
			ctx,
			ref.RootID,
			ref.CollectionID,
			sourceValue.ID,
			sourceValue.Kind,
			snapshot,
			sourcePlan,
			previousBySource[sourceValue.ID],
		)
		if err != nil {
			return Result{}, err
		}

		finalOccurrences = append(
			finalOccurrences,
			discovered.Occurrences...,
		)
		allDiagnostics = diagnostic.AppendDiagnostics(
			allDiagnostics,
			discovered.Diagnostics...,
		)
		candidates += discovered.Candidates
		maps.Copy(discoveredDefinitions, discovered.Definitions)
	}

	for sourceID := range plansBySource {
		if _, exists := expectedSourceRevisions[sourceID]; !exists {
			return Result{}, fmt.Errorf(
				"%w: discovery plan includes unattached source %q",
				basespec.ErrInvalid,
				sourceID,
			)
		}
	}

	catalog.SortOccurrences(finalOccurrences)

	digests := make([]cryptoutil.Digest, 0, len(discoveredDefinitions))
	for digest := range discoveredDefinitions {
		digests = append(digests, digest)
	}
	slices.Sort(digests)
	for _, digest := range digests {
		stored, err := s.definitions.Put(
			ctx,
			ref.RootID,
			discoveredDefinitions[digest],
		)
		if err != nil {
			return Result{}, err
		}
		canonicalStored, err := definition.Canonicalize(stored)
		if err != nil {
			return Result{}, fmt.Errorf(
				"%w: definition repository returned an invalid definition: %w",
				basespec.ErrInvalid,
				err,
			)
		}
		if canonicalStored.Digest != digest {
			return Result{}, fmt.Errorf("%w: definition repository changed digest", basespec.ErrDigestMismatch)
		}
	}

	existingArtifacts, err := s.artifacts.ListByCollection(ctx, ref)
	if err != nil {
		return Result{}, err
	}
	suppressions, err := s.artifacts.ListSuppressions(ctx, ref)
	if err != nil {
		return Result{}, err
	}

	reconciliation, err := s.reconciler.Reconcile(
		ctx,
		collectionValue,
		finalOccurrences,
		existingArtifacts,
		suppressions,
		s.definitions,
		policy,
	)
	if err != nil {
		return Result{}, err
	}

	allDiagnostics = diagnostic.AppendDiagnostics(
		allDiagnostics,
		reconciliation.Diagnostics...,
	)

	// Confirm immediately before publication, after all potentially slow
	// definition and policy work. A final instant of external change remains
	// unavoidable, but this closes the current large confirmation gap.
	for _, snapshot := range snapshots {
		if err := snapshot.Confirm(ctx); err != nil {
			return Result{}, err
		}
	}

	closeErr := closeRefreshSnapshots(snapshots)
	snapshots = nil
	if closeErr != nil {
		return Result{}, closeErr
	}

	planFingerprint, err := plan.Fingerprint()
	if err != nil {
		return Result{}, err
	}
	decoderFingerprint, err := s.discovery.DecoderFingerprint()
	if err != nil {
		return Result{}, err
	}

	publication := Publication{
		Ref:                         ref,
		ExpectedCatalogRevision:     previous.Revision,
		ExpectedCollectionRevision:  collectionValue.Revision,
		ExpectedAttachmentRevisions: expectedAttachmentRevisions,
		ExpectedSourceRevisions:     expectedSourceRevisions,
		SourceGenerations:           sourceGenerations,
		PlanFingerprint:             planFingerprint,
		DecoderFingerprint:          decoderFingerprint,
		Occurrences:                 finalOccurrences,
		ArtifactCreates:             reconciliation.Creates,
		ArtifactUpdates:             reconciliation.Updates,
		Diagnostics:                 allDiagnostics,
		PublishedAt:                 s.clock.Now().UTC(),
	}
	published, err := s.publisher.Publish(ctx, publication)
	if err != nil {
		return Result{}, err
	}
	if err := published.Validate(); err != nil {
		return Result{}, fmt.Errorf(
			"%w: publisher returned an invalid catalog: %w",
			basespec.ErrInvalid,
			err,
		)
	}

	expected := catalog.Snapshot{
		RootID:              ref.RootID,
		CollectionID:        ref.CollectionID,
		Revision:            previous.Revision + 1,
		CollectionRevision:  collectionValue.Revision,
		AttachmentRevisions: maps.Clone(expectedAttachmentRevisions),
		SourceRevisions:     maps.Clone(expectedSourceRevisions),
		SourceGenerations:   maps.Clone(sourceGenerations),
		PlanFingerprint:     planFingerprint,
		DecoderFingerprint:  decoderFingerprint,
		PublishedAt:         publication.PublishedAt,
		Diagnostics:         diagnostic.CloneDiagnostics(allDiagnostics),
		Occurrences:         make([]catalog.Occurrence, len(finalOccurrences)),
	}
	for index, occurrence := range finalOccurrences {
		expected.Occurrences[index] = catalog.CloneOccurrence(occurrence)
	}
	if err := expected.Validate(); err != nil {
		return Result{}, fmt.Errorf(
			"%w: refresh service produced an invalid expected catalog: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if !catalog.EqualSnapshot(published, expected) {
		return Result{}, fmt.Errorf(
			"%w: publisher returned a catalog that does not exactly match the publication",
			basespec.ErrInvalid,
		)
	}

	result := Result{
		Catalog:     catalog.CloneSnapshot(published),
		Diagnostics: diagnostic.CloneDiagnostics(allDiagnostics),
		Candidates:  candidates,
	}
	for _, value := range reconciliation.Creates {
		result.CreatedArtifacts = append(result.CreatedArtifacts, value.ID)
	}
	for _, value := range reconciliation.Updates {
		result.UpdatedArtifacts = append(result.UpdatedArtifacts, value.ArtifactID)
	}
	return result, nil
}

func closeRefreshSnapshots(values []source.Snapshot) error {
	var closeErr error
	for _, snapshot := range values {
		if snapshot == nil {
			continue
		}
		closeErr = errors.Join(closeErr, snapshot.Close())
	}
	return closeErr
}
