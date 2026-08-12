package server

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

// CleanupUnboundServerSecrets removes every deterministic secret slot
// declared by the current canonical server document but not retained by the
// current installation data.
//
// Unlike before/after-only cleanup, this operation remains retryable after the
// installation metadata write has committed. A retry can derive all current
// secret slots from the immutable server Definition and does not need the
// previous Artifact.Data or overlay value.
func CleanupUnboundServerSecrets(
	ctx context.Context,
	server artifact.ArtifactRef,
	document ServerDocument,
	data ServerData,
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
	if err := ValidateServerDataForDocument(
		server,
		document,
		data,
	); err != nil {
		return err
	}

	retainedValues, err := SecretReferences(data)
	if err != nil {
		return err
	}
	retained := make(map[string]struct{}, len(retainedValues))
	for _, value := range retainedValues {
		retained[value] = struct{}{}
	}

	targets, err := SecretInputTargets(document)
	if err != nil {
		return err
	}
	candidates := make(map[string]struct{})
	for _, target := range targets {
		var kind spec.MCPSecretKind
		switch target.Kind {
		case SecretInputTargetStdioEnv:
			kind = spec.MCPSecretKindStdioEnv
		case SecretInputTargetHTTPHeader:
			kind = spec.MCPSecretKindHTTPHeader
		default:
			return fmt.Errorf(
				"%w: unsupported MCP secret target %q",
				basespec.ErrInvalid,
				target.Kind,
			)
		}
		ref, err := secret.NewMCPSecretRefString(
			server,
			kind,
			target.Slot,
		)
		if err != nil {
			return err
		}
		candidates[ref] = struct{}{}
	}
	for _, declaration := range document.Extension.Install.Inputs {
		if declaration.Kind != InputOAuthClientCredentials {
			continue
		}
		ref, err := secret.NewMCPSecretRefString(
			server,
			spec.MCPSecretKindOAuthClientCredentials,
			"clientCredentials",
		)
		if err != nil {
			return err
		}
		candidates[ref] = struct{}{}
	}

	ordered := make([]string, 0, len(candidates))
	for value := range candidates {
		ordered = append(ordered, value)
	}
	sort.Strings(ordered)

	var output error
	for _, value := range ordered {
		if _, keep := retained[value]; keep {
			continue
		}
		output = errors.Join(output, cleaner.DeleteSecret(ctx, value))
	}

	tokenRef, err := secret.NewMCPSecretRefString(
		server,
		spec.MCPSecretKindOAuthToken,
		"token",
	)
	if err != nil {
		return errors.Join(output, err)
	}
	return errors.Join(output, cleaner.DeleteSecret(ctx, tokenRef))
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
