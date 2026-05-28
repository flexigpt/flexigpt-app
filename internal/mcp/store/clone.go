package store

import (
	"maps"
	"slices"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

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
		cp.CustomHeaders = maps.Clone(in.StreamableHTTP.CustomHeaders)
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

	if in.AuthRef != nil {
		cp := *in.AuthRef
		out.AuthRef = &cp
	}

	out.SoftDeletedAt = cloneTimePtr(in.SoftDeletedAt)
	return out
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

func cloneTimePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	v := *t
	return &v
}
