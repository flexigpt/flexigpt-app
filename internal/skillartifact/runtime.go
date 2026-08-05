package skillartifact

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

// ResolveRuntimePackage converts a verified Skill source locator into the
// native package directory required by the Agent Skills filesystem provider.
//
// Source adapters retain ownership of native path resolution, containment, and
// platform-specific behavior. Agentskills-go retains ownership of SKILL.md
// parsing, resource access, script policy, sandboxing, and execution.
func ResolveRuntimePackage(
	ctx context.Context,
	runtime source.Runtime,
	value source.Source,
	locator basespec.Locator,
	subresource basespec.SubresourceLocator,
	expectedGeneration string,
	expectedContentDigest cryptoutil.Digest,
) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf(
			"%w: Skill runtime package context is nil",
			basespec.ErrInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := basespec.ValidateLocator(locator, false); err != nil {
		return "", err
	}
	if err := basespec.ValidateSubresourceLocator(subresource); err != nil {
		return "", err
	}
	if subresource != "" {
		return "", fmt.Errorf(
			"%w: Agent Skill bindings cannot target a subresource",
			basespec.ErrUnsupported,
		)
	}
	if !strings.EqualFold(
		path.Base(string(locator)),
		DefinitionFileName,
	) {
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

	localPaths, supported := runtime.(source.LocalPathRuntime)
	if !supported || !localPaths.SupportsLocalPath(value.Kind) {
		return "", fmt.Errorf(
			"%w: Source kind %q has no trusted native package path",
			basespec.ErrUnsupported,
			value.Kind,
		)
	}

	// Verify the exact catalogued entry before requesting a native package
	// path. Source adapters remain responsible for containment and path
	// resolution, while the artifact boundary verifies the generation and
	// bytes used to derive this Skill definition.
	if err := source.VerifySnapshotContentDigest(
		ctx,
		runtime,
		value,
		locator,
		expectedGeneration,
		expectedContentDigest,
		basespec.MaxCandidateBytes,
	); err != nil {
		return "", err
	}

	location, err := localPaths.ResolveLocalPath(
		ctx,
		value,
		packageLocator,
	)
	if err != nil {
		return "", err
	}
	return location, nil
}
