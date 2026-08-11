package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/mcp/spec"
)

type discoverySnapshotDigestPayload struct {
	Server                    artifact.ArtifactRef               `json:"server"`
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
		Server:                    snap.Server,
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
