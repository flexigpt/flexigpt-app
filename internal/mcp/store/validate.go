package store

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/mcp/secret"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

const (
	maxMCPDisplayNameLen = 256
	maxMCPCommandLen     = 4096
	maxMCPURLLen         = 4096
	commandBash          = "bash"
)

func validateServerConfig(c *spec.MCPServerConfig) error {
	if c == nil {
		return errors.New("server config is nil")
	}
	if c.SchemaVersion != spec.MCPSchemaVersion {
		return fmt.Errorf("schemaVersion %q != %q", c.SchemaVersion, spec.MCPSchemaVersion)
	}
	if strings.TrimSpace(string(c.ID)) == "" {
		return errors.New("id must match ^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$")
	}
	if strings.TrimSpace(c.DisplayName) == "" {
		return errors.New("displayName is empty")
	}
	if strings.TrimSpace(c.DisplayName) != c.DisplayName {
		return errors.New("displayName has leading/trailing whitespace")
	}
	if len(c.DisplayName) > maxMCPDisplayNameLen {
		return fmt.Errorf("displayName too long > %d", maxMCPDisplayNameLen)
	}
	if c.CreatedAt.IsZero() || c.ModifiedAt.IsZero() {
		return errors.New("createdAt/modifiedAt is zero")
	}
	if c.ModifiedAt.Before(c.CreatedAt) {
		return errors.New("modifiedAt is before createdAt")
	}
	if isSoftDeleted(c) && c.Enabled {
		return errors.New("soft-deleted server cannot be enabled")
	}

	switch c.Availability {
	case "", spec.MCPServerAvailabilityManual, spec.MCPServerAvailabilityAutoAttach:
	default:
		return fmt.Errorf("invalid availability %q", c.Availability)
	}
	if c.Availability == "" {
		c.Availability = spec.MCPServerAvailabilityManual
	}

	switch c.TrustLevel {
	case "", spec.MCPTrustLevelUntrusted, spec.MCPTrustLevelTrusted:
	default:
		return fmt.Errorf("invalid trustLevel %q", c.TrustLevel)
	}
	if c.TrustLevel == "" {
		c.TrustLevel = spec.MCPTrustLevelUntrusted
	}

	if err := validatePolicy(c.DefaultPolicy); err != nil {
		return fmt.Errorf("defaultPolicy: %w", err)
	}
	for k, p := range c.ToolPolicies {
		if strings.TrimSpace(k) == "" {
			return errors.New("toolPolicies contains empty key")
		}
		if p.ToolName == "" {
			p.ToolName = k
		}
		if p.ToolName != k {
			return fmt.Errorf("toolPolicies key %q != toolName %q", k, p.ToolName)
		}
		if err := validateToolPolicyOverride(p); err != nil {
			return fmt.Errorf("toolPolicies[%s]: %w", k, err)
		}
	}

	switch c.Transport {
	case spec.MCPTransportStdio:
		if c.Stdio == nil {
			return errors.New("stdio config required")
		}
		if c.StreamableHTTP != nil {
			return errors.New("streamableHttp must be empty for stdio transport")
		}
		return validateStdioConfig(c.ID, c.Stdio)
	case spec.MCPTransportStreamableHTTP:
		if c.StreamableHTTP == nil {
			return errors.New("streamableHttp config required")
		}
		if c.Stdio != nil {
			return errors.New("stdio must be empty for streamableHttp transport")
		}
		if err := validateHTTPConfig(c.ID, c.StreamableHTTP); err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("invalid transport %q", c.Transport)
	}
}

func validateStdioConfig(serverID spec.MCPServerID, c *spec.MCPStdioConfig) error {
	if strings.TrimSpace(c.Command) == "" {
		return errors.New("stdio.command is empty")
	}
	if strings.TrimSpace(c.Command) != c.Command {
		return errors.New("stdio.command has leading/trailing whitespace")
	}
	if len(c.Command) > maxMCPCommandLen {
		return fmt.Errorf("stdio.command too long > %d", maxMCPCommandLen)
	}

	base := strings.ToLower(filepath.Base(c.Command))
	if base == commandBash || base == "sh" || base == "zsh" || base == "cmd" ||
		base == "cmd.exe" || base == "powershell" || base == "powershell.exe" ||
		base == "pwsh" || base == "pwsh.exe" {
		return errors.New("stdio.command must execute the server directly, not through a shell")
	}

	for k := range c.Env {
		if err := validateEnvKey(k); err != nil {
			return fmt.Errorf("stdio.env[%q]: %w", k, err)
		}
	}
	for k, ref := range c.SecretEnvRefs {
		if err := validateEnvKey(k); err != nil {
			return fmt.Errorf("stdio.secretEnvRefs[%q]: %w", k, err)
		}
		if strings.TrimSpace(ref) == "" {
			return fmt.Errorf("stdio.secretEnvRefs[%q] contains empty ref", k)
		}
		if err := secret.ValidateMCPSecretRef(ref, serverID, spec.MCPSecretKindStdioEnv, k); err != nil {
			return fmt.Errorf("stdio.secretEnvRefs[%q]: %w", k, err)
		}
	}
	if err := validateNoEnvKeyOverlap(c.Env, c.SecretEnvRefs); err != nil {
		return err
	}
	return nil
}

func validateHTTPConfig(serverID spec.MCPServerID, c *spec.MCPStreamableHTTPConfig) error {
	raw := strings.TrimSpace(c.URL)
	if raw == "" {
		return errors.New("streamableHttp.url is empty")
	}
	if raw != c.URL {
		return errors.New("streamableHttp.url has leading/trailing whitespace")
	}
	if len(raw) > maxMCPURLLen {
		return fmt.Errorf("streamableHttp.url too long > %d", maxMCPURLLen)
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("streamableHttp.url invalid: %w", err)
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return errors.New("streamableHttp.url scheme must be http/s")
	}
	if u.Host == "" {
		return errors.New("streamableHttp.url host is empty")
	}
	if u.User != nil {
		return errors.New("streamableHttp.url must not contain user info")
	}
	if u.Fragment != "" {
		return errors.New("streamableHttp.url must not contain a fragment")
	}
	if u.Scheme == "http" && !isLoopbackHost(u.Hostname()) {
		return errors.New("streamableHttp.url using http is only allowed for loopback hosts")
	}

	switch c.AuthMode {
	case "", spec.MCPHTTPAuthNone, spec.MCPHTTPAuthOAuth, spec.MCPHTTPAuthClientCredentials:
	default:
		return fmt.Errorf("invalid streamableHttp.authMode %q", c.AuthMode)
	}
	if c.AuthMode == "" {
		c.AuthMode = spec.MCPHTTPAuthNone
	}

	clientIDMetadataDocumentURL := strings.TrimSpace(c.ClientIDMetadataDocumentURL)
	if clientIDMetadataDocumentURL != "" {
		if clientIDMetadataDocumentURL != c.ClientIDMetadataDocumentURL {
			return errors.New(
				"streamableHttp.clientIDMetadataDocumentURL has leading/trailing whitespace",
			)
		}
		if err := validateClientIDMetadataDocumentURL(clientIDMetadataDocumentURL); err != nil {
			return fmt.Errorf("streamableHttp.clientIDMetadataDocumentURL: %w", err)
		}
	}

	switch c.AuthMode {
	case spec.MCPHTTPAuthClientCredentials:
		if clientIDMetadataDocumentURL != "" {
			return errors.New(
				"streamableHttp.clientIDMetadataDocumentURL is only allowed for oauth authMode",
			)
		}
		if strings.TrimSpace(c.ClientCredentialRef) == "" {
			return errors.New("streamableHttp.clientCredentialRef is required for clientCredentials authMode")
		}
		if err := validateOAuthClientCredentialRef(serverID, c.ClientCredentialRef); err != nil {
			return fmt.Errorf("streamableHttp.clientCredentialRef: %w", err)
		}
	case spec.MCPHTTPAuthOAuth:
		if strings.TrimSpace(c.ClientCredentialRef) != "" {
			if err := validateOAuthClientCredentialRef(serverID, c.ClientCredentialRef); err != nil {
				return fmt.Errorf("streamableHttp.clientCredentialRef: %w", err)
			}
		}
	case spec.MCPHTTPAuthNone:
		if strings.TrimSpace(c.ClientCredentialRef) != "" {
			return errors.New(
				"streamableHttp.clientCredentialRef is only allowed when authMode is oauth or clientCredentials",
			)
		}
		if clientIDMetadataDocumentURL != "" {
			return errors.New(
				"streamableHttp.clientIDMetadataDocumentURL is only allowed for oauth authMode",
			)
		}
	}
	return nil
}

func validateOAuthClientCredentialRef(serverID spec.MCPServerID, ref string) error {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return errors.New("streamableHttp.clientCredentialRef is empty")
	}
	return secret.ValidateMCPSecretRef(
		ref,
		serverID,
		spec.MCPSecretKindOAuthClientCredentials,
		"clientCredentials",
	)
}

func validateClientIDMetadataDocumentURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "https" {
		return errors.New("must use https")
	}
	if u.Host == "" {
		return errors.New("host is empty")
	}
	if u.User != nil {
		return errors.New("must not contain user info")
	}
	if u.Path == "" {
		return errors.New("must include a path")
	}
	if u.Fragment != "" {
		return errors.New("must not contain a fragment")
	}
	return nil
}

func validatePolicy(p spec.MCPServerPolicy) error {
	switch p.DefaultApprovalRule {
	case spec.MCPApprovalRuleAsk, spec.MCPApprovalRuleAllow, spec.MCPApprovalRuleDeny:
	default:
		return fmt.Errorf("invalid defaultApprovalRule %q", p.DefaultApprovalRule)
	}
	switch p.DefaultExecutionMode {
	case spec.MCPExecutionModeManual, spec.MCPExecutionModeAuto:
	default:
		return fmt.Errorf("invalid defaultExecutionMode %q", p.DefaultExecutionMode)
	}
	return nil
}

func validateToolPolicyOverride(p spec.MCPToolPolicyOverride) error {
	if strings.TrimSpace(p.ToolName) == "" {
		return errors.New("toolName is empty")
	}
	if p.ApprovalRule != nil {
		switch *p.ApprovalRule {
		case spec.MCPApprovalRuleAsk, spec.MCPApprovalRuleAllow, spec.MCPApprovalRuleDeny:
		default:
			return fmt.Errorf("invalid approvalRule %q", *p.ApprovalRule)
		}
	}
	if p.ExecutionMode != nil {
		switch *p.ExecutionMode {
		case spec.MCPExecutionModeManual, spec.MCPExecutionModeAuto:
		default:
			return fmt.Errorf("invalid executionMode %q", *p.ExecutionMode)
		}
	}
	return nil
}

func validateNoEnvKeyOverlap(env, secretEnvRefs map[string]string) error {
	seen := make(map[string]string, len(env))
	for k := range env {
		if err := validateEnvKey(k); err != nil {
			return fmt.Errorf("stdio.env[%q]: %w", k, err)
		}
		seen[strings.ToLower(strings.TrimSpace(k))] = "stdio.env"
	}
	for k := range secretEnvRefs {
		if strings.TrimSpace(k) == "" {
			return errors.New("stdio.secretEnvRefs contains empty key")
		}
		key := strings.ToLower(strings.TrimSpace(k))
		if prev, ok := seen[key]; ok {
			return fmt.Errorf("%s and stdio.secretEnvRefs both define %q", prev, k)
		}
	}
	return nil
}

func validateEnvKey(key string) error {
	if strings.TrimSpace(key) == "" {
		return errors.New("env key is empty")
	}
	if strings.TrimSpace(key) != key {
		return errors.New("env key has leading/trailing whitespace")
	}
	if strings.ContainsAny(key, "=\x00") {
		return errors.New("env key must not contain '=' or NUL")
	}
	for _, c := range key {
		if c < 0x20 || c == 0x7f {
			return fmt.Errorf("env key contains control character %q", c)
		}
	}
	return nil
}

func isSoftDeleted(c *spec.MCPServerConfig) bool {
	return c != nil && c.SoftDeletedAt != nil && !c.SoftDeletedAt.IsZero()
}

func isLoopbackHost(host string) bool {
	host = strings.TrimSpace(host)
	if host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
