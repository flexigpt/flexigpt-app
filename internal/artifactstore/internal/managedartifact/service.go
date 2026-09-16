package managedartifactimpl

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi"
	rootimpl "github.com/flexigpt/flexigpt-app/internal/artifactstore/internal/root"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type SourceState struct {
	Source     source.Summary
	Generation string
}

type GetSourceStateFunc func(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
) (SourceState, error)

type PublishPackageFunc func(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
	expectedSourceRevision uint64,
	publication source.ManagedPackagePublication,
) (SourceState, error)

type RemovePackageFunc func(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
	expectedSourceRevision uint64,
	address source.ManagedPackageAddress,
	expectedGeneration string,
) (SourceState, error)

type ArtifactCommands interface {
	Get(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) (artifact.Artifact, error)

	FindByOrigin(
		ctx context.Context,
		rootID root.RootID,
		binding artifact.SourceBinding,
		kind artifact.ArtifactKind,
	) (artifact.Artifact, error)
}

type SourceRunner interface {
	RefreshSource(
		ctx context.Context,
		rootID root.RootID,
		sourceID source.SourceID,
	) (refresh.RefreshSourceResult, error)
}

type Dependencies struct {
	Artifacts ArtifactCommands
	Refresh   SourceRunner
	Policy    root.RootPolicy

	GetSourceState          GetSourceStateFunc
	PublishPackage          PublishPackageFunc
	PublishProtectedPackage PublishPackageFunc
	RemovePackage           RemovePackageFunc
	RemoveProtectedPackage  RemovePackageFunc
}

type Service struct {
	dependencies Dependencies
}

func NewService(
	dependencies Dependencies,
) (*Service, error) {
	if dependencies.Artifacts == nil ||
		dependencies.Refresh == nil ||
		dependencies.GetSourceState == nil ||
		dependencies.PublishPackage == nil ||
		dependencies.PublishProtectedPackage == nil ||
		dependencies.RemovePackage == nil ||
		dependencies.RemoveProtectedPackage == nil {
		return nil, fmt.Errorf(
			"%w: managed Artifact service dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	return &Service{
		dependencies: dependencies,
	}, nil
}

func (s *Service) Publish(
	ctx context.Context,
	request artifact.PublishArtifactRequest,
) (artifact.PublishArtifactResult, error) {
	if s == nil {
		return artifact.PublishArtifactResult{}, basespec.ErrClosed
	}
	if err := request.RootID.Validate(); err != nil {
		return artifact.PublishArtifactResult{}, err
	}
	if err := request.Binding.Validate(); err != nil {
		return artifact.PublishArtifactResult{}, err
	}
	if err := request.ExpectedKind.Validate(); err != nil {
		return artifact.PublishArtifactResult{}, err
	}
	if err := request.ExpectedLogicalName.Validate(); err != nil {
		return artifact.PublishArtifactResult{}, err
	}
	if err := cryptoutil.ValidateDigest(
		request.ExpectedDefinition,
	); err != nil {
		return artifact.PublishArtifactResult{}, err
	}
	if err := s.requireMutable(
		ctx,
		request.RootID,
		request.AllowProtected,
	); err != nil {
		return artifact.PublishArtifactResult{}, err
	}

	publication, err := source.NormalizeManagedPackagePublication(
		request.Package,
	)
	if err != nil {
		return artifact.PublishArtifactResult{}, err
	}
	state, err := s.dependencies.GetSourceState(
		ctx,
		request.RootID,
		request.Binding.SourceID,
	)
	if err != nil {
		return artifact.PublishArtifactResult{}, err
	}
	if err := validateManagedSourceState(
		state,
		request.RootID,
		request.Binding.SourceID,
		true,
	); err != nil {
		return artifact.PublishArtifactResult{}, err
	}
	if publication.ExpectedGeneration == "" {
		publication.ExpectedGeneration = state.Generation
	}

	publish := s.dependencies.PublishPackage
	if request.AllowProtected {
		publish = s.dependencies.PublishProtectedPackage
	}
	published, err := publish(
		ctx,
		request.RootID,
		request.Binding.SourceID,
		state.Source.Revision,
		publication,
	)
	if err != nil {
		return artifact.PublishArtifactResult{}, err
	}
	if err := validateManagedSourceState(
		published,
		request.RootID,
		request.Binding.SourceID,
		true,
	); err != nil {
		return artifact.PublishArtifactResult{}, err
	}

	if _, err := s.dependencies.Refresh.RefreshSource(
		ctx,
		request.RootID,
		request.Binding.SourceID,
	); err != nil {
		return artifact.PublishArtifactResult{}, err
	}
	resolved, err := s.dependencies.Artifacts.FindByOrigin(
		ctx,
		request.RootID,
		request.Binding,
		request.ExpectedKind,
	)
	if err != nil {
		return artifact.PublishArtifactResult{}, err
	}
	if resolved.State != artifact.StateAvailable ||
		resolved.LogicalName != request.ExpectedLogicalName ||
		resolved.ResolvedDefinition == nil ||
		*resolved.ResolvedDefinition != request.ExpectedDefinition {
		return artifact.PublishArtifactResult{}, fmt.Errorf(
			"%w: managed package did not resolve to its expected Artifact",
			basespec.ErrReferenceUnresolved,
		)
	}
	return artifact.PublishArtifactResult{
		Artifact:   resolved,
		Source:     published.Source,
		Generation: published.Generation,
		Refreshed:  true,
	}, nil
}

func (s *Service) Remove(
	ctx context.Context,
	request artifact.RemoveArtifactRequest,
) error {
	if s == nil {
		return basespec.ErrClosed
	}
	if err := request.RootID.Validate(); err != nil {
		return err
	}
	if err := request.SourceID.Validate(); err != nil {
		return err
	}
	if err := request.Package.Validate(); err != nil {
		return err
	}
	if request.ExpectedGeneration != "" {
		if err := basespec.ValidateSourceGeneration(request.ExpectedGeneration); err != nil {
			return err
		}
	}
	if request.ExpectedArtifact != nil {
		if err := request.ExpectedArtifact.Validate(); err != nil {
			return err
		}
		if request.ExpectedArtifact.RootID != request.RootID {
			return fmt.Errorf(
				"%w: expected Artifact belongs to another Root",
				basespec.ErrInvalid,
			)
		}
	}
	if err := s.requireMutable(
		ctx,
		request.RootID,
		request.AllowProtected,
	); err != nil {
		return err
	}

	state, err := s.dependencies.GetSourceState(
		ctx,
		request.RootID,
		request.SourceID,
	)
	if err != nil {
		return err
	}
	if request.ExpectedGeneration != "" &&
		request.ExpectedGeneration != state.Generation {
		return fmt.Errorf(
			"%w: managed Source changed before package removal",
			basespec.ErrConflict,
		)
	}
	expectedGeneration := state.Generation
	if request.ExpectedGeneration != "" {
		expectedGeneration = request.ExpectedGeneration
	}
	if err := validateManagedSourceState(
		state,
		request.RootID,
		request.SourceID,
		false,
	); err != nil {
		return err
	}
	remove := s.dependencies.RemovePackage
	if request.AllowProtected {
		remove = s.dependencies.RemoveProtectedPackage
	}
	if _, err := remove(
		ctx,
		request.RootID,
		request.SourceID,
		state.Source.Revision,
		request.Package,
		expectedGeneration,
	); err != nil {
		return err
	}

	if _, err := s.dependencies.Refresh.RefreshSource(
		ctx,
		request.RootID,
		request.SourceID,
	); err != nil {
		return err
	}
	if request.ExpectedArtifact == nil {
		return nil
	}
	value, err := s.dependencies.Artifacts.Get(
		ctx,
		*request.ExpectedArtifact,
	)
	if err != nil {
		return err
	}
	if value.Binding.SourceID != request.SourceID ||
		value.State != artifact.StateMissing {
		return fmt.Errorf(
			"%w: managed package removal did not make expected Artifact missing",
			basespec.ErrConflict,
		)
	}
	return nil
}

func (s *Service) requireMutable(
	ctx context.Context,
	rootID root.RootID,
	allowProtected bool,
) error {
	if ctx == nil {
		return fmt.Errorf(
			"%w: managed Artifact context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := rootID.Validate(); err != nil {
		return err
	}
	if allowProtected {
		if s.dependencies.Policy == nil ||
			!s.dependencies.Policy.IsProtectedRoot(rootID) {
			return fmt.Errorf(
				"%w: managed protected operation requires a protected Root",
				basespec.ErrProtected,
			)
		}
		return installerapi.RequirePrivileged(ctx)
	}
	return rootimpl.RequireMutableRoot(
		ctx,
		s.dependencies.Policy,
		rootID,
	)
}

func validateManagedSourceState(
	state SourceState,
	rootID root.RootID,
	sourceID source.SourceID,
	requireEnabled bool,
) error {
	if err := state.Source.Validate(); err != nil {
		return fmt.Errorf(
			"%w: managed Source state: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if state.Source.RootID != rootID ||
		state.Source.ID != sourceID {
		return fmt.Errorf(
			"%w: managed Source state does not match request",
			basespec.ErrInvalid,
		)
	}
	if err := basespec.ValidateSourceGeneration(
		state.Generation,
	); err != nil {
		return err
	}
	if requireEnabled && !state.Source.Enabled {
		return fmt.Errorf(
			"%w: managed Source is disabled",
			basespec.ErrConflict,
		)
	}
	return nil
}
