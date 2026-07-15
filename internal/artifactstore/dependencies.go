package artifactstore

import (
	"context"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
)

func (s *Store) GetDependencies(ctx context.Context, recordID spec.RecordID) ([]spec.ArtifactSelector, error) {
	if err := s.ensureOpen(); err != nil {
		return nil, err
	}
	record, err := s.GetRecord(ctx, recordID)
	if err != nil {
		return nil, err
	}
	if record.LastResolvedDefinitionDigest == nil {
		return nil, fmt.Errorf("%w: record has no resolved definition", spec.ErrConflict)
	}
	definition, err := s.GetDefinitionByDigest(ctx, *record.LastResolvedDefinitionDigest)
	if err != nil {
		return nil, err
	}
	return append([]spec.ArtifactSelector(nil), definition.DependencySelectors...), nil
}

func (s *Store) BuildDependencyGraph(ctx context.Context, recordID spec.RecordID) (spec.DependencyGraph, error) {
	if err := s.ensureOpen(); err != nil {
		return spec.DependencyGraph{}, err
	}
	record, err := s.GetRecord(ctx, recordID)
	if err != nil {
		return spec.DependencyGraph{}, err
	}
	if record.LastResolvedDefinitionDigest == nil {
		return spec.DependencyGraph{}, fmt.Errorf("%w: record has no resolved definition", spec.ErrConflict)
	}
	graph := spec.DependencyGraph{
		RootRecordID: recordID,
		Nodes:        map[spec.Digest]spec.CanonicalDefinition{},
		Edges:        map[spec.Digest][]spec.DependencyExplanation{},
	}
	visiting := map[spec.Digest]bool{}
	var visit func(spec.Digest) error
	visit = func(digest spec.Digest) error {
		if visiting[digest] {
			graph.Diagnostics = append(
				graph.Diagnostics,
				spec.Diagnostic{
					Severity: spec.DiagnosticSeverityError,
					Code:     "artifactstore.dependency.cycle",
					Message:  fmt.Sprintf("dependency cycle detected at definition %q", digest),
				},
			)
			return nil
		}
		if _, ok := graph.Nodes[digest]; ok {
			return nil
		}
		definition, err := s.GetDefinitionByDigest(ctx, digest)
		if err != nil {
			return err
		}
		graph.Nodes[digest] = definition
		visiting[digest] = true
		defer delete(visiting, digest)
		for _, selector := range definition.DependencySelectors {
			explanation, err := s.ExplainDependencyResolution(ctx, record.RootID, selector)
			if err != nil {
				return err
			}
			graph.Edges[digest] = append(graph.Edges[digest], explanation)
			graph.Diagnostics = append(graph.Diagnostics, explanation.Diagnostics...)
			if len(explanation.Candidates) == 1 {
				if err := visit(explanation.Candidates[0].Definition.Digest); err != nil {
					return err
				}
			}
		}
		return nil
	}
	err = visit(*record.LastResolvedDefinitionDigest)
	return graph, err
}

func (s *Store) ExplainDependencyResolution(
	ctx context.Context,
	rootID spec.RootID,
	selector spec.ArtifactSelector,
) (spec.DependencyExplanation, error) {
	if err := s.ensureOpen(); err != nil {
		return spec.DependencyExplanation{}, err
	}
	candidates, err := s.FindCandidates(ctx, rootID, selector)
	if err != nil {
		return spec.DependencyExplanation{}, err
	}
	explanation := spec.DependencyExplanation{Selector: selector, Candidates: candidates}
	switch len(candidates) {
	case 0:
		explanation.Diagnostics = []spec.Diagnostic{
			{
				Severity: spec.DiagnosticSeverityError,
				Code:     "artifactstore.dependency.missing",
				Message:  "no catalog candidate matches dependency selector",
			},
		}
	case 1:
	default:
		explanation.Diagnostics = []spec.Diagnostic{
			{
				Severity: spec.DiagnosticSeverityError,
				Code:     "artifactstore.dependency.ambiguous",
				Message:  "multiple catalog candidates match dependency selector",
			},
		}
	}
	return explanation, nil
}

func (s *Store) FindCandidates(
	ctx context.Context,
	rootID spec.RootID,
	selector spec.ArtifactSelector,
) ([]spec.DependencyCandidate, error) {
	if err := s.ensureOpen(); err != nil {
		return nil, err
	}
	if _, err := s.repository.GetRoot(ctx, rootID, false); err != nil {
		return nil, err
	}
	resources, err := s.repository.ListCatalogResourcesForRoot(ctx, rootID)
	if err != nil {
		return nil, err
	}
	candidates := []spec.DependencyCandidate{}
	for _, resource := range resources {
		if resource.State != spec.CatalogStateValid || resource.CurrentDefinitionDigest == nil ||
			resource.Kind != selector.Kind {
			continue
		}
		if selector.LogicalName != "" && resource.LogicalName != selector.LogicalName {
			continue
		}
		definition, err := s.GetDefinitionByDigest(ctx, *resource.CurrentDefinitionDigest)
		if err != nil {
			return nil, err
		}
		if !selectorLabelsMatch(selector.Labels, definition.Labels) {
			continue
		}
		if selector.VersionConstraint != "" {
			matches, err := s.matchesVersionConstraint(
				ctx,
				resource,
				selector.VersionConstraint,
				definition.LogicalVersion,
			)
			if err != nil {
				return nil, err
			}
			if !matches {
				continue
			}
		}
		candidates = append(candidates, spec.DependencyCandidate{Resource: resource, Definition: definition})
	}
	sort.Slice(candidates, func(left, right int) bool {
		l := candidates[left].Resource
		r := candidates[right].Resource
		if l.SourceID != r.SourceID {
			return l.SourceID < r.SourceID
		}
		if l.Locator != r.Locator {
			return l.Locator < r.Locator
		}
		return l.SubresourceLocator < r.SubresourceLocator
	})
	return candidates, nil
}

func (s *Store) matchesVersionConstraint(
	ctx context.Context,
	resource spec.CatalogResource,
	constraint string,
	version spec.LogicalVersion,
) (bool, error) {
	if frontend, ok := s.frontendFor(resource.FrontendID); ok {
		if matcher, ok := frontend.(spec.FrontendVersionMatcher); ok {
			return matcher.MatchesVersionConstraint(ctx, constraint, version)
		}
	}
	matcher, ok := s.versionMatcherFor(resource.Kind)
	if !ok {
		return false, fmt.Errorf(
			"%w: artifact kind %q, constraint %q",
			spec.ErrVersionMatcherUnavailable,
			resource.Kind,
			constraint,
		)
	}
	matches, err := matcher.MatchesVersionConstraint(ctx, constraint, version)
	if err != nil {
		return false, fmt.Errorf(
			"match version constraint %q for artifact kind %q: %w",
			constraint,
			resource.Kind,
			err,
		)
	}
	return matches, nil
}

func selectorLabelsMatch(selector, labels map[string]string) bool {
	for key, value := range selector {
		if labels[key] != value {
			return false
		}
	}
	return true
}
