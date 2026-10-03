package main

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret"
	secretModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	mcpDomainSecret "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/secret"
	mcpOverlay "github.com/flexigpt/flexigpt-app/internal/mcp/store/overlay"
)

type artifactMCPSecretResolver struct {
	artifacts artifact.API
	bindings  secret.API
	runtime   secret.RuntimeAPI
}

func newArtifactMCPSecretResolver(
	artifacts artifact.API,
	bindings secret.API,
	runtime secret.RuntimeAPI,
) (*artifactMCPSecretResolver, error) {
	if artifacts == nil || bindings == nil || runtime == nil {
		return nil, fmt.Errorf(
			"%w: Artifact Store MCP secret dependencies are incomplete",
			spec.ErrInvalid,
		)
	}

	return &artifactMCPSecretResolver{
		artifacts: artifacts,
		bindings:  bindings,
		runtime:   runtime,
	}, nil
}

// SetMCPSecret preserves the existing MCP aggregate/runtime secret writer
// contract while storing the value through Artifact Store secret bindings.
func (r *artifactMCPSecretResolver) SetMCPSecret(
	ctx context.Context,
	logicalRef string,
	value string,
) (hash string, nonEmpty bool, err error) {
	key, err := r.bindingKey(logicalRef)
	if err != nil {
		return "", false, err
	}

	server, err := r.availableServer(ctx, key.Artifact)
	if err != nil {
		return "", false, err
	}

	current, found, err := r.bindings.GetBinding(ctx, key)
	if err != nil {
		return "", false, err
	}
	expectedBindingRevision := uint64(0)
	if found {
		expectedBindingRevision = current.Revision
	}

	binding, err := r.bindings.ReplaceBinding(
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

func (r *artifactMCPSecretResolver) ResolveSecret(
	ctx context.Context,
	logicalRef string,
) (string, error) {
	key, err := r.bindingKey(logicalRef)
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
			mcpDomainSecret.ErrNotFound,
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

// DeleteSecret is idempotent. It is used by current-installation cleanup,
// OAuth token cleanup, and managed MCP update flows.
func (r *artifactMCPSecretResolver) DeleteSecret(
	ctx context.Context,
	logicalRef string,
) error {
	key, err := r.bindingKey(logicalRef)
	if err != nil {
		return err
	}

	server, err := r.artifacts.Get(ctx, key.Artifact)
	if err != nil {
		return err
	}

	binding, found, err := r.bindings.GetBinding(ctx, key)
	if err != nil {
		return err
	}
	if !found || !binding.Active() {
		return nil
	}

	return r.bindings.ClearBinding(
		ctx,
		secretModel.ClearBindingRequest{
			Key:                      key,
			ExpectedArtifactRevision: server.Revision,
			ExpectedBindingRevision:  binding.Revision,
		},
	)
}

func (r *artifactMCPSecretResolver) bindingKey(
	logicalRef string,
) (secretModel.BindingKey, error) {
	if r == nil || r.artifacts == nil ||
		r.bindings == nil || r.runtime == nil {
		return secretModel.BindingKey{}, spec.ErrClosed
	}

	selector, err := mcpDomainSecret.ParseMCPSecretRef(logicalRef)
	if err != nil {
		return secretModel.BindingKey{}, err
	}
	slot, err := mcpDomainSecret.ArtifactBindingSlot(selector)
	if err != nil {
		return secretModel.BindingKey{}, err
	}

	key := secretModel.BindingKey{
		Artifact:  selector.Server,
		Namespace: mcpOverlay.InstallationNamespace,
		Slot:      slot,
	}
	if err := key.Validate(); err != nil {
		return secretModel.BindingKey{}, err
	}
	return key, nil
}

func (r *artifactMCPSecretResolver) availableServer(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (artifactModel.Artifact, error) {
	record, err := r.artifacts.Get(ctx, ref)
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
