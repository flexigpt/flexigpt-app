package artifactstore

import (
	"context"
	"fmt"
	"sort"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/validate"
)

func (s *Store) GetDependencies(ctx context.Context, recordID spec.RecordID) ([]spec.ArtifactSelector, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return nil, err
	}
	defer finish()
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
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return spec.DependencyGraph{}, err
	}
	defer finish()
	record, err := s.GetRecord(ctx, recordID)
	if err != nil {
		return spec.DependencyGraph{}, err
	}
	if record.LastResolvedDefinitionDigest == nil {
		return spec.DependencyGraph{}, fmt.Errorf("%w: record has no resolved definition", spec.ErrConflict)
	}
	generation, err := s.GetRootCatalogGeneration(ctx, record.RootID)
	if err != nil {
		return spec.DependencyGraph{}, err
	}
	if err := s.ensureRootCatalogCurrent(ctx, record.RootID, generation); err != nil {
		return spec.DependencyGraph{}, err
	}
	rootDefinitionDigest := *record.LastResolvedDefinitionDigest
	graph := spec.DependencyGraph{
		RootRecordID: recordID,
		Nodes:        map[spec.Digest]spec.CanonicalDefinition{},
		Edges:        map[spec.Digest][]spec.DependencyExplanation{},
	}
	visiting := map[spec.Digest]bool{}
	snapshots := make([]spec.ArtifactDependencySnapshot, 0)
	now := s.nowUTC()
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
		for selectorIndex, selector := range definition.DependencySelectors {
			explanation, err := s.ExplainDependencyResolution(ctx, record.RootID, selector)
			if err != nil {
				return err
			}
			graph.Edges[digest] = append(graph.Edges[digest], explanation)
			graph.Diagnostics = append(graph.Diagnostics, explanation.Diagnostics...)
			candidateRefs := make(
				[]spec.DependencyCandidateRef,
				0,
				len(explanation.Candidates),
			)
			for _, candidate := range explanation.Candidates {
				candidateRefs = append(candidateRefs, spec.DependencyCandidateRef{
					Resource: spec.CatalogResourceKey{
						SourceID:           candidate.Resource.SourceID,
						Locator:            candidate.Resource.Locator,
						SubresourceLocator: candidate.Resource.SubresourceLocator,
					},
					DefinitionDigest: candidate.Definition.Digest,
				})
			}
			state := spec.DependencyResolutionStateResolved
			switch len(candidateRefs) {
			case 0:
				state = spec.DependencyResolutionStateMissing
			case 1:
			default:
				state = spec.DependencyResolutionStateAmbiguous
			}
			snapshot := spec.ArtifactDependencySnapshot{
				RootID:               record.RootID,
				RecordID:             record.RecordID,
				CatalogGeneration:    generation.Generation,
				RootDefinitionDigest: rootDefinitionDigest,
				DefinitionDigest:     digest,
				SelectorIndex:        selectorIndex,
				Selector:             selector,
				State:                state,
				Candidates:           candidateRefs,
				Diagnostics:          append([]spec.Diagnostic(nil), explanation.Diagnostics...),
				ModifiedAt:           now,
			}
			if err := validate.ValidateArtifactDependencySnapshot(snapshot); err != nil {
				return err
			}
			snapshots = append(snapshots, snapshot)
			if len(explanation.Candidates) == 1 {
				if err := visit(explanation.Candidates[0].Definition.Digest); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := visit(rootDefinitionDigest); err != nil {
		return spec.DependencyGraph{}, err
	}
	currentGeneration, err := s.GetRootCatalogGeneration(ctx, record.RootID)
	if err != nil {
		return spec.DependencyGraph{}, err
	}
	if currentGeneration.Generation != generation.Generation {
		return spec.DependencyGraph{}, fmt.Errorf(
			"%w: root catalog changed while building dependency graph",
			spec.ErrConflict,
		)
	}
	if err := s.repository.ReplaceDependencySnapshots(
		ctx,
		spec.DependencySnapshotPublication{
			RootID:                   record.RootID,
			RecordID:                 record.RecordID,
			RootDefinitionDigest:     rootDefinitionDigest,
			CatalogGeneration:        generation.Generation,
			ExpectedRecordModifiedAt: record.ModifiedAt,
			Snapshots:                snapshots,
		},
	); err != nil {
		return spec.DependencyGraph{}, err
	}
	return graph, nil
}

func (s *Store) ExplainDependencyResolution(
	ctx context.Context,
	rootID spec.RootID,
	selector spec.ArtifactSelector,
) (spec.DependencyExplanation, error) {
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return spec.DependencyExplanation{}, err
	}
	defer finish()
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
	ctx, finish, err := s.beginOperation(ctx)
	if err != nil {
		return nil, err
	}
	defer finish()
	if err := validate.ValidateArtifactSelector(selector); err != nil {
		return nil, fmt.Errorf(
			"%w: dependency selector: %w",
			spec.ErrInvalidRequest,
			err,
		)
	}
	resources, err := s.ListCatalogResourcesForRoot(ctx, rootID)
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
