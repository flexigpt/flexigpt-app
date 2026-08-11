package installation

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/mcp/secret"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

// SecretCleaner removes an opaque installation-local secret reference.
//
// Implementations must be idempotent: deleting an already removed secret must
// return nil. This permits retry after a successful document publication but a
// failed local cleanup step.
type SecretCleaner interface {
	DeleteSecret(ctx context.Context, ref string) error
}

// CleanupRemovedServerSecrets removes every local secret binding and OAuth
// token after source reconciliation has confirmed that the server Artifact is
// missing and before its local metadata is purged.
func CleanupRemovedServerSecrets(
	ctx context.Context,
	server artifact.ArtifactRef,
	data ServerData,
	cleaner SecretCleaner,
) error {
	return CleanupReplacedServerSecrets(
		ctx,
		server,
		data,
		DefaultServerData(),
		cleaner,
	)
}

// CleanupReplacedServerSecrets removes secret bindings no longer referenced by
// a server installation update. OAuth token state is removed whenever the
// installation data changes because profile, credential, and endpoint changes
// invalidate prior authorization state.
func CleanupReplacedServerSecrets(
	ctx context.Context,
	server artifact.ArtifactRef,
	before ServerData,
	after ServerData,
	cleaner SecretCleaner,
) error {
	if err := server.Validate(); err != nil {
		return err
	}
	if cleaner == nil {
		return fmt.Errorf(
			"%w: MCP secret cleaner is unavailable",
			basespec.ErrInvalid,
		)
	}

	beforeRefs, err := SecretReferences(before)
	if err != nil {
		return err
	}
	afterRefs, err := SecretReferences(after)
	if err != nil {
		return err
	}

	retained := make(map[string]struct{}, len(afterRefs))
	for _, ref := range afterRefs {
		retained[ref] = struct{}{}
	}

	var output error
	for _, ref := range beforeRefs {
		if _, keep := retained[ref]; keep {
			continue
		}
		output = errors.Join(output, cleaner.DeleteSecret(ctx, ref))
	}

	tokenRef, err := secret.NewMCPSecretRefString(
		server,
		spec.MCPSecretKindOAuthToken,
		"token",
	)
	if err != nil {
		return errors.Join(output, err)
	}
	output = errors.Join(output, cleaner.DeleteSecret(ctx, tokenRef))
	return output
}

// SecretReferences returns the unique opaque secret references held by local
// server installation data. It never resolves or returns secret values.
func SecretReferences(data ServerData) ([]string, error) {
	if err := ValidateServerData(data); err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	for _, binding := range data.Inputs {
		if binding.SecretRef == "" {
			continue
		}
		seen[binding.SecretRef] = struct{}{}
	}

	output := make([]string, 0, len(seen))
	for value := range seen {
		output = append(output, value)
	}
	sort.Strings(output)
	return output, nil
}
