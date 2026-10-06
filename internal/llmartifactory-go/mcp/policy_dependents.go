package mcp

import (
	"context"
	"fmt"
	"sort"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	resourceFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/domain"
)

// ListMCPServersReferencingPolicy finds current Server Artifacts in one Root
// whose direct Policy declaration or additional installation policy references
// can be affected by a policy with policyName.
//
// The result is intentionally conservative for same-name direct policy
// references. That prevents a policy publication from leaving a connected
// server with a stale resolved policy.
func (a *Service) ListMCPServersReferencingPolicy(
	ctx context.Context,
	rootID rootModel.RootID,
	policyName spec.LogicalName,
) ([]artifactModel.ArtifactRef, error) {
	if a == nil || a.resources == nil {
		return nil, spec.ErrClosed
	}
	if err := rootID.Validate(); err != nil {
		return nil, err
	}
	if err := policyName.Validate(); err != nil {
		return nil, err
	}
	return resourceFlow.WithVerificationSession(
		ctx,
		a.resources,
		func(sessionCtx context.Context) ([]artifactModel.ArtifactRef, error) {
			return a.listMCPServersReferencingPolicy(
				sessionCtx,
				rootID,
				policyName,
			)
		},
	)
}

func (a *Service) listMCPServersReferencingPolicy(
	ctx context.Context,
	rootID rootModel.RootID,
	policyName spec.LogicalName,
) ([]artifactModel.ArtifactRef, error) {
	servers, err := a.ListServers(ctx, ListServersRequest{
		RootID: rootID,
	})
	if err != nil {
		return nil, err
	}

	output := make([]artifactModel.ArtifactRef, 0)
	seen := make(map[artifactModel.ArtifactRef]struct{})

	for _, record := range servers {
		if record.State != artifactModel.StateAvailable {
			continue
		}

		material, err := a.resolveServerMaterial(ctx, record.Ref)
		if err != nil {
			return nil, fmt.Errorf(
				"inspect MCP server %q for policy dependency: %w",
				record.Ref.ArtifactID,
				err,
			)
		}

		direct := material.Document.Configuration.Policy
		if direct != nil && direct.Name == policyName {
			if _, found := seen[record.Ref]; !found {
				seen[record.Ref] = struct{}{}
				output = append(output, record.Ref)
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
		if _, found := seen[record.Ref]; !found {
			seen[record.Ref] = struct{}{}
			output = append(output, record.Ref)
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

func (a *Service) additionalPoliciesReferenceName(
	ctx context.Context,
	refs []artifactModel.ArtifactRef,
	rootID rootModel.RootID,
	policyName spec.LogicalName,
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
