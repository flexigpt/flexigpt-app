package main

import (
	"context"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	mcpAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp"
	serverMCPDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/domain/server"
	mcpAggregate "github.com/flexigpt/flexigpt-app/internal/mcp/aggregate"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/server"
)

type MCPAggregateWrapper struct {
	service *mcpAggregate.Service
}

func withMCPAggregate[T any](
	w *MCPAggregateWrapper,
	fn func(*mcpAggregate.Service) (T, error),
) (T, error) {
	return withRecoveryResp(func() (T, error) {
		var zero T
		if err := w.ready(); err != nil {
			return zero, err
		}
		return fn(w.service)
	})
}

func withMCPAggregateError(
	w *MCPAggregateWrapper,
	fn func(*mcpAggregate.Service) error,
) error {
	return withRecovery(func() error {
		if err := w.ready(); err != nil {
			return err
		}
		return fn(w.service)
	})
}

func (w *MCPAggregateWrapper) GetMCPServer(
	ref artifactModel.ArtifactRef,
) (mcpAggregate.MCPServerDetails, error) {
	return withMCPAggregate(
		w,
		func(service *mcpAggregate.Service) (mcpAggregate.MCPServerDetails, error) {
			return service.GetMCPServer(context.Background(), ref)
		},
	)
}

func (w *MCPAggregateWrapper) ListMCPPluginServers(
	ref artifactModel.ArtifactRef,
) ([]mcpAggregate.MCPServerDetails, error) {
	return withMCPAggregate(
		w,
		func(service *mcpAggregate.Service) ([]mcpAggregate.MCPServerDetails, error) {
			// The outer runtime aggregate remains the execution owner.
			return service.ListMCPPluginServers(context.Background(), ref)
		},
	)
}

func (w *MCPAggregateWrapper) GetMCPServersForRuntimeServers(
	servers []mcpServer.ServerID,
) ([]mcpAggregate.MCPServerRuntimeDetails, error) {
	return withMCPAggregate(
		w,
		func(service *mcpAggregate.Service) ([]mcpAggregate.MCPServerRuntimeDetails, error) {
			return service.GetMCPServersForRuntimeServers(context.Background(), servers)
		},
	)
}

func (w *MCPAggregateWrapper) SaveMCPServerSettings(
	ref artifactModel.ArtifactRef,
	expectedSettingsRevision uint64,
	data serverMCPDomain.ServerData,
) (mcpAggregate.MCPServerDetails, error) {
	return withMCPAggregate(
		w,
		func(service *mcpAggregate.Service) (mcpAggregate.MCPServerDetails, error) {
			return service.SaveMCPServerSettings(
				context.Background(),
				ref,
				expectedSettingsRevision,
				data,
			)
		},
	)
}

func (w *MCPAggregateWrapper) SetMCPServerSecret(
	ref artifactModel.ArtifactRef,
	input string,
	value string,
) (mcpAggregate.MCPServerDetails, error) {
	return withMCPAggregate(
		w,
		func(service *mcpAggregate.Service) (mcpAggregate.MCPServerDetails, error) {
			return service.SetMCPServerSecret(
				context.Background(),
				ref,
				input,
				value,
			)
		},
	)
}

func (w *MCPAggregateWrapper) ClearMCPServerSecret(
	ref artifactModel.ArtifactRef,
	input string,
) (mcpAggregate.MCPServerDetails, error) {
	return withMCPAggregate(
		w,
		func(service *mcpAggregate.Service) (mcpAggregate.MCPServerDetails, error) {
			return service.ClearMCPServerSecret(
				context.Background(),
				ref,
				input,
			)
		},
	)
}

func (w *MCPAggregateWrapper) CreateMCPServer(
	request mcpAPI.ManagedMCPCreateRequest,
) (mcpAPI.ManagedMCPCreateResult, error) {
	return withMCPAggregate(
		w,
		func(service *mcpAggregate.Service) (mcpAPI.ManagedMCPCreateResult, error) {
			return service.CreateMCPServer(context.Background(), request)
		},
	)
}

func (w *MCPAggregateWrapper) UpdateMCPServer(
	request mcpAPI.ManagedMCPReplaceRequest,
) (mcpAPI.ManagedMCPReplaceResult, error) {
	return withMCPAggregate(
		w,
		func(service *mcpAggregate.Service) (mcpAPI.ManagedMCPReplaceResult, error) {
			return service.UpdateMCPServer(context.Background(), request)
		},
	)
}

func (w *MCPAggregateWrapper) DeleteMCPServer(
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
) error {
	return withMCPAggregateError(w, func(service *mcpAggregate.Service) error {
		return service.DeleteMCPServer(
			context.Background(),
			ref,
			expectedRevision,
		)
	})
}

func (w *MCPAggregateWrapper) SaveMCPPolicy(
	request mcpAPI.ManagedMCPPolicyUpsertRequest,
) (mcpAPI.ManagedMCPPolicyUpsertResult, error) {
	return withMCPAggregate(
		w,
		func(service *mcpAggregate.Service) (mcpAPI.ManagedMCPPolicyUpsertResult, error) {
			return service.SaveMCPPolicy(context.Background(), request)
		},
	)
}

func (w *MCPAggregateWrapper) DeleteMCPPolicy(
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
) error {
	return withMCPAggregateError(w, func(service *mcpAggregate.Service) error {
		return service.DeleteMCPPolicy(
			context.Background(),
			ref,
			expectedRevision,
		)
	})
}

func (w *MCPAggregateWrapper) ready() error {
	if w == nil || w.service == nil {
		return spec.ErrClosed
	}
	return nil
}

func (w *MCPAggregateWrapper) close() {
	if w == nil {
		return
	}
	w.service = nil
}
