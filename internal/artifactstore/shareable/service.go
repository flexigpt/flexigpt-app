package shareable

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/protection"
)

type Service struct {
	registry    *Registry
	documents   DocumentRepository
	bindings    CollectionBindingRepository
	collections collection.Reader
	policy      protection.RootPolicy
}

func NewService(
	registry *Registry,
	documents DocumentRepository,
	bindings CollectionBindingRepository,
	collections collection.Reader,
	policy protection.RootPolicy,
) (*Service, error) {
	if registry == nil ||
		documents == nil ||
		bindings == nil ||
		collections == nil {
		return nil, fmt.Errorf(
			"%w: shareable document service dependencies are incomplete",
			basespec.ErrInvalid,
		)
	}
	return &Service{
		registry:    registry,
		documents:   documents,
		bindings:    bindings,
		collections: collections,
		policy:      policy,
	}, nil
}

func (s *Service) StoreCollection(
	ctx context.Context,
	ref collection.CollectionRef,
	raw []byte,
) (CollectionDocument, error) {
	if err := s.check(ctx); err != nil {
		return CollectionDocument{}, err
	}
	if err := ref.Validate(); err != nil {
		return CollectionDocument{}, err
	}
	if err := protection.RequireMutableRoot(ctx, s.policy, ref.RootID); err != nil {
		return CollectionDocument{}, err
	}

	local, err := s.collections.Get(ctx, ref)
	if err != nil {
		return CollectionDocument{}, err
	}
	parsed, err := s.registry.Canonicalize(ctx, raw)
	if err != nil {
		return CollectionDocument{}, err
	}
	if parsed.Key.Kind != local.Kind {
		return CollectionDocument{}, fmt.Errorf(
			"%w: shareable collection kind %q does not match local collection kind %q",
			basespec.ErrInvalid,
			parsed.Key.Kind,
			local.Kind,
		)
	}
	if err := s.documents.Put(
		ctx,
		ref.RootID,
		parsed.Digest,
		parsed.Raw,
	); err != nil {
		return CollectionDocument{}, err
	}

	binding := CollectionDocumentBinding{
		Collection: ref,
		Key:        parsed.Key,
		Digest:     parsed.Digest,
	}
	if err := s.bindings.PutCollectionDocument(ctx, binding); err != nil {
		return CollectionDocument{}, err
	}
	return CollectionDocument{
		Binding: binding,
		Raw:     append([]byte(nil), parsed.Raw...),
	}, nil
}

func (s *Service) GetCollection(
	ctx context.Context,
	ref collection.CollectionRef,
) (CollectionDocument, error) {
	if err := s.check(ctx); err != nil {
		return CollectionDocument{}, err
	}
	if err := ref.Validate(); err != nil {
		return CollectionDocument{}, err
	}
	binding, err := s.bindings.GetCollectionDocument(ctx, ref)
	if err != nil {
		return CollectionDocument{}, err
	}
	raw, err := s.documents.Get(ctx, ref.RootID, binding.Digest)
	if err != nil {
		return CollectionDocument{}, err
	}
	parsed, err := s.registry.Canonicalize(ctx, raw)
	if err != nil {
		return CollectionDocument{}, err
	}
	if parsed.Key != binding.Key || parsed.Digest != binding.Digest {
		return CollectionDocument{}, fmt.Errorf(
			"%w: stored shareable collection document does not match its binding",
			basespec.ErrDigestMismatch,
		)
	}
	return CollectionDocument{
		Binding: binding,
		Raw:     append([]byte(nil), parsed.Raw...),
	}, nil
}

func (s *Service) check(ctx context.Context) error {
	if s == nil ||
		s.registry == nil ||
		s.documents == nil ||
		s.bindings == nil ||
		s.collections == nil {
		return basespec.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf(
			"%w: shareable document context is nil",
			basespec.ErrInvalid,
		)
	}
	return ctx.Err()
}
