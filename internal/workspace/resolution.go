package workspace

import (
	"context"
	"fmt"
	"strings"

	artifactstoreSpec "github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
)

type exactVersionMatcher struct {
	kind artifactstoreSpec.ArtifactKind
}

func (m exactVersionMatcher) Kind() artifactstoreSpec.ArtifactKind {
	return m.kind
}

func (exactVersionMatcher) MatchesVersionConstraint(
	_ context.Context,
	constraint string,
	version artifactstoreSpec.LogicalVersion,
) (bool, error) {
	constraint = strings.TrimSpace(constraint)
	if after, ok := strings.CutPrefix(constraint, "="); ok {
		constraint = strings.TrimSpace(after)
	}
	return constraint == string(version), nil
}

type workspaceDependencyResolver struct{}

func (workspaceDependencyResolver) RootKind() artifactstoreSpec.RootKind {
	return RootKind
}

func (workspaceDependencyResolver) ResolveDependency(
	_ context.Context,
	root artifactstoreSpec.ArtifactRoot,
	attachments []artifactstoreSpec.RootSourceAttachment,
	_ artifactstoreSpec.ArtifactSelector,
	candidates []artifactstoreSpec.DependencyCandidate,
) (*artifactstoreSpec.DependencyCandidate, []artifactstoreSpec.Diagnostic) {
	return selectWorkspaceCandidate(
		Workspace{Root: root, Attachments: attachments},
		candidates,
	)
}

func (s *Service) buildWorkspaceDependencyGraph(
	ctx context.Context,
	catalog Catalog,
	root CatalogResource,
) (artifactstoreSpec.DependencyGraph, error) {
	graph, err := s.store.BuildDependencyGraph(ctx, root.Record.RecordID)
	if err != nil {
		return artifactstoreSpec.DependencyGraph{}, err
	}
	if err := s.ensureCatalogGeneration(ctx, catalog.Generation); err != nil {
		return artifactstoreSpec.DependencyGraph{}, err
	}
	return graph, nil
}

func selectWorkspaceCandidate(
	workspace Workspace,
	candidates []artifactstoreSpec.DependencyCandidate,
) (*artifactstoreSpec.DependencyCandidate, []artifactstoreSpec.Diagnostic) {
	if len(candidates) == 0 {
		return nil, workspaceDiagnostics(
			"workspace.dependency.missing",
			"no Workspace catalog candidate matches the selector",
		)
	}

	priorities := make(map[artifactstoreSpec.SourceID]int, len(workspace.Attachments))
	for _, attachment := range workspace.Attachments {
		if attachment.Enabled {
			priorities[attachment.SourceID] = attachment.Priority
		}
	}
	highestSet := false
	highest := 0
	top := make([]artifactstoreSpec.DependencyCandidate, 0, 1)
	for _, candidate := range candidates {
		priority, attached := priorities[candidate.Resource.SourceID]
		if !attached {
			continue
		}
		if !highestSet || priority > highest {
			highestSet = true
			highest = priority
			top = top[:0]
			top = append(top, candidate)
			continue
		}
		if priority == highest {
			top = append(top, candidate)
		}
	}
	if len(top) != 1 {
		return nil, workspaceDiagnostics(
			"workspace.dependency.ambiguous",
			fmt.Sprintf(
				"%d Workspace candidates tie at the highest attachment priority",
				len(top),
			),
		)
	}
	selected := top[0]
	return &selected, nil
}

var (
	_ artifactstoreSpec.ArtifactVersionMatcher = exactVersionMatcher{}
	_ artifactstoreSpec.DependencyResolver     = workspaceDependencyResolver{}
)
