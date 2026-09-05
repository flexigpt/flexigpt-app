package main

import (
	"context"
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
)

func EnsureBuiltinArtifactTopology(
	ctx context.Context,
	store *artifactstore.API,
	skills *SkillStoreWrapper,
	mcp *MCPStoreWrapper,
) error {
	if store == nil ||
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
		store,
		store,
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
