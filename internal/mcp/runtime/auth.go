package runtime

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

const maxResolvedHTTPHeaderValueLen = 4096

type SecretResolver interface {
	ResolveSecret(ctx context.Context, ref string) (string, error)
}

type StaticSecretResolver map[string]string

func (r StaticSecretResolver) ResolveSecret(ctx context.Context, ref string) (string, error) {
	v, ok := r[ref]
	if !ok {
		return "", fmt.Errorf("secret ref not found: %s", ref)
	}
	return v, nil
}

type ResolvedTransportAuth struct {
	Headers         map[string]string
	Env             map[string]string
	SensitiveValues []string
	Status          spec.MCPAuthStatus
}

type AuthManager struct {
	secrets SecretResolver
}

func NewAuthManager(secrets SecretResolver) *AuthManager {
	if secrets == nil {
		secrets = StaticSecretResolver{}
	}
	return &AuthManager{secrets: secrets}
}

func (m *AuthManager) PrepareTransportAuth(
	ctx context.Context,
	cfg spec.MCPServerConfig,
) (ResolvedTransportAuth, error) {
	out := ResolvedTransportAuth{
		Headers: map[string]string{},
		Env:     map[string]string{},
		Status: spec.MCPAuthStatus{
			ServerID: cfg.ID,
			AuthMode: spec.MCPHTTPAuthNone,
			State:    spec.MCPAuthStateNotRequired,
		},
	}

	if cfg.Transport == spec.MCPTransportStdio && cfg.Stdio != nil {
		out.Env = cloneStringMapNonNil(cfg.Stdio.Env)
		for key, ref := range cfg.Stdio.SecretEnvRefs {
			v, err := m.secrets.ResolveSecret(ctx, ref)
			if err != nil {
				return out, err
			}
			out.Env[key] = v
			out.SensitiveValues = append(out.SensitiveValues, v)
		}
		return out, nil
	}

	if cfg.Transport != spec.MCPTransportStreamableHTTP || cfg.StreamableHTTP == nil {
		return out, nil
	}

	httpCfg := cfg.StreamableHTTP
	out.Headers = cloneStringMapNonNil(httpCfg.CustomHeaders)
	for key, ref := range httpCfg.SecretHeaderRefs {
		v, err := m.secrets.ResolveSecret(ctx, ref)
		if err != nil {
			return out, err
		}
		out.Headers[key] = v
		out.SensitiveValues = append(out.SensitiveValues, v)
	}

	mode := httpCfg.AuthMode
	if cfg.AuthRef != nil {
		authMode := spec.MCPHTTPAuthMode(strings.TrimSpace(string(cfg.AuthRef.AuthMode)))
		if authMode != "" && authMode != mode {
			out.Status.State = spec.MCPAuthStateError
			out.Status.LastError = fmt.Sprintf(
				"authRef.authMode %q does not match streamableHttp.authMode %q",
				authMode,
				mode,
			)
			return out, fmt.Errorf("%w: %s", spec.ErrMCPInvalidRequest, out.Status.LastError)
		}
	}

	out.Status.AuthMode = mode

	switch mode {
	case "", spec.MCPHTTPAuthNone:
		out.Status.State = spec.MCPAuthStateNotRequired

	case spec.MCPHTTPAuthCustomBearer:
		if cfg.AuthRef == nil || strings.TrimSpace(cfg.AuthRef.TokenRef) == "" {
			out.Status.State = spec.MCPAuthStateRequired
			return out, spec.ErrMCPAuthRequired
		}
		token, err := m.secrets.ResolveSecret(ctx, cfg.AuthRef.TokenRef)
		if err != nil {
			out.Status.State = spec.MCPAuthStateError
			out.Status.LastError = err.Error()
			return out, err
		}
		out.Headers["Authorization"] = "Bearer " + token
		out.SensitiveValues = append(out.SensitiveValues, token)
		out.Status.State = spec.MCPAuthStateAuthorized

	case spec.MCPHTTPAuthCustomHeaders:
		out.Status.State = spec.MCPAuthStateAuthorized

	case spec.MCPHTTPAuthOAuth, spec.MCPHTTPAuthClientCredentials:
		out.Status.State = spec.MCPAuthStateRequired
		out.Status.LastError = "OAuth token acquisition is scaffolded but not wired to frontend callback yet"
		return out, fmt.Errorf("%w: %s", spec.ErrMCPAuthRequired, mode)

	default:
		out.Status.State = spec.MCPAuthStateError
		out.Status.LastError = "unsupported auth mode"
		return out, fmt.Errorf("%w: unsupported auth mode %s", spec.ErrMCPInvalidRequest, mode)
	}
	if err := validateResolvedHTTPHeaders(out.Headers); err != nil {
		out.Status.State = spec.MCPAuthStateError
		out.Status.LastError = err.Error()
		return out, err
	}
	return out, nil
}

func Expired(expiresAt *time.Time) bool {
	if expiresAt == nil || expiresAt.IsZero() {
		return false
	}
	return time.Now().UTC().After(expiresAt.Add(-30 * time.Second))
}

func validateResolvedHTTPHeaders(headers map[string]string) error {
	for key, value := range headers {
		if strings.TrimSpace(key) == "" {
			return errors.New("resolved HTTP header name cannot be empty")
		}
		if err := validateResolvedHTTPHeaderName(key); err != nil {
			return err
		}

		if strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("resolved HTTP header %q contains newline characters", key)
		}
		if len(value) > maxResolvedHTTPHeaderValueLen {
			return fmt.Errorf(
				"resolved HTTP header %q exceeds maximum length of %d",
				key,
				maxResolvedHTTPHeaderValueLen,
			)
		}
	}
	return nil
}

func validateResolvedHTTPHeaderName(name string) error {
	for _, c := range name {
		if c <= 0x20 || c > 0x7E || c == ':' {
			return fmt.Errorf("invalid resolved HTTP header name %q", name)
		}
	}
	return nil
}

func cloneStringMapNonNil(in map[string]string) map[string]string {
	out := maps.Clone(in)
	if out == nil {
		return map[string]string{}
	}
	return out
}
