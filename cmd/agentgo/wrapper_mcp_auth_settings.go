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
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/mcp/auth"
	"github.com/flexigpt/flexigpt-app/internal/mcp/overlay"
	"github.com/flexigpt/flexigpt-app/internal/mcp/secret"
	"github.com/flexigpt/flexigpt-app/internal/mcp/server"
	settingSpec "github.com/flexigpt/flexigpt-app/internal/setting/spec"
	"golang.org/x/oauth2"
)

const (
	mcpSettingsNamespace        = "mcp-settings-v1"
	mcpSettingsIndexLogicalKey  = "mcp-settings-v1:index"
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

	DeleteAuthKey(
		ctx context.Context,
		req *settingSpec.DeleteAuthKeyRequest,
	) (*settingSpec.DeleteAuthKeyResponse, error)
}

// mcpSettingsAdapter stores non-secret MCP installation metadata through the
// existing Setting Store. The mutex gives the desktop process an atomic
// compare-and-swap boundary around Get/Set APIs that do not expose a native
// revision operation.
type mcpSettingsAdapter struct {
	store mcpAuthKeyStore
	mu    sync.Mutex
}

type mcpSettingsIndex struct {
	Keys map[string]string `json:"keys"`
}

type mcpGlobalSettingsRecord struct {
	Revision uint64               `json:"revision"`
	Settings auth.MCPAuthSettings `json:"settings"`
}

func newMCPSettingsAdapter(
	store mcpAuthKeyStore,
) (*mcpSettingsAdapter, error) {
	if store == nil {
		return nil, errors.New("MCP Setting Store is required")
	}
	return &mcpSettingsAdapter{store: store}, nil
}

func (s *mcpSettingsAdapter) GetMCPInstallationValue(
	ctx context.Context,
	key string,
) (json.RawMessage, bool, error) {
	if err := validateMCPSettingsKey(key); err != nil {
		return nil, false, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	raw, found, err := s.readRawLocked(ctx, key)
	if err != nil || !found {
		return nil, found, err
	}
	return append(json.RawMessage(nil), raw...), true, nil
}

func (s *mcpSettingsAdapter) PutMCPInstallationValue(
	ctx context.Context,
	key string,
	expectedRevision uint64,
	value json.RawMessage,
) error {
	if err := validateMCPSettingsKey(key); err != nil {
		return err
	}
	canonical, err := jsonutil.CanonicalizeObject(
		value,
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return err
	}

	nextRevision, err := revisionOf(canonical)
	if err != nil {
		return err
	}
	if nextRevision != expectedRevision+1 {
		return fmt.Errorf(
			"%w: invalid MCP settings revision transition",
			basespec.ErrInvalid,
		)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	current, found, err := s.readRawLocked(ctx, key)
	if err != nil {
		return err
	}
	currentRevision := uint64(0)
	if found {
		currentRevision, err = revisionOf(current)
		if err != nil {
			return err
		}
	}
	if currentRevision != expectedRevision {
		return basespec.ErrConflict
	}

	index, err := s.readIndexLocked(ctx)
	if err != nil {
		return err
	}
	index.Keys[key] = mcpSettingsStorageKey(key)
	if err := s.writeIndexLocked(ctx, index); err != nil {
		return err
	}

	// Index first. If the value write fails, prefix cleanup can still find the
	// dangling key and a normal retry still observes the previous revision.
	return s.writeRawLocked(ctx, key, canonical)
}

func (s *mcpSettingsAdapter) DeleteMCPInstallationValue(
	ctx context.Context,
	key string,
	expectedRevision uint64,
) error {
	if err := validateMCPSettingsKey(key); err != nil {
		return err
	}
	if expectedRevision == 0 {
		return fmt.Errorf(
			"%w: expected MCP settings revision is required",
			basespec.ErrInvalid,
		)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	current, found, err := s.readRawLocked(ctx, key)
	if err != nil {
		return err
	}
	if !found {
		return basespec.ErrConflict
	}
	currentRevision, err := revisionOf(current)
	if err != nil {
		return err
	}
	if currentRevision != expectedRevision {
		return basespec.ErrConflict
	}

	if err := s.deleteRawLocked(ctx, key); err != nil {
		return err
	}
	index, err := s.readIndexLocked(ctx)
	if err != nil {
		return err
	}
	delete(index.Keys, key)
	return s.writeIndexLocked(ctx, index)
}

// DeleteMCPInstallationPrefix is used only by trusted protected-topology
// reset. It clears overlays and their opaque secret references together.
func (s *mcpSettingsAdapter) DeleteMCPInstallationPrefix(
	ctx context.Context,
	prefix string,
) error {
	if err := validateMCPSettingsKey(prefix); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	index, err := s.readIndexLocked(ctx)
	if err != nil {
		return err
	}

	keys := make([]string, 0)
	for key := range index.Keys {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	var output error
	for _, key := range keys {
		raw, found, err := s.readRawLocked(ctx, key)
		if err != nil {
			output = errors.Join(output, err)
			continue
		}
		if found {
			output = errors.Join(
				output,
				s.deleteOverlaySecretsLocked(ctx, key, raw),
			)
		}
		output = errors.Join(output, s.deleteRawLocked(ctx, key))
		delete(index.Keys, key)
	}
	output = errors.Join(output, s.writeIndexLocked(ctx, index))
	return output
}

func (s *mcpSettingsAdapter) GetMCPGlobalSettings(
	ctx context.Context,
) (auth.MCPAuthSettings, uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	raw, found, err := s.readRawLocked(ctx, mcpGlobalSettingsLogicalKey)
	if err != nil {
		return auth.MCPAuthSettings{}, 0, err
	}
	if !found {
		return auth.MCPAuthSettings{}, 0, nil
	}

	var value mcpGlobalSettingsRecord
	if err := json.Unmarshal(raw, &value); err != nil {
		return auth.MCPAuthSettings{}, 0, err
	}
	normalized, err := normalizeMCPGlobalSettings(value.Settings)
	if err != nil {
		return auth.MCPAuthSettings{}, 0, err
	}
	return normalized, value.Revision, nil
}

func (s *mcpSettingsAdapter) PutMCPGlobalSettings(
	ctx context.Context,
	expectedRevision uint64,
	value auth.MCPAuthSettings,
) (uint64, error) {
	value, err := normalizeMCPGlobalSettings(value)
	if err != nil {
		return 0, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	current, found, err := s.readRawLocked(ctx, mcpGlobalSettingsLogicalKey)
	if err != nil {
		return 0, err
	}

	currentRevision := uint64(0)
	if found {
		currentRevision, err = revisionOf(current)
		if err != nil {
			return 0, err
		}
	}
	if currentRevision != expectedRevision {
		return 0, basespec.ErrConflict
	}

	next := mcpGlobalSettingsRecord{
		Revision: expectedRevision + 1,
		Settings: value,
	}
	raw, err := jsonutil.CanonicalizeObject(
		mustMarshal(next),
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return 0, err
	}
	if err := s.writeRawLocked(ctx, mcpGlobalSettingsLogicalKey, raw); err != nil {
		return 0, err
	}

	index, err := s.readIndexLocked(ctx)
	if err != nil {
		return 0, err
	}
	index.Keys[mcpGlobalSettingsLogicalKey] = mcpSettingsStorageKey(
		mcpGlobalSettingsLogicalKey,
	)
	if err := s.writeIndexLocked(ctx, index); err != nil {
		return 0, err
	}
	return next.Revision, nil
}

func (s *mcpSettingsAdapter) readIndexLocked(
	ctx context.Context,
) (mcpSettingsIndex, error) {
	raw, found, err := s.readRawLocked(ctx, mcpSettingsIndexLogicalKey)
	if err != nil {
		return mcpSettingsIndex{}, err
	}
	if !found {
		return mcpSettingsIndex{
			Keys: map[string]string{},
		}, nil
	}

	var index mcpSettingsIndex
	if err := json.Unmarshal(raw, &index); err != nil {
		return mcpSettingsIndex{}, err
	}
	if index.Keys == nil {
		index.Keys = map[string]string{}
	}
	return index, nil
}

func (s *mcpSettingsAdapter) writeIndexLocked(
	ctx context.Context,
	index mcpSettingsIndex,
) error {
	raw, err := jsonutil.CanonicalizeObject(
		mustMarshal(index),
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return err
	}
	return s.writeRawLocked(ctx, mcpSettingsIndexLogicalKey, raw)
}

func (s *mcpSettingsAdapter) readRawLocked(
	ctx context.Context,
	logicalKey string,
) (json.RawMessage, bool, error) {
	response, err := s.store.GetAuthKey(
		ctx,
		&settingSpec.GetAuthKeyRequest{
			Type: settingSpec.AuthKeyTypeMCP,
			KeyName: settingSpec.AuthKeyName(
				mcpSettingsStorageKey(logicalKey),
			),
		},
	)
	if err != nil {
		if isMissingMCPSetting(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if response == nil || response.Body == nil || !response.Body.NonEmpty {
		return nil, false, nil
	}

	raw, err := jsonutil.CanonicalizeObject(
		[]byte(response.Body.Secret),
		basespec.MaxLocalDataBytes,
	)
	if err != nil {
		return nil, false, err
	}
	return json.RawMessage(raw), true, nil
}

func (s *mcpSettingsAdapter) writeRawLocked(
	ctx context.Context,
	logicalKey string,
	raw json.RawMessage,
) error {
	_, err := s.store.SetAuthKey(
		ctx,
		&settingSpec.SetAuthKeyRequest{
			Type: settingSpec.AuthKeyTypeMCP,
			KeyName: settingSpec.AuthKeyName(
				mcpSettingsStorageKey(logicalKey),
			),
			Body: &settingSpec.SetAuthKeyRequestBody{
				Secret: string(raw),
			},
		},
	)
	return err
}

func (s *mcpSettingsAdapter) deleteRawLocked(
	ctx context.Context,
	logicalKey string,
) error {
	_, err := s.store.DeleteAuthKey(
		ctx,
		&settingSpec.DeleteAuthKeyRequest{
			Type: settingSpec.AuthKeyTypeMCP,
			KeyName: settingSpec.AuthKeyName(
				mcpSettingsStorageKey(logicalKey),
			),
		},
	)
	if isMissingMCPSetting(err) {
		return nil
	}
	return err
}

func (s *mcpSettingsAdapter) deleteOverlaySecretsLocked(
	ctx context.Context,
	logicalKey string,
	raw json.RawMessage,
) error {
	if !strings.Contains(logicalKey, "/servers/") {
		return nil
	}

	var ovr overlay.ServerOverlay
	if err := json.Unmarshal(raw, &ovr); err != nil {
		return err
	}

	refs, err := server.SecretReferences(ovr.ServerData)
	if err != nil {
		return err
	}

	var output error
	for _, ref := range refs {
		output = errors.Join(
			output,
			s.deleteSecretRefLocked(ctx, ref),
		)
	}

	parts := strings.Split(logicalKey, "/")
	if len(parts) != 4 {
		return output
	}
	srv := artifact.ArtifactRef{
		RootID:     basespec.RootID(parts[1]),
		ArtifactID: basespec.ArtifactID(parts[3]),
	}
	if err := srv.Validate(); err != nil {
		return errors.Join(output, err)
	}

	tokenRef, err := secret.NewMCPSecretRefString(
		srv,
		secret.MCPSecretKindOAuthToken,
		"token",
	)
	if err != nil {
		return errors.Join(output, err)
	}
	return errors.Join(output, s.deleteSecretRefLocked(ctx, tokenRef))
}

func (s *mcpSettingsAdapter) deleteSecretRefLocked(
	ctx context.Context,
	ref string,
) error {
	parsed, err := secret.ParseMCPSecretRef(ref)
	if err != nil {
		return err
	}
	_, err = s.store.DeleteAuthKey(
		ctx,
		&settingSpec.DeleteAuthKeyRequest{
			Type: settingSpec.AuthKeyTypeMCP,
			KeyName: settingSpec.AuthKeyName(
				secret.GetMCPSecretRefStorageKey(parsed),
			),
		},
	)
	if isMissingMCPSetting(err) {
		return nil
	}
	return err
}

func revisionOf(raw json.RawMessage) (uint64, error) {
	var value struct {
		Revision uint64 `json:"revision"`
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, err
	}
	return value.Revision, nil
}

func mcpSettingsStorageKey(logicalKey string) string {
	sum := sha256.Sum256([]byte(logicalKey))
	return mcpSettingsNamespace + ":" + hex.EncodeToString(sum[:])
}

func validateMCPSettingsKey(value string) error {
	if strings.TrimSpace(value) == "" ||
		strings.TrimSpace(value) != value ||
		strings.ContainsRune(value, 0) {
		return fmt.Errorf("%w: invalid MCP settings key", basespec.ErrInvalid)
	}
	return nil
}

func normalizeMCPGlobalSettings(
	value auth.MCPAuthSettings,
) (auth.MCPAuthSettings, error) {
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
		return auth.MCPAuthSettings{}, fmt.Errorf(
			"%w: OAuth loopback listen address must be host:port",
			basespec.ErrInvalid,
		)
	}
	if !isLoopbackMCPSettingsHost(host) {
		return auth.MCPAuthSettings{}, fmt.Errorf(
			"%w: OAuth loopback listen host must be loopback",
			basespec.ErrInvalid,
		)
	}
	number, err := strconv.Atoi(port)
	if err != nil || number <= 0 || number > 65535 {
		return auth.MCPAuthSettings{}, fmt.Errorf(
			"%w: OAuth loopback listen port must be 1..65535",
			basespec.ErrInvalid,
		)
	}
	return value, nil
}

func isLoopbackMCPSettingsHost(host string) bool {
	if strings.EqualFold(strings.TrimSpace(host), "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

type settingMCPSecretResolver struct {
	store mcpAuthKeyStore
}

func newSettingMCPSecretResolver(
	store mcpAuthKeyStore,
) *settingMCPSecretResolver {
	return &settingMCPSecretResolver{store: store}
}

func (r *settingMCPSecretResolver) LoadOAuthToken(
	ctx context.Context,
	status auth.MCPAuthStatus,
) (*oauth2.Token, error) {
	ref, err := oauthTokenSecretRef(status.Server)
	if err != nil {
		return nil, err
	}

	raw, err := r.ResolveSecret(ctx, ref)
	if err != nil {
		if isMissingMCPSetting(err) ||
			errors.Is(err, basespec.ErrReferenceUnresolved) {
			return nil, auth.ErrOAuthTokenNotFound
		}
		return nil, err
	}

	var token oauth2.Token
	if err := json.Unmarshal([]byte(raw), &token); err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *settingMCPSecretResolver) SaveOAuthToken(
	ctx context.Context,
	status auth.MCPAuthStatus,
	token *oauth2.Token,
) error {
	if token == nil || !token.Valid() {
		return nil
	}
	ref, err := oauthTokenSecretRef(status.Server)
	if err != nil {
		return err
	}
	//nolint:gosec // Auth token.
	raw, err := json.Marshal(token)
	if err != nil {
		return err
	}
	_, _, err = r.SetMCPSecret(ctx, ref, string(raw))
	return err
}

func (r *settingMCPSecretResolver) DeleteOAuthToken(
	ctx context.Context,
	status auth.MCPAuthStatus,
) error {
	ref, err := oauthTokenSecretRef(status.Server)
	if err != nil {
		return err
	}
	return r.DeleteSecret(ctx, ref)
}

func (r *settingMCPSecretResolver) SetMCPSecret(
	ctx context.Context,
	ref string,
	value string,
) (hash string, nonEmpty bool, err error) {
	if r == nil || r.store == nil {
		return "", false, errors.New("MCP secret writer is not configured")
	}

	parsed, err := secret.ParseMCPSecretRef(ref)
	if err != nil {
		return "", false, err
	}

	keyName := settingSpec.AuthKeyName(
		secret.GetMCPSecretRefStorageKey(parsed),
	)
	_, err = r.store.SetAuthKey(
		ctx,
		&settingSpec.SetAuthKeyRequest{
			Type:    settingSpec.AuthKeyTypeMCP,
			KeyName: keyName,
			Body: &settingSpec.SetAuthKeyRequestBody{
				Secret: value,
			},
		},
	)
	if err != nil {
		return "", false, err
	}

	// SetAuthKey intentionally returns an empty response body. Read the
	// persisted metadata back so this MCP API returns the Setting Store's
	// actual SHA256 and NonEmpty semantics. In particular, the Setting Store
	// treats a whitespace-only value as non-empty, while a local TrimSpace
	// calculation would report a different result.
	response, err := r.store.GetAuthKey(
		ctx,
		&settingSpec.GetAuthKeyRequest{
			Type:    settingSpec.AuthKeyTypeMCP,
			KeyName: keyName,
		},
	)
	if err != nil {
		return "", false, err
	}
	if response == nil || response.Body == nil {
		return "", false, errors.New(
			"MCP secret write returned no persisted metadata",
		)
	}
	return response.Body.SHA256, response.Body.NonEmpty, nil
}

func (r *settingMCPSecretResolver) ResolveSecret(
	ctx context.Context,
	ref string,
) (string, error) {
	if r == nil || r.store == nil {
		return "", errors.New("MCP secret resolver is not configured")
	}

	parsed, err := secret.ParseMCPSecretRef(ref)
	if err != nil {
		return "", err
	}
	response, err := r.store.GetAuthKey(
		ctx,
		&settingSpec.GetAuthKeyRequest{
			Type: settingSpec.AuthKeyTypeMCP,
			KeyName: settingSpec.AuthKeyName(
				secret.GetMCPSecretRefStorageKey(parsed),
			),
		},
	)
	if err != nil {
		return "", err
	}
	if response == nil || response.Body == nil || !response.Body.NonEmpty {
		return "", fmt.Errorf(
			"%w: MCP secret %q is unavailable",
			basespec.ErrReferenceUnresolved,
			ref,
		)
	}
	return response.Body.Secret, nil
}

func (r *settingMCPSecretResolver) DeleteSecret(
	ctx context.Context,
	ref string,
) error {
	if r == nil || r.store == nil {
		return errors.New("MCP secret cleaner is not configured")
	}

	parsed, err := secret.ParseMCPSecretRef(ref)
	if err != nil {
		return err
	}
	_, err = r.store.DeleteAuthKey(
		ctx,
		&settingSpec.DeleteAuthKeyRequest{
			Type: settingSpec.AuthKeyTypeMCP,
			KeyName: settingSpec.AuthKeyName(
				secret.GetMCPSecretRefStorageKey(parsed),
			),
		},
	)
	if isMissingMCPSetting(err) {
		return nil
	}
	return err
}

func isMissingMCPSetting(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "not found") ||
		strings.Contains(message, "does not exist")
}

func oauthTokenSecretRef(
	srv artifact.ArtifactRef,
) (string, error) {
	return secret.NewMCPSecretRefString(
		srv,
		secret.MCPSecretKindOAuthToken,
		"token",
	)
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

func mustMarshal(value any) []byte {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return raw
}
