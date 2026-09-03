package aggregate

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	mcpSpec "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/spec"
)

const (
	artifactServerIDPrefix  = "artifact-server:v1:"
	artifactCatalogIDPrefix = "artifact-catalog:v1:"
)

func RuntimeServerIDForArtifact(
	ref artifact.ArtifactRef,
) (mcpSpec.ServerID, error) {
	if err := ref.Validate(); err != nil {
		return "", err
	}
	raw := string(ref.RootID) + "\x00" + string(ref.ArtifactID)
	return mcpSpec.ServerID(
		artifactServerIDPrefix +
			base64.RawURLEncoding.EncodeToString([]byte(raw)),
	), nil
}

func ArtifactRefForRuntimeServerID(
	id mcpSpec.ServerID,
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
		RootID:     basespec.RootID(rootID),
		ArtifactID: basespec.ArtifactID(artifactID),
	}
	if err := ref.Validate(); err != nil {
		return artifact.ArtifactRef{}, err
	}
	return ref, nil
}

func RuntimeCatalogIDForCollection(
	ref collection.CollectionRef,
) (mcpSpec.CatalogID, error) {
	if err := ref.Validate(); err != nil {
		return "", err
	}
	raw := string(ref.RootID) + "\x00" + string(ref.CollectionID)
	return mcpSpec.CatalogID(
		artifactCatalogIDPrefix +
			base64.RawURLEncoding.EncodeToString([]byte(raw)),
	), nil
}

func CollectionRefForRuntimeCatalogID(
	id mcpSpec.CatalogID,
) (collection.CollectionRef, error) {
	if err := id.Validate(); err != nil {
		return collection.CollectionRef{}, err
	}
	raw, found := strings.CutPrefix(string(id), artifactCatalogIDPrefix)
	if !found {
		return collection.CollectionRef{}, fmt.Errorf(
			"%w: unsupported MCP runtime catalog ID",
			basespec.ErrInvalid,
		)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return collection.CollectionRef{}, fmt.Errorf(
			"%w: decode MCP runtime catalog ID: %w",
			basespec.ErrInvalid,
			err,
		)
	}
	rootID, collectionID, found := strings.Cut(string(decoded), "\x00")
	if !found {
		return collection.CollectionRef{}, fmt.Errorf(
			"%w: malformed MCP runtime catalog ID",
			basespec.ErrInvalid,
		)
	}
	ref := collection.CollectionRef{
		RootID:       basespec.RootID(rootID),
		CollectionID: basespec.CollectionID(collectionID),
	}
	if err := ref.Validate(); err != nil {
		return collection.CollectionRef{}, err
	}
	return ref, nil
}
