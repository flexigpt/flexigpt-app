package refresh

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/record"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

type Service struct {
	roots       RootReader
	catalogs    catalog.Reader
	sources     source.Runtime
	records     RecordReader
	discovery   *discovery.Engine
	definitions definition.Repository
	reconciler  *record.Reconciler
	publisher   Publisher
	clock       artifactstore.Clock
}

func NewService(
	roots RootReader,
	catalogs catalog.Reader,
	sources source.Runtime,
	records RecordReader,
	discoveryEngine *discovery.Engine,
	definitions definition.Repository,
	reconciler *record.Reconciler,
	publisher Publisher,
	clock artifactstore.Clock,
) (*Service, error) {
	if roots == nil ||
		catalogs == nil ||
		sources == nil ||
		records == nil ||
		discoveryEngine == nil ||
		definitions == nil ||
		reconciler == nil ||
		publisher == nil ||
		clock == nil {
		return nil, fmt.Errorf(
			"%w: refresh service dependencies are incomplete",
			artifactstore.ErrInvalid,
		)
	}
	return &Service{
		roots:       roots,
		catalogs:    catalogs,
		sources:     sources,
		records:     records,
		discovery:   discoveryEngine,
		definitions: definitions,
		reconciler:  reconciler,
		publisher:   publisher,
		clock:       clock,
	}, nil
}

func (s *Service) Refresh(
	ctx context.Context,
	rootID artifactstore.RootID,
	plan discovery.Plan,
	policy record.Policy,
) (Result, error) {
	if policy == nil {
		return Result{}, fmt.Errorf(
			"%w: record policy is required",
			artifactstore.ErrInvalid,
		)
	}
	if err := plan.Validate(); err != nil {
		return Result{}, err
	}

	root, err := s.roots.Get(ctx, rootID, false)
	if err != nil {
		return Result{}, err
	}
	if !root.Enabled {
		return Result{}, fmt.Errorf(
			"%w: root %q is disabled",
			artifactstore.ErrConflict,
			rootID,
		)
	}

	attachments, err := s.roots.ListAttachments(ctx, rootID)
	if err != nil {
		return Result{}, err
	}
	plansBySource := plan.BySource()

	var previous catalog.Snapshot
	previous, err = s.catalogs.GetCurrent(ctx, rootID)
	if err != nil && !errors.Is(err, artifactstore.ErrCatalogUnavailable) {
		return Result{}, err
	}
	previousBySource := make(
		map[artifactstore.SourceID][]catalog.Occurrence,
	)
	for _, occurrence := range previous.Occurrences {
		previousBySource[occurrence.Key.SourceID] = append(
			previousBySource[occurrence.Key.SourceID],
			occurrence,
		)
	}

	expectedSourceRevisions := make(map[artifactstore.SourceID]uint64)
	sourceGenerations := make(map[artifactstore.SourceID]string)
	finalOccurrences := make([]catalog.Occurrence, 0)
	allDiagnostics := make([]artifactstore.Diagnostic, 0)
	definitions := make(map[artifactstore.Digest]definition.Definition)
	snapshots := make([]source.Snapshot, 0)
	candidates := 0

	defer func() {
		for _, snapshot := range snapshots {
			_ = snapshot.Close()
		}
	}()

	sort.Slice(attachments, func(left, right int) bool {
		return attachments[left].SourceID < attachments[right].SourceID
	})

	for _, attachment := range attachments {

		sourceValue, err := s.sources.Get(ctx, attachment.SourceID)
		if err != nil {
			return Result{}, err
		}
		if _, planned := plansBySource[sourceValue.ID]; planned &&
			(!attachment.Enabled || !sourceValue.Enabled) {
			return Result{}, fmt.Errorf(
				"%w: discovery plan includes disabled source %q",
				artifactstore.ErrInvalid,
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
				artifactstore.ErrInvalid,
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
			rootID,
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
		allDiagnostics = artifactstore.AppendDiagnostics(
			allDiagnostics,
			discovered.Diagnostics...,
		)
		candidates += discovered.Candidates
		maps.Copy(definitions, discovered.Definitions)
	}

	for sourceID := range plansBySource {
		if _, exists := expectedSourceRevisions[sourceID]; !exists {
			return Result{}, fmt.Errorf(
				"%w: discovery plan includes unattached source %q",
				artifactstore.ErrInvalid,
				sourceID,
			)
		}
	}
	for _, value := range definitions {
		if _, err := s.definitions.Put(ctx, value); err != nil {
			return Result{}, err
		}
	}

	for _, snapshot := range snapshots {
		if err := snapshot.Confirm(ctx); err != nil {
			return Result{}, err
		}
	}

	existingRecords, err := s.records.ListByRoot(ctx, rootID)
	if err != nil {
		return Result{}, err
	}
	reconciliation, err := s.reconciler.Reconcile(
		ctx,
		root,
		finalOccurrences,
		existingRecords,
		s.definitions,
		policy,
	)
	if err != nil {
		return Result{}, err
	}
	allDiagnostics = artifactstore.AppendDiagnostics(
		allDiagnostics,
		reconciliation.Diagnostics...,
	)

	publication := Publication{
		RootID:                  rootID,
		ExpectedRootRevision:    root.Revision,
		ExpectedSourceRevisions: expectedSourceRevisions,
		SourceGenerations:       sourceGenerations,
		Occurrences:             finalOccurrences,
		RecordCreates:           reconciliation.Creates,
		RecordUpdates:           reconciliation.Updates,
		Diagnostics:             allDiagnostics,
		PublishedAt:             s.clock.Now().UTC(),
	}
	published, err := s.publisher.Publish(ctx, publication)
	if err != nil {
		return Result{}, err
	}

	result := Result{
		Catalog:     published,
		Diagnostics: allDiagnostics,
		Candidates:  candidates,
	}
	for _, value := range reconciliation.Creates {
		result.CreatedRecords = append(result.CreatedRecords, value.ID)
	}
	for _, value := range reconciliation.Updates {
		result.UpdatedRecords = append(result.UpdatedRecords, value.RecordID)
	}
	return result, nil
}
