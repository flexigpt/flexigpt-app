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
	out.Tools = slices.Clone(in.Tools)
	out.Resources = slices.Clone(in.Resources)
	out.ResourceTemplates = slices.Clone(in.ResourceTemplates)
	out.Prompts = slices.Clone(in.Prompts)
	return out
}

func cloneTimePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	v := *t
	return &v
}
