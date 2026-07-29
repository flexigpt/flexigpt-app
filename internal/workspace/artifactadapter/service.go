package artifactadapter

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/workspace/attachmentdata"
	"github.com/flexigpt/flexigpt-app/internal/workspace/collectiondata"
	"github.com/flexigpt/flexigpt-app/internal/workspace/spec"
)

type Service struct {
	collections             workspaceCollectionStore
	sources                 sourceSummaryLookup
	discoveryPolicyRevision string
}

func NewService(
	collections workspaceCollectionStore,
	sources sourceSummaryLookup,
	discoveryPolicyRevision string,
) (*Service, error) {
	if collections == nil || sources == nil {
		return nil, fmt.Errorf(
			"%w: Workspace service dependencies are incomplete",
			spec.ErrInvalidWorkspace,
		)
	}
	if err := basespec.ValidateRequiredText(
		"workspace discovery policy revision",
		discoveryPolicyRevision,
		basespec.MaxVersionBytes,
	); err != nil {
		return nil, err
	}
	return &Service{
		collections:             collections,
		sources:                 sources,
		discoveryPolicyRevision: discoveryPolicyRevision,
	}, nil
}

func (s *Service) CreateEmpty(
	ctx context.Context,
	request spec.EmptyWorkspaceRequest,
) (spec.Workspace, error) {
	if err := basespec.ValidateRootID(request.RootID); err != nil {
		return spec.Workspace{}, err
	}
	data := spec.CollectionData{
		DiscoveryPolicyRevision: s.discoveryPolicyRevision,
		Discovery:               request.Discovery,
	}
	raw, err := collectiondata.EncodeCollectionData(data)
	if err != nil {
		return spec.Workspace{}, err
	}
	created, _, err := s.collections.Create(
		ctx,
		request.RootID,
		collection.Draft{
			Kind:        spec.CollectionKind,
			DisplayName: request.DisplayName,
			Description: request.Description,
			Enabled:     true,
			Data:        raw,
		},
		nil,
	)
	if err != nil {
		return spec.Workspace{}, err
	}
	return s.Get(ctx, created.Ref())
}

func (s *Service) CreateFilesystem(
	ctx context.Context,
	request spec.FilesystemWorkspaceRequest,
) (spec.Workspace, error) {
	if err := basespec.ValidateRootID(request.RootID); err != nil {
		return spec.Workspace{}, err
	}
	if err := basespec.ValidateSourceID(request.PrimarySourceID); err != nil {
		return spec.Workspace{}, err
	}
	sourceValue, err := s.sources.Get(
		ctx,
		request.RootID,
		request.PrimarySourceID,
	)
	if err != nil {
		return spec.Workspace{}, err
	}
	primaryOperation, _ := attachmentdata.AttachmentOperationFor(spec.RolePrimary)
	if sourceValue.Kind != primaryOperation.RequiredSourceKind {
		return spec.Workspace{}, fmt.Errorf(
			"%w: primary source must have kind %q",
			spec.ErrInvalidWorkspace,
			primaryOperation.RequiredSourceKind,
		)
	}
	if !sourceValue.Enabled {
		return spec.Workspace{}, fmt.Errorf(
			"%w: primary source must be enabled",
			spec.ErrInvalidWorkspace,
		)
	}
	data := spec.CollectionData{
		DiscoveryPolicyRevision: s.discoveryPolicyRevision,
		Discovery:               request.Discovery,
	}
	raw, err := collectiondata.EncodeCollectionData(data)
	if err != nil {
		return spec.Workspace{}, err
	}
	attachmentData, err := attachmentdata.EncodeAttachmentData(spec.AttachmentData{})
	if err != nil {
		return spec.Workspace{}, err
	}
	created, _, err := s.collections.Create(
		ctx,
		request.RootID,
		collection.Draft{
			Kind:        spec.CollectionKind,
			DisplayName: request.DisplayName,
			Description: request.Description,
			Enabled:     true,
			Data:        raw,
		},
		[]collection.AttachmentDraft{{
			SourceID: sourceValue.ID,
			Role:     spec.RolePrimary,
			Enabled:  true,
			Data:     attachmentData,
		}},
	)
	if err != nil {
		return spec.Workspace{}, err
	}
	return s.Get(ctx, created.Ref())
}

func (s *Service) List(
	ctx context.Context,
	rootID basespec.RootID,
) ([]spec.Workspace, error) {
	if err := basespec.ValidateRootID(rootID); err != nil {
		return nil, err
	}
	collections, err := s.collections.ListByRoot(ctx, rootID)
	if err != nil {
		return nil, err
	}
	output := make([]spec.Workspace, 0)
	for _, value := range collections {
		if value.Kind != spec.CollectionKind {
			continue
		}
		workspaceValue, err := s.Get(ctx, value.Ref())
		if err != nil {
			return nil, err
		}
		output = append(output, workspaceValue)

	}
	return output, nil
}

func (s *Service) Update(
	ctx context.Context,
	request spec.UpdateRequest,
) (spec.Workspace, error) {
	if err := request.Workspace.Validate(); err != nil {
		return spec.Workspace{}, err
	}
	current, err := s.Get(ctx, request.Workspace)
	if err != nil {
		return spec.Workspace{}, err
	}
	data := current.Data
	data.DiscoveryPolicyRevision = s.discoveryPolicyRevision
	data.Discovery = request.Discovery

	raw, err := collectiondata.EncodeCollectionData(data)
	if err != nil {
		return spec.Workspace{}, err
	}
	_, err = s.collections.Update(
		ctx,
		request.Workspace,
		collection.Update{
			ExpectedRevision: request.ExpectedRevision,
			DisplayName:      request.DisplayName,
			Description:      request.Description,
			Enabled:          request.Enabled,
			Data:             raw,
		},
	)
	if err != nil {
		return spec.Workspace{}, err
	}
	return s.Get(ctx, request.Workspace)
}

func (s *Service) Attach(
	ctx context.Context,
	request spec.AttachRequest,
) (spec.Workspace, error) {
	if err := validateRole(request.Role); err != nil {
		return spec.Workspace{}, err
	}
	operation, _ := attachmentdata.AttachmentOperationFor(request.Role)
	if !operation.CanAttach {
		return spec.Workspace{}, spec.ErrPrimarySourceImmutable
	}
	if request.ExpectedCollectionRevision == 0 {
		return spec.Workspace{}, fmt.Errorf(
			"%w: expected collection revision is required",
			spec.ErrInvalidWorkspace,
		)
	}
	if _, err := s.Get(ctx, request.Workspace); err != nil {
		return spec.Workspace{}, err
	}
	sourceValue, err := s.sources.Get(
		ctx,
		request.Workspace.RootID,
		request.SourceID,
	)
	if err != nil {
		return spec.Workspace{}, err
	}
	if !sourceValue.Enabled && request.Enabled {
		return spec.Workspace{}, fmt.Errorf(
			"%w: enabled attachment cannot use disabled source",
			spec.ErrInvalidWorkspace,
		)
	}
	data, err := attachmentdata.EncodeAttachmentData(request.Data)
	if err != nil {
		return spec.Workspace{}, err
	}
	if err := attachmentdata.ValidateAttachmentDataForRole(request.Role, request.Data); err != nil {
		return spec.Workspace{}, err
	}
	if _, _, err := s.collections.Attach(

		ctx,
		request.Workspace,
		request.ExpectedCollectionRevision,
		collection.AttachmentDraft{
			SourceID: request.SourceID,
			Role:     request.Role,
			Enabled:  request.Enabled,
			Data:     data,
		},
	); err != nil {
		return spec.Workspace{}, err
	}
	return s.Get(ctx, request.Workspace)
}

func (s *Service) UpdateAttachment(
	ctx context.Context,
	request spec.UpdateAttachmentRequest,
) (spec.Workspace, error) {
	if err := validateRole(request.Role); err != nil {
		return spec.Workspace{}, err
	}
	targetOperation, _ := attachmentdata.AttachmentOperationFor(request.Role)
	if !targetOperation.CanAttach {
		return spec.Workspace{}, spec.ErrPrimarySourceImmutable
	}
	if _, err := s.Get(ctx, request.Workspace); err != nil {
		return spec.Workspace{}, err
	}
	current, err := s.collections.GetAttachment(
		ctx,
		request.Workspace,
		request.SourceID,
	)
	if err != nil {
		return spec.Workspace{}, err
	}
	currentOperation, _ := attachmentdata.AttachmentOperationFor(current.Role)
	if !currentOperation.CanAttach {
		return spec.Workspace{}, spec.ErrPrimarySourceImmutable
	}
	sourceValue, err := s.sources.Get(
		ctx,
		request.Workspace.RootID,
		request.SourceID,
	)
	if err != nil {
		return spec.Workspace{}, err
	}
	if request.Enabled && !sourceValue.Enabled {
		return spec.Workspace{}, fmt.Errorf(
			"%w: enabled attachment cannot use disabled source",
			spec.ErrInvalidWorkspace,
		)
	}
	data, err := attachmentdata.EncodeAttachmentData(request.Data)
	if err != nil {
		return spec.Workspace{}, err
	}
	if err := attachmentdata.ValidateAttachmentDataForRole(request.Role, request.Data); err != nil {
		return spec.Workspace{}, err
	}
	if _, _, err := s.collections.UpdateAttachment(
		ctx,
		request.Workspace,
		request.SourceID,
		collection.AttachmentUpdate{
			ExpectedCollectionRevision: request.ExpectedCollectionRevision,
			ExpectedAttachmentRevision: request.ExpectedAttachmentRevision,
			Role:                       request.Role,
			Enabled:                    request.Enabled,
			Data:                       data,
		},
	); err != nil {
		return spec.Workspace{}, err
	}
	return s.Get(ctx, request.Workspace)
}

// SetPrimary explicitly transitions a Workspace between empty and filesystem
// modes, or replaces its existing primary Source. Generic attachment APIs
// intentionally cannot mutate the primary relationship.
func (s *Service) SetPrimary(
	ctx context.Context,
	request spec.SetPrimaryRequest,
) (spec.Workspace, error) {
	if err := request.Workspace.Validate(); err != nil {
		return spec.Workspace{}, err
	}
	if request.ExpectedCollectionRevision == 0 {
		return spec.Workspace{}, fmt.Errorf(
			"%w: expected collection revision is required",
			spec.ErrInvalidWorkspace,
		)
	}
	if request.Clear == (request.SourceID != "") {
		return spec.Workspace{}, fmt.Errorf(
			"%w: exactly one of sourceID or clear is required",
			spec.ErrInvalidWorkspace,
		)
	}
	if request.SourceID != "" {
		if err := basespec.ValidateSourceID(request.SourceID); err != nil {
			return spec.Workspace{}, err
		}
	}
	if request.PreviousSourceID != "" {
		if err := basespec.ValidateSourceID(request.PreviousSourceID); err != nil {
			return spec.Workspace{}, err
		}
	}

	current, err := s.Get(ctx, request.Workspace)
	if err != nil {
		return spec.Workspace{}, err
	}
	if current.Collection.Revision != request.ExpectedCollectionRevision {
		return spec.Workspace{}, basespec.ErrConflict
	}

	if current.PrimarySourceID == "" {
		if request.PreviousSourceID != "" ||
			request.PreviousAttachmentRevision != 0 {
			return spec.Workspace{}, basespec.ErrConflict
		}
		if request.Clear {
			return spec.Workspace{}, fmt.Errorf(
				"%w: an empty Workspace has no primary Source to clear",
				spec.ErrInvalidWorkspace,
			)
		}
		if err := s.requirePrimarySource(
			ctx,
			request.Workspace.RootID,
			request.SourceID,
		); err != nil {
			return spec.Workspace{}, err
		}

		data, err := attachmentdata.EncodeAttachmentData(spec.AttachmentData{})
		if err != nil {
			return spec.Workspace{}, err
		}
		if _, _, err := s.collections.Attach(
			ctx,
			request.Workspace,
			request.ExpectedCollectionRevision,
			collection.AttachmentDraft{
				SourceID: request.SourceID,
				Role:     spec.RolePrimary,
				Enabled:  true,
				Data:     data,
			},
		); err != nil {
			return spec.Workspace{}, err
		}
		return s.Get(ctx, request.Workspace)
	}

	if request.PreviousSourceID == "" ||
		request.PreviousSourceID != current.PrimarySourceID {
		return spec.Workspace{}, basespec.ErrConflict
	}
	if request.PreviousAttachmentRevision == 0 {
		return spec.Workspace{}, fmt.Errorf(
			"%w: expected primary attachment revision is required",
			spec.ErrInvalidWorkspace,
		)
	}

	previous, err := s.collections.GetAttachment(
		ctx,
		request.Workspace,
		current.PrimarySourceID,
	)
	if err != nil {
		return spec.Workspace{}, err
	}
	if previous.Revision != request.PreviousAttachmentRevision {
		return spec.Workspace{}, basespec.ErrConflict
	}

	if request.Clear {
		if _, err := s.collections.Detach(
			ctx,
			request.Workspace,
			current.PrimarySourceID,
			request.ExpectedCollectionRevision,
			request.PreviousAttachmentRevision,
		); err != nil {
			return spec.Workspace{}, err
		}
		return s.Get(ctx, request.Workspace)
	}

	if request.SourceID == current.PrimarySourceID {
		return current, nil
	}

	if err := s.requirePrimarySource(
		ctx,
		request.Workspace.RootID,
		request.SourceID,
	); err != nil {
		return spec.Workspace{}, err
	}

	if _, err := s.collections.GetAttachment(
		ctx,
		request.Workspace,
		request.SourceID,
	); err == nil {
		return spec.Workspace{}, fmt.Errorf(
			"%w: replacement primary source is already attached to the Workspace",
			basespec.ErrConflict,
		)
	} else if !errors.Is(err, basespec.ErrAttachmentNotFound) {
		return spec.Workspace{}, err
	}

	data, err := attachmentdata.EncodeAttachmentData(spec.AttachmentData{})
	if err != nil {
		return spec.Workspace{}, err
	}
	_, _, err = s.collections.ReplaceAttachment(
		ctx,
		request.Workspace,
		collection.AttachmentReplacement{
			ExpectedCollectionRevision: request.ExpectedCollectionRevision,
			PreviousSourceID:           current.PrimarySourceID,
			PreviousAttachmentRevision: request.PreviousAttachmentRevision,
			Replacement: collection.AttachmentDraft{
				SourceID: request.SourceID,
				Role:     spec.RolePrimary,
				Enabled:  true,
				Data:     data,
			},
		},
	)
	if err != nil {
		return spec.Workspace{}, err
	}
	return s.Get(ctx, request.Workspace)
}

func (s *Service) ReplacePrimary(
	ctx context.Context,
	request spec.ReplacePrimaryRequest,
) (spec.Workspace, error) {
	return s.SetPrimary(ctx, spec.SetPrimaryRequest{
		Workspace:                  request.Workspace,
		ExpectedCollectionRevision: request.ExpectedCollectionRevision,
		PreviousSourceID:           request.PreviousSourceID,
		PreviousAttachmentRevision: request.PreviousAttachmentRevision,
		SourceID:                   request.SourceID,
	})
}

func (s *Service) Detach(
	ctx context.Context,
	ref collection.CollectionRef,
	sourceID basespec.SourceID,
	expectedCollectionRevision uint64,
	expectedAttachmentRevision uint64,
) (spec.Workspace, error) {
	if expectedCollectionRevision == 0 || expectedAttachmentRevision == 0 {
		return spec.Workspace{}, fmt.Errorf(
			"%w: expected collection and attachment revisions are required",
			spec.ErrInvalidWorkspace,
		)
	}
	if _, err := s.Get(ctx, ref); err != nil {
		return spec.Workspace{}, err
	}
	attachment, err := s.collections.GetAttachment(ctx, ref, sourceID)
	if err != nil {
		return spec.Workspace{}, err
	}
	operation, _ := attachmentdata.AttachmentOperationFor(attachment.Role)
	if !operation.CanAttach {
		return spec.Workspace{}, spec.ErrPrimarySourceImmutable
	}
	if _, err := s.collections.Detach(
		ctx,
		ref,
		sourceID,
		expectedCollectionRevision,
		expectedAttachmentRevision,
	); err != nil {
		return spec.Workspace{}, err
	}
	return s.Get(ctx, ref)
}

func (s *Service) Retire(
	ctx context.Context,
	ref collection.CollectionRef,
	expectedRevision uint64,
) (collection.Collection, error) {
	if _, err := s.Get(ctx, ref); err != nil {
		return collection.Collection{}, err
	}
	return s.collections.Retire(ctx, ref, expectedRevision)
}

// Purge destructively removes a retired Workspace Collection and its
// Collection-scoped metadata. It deliberately verifies the persisted kind
// before delegating to generic Collection persistence, so Workspace APIs
// cannot purge another domain's retired Collection.
func (s *Service) Purge(
	ctx context.Context,
	ref collection.CollectionRef,
	expectedRevision uint64,
) error {
	if err := ref.Validate(); err != nil {
		return err
	}
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected Workspace revision is required",
			spec.ErrInvalidWorkspace,
		)
	}
	value, err := s.collections.GetRetired(ctx, ref)
	if err != nil {
		return err
	}
	if value.Kind != spec.CollectionKind {
		return fmt.Errorf("%w: collection %q", spec.ErrNotWorkspace, ref.CollectionID)
	}
	if value.Revision != expectedRevision {
		return basespec.ErrConflict
	}
	return s.collections.Purge(ctx, ref, expectedRevision)
}

// PrepareRefresh converges the local policy revision before planning a
// publication. A policy revision is local metadata, so a user-triggered
// refresh is the appropriate point to persist an implementation upgrade.
func (s *Service) PrepareRefresh(
	ctx context.Context,
	ref collection.CollectionRef,
) (spec.Workspace, error) {
	current, err := s.Get(ctx, ref)
	if err != nil {
		return spec.Workspace{}, err
	}
	if !current.Collection.Enabled ||
		current.Data.DiscoveryPolicyRevision == s.discoveryPolicyRevision {
		return current, nil
	}

	data := current.Data
	data.DiscoveryPolicyRevision = s.discoveryPolicyRevision
	raw, err := collectiondata.EncodeCollectionData(data)
	if err != nil {
		return spec.Workspace{}, err
	}
	if _, err := s.collections.Update(
		ctx,
		ref,
		collection.Update{
			ExpectedRevision: current.Collection.Revision,
			DisplayName:      current.Collection.DisplayName,
			Description:      current.Collection.Description,
			Enabled:          current.Collection.Enabled,
			Data:             raw,
		},
	); err != nil {
		return spec.Workspace{}, err
	}
	return s.Get(ctx, ref)
}

func (s *Service) Get(
	ctx context.Context,
	ref collection.CollectionRef,
) (spec.Workspace, error) {
	if err := ref.Validate(); err != nil {
		return spec.Workspace{}, err
	}
	value, err := s.collections.Get(ctx, ref)
	if err != nil {
		return spec.Workspace{}, err
	}
	if err := value.Validate(); err != nil {
		return spec.Workspace{}, fmt.Errorf(
			"%w: Collection reader returned an invalid Workspace Collection: %w",
			spec.ErrInvalidWorkspace,
			err,
		)
	}
	if value.Ref() != ref {
		return spec.Workspace{}, fmt.Errorf(
			"%w: Collection reader returned another Workspace",
			spec.ErrInvalidWorkspace,
		)
	}
	if value.Kind != spec.CollectionKind {
		return spec.Workspace{}, fmt.Errorf(
			"%w: collection %q has kind %q",
			spec.ErrNotWorkspace,
			ref.CollectionID,
			value.Kind,
		)
	}
	data, err := collectiondata.DecodeCollectionData(value.Data)
	if err != nil {
		return spec.Workspace{}, fmt.Errorf("%w: %w", spec.ErrInvalidWorkspace, err)
	}
	attachments, err := s.collections.ListAttachments(ctx, ref)
	if err != nil {
		return spec.Workspace{}, err
	}

	sort.Slice(attachments, func(left, right int) bool {
		return attachments[left].SourceID < attachments[right].SourceID
	})

	sources := make([]source.Summary, 0, len(attachments))
	for _, attachment := range attachments {
		sourceValue, err := s.sources.Get(
			ctx,
			ref.RootID,
			attachment.SourceID,
		)
		if err != nil {
			return spec.Workspace{}, err
		}
		sources = append(sources, sourceValue)
	}
	mode, primarySourceID, err := validateWorkspaceState(
		value,
		data,
		attachments,
		sources,
	)
	if err != nil {
		return spec.Workspace{}, err
	}
	sort.Slice(sources, func(left, right int) bool {
		return sources[left].ID < sources[right].ID
	})
	return spec.Workspace{
		Collection:      value,
		Data:            data,
		Mode:            mode,
		PrimarySourceID: primarySourceID,
		Attachments:     attachments,
		Sources:         sources,
	}, nil
}

func (s *Service) requirePrimarySource(
	ctx context.Context,
	rootID basespec.RootID,
	sourceID basespec.SourceID,
) error {
	sourceValue, err := s.sources.Get(ctx, rootID, sourceID)
	if err != nil {
		return err
	}
	operation, found := attachmentdata.AttachmentOperationFor(spec.RolePrimary)
	if !found {
		return fmt.Errorf(
			"%w: Workspace primary attachment policy is unavailable",
			spec.ErrInvalidWorkspace,
		)
	}
	if sourceValue.Kind != operation.RequiredSourceKind ||
		!sourceValue.Enabled {
		return fmt.Errorf(
			"%w: primary source must be an enabled filesystem source",
			spec.ErrInvalidWorkspace,
		)
	}
	return nil
}
