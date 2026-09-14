package main

import (
	"context"
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi"
)

func EnsureBuiltinArtifactTopology(
	ctx context.Context,
	topologyAPI installerapi.API,
	skills builtin.HydrationInstaller,
	mcp builtin.HydrationInstaller,
) error {
	if topologyAPI == nil ||
		skills == nil ||
		mcp == nil {
		return errors.New("built-in topology dependencies are incomplete")
	}
	if err := builtin.ValidateApplicationTopology(); err != nil {
		return err
	}

	bootstrap, err := builtin.NewBootstrapRegistry(
		builtin.BuiltinTopologyDeclaration(),
		topologyAPI,
		topologyAPI,
	)
	if err != nil {
		return err
	}
	if err := bootstrap.Register(skills); err != nil {
		return err
	}
	if err := bootstrap.Register(mcp); err != nil {
		return err
	}
	return bootstrap.Ensure(ctx)
}
