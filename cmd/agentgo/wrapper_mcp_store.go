package main

import (
	"context"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	mcpConsumerAPI "github.com/flexigpt/flexigpt-app/internal/mcp/store/consumerapi"
	"github.com/flexigpt/flexigpt-app/internal/middleware"
)

type MCPStoreWrapper struct {
	api   *mcpConsumerAPI.API
	roots compositionapi.RootAPI
}

func withMCPStore[T any](
	w *MCPStoreWrapper,
	fn func(*mcpConsumerAPI.API) (T, error),
) (T, error) {
	return middleware.WithRecoveryResp(func() (T, error) {
		var zero T
		if w == nil || w.api == nil {
			return zero, basespec.ErrClosed
		}
		return fn(w.api)
	})
}

func (w *MCPStoreWrapper) ListMCPServers(
	rootID root.RootID,
) ([]artifact.Artifact, error) {
	return withMCPStore(w, func(api *mcpConsumerAPI.API) ([]artifact.Artifact, error) {
		return api.ListServers(context.Background(), rootID)
	})
}

func (w *MCPStoreWrapper) ListMCPPolicies(
	rootID root.RootID,
) ([]artifact.Artifact, error) {
	return withMCPStore(w, func(api *mcpConsumerAPI.API) ([]artifact.Artifact, error) {
		return api.ListPolicies(context.Background(), rootID)
	})
}

func (w *MCPStoreWrapper) ListMCPServersForManagement() (
	[]artifact.Artifact,
	error,
) {
	return middleware.WithRecoveryResp(func() ([]artifact.Artifact, error) {
		if w == nil || w.api == nil || w.roots == nil {
			return nil, basespec.ErrClosed
		}
		roots, err := w.roots.List(context.Background())
		if err != nil {
			return nil, err
		}
		output := make([]artifact.Artifact, 0)
		for _, rootValue := range roots {
			values, err := w.api.ListServers(
				context.Background(),
				rootValue.ID,
			)
			if err != nil {
				return nil, err
			}
			output = append(output, values...)
		}
		sort.Slice(output, func(left, right int) bool {
			if output[left].RootID != output[right].RootID {
				return output[left].RootID < output[right].RootID
			}
			if output[left].LogicalName != output[right].LogicalName {
				return output[left].LogicalName <
					output[right].LogicalName
			}
			return output[left].ID < output[right].ID
		})
		return output, nil
	})
}

func (w *MCPStoreWrapper) GetMCPServerInstallation(
	ref artifact.ArtifactRef,
) (mcpConsumerAPI.ServerInstallationView, error) {
	return withMCPStore(w, func(api *mcpConsumerAPI.API) (mcpConsumerAPI.ServerInstallationView, error) {
		return api.GetServerInstallation(context.Background(), ref)
	})
}

func (w *MCPStoreWrapper) InspectMCPServer(
	ref artifact.ArtifactRef,
) (mcpConsumerAPI.ServerInstallationView, error) {
	return withMCPStore(w, func(api *mcpConsumerAPI.API) (mcpConsumerAPI.ServerInstallationView, error) {
		return api.GetServerInstallation(context.Background(), ref)
	})
}

func (w *MCPStoreWrapper) InspectMCPPolicy(
	ref artifact.ArtifactRef,
) (mcpConsumerAPI.PolicyView, error) {
	return withMCPStore(w, func(api *mcpConsumerAPI.API) (mcpConsumerAPI.PolicyView, error) {
		return api.InspectMCPPolicyForRuntime(context.Background(), ref)
	})
}

func (w *MCPStoreWrapper) close() {
	if w == nil {
		return
	}
	w.api = nil
	w.roots = nil
}
