package artifactsetup

import (
	"context"
	"errors"
	"fmt"
	"sort"

	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
)

type SkillBaselineEnsurer interface {
	EnsureSkillBaselinePlugin(
		ctx context.Context,
		rootID rootModel.RootID,
	) (plugin.PluginView, error)
}

type MCPBaselineEnsurer interface {
	EnsureMCPBaselinePlugin(
		ctx context.Context,
		rootID rootModel.RootID,
	) (plugin.PluginView, error)
}

type AgentBaselineEnsurer interface {
	EnsureAgentBaselinePlugin(
		ctx context.Context,
		rootID rootModel.RootID,
	) (plugin.PluginView, error)
}

// EnsureBuiltInTopology installs and reconciles all application-owned
// protected package installers as one generic installation bootstrap.
func EnsureBuiltInTopology(
	ctx context.Context,
	topo installFlow.API,
	installers ...installFlow.Installer,
) error {
	if ctx == nil {
		return errors.New("built-in topology context is nil")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if topo == nil || len(installers) == 0 {
		return errors.New("built-in topology dependencies are incomplete")
	}
	for index, installer := range installers {
		if installer == nil {
			return fmt.Errorf(
				"built-in installer %d is nil",
				index,
			)
		}
	}
	if err := topology.ValidateApplicationTopology(); err != nil {
		return err
	}

	bootstrap, err := installFlow.NewBootstrap(
		topology.BuiltinTopologyDeclaration(),
		topo,
		installers...,
	)
	if err != nil {
		return err
	}
	return bootstrap.Ensure(root.WithInstallerPrivilege(ctx))
}

// EnsureRootBaselines provisions mutable-root family baseline Plugins.
func EnsureRootBaselines(
	ctx context.Context,
	rootID rootModel.RootID,
	skills SkillBaselineEnsurer,
	mcp MCPBaselineEnsurer,
	agents AgentBaselineEnsurer,
) error {
	if skills == nil || mcp == nil || agents == nil {
		return errors.New("user Artifact baseline dependencies are incomplete")
	}
	if err := rootID.Validate(); err != nil {
		return err
	}

	var result error
	if _, err := skills.EnsureSkillBaselinePlugin(ctx, rootID); err != nil {
		result = errors.Join(
			result,
			fmt.Errorf("ensure Skill baseline Plugin: %w", err),
		)
	}
	if ctx.Err() != nil {
		return errors.Join(result, ctx.Err())
	}

	if _, err := mcp.EnsureMCPBaselinePlugin(ctx, rootID); err != nil {
		result = errors.Join(
			result,
			fmt.Errorf("ensure MCP baseline Plugin: %w", err),
		)
	}
	if ctx.Err() != nil {
		return errors.Join(result, ctx.Err())
	}

	if _, err := agents.EnsureAgentBaselinePlugin(ctx, rootID); err != nil {
		result = errors.Join(
			result,
			fmt.Errorf("ensure Agent baseline Plugin: %w", err),
		)
	}
	return result
}

// EnsureMutableRootBaselines provisions family baseline Plugins for each
// currently active non-protected Root.
func EnsureMutableRootBaselines(
	ctx context.Context,
	roots root.API,
	protection root.ProtectionAPI,
	skills SkillBaselineEnsurer,
	mcp MCPBaselineEnsurer,
	agents AgentBaselineEnsurer,
) error {
	if roots == nil || protection == nil {
		return errors.New("user Artifact Root lifecycle dependencies are incomplete")
	}

	values, err := roots.List(ctx)
	if err != nil {
		return err
	}
	sort.Slice(values, func(left, right int) bool {
		return values[left].ID < values[right].ID
	})

	var result error
	for _, value := range values {
		if protection.IsProtectedRoot(value.ID) {
			continue
		}
		if err := EnsureRootBaselines(ctx, value.ID, skills, mcp, agents); err != nil {
			if ctx.Err() != nil {
				return errors.Join(result, ctx.Err())
			}
			result = errors.Join(
				result,
				fmt.Errorf(
					"ensure Artifact baselines for Root %q: %w",
					value.ID,
					err,
				),
			)
		}
	}
	return result
}
