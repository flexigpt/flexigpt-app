package bundle

import (
	"encoding/json"
	"fmt"
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	mcpDomainPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/policy"
	mcpDomainServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/server"
)

// ServerFromCanonicalBundle projects one server from a Bundle that has
// already been accepted and canonicalized by the Artifact Store shareable
// schema registry.
//
// It deliberately does not call CanonicalizeServer. MCP lifecycle code must
// not create a second portable-document validation path after registry
// canonicalization.
func ServerFromCanonicalBundle(
	bundle BundleDocument,
	name string,
) (mcpDomainServer.ServerDocument, error) {
	core, found := bundle.MCPServers[name]
	if !found {
		return mcpDomainServer.ServerDocument{}, fmt.Errorf(
			"%w: MCP server %q is not in the Bundle document",
			basespec.ErrNotFound,
			name,
		)
	}
	extension, found := bundle.BundleExtension.Servers[name]
	if !found {
		return mcpDomainServer.ServerDocument{}, fmt.Errorf(
			"%w: canonical MCP Bundle has no extension for server %q",
			basespec.ErrInvalid,
			name,
		)
	}
	return jsonutil.CloneJSON(mcpDomainServer.ServerDocument{
		Kind:           artifactbuiltin.ServerKind,
		SchemaID:       artifactbuiltin.ServerSchemaID,
		SchemaVersion:  artifactbuiltin.MCPSchemaVersion,
		LogicalName:    basespec.LogicalName(name),
		LogicalVersion: extension.LogicalVersion,
		DisplayName:    extension.DisplayName,
		Description:    extension.Description,
		Labels:         maps.Clone(extension.Labels),
		MCPServer:      core,
		Extension:      extension,
	})
}

func validateDocument(value BundleDocument) error {
	if value.Kind != artifactbuiltin.BundleKind ||
		value.SchemaID != artifactbuiltin.BundleSchemaID ||
		value.SchemaVersion != artifactbuiltin.MCPSchemaVersion {
		return fmt.Errorf(
			"%w: unsupported MCP Bundle schema",
			basespec.ErrInvalid,
		)
	}
	if err := basespec.ValidatePortableMetadata(
		value.LogicalName,
		value.LogicalVersion,
		value.DisplayName,
		value.Description,
		value.Labels,
		nil,
	); err != nil {
		return err
	}
	if len(value.MCPServers) > basespec.MaxDiscoveryCandidates {
		return fmt.Errorf(
			"%w: MCP Bundle server count exceeds limit",
			basespec.ErrInvalid,
		)
	}
	if len(value.BundleExtension.Policies) > basespec.MaxDiscoveryCandidates {
		return fmt.Errorf(
			"%w: MCP Bundle policy count exceeds limit",
			basespec.ErrInvalid,
		)
	}

	for name := range value.BundleExtension.Servers {
		if _, found := value.MCPServers[name]; !found {
			return fmt.Errorf(
				"%w: bundleExtension.servers[%q] has no mcpServers entry",
				basespec.ErrInvalid,
				name,
			)
		}
	}

	for name, core := range value.MCPServers {
		document, err := ServerFromCanonicalBundle(value, name)
		if err != nil {
			return err
		}
		document.MCPServer = core
		if err := document.Validate(); err != nil {
			return fmt.Errorf("MCP server %q: %w", name, err)
		}
	}

	for name, policyValue := range value.BundleExtension.Policies {
		if err := basespec.ValidatePortableName("MCP policy name", name); err != nil {
			return err
		}
		if string(policyValue.LogicalName) != name {
			return fmt.Errorf(
				"%w: policy map key %q does not match logicalName %q",
				basespec.ErrInvalid,
				name,
				policyValue.LogicalName,
			)
		}
		if err := policyValue.Validate(); err != nil {
			return fmt.Errorf("MCP policy %q: %w", name, err)
		}
	}
	if err := requiredPolicyReferences(value); err != nil {
		return err
	}

	return nil
}

func requiredPolicyReferences(
	value BundleDocument,
) error {
	for name, extension := range value.BundleExtension.Servers {
		if extension.Policy == nil || !extension.Policy.Required {
			continue
		}
		if _, found := value.BundleExtension.Policies[string(extension.Policy.Ref)]; found {
			continue
		}
		return fmt.Errorf(
			"%w: MCP server %q requires missing inline policy %q",
			basespec.ErrReferenceUnresolved,
			name,
			extension.Policy.Ref,
		)
	}
	return nil
}

func CanonicalizeBundle(
	input BundleDocument,
) (BundleDocument, json.RawMessage, error) {
	value, err := jsonutil.CloneJSON(input)
	if err != nil {
		return BundleDocument{}, nil, err
	}
	value.MCPServers = maps.Clone(value.MCPServers)
	value.BundleExtension.Servers = maps.Clone(value.BundleExtension.Servers)
	value.BundleExtension.Policies = maps.Clone(value.BundleExtension.Policies)
	value.Labels = maps.Clone(value.Labels)

	if value.MCPServers == nil {
		value.MCPServers = map[string]mcpDomainServer.CoreServer{}
	}
	if value.BundleExtension.Servers == nil {
		value.BundleExtension.Servers = map[string]mcpDomainServer.ServerExtension{}
	}
	if value.BundleExtension.Policies == nil {
		value.BundleExtension.Policies = map[string]mcpDomainPolicy.PolicyDocument{}
	}

	for name, core := range value.MCPServers {
		core = mcpDomainServer.NormalizeCoreServer(core)
		value.MCPServers[name] = core

		extension := mcpDomainServer.NormalizeServerExtension(
			name,
			value.BundleExtension.Servers[name],
		)
		value.BundleExtension.Servers[name] = extension
	}

	for name, policyValue := range value.BundleExtension.Policies {
		canonical, _, err := mcpDomainPolicy.CanonicalizePolicy(policyValue)
		if err != nil {
			return BundleDocument{}, nil, fmt.Errorf(
				"policy %q: %w",
				name,
				err,
			)
		}
		value.BundleExtension.Policies[name] = canonical
	}

	if err := value.Validate(); err != nil {
		return BundleDocument{}, nil, err
	}

	supplied := value.Digest
	value.Digest = ""
	calculated, err := cryptoutil.CanonicalDigest(value)
	if err != nil {
		return BundleDocument{}, nil, err
	}
	if supplied != "" && supplied != calculated {
		return BundleDocument{}, nil, fmt.Errorf(
			"%w: supplied MCP Bundle digest %q, calculated %q",
			basespec.ErrDigestMismatch,
			supplied,
			calculated,
		)
	}
	value.Digest = calculated

	raw, err := jsonutil.MarshalCanonicalObject(value, basespec.MaxDefinitionBytes)
	if err != nil {
		return BundleDocument{}, nil, err
	}
	return value, raw, nil
}

func DefinitionsForDocument(
	document BundleDocument,
) (map[basespec.SubresourceLocator]definition.Definition, error) {
	output := make(
		map[basespec.SubresourceLocator]definition.Definition,
		len(document.MCPServers)+len(document.BundleExtension.Policies),
	)
	for name := range document.MCPServers {
		serverDocument, err := ServerFromCanonicalBundle(document, name)
		if err != nil {
			return nil, err
		}
		value, err := mcpDomainServer.DefinitionForCanonicalServer(serverDocument)
		if err != nil {
			return nil, err
		}
		output[mcpDomainServer.ServerSubresource(basespec.LogicalName(name))] = value
	}
	for name, policyDocument := range document.BundleExtension.Policies {
		value, err := mcpDomainPolicy.DefinitionForCanonicalPolicy(policyDocument)
		if err != nil {
			return nil, err
		}
		output[mcpDomainPolicy.PolicySubresource(basespec.LogicalName(name))] = value
	}
	return output, nil
}

func DefinitionForArtifact(
	snapshot catalog.Snapshot,
	record artifact.Artifact,
) (definition.Definition, error) {
	if record.ResolvedDefinition == nil {
		return definition.Definition{}, fmt.Errorf(
			"%w: MCP Artifact %q has no resolved definition fingerprint",
			basespec.ErrReferenceUnresolved,
			record.ID,
		)
	}
	value, err := snapshot.DefinitionForOccurrence(catalog.OccurrenceKey{
		CollectionID:       record.CollectionID,
		SourceID:           record.Binding.SourceID,
		Locator:            record.Binding.Locator,
		SubresourceLocator: record.Binding.SubresourceLocator,
	})
	if err != nil {
		return definition.Definition{}, err
	}
	if value.Digest != *record.ResolvedDefinition {
		return definition.Definition{}, fmt.Errorf(
			"%w: MCP Artifact %q does not match its current catalog definition",
			basespec.ErrCatalogStale,
			record.ID,
		)
	}
	return value, nil
}
