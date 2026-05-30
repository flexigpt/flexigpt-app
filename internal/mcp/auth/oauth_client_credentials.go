package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
)

func ValidateOAuthClientCredentialsSecret(raw string, requireClientSecret bool) error {
	_, _, err := parseOAuthClientCredentialsSecret(raw, requireClientSecret)
	return err
}

func resolveOAuthClientCredentials(
	ctx context.Context,
	secrets SecretResolver,
	ref string,
	requireClientSecret bool,
) (*oauthex.ClientCredentials, []string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, nil, fmt.Errorf("%w: clientCredentialRef is required", spec.ErrMCPInvalidRequest)
	}
	if secrets == nil {
		return nil, nil, fmt.Errorf("%w: secret resolver is not configured", spec.ErrMCPAuthRequired)
	}

	raw, err := secrets.ResolveSecret(ctx, ref)
	if err != nil {
		return nil, nil, err
	}

	creds, sensitive, err := parseOAuthClientCredentialsSecret(raw, requireClientSecret)
	if err != nil {
		return nil, nil, err
	}

	// Redact both the full JSON payload and the secret itself.
	sensitive = append([]string{raw}, sensitive...)
	return creds, sensitive, nil
}
