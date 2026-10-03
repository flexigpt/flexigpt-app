package internal

import (
	"context"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	managedpackageFlowModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managedpackage/model"
	refreshModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh/model"
	rootimpl "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/impl"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type SourceState struct {
	Source     sourceModel.Summary
	Generation string
}

type GetSourceStateFunc func(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
) (SourceState, error)

type PublishPackageFunc func(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	expectedSourceRevision uint64,
	publication managedpackageModel.ManagedPackagePublication,
) (SourceState, error)

type RemovePackageFunc func(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	expectedSourceRevision uint64,
	address managedpackageModel.ManagedPackageAddress,
	expectedGeneration string,
) (SourceState, error)

// PruneDiscoveryLocatorFunc updates managed Source discovery after physical
// package removal and before the managed Artifact service performs its final
// refresh. The callback is optional because some packages emit multiple
// declaration origins or are discovered through directory scopes.
type PruneDiscoveryLocatorFunc func(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	expectedSourceRevision uint64,
	locator spec.Locator,
) (SourceState, error)

type ArtifactCommands interface {
	Get(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
	) (artifactModel.Artifact, error)

	FindByOrigin(
		ctx context.Context,
		rootID rootModel.RootID,
		binding artifactModel.SourceBinding,
		kind artifactModel.ArtifactKind,
	) (artifactModel.Artifact, error)
}

type SourceRunner interface {
	RefreshSource(
		ctx context.Context,
		rootID rootModel.RootID,
		sourceID sourceModel.SourceID,
	) (refreshModel.RefreshSourceResult, error)

	InspectSource(
		ctx context.Context,
		rootID rootModel.RootID,
		sourceID sourceModel.SourceID,
	) (refreshModel.Inspection, error)
}

type Dependencies struct {
	Artifacts ArtifactCommands
	Refresh   SourceRunner
	Policy    rootModel.RootPolicy

	GetSourceState          GetSourceStateFunc
	PublishPackage          PublishPackageFunc
	PublishProtectedPackage PublishPackageFunc
	RemovePackage           RemovePackageFunc
	RemoveProtectedPackage  RemovePackageFunc
	PruneDiscoveryLocator   PruneDiscoveryLocatorFunc
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
			spec.ErrInvalid,
		)
	}
	return &Service{
		dependencies: dependencies,
	}, nil
}

func (s *Service) Publish(
	ctx context.Context,
	request managedpackageFlowModel.PublishRequest,
) (managedpackageFlowModel.PublishResult, error) {
	if s == nil {
		return managedpackageFlowModel.PublishResult{}, spec.ErrClosed
	}
	if err := request.RootID.Validate(); err != nil {
		return managedpackageFlowModel.PublishResult{}, err
	}
	if err := request.Binding.Validate(); err != nil {
		return managedpackageFlowModel.PublishResult{}, err
	}
	if err := request.ExpectedKind.Validate(); err != nil {
		return managedpackageFlowModel.PublishResult{}, err
	}
	if err := request.ExpectedLogicalName.Validate(); err != nil {
		return managedpackageFlowModel.PublishResult{}, err
	}
	if err := cryptoutil.ValidateDigest(
		request.ExpectedDefinition,
	); err != nil {
		return managedpackageFlowModel.PublishResult{}, err
	}
	if err := s.requireMutable(
		ctx,
		request.RootID,
		request.AllowProtected,
	); err != nil {
		return managedpackageFlowModel.PublishResult{}, err
	}

	publication, err := managedpackageModel.NormalizeManagedPackagePublication(
		request.Package,
	)
	if err != nil {
		return managedpackageFlowModel.PublishResult{}, err
	}
	state, err := s.dependencies.GetSourceState(
		ctx,
		request.RootID,
		request.Binding.SourceID,
	)
	if err != nil {
		return managedpackageFlowModel.PublishResult{}, err
	}
	if err := validateManagedSourceState(
		state,
		request.RootID,
		request.Binding.SourceID,
		true,
	); err != nil {
		return managedpackageFlowModel.PublishResult{}, err
	}
	if publication.ExpectedGeneration == "" &&
		request.AllowPackageReplacement {
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
		return managedpackageFlowModel.PublishResult{}, err
	}
	if err := validateManagedSourceState(
		published,
		request.RootID,
		request.Binding.SourceID,
		true,
	); err != nil {
		return managedpackageFlowModel.PublishResult{}, err
	}

	if published.Source.Revision == state.Source.Revision &&
		published.Generation == state.Generation {
		inspection, inspectErr := s.dependencies.Refresh.InspectSource(
			ctx,
			request.RootID,
			request.Binding.SourceID,
		)
		if inspectErr == nil && inspection.IsCurrent() {
			resolved, findErr := s.dependencies.Artifacts.FindByOrigin(
				ctx,
				request.RootID,
				request.Binding,
				request.ExpectedKind,
			)
			if findErr == nil &&
				matchesPublishExpectation(resolved, request) {
				return managedpackageFlowModel.PublishResult{
					Artifact:   resolved,
					Source:     published.Source,
					Generation: published.Generation,
					Refreshed:  false,
				}, nil
			}
		}
	}

	if _, err := s.dependencies.Refresh.RefreshSource(
		ctx,
		request.RootID,
		request.Binding.SourceID,
	); err != nil {
		return managedpackageFlowModel.PublishResult{}, err
	}
	resolved, err := s.dependencies.Artifacts.FindByOrigin(
		ctx,
		request.RootID,
		request.Binding,
		request.ExpectedKind,
	)
	if err != nil {
		return managedpackageFlowModel.PublishResult{}, err
	}
	if resolved.State != artifactModel.StateAvailable ||
		resolved.LogicalName != request.ExpectedLogicalName ||
		resolved.ResolvedDefinition == nil ||
		*resolved.ResolvedDefinition != request.ExpectedDefinition {
		return managedpackageFlowModel.PublishResult{}, fmt.Errorf(
			"%w: managed package did not resolve to its expected Artifact",
			spec.ErrReferenceUnresolved,
		)
	}
	return managedpackageFlowModel.PublishResult{
		Artifact:   resolved,
		Source:     published.Source,
		Generation: published.Generation,
		Refreshed:  true,
	}, nil
}

func matchesPublishExpectation(
	value artifactModel.Artifact,
	request managedpackageFlowModel.PublishRequest,
) bool {
	return value.State == artifactModel.StateAvailable &&
		value.LogicalName == request.ExpectedLogicalName &&
		value.ResolvedDefinition != nil &&
		*value.ResolvedDefinition == request.ExpectedDefinition
}

func (s *Service) Remove(
	ctx context.Context,
	request managedpackageFlowModel.RemoveRequest,
) error {
	if s == nil {
		return spec.ErrClosed
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
		if err := spec.ValidateSourceGeneration(request.ExpectedGeneration); err != nil {
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
				spec.ErrInvalid,
			)
		}
	}
	if request.PruneDiscoveryLocator != nil {
		if err := request.PruneDiscoveryLocator.Validate(false); err != nil {
			return err
		}
		if s.dependencies.PruneDiscoveryLocator == nil {
			return fmt.Errorf(
				"%w: managed discovery locator pruning is unavailable",
				spec.ErrUnsupported,
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
			spec.ErrConflict,
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
		true,
	); err != nil {
		return err
	}
	if request.PruneDiscoveryLocator != nil &&
		!state.Source.Discovery.Authoritative {
		return fmt.Errorf(
			"%w: managed discovery locator pruning requires an authoritative Source",
			spec.ErrInvalid,
		)
	}

	remove := s.dependencies.RemovePackage
	if request.AllowProtected {
		remove = s.dependencies.RemoveProtectedPackage
	}
	removed, err := remove(
		ctx,
		request.RootID,
		request.SourceID,
		state.Source.Revision,
		request.Package,
		expectedGeneration,
	)
	if err != nil {
		return err
	}
	if err := validateManagedSourceState(
		removed,
		request.RootID,
		request.SourceID,
		true,
	); err != nil {
		return err
	}
	if request.PruneDiscoveryLocator != nil {
		removed, err = s.dependencies.PruneDiscoveryLocator(
			ctx,
			request.RootID,
			request.SourceID,
			removed.Source.Revision,
			*request.PruneDiscoveryLocator,
		)
		if err != nil {
			return fmt.Errorf(
				"prune managed declaration discovery locator: %w",
				err,
			)
		}
		if err := validateManagedSourceState(
			removed,
			request.RootID,
			request.SourceID,
			true,
		); err != nil {
			return err
		}
	}

	if !removed.Source.Discovery.Empty() {
		if _, err := s.dependencies.Refresh.RefreshSource(
			ctx,
			request.RootID,
			request.SourceID,
		); err != nil {
			return err
		}
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
		value.State != artifactModel.StateMissing {
		return fmt.Errorf(
			"%w: managed package removal did not make expected Artifact missing",
			spec.ErrConflict,
		)
	}
	return nil
}

func (s *Service) requireMutable(
	ctx context.Context,
	rootID rootModel.RootID,
	allowProtected bool,
) error {
	if ctx == nil {
		return fmt.Errorf(
			"%w: managed Artifact context is nil",
			spec.ErrInvalid,
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
				spec.ErrProtected,
			)
		}
		return installFlow.RequirePrivileged(ctx)
	}
	return rootimpl.RequireMutableRoot(
		ctx,
		s.dependencies.Policy,
		rootID,
	)
}

func validateManagedSourceState(
	state SourceState,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	requireEnabled bool,
) error {
	if err := state.Source.Validate(); err != nil {
		return fmt.Errorf(
			"%w: managed Source state: %w",
			spec.ErrInvalid,
			err,
		)
	}
	if state.Source.RootID != rootID ||
		state.Source.ID != sourceID {
		return fmt.Errorf(
			"%w: managed Source state does not match request",
			spec.ErrInvalid,
		)
	}
	if err := spec.ValidateSourceGeneration(
		state.Generation,
	); err != nil {
		return err
	}
	if requireEnabled && !state.Source.Enabled {
		return fmt.Errorf(
			"%w: managed Source is disabled",
			spec.ErrConflict,
		)
	}
	return nil
}
