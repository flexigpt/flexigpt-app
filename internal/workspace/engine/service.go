package engine

import (
	"context"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
)

type Service struct {
	roots   workspaceRootStore
	sources sourceSummaryLookup
}

func NewService(
	roots workspaceRootStore,
	sources sourceSummaryLookup,
) (*Service, error) {
	if roots == nil || sources == nil {
		return nil, fmt.Errorf(
			"%w: Workspace service dependencies are incomplete",
			ErrInvalidWorkspace,
		)
	}
	return &Service{
		roots:   roots,
		sources: sources,
	}, nil
}

func (s *Service) CreateEmpty(
	ctx context.Context,
	request EmptyWorkspaceRequest,
) (Workspace, error) {
	data := RootData{
		Mode:      ModeEmpty,
		Discovery: request.Discovery,
	}
	raw, err := encodeRootData(data)
	if err != nil {
		return Workspace{}, err
	}
	createdRoot, _, err := s.roots.Create(
		ctx,
		root.RootDraft{
			Kind:        RootKind,
			DisplayName: request.DisplayName,
			Description: request.Description,
			Enabled:     true,
			Data:        raw,
		},
		nil,
	)
	if err != nil {
		return Workspace{}, err
	}
	return s.Get(ctx, createdRoot.ID)
}

func (s *Service) CreateFilesystem(
	ctx context.Context,
	request FilesystemWorkspaceRequest,
) (Workspace, error) {
	sourceValue, err := s.sources.Get(ctx, request.PrimarySourceID)
	if err != nil {
		return Workspace{}, err
	}
	primaryOperation, _ := attachmentOperationFor(RolePrimary)
	if sourceValue.Kind != primaryOperation.requiredSourceKind {
		return Workspace{}, fmt.Errorf(
			"%w: primary source must have kind %q",
			ErrInvalidWorkspace,
			primaryOperation.requiredSourceKind,
		)
	}
	if !sourceValue.Enabled {
		return Workspace{}, fmt.Errorf(
			"%w: primary source must be enabled",
			ErrInvalidWorkspace,
		)
	}
	data := RootData{
		Mode:            ModeFilesystem,
		PrimarySourceID: sourceValue.ID,
		Discovery:       request.Discovery,
	}
	raw, err := encodeRootData(data)
	if err != nil {
		return Workspace{}, err
	}
	attachmentData, err := encodeAttachmentData(AttachmentData{})
	if err != nil {
		return Workspace{}, err
	}
	createdRoot, _, err := s.roots.Create(
		ctx,
		root.RootDraft{
			Kind:        RootKind,
			DisplayName: request.DisplayName,
			Description: request.Description,
			Enabled:     true,
			Data:        raw,
		},
		[]root.AttachmentDraft{{
			SourceID: sourceValue.ID,
			Role:     RolePrimary,
			Enabled:  true,
			Data:     attachmentData,
		}},
	)
	if err != nil {
		return Workspace{}, err
	}
	return s.Get(ctx, createdRoot.ID)
}

func (s *Service) List(
	ctx context.Context,
) ([]Workspace, error) {
	roots, err := s.roots.List(ctx)
	if err != nil {
		return nil, err
	}
	output := make([]Workspace, 0)
	for _, root := range roots {
		if root.Kind != RootKind {
			continue
		}
		value, err := s.Get(ctx, root.ID)
		if err != nil {
			return nil, err
		}
		output = append(output, value)
	}
	return output, nil
}

func (s *Service) Update(
	ctx context.Context,
	request UpdateRequest,
) (Workspace, error) {
	current, err := s.Get(ctx, request.RootID)
	if err != nil {
		return Workspace{}, err
	}
	data := current.Data
	data.Discovery = request.Discovery

	raw, err := encodeRootData(data)
	if err != nil {
		return Workspace{}, err
	}
	_, err = s.roots.Update(
		ctx,
		request.RootID,
		root.RootUpdate{
			ExpectedRevision: request.ExpectedRevision,
			DisplayName:      request.DisplayName,
			Description:      request.Description,
			Enabled:          request.Enabled,
			Data:             raw,
		},
	)
	if err != nil {
		return Workspace{}, err
	}
	return s.Get(ctx, request.RootID)
}

func (s *Service) Attach(
	ctx context.Context,
	request AttachRequest,
) (Workspace, error) {
	if err := validateRole(request.Role); err != nil {
		return Workspace{}, err
	}
	operation, _ := attachmentOperationFor(request.Role)
	if !operation.canAttach {
		return Workspace{}, ErrPrimarySourceImmutable
	}
	if _, err := s.Get(ctx, request.RootID); err != nil {
		return Workspace{}, err
	}
	sourceValue, err := s.sources.Get(ctx, request.SourceID)
	if err != nil {
		return Workspace{}, err
	}
	if !sourceValue.Enabled && request.Enabled {
		return Workspace{}, fmt.Errorf(
			"%w: enabled attachment cannot use disabled source",
			ErrInvalidWorkspace,
		)
	}
	data, err := encodeAttachmentData(request.Data)
	if err != nil {
		return Workspace{}, err
	}
	if _, _, err := s.roots.Attach(
		ctx,
		request.RootID,
		request.ExpectedRootRevision,
		root.AttachmentDraft{
			SourceID: request.SourceID,
			Role:     request.Role,
			Enabled:  request.Enabled,
			Data:     data,
		},
	); err != nil {
		return Workspace{}, err
	}
	return s.Get(ctx, request.RootID)
}

func (s *Service) UpdateAttachment(
	ctx context.Context,
	request UpdateAttachmentRequest,
) (Workspace, error) {
	if err := validateRole(request.Role); err != nil {
		return Workspace{}, err
	}
	targetOperation, _ := attachmentOperationFor(request.Role)
	if !targetOperation.canAttach {
		return Workspace{}, ErrPrimarySourceImmutable
	}
	if _, err := s.Get(ctx, request.RootID); err != nil {
		return Workspace{}, err
	}
	current, err := s.roots.GetAttachment(
		ctx,
		request.RootID,
		request.SourceID,
	)
	if err != nil {
		return Workspace{}, err
	}
	currentOperation, _ := attachmentOperationFor(current.Role)
	if !currentOperation.canAttach {
		return Workspace{}, ErrPrimarySourceImmutable
	}
	sourceValue, err := s.sources.Get(ctx, request.SourceID)
	if err != nil {
		return Workspace{}, err
	}
	if request.Enabled && !sourceValue.Enabled {
		return Workspace{}, fmt.Errorf(
			"%w: enabled attachment cannot use disabled source",
			ErrInvalidWorkspace,
		)
	}
	data, err := encodeAttachmentData(request.Data)
	if err != nil {
		return Workspace{}, err
	}
	if _, _, err := s.roots.UpdateAttachment(
		ctx,
		request.RootID,
		request.SourceID,
		root.AttachmentUpdate{
			ExpectedRootRevision:       request.ExpectedRootRevision,
			ExpectedAttachmentRevision: request.ExpectedAttachmentRevision,
			Role:                       request.Role,
			Enabled:                    request.Enabled,
			Data:                       data,
		},
	); err != nil {
		return Workspace{}, err
	}
	return s.Get(ctx, request.RootID)
}

func (s *Service) Detach(
	ctx context.Context,
	rootID artifactstore.RootID,
	sourceID artifactstore.SourceID,
	expectedRootRevision uint64,
	expectedAttachmentRevision uint64,
) (Workspace, error) {
	if _, err := s.Get(ctx, rootID); err != nil {
		return Workspace{}, err
	}
	attachment, err := s.roots.GetAttachment(ctx, rootID, sourceID)
	if err != nil {
		return Workspace{}, err
	}
	operation, _ := attachmentOperationFor(attachment.Role)
	if !operation.canAttach {
		return Workspace{}, ErrPrimarySourceImmutable
	}
	if _, err := s.roots.Detach(
		ctx,
		rootID,
		sourceID,
		expectedRootRevision,
		expectedAttachmentRevision,
	); err != nil {
		return Workspace{}, err
	}
	return s.Get(ctx, rootID)
}

func (s *Service) Delete(
	ctx context.Context,
	rootID artifactstore.RootID,
	expectedRevision uint64,
) (root.Root, error) {
	if _, err := s.Get(ctx, rootID); err != nil {
		return root.Root{}, err
	}
	return s.roots.Delete(ctx, rootID, expectedRevision)
}

func (s *Service) Get(
	ctx context.Context,
	rootID artifactstore.RootID,
) (Workspace, error) {
	r, err := s.roots.Get(ctx, rootID)
	if err != nil {
		return Workspace{}, err
	}
	if r.Kind != RootKind {
		return Workspace{}, fmt.Errorf(
			"%w: root %q has kind %q",
			ErrNotWorkspace,
			rootID,
			r.Kind,
		)
	}
	data, err := decodeRootData(r.Data)
	if err != nil {
		return Workspace{}, fmt.Errorf("%w: %w", ErrInvalidWorkspace, err)
	}
	attachments, err := s.roots.ListAttachments(ctx, rootID)
	if err != nil {
		return Workspace{}, err
	}
	sources := make([]source.Summary, 0, len(attachments))
	for _, attachment := range attachments {
		value, err := s.sources.Get(ctx, attachment.SourceID)
		if err != nil {
			return Workspace{}, err
		}
		sources = append(sources, value)
	}
	if err := validateWorkspaceState(r, data, attachments, sources); err != nil {
		return Workspace{}, err
	}
	sort.Slice(sources, func(left, right int) bool {
		return sources[left].ID < sources[right].ID
	})
	return Workspace{
		Root:        r,
		Data:        data,
		Attachments: attachments,
		Sources:     sources,
	}, nil
}

func attachmentOperationFor(
	role artifactstore.AttachmentRole,
) (attachmentOperation, bool) {
	for _, operation := range attachmentOperationMatrix {
		if operation.role == role {
			return operation, true
		}
	}
	return attachmentOperation{}, false
}
