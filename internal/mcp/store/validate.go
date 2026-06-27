package store

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/bundleitemutils"
	"github.com/flexigpt/flexigpt-app/internal/mcp/secret"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

const (
	maxMCPDisplayNameLen = 256
	maxMCPCommandLen     = 4096
	maxMCPURLLen         = 4096
	maxMCPSetupTextLen   = 4096

	commandBash = "bash"
)

var mcpServerIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

func validateBundle(b *spec.MCPBundle) error {
	if b == nil {
		return errors.New("bundle is nil")
	}
	if b.SchemaVersion != spec.MCPSchemaVersion {
		return fmt.Errorf("schemaVersion %q != %q", b.SchemaVersion, spec.MCPSchemaVersion)
	}
	if err := bundleitemutils.ValidateBundleSlug(b.Slug); err != nil {
		return fmt.Errorf("slug: %w", err)
	}
	if strings.TrimSpace(string(b.ID)) == "" {
		return errors.New("id is empty")
	}
	if strings.TrimSpace(b.DisplayName) == "" {
		return errors.New("displayName is empty")
	}
	if strings.TrimSpace(b.DisplayName) != b.DisplayName {
		return errors.New("displayName has leading/trailing whitespace")
	}
	if len(b.DisplayName) > maxMCPDisplayNameLen {
		return fmt.Errorf("displayName too long > %d", maxMCPDisplayNameLen)
	}
	if b.CreatedAt.IsZero() || b.ModifiedAt.IsZero() {
		return errors.New("createdAt/modifiedAt is zero")
	}
	if b.ModifiedAt.Before(b.CreatedAt) {
		return errors.New("modifiedAt is before createdAt")
	}
	return nil
}

func validateServerConfig(c *spec.MCPServerConfig) error {
	if c == nil {
		return errors.New("server config is nil")
	}
	if c.SchemaVersion != spec.MCPSchemaVersion {
		return fmt.Errorf("schemaVersion %q != %q", c.SchemaVersion, spec.MCPSchemaVersion)
	}
	if strings.TrimSpace(string(c.BundleID)) == "" {
		return errors.New("bundleID is empty")
	}
	id := strings.TrimSpace(string(c.ID))
	if id == "" || id != string(c.ID) || !mcpServerIDPattern.MatchString(id) {
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
		c.ToolPolicies[k] = p
	}
	if err := validateServerSetup(c); err != nil {
		return fmt.Errorf("setup: %w", err)
	}
	switch c.Transport {
	case spec.MCPTransportStdio:
		if c.Stdio == nil {
			return errors.New("stdio config required")
		}
		if c.StreamableHTTP != nil {
			return errors.New("streamableHttp must be empty for stdio transport")
		}
		return validateStdioConfig(c.BundleID, c.ID, c.Stdio)
	case spec.MCPTransportStreamableHTTP:
		if c.StreamableHTTP == nil {
			return errors.New("streamableHttp config required")
		}
		if c.Stdio != nil {
			return errors.New("stdio must be empty for streamableHttp transport")
		}
		if err := validateHTTPConfig(
			c.BundleID,
			c.ID,
			c.StreamableHTTP,
			setupDeclaresClientCredentials(c.Setup),
			setupDeclaresAPIKeyHeader(c.Setup),
		); err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("invalid transport %q", c.Transport)
	}
}

func validateStdioConfig(
	bundleID bundleitemutils.BundleID,
	serverID spec.MCPServerID,
	c *spec.MCPStdioConfig,
) error {
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
	seenSecretSlots := map[string]string{}
	for k, ref := range c.SecretEnvRefs {
		if err := validateEnvKey(k); err != nil {
			return fmt.Errorf("stdio.secretEnvRefs[%q]: %w", k, err)
		}
		slotKey := strings.ToLower(strings.TrimSpace(k))
		if prev := seenSecretSlots[slotKey]; prev != "" {
			return fmt.Errorf("stdio.secretEnvRefs keys %q and %q collide after secret-slot normalization", prev, k)
		}
		seenSecretSlots[slotKey] = k
		if strings.TrimSpace(ref) == "" {
			return fmt.Errorf("stdio.secretEnvRefs[%q] contains empty ref", k)
		}
		if err := secret.ValidateMCPSecretRef(ref, bundleID, serverID, spec.MCPSecretKindStdioEnv, k); err != nil {
			return fmt.Errorf("stdio.secretEnvRefs[%q]: %w", k, err)
		}
	}
	if err := validateNoEnvKeyOverlap(c.Env, c.SecretEnvRefs); err != nil {
		return err
	}
	return nil
}

func validateHTTPConfig(
	bundleID bundleitemutils.BundleID,
	serverID spec.MCPServerID,
	c *spec.MCPStreamableHTTPConfig,
	allowDeferredClientCredentialRef bool,
	allowDeferredAPIKey bool,
) error {
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
	case "", spec.MCPHTTPAuthNone, spec.MCPHTTPAuthAPIKey, spec.MCPHTTPAuthOAuth, spec.MCPHTTPAuthClientCredentials:
	default:
		return fmt.Errorf("invalid streamableHttp.authMode %q", c.AuthMode)
	}
	if c.AuthMode == "" {
		c.AuthMode = spec.MCPHTTPAuthNone
	}
	if err := validateHTTPHeaderConfig(bundleID, serverID, c); err != nil {
		return err
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
			// A server may defer the credential to a declared oauthClientCredentials
			// setup input. Connect-time gating ensures the credential is present
			// before a session is established.
			if !allowDeferredClientCredentialRef {
				return errors.New(
					"streamableHttp.clientCredentialRef is required for clientCredentials authMode",
				)
			}
		} else if err := validateOAuthClientCredentialRef(bundleID, serverID, c.ClientCredentialRef); err != nil {
			return fmt.Errorf("streamableHttp.clientCredentialRef: %w", err)
		}
	case spec.MCPHTTPAuthOAuth:
		if strings.TrimSpace(c.ClientCredentialRef) != "" {
			if err := validateOAuthClientCredentialRef(bundleID, serverID, c.ClientCredentialRef); err != nil {
				return fmt.Errorf("streamableHttp.clientCredentialRef: %w", err)
			}
		}
	case spec.MCPHTTPAuthAPIKey:
		if clientIDMetadataDocumentURL != "" {
			return errors.New(
				"streamableHttp.clientIDMetadataDocumentURL is only allowed for oauth authMode",
			)
		}
		if strings.TrimSpace(c.ClientCredentialRef) != "" {
			return errors.New(
				"streamableHttp.clientCredentialRef is only allowed when authMode is oauth or clientCredentials",
			)
		}
		if len(c.SecretHeaderRefs) == 0 && !allowDeferredAPIKey {
			return errors.New(
				"streamableHttp apikey authMode requires a secret header (provide it via setup or secretHeaderRefs)",
			)
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

func validateOAuthClientCredentialRef(
	bundleID bundleitemutils.BundleID,
	serverID spec.MCPServerID,
	ref string,
) error {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return errors.New("streamableHttp.clientCredentialRef is empty")
	}
	return secret.ValidateMCPSecretRef(
		ref,
		bundleID,
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
	if u.Path == "" || u.Path == "/" {
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

func validateMCPSettings(s spec.MCPSettings) error {
	addr := strings.TrimSpace(s.OAuthLoopbackListenAddr)
	if addr == "" {
		return nil
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("oauthLoopbackListenAddr must be host:port: %w", err)
	}
	if !isLoopbackHost(host) {
		return errors.New("oauthLoopbackListenAddr host must be loopback")
	}
	n, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("oauthLoopbackListenAddr port must be numeric: %w", err)
	}
	if n <= 0 || n > 65535 {
		return errors.New("oauthLoopbackListenAddr port must be 1..65535")
	}
	return nil
}

func validateHTTPHeaderConfig(
	bundleID bundleitemutils.BundleID,
	serverID spec.MCPServerID,
	c *spec.MCPStreamableHTTPConfig,
) error {
	for name, value := range c.Headers {
		if err := validateHTTPHeaderName(name); err != nil {
			return fmt.Errorf("streamableHttp.headers[%q]: %w", name, err)
		}
		if strings.ContainsAny(value, "\r\n\x00") {
			return fmt.Errorf("streamableHttp.headers[%q]: value must not contain CR/LF/NUL", name)
		}
	}

	seen := map[string]string{}
	for name, ref := range c.SecretHeaderRefs {
		if err := validateHTTPHeaderName(name); err != nil {
			return fmt.Errorf("streamableHttp.secretHeaderRefs[%q]: %w", name, err)
		}
		key := strings.ToLower(strings.TrimSpace(name))
		if prev := seen[key]; prev != "" {
			return fmt.Errorf("streamableHttp.secretHeaderRefs %q and %q collide", prev, name)
		}
		seen[key] = name
		if strings.TrimSpace(ref) == "" {
			return fmt.Errorf("streamableHttp.secretHeaderRefs[%q] is empty", name)
		}
		if err := secret.ValidateMCPSecretRef(ref, bundleID, serverID, spec.MCPSecretKindHTTPHeader, name); err != nil {
			return fmt.Errorf("streamableHttp.secretHeaderRefs[%q]: %w", name, err)
		}
	}

	for name := range c.Headers {
		if _, ok := seen[strings.ToLower(strings.TrimSpace(name))]; ok {
			return fmt.Errorf("streamableHttp header %q defined as both plain and secret", name)
		}
	}

	switch normalizeHTTPAuthMode(c.AuthMode) {
	case spec.MCPHTTPAuthOAuth, spec.MCPHTTPAuthClientCredentials:
		if hasHTTPHeader(c.Headers, "Authorization") || hasHTTPHeader(c.SecretHeaderRefs, "Authorization") {
			return errors.New("streamableHttp Authorization header is not allowed when OAuth authMode is enabled")
		}
	default:
	}
	return nil
}

func hasHTTPHeader(m map[string]string, name string) bool {
	for k := range m {
		if strings.EqualFold(strings.TrimSpace(k), name) {
			return true
		}
	}
	return false
}

func validateServerSetup(c *spec.MCPServerConfig) error {
	if c == nil || c.Setup == nil {
		return nil
	}
	if len(c.Setup.Note) > maxMCPSetupTextLen {
		return fmt.Errorf("note too long > %d", maxMCPSetupTextLen)
	}

	seen := map[string]struct{}{}
	for i := range c.Setup.Inputs {
		input := &c.Setup.Inputs[i]
		input.ID = strings.TrimSpace(input.ID)
		if input.ID == "" {
			return fmt.Errorf("inputs[%d].id is empty", i)
		}
		if _, ok := seen[input.ID]; ok {
			return fmt.Errorf("inputs[%d].id %q is duplicated", i, input.ID)
		}
		seen[input.ID] = struct{}{}
		if len(input.Description) > maxMCPSetupTextLen || len(input.Note) > maxMCPSetupTextLen {
			return fmt.Errorf("inputs[%d]: text too long", i)
		}
		if err := validateSetupInputUnion(c, i, input); err != nil {
			return err
		}
	}
	return nil
}

func validateSetupInputUnion(c *spec.MCPServerConfig, i int, input *spec.MCPServerSetupInput) error {
	set := 0
	if input.OAuthClientCredentials != nil {
		set++
	}
	if input.HTTPHeader != nil {
		set++
	}
	if input.StdioEnv != nil {
		set++
	}
	if input.StreamableHTTPURL != nil {
		set++
	}
	if input.ClientIDMetadataDocumentURL != nil {
		set++
	}
	if set != 1 {
		return fmt.Errorf("inputs[%d]: exactly one kind-specific block must be set", i)
	}

	httpMode := spec.MCPHTTPAuthNone
	if c.StreamableHTTP != nil {
		httpMode = normalizeHTTPAuthMode(c.StreamableHTTP.AuthMode)
	}

	switch input.Kind {
	case spec.MCPSetupKindOAuthClientCredentials:
		if input.OAuthClientCredentials == nil {
			return fmt.Errorf("inputs[%d]: kind/block mismatch", i)
		}
		if c.Transport != spec.MCPTransportStreamableHTTP {
			return fmt.Errorf("inputs[%d]: oauthClientCredentials requires streamableHttp", i)
		}
		if httpMode != spec.MCPHTTPAuthOAuth && httpMode != spec.MCPHTTPAuthClientCredentials {
			return fmt.Errorf("inputs[%d]: oauthClientCredentials requires oauth/clientCredentials authMode", i)
		}

	case spec.MCPSetupKindHTTPHeader:
		if input.HTTPHeader == nil {
			return fmt.Errorf("inputs[%d]: kind/block mismatch", i)
		}
		if c.Transport != spec.MCPTransportStreamableHTTP {
			return fmt.Errorf("inputs[%d]: httpHeader requires streamableHttp", i)
		}
		if err := validateHTTPHeaderName(input.HTTPHeader.HeaderName); err != nil {
			return fmt.Errorf("inputs[%d].httpHeader.headerName: %w", i, err)
		}
		if (httpMode == spec.MCPHTTPAuthOAuth || httpMode == spec.MCPHTTPAuthClientCredentials) &&
			strings.EqualFold(strings.TrimSpace(input.HTTPHeader.HeaderName), "Authorization") {
			return fmt.Errorf("inputs[%d]: Authorization header not allowed with OAuth authMode", i)
		}

	case spec.MCPSetupKindStdioEnv:
		if input.StdioEnv == nil {
			return fmt.Errorf("inputs[%d]: kind/block mismatch", i)
		}
		if c.Transport != spec.MCPTransportStdio {
			return fmt.Errorf("inputs[%d]: stdioEnv requires stdio", i)
		}
		if err := validateEnvKey(input.StdioEnv.EnvName); err != nil {
			return fmt.Errorf("inputs[%d].stdioEnv.envName: %w", i, err)
		}

	case spec.MCPSetupKindStreamableHTTPURL:
		if input.StreamableHTTPURL == nil || c.Transport != spec.MCPTransportStreamableHTTP {
			return fmt.Errorf("inputs[%d]: streamableHttpUrl requires streamableHttp", i)
		}

	case spec.MCPSetupKindClientIDMetadataDocURL:
		if input.ClientIDMetadataDocumentURL == nil || c.Transport != spec.MCPTransportStreamableHTTP {
			return fmt.Errorf("inputs[%d]: clientIDMetadataDocumentURL requires streamableHttp", i)
		}
		if httpMode != spec.MCPHTTPAuthOAuth {
			return fmt.Errorf("inputs[%d]: clientIDMetadataDocumentURL requires oauth authMode", i)
		}

	default:
		return fmt.Errorf("inputs[%d].kind %q is invalid", i, input.Kind)
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

func validateHTTPHeaderName(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("header name is empty")
	}
	if strings.TrimSpace(name) != name {
		return errors.New("header name has leading/trailing whitespace")
	}
	for _, r := range name {
		if !isHTTPTokenRune(r) {
			return fmt.Errorf("header name contains invalid character %q", r)
		}
	}
	return nil
}

func isHTTPTokenRune(r rune) bool {
	if r >= 'A' && r <= 'Z' ||
		r >= 'a' && r <= 'z' ||
		r >= '0' && r <= '9' {
		return true
	}

	switch r {
	case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
		return true
	default:
		return false
	}
}

func normalizeHTTPAuthMode(mode spec.MCPHTTPAuthMode) spec.MCPHTTPAuthMode {
	mode = spec.MCPHTTPAuthMode(strings.TrimSpace(string(mode)))
	if mode == "" {
		return spec.MCPHTTPAuthNone
	}
	return mode
}

func setupDeclaresClientCredentials(setup *spec.MCPServerSetup) bool {
	if setup == nil {
		return false
	}
	for i := range setup.Inputs {
		in := setup.Inputs[i]
		if in.Required && in.Kind == spec.MCPSetupKindOAuthClientCredentials && in.OAuthClientCredentials != nil {
			return true
		}
	}
	return false
}

func setupDeclaresAPIKeyHeader(setup *spec.MCPServerSetup) bool {
	if setup == nil {
		return false
	}
	for i := range setup.Inputs {
		in := setup.Inputs[i]
		if in.Required && in.Kind == spec.MCPSetupKindHTTPHeader && in.HTTPHeader != nil && in.HTTPHeader.Secret {
			return true
		}
	}
	return false
}

func isBundleSoftDeleted(b spec.MCPBundle) bool {
	return b.SoftDeletedAt != nil && !b.SoftDeletedAt.IsZero()
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
