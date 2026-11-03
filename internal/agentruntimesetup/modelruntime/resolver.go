package modelruntime

import (
	"context"
	"fmt"

	storeSecret "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret"
	secretModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// ModelCredentialsResolver is the trusted application-side bridge from a Model Provider
// credential binding to runtime-only inference credentials.
type ModelCredentialsResolver struct {
	secrets storeSecret.RuntimeAPI
}

func NewModelCredentialsResolver(
	secrets storeSecret.RuntimeAPI,
) (*ModelCredentialsResolver, error) {
	if secrets == nil {
		return nil, fmt.Errorf(
			"%w: Artifact Store Model credential runtime is required",
			storeSpec.ErrInvalid,
		)
	}
	return &ModelCredentialsResolver{secrets: secrets}, nil
}

func (r *ModelCredentialsResolver) ResolveModelCredential(
	ctx context.Context,
	binding secretModel.Binding,
) (Credential, error) {
	if err := binding.Validate(); err != nil {
		return Credential{}, err
	}
	if !binding.Active() {
		return Credential{}, fmt.Errorf(
			"%w: Model Provider credential binding is not configured",
			storeSpec.ErrReferenceUnresolved,
		)
	}

	value, current, err := r.secrets.ReadBinding(
		ctx,
		binding.Key,
		*binding.Ref,
	)
	if err != nil {
		return Credential{}, err
	}
	if current.Revision != binding.Revision ||
		current.SHA256 != binding.SHA256 ||
		!current.Active() ||
		*current.Ref != *binding.Ref {
		return Credential{}, fmt.Errorf(
			"%w: Model Provider credential changed during resolution",
			storeSpec.ErrConflict,
		)
	}

	return Credential{
		APIKey:  value,
		Version: binding.SHA256,
	}, nil
}
