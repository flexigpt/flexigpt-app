package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	mcpAuth "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/auth"
	settingSpec "github.com/flexigpt/flexigpt-app/internal/setting/spec"
)

const (
	mcpSettingsNamespace        = "mcp-settings-v1"
	mcpGlobalSettingsLogicalKey = "mcp-settings-v1:global"
)

type mcpAuthKeyStore interface {
	GetAuthKey(
		ctx context.Context,
		req *settingSpec.GetAuthKeyRequest,
	) (*settingSpec.GetAuthKeyResponse, error)

	SetAuthKey(
		ctx context.Context,
		req *settingSpec.SetAuthKeyRequest,
	) (*settingSpec.SetAuthKeyResponse, error)
}

type mcpSettingsAdapter struct {
	store mcpAuthKeyStore
	mu    sync.Mutex
}

type mcpGlobalSettingsRecord struct {
	Revision uint64                  `json:"revision"`
	Settings mcpAuth.MCPAuthSettings `json:"settings"`
}

func newMCPSettingsAdapter(
	store mcpAuthKeyStore,
) (*mcpSettingsAdapter, error) {
	if store == nil {
		return nil, errors.New("MCP Setting Store is required")
	}
	return &mcpSettingsAdapter{store: store}, nil
}

func (s *mcpSettingsAdapter) GetMCPGlobalSettings(
	ctx context.Context,
) (mcpAuth.MCPAuthSettings, uint64, error) {
	if s == nil || s.store == nil {
		return mcpAuth.MCPAuthSettings{}, 0, basespec.ErrClosed
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	raw, found, err := s.readGlobalLocked(ctx)
	if err != nil {
		return mcpAuth.MCPAuthSettings{}, 0, err
	}
	if !found {
		return mcpAuth.MCPAuthSettings{}, 0, nil
	}

	var value mcpGlobalSettingsRecord
	if err := json.Unmarshal(raw, &value); err != nil {
		return mcpAuth.MCPAuthSettings{}, 0, err
	}

	normalized, err := normalizeMCPGlobalSettings(value.Settings)
	if err != nil {
		return mcpAuth.MCPAuthSettings{}, 0, err
	}
	return normalized, value.Revision, nil
}

func (s *mcpSettingsAdapter) PutMCPGlobalSettings(
	ctx context.Context,
	expectedRevision uint64,
	value mcpAuth.MCPAuthSettings,
) (uint64, error) {
	if s == nil || s.store == nil {
		return 0, basespec.ErrClosed
	}

	normalized, err := normalizeMCPGlobalSettings(value)
	if err != nil {
		return 0, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	currentRaw, found, err := s.readGlobalLocked(ctx)
	if err != nil {
		return 0, err
	}

	currentRevision := uint64(0)
	if found {
		currentRevision, err = revisionOf(currentRaw)
		if err != nil {
			return 0, err
		}
	}
	if currentRevision != expectedRevision {
		return 0, basespec.ErrConflict
	}

	next := mcpGlobalSettingsRecord{
		Revision: expectedRevision + 1,
		Settings: normalized,
	}
	raw, err := jsonutil.MarshalCanonicalObject(
		next,
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return 0, err
	}

	_, err = s.store.SetAuthKey(
		ctx,
		&settingSpec.SetAuthKeyRequest{
			Type:    settingSpec.AuthKeyTypeMCP,
			KeyName: settingSpec.AuthKeyName(mcpSettingsStorageKey(mcpGlobalSettingsLogicalKey)),
			Body: &settingSpec.SetAuthKeyRequestBody{
				Secret: string(raw),
			},
		},
	)
	if err != nil {
		return 0, err
	}
	return next.Revision, nil
}

func (s *mcpSettingsAdapter) readGlobalLocked(
	ctx context.Context,
) (raw json.RawMessage, found bool, err error) {
	response, err := s.store.GetAuthKey(
		ctx,
		&settingSpec.GetAuthKeyRequest{
			Type:    settingSpec.AuthKeyTypeMCP,
			KeyName: settingSpec.AuthKeyName(mcpSettingsStorageKey(mcpGlobalSettingsLogicalKey)),
		},
	)
	if errors.Is(err, settingSpec.ErrAuthKeyNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if response == nil ||
		response.Body == nil ||
		!response.Body.NonEmpty {
		return nil, false, nil
	}

	raw, err = jsonutil.CanonicalizeObject(
		[]byte(response.Body.Secret),
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return nil, false, err
	}
	return raw, true, nil
}

func revisionOf(
	raw json.RawMessage,
) (uint64, error) {
	var value mcpGlobalSettingsRecord
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, err
	}
	return value.Revision, nil
}

func mcpSettingsStorageKey(
	logicalKey string,
) string {
	sum := sha256.Sum256(
		[]byte(mcpSettingsNamespace + ":" + logicalKey),
	)
	return mcpSettingsNamespace + ":" + hex.EncodeToString(sum[:])
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
			basespec.ErrInvalid,
		)
	}
	if !isLoopbackMCPSettingsHost(host) {
		return mcpAuth.MCPAuthSettings{}, fmt.Errorf(
			"%w: OAuth loopback listen host must be loopback",
			basespec.ErrInvalid,
		)
	}
	number, err := strconv.Atoi(port)
	if err != nil || number <= 0 || number > 65535 {
		return mcpAuth.MCPAuthSettings{}, fmt.Errorf(
			"%w: OAuth loopback listen port must be 1..65535",
			basespec.ErrInvalid,
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
