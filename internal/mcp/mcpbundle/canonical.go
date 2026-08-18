package mcpbundle

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/shareable"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/mcp/policy"
	"github.com/flexigpt/flexigpt-app/internal/mcp/server"
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
) (server.ServerDocument, error) {
	core, found := bundle.MCPServers[name]
	if !found {
		return server.ServerDocument{}, fmt.Errorf(
			"%w: MCP server %q is not in the Bundle document",
			basespec.ErrNotFound,
			name,
		)
	}
	extension, found := bundle.BundleExtension.Servers[name]
	if !found {
		return server.ServerDocument{}, fmt.Errorf(
			"%w: canonical MCP Bundle has no extension for server %q",
			basespec.ErrInvalid,
			name,
		)
	}
	return jsonutil.CloneJSON(server.ServerDocument{
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

func ValidateBundle(value BundleDocument) error {
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
		if err := basespec.ValidatePortableName("MCP server name", name); err != nil {
			return err
		}
		extension := value.BundleExtension.Servers[name]
		if err := server.ValidateServerParts(name, core, extension); err != nil {
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
		if err := policy.ValidatePolicy(policyValue); err != nil {
			return fmt.Errorf("MCP policy %q: %w", name, err)
		}
	}
	if err := validateRequiredBundlePolicyReferences(value); err != nil {
		return err
	}

	return nil
}

func validateRequiredBundlePolicyReferences(
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
		value.MCPServers = map[string]server.CoreServer{}
	}
	if value.BundleExtension.Servers == nil {
		value.BundleExtension.Servers = map[string]server.ServerExtension{}
	}
	if value.BundleExtension.Policies == nil {
		value.BundleExtension.Policies = map[string]policy.PolicyDocument{}
	}

	for name, core := range value.MCPServers {
		core = server.NormalizeCoreServer(core)
		value.MCPServers[name] = core

		extension := server.NormalizeServerExtension(
			name,
			value.BundleExtension.Servers[name],
		)
		value.BundleExtension.Servers[name] = extension
	}

	for name, policyValue := range value.BundleExtension.Policies {
		canonical, _, err := policy.CanonicalizePolicy(policyValue)
		if err != nil {
			return BundleDocument{}, nil, fmt.Errorf(
				"policy %q: %w",
				name,
				err,
			)
		}
		value.BundleExtension.Policies[name] = canonical
	}

	if err := ValidateBundle(value); err != nil {
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

// canonicalizeBundleBytes is the only public-lifecycle ingress for portable
// MCP Bundle bytes. The input remains raw until Artifact Store dispatches the
// expected registered codec, executes JSON Schema validation, verifies
// canonical JSON, and invokes MCP semantic canonicalization.
func (a *API) canonicalizeBundleBytes(
	ctx context.Context,
	raw []byte,
) (BundleDocument, shareable.ParsedDocument, error) {
	if a == nil || a.dependencies.ShareableDocuments == nil {
		return BundleDocument{}, shareable.ParsedDocument{}, fmt.Errorf(
			"%w: Artifact Store shareable document canonicalizer is unavailable",
			basespec.ErrClosed,
		)
	}
	if len(raw) == 0 {
		return BundleDocument{}, shareable.ParsedDocument{}, fmt.Errorf(
			"%w: MCP Bundle document is required",
			basespec.ErrInvalid,
		)
	}

	parsed, err := a.dependencies.ShareableDocuments.CanonicalizeExpected(
		ctx,
		artifactbuiltin.MCPBundleSchemaKey,
		raw,
	)
	if err != nil {
		return BundleDocument{}, shareable.ParsedDocument{}, fmt.Errorf(
			"canonicalize MCP Bundle through Artifact Store schema registry: %w",
			err,
		)
	}

	document, err := BundleFromParsedDocument(parsed)
	if err != nil {
		return BundleDocument{}, shareable.ParsedDocument{}, err
	}
	return document, parsed.Clone(), nil
}
