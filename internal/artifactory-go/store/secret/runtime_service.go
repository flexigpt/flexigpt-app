package secret

import (
	"context"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	overlayModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay/model"
	secretModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/value"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

type runtimeArtifactReader interface {
	Get(ctx context.Context, a artifactModel.ArtifactRef) (artifactModel.Artifact, error)
}

// RuntimeService is the only secret role that can read plaintext. Its dynamic
// method set does not include binding mutation or lifecycle maintenance.
type RuntimeService struct {
	repository RuntimeRepository
	artifacts  runtimeArtifactReader
	values     value.ValueStore
	namespaces map[overlayModel.Namespace]struct{}
}

func NewRuntimeService(
	repository RuntimeRepository,
	artifacts runtimeArtifactReader,
	namespaces []overlayModel.Namespace,
	values value.ValueStore,
) (*RuntimeService, error) {
	if repository == nil || artifacts == nil {
		return nil, fmt.Errorf("%w: secret runtime dependencies are incomplete", spec.ErrInvalid)
	}
	if values != nil {
		if err := value.ValidateValueStore(values); err != nil {
			return nil, err
		}
	}
	registered, err := namespaceSet(namespaces)
	if err != nil {
		return nil, err
	}
	return &RuntimeService{repository: repository, artifacts: artifacts, values: values, namespaces: registered}, nil
}

func (s *RuntimeService) ReadBinding(
	ctx context.Context,
	key secretModel.BindingKey,
	expectedRef secretModel.Ref,
) (string, secretModel.Binding, error) {
	if err := key.Validate(); err != nil {
		return "", secretModel.Binding{}, err
	}
	if err := expectedRef.Validate(); err != nil {
		return "", secretModel.Binding{}, err
	}
	if err := requireNamespace(s.namespaces, key.Namespace); err != nil {
		return "", secretModel.Binding{}, err
	}
	if s.values == nil {
		return "", secretModel.Binding{}, fmt.Errorf(
			"%w: Artifact Store secret value backend is not configured",
			spec.ErrUnsupported,
		)
	}
	artifactValue, err := s.artifacts.Get(ctx, key.Artifact)
	if err != nil {
		return "", secretModel.Binding{}, err
	}
	if artifactValue.State != artifactModel.StateAvailable {
		return "", secretModel.Binding{}, fmt.Errorf(
			"%w: Artifact %q is unavailable",
			spec.ErrReferenceUnresolved,
			artifactValue.ID,
		)
	}
	binding, found, err := s.repository.GetBinding(ctx, key)
	if err != nil {
		return "", secretModel.Binding{}, err
	}
	if !found || !binding.Active() {
		return "", secretModel.Binding{}, fmt.Errorf(
			"%w: secret binding is not configured",
			spec.ErrReferenceUnresolved,
		)
	}
	if *binding.Ref != expectedRef {
		return "", secretModel.Binding{}, fmt.Errorf(
			"%w: secret binding changed during runtime resolution",
			spec.ErrConflict,
		)
	}
	plaintext, err := s.values.Get(ctx, expectedRef)
	if err != nil {
		return "", secretModel.Binding{}, err
	}
	if secretModel.SHA256(plaintext) != binding.SHA256 {
		return "", secretModel.Binding{}, fmt.Errorf(
			"%w: physical secret value does not match binding SHA-256",
			spec.ErrDigestMismatch,
		)
	}
	return plaintext, binding.Clone(), nil
}
