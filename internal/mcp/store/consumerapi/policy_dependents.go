package consumerapi

import (
	"context"
	"fmt"
	"maps"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/policy"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain"
)

// GetMCPEffectivePolicy returns the safe effective policy projection for one
// Server Artifact. It deliberately returns no materialized connection values,
// secrets, runtime headers, or environment data.
func (a *API) GetMCPEffectivePolicy(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (mcpPolicy.Effective, error) {
	resolved, err := a.resolveMCPServer(ctx, ref)
	if err != nil {
		return mcpPolicy.Effective{}, err
	}

	output := resolved.Policy
	output.Body = mcpPolicy.Clone(resolved.Policy.Body)
	output.Conflicts = maps.Clone(resolved.Policy.Conflicts)
	return output, nil
}

// ListMCPServersReferencingPolicy finds current Server Artifacts in one Root
// whose direct Policy declaration or additional installation policy references
// can be affected by a policy with policyName.
//
// The result is intentionally conservative for same-name direct policy
// references. That prevents a policy publication from leaving a connected
// server with a stale resolved policy.
func (a *API) ListMCPServersReferencingPolicy(
	ctx context.Context,
	rootID root.RootID,
	policyName basespec.LogicalName,
) ([]artifact.ArtifactRef, error) {
	if a == nil {
		return nil, basespec.ErrClosed
	}
	if err := rootID.Validate(); err != nil {
		return nil, err
	}
	if err := policyName.Validate(); err != nil {
		return nil, err
	}

	servers, err := a.ListServers(ctx, rootID)
	if err != nil {
		return nil, err
	}

	output := make([]artifact.ArtifactRef, 0)
	seen := make(map[artifact.ArtifactRef]struct{})

	for _, record := range servers {
		if record.State != artifact.StateAvailable {
			continue
		}

		material, err := a.resolveServerMaterial(ctx, record.Ref())
		if err != nil {
			return nil, fmt.Errorf(
				"inspect MCP server %q for policy dependency: %w",
				record.ID,
				err,
			)
		}

		direct := material.Document.Configuration.Policy
		if direct != nil && direct.Name == policyName {
			if _, found := seen[record.Ref()]; !found {
				seen[record.Ref()] = struct{}{}
				output = append(output, record.Ref())
			}
			continue
		}

		referenced, err := a.additionalPoliciesReferenceName(
			ctx,
			material.Installation.AdditionalPolicies,
			rootID,
			policyName,
		)
		if err != nil {
			return nil, err
		}
		if !referenced {
			continue
		}
		if _, found := seen[record.Ref()]; !found {
			seen[record.Ref()] = struct{}{}
			output = append(output, record.Ref())
		}
	}

	sort.Slice(output, func(left, right int) bool {
		if output[left].RootID != output[right].RootID {
			return output[left].RootID < output[right].RootID
		}
		return output[left].ArtifactID < output[right].ArtifactID
	})
	return output, nil
}

func (a *API) additionalPoliciesReferenceName(
	ctx context.Context,
	refs []artifact.ArtifactRef,
	rootID root.RootID,
	policyName basespec.LogicalName,
) (bool, error) {
	for _, ref := range refs {
		terminal, err := a.resolveDeclarationArtifact(ctx, ref)
		if err != nil {
			return false, err
		}

		record, err := a.artifacts.Get(ctx, terminal)
		if err != nil {
			return false, err
		}
		if record.RootID != rootID ||
			record.Kind != mcpDomain.MCPPolicyArtifactKind ||
			record.LogicalName != policyName {
			continue
		}
		return true, nil
	}
	return false, nil
}
