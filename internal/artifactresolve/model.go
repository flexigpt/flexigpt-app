// Package artifactresolve resolves typed Artifact declaration graphs above the
// generic Artifact Store boundary.
//
// Artifact Store indexes source-backed Definitions and Artifacts. This package
// resolves symbolic references, typed composition, Collection inclusion, Agent
// members, Team members, Loops, Workflows, and Workspace roots.
package artifactresolve

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
)

type LocatorRequest struct {
	RootID       root.RootID
	From         *artifact.Artifact
	Locator      artifactcontract.Locator
	ExpectedType artifactcontract.Type
}

// LocatorResolver is implemented by application-owned path, URL, Git,
// package, archive, or registry integrations. Command locators intentionally
// remain implementation data and are never opened by this resolver.
type LocatorResolver interface {
	ResolveArtifactLocator(
		ctx context.Context,
		request LocatorRequest,
	) (artifact.ArtifactRef, error)
}

type Limits struct {
	MaxDepth int
	MaxNodes int
}

func DefaultLimits() Limits {
	return Limits{
		MaxDepth: 64,
		MaxNodes: 16_384,
	}
}

func (l Limits) Normalized() Limits {
	if l.MaxDepth == 0 {
		l.MaxDepth = DefaultLimits().MaxDepth
	}
	if l.MaxNodes == 0 {
		l.MaxNodes = DefaultLimits().MaxNodes
	}
	return l
}

func (l Limits) Validate() error {
	l = l.Normalized()
	if l.MaxDepth <= 0 || l.MaxDepth > basespec.MaxDiscoveryDepth {
		return fmt.Errorf(
			"%w: Artifact resolver depth limit is invalid",
			basespec.ErrInvalid,
		)
	}
	if l.MaxNodes <= 0 || l.MaxNodes > basespec.MaxDiscoveryEntries {
		return fmt.Errorf(
			"%w: Artifact resolver node limit is invalid",
			basespec.ErrInvalid,
		)
	}
	return nil
}

type Options struct {
	IncludeDisabled bool
}

type Resolver struct {
	artifacts compositionapi.ArtifactAPI
	locators  LocatorResolver
	limits    Limits
	options   Options
}

func New(
	artifacts compositionapi.ArtifactAPI,
	locators LocatorResolver,
	limits Limits,
	options Options,
) (*Resolver, error) {
	if artifacts == nil {
		return nil, fmt.Errorf(
			"%w: Artifact resolver Artifact API is nil",
			basespec.ErrInvalid,
		)
	}
	limits = limits.Normalized()
	if err := limits.Validate(); err != nil {
		return nil, err
	}
	return &Resolver{
		artifacts: artifacts,
		locators:  locators,
		limits:    limits,
		options:   options,
	}, nil
}

func (r *Resolver) ResolveInline(
	ctx context.Context,
	rootID root.RootID,
	entry artifactcontract.Entry,
) (Graph, error) {
	if err := rootID.Validate(); err != nil {
		return Graph{}, err
	}
	state := resolutionState{
		collections: make(map[artifact.ArtifactRef]struct{}),
	}
	return r.resolveInlineGraph(ctx, &state, rootID, entry)
}

type Graph struct {
	Root *ResolvedEntry
}

type ResolvedEntry struct {
	Type artifactcontract.Type

	scopeRootID root.RootID

	Artifact   *artifact.Artifact
	Definition *definition.Definition
	Inline     *artifactcontract.Entry

	Members      []*ResolvedEntry
	AllowedTools []*ResolvedEntry
	Program      *ResolvedEntry

	Loop      *ResolvedLoop
	Workflow  *ResolvedWorkflow
	Workspace *ResolvedWorkspace
}

type ResolvedLoop struct {
	Body          *ResolvedEntry
	MaxIterations int
	Until         *artifactcontract.OutputMatch
}

type ResolvedWorkflow struct {
	Start []string
	Nodes []ResolvedWorkflowNode
	Edges []ResolvedWorkflowEdge
}

type ResolvedWorkflowNode struct {
	ID     string
	Join   string
	Target *ResolvedEntry
}

type ResolvedWorkflowEdge struct {
	From  string
	To    string
	Match *artifactcontract.OutputMatch
}

type ResolvedWorkspace struct {
	Roots []*ResolvedEntry
}

func (r *ResolvedEntry) ArtifactRef() (artifact.ArtifactRef, bool) {
	if r == nil || r.Artifact == nil {
		return artifact.ArtifactRef{}, false
	}
	return r.Artifact.Ref(), true
}

func (r *ResolvedEntry) RootID() (root.RootID, bool) {
	if r == nil {
		return "", false
	}
	if r.Artifact != nil {
		return r.Artifact.RootID, true
	}
	if r.scopeRootID != "" {
		return r.scopeRootID, true
	}
	return "", false
}
