package main

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/compose/local"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/secret"
	"github.com/flexigpt/flexigpt-app/internal/model/inferenceadapter"
)

type artifactModelCredentialResolver struct {
	secrets local.SecretRuntimeAPI
}

func newArtifactModelCredentialResolver(
	secrets local.SecretRuntimeAPI,
) (*artifactModelCredentialResolver, error) {
	if secrets == nil {
		return nil, fmt.Errorf(
			"%w: Artifact Store Model credential runtime is required",
			model.ErrInvalid,
		)
	}
	return &artifactModelCredentialResolver{
		secrets: secrets,
	}, nil
}

func (r *artifactModelCredentialResolver) ResolveModelCredential(
	ctx context.Context,
	binding secret.Binding,
) (inferenceadapter.Credential, error) {
	if r == nil || r.secrets == nil {
		return inferenceadapter.Credential{}, model.ErrClosed
	}
	if err := binding.Validate(); err != nil {
		return inferenceadapter.Credential{}, err
	}
	if !binding.Active() {
		return inferenceadapter.Credential{}, fmt.Errorf(
			"%w: Model Provider credential is not configured",
			model.ErrReferenceUnresolved,
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
			model.ErrConflict,
		)
	}

	return inferenceadapter.Credential{
		APIKey:  value,
		Version: binding.SHA256,
	}, nil
}
