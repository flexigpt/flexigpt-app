package artifact

import (
	"context"
	"fmt"
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

// ResolveRuntimePackage applies only Agent Skill binding rules. Source
// generation verification, digest verification, snapshot reuse, safe close,
// and native path resolution belong to source.ResolveVerifiedLocalPath.
func ResolveRuntimePackage(
	ctx context.Context,
	runtime source.Runtime,
	value source.Source,
	locator basespec.Locator,
	subresource basespec.SubresourceLocator,
	expectedGeneration string,
	expectedContentDigest cryptoutil.Digest,
) (string, error) {
	if subresource != "" {
		return "", fmt.Errorf(
			"%w: Agent Skill bindings cannot target a subresource",
			basespec.ErrUnsupported,
		)
	}
	if basespec.Locator(path.Base(string(locator))) != DefinitionFileName {
		return "", fmt.Errorf(
			"%w: Agent Skill locator %q is not %q",
			basespec.ErrInvalid,
			locator,
			DefinitionFileName,
		)
	}

	packageLocator := basespec.Locator(path.Dir(string(locator)))
	if packageLocator == "." {
		return "", fmt.Errorf(
			"%w: Agent Skill package cannot be the Source root",
			basespec.ErrInvalid,
		)
	}

	return source.ResolveVerifiedLocalPath(
		ctx,
		runtime,
		value,
		locator,
		packageLocator,
		expectedGeneration,
		expectedContentDigest,
		basespec.MaxCandidateBytes,
	)
}
