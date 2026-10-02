package aggregate

import (
	"encoding/base64"
	"fmt"
	"strings"

	artifact "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	root "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/server"
)

const (
	artifactServerIDPrefix  = "artifact-server:v1:"
	artifactCatalogIDPrefix = "artifact-root:v1:"
)

func runtimeServerIDForArtifact(
	ref artifact.ArtifactRef,
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

func artifactRefForRuntimeServerID(
	id mcpServer.ServerID,
) (artifact.ArtifactRef, error) {
	if err := id.Validate(); err != nil {
		return artifact.ArtifactRef{}, err
	}
	raw, found := strings.CutPrefix(string(id), artifactServerIDPrefix)
	if !found {
		return artifact.ArtifactRef{}, fmt.Errorf(
			"%w: unsupported MCP runtime server ID",
			spec.ErrInvalid,
		)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return artifact.ArtifactRef{}, fmt.Errorf(
			"%w: decode MCP runtime server ID: %w",
			spec.ErrInvalid,
			err,
		)
	}
	rootID, artifactID, found := strings.Cut(string(decoded), "\x00")
	if !found {
		return artifact.ArtifactRef{}, fmt.Errorf(
			"%w: malformed MCP runtime server ID",
			spec.ErrInvalid,
		)
	}
	ref := artifact.ArtifactRef{
		RootID:     root.RootID(rootID),
		ArtifactID: artifact.ArtifactID(artifactID),
	}
	if err := ref.Validate(); err != nil {
		return artifact.ArtifactRef{}, err
	}
	return ref, nil
}
