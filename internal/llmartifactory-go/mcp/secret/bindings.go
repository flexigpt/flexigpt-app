package secret

import (
	"context"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	overlayModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay/model"
	storeSecret "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret"
	secretModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

type artifactReader interface {
	Get(
		ctx context.Context,
		ref artifactModel.ArtifactRef,
	) (artifactModel.Artifact, error)
}

// BindingService owns the MCP logical-secret-selector to Artifact Store
// binding-key translation. It can create, replace, and clear bindings, but it
// deliberately has no plaintext read capability.
type BindingService struct {
	artifacts artifactReader
	bindings  storeSecret.API
	namespace overlayModel.Namespace
}

func NewBindingService(
	artifacts artifactReader,
	bindings storeSecret.API,
	namespace overlayModel.Namespace,
) (*BindingService, error) {
	if artifacts == nil || bindings == nil {
		return nil, fmt.Errorf(
			"%w: MCP secret binding dependencies are incomplete",
			spec.ErrInvalid,
		)
	}
	if err := namespace.Validate(); err != nil {
		return nil, err
	}

	return &BindingService{
		artifacts: artifacts,
		bindings:  bindings,
		namespace: namespace,
	}, nil
}

func (s *BindingService) SetMCPSecret(
	ctx context.Context,
	logicalRef string,
	value string,
) (hash string, nonEmpty bool, err error) {
	key, err := bindingKey(logicalRef, s.namespace)
	if err != nil {
		return "", false, err
	}

	server, err := s.availableServer(ctx, key.Artifact)
	if err != nil {
		return "", false, err
	}

	current, found, err := s.bindings.GetBinding(ctx, key)
	if err != nil {
		return "", false, err
	}

	expectedBindingRevision := uint64(0)
	if found {
		expectedBindingRevision = current.Revision
	}

	binding, err := s.bindings.ReplaceBinding(
		ctx,
		secretModel.ReplaceBindingRequest{
			Key:                      key,
			ExpectedArtifactRevision: server.Revision,
			ExpectedBindingRevision:  expectedBindingRevision,
			Value:                    value,
		},
	)
	if err != nil {
		return "", false, err
	}
	return binding.SHA256, binding.Active(), nil
}

// DeleteSecret is idempotent. Generic Artifact Store Secret lifecycle owns
// durable physical cleanup after the MCP logical binding is detached.
func (s *BindingService) DeleteSecret(
	ctx context.Context,
	logicalRef string,
) error {
	key, err := bindingKey(logicalRef, s.namespace)
	if err != nil {
		return err
	}

	server, err := s.artifacts.Get(ctx, key.Artifact)
	if err != nil {
		return err
	}

	binding, found, err := s.bindings.GetBinding(ctx, key)
	if err != nil {
		return err
	}
	if !found || !binding.Active() {
		return nil
	}

	return s.bindings.ClearBinding(
		ctx,
		secretModel.ClearBindingRequest{
			Key:                      key,
			ExpectedArtifactRevision: server.Revision,
			ExpectedBindingRevision:  binding.Revision,
		},
	)
}

func (s *BindingService) availableServer(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (artifactModel.Artifact, error) {
	record, err := s.artifacts.Get(ctx, ref)
	if err != nil {
		return artifactModel.Artifact{}, err
	}
	if record.State != artifactModel.StateAvailable {
		return artifactModel.Artifact{}, fmt.Errorf(
			"%w: MCP Server Artifact is unavailable",
			spec.ErrReferenceUnresolved,
		)
	}
	return record, nil
}

func bindingKey(
	logicalRef string,
	namespace overlayModel.Namespace,
) (secretModel.BindingKey, error) {
	selector, err := ParseMCPSecretRef(logicalRef)
	if err != nil {
		return secretModel.BindingKey{}, err
	}

	slot, err := ArtifactBindingSlot(selector)
	if err != nil {
		return secretModel.BindingKey{}, err
	}

	key := secretModel.BindingKey{
		Artifact:  selector.Server,
		Namespace: namespace,
		Slot:      slot,
	}
	if err := key.Validate(); err != nil {
		return secretModel.BindingKey{}, err
	}
	return key, nil
}
