package localstate

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/overlay"
	"github.com/flexigpt/flexigpt-app/internal/clockutil"
)

func (s *Service) GetStoreOverlay(
	ctx context.Context,
	namespace overlay.Namespace,
) (overlay.StoreRecord, bool, error) {
	if err := s.ready(ctx); err != nil {
		return overlay.StoreRecord{}, false, err
	}
	if err := s.requireStoreNamespace(namespace); err != nil {
		return overlay.StoreRecord{}, false, err
	}
	return s.repository.GetStoreOverlay(ctx, namespace)
}

func (s *Service) PutStoreOverlay(
	ctx context.Context,
	request overlay.StorePutRequest,
) (overlay.StoreRecord, error) {
	if err := s.ready(ctx); err != nil {
		return overlay.StoreRecord{}, err
	}
	if err := request.Validate(); err != nil {
		return overlay.StoreRecord{}, err
	}
	if err := s.requireStoreNamespace(request.Namespace); err != nil {
		return overlay.StoreRecord{}, err
	}

	payload, err := overlay.CanonicalPayload(request.Payload)
	if err != nil {
		return overlay.StoreRecord{}, err
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
	namespace overlay.Namespace,
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
			model.ErrInvalid,
		)
	}
	return s.repository.DeleteStoreOverlay(
		ctx,
		namespace,
		expectedRevision,
	)
}

func (s *Service) requireStoreNamespace(
	namespace overlay.Namespace,
) error {
	if err := namespace.Validate(); err != nil {
		return err
	}
	if _, found := s.storeNamespaces[namespace]; !found {
		return fmt.Errorf(
			"%w: store overlay namespace %q is not registered",
			model.ErrUnsupported,
			namespace,
		)
	}
	return nil
}
