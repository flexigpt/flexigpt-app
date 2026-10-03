package internal

import (
	"context"
	"fmt"

	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func (s *Service) removePackage(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	expectedSourceRevision uint64,
	address managedpackageModel.ManagedPackageAddress,
	expectedGeneration string,
) (SourceState, error) {
	if s == nil ||
		s.dependencies.Runtime == nil ||
		s.dependencies.ContentMutation == nil ||
		s.dependencies.Packages == nil {
		return SourceState{}, spec.ErrClosed
	}
	if err := address.Validate(); err != nil {
		return SourceState{}, err
	}
	if err := spec.ValidateSourceGeneration(expectedGeneration); err != nil {
		return SourceState{}, err
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

	if beforeGeneration != expectedGeneration {
		exists, err := managedPackageExists(
			ctx,
			s.dependencies.Runtime,
			value,
			address,
		)
		if err != nil {
			return SourceState{}, err
		}
		if exists {
			return SourceState{}, fmt.Errorf(
				"%w: managed Source changed before package removal",
				spec.ErrConflict,
			)
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

		return SourceState{
			Source:     updated,
			Generation: beforeGeneration,
		}, nil
	}

	if err := s.dependencies.Packages.RemovePackage(
		ctx,
		value,
		address,
		expectedGeneration,
	); err != nil {
		return SourceState{}, err
	}

	generation, err := sourceSnapshotGeneration(
		ctx,
		s.dependencies.Runtime,
		value,
	)
	if err != nil {
		return SourceState{}, err
	}
	if generation == beforeGeneration {
		return SourceState{
			Source:     value.Summary(),
			Generation: generation,
		}, nil
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

	return SourceState{
		Source:     updated,
		Generation: generation,
	}, nil
}
