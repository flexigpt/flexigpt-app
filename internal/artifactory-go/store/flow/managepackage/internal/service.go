package internal

import (
	"context"
	"errors"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"

	managepackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managepackage/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
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

type Service struct {
	dependencies Dependencies
}

func NewService(
	dependencies Dependencies,
) (*Service, error) {
	if dependencies.Artifacts == nil ||
		dependencies.Refresh == nil ||
		dependencies.Sources == nil ||
		dependencies.Runtime == nil ||
		dependencies.ContentMutation == nil ||
		dependencies.Packages == nil {
		return nil, fmt.Errorf(
			"%w: manage package service dependencies are incomplete",
			spec.ErrInvalid,
		)
	}

	return &Service{
		dependencies: dependencies,
	}, nil
}

func (s *Service) Publish(
	ctx context.Context,
	request managepackageModel.PublishRequest,
) (result managepackageModel.PublishResult, returnErr error) {
	if err := request.RootID.Validate(); err != nil {
		return managepackageModel.PublishResult{}, err
	}
	if err := request.Binding.Validate(); err != nil {
		return managepackageModel.PublishResult{}, err
	}
	if err := request.ExpectedKind.Validate(); err != nil {
		return managepackageModel.PublishResult{}, err
	}
	if err := request.ExpectedLogicalName.Validate(); err != nil {
		return managepackageModel.PublishResult{}, err
	}
	if err := cryptoutil.ValidateDigest(request.ExpectedDefinition); err != nil {
		return managepackageModel.PublishResult{}, err
	}
	if err := s.requireMutation(ctx, request.RootID); err != nil {
		return managepackageModel.PublishResult{}, err
	}

	publication, err := managedpackageModel.NormalizeManagedPackagePublication(
		request.Package,
	)
	if err != nil {
		return managepackageModel.PublishResult{}, err
	}

	state, err := s.sourceState(
		ctx,
		request.RootID,
		request.Binding.SourceID,
	)
	if err != nil {
		return managepackageModel.PublishResult{}, err
	}
	before := state
	defer func() {
		if before.Generation == "" {
			return
		}

		after, afterErr := s.sourceState(
			context.WithoutCancel(ctx),
			request.RootID,
			request.Binding.SourceID,
		)
		if afterErr != nil {
			returnErr = errors.Join(returnErr, afterErr)
			return
		}

		if result.Outcome.PhysicalPackage == managepackageModel.PhysicalPackageUnknown {
			result.Outcome.PhysicalPackage = managepackageModel.PhysicalPackageUnchanged
			if after.Generation != before.Generation {
				result.Outcome.PhysicalPackage = managepackageModel.PhysicalPackageChanged
			}
		}
		result.Outcome.SourceGeneration = after.Generation
		if returnErr == nil {
			result.Outcome.SourceAcknowledged = true
			result.Outcome.RefreshCompleted =
				result.Refreshed || result.Artifact.ID != ""
			result.Outcome.ArtifactVerified =
				result.Artifact.ID != ""
			return
		}
		if result.Outcome.PhysicalPackage ==
			managepackageModel.PhysicalPackageChanged {
			result.Outcome.RecoveryRequired = true
			returnErr = &managepackageModel.PublicationError{
				Outcome: result.Outcome,
				Cause:   returnErr,
			}
		}
	}()
	if err := validateManagedSourceState(
		state,
		request.RootID,
		request.Binding.SourceID,
		true,
	); err != nil {
		return managepackageModel.PublishResult{}, err
	}

	if publication.ExpectedGeneration == "" &&
		request.AllowPackageReplacement {
		publication.ExpectedGeneration = state.Generation
	}

	published, err := s.publishPackage(
		ctx,
		request.RootID,
		request.Binding.SourceID,
		state.Source.Revision,
		publication,
	)
	if err != nil {
		return managepackageModel.PublishResult{}, err
	}
	if err := validateManagedSourceState(
		published,
		request.RootID,
		request.Binding.SourceID,
		true,
	); err != nil {
		return managepackageModel.PublishResult{}, err
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
				return managepackageModel.PublishResult{
					Artifact:   resolved,
					Source:     published.Source,
					Generation: published.Generation,
					Refreshed:  false,
					Outcome: managepackageModel.PublicationOutcome{
						PhysicalPackage:    managepackageModel.PhysicalPackageUnchanged,
						SourceGeneration:   published.Generation,
						SourceAcknowledged: true,
						RefreshCompleted:   true,
						ArtifactVerified:   true,
					},
				}, nil
			}
		}
	}

	if _, err := s.dependencies.Refresh.RefreshSource(
		ctx,
		request.RootID,
		request.Binding.SourceID,
	); err != nil {
		return managepackageModel.PublishResult{}, err
	}

	resolved, err := s.dependencies.Artifacts.FindByOrigin(
		ctx,
		request.RootID,
		request.Binding,
		request.ExpectedKind,
	)
	if err != nil {
		return managepackageModel.PublishResult{}, err
	}
	if !matchesPublishExpectation(resolved, request) {
		return managepackageModel.PublishResult{}, fmt.Errorf(
			"%w: managed package did not resolve to its expected Artifact",
			spec.ErrReferenceUnresolved,
		)
	}

	return managepackageModel.PublishResult{
		Artifact:   resolved,
		Source:     published.Source,
		Generation: published.Generation,
		Refreshed:  true,
		Outcome: managepackageModel.PublicationOutcome{
			PhysicalPackage: func() managepackageModel.PhysicalPackageState {
				if published.Generation == before.Generation {
					return managepackageModel.PhysicalPackageUnchanged
				}
				return managepackageModel.PhysicalPackageChanged
			}(),
			SourceGeneration:   published.Generation,
			SourceAcknowledged: true,
			RefreshCompleted:   true,
			ArtifactVerified:   true,
		},
	}, nil
}

func (s *Service) Remove(
	ctx context.Context,
	request managepackageModel.RemoveRequest,
) error {
	_, err := s.RemoveWithOutcome(ctx, request)
	return err
}

func (s *Service) RemoveWithOutcome(
	ctx context.Context,
	request managepackageModel.RemoveRequest,
) (outcome managepackageModel.RemovalOutcome, returnErr error) {
	before, beforeErr := s.sourceState(
		ctx,
		request.RootID,
		request.SourceID,
	)
	if beforeErr != nil {
		return managepackageModel.RemovalOutcome{}, beforeErr
	}

	defer func() {
		after, afterErr := s.sourceState(
			context.WithoutCancel(ctx),
			request.RootID,
			request.SourceID,
		)
		if afterErr != nil {
			returnErr = errors.Join(returnErr, afterErr)
			return
		}

		outcome.SourceGeneration = after.Generation
		outcome.PhysicalPackage = managepackageModel.PhysicalPackageUnchanged
		if after.Generation != before.Generation {
			outcome.PhysicalPackage = managepackageModel.PhysicalPackageChanged
		}

		if returnErr == nil {
			outcome.SourceAcknowledged = true
			outcome.DiscoveryPruned =
				request.PruneDiscoveryLocator != nil &&
					after.Source.Discovery.Empty()
			if after.Source.Discovery.Empty() {
				outcome.RefreshCompleted = true
			} else if inspection, err := s.dependencies.Refresh.InspectSource(
				context.WithoutCancel(ctx),
				request.RootID,
				request.SourceID,
			); err == nil {
				outcome.RefreshCompleted = inspection.IsCurrent()
			}

			if request.ExpectedArtifact != nil {
				record, err := s.dependencies.Artifacts.Get(
					context.WithoutCancel(ctx),
					*request.ExpectedArtifact,
				)
				if err != nil {
					returnErr = err
					return
				}
				outcome.ExpectedArtifactMissing =
					record.State == artifactModel.StateMissing
			}
			return
		}

		if outcome.PhysicalPackage ==
			managepackageModel.PhysicalPackageChanged {
			outcome.RecoveryRequired = true
			returnErr = &managepackageModel.RemovalError{
				Outcome: outcome,
				Cause:   returnErr,
			}
		}
	}()

	return outcome, s.remove(ctx, request)
}

func (s *Service) remove(
	ctx context.Context,
	request managepackageModel.RemoveRequest,
) error {
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
		if err := spec.ValidateSourceGeneration(
			request.ExpectedGeneration,
		); err != nil {
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
	}
	if err := s.requireMutation(ctx, request.RootID); err != nil {
		return err
	}

	state, err := s.sourceState(
		ctx,
		request.RootID,
		request.SourceID,
	)
	if err != nil {
		return err
	}
	if err := validateManagedSourceState(
		state,
		request.RootID,
		request.SourceID,
		true,
	); err != nil {
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

	if request.PruneDiscoveryLocator != nil &&
		!state.Source.Discovery.Authoritative {
		return fmt.Errorf(
			"%w: managed discovery locator pruning requires an authoritative Source",
			spec.ErrInvalid,
		)
	}

	removed, err := s.removePackage(
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
		removed, err = s.pruneDiscoveryLocator(
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

func matchesPublishExpectation(
	value artifactModel.Artifact,
	request managepackageModel.PublishRequest,
) bool {
	return value.State == artifactModel.StateAvailable &&
		value.LogicalName == request.ExpectedLogicalName &&
		value.ResolvedDefinition != nil &&
		*value.ResolvedDefinition == request.ExpectedDefinition
}

func (s *Service) requireMutation(
	ctx context.Context,
	rootID rootModel.RootID,
) error {
	if err := rootID.Validate(); err != nil {
		return err
	}
	return root.RequireMutableRoot(ctx, s.dependencies.Policy, rootID)
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
	if err := spec.ValidateSourceGeneration(state.Generation); err != nil {
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
