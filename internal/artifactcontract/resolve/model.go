// Package resolve resolves typed Artifact declaration graphs above the
// generic Artifact Store boundary.
//
// Artifact Store indexes source-backed Definitions and Artifacts. This package
// resolves symbolic references, typed composition, Collection inclusion, Agent
// members, Team members, Loops, Workflows, and Workspace roots.
package resolve

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
)

type LocatorRequest struct {
	RootID              root.RootID
	From                *artifact.Artifact
	Entry               declaration.Entry
	Locator             declaration.Locator
	ExpectedType        declaration.Type
	ExpectedLogicalName basespec.LogicalName
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
	artifacts ArtifactReader
	locators  LocatorResolver
	limits    Limits
	options   Options
}

func New(
	artifacts ArtifactReader,
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
	entry declaration.Entry,
) (Graph, error) {
	if err := validateResolutionContext(ctx); err != nil {
		return Graph{}, err
	}
	if err := rootID.Validate(); err != nil {
		return Graph{}, err
	}
	state := newResolutionState()
	return r.resolveInlineGraph(ctx, &state, rootID, entry)
}

type Graph struct {
	Root *ResolvedEntry
}

// ResolutionStatus describes the declaration-resolution state of one
// composition edge. It does not represent consumer-specific runtime readiness.
type ResolutionStatus string

const (
	ResolutionAvailable   ResolutionStatus = "available"
	ResolutionUnavailable ResolutionStatus = "unavailable"
	ResolutionAmbiguous   ResolutionStatus = "ambiguous"
)

type ResolutionIssue struct {
	Code    string
	Message string
}

// ResolvedRelationship preserves every declared composition occurrence. An
// unavailable or ambiguous edge does not invalidate its containing
// Collection, Agent, Team, Workflow, Loop, or Workspace declaration.
type ResolvedRelationship struct {
	Declared declaration.Entry
	Status   ResolutionStatus
	Resolved *ResolvedEntry
	Issue    *ResolutionIssue
}

func (r ResolvedRelationship) IsAvailable() bool {
	return r.Status == ResolutionAvailable && r.Resolved != nil
}

type ResolvedEntry struct {
	Type declaration.Type

	scopeRootID       root.RootID
	DeclarationOrigin *artifact.Artifact

	Artifact   *artifact.Artifact
	Definition *definition.Definition
	Inline     *declaration.Entry

	// Members and AllowedTools are compatibility projections containing only
	// currently available entries. New consumers must use the corresponding
	// relationship result fields to preserve partial composition state.
	Members            []*ResolvedEntry
	MemberResults      []ResolvedRelationship
	AllowedTools       []*ResolvedEntry
	AllowedToolResults []ResolvedRelationship
	Program            *ResolvedEntry
	ProgramResult      *ResolvedRelationship

	Loop      *ResolvedLoop
	Workflow  *ResolvedWorkflow
	Workspace *ResolvedWorkspace
}

type ResolvedLoop struct {
	BodyResult    *ResolvedRelationship
	Body          *ResolvedEntry
	MaxIterations int
	Until         *declaration.OutputMatch
}

type ResolvedWorkflow struct {
	Start []string
	Nodes []ResolvedWorkflowNode
	Edges []ResolvedWorkflowEdge
}

type ResolvedWorkflowNode struct {
	ID           string
	Join         string
	Target       *ResolvedEntry
	TargetResult *ResolvedRelationship
}

type ResolvedWorkflowEdge struct {
	From  string
	To    string
	Match *declaration.OutputMatch
}

type ResolvedWorkspace struct {
	Roots       []*ResolvedEntry
	RootResults []ResolvedRelationship
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

func validateResolutionContext(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf(
			"%w: Artifact resolver context is nil",
			basespec.ErrInvalid,
		)
	}
	return ctx.Err()
}
