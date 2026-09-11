package consumerapi

import (
	"context"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/workspacecollectionv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	workspaceDomain "github.com/flexigpt/flexigpt-app/internal/workspace/store/domain"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/domain/artifactadapter"
	"github.com/flexigpt/flexigpt-app/internal/workspace/store/domain/collectiondata"
)

type Service struct {
	collections     compositionapi.CollectionAPI
	sources         compositionapi.SourceAPI
	workspaceRootID root.RootID
}

func NewService(
	collections compositionapi.CollectionAPI,
	sources compositionapi.SourceAPI,
	workspaceRootID root.RootID,
) (*Service, error) {
	if collections == nil || sources == nil {
		return nil, fmt.Errorf(
			"%w: Workspace service dependencies are incomplete",
			workspaceDomain.ErrInvalidWorkspace,
		)
	}
	if err := workspaceRootID.Validate(); err != nil {
		return nil, err
	}

	return &Service{
		collections:     collections,
		sources:         sources,
		workspaceRootID: workspaceRootID,
	}, nil
}

func (s *Service) List(
	ctx context.Context,
	rootID root.RootID,
) ([]workspaceDomain.Workspace, error) {
	if err := rootID.Validate(); err != nil {
		return nil, err
	}
	if err := s.requireWorkspaceRoot(rootID); err != nil {
		return nil, err
	}

	collections, err := s.collections.ListByRoot(ctx, rootID)
	if err != nil {
		return nil, err
	}

	output := make([]workspaceDomain.Workspace, 0)
	for _, value := range collections {
		if value.Kind != workspacecollectionv1.WorkspaceCollectionKind {
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

func (s *Service) Get(
	ctx context.Context,
	ref collection.CollectionRef,
) (workspaceDomain.Workspace, error) {
	if err := ref.Validate(); err != nil {
		return workspaceDomain.Workspace{}, err
	}
	if err := s.requireWorkspaceRoot(ref.RootID); err != nil {
		return workspaceDomain.Workspace{}, err
	}

	value, err := s.collections.Get(ctx, ref)
	if err != nil {
		return workspaceDomain.Workspace{}, err
	}
	if err := value.Validate(); err != nil {
		return workspaceDomain.Workspace{}, fmt.Errorf(
			"%w: Collection reader returned an invalid Workspace Collection: %w",
			workspaceDomain.ErrInvalidWorkspace,
			err,
		)
	}
	if value.Ref() != ref {
		return workspaceDomain.Workspace{}, fmt.Errorf(
			"%w: Collection reader returned another Workspace",
			workspaceDomain.ErrInvalidWorkspace,
		)
	}
	if value.Kind != workspacecollectionv1.WorkspaceCollectionKind {
		return workspaceDomain.Workspace{}, fmt.Errorf(
			"%w: collection %q has kind %q",
			workspaceDomain.ErrNotWorkspace,
			ref.CollectionID,
			value.Kind,
		)
	}

	data, err := collectiondata.DecodeCollectionData(value.Data)
	if err != nil {
		return workspaceDomain.Workspace{}, fmt.Errorf(
			"%w: %w",
			workspaceDomain.ErrInvalidWorkspace,
			err,
		)
	}

	attachments, err := s.collections.ListAttachments(ctx, ref)
	if err != nil {
		return workspaceDomain.Workspace{}, err
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
			return workspaceDomain.Workspace{}, err
		}
		sources = append(sources, sourceValue)
	}

	mode, primarySourceID, err := artifactadapter.DeriveWorkspaceTopology(
		value,
		data,
		attachments,
		sources,
	)
	if err != nil {
		return workspaceDomain.Workspace{}, err
	}

	sort.Slice(sources, func(left, right int) bool {
		return sources[left].ID < sources[right].ID
	})

	return workspaceDomain.Workspace{
		Collection:      value,
		Data:            data,
		Mode:            mode,
		PrimarySourceID: primarySourceID,
		Attachments:     attachments,
		Sources:         sources,
	}, nil
}

// requireWorkspaceRoot receives an already-validated Root ID.
func (s *Service) requireWorkspaceRoot(
	rootID root.RootID,
) error {
	if rootID != s.workspaceRootID {
		return fmt.Errorf(
			"%w: Workspace Root must be %q",
			workspaceDomain.ErrInvalidWorkspace,
			s.workspaceRootID,
		)
	}
	return nil
}
