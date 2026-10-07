package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition"
	managepackageFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managepackage"
	refreshFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/toolruntime"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	pluginAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
	toolAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool"
	toolv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool/contract/v1"
	toolDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool/domain"
	"github.com/flexigpt/flexigpt-app/internal/llmtoolsutil"
)

type ToolStoreWrapper struct {
	api     *toolAPI.Service
	runtime *toolruntime.Adapter
}

func InitToolStoreWrapper(
	wrapper *ToolStoreWrapper,
	runtimeAdapter *toolruntime.Adapter,
	sources source.API,
	discovery refreshFlow.API,
	artifacts artifact.API,
	managedArtifacts managepackageFlow.API,
	protection root.ProtectionAPI,
	cat catalog.API,
	definitions definition.API,
	builtin toolDomain.BuiltinCatalog,
	resolver *composition.Resolver,
) error {
	if wrapper == nil || runtimeAdapter == nil {
		return errors.New(
			"tool store wrapper dependencies are incomplete",
		)
	}

	api, err := toolAPI.New(
		sources,
		discovery,
		artifacts,
		managedArtifacts,
		protection,
		cat,
		definitions,
		builtin,
		resolver,
	)
	if err != nil {
		return err
	}
	wrapper.api = api
	wrapper.runtime = runtimeAdapter
	return nil
}

func withToolStore[T any](
	w *ToolStoreWrapper,
	fn func(*toolAPI.Service) (T, error),
) (T, error) {
	return withRecoveryResp(func() (T, error) {
		var zero T
		if w == nil || w.api == nil {
			return zero, spec.ErrClosed
		}
		return fn(w.api)
	})
}

func (w *ToolStoreWrapper) ListToolPlugins() (
	[]pluginAPI.ListItem,
	error,
) {
	return withToolStore(
		w,
		func(api *toolAPI.Service) ([]pluginAPI.ListItem, error) {
			return api.ListToolPlugins(context.Background())
		},
	)
}

func (w *ToolStoreWrapper) GetToolPlugin(
	ref artifactModel.ArtifactRef,
) (pluginAPI.PluginView, error) {
	return withToolStore(
		w,
		func(api *toolAPI.Service) (pluginAPI.PluginView, error) {
			return api.GetToolPlugin(context.Background(), ref)
		},
	)
}

func (w *ToolStoreWrapper) ListPluginTools(
	ref artifactModel.ArtifactRef,
) ([]toolAPI.ToolListItem, error) {
	return withToolStore(
		w,
		func(api *toolAPI.Service) ([]toolAPI.ToolListItem, error) {
			return api.ListTools(context.Background(), ref)
		},
	)
}

func (w *ToolStoreWrapper) GetTool(
	ref artifactModel.ArtifactRef,
) (toolAPI.ToolView, error) {
	return withToolStore(
		w,
		func(api *toolAPI.Service) (toolAPI.ToolView, error) {
			return api.GetTool(context.Background(), ref)
		},
	)
}

func (w *ToolStoreWrapper) SetToolEnabled(
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (toolAPI.ToolView, error) {
	return withToolStore(
		w,
		func(api *toolAPI.Service) (toolAPI.ToolView, error) {
			return api.SetToolEnabled(
				context.Background(),
				ref,
				expectedRevision,
				enabled,
			)
		},
	)
}

func (w *ToolStoreWrapper) SetToolPluginEnabled(
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
	enabled bool,
) (pluginAPI.PluginView, error) {
	return withToolStore(
		w,
		func(api *toolAPI.Service) (pluginAPI.PluginView, error) {
			return api.SetToolPluginEnabled(
				context.Background(),
				ref,
				expectedRevision,
				enabled,
			)
		},
	)
}

// MapToolTarget returns the canonical source-backed capability target for one
// enabled Tool. It is safe to persist in conversation ToolSelection state.
func (w *ToolStoreWrapper) MapToolTarget(
	ref artifactModel.ArtifactRef,
) (composition.CapabilityTarget, error) {
	return withToolStore(
		w,
		func(api *toolAPI.Service) (
			composition.CapabilityTarget,
			error,
		) {
			return api.MapToolTarget(context.Background(), ref)
		},
	)
}

// ResolveToolTarget returns an enabled Tool together with the enabled Tool
// Plugin that currently exposes it.
func (w *ToolStoreWrapper) ResolveToolTarget(
	target composition.CapabilityTarget,
) (toolAPI.ResolvedToolView, error) {
	return withToolStore(
		w,
		func(api *toolAPI.Service) (
			toolAPI.ResolvedToolView,
			error,
		) {
			return api.ResolveToolTarget(
				context.Background(),
				target,
			)
		},
	)
}

// InvokeGoToolTarget invokes only an enabled Go Tool selected by an
// Artifact-backed capability target.
//
// SDK Tools intentionally fail here. They are provider-native inference
// ToolChoices and must be invoked by inference, never by the local Go Tool
// runtime.
func (w *ToolStoreWrapper) InvokeGoToolTarget(
	target composition.CapabilityTarget,
	args string,
	timeoutMS int,
) (*llmtoolsutil.InvokeResponse, error) {
	return withRecoveryResp(func() (*llmtoolsutil.InvokeResponse, error) {
		if w == nil || w.api == nil || w.runtime == nil {
			return nil, spec.ErrClosed
		}
		if timeoutMS < 0 {
			return nil, fmt.Errorf(
				"%w: Tool timeout cannot be negative",
				spec.ErrInvalid,
			)
		}

		resolved, err := w.api.ResolveToolTarget(
			context.Background(),
			target,
		)
		if err != nil {
			return nil, err
		}
		if resolved.Tool.Implementation.Kind !=
			toolv1.ImplementationKindGo {
			return nil, fmt.Errorf(
				"%w: SDK Tool %q cannot be invoked by the local Go Tool runtime",
				spec.ErrUnsupported,
				resolved.Tool.Name,
			)
		}

		raw := json.RawMessage(`{}`)
		if value := strings.TrimSpace(args); value != "" {
			raw = json.RawMessage(value)
			if !json.Valid(raw) {
				return nil, fmt.Errorf(
					"%w: Tool arguments must be valid JSON",
					spec.ErrInvalid,
				)
			}
		}

		var timeout time.Duration
		if timeoutMS > 0 {
			timeout = time.Duration(timeoutMS) * time.Millisecond
		}

		outputs, invokeErr := w.runtime.CallGoTool(
			context.Background(),
			resolved.Tool.Implementation.Function,
			raw,
			timeout,
		)
		response := &llmtoolsutil.InvokeResponse{
			Outputs: outputs,
			IsError: invokeErr != nil,
		}
		if invokeErr != nil {
			response.ErrorMessage = invokeErr.Error()
		}
		return response, nil
	})
}

func (w *ToolStoreWrapper) close() {
	if w == nil {
		return
	}
	w.api = nil
	w.runtime = nil
}
