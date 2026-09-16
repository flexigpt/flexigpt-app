package aggregate

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	mcpServer "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/server"
)

const (
	artifactServerIDPrefix  = "artifact-server:v1:"
	artifactCatalogIDPrefix = "artifact-root:v1:"
)

func RuntimeServerIDForArtifact(
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

func ArtifactRefForRuntimeServerID(
	id mcpServer.ServerID,
) (artifact.ArtifactRef, error) {
	if err := id.Validate(); err != nil {
		return artifact.ArtifactRef{}, err
	}
	raw, found := strings.CutPrefix(string(id), artifactServerIDPrefix)
	if !found {
		return artifact.ArtifactRef{}, fmt.Errorf(
			"%w: unsupported MCP runtime server ID",
			basespec.ErrInvalid,
		)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return artifact.ArtifactRef{}, fmt.Errorf(
			"%w: decode MCP runtime server ID: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	rootID, artifactID, found := strings.Cut(string(decoded), "\x00")
	if !found {
		return artifact.ArtifactRef{}, fmt.Errorf(
			"%w: malformed MCP runtime server ID",
			basespec.ErrInvalid,
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

func RootIDForRuntimeCatalogID(
	id mcpServer.CatalogID,
) (root.RootID, error) {
	if err := id.Validate(); err != nil {
		return "", err
	}
	raw, found := strings.CutPrefix(string(id), artifactCatalogIDPrefix)
	if !found {
		return "", fmt.Errorf(
			"%w: unsupported MCP runtime catalog ID",
			basespec.ErrInvalid,
		)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return "", fmt.Errorf(
			"%w: decode MCP runtime catalog ID: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	rootID := root.RootID(decoded)
	if err := rootID.Validate(); err != nil {
		return "", err
	}

	return rootID, nil
}
