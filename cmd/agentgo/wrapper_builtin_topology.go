package main

import (
	"context"
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/system"
	builtinSchema "github.com/flexigpt/flexigpt-app/internal/builtin/schema"
)

func EnsureBuiltinArtifactTopology(
	ctx context.Context,
	components *system.Components,
	skills *SkillBundleWrapper,
	mcp *MCPWrapper,
) error {
	if components == nil ||
		skills == nil ||
		mcp == nil ||
		skills.builtInInstaller == nil ||
		mcp.builtInInstaller == nil {
		return errors.New("built-in topology dependencies are incomplete")
	}
	if err := builtinSchema.ValidateApplicationTopology(); err != nil {
		return err
	}

	bootstrap, err := builtinSchema.NewBootstrapRegistry(
		builtinSchema.BuiltinTopologyDeclaration(),
		components,
		components,
	)
	if err != nil {
		return err
	}
	if err := bootstrap.Register(skills.builtInInstaller); err != nil {
		return err
	}
	if err := bootstrap.Register(mcp.builtInInstaller); err != nil {
		return err
	}
	return bootstrap.Ensure(ctx)
}
