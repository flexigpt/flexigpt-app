package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	mcpAuth "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/auth"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay"
	overlayModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/overlay/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	mcpOverlay "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/overlay"
)

const mcpGlobalSettingsSchemaVersion = "v1"

type mcpGlobalSettingsPayload struct {
	OAuthLoopbackListenAddr string `json:"oauthLoopbackListenAddr,omitempty"`
}

type mcpSettingsAdapter struct {
	overlays overlay.StoreAPI
}

func newMCPSettingsAdapter(
	overlays overlay.StoreAPI,
) (*mcpSettingsAdapter, error) {
	if overlays == nil {
		return nil, fmt.Errorf(
			"%w: MCP global settings overlay store is required",
			spec.ErrInvalid,
		)
	}
	return &mcpSettingsAdapter{
		overlays: overlays,
	}, nil
}

func (s *mcpSettingsAdapter) getMCPSettings(
	ctx context.Context,
) (mcpAuth.MCPAuthSettings, uint64, error) {
	if s == nil || s.overlays == nil {
		return mcpAuth.MCPAuthSettings{}, 0, spec.ErrClosed
	}

	record, found, err := s.overlays.GetStoreOverlay(
		ctx,
		mcpOverlay.GlobalSettingsNamespace,
	)
	if err != nil {
		return mcpAuth.MCPAuthSettings{}, 0, err
	}
	if !found {
		return mcpAuth.MCPAuthSettings{}, 0, nil
	}

	var payload mcpGlobalSettingsPayload
	if err := jsonutil.DecodeCanonicalObjectExactInto(
		record.Payload,
		&payload,
		spec.MaxLocalDataBytes,
	); err != nil {
		return mcpAuth.MCPAuthSettings{}, 0, fmt.Errorf(
			"%w: decode MCP global settings overlay: %w",
			spec.ErrInvalid,
			err,
		)
	}

	settings, err := normalizeMCPGlobalSettings(
		mcpAuth.MCPAuthSettings{
			OAuthLoopbackListenAddr: payload.OAuthLoopbackListenAddr,
		},
	)
	if err != nil {
		return mcpAuth.MCPAuthSettings{}, 0, err
	}
	return settings, record.Revision, nil
}

func (s *mcpSettingsAdapter) putMCPSettings(
	ctx context.Context,
	expectedRevision uint64,
	value mcpAuth.MCPAuthSettings,
) (uint64, error) {
	if s == nil || s.overlays == nil {
		return 0, spec.ErrClosed
	}

	settings, err := normalizeMCPGlobalSettings(value)
	if err != nil {
		return 0, err
	}

	payload, err := jsonutil.MarshalCanonicalObject(
		mcpGlobalSettingsPayload{
			OAuthLoopbackListenAddr: settings.OAuthLoopbackListenAddr,
		},
		spec.MaxLocalDataBytes,
	)
	if err != nil {
		return 0, err
	}

	record, err := s.overlays.PutStoreOverlay(
		ctx,
		overlayModel.StorePutRequest{
			Namespace:        mcpOverlay.GlobalSettingsNamespace,
			SchemaVersion:    mcpGlobalSettingsSchemaVersion,
			Payload:          payload,
			ExpectedRevision: expectedRevision,
		},
	)
	if err != nil {
		return 0, err
	}
	return record.Revision, nil
}

func normalizeMCPGlobalSettings(
	value mcpAuth.MCPAuthSettings,
) (mcpAuth.MCPAuthSettings, error) {
	value.OAuthLoopbackListenAddr = strings.TrimSpace(
		value.OAuthLoopbackListenAddr,
	)
	if value.OAuthLoopbackListenAddr == "" {
		return value, nil
	}

	host, port, err := net.SplitHostPort(
		value.OAuthLoopbackListenAddr,
	)
	if err != nil {
		return mcpAuth.MCPAuthSettings{}, fmt.Errorf(
			"%w: OAuth loopback listen address must be host:port",
			spec.ErrInvalid,
		)
	}
	if !isLoopbackMCPSettingsHost(host) {
		return mcpAuth.MCPAuthSettings{}, fmt.Errorf(
			"%w: OAuth loopback listen host must be loopback",
			spec.ErrInvalid,
		)
	}
	number, err := strconv.Atoi(port)
	if err != nil || number <= 0 || number > 65535 {
		return mcpAuth.MCPAuthSettings{}, fmt.Errorf(
			"%w: OAuth loopback listen port must be 1..65535",
			spec.ErrInvalid,
		)
	}
	return value, nil
}

func isLoopbackMCPSettingsHost(
	host string,
) bool {
	if strings.EqualFold(strings.TrimSpace(host), "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

type mcpEnvironmentResolver struct{}

func (mcpEnvironmentResolver) ResolveEnvironment(
	ctx context.Context,
	name string,
) (value string, found bool, err error) {
	if err := ctx.Err(); err != nil {
		return "", false, err
	}
	value, found = os.LookupEnv(name)
	return value, found, nil
}
