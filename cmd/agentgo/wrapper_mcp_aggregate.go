package main

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/artifact"
	mcpAggregate "github.com/flexigpt/flexigpt-app/internal/mcp/aggregate"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/server"
	mcpConsumerAPI "github.com/flexigpt/flexigpt-app/internal/mcp/store/consumerapi"
	mcpDomainServer "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain/server"
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
	ref artifact.ArtifactRef,
) (mcpAggregate.MCPServerDetails, error) {
	return withMCPAggregate(
		w,
		func(service *mcpAggregate.Service) (mcpAggregate.MCPServerDetails, error) {
			return service.GetMCPServer(context.Background(), ref)
		},
	)
}

func (w *MCPAggregateWrapper) ListMCPCollectionServers(
	ref artifact.ArtifactRef,
) ([]mcpAggregate.MCPServerDetails, error) {
	return withMCPAggregate(
		w,
		func(service *mcpAggregate.Service) ([]mcpAggregate.MCPServerDetails, error) {
			return service.ListMCPCollectionServers(context.Background(), ref)
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
	ref artifact.ArtifactRef,
	expectedSettingsRevision uint64,
	data mcpDomainServer.ServerData,
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
	ref artifact.ArtifactRef,
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
	ref artifact.ArtifactRef,
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
	request mcpConsumerAPI.ManagedMCPCreateRequest,
) (mcpConsumerAPI.ManagedMCPCreateResult, error) {
	return withMCPAggregate(
		w,
		func(service *mcpAggregate.Service) (mcpConsumerAPI.ManagedMCPCreateResult, error) {
			return service.CreateMCPServer(context.Background(), request)
		},
	)
}

func (w *MCPAggregateWrapper) UpdateMCPServer(
	request mcpConsumerAPI.ManagedMCPReplaceRequest,
) (mcpConsumerAPI.ManagedMCPReplaceResult, error) {
	return withMCPAggregate(
		w,
		func(service *mcpAggregate.Service) (mcpConsumerAPI.ManagedMCPReplaceResult, error) {
			return service.UpdateMCPServer(context.Background(), request)
		},
	)
}

func (w *MCPAggregateWrapper) DeleteMCPServer(
	ref artifact.ArtifactRef,
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
	request mcpConsumerAPI.ManagedMCPPolicyUpsertRequest,
) (mcpConsumerAPI.ManagedMCPPolicyUpsertResult, error) {
	return withMCPAggregate(
		w,
		func(service *mcpAggregate.Service) (mcpConsumerAPI.ManagedMCPPolicyUpsertResult, error) {
			return service.SaveMCPPolicy(context.Background(), request)
		},
	)
}

func (w *MCPAggregateWrapper) DeleteMCPPolicy(
	ref artifact.ArtifactRef,
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
		return model.ErrClosed
	}
	return nil
}

func (w *MCPAggregateWrapper) close() {
	if w == nil {
		return
	}
	w.service = nil
}
