package main

import (
	"context"
	"fmt"

	storeSecret "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret"
	secretModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/model/inferenceadapter"
)

type artifactModelCredentialResolver struct {
	secrets storeSecret.RuntimeAPI
}

func newArtifactModelCredentialResolver(
	secrets storeSecret.RuntimeAPI,
) (*artifactModelCredentialResolver, error) {
	if secrets == nil {
		return nil, fmt.Errorf(
			"%w: Artifact Store Model credential runtime is required",
			spec.ErrInvalid,
		)
	}
	return &artifactModelCredentialResolver{
		secrets: secrets,
	}, nil
}

func (r *artifactModelCredentialResolver) ResolveModelCredential(
	ctx context.Context,
	binding secretModel.Binding,
) (inferenceadapter.Credential, error) {
	if r == nil || r.secrets == nil {
		return inferenceadapter.Credential{}, spec.ErrClosed
	}
	if err := binding.Validate(); err != nil {
		return inferenceadapter.Credential{}, err
	}
	if !binding.Active() {
		return inferenceadapter.Credential{}, fmt.Errorf(
			"%w: Model Provider credential is not configured",
			spec.ErrReferenceUnresolved,
		)
	}

	value, current, err := r.secrets.ReadBinding(
		ctx,
		binding.Key,
		*binding.Ref,
	)
	if err != nil {
		return inferenceadapter.Credential{}, err
	}
	if current.Revision != binding.Revision ||
		current.SHA256 != binding.SHA256 ||
		!current.Active() ||
		*current.Ref != *binding.Ref {
		return inferenceadapter.Credential{}, fmt.Errorf(
			"%w: Model Provider credential changed during resolution",
			spec.ErrConflict,
		)
	}

	return inferenceadapter.Credential{
		APIKey:  value,
		Version: binding.SHA256,
	}, nil
}
