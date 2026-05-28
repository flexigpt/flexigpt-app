package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

var providerNameUnsafe = regexp.MustCompile(`[^A-Za-z0-9_]`)

const (
	maxProviderToolNameLen = 64
	maxServerPartLen       = 24
	minToolPartLen         = 8
)

func ProviderToolName(serverID spec.MCPServerID, toolName string) string {
	server := sanitizeName(string(serverID))
	tool := sanitizeName(toolName)

	sum := sha256.Sum256([]byte(string(serverID) + "\x00" + toolName))
	suffix := hex.EncodeToString(sum[:])[:8]

	if len(server) > maxServerPartLen {
		server = server[:maxServerPartLen]
	}

	maxTool := max(maxProviderToolNameLen-
		len("mcp__")-
		len(server)-
		len("__")-
		len("__")-
		len(suffix), minToolPartLen)
	if len(tool) > maxTool {
		tool = tool[:maxTool]
	}
	return "mcp__" + server + "__" + tool + "__" + suffix
}

func ChoiceID(serverID spec.MCPServerID, toolName string) string {
	sum := sha256.Sum256([]byte(string(serverID) + "\x00" + toolName))
	return "mcp-" + hex.EncodeToString(sum[:])[:16]
}

func sanitizeName(s string) string {
	s = strings.TrimSpace(s)
	s = providerNameUnsafe.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	if s == "" {
		return "x"
	}
	if s[0] >= '0' && s[0] <= '9' {
		s = "s_" + s
	}
	return s
}
