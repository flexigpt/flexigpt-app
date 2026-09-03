package aggregate

import (
	"context"
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	mcpRuntime "github.com/flexigpt/flexigpt-app/internal/mcp/runtime"
	mcpSpec "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/spec"
	mcpStore "github.com/flexigpt/flexigpt-app/internal/mcp/store"
	mcpArtifact "github.com/flexigpt/flexigpt-app/internal/mcp/store/artifact"
)

type Service struct {
	store      *mcpStore.API
	source     *ServerSource
	runtime    *mcpRuntime.MCPRuntimeManager
	toolBridge *mcpRuntime.ToolBridge
}

func New(
	store *mcpStore.API,
	source *ServerSource,
	runtimeService *mcpRuntime.MCPRuntimeManager,
	toolBridge *mcpRuntime.ToolBridge,
) (*Service, error) {
	if store == nil ||
		source == nil ||
		runtimeService == nil ||
		toolBridge == nil {
		return nil, errors.New("MCP aggregate dependencies are incomplete")
	}
	return &Service{
		store:      store,
		source:     source,
		runtime:    runtimeService,
		toolBridge: toolBridge,
	}, nil
}

func (s *Service) ResolveMCPServer(
	ctx context.Context,
	serverID mcpSpec.ServerID,
) (mcpArtifact.Resolved, error) {
	if s == nil || s.store == nil {
		return mcpArtifact.Resolved{}, mcpSpec.ErrClosed
	}
	ref, err := ArtifactRefForServerID(serverID)
	if err != nil {
		return mcpArtifact.Resolved{}, err
	}
	return s.store.ResolveMCPServer(ctx, ref)
}

func (s *Service) InspectRuntimeConfig(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (mcpSpec.RuntimeConfig, mcpArtifact.Resolved, error) {
	if s == nil || s.source == nil {
		return mcpSpec.RuntimeConfig{},
			mcpArtifact.Resolved{},
			mcpSpec.ErrClosed
	}
	return s.source.InspectRuntimeConfig(ctx, ref)
}

func (s *Service) InvalidateServer(
	ctx context.Context,
	ref artifact.ArtifactRef,
) error {
	serverID, err := ServerIDForArtifact(ref)
	if err != nil {
		return err
	}
	return s.runtime.Invalidate(ctx, serverID)
}

func (s *Service) InvalidateCollection(
	ctx context.Context,
	ref collection.CollectionRef,
) error {
	catalogID, err := CatalogIDForCollection(ref)
	if err != nil {
		return err
	}
	return s.runtime.InvalidateCollection(ctx, catalogID)
}

func (s *Service) ReplaceDocument(
	ctx context.Context,
	request mcpStore.ReplaceDocumentRequest,
) (mcpStore.Bundle, error) {
	if err := s.InvalidateCollection(ctx, request.Bundle); err != nil {
		return mcpStore.Bundle{}, err
	}
	return s.store.ReplaceDocument(ctx, request)
}

func (s *Service) RefreshBundle(
	ctx context.Context,
	ref collection.CollectionRef,
	allowProtected bool,
) (mcpStore.Bundle, error) {
	if err := s.InvalidateCollection(ctx, ref); err != nil {
		return mcpStore.Bundle{}, err
	}
	return s.store.Refresh(ctx, ref, allowProtected)
}

func (s *Service) UpdateBundleEnabled(
	ctx context.Context,
	ref collection.CollectionRef,
	expectedRevision uint64,
	enabled bool,
) (mcpStore.Bundle, error) {
	if err := s.InvalidateCollection(ctx, ref); err != nil {
		return mcpStore.Bundle{}, err
	}
	return s.store.UpdateBundleEnabled(
		ctx,
		ref,
		expectedRevision,
		enabled,
	)
}

func (s *Service) RetireBundle(
	ctx context.Context,
	ref collection.CollectionRef,
	expectedRevision uint64,
) (collection.Collection, error) {
	if err := s.InvalidateCollection(ctx, ref); err != nil {
		return collection.Collection{}, err
	}
	return s.store.Retire(ctx, ref, expectedRevision)
}

func (s *Service) PurgeBundle(
	ctx context.Context,
	ref collection.CollectionRef,
	expectedRevision uint64,
) error {
	if err := s.InvalidateCollection(ctx, ref); err != nil {
		return err
	}
	return s.store.Purge(ctx, ref, expectedRevision)
}

func (s *Service) UpdateProtectedBundleInstallation(
	ctx context.Context,
	ref collection.CollectionRef,
	expectedOverlayRevision uint64,
	runtimeEnabled bool,
) error {
	if err := s.InvalidateCollection(ctx, ref); err != nil {
		return err
	}
	return s.store.UpdateProtectedBundleInstallation(
		ctx,
		ref,
		expectedOverlayRevision,
		runtimeEnabled,
	)
}

func (s *Service) UpdateServerInstallation(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedArtifactRevision uint64,
	data mcpArtifact.ServerData,
) (artifact.Artifact, error) {
	if err := s.InvalidateServer(ctx, ref); err != nil {
		return artifact.Artifact{}, err
	}
	return s.store.UpdateServerInstallation(
		ctx,
		ref,
		expectedArtifactRevision,
		data,
	)
}

func (s *Service) UpdateProtectedServerInstallation(
	ctx context.Context,
	ref artifact.ArtifactRef,
	expectedOverlayRevision uint64,
	runtimeEnabled bool,
	data mcpArtifact.ServerData,
) error {
	if err := s.InvalidateServer(ctx, ref); err != nil {
		return err
	}
	return s.store.UpdateProtectedServerInstallation(
		ctx,
		ref,
		expectedOverlayRevision,
		runtimeEnabled,
		data,
	)
}

func (s *Service) StartConnect(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (*mcpRuntime.MCPServerRuntimeSnapshot, error) {
	serverID, err := ServerIDForArtifact(ref)
	if err != nil {
		return nil, err
	}
	return s.runtime.StartConnect(ctx, serverID)
}

func (s *Service) Disconnect(
	ctx context.Context,
	ref artifact.ArtifactRef,
) error {
	serverID, err := ServerIDForArtifact(ref)
	if err != nil {
		return err
	}
	return s.runtime.Disconnect(ctx, serverID)
}

func (s *Service) RefreshServer(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (*mcpRuntime.MCPServerRuntimeSnapshot, error) {
	serverID, err := ServerIDForArtifact(ref)
	if err != nil {
		return nil, err
	}
	return s.runtime.Refresh(ctx, serverID)
}

func (s *Service) Status(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (*mcpRuntime.MCPServerRuntimeSnapshot, error) {
	serverID, err := ServerIDForArtifact(ref)
	if err != nil {
		return nil, err
	}
	return s.runtime.Status(ctx, serverID)
}

func (s *Service) ListTools(
	ctx context.Context,
	ref artifact.ArtifactRef,
) ([]mcpRuntime.MCPToolCapability, error) {
	serverID, err := ServerIDForArtifact(ref)
	if err != nil {
		return nil, err
	}
	return s.runtime.ListTools(ctx, serverID)
}

func (s *Service) ListResources(
	ctx context.Context,
	ref artifact.ArtifactRef,
) ([]mcpRuntime.MCPResourceRef, error) {
	serverID, err := ServerIDForArtifact(ref)
	if err != nil {
		return nil, err
	}
	return s.runtime.ListResources(ctx, serverID)
}

func (s *Service) ListResourceTemplates(
	ctx context.Context,
	ref artifact.ArtifactRef,
) ([]mcpRuntime.MCPResourceTemplateRef, error) {
	serverID, err := ServerIDForArtifact(ref)
	if err != nil {
		return nil, err
	}
	return s.runtime.ListResourceTemplates(ctx, serverID)
}

func (s *Service) ListPrompts(
	ctx context.Context,
	ref artifact.ArtifactRef,
) ([]mcpRuntime.MCPPromptRef, error) {
	serverID, err := ServerIDForArtifact(ref)
	if err != nil {
		return nil, err
	}
	return s.runtime.ListPrompts(ctx, serverID)
}

func (s *Service) ReadResource(
	ctx context.Context,
	ref artifact.ArtifactRef,
	uri string,
) (*mcpRuntime.MCPReadResourceResponseBody, error) {
	serverID, err := ServerIDForArtifact(ref)
	if err != nil {
		return nil, err
	}
	return s.runtime.ReadResource(ctx, serverID, uri)
}

func (s *Service) GetPrompt(
	ctx context.Context,
	ref artifact.ArtifactRef,
	name string,
	arguments map[string]string,
) (*mcpRuntime.MCPGetPromptResponseBody, error) {
	serverID, err := ServerIDForArtifact(ref)
	if err != nil {
		return nil, err
	}
	return s.runtime.GetPrompt(ctx, serverID, name, arguments)
}

func (s *Service) Complete(
	ctx context.Context,
	ref artifact.ArtifactRef,
	request mcpRuntime.MCPCompleteArgumentRequestBody,
) (*mcpRuntime.MCPCompletionResult, error) {
	serverID, err := ServerIDForArtifact(ref)
	if err != nil {
		return nil, err
	}
	return s.runtime.Complete(ctx, serverID, request)
}

func (s *Service) Evaluate(
	ctx context.Context,
	ref artifact.ArtifactRef,
	request mcpRuntime.InvokeMCPToolRequestBody,
) (*mcpRuntime.MCPApprovalEvaluation, error) {
	serverID, err := ServerIDForArtifact(ref)
	if err != nil {
		return nil, err
	}
	return s.toolBridge.Evaluate(ctx, serverID, request)
}

func (s *Service) Invoke(
	ctx context.Context,
	ref artifact.ArtifactRef,
	request mcpRuntime.InvokeMCPToolRequestBody,
) (*mcpRuntime.InvokeMCPToolResponseBody, error) {
	serverID, err := ServerIDForArtifact(ref)
	if err != nil {
		return nil, err
	}
	return s.toolBridge.Invoke(ctx, serverID, request)
}

func (s *Service) EvaluateMapped(
	ctx context.Context,
	mapping mcpRuntime.MCPProviderToolMapping,
	request mcpRuntime.InvokeMCPToolRequestBody,
) (*mcpRuntime.MCPApprovalEvaluation, error) {
	return s.toolBridge.EvaluateMapped(ctx, mapping, request)
}

func (s *Service) InvokeMapped(
	ctx context.Context,
	mapping mcpRuntime.MCPProviderToolMapping,
	request mcpRuntime.InvokeMCPToolRequestBody,
) (*mcpRuntime.InvokeMCPToolResponseBody, error) {
	return s.toolBridge.InvokeMapped(ctx, mapping, request)
}

func (s *Service) ResolveApproval(
	ctx context.Context,
	approvalID string,
	resolution mcpRuntime.MCPApprovalResolution,
) (mcpRuntime.MCPApprovalResolutionResult, error) {
	if s == nil || s.toolBridge == nil {
		return mcpRuntime.MCPApprovalResolutionResult{},
			fmt.Errorf("%w: MCP aggregate is unavailable", mcpSpec.ErrClosed)
	}
	return s.toolBridge.ResolveApproval(ctx, approvalID, resolution)
}
