package store

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

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
		return validateStdioConfig(c.Stdio)
	case spec.MCPTransportStreamableHTTP:
		if c.StreamableHTTP == nil {
			return errors.New("streamableHttp config required")
		}
		if c.Stdio != nil {
			return errors.New("stdio must be empty for streamableHttp transport")
		}
		return validateHTTPConfig(c.StreamableHTTP)
	default:
		return fmt.Errorf("invalid transport %q", c.Transport)
	}
}

func validateStdioConfig(c *spec.MCPStdioConfig) error {
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
	}
	return nil
}

func validateHTTPConfig(c *spec.MCPStreamableHTTPConfig) error {
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
		if len(k) > maxMCPHeaderNameLen || len(v) > maxMCPHeaderValueLen {
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

func isSoftDeleted(c *spec.MCPServerConfig) bool {
	return c != nil && c.SoftDeletedAt != nil && !c.SoftDeletedAt.IsZero()
}
