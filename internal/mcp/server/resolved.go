package server

import (
	"context"
	"fmt"
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/mcp/installation"
	"github.com/flexigpt/flexigpt-app/internal/mcp/policy"
	"github.com/flexigpt/flexigpt-app/internal/mcp/schema"
	mcpSpec "github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

type RuntimeConfig struct {
	Server     artifact.ArtifactRef
	Collection collection.CollectionRef

	LogicalName string
	DisplayName string

	Transport      mcpSpec.MCPTransportType
	Stdio          *mcpSpec.MCPStdioConfig
	StreamableHTTP *mcpSpec.MCPStreamableHTTPConfig

	TrustLevel    mcpSpec.MCPTrustLevel
	DefaultPolicy mcpSpec.MCPServerPolicy
	ToolPolicies  map[string]mcpSpec.MCPToolPolicyOverride
	AppsPolicy    mcpSpec.MCPAppsPolicy

	SensitiveValues []string
}

type Resolved struct {
	Server     artifact.ArtifactRef
	Collection collection.CollectionRef

	ArtifactRevision uint64
	CatalogRevision  uint64

	DefinitionDigest    cryptoutil.Digest
	SourceContentDigest cryptoutil.Digest
	SourceGeneration    string

	Document     schema.ServerDocument
	Installation installation.ServerData
	Policy       policy.Effective

	InstallationRevision uint64
	RuntimeEnabled       bool
	BuiltIn              bool
	Version              cryptoutil.Digest
}

type Resolver interface {
	ResolveMCPServer(
		ctx context.Context,
		ref artifact.ArtifactRef,
	) (Resolved, error)
}

func (r Resolved) Materialize(
	ctx context.Context,
	secrets installation.SecretResolver,
	environment installation.EnvironmentResolver,
) (RuntimeConfig, error) {
	if err := r.Validate(); err != nil {
		return RuntimeConfig{}, err
	}
	return r.MaterializeTrusted(ctx, secrets, environment)
}

// MaterializeTrusted is the resolver-to-runtime fast path. Resolver output has
// already passed full Artifact, Catalog, Definition, policy, and installation
// validation. This method validates only values that do not exist until profile
// application and local substitution occur.
func (r Resolved) MaterializeTrusted(
	ctx context.Context,
	secrets installation.SecretResolver,
	environment installation.EnvironmentResolver,
) (RuntimeConfig, error) {
	if !r.RuntimeEnabled {
		return RuntimeConfig{}, fmt.Errorf(
			"%w: MCP Server is not enabled for this installation",
			basespec.ErrReferenceUnresolved,
		)
	}

	materialized, err := installation.MaterializeValidated(
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
	if err := schema.ValidateServer(r.Document); err != nil {
		return err
	}
	if err := installation.ValidateServerData(r.Installation); err != nil {
		return err
	}
	if err := r.Policy.Validate(); err != nil {
		return err
	}
	return cryptoutil.ValidateDigest(r.Version)
}

func runtimeConfigFromMaterialized(
	r Resolved,
	materialized installation.Materialized,
) (RuntimeConfig, error) {
	config := RuntimeConfig{
		Server:          r.Server,
		Collection:      r.Collection,
		LogicalName:     string(r.Document.LogicalName),
		DisplayName:     r.Document.DisplayName,
		TrustLevel:      r.Policy.Body.TrustLevel,
		DefaultPolicy:   r.Policy.Body.DefaultPolicy,
		ToolPolicies:    r.Policy.Body.ToolPolicies,
		AppsPolicy:      r.Policy.Body.AppsPolicy,
		SensitiveValues: materialized.SensitiveValues,
	}

	switch materialized.Core.Type {
	case schema.ServerTypeStdio:
		config.Transport = mcpSpec.MCPTransportStdio
		config.Stdio = &mcpSpec.MCPStdioConfig{
			Command: materialized.Core.Command,
			Args:    append([]string(nil), materialized.Core.Args...),
			Env:     maps.Clone(materialized.Core.Env),
		}

	case schema.ServerTypeHTTP:
		config.Transport = mcpSpec.MCPTransportStreamableHTTP
		config.StreamableHTTP = &mcpSpec.MCPStreamableHTTPConfig{
			URL:                         materialized.Core.URL,
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
