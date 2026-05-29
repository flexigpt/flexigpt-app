package auth

import (
	"context"
	"fmt"
)

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
