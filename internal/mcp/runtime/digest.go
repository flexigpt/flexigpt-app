package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

type discoverySnapshotDigestPayload struct {
	ServerID                  spec.MCPServerID                   `json:"serverID"`
	NegotiatedProtocolVersion string                             `json:"negotiatedProtocolVersion,omitempty"`
	ServerInfo                *spec.MCPImplementationInfo        `json:"serverInfo,omitempty"`
	ServerCapabilities        *spec.MCPServerCapabilitiesSummary `json:"serverCapabilities,omitempty"`
	Instructions              string                             `json:"instructions,omitempty"`
	Tools                     []spec.MCPToolCapability           `json:"tools,omitempty"`
	Resources                 []spec.MCPResourceRef              `json:"resources,omitempty"`
	ResourceTemplates         []spec.MCPResourceTemplateRef      `json:"resourceTemplates,omitempty"`
	Prompts                   []spec.MCPPromptRef                `json:"prompts,omitempty"`
}

func computeDiscoverySnapshotDigest(snap spec.MCPDiscoverySnapshot) string {
	raw, err := json.Marshal(discoverySnapshotDigestPayload{
		ServerID:                  snap.ServerID,
		NegotiatedProtocolVersion: snap.NegotiatedProtocolVersion,
		ServerInfo:                snap.ServerInfo,
		ServerCapabilities:        snap.ServerCapabilities,
		Instructions:              snap.Instructions,
		Tools:                     snap.Tools,
		Resources:                 snap.Resources,
		ResourceTemplates:         snap.ResourceTemplates,
		Prompts:                   snap.Prompts,
	})
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
