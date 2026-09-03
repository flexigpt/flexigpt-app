package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	mcpPolicy "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/policy"
)

var (
	ErrInvalid             = errors.New("invalid MCP runtime specification")
	ErrClosed              = errors.New("MCP runtime is closed")
	ErrConflict            = errors.New("MCP runtime conflict")
	ErrReferenceUnresolved = errors.New("MCP runtime reference unresolved")
	ErrDigestMismatch      = errors.New("MCP runtime digest mismatch")
)

const (
	MaxDisplayNameBytes    = 512
	MaxDescriptionBytes    = 1 << 20
	MaxURIBytes            = 16 << 10
	MaxLocatorBytes        = 16 << 10
	MaxFingerprintBytes    = 512
	MaxKindBytes           = 256
	MaxDiscoveryCandidates = 4096

	DefaultConnectionTimeoutMS = 30_000
	MaxConnectionTimeoutMS     = 10 * 60 * 1_000
)

type (
	ServerID  string
	CatalogID string
	Digest    string
)

func (id ServerID) Validate() error {
	return validateOpaqueID("MCP server ID", string(id))
}

func (id CatalogID) Validate() error {
	return validateOpaqueID("MCP catalog ID", string(id))
}

func validateOpaqueID(subject, value string) error {
	if value == "" ||
		strings.TrimSpace(value) != value ||
		len(value) > MaxURIBytes ||
		strings.ContainsRune(value, 0) {
		return fmt.Errorf("%w: %s is invalid", ErrInvalid, subject)
	}
	return nil
}

func ValidateRequiredText(subject, value string, maximum int) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%w: %s is required", ErrInvalid, subject)
	}
	return ValidateOptionalText(subject, value, maximum)
}

func ValidateOptionalText(subject, value string, maximum int) error {
	if !utf8.ValidString(value) || len(value) > maximum {
		return fmt.Errorf("%w: %s is invalid", ErrInvalid, subject)
	}
	return nil
}

func DigestBytes(value []byte) Digest {
	sum := sha256.Sum256(value)
	return Digest("sha256:" + hex.EncodeToString(sum[:]))
}

func ValidateDigest(value Digest) error {
	raw, found := strings.CutPrefix(string(value), "sha256:")
	if !found || len(raw) != sha256.Size*2 {
		return fmt.Errorf("%w: invalid digest", ErrInvalid)
	}
	if _, err := hex.DecodeString(raw); err != nil {
		return fmt.Errorf("%w: invalid digest: %w", ErrInvalid, err)
	}
	return nil
}

type ClientInfo struct {
	Name    string
	Version string
}

func (value ClientInfo) Validate() error {
	if err := ValidateRequiredText(
		"MCP client name",
		value.Name,
		MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	return ValidateRequiredText(
		"MCP client version",
		value.Version,
		MaxDisplayNameBytes,
	)
}

type MCPHTTPAuthMode string

const (
	MCPHTTPAuthNone              MCPHTTPAuthMode = "none"
	MCPHTTPAuthAPIKey            MCPHTTPAuthMode = "apiKey"
	MCPHTTPAuthOAuth             MCPHTTPAuthMode = "oauth"
	MCPHTTPAuthClientCredentials MCPHTTPAuthMode = "clientCredentials"
)

type MCPTransportType string

const (
	MCPTransportStreamableHTTP MCPTransportType = "streamableHttp"
	MCPTransportStdio          MCPTransportType = "stdio"
)

type MCPRuntimeStdioConfig struct {
	Command          string            `json:"command"`
	Args             []string          `json:"args,omitempty"`
	Env              map[string]string `json:"env,omitempty"`
	StartupTimeoutMS int               `json:"startupTimeoutMS,omitempty"`
}

type MCPRuntimeStreamableHTTPConfig struct {
	URL       string          `json:"url"`
	TimeoutMS int             `json:"timeoutMS,omitempty"`
	AuthMode  MCPHTTPAuthMode `json:"authMode"`

	Headers map[string]string `json:"headers,omitempty"`

	ClientCredentialRef         string `json:"clientCredentialRef,omitempty"`
	ClientIDMetadataDocumentURL string `json:"clientIDMetadataDocumentURL,omitempty"`
}

type RuntimeConfig struct {
	Server  ServerID
	Catalog CatalogID

	LogicalName string
	DisplayName string

	Transport                 MCPTransportType
	Stdio                     *MCPRuntimeStdioConfig
	StreamableHTTP            *MCPRuntimeStreamableHTTPConfig
	OAuthClientSecretRequired bool

	Policy mcpPolicy.MCPPolicy

	SensitiveValues []string
}

func (config RuntimeConfig) Validate() error {
	if err := config.Server.Validate(); err != nil {
		return err
	}
	if err := config.Catalog.Validate(); err != nil {
		return err
	}
	if err := ValidateRequiredText(
		"MCP logical name",
		config.LogicalName,
		MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := ValidateOptionalText(
		"MCP display name",
		config.DisplayName,
		MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	if err := validatePolicy(config); err != nil {
		return err
	}

	switch config.Transport {
	case MCPTransportStdio:
		if config.Stdio == nil || config.StreamableHTTP != nil {
			return fmt.Errorf(
				"%w: stdio transport configuration is incomplete",
				ErrInvalid,
			)
		}
		return validateStdio(config.Stdio)

	case MCPTransportStreamableHTTP:
		if config.StreamableHTTP == nil || config.Stdio != nil {
			return fmt.Errorf(
				"%w: HTTP transport configuration is incomplete",
				ErrInvalid,
			)
		}
		return validateHTTP(config.StreamableHTTP)

	default:
		return fmt.Errorf(
			"%w: unsupported MCP transport %q",
			ErrInvalid,
			config.Transport,
		)
	}
}

func validatePolicy(config RuntimeConfig) error {
	if err := mcpPolicy.ValidateMCPPolicy(config.Policy); err != nil {
		return fmt.Errorf("%w: invalid MCP runtime policy: %w", ErrInvalid, err)
	}
	return nil
}

func validateStdio(config *MCPRuntimeStdioConfig) error {
	if config == nil {
		return fmt.Errorf("%w: nil stdio config", ErrInvalid)
	}
	if err := ValidateRequiredText(
		"MCP stdio command",
		config.Command,
		MaxLocatorBytes,
	); err != nil {
		return err
	}
	if shellCommand(config.Command) {
		return fmt.Errorf(
			"%w: MCP stdio command must execute the server directly",
			ErrInvalid,
		)
	}
	if err := validateTimeout(config.StartupTimeoutMS); err != nil {
		return err
	}
	for name := range config.Env {
		if err := validateEnvironmentName(name); err != nil {
			return err
		}
	}
	return nil
}

func validateHTTP(config *MCPRuntimeStreamableHTTPConfig) error {
	if config == nil {
		return fmt.Errorf("%w: nil HTTP config", ErrInvalid)
	}
	if err := validateRuntimeURL(config.URL); err != nil {
		return err
	}
	if err := validateTimeout(config.TimeoutMS); err != nil {
		return err
	}
	switch config.AuthMode {
	case MCPHTTPAuthNone,
		MCPHTTPAuthAPIKey,
		MCPHTTPAuthOAuth,
		MCPHTTPAuthClientCredentials:
	default:
		return fmt.Errorf(
			"%w: unsupported MCP HTTP auth mode %q",
			ErrInvalid,
			config.AuthMode,
		)
	}
	for name, value := range config.Headers {
		if err := validateHeaderName(name); err != nil {
			return err
		}
		if strings.ContainsAny(value, "\r\n\x00") {
			return fmt.Errorf(
				"%w: MCP HTTP header %q contains CR, LF, or NUL",
				ErrInvalid,
				name,
			)
		}
	}
	if config.AuthMode == MCPHTTPAuthClientCredentials &&
		strings.TrimSpace(config.ClientCredentialRef) == "" {
		return fmt.Errorf(
			"%w: client credentials auth requires a credential reference",
			ErrReferenceUnresolved,
		)
	}
	return nil
}

func validateTimeout(value int) error {
	if value < 0 || value > MaxConnectionTimeoutMS {
		return fmt.Errorf(
			"%w: MCP timeout must be between zero and %d milliseconds",
			ErrInvalid,
			MaxConnectionTimeoutMS,
		)
	}
	return nil
}

func validateRuntimeURL(raw string) error {
	if strings.TrimSpace(raw) == "" ||
		strings.TrimSpace(raw) != raw ||
		len(raw) > MaxURIBytes {
		return fmt.Errorf("%w: invalid MCP HTTP URL", ErrInvalid)
	}
	value, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%w: invalid MCP HTTP URL: %w", ErrInvalid, err)
	}
	if value.User != nil ||
		value.Fragment != "" ||
		value.Host == "" {
		return fmt.Errorf(
			"%w: MCP HTTP URL has disallowed components",
			ErrInvalid,
		)
	}
	switch value.Scheme {
	case "https":
		return nil
	case "http":
		if isLoopback(value.Hostname()) {
			return nil
		}
	}
	return fmt.Errorf(
		"%w: MCP HTTP URL must use HTTPS or loopback HTTP",
		ErrInvalid,
	)
}

func validateHeaderName(name string) error {
	if name == "" || strings.TrimSpace(name) != name {
		return fmt.Errorf("%w: invalid MCP HTTP header name", ErrInvalid)
	}
	for _, character := range name {
		if regexp.MustCompile(`^[A-Za-z0-9!#$%&'*+\-.^_` + "`" + `|~]$`).
			MatchString(string(character)) {
			continue
		}
		return fmt.Errorf(
			"%w: invalid MCP HTTP header character %q",
			ErrInvalid,
			character,
		)
	}
	return nil
}

func validateEnvironmentName(name string) error {
	if name == "" ||
		strings.TrimSpace(name) != name ||
		strings.ContainsAny(name, "=\x00") {
		return fmt.Errorf("%w: invalid MCP environment name", ErrInvalid)
	}
	return nil
}

func shellCommand(command string) bool {
	command = strings.ReplaceAll(command, "\\", "/")
	parts := strings.Split(command, "/")
	base := strings.ToLower(parts[len(parts)-1])
	switch base {
	case "bash", "sh", "zsh", "cmd", "cmd.exe",
		"powershell", "powershell.exe", "pwsh", "pwsh.exe":
		return true
	default:
		return false
	}
}

func isLoopback(host string) bool {
	if strings.EqualFold(strings.TrimSpace(host), "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

type ResolvedServer struct {
	Server  ServerID
	Catalog CatalogID
	Version Digest
	Config  RuntimeConfig
}

func (value ResolvedServer) Validate() error {
	if err := value.Server.Validate(); err != nil {
		return err
	}
	if err := value.Catalog.Validate(); err != nil {
		return err
	}
	if value.Config.Server != value.Server ||
		value.Config.Catalog != value.Catalog {
		return fmt.Errorf(
			"%w: resolved MCP identity differs from runtime config",
			ErrInvalid,
		)
	}
	if err := ValidateDigest(value.Version); err != nil {
		return err
	}
	return value.Config.Validate()
}

type ServerSource interface {
	ResolveServer(
		ctx context.Context,
		server ServerID,
	) (ResolvedServer, error)
}
