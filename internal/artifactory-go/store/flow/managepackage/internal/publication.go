package internal

import (
	"context"

	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func (s *Service) publishPackage(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	expectedSourceRevision uint64,
	publication managedpackageModel.ManagedPackagePublication,
) (SourceState, error) {
	if s == nil ||
		s.dependencies.Runtime == nil ||
		s.dependencies.ContentMutation == nil ||
		s.dependencies.Packages == nil {
		return SourceState{}, spec.ErrClosed
	}

	value, err := s.managedSource(
		ctx,
		rootID,
		sourceID,
		expectedSourceRevision,
	)
	if err != nil {
		return SourceState{}, err
	}

	beforeGeneration, err := sourceSnapshotGeneration(
		ctx,
		s.dependencies.Runtime,
		value,
	)
	if err != nil {
		return SourceState{}, err
	}

	generation, err := s.dependencies.Packages.PublishPackage(
		ctx,
		value,
		publication,
	)
	if err != nil {
		return SourceState{}, err
	}

	result := SourceState{
		Source:     value.Summary(),
		Generation: generation,
	}
	if generation == beforeGeneration {
		return result, nil
	}

	updated, err := s.dependencies.ContentMutation.MarkContentChanged(
		ctx,
		rootID,
		sourceID,
		expectedSourceRevision,
	)
	if err != nil {
		return SourceState{}, err
	}

	result.Source = updated
	return result, nil
}
