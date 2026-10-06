package inferenceadapter

import (
	"encoding/base64"
	"fmt"
	"strings"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/server"
)

const (
	artifactServerIDPrefix  = "artifact-server:v1:"
	artifactCatalogIDPrefix = "artifact-root:v1:"
)

// ServerIDForArtifact preserves the existing opaque runtime identity format.
func ServerIDForArtifact(
	ref artifactModel.ArtifactRef,
) (mcpServer.ServerID, error) {
	if err := ref.Validate(); err != nil {
		return "", err
	}
	raw := string(ref.RootID) + "\x00" + string(ref.ArtifactID)
	return mcpServer.ServerID(
		artifactServerIDPrefix +
			base64.RawURLEncoding.EncodeToString([]byte(raw)),
	), nil
}

func ArtifactRefForServerID(
	id mcpServer.ServerID,
) (artifactModel.ArtifactRef, error) {
	if err := id.Validate(); err != nil {
		return artifactModel.ArtifactRef{}, err
	}
	raw, found := strings.CutPrefix(string(id), artifactServerIDPrefix)
	if !found {
		return artifactModel.ArtifactRef{}, fmt.Errorf(
			"%w: unsupported MCP runtime server ID",
			spec.ErrInvalid,
		)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return artifactModel.ArtifactRef{}, fmt.Errorf(
			"%w: decode MCP runtime server ID: %w",
			spec.ErrInvalid,
			err,
		)
	}
	rootID, artifactID, found := strings.Cut(string(decoded), "\x00")
	if !found {
		return artifactModel.ArtifactRef{}, fmt.Errorf(
			"%w: malformed MCP runtime server ID",
			spec.ErrInvalid,
		)
	}

	ref := artifactModel.ArtifactRef{
		RootID:     rootModel.RootID(rootID),
		ArtifactID: artifactModel.ArtifactID(artifactID),
	}
	if err := ref.Validate(); err != nil {
		return artifactModel.ArtifactRef{}, err
	}
	return ref, nil
}

func runtimeCatalogIDForRoot(
	rootID rootModel.RootID,
) (mcpServer.CatalogID, error) {
	if err := rootID.Validate(); err != nil {
		return "", err
	}
	return mcpServer.CatalogID(
		artifactCatalogIDPrefix +
			base64.RawURLEncoding.EncodeToString([]byte(rootID)),
	), nil
}
