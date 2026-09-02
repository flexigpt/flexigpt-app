package main

import (
	"context"
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/system"
)

func EnsureBuiltinArtifactTopology(
	ctx context.Context,
	components *system.Components,
	skills *SkillStoreWrapper,
	mcp *MCPWrapper,
) error {
	if components == nil ||
		skills == nil ||
		mcp == nil ||
		skills.builtInInstaller == nil ||
		mcp.builtInInstaller == nil {
		return errors.New("built-in topology dependencies are incomplete")
	}
	if err := artifactbuiltin.ValidateApplicationTopology(); err != nil {
		return err
	}

	bootstrap, err := artifactbuiltin.NewBootstrapRegistry(
		artifactbuiltin.BuiltinTopologyDeclaration(),
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
