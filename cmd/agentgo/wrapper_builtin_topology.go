package main

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/collection"
)

type skillBaselineEnsurer interface {
	EnsureSkillBaselineCollection(
		ctx context.Context,
		rootID rootModel.RootID,
	) (collection.CollectionView, error)
}

type mcpBaselineEnsurer interface {
	EnsureMCPBaselineCollection(
		ctx context.Context,
		rootID rootModel.RootID,
	) (collection.CollectionView, error)
}

type agentBaselineEnsurer interface {
	EnsureAgentBaselineCollection(
		ctx context.Context,
		rootID rootModel.RootID,
	) (collection.CollectionView, error)
}

func ensureBuiltinArtifactTopology(
	ctx context.Context,
	topologyAPI installFlow.API,
	tools builtin.HydrationInstaller,
	models builtin.HydrationInstaller,
	skills builtin.HydrationInstaller,
	mcp builtin.HydrationInstaller,
	agents builtin.HydrationInstaller,
) error {
	if topologyAPI == nil ||
		tools == nil ||
		models == nil ||
		skills == nil ||
		mcp == nil ||
		agents == nil {
		return errors.New("built-in topology dependencies are incomplete")
	}
	if err := documentTopology.ValidateApplicationTopology(); err != nil {
		return err
	}

	bootstrap, err := builtin.NewDefaultBootstrapRegistry(
		topologyAPI,
		topologyAPI,
	)
	if err != nil {
		return err
	}
	if err := bootstrap.Register(tools); err != nil {
		return err
	}
	if err := bootstrap.Register(models); err != nil {
		return err
	}
	if err := bootstrap.Register(skills); err != nil {
		return err
	}
	if err := bootstrap.Register(mcp); err != nil {
		return err
	}
	if err := bootstrap.Register(agents); err != nil {
		return err
	}
	return bootstrap.Ensure(ctx)
}

func ensureUserArtifactBaselineCollectionsForRoot(
	ctx context.Context,
	rootID rootModel.RootID,
	skills skillBaselineEnsurer,
	mcp mcpBaselineEnsurer,
	agents agentBaselineEnsurer,
) error {
	if skills == nil || mcp == nil || agents == nil {
		return errors.New("user Artifact baseline dependencies are incomplete")
	}
	var result error
	if _, err := skills.EnsureSkillBaselineCollection(ctx, rootID); err != nil {
		result = errors.Join(result, fmt.Errorf("ensure Skill baseline Collection: %w", err))
	}
	if ctx.Err() != nil {
		return errors.Join(result, ctx.Err())
	}
	if _, err := mcp.EnsureMCPBaselineCollection(ctx, rootID); err != nil {
		result = errors.Join(result, fmt.Errorf("ensure MCP baseline Collection: %w", err))
	}
	if ctx.Err() != nil {
		return errors.Join(result, ctx.Err())
	}
	if _, err := agents.EnsureAgentBaselineCollection(ctx, rootID); err != nil {
		result = errors.Join(result, fmt.Errorf("ensure Agent baseline Collection: %w", err))
	}
	return result
}

func ensureUserArtifactBaselineCollections(
	ctx context.Context,
	roots root.API,
	protection root.ProtectionAPI,
	skills skillBaselineEnsurer,
	mcp mcpBaselineEnsurer,
	agents agentBaselineEnsurer,
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
		if err := ensureUserArtifactBaselineCollectionsForRoot(
			ctx,
			value.ID,
			skills,
			mcp,
			agents,
		); err != nil {
			if ctx.Err() != nil {
				return errors.Join(result, ctx.Err())
			}
			result = errors.Join(
				result,
				fmt.Errorf(
					"ensure user Artifact baselines for Root %q: %w",
					value.ID,
					err,
				),
			)
		}
	}
	return result
}

// artifactTargetMappers returns ArtifactRef-to-mapped-target adapters used by
// Artifact consumers. Tool and Model mapping belong to their aggregates
// because they validate source-backed enablement and runtime availability
// before mapping.
func artifactTargetMappers(
	tools *ToolAggregateWrapper,
	models *ModelAggregateWrapper,
) (map[declaration.Type]resolve.ArtifactTargetMapper, error) {
	if tools == nil || models == nil {
		return nil, errors.New("artifact target mapper aggregates are incomplete")
	}
	toolMappers, err := tools.targetMappers()
	if err != nil {
		return nil, err
	}
	modelMappers, err := models.targetMappers()
	if err != nil {
		return nil, err
	}
	maps.Copy(toolMappers, modelMappers)
	return toolMappers, nil
}
