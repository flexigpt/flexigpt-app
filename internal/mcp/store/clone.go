package store

import (
	"maps"
	"slices"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

func cloneBundleMap(in map[bundleitemutils.BundleID]spec.MCPBundle) map[bundleitemutils.BundleID]spec.MCPBundle {
	out := make(map[bundleitemutils.BundleID]spec.MCPBundle, len(in))
	for id, b := range in {
		out[id] = cloneBundle(b)
	}
	return out
}

func cloneBundle(in spec.MCPBundle) spec.MCPBundle {
	out := in
	out.SoftDeletedAt = clonePtr(in.SoftDeletedAt)
	return out
}

func cloneAllServerMaps(
	in map[bundleitemutils.BundleID]map[spec.MCPServerID]spec.MCPServerConfig,
) map[bundleitemutils.BundleID]map[spec.MCPServerID]spec.MCPServerConfig {
	out := make(map[bundleitemutils.BundleID]map[spec.MCPServerID]spec.MCPServerConfig, len(in))
	for bid, servers := range in {
		out[bid] = cloneServerMap(servers)
	}
	return out
}

func cloneServerMap(in map[spec.MCPServerID]spec.MCPServerConfig) map[spec.MCPServerID]spec.MCPServerConfig {
	out := make(map[spec.MCPServerID]spec.MCPServerConfig, len(in))
	for id, cfg := range in {
		out[id] = cloneServerConfig(cfg)
	}
	return out
}

func cloneServerConfig(in spec.MCPServerConfig) spec.MCPServerConfig {
	out := in

	if in.Stdio != nil {
		cp := *in.Stdio
		cp.Args = slices.Clone(in.Stdio.Args)
		cp.Env = maps.Clone(in.Stdio.Env)
		cp.SecretEnvRefs = maps.Clone(in.Stdio.SecretEnvRefs)
		out.Stdio = &cp
	}

	if in.StreamableHTTP != nil {
		cp := *in.StreamableHTTP
		cp.Headers = maps.Clone(in.StreamableHTTP.Headers)
		cp.SecretHeaderRefs = maps.Clone(in.StreamableHTTP.SecretHeaderRefs)
		out.StreamableHTTP = &cp
	}

	if in.DefaultPolicy == (spec.MCPServerPolicy{}) {
		out.DefaultPolicy = spec.DefaultMCPServerPolicy()
	}

	out.ToolPolicies = maps.Clone(in.ToolPolicies)

	if in.AppsPolicy != nil {
		cp := *in.AppsPolicy
		out.AppsPolicy = &cp
	}
	out.Setup = cloneServerSetup(in.Setup)
	out.SoftDeletedAt = clonePtr(in.SoftDeletedAt)
	return out
}

func cloneServerSetup(in *spec.MCPServerSetup) *spec.MCPServerSetup {
	if in == nil {
		return nil
	}
	out := *in
	out.Inputs = make([]spec.MCPServerSetupInput, len(in.Inputs))
	for i, input := range in.Inputs {
		cp := input
		cp.OAuthClientCredentials = clonePtr(input.OAuthClientCredentials)
		cp.HTTPHeader = clonePtr(input.HTTPHeader)
		cp.StdioEnv = clonePtr(input.StdioEnv)
		cp.StreamableHTTPURL = clonePtr(input.StreamableHTTPURL)
		cp.ClientIDMetadataDocumentURL = clonePtr(input.ClientIDMetadataDocumentURL)
		out.Inputs[i] = cp
	}
	return &out
}

func cloneDiscoverySnapshot(in spec.MCPDiscoverySnapshot) spec.MCPDiscoverySnapshot {
	out := in

	if in.ServerInfo != nil {
		cp := *in.ServerInfo
		out.ServerInfo = &cp
	}
	if in.ServerCapabilities != nil {
		cp := *in.ServerCapabilities
		cp.Experimental = maps.Clone(in.ServerCapabilities.Experimental)
		cp.Extensions = maps.Clone(in.ServerCapabilities.Extensions)
		out.ServerCapabilities = &cp
	}

	out.Tools = slices.Clone(in.Tools)
	for i := range out.Tools {
		out.Tools[i].InputSchema = maps.Clone(in.Tools[i].InputSchema)
		out.Tools[i].OutputSchema = maps.Clone(in.Tools[i].OutputSchema)
		if in.Tools[i].Annotations != nil {
			cp := *in.Tools[i].Annotations
			out.Tools[i].Annotations = &cp
		}
		if in.Tools[i].App != nil {
			cp := *in.Tools[i].App
			cp.Visibility = slices.Clone(in.Tools[i].App.Visibility)
			out.Tools[i].App = &cp
		}
	}

	out.Resources = slices.Clone(in.Resources)
	for i := range out.Resources {
		out.Resources[i].Annotations = maps.Clone(in.Resources[i].Annotations)
	}

	out.ResourceTemplates = slices.Clone(in.ResourceTemplates)
	for i := range out.ResourceTemplates {
		out.ResourceTemplates[i].Arguments = maps.Clone(in.ResourceTemplates[i].Arguments)
		out.ResourceTemplates[i].Annotations = maps.Clone(in.ResourceTemplates[i].Annotations)
	}

	out.Prompts = slices.Clone(in.Prompts)
	for i := range out.Prompts {
		out.Prompts[i].Arguments = maps.Clone(in.Prompts[i].Arguments)
	}

	return out
}

func clonePtr[T any](in *T) *T {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}
