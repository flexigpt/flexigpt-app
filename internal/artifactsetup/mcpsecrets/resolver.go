package mcpsecrets

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	storeSecret "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret"
	secretModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/secret/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	mcpOverlay "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/overlay"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/secret"
)

// Resolver binds MCP logical secret selectors to generic Artifact Store secret
// bindings. It owns application assembly adaptation only; MCP owns selector
// grammar and Artifact Store owns physical secret lifecycle.
type Resolver struct {
	artifacts artifact.API
	bindings  storeSecret.API
	runtime   storeSecret.RuntimeAPI
}

func New(
	artifacts artifact.API,
	bindings storeSecret.API,
	runtime storeSecret.RuntimeAPI,
) (*Resolver, error) {
	if artifacts == nil || bindings == nil || runtime == nil {
		return nil, fmt.Errorf(
			"%w: Artifact Store MCP secret dependencies are incomplete",
			spec.ErrInvalid,
		)
	}

	return &Resolver{
		artifacts: artifacts,
		bindings:  bindings,
		runtime:   runtime,
	}, nil
}

func (r *Resolver) SetMCPSecret(
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

func (r *Resolver) ResolveSecret(
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
			secret.ErrNotFound,
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

// DeleteSecret is idempotent. Generic Secret lifecycle owns durable physical
// value deletion after the binding is detached.
func (r *Resolver) DeleteSecret(
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

func (r *Resolver) bindingKey(
	logicalRef string,
) (secretModel.BindingKey, error) {
	selector, err := secret.ParseMCPSecretRef(logicalRef)
	if err != nil {
		return secretModel.BindingKey{}, err
	}

	slot, err := secret.ArtifactBindingSlot(selector)
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

func (r *Resolver) availableServer(
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
