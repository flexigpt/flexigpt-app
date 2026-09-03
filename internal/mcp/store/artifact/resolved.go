package artifact

import (
	"context"
	"fmt"
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type SecretResolver interface {
	ResolveSecret(
		ctx context.Context,
		ref string,
	) (string, error)
}

type EnvironmentResolver interface {
	ResolveEnvironment(
		ctx context.Context,
		name string,
	) (string, bool, error)
}

type Resolver interface {
	ResolveMCPServer(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) (Resolved, error)
}

func (r Resolved) Materialize(
	ctx context.Context,
	secrets SecretResolver,
	environment EnvironmentResolver,
) (RuntimeConfig, error) {
	if err := r.Validate(); err != nil {
		return RuntimeConfig{}, err
	}
	return r.MaterializeTrusted(ctx, secrets, environment)
}

// MaterializeForInspection resolves local installation inputs without
// requiring RuntimeEnabled. It is used only for sanitized auth-health
// projection and never resolves or exposes a secret value.
func (r Resolved) MaterializeForInspection(
	ctx context.Context,
	environment EnvironmentResolver,
) (RuntimeConfig, error) {
	if err := r.Validate(); err != nil {
		return RuntimeConfig{}, err
	}
	materialized, err := MaterializeInspectionValidated(
		ctx,
		r.Server,
		r.Document,
		r.Installation,
		environment,
	)
	if err != nil {
		return RuntimeConfig{}, err
	}
	return runtimeConfigFromMaterialized(r, materialized)
}

// MaterializeTrusted is the resolver-to-runtime fast path. Resolver output has
// already passed full Artifact, Catalog, Definition, policy, and installation
// validation. This method validates only values that do not exist until profile
// application and local substitution occur.
func (r Resolved) MaterializeTrusted(
	ctx context.Context,
	secrets SecretResolver,
	environment EnvironmentResolver,
) (RuntimeConfig, error) {
	if !r.RuntimeEnabled {
		return RuntimeConfig{}, fmt.Errorf(
			"%w: MCP Server is not enabled for this installation",
			basespec.ErrReferenceUnresolved,
		)
	}
	materialized, err := MaterializeValidated(
		ctx,
		r.Server,
		r.Document,
		r.Installation,
		secrets,
		environment,
	)
	if err != nil {
		return RuntimeConfig{}, err
	}
	return runtimeConfigFromMaterialized(r, materialized)
}

func (r Resolved) Validate() error {
	if err := r.Server.Validate(); err != nil {
		return err
	}
	if err := r.Collection.Validate(); err != nil {
		return err
	}
	if r.Server.RootID != r.Collection.RootID {
		return fmt.Errorf(
			"%w: MCP Server and Collection belong to different Roots",
			basespec.ErrInvalid,
		)
	}
	if r.ArtifactRevision == 0 || r.CatalogRevision == 0 {
		return fmt.Errorf(
			"%w: resolved MCP revisions are required",
			basespec.ErrInvalid,
		)
	}
	if err := cryptoutil.ValidateDigest(r.DefinitionDigest); err != nil {
		return err
	}
	if err := cryptoutil.ValidateDigest(r.SourceContentDigest); err != nil {
		return err
	}
	if err := basespec.ValidateSourceGeneration(r.SourceGeneration); err != nil {
		return err
	}
	if err := ValidateServer(r.Document); err != nil {
		return err
	}
	if err := ValidateServerData(r.Installation); err != nil {
		return err
	}
	if err := r.Policy.Validate(); err != nil {
		return err
	}
	return cryptoutil.ValidateDigest(r.Version)
}

func runtimeConfigFromMaterialized(
	r Resolved,
	materialized Materialized,
) (RuntimeConfig, error) {
	config := RuntimeConfig{
		Server:                    r.Server,
		Collection:                r.Collection,
		LogicalName:               string(r.Document.LogicalName),
		DisplayName:               r.Document.DisplayName,
		OAuthClientSecretRequired: materialized.ClientCredentialSecretRequired,
		TrustLevel:                r.Policy.Body.TrustLevel,
		DefaultPolicy:             r.Policy.Body.DefaultPolicy,
		ToolPolicies:              r.Policy.Body.ToolPolicies,
		AppsPolicy:                r.Policy.Body.AppsPolicy,
		SensitiveValues:           materialized.SensitiveValues,
	}

	switch materialized.Core.Type {
	case ServerTypeStdio:
		config.Transport = MCPTransportStdio
		config.Stdio = &MCPRuntimeStdioConfig{
			Command:          materialized.Core.Command,
			Args:             append([]string(nil), materialized.Core.Args...),
			Env:              maps.Clone(materialized.Core.Env),
			StartupTimeoutMS: materialized.TimeoutMS,
		}

	case ServerTypeHTTP:
		config.Transport = MCPTransportStreamableHTTP
		config.StreamableHTTP = &MCPRuntimeStreamableHTTPConfig{
			URL:                         materialized.Core.URL,
			TimeoutMS:                   materialized.TimeoutMS,
			AuthMode:                    materialized.Auth.Mode,
			Headers:                     maps.Clone(materialized.Core.Headers),
			ClientCredentialRef:         materialized.ClientCredentialRef,
			ClientIDMetadataDocumentURL: materialized.Auth.ClientIDMetadataDocumentURL,
		}

	default:
		return RuntimeConfig{}, fmt.Errorf(
			"%w: unsupported materialized MCP transport",
			basespec.ErrInvalid,
		)
	}

	return config, nil
}
