package managedartifact

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/discovery"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/protection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type SourceState struct {
	Source     source.Summary
	Generation string
}

type GetSourceStateFunc func(
	ctx context.Context,
	rootID basespec.RootID,
	sourceID basespec.SourceID,
) (SourceState, error)

type PublishPackageFunc func(
	ctx context.Context,
	rootID basespec.RootID,
	sourceID basespec.SourceID,
	expectedSourceRevision uint64,
	publication source.ManagedPackagePublication,
) (SourceState, error)

type RemovePackageFunc func(
	ctx context.Context,
	rootID basespec.RootID,
	sourceID basespec.SourceID,
	expectedSourceRevision uint64,
	directory basespec.Locator,
	expectedGeneration string,
) (SourceState, error)

type Dependencies struct {
	Artifacts   *artifact.Service
	Collections collection.Reader
	Refresh     refresh.Runner
	Policy      protection.RootPolicy

	GetSourceState          GetSourceStateFunc
	PublishPackage          PublishPackageFunc
	PublishProtectedPackage PublishPackageFunc
	RemovePackage           RemovePackageFunc
	RemoveProtectedPackage  RemovePackageFunc
}

type Service struct {
	dependencies Dependencies
}

func NewService(dependencies Dependencies) (*Service, error) {
	if dependencies.Artifacts == nil ||
		dependencies.Collections == nil ||
		dependencies.Refresh == nil ||
		dependencies.GetSourceState == nil ||
		dependencies.PublishPackage == nil ||
		dependencies.PublishProtectedPackage == nil ||
		dependencies.RemovePackage == nil ||
		dependencies.RemoveProtectedPackage == nil {
		return nil, fmt.Errorf(
			"%w: managed artifact service dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	return &Service{dependencies: dependencies}, nil
}

type PublishRequest struct {
	Artifact           artifact.Artifact
	ExpectedDefinition cryptoutil.Digest
	Package            source.ManagedPackagePublication
	Plan               discovery.Plan
	RefreshPolicy      artifact.Policy
	AllowProtected     bool
}

type PublishResult struct {
	Artifact   artifact.Artifact
	Source     source.Summary
	Generation string
	Refreshed  bool
}

type PublishCollectionRequest struct {
	Collection     collection.CollectionRef
	SourceID       basespec.SourceID
	Package        source.ManagedPackagePublication
	Plan           discovery.Plan
	RefreshPolicy  artifact.Policy
	AllowProtected bool
	ForceRefresh   bool
}

type PublishCollectionResult struct {
	Source     source.Summary
	Generation string
	Refreshed  bool
}

// PublishCollection publishes one complete managed Source package and,
// when required, refreshes the owning Collection. It does not interpret
// package content or Artifact kind semantics.
func (s *Service) PublishCollection(
	ctx context.Context,
	request PublishCollectionRequest,
) (PublishCollectionResult, error) {
	if s == nil {
		return PublishCollectionResult{}, basespec.ErrClosed
	}
	if err := request.Collection.Validate(); err != nil {
		return PublishCollectionResult{}, err
	}
	if err := basespec.ValidateSourceID(request.SourceID); err != nil {
		return PublishCollectionResult{}, err
	}
	if request.RefreshPolicy == nil {
		return PublishCollectionResult{}, fmt.Errorf(
			"%w: managed collection refresh policy is required",
			basespec.ErrInvalid,
		)
	}
	if err := request.Plan.Validate(); err != nil {
		return PublishCollectionResult{}, err
	}
	if err := s.requireMutable(
		ctx,
		request.Collection.RootID,
		request.AllowProtected,
	); err != nil {
		return PublishCollectionResult{}, err
	}
	if err := s.requireCollectionSource(
		ctx,
		request.Collection,
		request.SourceID,
		true,
	); err != nil {
		return PublishCollectionResult{}, err
	}
	if _, found := request.Plan.BySource()[request.SourceID]; !found {
		return PublishCollectionResult{}, fmt.Errorf(
			"%w: managed collection publication plan does not include source %q",
			basespec.ErrInvalid,
			request.SourceID,
		)
	}

	publication, err := source.NormalizeManagedPackagePublication(
		request.Package,
	)
	if err != nil {
		return PublishCollectionResult{}, err
	}
	state, err := s.dependencies.GetSourceState(
		ctx,
		request.Collection.RootID,
		request.SourceID,
	)
	if err != nil {
		return PublishCollectionResult{}, err
	}
	if err := validateManagedSourceState(
		state,
		request.Collection.RootID,
		request.SourceID,
		true,
	); err != nil {
		return PublishCollectionResult{}, err
	}
	if state.Source.ID != request.SourceID ||
		state.Source.RootID != request.Collection.RootID {
		return PublishCollectionResult{}, fmt.Errorf(
			"%w: managed source state does not match collection publication",
			basespec.ErrInvalid,
		)
	}
	if !state.Source.Enabled {
		return PublishCollectionResult{}, fmt.Errorf(
			"%w: managed collection source is disabled",
			basespec.ErrConflict,
		)
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
		request.Collection.RootID,
		request.SourceID,
		state.Source.Revision,
		publication,
	)
	if err != nil {
		return PublishCollectionResult{}, err
	}
	if published.Source.ID != request.SourceID ||
		published.Source.RootID != request.Collection.RootID {
		return PublishCollectionResult{}, fmt.Errorf(
			"%w: managed collection publication returned another source",
			basespec.ErrInvalid,
		)
	}
	if err := validateManagedSourceState(
		published,
		request.Collection.RootID,
		request.SourceID,
		true,
	); err != nil {
		return PublishCollectionResult{}, err
	}

	changed := published.Source.Revision != state.Source.Revision ||
		published.Generation != state.Generation
	if request.ForceRefresh || changed {
		if _, err := s.dependencies.Refresh.Refresh(
			ctx,
			request.Collection,
			request.Plan,
			request.RefreshPolicy,
		); err != nil {
			return PublishCollectionResult{}, err
		}
	}
	return PublishCollectionResult{
		Source:     published.Source,
		Generation: published.Generation,
		Refreshed:  request.ForceRefresh || changed,
	}, nil
}

func (s *Service) Publish(
	ctx context.Context,
	request PublishRequest,
) (PublishResult, error) {
	if err := s.validatePublishRequest(ctx, request); err != nil {
		return PublishResult{}, err
	}
	normalized, err := source.NormalizeManagedPackagePublication(request.Package)
	if err != nil {
		return PublishResult{}, err
	}
	request.Package = normalized

	current, err := s.dependencies.Artifacts.Get(ctx, request.Artifact.Ref())
	if err != nil {
		return PublishResult{}, err
	}
	if current.Revision != request.Artifact.Revision ||
		current.RootID != request.Artifact.RootID ||
		current.CollectionID != request.Artifact.CollectionID ||
		current.Binding != request.Artifact.Binding ||
		current.Adoption != artifact.AdoptionPinned {
		return PublishResult{}, fmt.Errorf(
			"%w: managed artifact changed before package publication",
			basespec.ErrConflict,
		)
	}
	collectionRef := collection.CollectionRef{
		RootID:       current.RootID,
		CollectionID: current.CollectionID,
	}
	if err := s.requireCollectionSource(
		ctx,
		collectionRef,
		current.Binding.SourceID,
		true,
	); err != nil {
		return PublishResult{}, err
	}
	if _, found := request.Plan.BySource()[current.Binding.SourceID]; !found {
		return PublishResult{}, fmt.Errorf(
			"%w: managed artifact publication plan does not include its source",
			basespec.ErrInvalid,
		)
	}

	state, err := s.dependencies.GetSourceState(
		ctx,
		current.RootID,
		current.Binding.SourceID,
	)
	if err != nil {
		return PublishResult{}, err
	}
	if err := validateManagedSourceState(
		state,
		current.RootID,
		current.Binding.SourceID,
		true,
	); err != nil {
		return PublishResult{}, err
	}
	if state.Source.ID != current.Binding.SourceID ||
		state.Source.RootID != current.RootID {
		return PublishResult{}, fmt.Errorf(
			"%w: managed source state does not match artifact binding",
			basespec.ErrInvalid,
		)
	}

	publication := request.Package
	if publication.ExpectedGeneration == "" {
		publication.ExpectedGeneration = state.Generation
	}
	publish := s.dependencies.PublishPackage
	if request.AllowProtected {
		publish = s.dependencies.PublishProtectedPackage
	}
	published, err := publish(
		ctx,
		current.RootID,
		current.Binding.SourceID,
		state.Source.Revision,
		publication,
	)
	if err != nil {
		return PublishResult{}, err
	}
	if published.Source.ID != current.Binding.SourceID ||
		published.Source.RootID != current.RootID {
		return PublishResult{}, fmt.Errorf(
			"%w: managed package publication returned another source",
			basespec.ErrInvalid,
		)
	}
	if err := validateManagedSourceState(
		published,
		current.RootID,
		current.Binding.SourceID,
		true,
	); err != nil {
		return PublishResult{}, err
	}

	if published.Source.Revision == state.Source.Revision &&
		published.Generation == state.Generation {
		resolved, err := s.dependencies.Artifacts.Get(ctx, current.Ref())
		if err != nil {
			return PublishResult{}, err
		}
		if artifactMatchesDefinition(resolved, request.ExpectedDefinition) {
			return PublishResult{
				Artifact:   resolved,
				Source:     published.Source,
				Generation: published.Generation,
				Refreshed:  false,
			}, nil
		}
	}

	if _, err := s.dependencies.Refresh.Refresh(
		ctx,
		collection.CollectionRef{
			RootID:       current.RootID,
			CollectionID: current.CollectionID,
		},
		request.Plan,
		request.RefreshPolicy,
	); err != nil {
		return PublishResult{}, err
	}
	resolved, err := s.dependencies.Artifacts.Get(ctx, current.Ref())
	if err != nil {
		return PublishResult{}, err
	}
	if !artifactMatchesDefinition(resolved, request.ExpectedDefinition) {
		return PublishResult{}, fmt.Errorf(
			"%w: managed package did not resolve to its pinned artifact",
			basespec.ErrReferenceUnresolved,
		)
	}
	return PublishResult{
		Artifact:   resolved,
		Source:     published.Source,
		Generation: published.Generation,
		Refreshed:  true,
	}, nil
}

type RemoveRequest struct {
	Artifact       artifact.Artifact
	Package        basespec.Locator
	Plan           discovery.Plan
	RefreshPolicy  artifact.Policy
	AllowProtected bool
}

func (s *Service) Remove(
	ctx context.Context,
	request RemoveRequest,
) error {
	if err := s.validateRemoveRequest(ctx, request); err != nil {
		return err
	}

	current, err := s.dependencies.Artifacts.Get(ctx, request.Artifact.Ref())
	if err != nil {
		return err
	}
	if current.Revision != request.Artifact.Revision ||
		current.Binding != request.Artifact.Binding ||
		current.Adoption != artifact.AdoptionPinned {
		return fmt.Errorf(
			"%w: managed artifact changed before package removal",
			basespec.ErrConflict,
		)
	}
	collectionRef := collection.CollectionRef{
		RootID:       current.RootID,
		CollectionID: current.CollectionID,
	}
	if err := s.requireCollectionSource(
		ctx,
		collectionRef,
		current.Binding.SourceID,
		false,
	); err != nil {
		return err
	}
	if _, found := request.Plan.BySource()[current.Binding.SourceID]; !found {
		return fmt.Errorf(
			"%w: managed artifact removal plan does not include its source",
			basespec.ErrInvalid,
		)
	}

	state, err := s.dependencies.GetSourceState(
		ctx,
		current.RootID,
		current.Binding.SourceID,
	)
	if err != nil {
		return err
	}
	if err := validateManagedSourceState(
		state,
		current.RootID,
		current.Binding.SourceID,
		false,
	); err != nil {
		return err
	}
	if state.Source.ID != current.Binding.SourceID ||
		state.Source.RootID != current.RootID {
		return fmt.Errorf(
			"%w: managed source state does not match artifact binding",
			basespec.ErrInvalid,
		)
	}
	remove := s.dependencies.RemovePackage
	if request.AllowProtected {
		remove = s.dependencies.RemoveProtectedPackage
	}
	if _, err := remove(
		ctx,
		current.RootID,
		current.Binding.SourceID,
		state.Source.Revision,
		request.Package,
		state.Generation,
	); err != nil {
		return err
	}
	if err := s.dependencies.Artifacts.Purge(
		ctx,
		current.Ref(),
		current.Revision,
	); err != nil {
		return err
	}
	_, err = s.dependencies.Refresh.Refresh(
		ctx,
		collection.CollectionRef{
			RootID:       current.RootID,
			CollectionID: current.CollectionID,
		},
		request.Plan,
		request.RefreshPolicy,
	)
	return err
}

func (s *Service) validatePublishRequest(
	ctx context.Context,
	request PublishRequest,
) error {
	if s == nil {
		return basespec.ErrClosed
	}
	if err := request.Artifact.Validate(); err != nil {
		return err
	}
	if err := cryptoutil.ValidateDigest(request.ExpectedDefinition); err != nil {
		return err
	}
	if request.RefreshPolicy == nil {
		return fmt.Errorf("%w: managed artifact refresh policy is required", basespec.ErrInvalid)
	}
	if err := request.Plan.Validate(); err != nil {
		return err
	}
	if _, err := source.NormalizeManagedPackagePublication(request.Package); err != nil {
		return err
	}
	return s.requireMutable(ctx, request.Artifact.RootID, request.AllowProtected)
}

func (s *Service) validateRemoveRequest(
	ctx context.Context,
	request RemoveRequest,
) error {
	if s == nil {
		return basespec.ErrClosed
	}
	if err := request.Artifact.Validate(); err != nil {
		return err
	}
	if err := source.ValidateManagedPackageDirectory(request.Package); err != nil {
		return err
	}
	if request.RefreshPolicy == nil {
		return fmt.Errorf("%w: managed artifact refresh policy is required", basespec.ErrInvalid)
	}
	if err := request.Plan.Validate(); err != nil {
		return err
	}
	return s.requireMutable(ctx, request.Artifact.RootID, request.AllowProtected)
}

func (s *Service) requireMutable(
	ctx context.Context,
	rootID basespec.RootID,
	allowProtected bool,
) error {
	if ctx == nil {
		return fmt.Errorf(
			"%w: managed artifact context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if allowProtected {
		return protection.RequirePrivilegedInstaller(ctx)
	}
	return protection.RequireMutableRoot(ctx, s.dependencies.Policy, rootID)
}

func (s *Service) requireCollectionSource(
	ctx context.Context,
	ref collection.CollectionRef,
	sourceID basespec.SourceID,
	requireEnabled bool,
) error {
	value, err := s.dependencies.Collections.Get(ctx, ref)
	if err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return fmt.Errorf(
			"%w: collection reader returned an invalid collection: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if requireEnabled && !value.Enabled {
		return fmt.Errorf(
			"%w: managed artifact collection %q is disabled",
			basespec.ErrConflict,
			ref.CollectionID,
		)
	}
	attachment, err := s.dependencies.Collections.GetAttachment(
		ctx,
		ref,
		sourceID,
	)
	if err != nil {
		return err
	}
	if err := attachment.Validate(); err != nil {
		return fmt.Errorf(
			"%w: collection reader returned an invalid attachment: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	if requireEnabled && !attachment.Enabled {
		return fmt.Errorf(
			"%w: managed artifact source attachment is disabled",
			basespec.ErrConflict,
		)
	}
	return nil
}

func validateManagedSourceState(
	state SourceState,
	rootID basespec.RootID,
	sourceID basespec.SourceID,
	requireEnabled bool,
) error {
	if err := state.Source.Validate(); err != nil {
		return fmt.Errorf("%w: managed source state: %w", basespec.ErrInvalid, err)
	}
	if state.Source.RootID != rootID || state.Source.ID != sourceID {
		return fmt.Errorf(
			"%w: managed source state does not match its requested source",
			basespec.ErrInvalid,
		)
	}
	if err := basespec.ValidateSourceGeneration(state.Generation); err != nil {
		return err
	}
	if requireEnabled && !state.Source.Enabled {
		return fmt.Errorf("%w: managed source is disabled", basespec.ErrConflict)
	}
	return nil
}

func artifactMatchesDefinition(
	value artifact.Artifact,
	expected cryptoutil.Digest,
) bool {
	return value.State == artifact.StateAvailable &&
		value.ResolvedDefinition != nil &&
		*value.ResolvedDefinition == expected
}
