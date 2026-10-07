package modelcredentials

import (
	"context"
	"fmt"

	storeSecret "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret"
	secretModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/modelruntime"
)

// Resolver is the trusted application-side bridge from a Model Provider
// credential binding to runtime-only inference credentials.
type Resolver struct {
	secrets storeSecret.RuntimeAPI
}

func New(
	secrets storeSecret.RuntimeAPI,
) (*Resolver, error) {
	if secrets == nil {
		return nil, fmt.Errorf(
			"%w: Artifact Store Model credential runtime is required",
			spec.ErrInvalid,
		)
	}
	return &Resolver{secrets: secrets}, nil
}

func (r *Resolver) ResolveModelCredential(
	ctx context.Context,
	binding secretModel.Binding,
) (modelruntime.Credential, error) {
	if err := binding.Validate(); err != nil {
		return modelruntime.Credential{}, err
	}
	if !binding.Active() {
		return modelruntime.Credential{}, fmt.Errorf(
			"%w: Model Provider credential binding is not configured",
			spec.ErrReferenceUnresolved,
		)
	}

	value, current, err := r.secrets.ReadBinding(
		ctx,
		binding.Key,
		*binding.Ref,
	)
	if err != nil {
		return modelruntime.Credential{}, err
	}
	if current.Revision != binding.Revision ||
		current.SHA256 != binding.SHA256 ||
		!current.Active() ||
		*current.Ref != *binding.Ref {
		return modelruntime.Credential{}, fmt.Errorf(
			"%w: Model Provider credential changed during resolution",
			spec.ErrConflict,
		)
	}

	return modelruntime.Credential{
		APIKey:  value,
		Version: binding.SHA256,
	}, nil
}
