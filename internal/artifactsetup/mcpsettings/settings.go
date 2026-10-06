package mcpsettings

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

const schemaVersion = "v1"

type payload struct {
	OAuthLoopbackListenAddr string `json:"oauthLoopbackListenAddr,omitempty"`
}

// Settings persists application-level MCP runtime settings through the
// generic Store-scoped overlay capability.
type Settings struct {
	overlays overlay.StoreAPI
}

func New(
	overlays overlay.StoreAPI,
) (*Settings, error) {
	if overlays == nil {
		return nil, fmt.Errorf(
			"%w: MCP global settings overlay store is required",
			spec.ErrInvalid,
		)
	}
	return &Settings{overlays: overlays}, nil
}

func (s *Settings) Get(
	ctx context.Context,
) (mcpAuth.MCPAuthSettings, uint64, error) {
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

	var value payload
	if err := jsonutil.DecodeCanonicalObjectExactInto(
		record.Payload,
		&value,
		spec.MaxLocalDataBytes,
	); err != nil {
		return mcpAuth.MCPAuthSettings{}, 0, fmt.Errorf(
			"%w: decode MCP global settings overlay: %w",
			spec.ErrInvalid,
			err,
		)
	}

	settings, err := normalize(
		mcpAuth.MCPAuthSettings{
			OAuthLoopbackListenAddr: value.OAuthLoopbackListenAddr,
		},
	)
	if err != nil {
		return mcpAuth.MCPAuthSettings{}, 0, err
	}
	return settings, record.Revision, nil
}

func (s *Settings) Put(
	ctx context.Context,
	expectedRevision uint64,
	value mcpAuth.MCPAuthSettings,
) (uint64, error) {
	settings, err := normalize(value)
	if err != nil {
		return 0, err
	}

	raw, err := jsonutil.MarshalCanonicalObject(
		payload{
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
			SchemaVersion:    schemaVersion,
			Payload:          raw,
			ExpectedRevision: expectedRevision,
		},
	)
	if err != nil {
		return 0, err
	}
	return record.Revision, nil
}

func normalize(
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
	if !isLoopbackHost(host) {
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

func isLoopbackHost(
	host string,
) bool {
	if strings.EqualFold(strings.TrimSpace(host), "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// EnvironmentResolver is the application environment capability supplied to
// trusted MCP runtime materialization. It does not own request cancellation.
type EnvironmentResolver struct{}

func (EnvironmentResolver) ResolveEnvironment(
	_ context.Context,
	name string,
) (value string, found bool, err error) {
	value, found = os.LookupEnv(name)
	return value, found, nil
}
