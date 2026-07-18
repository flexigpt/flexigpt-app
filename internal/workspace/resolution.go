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

func (s *Service) buildWorkspaceDependencyGraph(
	ctx context.Context,
	catalog Catalog,
	root CatalogResource,
) (artifactstoreSpec.DependencyGraph, error) {
	graph := artifactstoreSpec.DependencyGraph{
		RootRecordID: root.Record.RecordID,
		Nodes:        map[artifactstoreSpec.Digest]artifactstoreSpec.CanonicalDefinition{},
		Edges:        map[artifactstoreSpec.Digest][]artifactstoreSpec.DependencyExplanation{},
	}
	visiting := map[artifactstoreSpec.Digest]bool{}

	var visit func(artifactstoreSpec.CanonicalDefinition) error
	visit = func(definition artifactstoreSpec.CanonicalDefinition) error {
		if visiting[definition.Digest] {
			graph.Diagnostics = append(graph.Diagnostics, artifactstoreSpec.Diagnostic{
				Severity: artifactstoreSpec.DiagnosticSeverityError,
				Code:     "workspace.dependency.cycle",
				Message:  fmt.Sprintf("dependency cycle detected at definition %q", definition.Digest),
			})
			return nil
		}
		if _, exists := graph.Nodes[definition.Digest]; exists {
			return nil
		}
		graph.Nodes[definition.Digest] = definition
		visiting[definition.Digest] = true
		defer delete(visiting, definition.Digest)

		for _, selector := range definition.DependencySelectors {
			candidates, err := s.store.FindCandidates(ctx, catalog.Workspace.Root.RootID, selector)
			if err != nil {
				return err
			}
			selected, diagnostics := selectWorkspaceCandidate(catalog.Workspace, candidates)
			explanation := artifactstoreSpec.DependencyExplanation{
				Selector:    selector,
				Candidates:  candidates,
				Diagnostics: append([]artifactstoreSpec.Diagnostic(nil), diagnostics...),
			}
			graph.Edges[definition.Digest] = append(
				graph.Edges[definition.Digest],
				explanation,
			)
			graph.Diagnostics = append(graph.Diagnostics, diagnostics...)
			if selected != nil {
				if err := visit(selected.Definition); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := visit(root.Definition); err != nil {
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

var _ artifactstoreSpec.ArtifactVersionMatcher = exactVersionMatcher{}
