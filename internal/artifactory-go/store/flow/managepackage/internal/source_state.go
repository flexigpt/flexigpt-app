package internal

import (
	"context"
	"errors"
	"fmt"

	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func (s *Service) sourceState(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
) (SourceState, error) {
	if err := rootID.Validate(); err != nil {
		return SourceState{}, err
	}
	if err := sourceID.Validate(); err != nil {
		return SourceState{}, err
	}

	value, err := s.dependencies.Runtime.Get(
		ctx,
		rootID,
		sourceID,
	)
	if err != nil {
		return SourceState{}, err
	}
	if !s.dependencies.Packages.SupportsManagedPackages(value.Kind) {
		return SourceState{}, fmt.Errorf(
			"%w: source kind %q is not writable",
			spec.ErrUnsupported,
			value.Kind,
		)
	}

	generation, err := sourceSnapshotGeneration(
		ctx,
		s.dependencies.Runtime,
		value,
	)
	if err != nil {
		return SourceState{}, err
	}

	return SourceState{
		Source:     value.Summary(),
		Generation: generation,
	}, nil
}

func (s *Service) managedSource(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	expectedSourceRevision uint64,
) (sourceModel.Source, error) {
	if expectedSourceRevision == 0 {
		return sourceModel.Source{}, fmt.Errorf(
			"%w: expected Source revision is required",
			spec.ErrInvalid,
		)
	}

	value, err := s.dependencies.Runtime.Get(ctx, rootID, sourceID)
	if err != nil {
		return sourceModel.Source{}, err
	}
	if value.Revision != expectedSourceRevision {
		return sourceModel.Source{}, spec.ErrConflict
	}
	if !s.dependencies.Packages.SupportsManagedPackages(value.Kind) {
		return sourceModel.Source{}, fmt.Errorf(
			"%w: source kind %q is not writable",
			spec.ErrUnsupported,
			value.Kind,
		)
	}

	return value, nil
}

func sourceSnapshotGeneration(
	ctx context.Context,
	runtime source.Runtime,
	value sourceModel.Source,
) (string, error) {
	snapshot, err := runtime.Open(ctx, value)
	if err != nil {
		return "", err
	}

	generation := snapshot.Generation()
	confirmErr := snapshot.Confirm(ctx)
	closeErr := snapshot.Close()

	if err := errors.Join(confirmErr, closeErr); err != nil {
		return "", err
	}

	return generation, nil
}

func managedPackageExists(
	ctx context.Context,
	runtime source.Runtime,
	value sourceModel.Source,
	address managedpackageModel.ManagedPackageAddress,
) (exists bool, returnErr error) {
	snapshot, err := runtime.Open(ctx, value)
	if err != nil {
		return false, err
	}
	defer func() {
		returnErr = errors.Join(returnErr, snapshot.Close())
	}()

	directory, err := address.Directory()
	if err != nil {
		return false, err
	}

	entry, statErr := snapshot.Stat(ctx, directory)
	confirmErr := snapshot.Confirm(ctx)
	if confirmErr != nil {
		return false, errors.Join(statErr, confirmErr)
	}

	if errors.Is(statErr, spec.ErrNotFound) {
		return false, nil
	}
	if statErr != nil {
		return false, statErr
	}
	if !entry.IsDirectory {
		return false, fmt.Errorf(
			"%w: managed package %q is not a directory",
			spec.ErrInvalid,
			address,
		)
	}

	return true, nil
}
