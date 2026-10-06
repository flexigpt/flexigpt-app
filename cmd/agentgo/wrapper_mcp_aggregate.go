package main

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	mcpAuth "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/auth"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/server"
	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin/mcpcatalog/inferenceadapter"
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	mcpAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp"
	serverMCPDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/domain/server"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/secret"
	"github.com/flexigpt/flexigpt-app/internal/mcppolicy"
)

type mcpAggregateRuntime interface {
	Invalidate(ctx context.Context, server mcpServer.ServerID) error

	Status(
		ctx context.Context,
		server mcpServer.ServerID,
	) (*mcpServer.MCPServerRuntimeSnapshot, error)
}

type mcpAggregateAuth interface {
	ClearAuthStatus(server mcpServer.ServerID)

	BuildAuthHealth(
		ctx context.Context,
		config mcpServer.RuntimeConfig,
	) mcpAuth.MCPAuthHealth

	GetAuthStatus(
		server mcpServer.ServerID,
	) (mcpAuth.MCPAuthStatus, bool)

	PendingAuthorizations() []mcpAuth.MCPOAuthAuthorization
}

type mcpRuntimeConfigInspector interface {
	InspectRuntimeConfig(
		ctx context.Context,
		resolved serverMCPDomain.Resolved,
	) (mcpServer.RuntimeConfig, error)
}

type mcpSecretWriter interface {
	SetMCPSecret(
		ctx context.Context,
		ref string,
		value string,
	) (hash string, nonEmpty bool, err error)
}

// MCPAggregateWrapper owns management API composition and runtime side effects.
// It does not own connection execution, Artifact persistence, or completion
// preparation. Dependencies are wired before serving requests.
type MCPAggregateWrapper struct {
	mu sync.Mutex

	store   mcpAPI.ManagementStore
	source  mcpRuntimeConfigInspector
	runtime mcpAggregateRuntime
	auth    mcpAggregateAuth
	secrets mcpSecretWriter
}

type MCPServerDetails struct {
	Settings      mcpAPI.ServerInstallationView      `json:"settings"`
	Policy        mcppolicy.Effective                `json:"policy"`
	Authorization mcpAuth.MCPAuthHealth              `json:"authorization"`
	Connection    mcpServer.MCPServerRuntimeSnapshot `json:"connection"`
}

// MCPServerRuntimeDetails contains only process-local runtime/auth observations.
// Configuration health and effective policy belong to the verified detail read.
type MCPServerRuntimeDetails struct {
	Ref                  artifactModel.ArtifactRef          `json:"ref"`
	Connection           mcpServer.MCPServerRuntimeSnapshot `json:"connection"`
	Authorization        *mcpAuth.MCPAuthStatus             `json:"authorization,omitempty"`
	PendingAuthorization *mcpAuth.MCPOAuthAuthorization     `json:"pendingAuthorization,omitempty"`
}

func withMCPAggregate[T any](
	w *MCPAggregateWrapper,
	fn func(context.Context) (T, error),
) (T, error) {
	return withRecoveryResp(func() (T, error) {
		var zero T
		if err := w.lock(); err != nil {
			return zero, err
		}
		defer w.mu.Unlock()

		return fn(context.Background())
	})
}

func withMCPAggregateError(
	w *MCPAggregateWrapper,
	fn func(context.Context) error,
) error {
	_, err := withMCPAggregate(w, func(ctx context.Context) (struct{}, error) {
		return struct{}{}, fn(ctx)
	})
	return err
}

func (w *MCPAggregateWrapper) GetMCPServer(
	ref artifactModel.ArtifactRef,
) (MCPServerDetails, error) {
	return withMCPAggregate(w, func(ctx context.Context) (MCPServerDetails, error) {
		return w.getMCPServer(ctx, ref)
	})
}

func (w *MCPAggregateWrapper) ListMCPPluginServers(
	ref artifactModel.ArtifactRef,
) ([]MCPServerDetails, error) {
	return withMCPAggregate(w, func(ctx context.Context) ([]MCPServerDetails, error) {
		reads, err := w.store.ListMCPPluginServers(ctx, ref)
		if err != nil {
			return nil, err
		}

		output := make([]MCPServerDetails, 0, len(reads))
		for _, read := range reads {
			details, err := w.serverDetails(ctx, read)
			if err != nil {
				return nil, err
			}
			output = append(output, details)
		}
		return output, nil
	})
}

func (w *MCPAggregateWrapper) GetMCPServersForRuntimeServers(
	servers []mcpServer.ServerID,
) ([]MCPServerRuntimeDetails, error) {
	return withMCPAggregate(w, func(ctx context.Context) ([]MCPServerRuntimeDetails, error) {
		if len(servers) > mcpServer.MaxMCPServerPageSize {
			return nil, fmt.Errorf(
				"%w: too many MCP runtime status requests",
				mcpServer.ErrInvalid,
			)
		}

		// Validate every ID before reading runtime state. The map also
		// deduplicates the request while retaining first-occurrence order.
		refs := make(map[mcpServer.ServerID]artifactModel.ArtifactRef, len(servers))
		for _, server := range servers {
			ref, err := inferenceadapter.ArtifactRefForServerID(server)
			if err != nil {
				return nil, err
			}
			refs[server] = ref
		}

		pending := make(map[mcpServer.ServerID]mcpAuth.MCPOAuthAuthorization)
		for _, value := range w.auth.PendingAuthorizations() {
			pending[value.Server] = value
		}

		output := make([]MCPServerRuntimeDetails, 0, len(refs))
		for _, server := range servers {
			ref, found := refs[server]
			if !found {
				continue
			}
			delete(refs, server)

			connection, err := w.runtime.Status(ctx, server)
			if err != nil {
				return nil, err
			}
			if connection == nil {
				return nil, mcpServer.ErrClosed
			}

			value := MCPServerRuntimeDetails{
				Ref:        ref,
				Connection: *connection,
			}
			if status, found := w.auth.GetAuthStatus(server); found {
				value.Authorization = &status
			}
			if authorization, found := pending[server]; found {
				value.PendingAuthorization = &authorization
			}
			output = append(output, value)
		}
		return output, nil
	})
}

func (w *MCPAggregateWrapper) SaveMCPServerSettings(
	ref artifactModel.ArtifactRef,
	expectedSettingsRevision uint64,
	data serverMCPDomain.ServerData,
) (MCPServerDetails, error) {
	return withMCPAggregate(w, func(ctx context.Context) (MCPServerDetails, error) {
		read, err := w.store.ResolveMCPServer(ctx, ref)
		if err != nil {
			return MCPServerDetails{}, err
		}

		data = retainServerSecretBindings(
			read.Resolved.Installation,
			data,
			read.Resolved.Document,
		)
		return w.saveMCPServerSettings(
			ctx,
			ref,
			expectedSettingsRevision,
			data,
		)
	})
}

func (w *MCPAggregateWrapper) CreateMCPServer(
	request mcpAPI.ManagedMCPCreateRequest,
) (mcpAPI.ManagedMCPCreateResult, error) {
	return withMCPAggregate(w, func(ctx context.Context) (mcpAPI.ManagedMCPCreateResult, error) {
		result, err := w.store.CreateMCPServer(ctx, request)
		if err != nil {
			return mcpAPI.ManagedMCPCreateResult{}, err
		}

		if err := w.invalidateServer(ctx, result.Artifact.Ref()); err != nil {
			// Persistence already committed. Preserve the result instead
			// of reporting a zero value that looks like creation failed.
			return result, fmt.Errorf(
				"MCP server was created but runtime invalidation failed: %w",
				err,
			)
		}
		return result, nil
	})
}

func (w *MCPAggregateWrapper) UpdateMCPServer(
	request mcpAPI.ManagedMCPReplaceRequest,
) (mcpAPI.ManagedMCPReplaceResult, error) {
	return withMCPAggregate(w, func(ctx context.Context) (mcpAPI.ManagedMCPReplaceResult, error) {
		if err := w.invalidateServer(ctx, request.Artifact); err != nil {
			return mcpAPI.ManagedMCPReplaceResult{}, err
		}
		return w.store.UpdateMCPServer(ctx, request)
	})
}

func (w *MCPAggregateWrapper) DeleteMCPServer(
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
) error {
	return withMCPAggregateError(w, func(ctx context.Context) error {
		if err := w.invalidateServer(ctx, ref); err != nil {
			return err
		}
		return w.store.DeleteMCPServer(ctx, ref, expectedRevision)
	})
}

func (w *MCPAggregateWrapper) SaveMCPPolicy(
	request mcpAPI.ManagedMCPPolicyUpsertRequest,
) (mcpAPI.ManagedMCPPolicyUpsertResult, error) {
	return withMCPAggregate(w, func(ctx context.Context) (mcpAPI.ManagedMCPPolicyUpsertResult, error) {
		// These fields are used for invalidation before the Store write,
		// so validate them before performing that side effect.
		if err := request.Plugin.Validate(); err != nil {
			return mcpAPI.ManagedMCPPolicyUpsertResult{}, err
		}
		if err := request.Name.Validate(); err != nil {
			return mcpAPI.ManagedMCPPolicyUpsertResult{}, err
		}

		affected, err := w.store.ListMCPServersReferencingPolicy(
			ctx,
			request.Plugin.RootID,
			request.Name,
		)
		if err != nil {
			return mcpAPI.ManagedMCPPolicyUpsertResult{}, err
		}
		if err := w.invalidateServers(ctx, affected); err != nil {
			return mcpAPI.ManagedMCPPolicyUpsertResult{}, err
		}
		return w.store.SaveMCPPolicy(ctx, request)
	})
}

func (w *MCPAggregateWrapper) DeleteMCPPolicy(
	ref artifactModel.ArtifactRef,
	expectedRevision uint64,
) error {
	return withMCPAggregateError(w, func(ctx context.Context) error {
		policy, err := w.store.GetMCPPolicy(ctx, ref)
		if err != nil {
			return err
		}
		affected, err := w.store.ListMCPServersReferencingPolicy(
			ctx,
			policy.Artifact.RootID,
			policy.Artifact.LogicalName,
		)
		if err != nil {
			return err
		}
		if err := w.invalidateServers(ctx, affected); err != nil {
			return err
		}
		return w.store.DeleteMCPPolicy(ctx, ref, expectedRevision)
	})
}

func (w *MCPAggregateWrapper) SetMCPServerSecret(
	ref artifactModel.ArtifactRef,
	input string,
	value string,
) (MCPServerDetails, error) {
	return withMCPAggregate(w, func(ctx context.Context) (MCPServerDetails, error) {
		read, err := w.store.ResolveMCPServer(ctx, ref)
		if err != nil {
			return MCPServerDetails{}, err
		}
		kind, slot, err := mcpServerSecretTarget(read.Resolved, input)
		if err != nil {
			return MCPServerDetails{}, err
		}

		if kind == secret.MCPSecretKindOAuthClientCredentials {
			if err := mcpAuth.ValidateOAuthClientCredentialsSecret(
				value,
				read.Resolved.Document.OAuthClientSecretRequired(),
			); err != nil {
				return MCPServerDetails{}, err
			}
		}
		if kind == secret.MCPSecretKindHTTPHeader &&
			(strings.TrimSpace(value) == "" ||
				strings.ContainsAny(value, "\r\n\x00")) {
			return MCPServerDetails{}, fmt.Errorf(
				"%w: invalid HTTP header secret value",
				mcpAuth.ErrMCPInvalidAuthRequest,
			)
		}

		secretRef, err := secret.NewMCPSecretRefString(ref, kind, slot)
		if err != nil {
			return MCPServerDetails{}, err
		}
		data := read.Resolved.Installation.Clone()
		if data.Inputs == nil {
			data.Inputs = map[string]serverMCPDomain.InputBinding{}
		}
		data.Inputs[input] = serverMCPDomain.InputBinding{
			SecretRef: secretRef,
		}

		// Replacing an existing secret changes live configuration even if
		// the subsequent installation write fails. Evict the old runtime
		// before performing either persistent write.
		if err := w.invalidateServer(ctx, ref); err != nil {
			return MCPServerDetails{}, err
		}
		if _, _, err := w.secrets.SetMCPSecret(ctx, secretRef, value); err != nil {
			return MCPServerDetails{}, err
		}
		if err := w.store.SaveServerSettings(
			ctx,
			ref,
			read.Settings.InstallationRevision,
			data,
		); err != nil {
			// These persistence operations are not one transaction.
			// Do not "roll back" a shared secret slot with another write.
			return MCPServerDetails{}, fmt.Errorf(
				"MCP secret was saved but installation settings were not saved: %w",
				err,
			)
		}
		return w.getMCPServer(ctx, ref)
	})
}

func (w *MCPAggregateWrapper) ClearMCPServerSecret(
	ref artifactModel.ArtifactRef,
	input string,
) (MCPServerDetails, error) {
	return withMCPAggregate(w, func(ctx context.Context) (MCPServerDetails, error) {
		read, err := w.store.ResolveMCPServer(ctx, ref)
		if err != nil {
			return MCPServerDetails{}, err
		}
		if _, _, err := mcpServerSecretTarget(read.Resolved, input); err != nil {
			return MCPServerDetails{}, err
		}

		data := read.Resolved.Installation.Clone()
		binding, found := data.Inputs[input]
		if !found || binding.SecretRef == "" {
			return w.serverDetails(ctx, read)
		}
		delete(data.Inputs, input)

		// Store owns cleanup of references removed from the installation.
		return w.saveMCPServerSettings(
			ctx,
			ref,
			read.Settings.InstallationRevision,
			data,
		)
	})
}

func mcpServerSecretTarget(
	resolved serverMCPDomain.Resolved,
	input string,
) (secret.MCPSecretKind, string, error) {
	declaration, found := resolved.Document.Configuration.Install.Inputs[input]
	if !found {
		return "", "", fmt.Errorf(
			"%w: MCP secret input %q is not declared",
			mcpAuth.ErrMCPInvalidAuthRequest,
			input,
		)
	}

	switch declaration.Kind {
	case serverMCPDomain.InputOAuthClientCredentials:
		if resolved.Document.Configuration.Auth.ClientCredentialsInput != input {
			return "", "", fmt.Errorf(
				"%w: MCP secret input %q is not an OAuth client credential input",
				mcpAuth.ErrMCPInvalidAuthRequest,
				input,
			)
		}
		return secret.MCPSecretKindOAuthClientCredentials, "clientCredentials", nil

	case serverMCPDomain.InputSecret:
		targets, err := resolved.Document.SecretInputTargets()
		if err != nil {
			return "", "", err
		}
		target, found := targets[input]
		if !found {
			return "", "", fmt.Errorf(
				"%w: MCP secret input %q has no target",
				mcpAuth.ErrMCPInvalidAuthRequest,
				input,
			)
		}
		if target.Kind == serverMCPDomain.SecretInputTargetHTTPHeader {
			return secret.MCPSecretKindHTTPHeader, target.Slot, nil
		}
		return secret.MCPSecretKindStdioEnv, target.Slot, nil

	default:
		return "", "", fmt.Errorf(
			"%w: MCP input %q does not accept a secret",
			mcpAuth.ErrMCPInvalidAuthRequest,
			input,
		)
	}
}

func (w *MCPAggregateWrapper) getMCPServer(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (MCPServerDetails, error) {
	read, err := w.store.ResolveMCPServer(ctx, ref)
	if err != nil {
		return MCPServerDetails{}, err
	}
	return w.serverDetails(ctx, read)
}

func (w *MCPAggregateWrapper) serverDetails(
	ctx context.Context,
	read mcpAPI.ServerRead,
) (MCPServerDetails, error) {
	authorization, err := w.serverAuthHealth(ctx, read.Resolved)
	if err != nil {
		return MCPServerDetails{}, err
	}

	serverID, err := inferenceadapter.ServerIDForArtifact(read.Resolved.Server)
	if err != nil {
		return MCPServerDetails{}, err
	}
	connection, err := w.runtime.Status(ctx, serverID)
	if err != nil {
		return MCPServerDetails{}, err
	}
	if connection == nil {
		return MCPServerDetails{}, mcpServer.ErrClosed
	}

	return MCPServerDetails{
		Settings:      read.Settings,
		Policy:        read.Resolved.Policy,
		Authorization: authorization,
		Connection:    *connection,
	}, nil
}

func (w *MCPAggregateWrapper) serverAuthHealth(
	ctx context.Context,
	resolved serverMCPDomain.Resolved,
) (mcpAuth.MCPAuthHealth, error) {
	config, err := w.source.InspectRuntimeConfig(ctx, resolved)
	if err == nil {
		return w.auth.BuildAuthHealth(ctx, config), nil
	}
	if ctx.Err() != nil {
		return mcpAuth.MCPAuthHealth{}, ctx.Err()
	}

	serverID, idErr := inferenceadapter.ServerIDForArtifact(resolved.Server)
	if idErr != nil {
		return mcpAuth.MCPAuthHealth{}, idErr
	}
	mode, modeErr := inferenceadapter.HTTPAuthMode(
		resolved.Document.Configuration.Auth.Mode,
	)
	if modeErr != nil {
		return mcpAuth.MCPAuthHealth{}, modeErr
	}

	// Do not expose materialization errors or secret values in a
	// management response for an incompletely configured installation.
	return mcpAuth.MCPAuthHealth{
		Server:     serverID,
		AuthMode:   mode,
		State:      mcpAuth.MCPAuthHealthStateNotConfigured,
		Configured: false,
		LastError:  "required MCP installation input is not configured",
	}, nil
}

func (w *MCPAggregateWrapper) saveMCPServerSettings(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
	expectedSettingsRevision uint64,
	data serverMCPDomain.ServerData,
) (MCPServerDetails, error) {
	if err := w.invalidateServer(ctx, ref); err != nil {
		return MCPServerDetails{}, err
	}
	if err := w.store.SaveServerSettings(
		ctx,
		ref,
		expectedSettingsRevision,
		data,
	); err != nil {
		return MCPServerDetails{}, err
	}
	return w.getMCPServer(ctx, ref)
}

func retainServerSecretBindings(
	current serverMCPDomain.ServerData,
	next serverMCPDomain.ServerData,
	document serverMCPDomain.ServerDocument,
) serverMCPDomain.ServerData {
	output := next.Clone()
	if output.Inputs == nil {
		output.Inputs = map[string]serverMCPDomain.InputBinding{}
	}

	for name, declaration := range document.Configuration.Install.Inputs {
		switch declaration.Kind {
		case serverMCPDomain.InputSecret,
			serverMCPDomain.InputOAuthClientCredentials:
		default:
			continue
		}

		if binding, found := current.Inputs[name]; found {
			output.Inputs[name] = binding
		} else {
			delete(output.Inputs, name)
		}
	}
	return output
}

// invalidateServer clears process-local auth observations only after eviction
// succeeds. A later connection rebuilds them from the current installation.
func (w *MCPAggregateWrapper) invalidateServer(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) error {
	serverID, err := inferenceadapter.ServerIDForArtifact(ref)
	if err != nil {
		return err
	}
	if err := w.runtime.Invalidate(ctx, serverID); err != nil {
		return err
	}
	w.auth.ClearAuthStatus(serverID)
	return nil
}

// invalidateServers visits a deterministic unique set and reports all failures.
// Policy changes invalidate only servers whose effective policy can change.
func (w *MCPAggregateWrapper) invalidateServers(
	ctx context.Context,
	refs []artifactModel.ArtifactRef,
) error {
	unique := make(map[artifactModel.ArtifactRef]struct{}, len(refs))
	for _, ref := range refs {
		unique[ref] = struct{}{}
	}

	ordered := make([]artifactModel.ArtifactRef, 0, len(unique))
	for ref := range unique {
		ordered = append(ordered, ref)
	}
	sort.Slice(ordered, func(left, right int) bool {
		if ordered[left].RootID != ordered[right].RootID {
			return ordered[left].RootID < ordered[right].RootID
		}
		return ordered[left].ArtifactID < ordered[right].ArtifactID
	})

	var result error
	for _, ref := range ordered {
		result = errors.Join(result, w.invalidateServer(ctx, ref))
	}
	return result
}

// lock acquires the wrapper lifecycle lock on success. Internal helpers run
// under this lock and do not repeat dependency checks or acquire it again.
func (w *MCPAggregateWrapper) lock() error {
	if w == nil {
		return spec.ErrClosed
	}
	w.mu.Lock()
	if w.store == nil ||
		w.source == nil ||
		w.runtime == nil ||
		w.auth == nil ||
		w.secrets == nil {
		w.mu.Unlock()
		return spec.ErrClosed
	}
	return nil
}

func (w *MCPAggregateWrapper) close() {
	if w == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()

	w.store = nil
	w.source = nil
	w.runtime = nil
	w.auth = nil
	w.secrets = nil
}
