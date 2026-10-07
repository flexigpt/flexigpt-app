package secret

import (
	"context"
	"fmt"

	overlayModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay/model"
	storeSecret "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret"
	secretModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

type bindingReader interface {
	GetBinding(
		ctx context.Context,
		key secretModel.BindingKey,
	) (secretModel.Binding, bool, error)
}

// RuntimeResolver is the MCP trusted plaintext-resolution capability. It has
// no mutation or cleanup method set.
type RuntimeResolver struct {
	bindings  bindingReader
	runtime   storeSecret.RuntimeAPI
	namespace overlayModel.Namespace
}

func NewRuntimeResolver(
	bindings bindingReader,
	runtime storeSecret.RuntimeAPI,
	namespace overlayModel.Namespace,
) (*RuntimeResolver, error) {
	if bindings == nil || runtime == nil {
		return nil, fmt.Errorf(
			"%w: MCP secret runtime dependencies are incomplete",
			spec.ErrInvalid,
		)
	}
	if err := namespace.Validate(); err != nil {
		return nil, err
	}

	return &RuntimeResolver{
		bindings:  bindings,
		runtime:   runtime,
		namespace: namespace,
	}, nil
}

func (r *RuntimeResolver) ResolveSecret(
	ctx context.Context,
	logicalRef string,
) (string, error) {
	key, err := bindingKey(logicalRef, r.namespace)
	if err != nil {
		return "", err
	}

	binding, found, err := r.bindings.GetBinding(ctx, key)
	if err != nil {
		return "", err
	}
	if !found || !binding.Active() {
		return "", fmt.Errorf(
			"%w: %w: MCP secret is unavailable",
			spec.ErrReferenceUnresolved,
			ErrNotFound,
		)
	}

	value, current, err := r.runtime.ReadBinding(
		ctx,
		key,
		*binding.Ref,
	)
	if err != nil {
		return "", err
	}
	if current.Revision != binding.Revision ||
		current.SHA256 != binding.SHA256 ||
		!current.Active() ||
		*current.Ref != *binding.Ref {
		return "", fmt.Errorf(
			"%w: MCP secret changed during resolution",
			spec.ErrConflict,
		)
	}
	return value, nil
}

// TrustedRuntimeStore is the explicit adapter passed only to trusted MCP OAuth
// runtime construction. It combines the separately owned binding mutation and
// plaintext resolution roles solely because the external runtime contract
// requires both. It is never supplied to MCP management or family services.
type TrustedRuntimeStore struct {
	bindings *BindingService
	resolver *RuntimeResolver
}

func NewTrustedRuntimeStore(
	bindings *BindingService,
	resolver *RuntimeResolver,
) (*TrustedRuntimeStore, error) {
	if bindings == nil || resolver == nil {
		return nil, fmt.Errorf(
			"%w: MCP trusted secret runtime dependencies are incomplete",
			spec.ErrInvalid,
		)
	}
	return &TrustedRuntimeStore{
		bindings: bindings,
		resolver: resolver,
	}, nil
}

func (s *TrustedRuntimeStore) ResolveSecret(
	ctx context.Context,
	logicalRef string,
) (string, error) {
	if s == nil || s.resolver == nil {
		return "", spec.ErrClosed
	}
	return s.resolver.ResolveSecret(ctx, logicalRef)
}

func (s *TrustedRuntimeStore) SetMCPSecret(
	ctx context.Context,
	logicalRef string,
	value string,
) (hash string, nonEmpty bool, err error) {
	if s == nil || s.bindings == nil {
		return "", false, spec.ErrClosed
	}
	return s.bindings.SetMCPSecret(ctx, logicalRef, value)
}

func (s *TrustedRuntimeStore) DeleteSecret(
	ctx context.Context,
	logicalRef string,
) error {
	if s == nil || s.bindings == nil {
		return spec.ErrClosed
	}
	return s.bindings.DeleteSecret(ctx, logicalRef)
}
