package internal

import (
	"context"
	"fmt"

	overlayModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
)

func (s *Service) GetStoreOverlay(
	ctx context.Context,
	namespace overlayModel.Namespace,
) (overlayModel.StoreRecord, bool, error) {
	if err := s.ready(ctx); err != nil {
		return overlayModel.StoreRecord{}, false, err
	}
	if err := s.requireStoreNamespace(namespace); err != nil {
		return overlayModel.StoreRecord{}, false, err
	}
	return s.repository.GetStoreOverlay(ctx, namespace)
}

func (s *Service) PutStoreOverlay(
	ctx context.Context,
	request overlayModel.StorePutRequest,
) (overlayModel.StoreRecord, error) {
	if err := s.ready(ctx); err != nil {
		return overlayModel.StoreRecord{}, err
	}
	if err := request.Validate(); err != nil {
		return overlayModel.StoreRecord{}, err
	}
	if err := s.requireStoreNamespace(request.Namespace); err != nil {
		return overlayModel.StoreRecord{}, err
	}

	payload, err := overlayModel.CanonicalPayload(request.Payload)
	if err != nil {
		return overlayModel.StoreRecord{}, err
	}
	request.Payload = payload

	return s.repository.PutStoreOverlay(
		ctx,
		request,
		clockutil.NowUTC(s.clock),
	)
}

func (s *Service) DeleteStoreOverlay(
	ctx context.Context,
	namespace overlayModel.Namespace,
	expectedRevision uint64,
) error {
	if err := s.ready(ctx); err != nil {
		return err
	}
	if err := s.requireStoreNamespace(namespace); err != nil {
		return err
	}
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected store overlay revision is required",
			spec.ErrInvalid,
		)
	}
	return s.repository.DeleteStoreOverlay(
		ctx,
		namespace,
		expectedRevision,
	)
}

func (s *Service) requireStoreNamespace(
	namespace overlayModel.Namespace,
) error {
	if err := namespace.Validate(); err != nil {
		return err
	}
	if _, found := s.storeNamespaces[namespace]; !found {
		return fmt.Errorf(
			"%w: store overlay namespace %q is not registered",
			spec.ErrUnsupported,
			namespace,
		)
	}
	return nil
}
