package store

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/mcp/secret"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

const (
	maxMCPDisplayNameLen = 256
	maxMCPCommandLen     = 4096
	maxMCPURLLen         = 4096
	maxMCPHeaderNameLen  = 128
	maxMCPHeaderValueLen = 4096
)

var mcpIDRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

func validateServerConfig(c *spec.MCPServerConfig) error {
	if c == nil {
		return errors.New("server config is nil")
	}
	if c.SchemaVersion != spec.MCPSchemaVersion {
		return fmt.Errorf("schemaVersion %q != %q", c.SchemaVersion, spec.MCPSchemaVersion)
	}
	if !mcpIDRe.MatchString(string(c.ID)) {
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
		return validateHTTPAuthRef(c)
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
	if base == "sh" || base == "bash" || base == "zsh" || base == "cmd" ||
		base == "cmd.exe" || base == "powershell" || base == "powershell.exe" ||
		base == "pwsh" || base == "pwsh.exe" {
		return errors.New("stdio.command must execute the server directly, not through a shell")
	}

	for k := range c.Env {
		if strings.TrimSpace(k) == "" {
			return errors.New("stdio.env contains empty key")
		}
	}
	for k, ref := range c.SecretEnvRefs {
		if strings.TrimSpace(k) == "" || strings.TrimSpace(ref) == "" {
			return errors.New("stdio.secretEnvRefs contains empty key or ref")
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
	if len(raw) > maxMCPURLLen {
		return fmt.Errorf("streamableHttp.url too long > %d", maxMCPURLLen)
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("streamableHttp.url invalid: %w", err)
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return errors.New("streamableHttp.url scheme must be http or https")
	}
	if u.Host == "" {
		return errors.New("streamableHttp.url host is empty")
	}

	switch c.AuthMode {
	case "", spec.MCPHTTPAuthNone, spec.MCPHTTPAuthOAuth, spec.MCPHTTPAuthClientCredentials,
		spec.MCPHTTPAuthCustomBearer, spec.MCPHTTPAuthCustomHeaders:
	default:
		return fmt.Errorf("invalid streamableHttp.authMode %q", c.AuthMode)
	}
	if c.AuthMode == "" {
		c.AuthMode = spec.MCPHTTPAuthNone
	}

	for k, v := range c.CustomHeaders {
		if strings.TrimSpace(k) == "" {
			return errors.New("streamableHttp.customHeaders contains empty header")
		}
		if strings.EqualFold(k, "authorization") {
			return errors.New("authorization must use authRef/tokenRef, not customHeaders")
		}
		if err := validateHTTPHeaderName(k); err != nil {
			return fmt.Errorf("streamableHttp.customHeaders[%q]: %w", k, err)
		}
		if len(v) > maxMCPHeaderValueLen {
			return errors.New("streamableHttp.customHeaders contains oversized key/value")
		}
	}
	for k, ref := range c.SecretHeaderRefs {
		if strings.TrimSpace(k) == "" || strings.TrimSpace(ref) == "" {
			return errors.New("streamableHttp.secretHeaderRefs contains empty key or ref")
		}
		if strings.EqualFold(k, "authorization") {
			return errors.New("authorization must use authRef/tokenRef, not secretHeaderRefs")
		}
		if err := validateHTTPHeaderName(k); err != nil {
			return fmt.Errorf("streamableHttp.secretHeaderRefs[%q]: %w", k, err)
		}
		if err := secret.ValidateMCPSecretRef(ref, serverID, spec.MCPSecretKindHTTPHeader, k); err != nil {
			return fmt.Errorf("streamableHttp.secretHeaderRefs[%q]: %w", k, err)
		}
	}
	if err := validateNoHeaderOverlap(c.CustomHeaders, c.SecretHeaderRefs); err != nil {
		return err
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
		if strings.TrimSpace(k) == "" {
			return errors.New("stdio.env contains empty key")
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

func validateHTTPAuthRef(c *spec.MCPServerConfig) error {
	if c == nil || c.StreamableHTTP == nil || c.AuthRef == nil {
		return nil
	}

	mode := c.StreamableHTTP.AuthMode
	authMode := spec.MCPHTTPAuthMode(strings.TrimSpace(string(c.AuthRef.AuthMode)))
	tokenRef := strings.TrimSpace(c.AuthRef.TokenRef)
	clientCredentialRef := strings.TrimSpace(c.AuthRef.ClientCredentialRef)
	metadataRef := strings.TrimSpace(c.AuthRef.MetadataRef)

	if tokenRef != "" && mode != spec.MCPHTTPAuthCustomBearer {
		return errors.New("authRef.tokenRef is only valid when streamableHttp.authMode is customBearer")
	}

	if mode == spec.MCPHTTPAuthCustomBearer {
		if tokenRef == "" {
			return errors.New("authRef.tokenRef is required for customBearer authMode")
		}
		if err := secret.ValidateMCPSecretRef(
			tokenRef,
			c.ID,
			spec.MCPSecretKindHTTPToken,
			"authorization",
		); err != nil {
			return err
		}
		if authMode != "" && authMode != mode {
			return fmt.Errorf("authRef.authMode %q must match streamableHttp.authMode %q", authMode, mode)
		}
		if clientCredentialRef != "" || metadataRef != "" {
			return errors.New(
				"authRef.clientCredentialRef and authRef.metadataRef must be empty for customBearer authMode",
			)
		}
	} else if authMode == spec.MCPHTTPAuthCustomBearer {
		return fmt.Errorf("authRef.authMode %q does not match streamableHttp.authMode %q", authMode, mode)
	}

	return nil
}

func validateNoHeaderOverlap(customHeaders, secretHeaderRefs map[string]string) error {
	seen := make(map[string]string, len(customHeaders)+len(secretHeaderRefs))

	add := func(kind string, headers map[string]string, valuesAreRefs bool) error {
		for k, v := range headers {
			if strings.TrimSpace(k) == "" {
				return fmt.Errorf("%s contains empty header name", kind)
			}
			if strings.EqualFold(k, "authorization") {
				return fmt.Errorf("%s must not define authorization", kind)
			}
			if err := validateHTTPHeaderName(k); err != nil {
				return fmt.Errorf("%s[%q]: %w", kind, k, err)
			}
			if valuesAreRefs && strings.TrimSpace(v) == "" {
				return fmt.Errorf("%s[%q] contains empty secret ref", kind, k)
			}

			lower := strings.ToLower(k)
			if prev, ok := seen[lower]; ok {
				return fmt.Errorf("%s overlaps %s on header %q", prev, kind, k)
			}
			seen[lower] = kind
		}
		return nil
	}

	if err := add("streamableHttp.customHeaders", customHeaders, false); err != nil {
		return err
	}
	if err := add("streamableHttp.secretHeaderRefs", secretHeaderRefs, true); err != nil {
		return err
	}
	return nil
}

func validateHTTPHeaderName(name string) error {
	for _, c := range name {
		if c <= 0x20 || c > 0x7E || c == ':' {
			return fmt.Errorf("invalid header name %q", name)
		}
	}
	return nil
}

func isSoftDeleted(c *spec.MCPServerConfig) bool {
	return c != nil && c.SoftDeletedAt != nil && !c.SoftDeletedAt.IsZero()
}
