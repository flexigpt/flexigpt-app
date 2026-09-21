// Package resolve resolves validated portable Artifact declarations.
//
// It owns declaration lookup, alias traversal, composition expansion,
// member-selector expansion, fallback dispatch, and explicit refresh closure
// coordination. It does not execute Artifacts, materialize resources, manage
// secrets, connect MCP servers, or schedule Workflows.
package resolve

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/definition"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
)

const (
	membersStr  = "members"
	loopStr     = "loop"
	workflowStr = "workflow"
)

type LocatorRequest struct {
	RootID              root.RootID
	From                *artifact.Artifact
	Entry               declaration.Entry
	Locator             declaration.Locator
	ExpectedType        declaration.Type
	ExpectedLogicalName basespec.LogicalName
}

// LocatorResolver resolves one supported external declaration locator kind.
//
// The initial application registration supports local path locators. URL, Git,
// package, archive, and other Locator kinds remain portable declaration data
// and return ErrLocatorUnresolved until a provider is registered for them.
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

type ResolverOptions struct {
	Artifacts ArtifactReader

	// SourceArtifacts is used only for selector expansion. Normal named,
	// located, and contained relationship resolution does not require it.
	SourceArtifacts SourceArtifactReader

	// SourceEntries verifies that a selector base is a directory in the
	// declaring Source. It is used only for member-selector resolution.
	SourceEntries SourceEntryInspector

	Locators LocatorResolver
	Registry *Registry

	// ProtectedBuiltinRoot enables current Root followed by protected built-in
	// Root lookup for named external members.
	ProtectedBuiltinRoot root.RootID

	// Tool and Model are the initial mapped-fallback-capable types. A future
	// type may register a provider when its resolver and consumer support it.
	FallbackProviders map[declaration.Type]FallbackProvider

	// Refresh is never used by normal resolution. It is used only by
	// RefreshPlugin, RefreshAgent, RefreshTeam, and RefreshWorkspace.
	Refresh RefreshCoordinator
	Limits  Limits
}

type Resolver struct {
	artifacts       ArtifactReader
	sourceArtifacts SourceArtifactReader
	locators        LocatorResolver
	sourceEntries   SourceEntryInspector
	registry        *Registry
	builtinRoot     root.RootID
	refresh         RefreshCoordinator
	limits          Limits
}

func NewWithOptions(options ResolverOptions) (*Resolver, error) {
	if options.Artifacts == nil {
		return nil, fmt.Errorf(
			"%w: Artifact resolver ArtifactReader is nil",
			basespec.ErrInvalid,
		)
	}
	limits := options.Limits.Normalized()
	if err := limits.Validate(); err != nil {
		return nil, err
	}
	if options.ProtectedBuiltinRoot != "" {
		if err := options.ProtectedBuiltinRoot.Validate(); err != nil {
			return nil, err
		}
	}

	registry := options.Registry
	if registry == nil {
		registry = DefaultRegistry()
	}
	if err := registry.ValidateComplete(); err != nil {
		return nil, err
	}
	registry = registry.Clone()
	for declarationType, provider := range options.FallbackProviders {
		var err error
		registry, err = registry.WithFallback(
			declarationType,
			provider,
		)
		if err != nil {
			return nil, err
		}
	}

	sourceArtifacts := options.SourceArtifacts
	if sourceArtifacts == nil {
		if value, found := options.Artifacts.(SourceArtifactReader); found {
			sourceArtifacts = value
		}
	}

	sourceEntries := options.SourceEntries
	if sourceEntries == nil {
		if value, found := options.Artifacts.(SourceEntryInspector); found {
			sourceEntries = value
		}
	}

	return &Resolver{
		artifacts:       options.Artifacts,
		sourceArtifacts: sourceArtifacts,
		locators:        options.Locators,
		sourceEntries:   sourceEntries,
		registry:        registry,
		builtinRoot:     options.ProtectedBuiltinRoot,
		refresh:         options.Refresh,
		limits:          limits,
	}, nil
}

type ResolutionStatus string

const (
	ResolutionAvailable   ResolutionStatus = "available"
	ResolutionUnavailable ResolutionStatus = "unavailable"
	ResolutionAmbiguous   ResolutionStatus = "ambiguous"
)

type ResolutionIssue struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type FallbackRequest struct {
	RootID root.RootID
	Type   declaration.Type
	Name   basespec.LogicalName
	Scope  declaration.LookupScope
}

// FallbackProvider is called only for a named external relationship after the
// applicable current and protected built-in Root lookups found no target.
//
// It is never called for located members, contained members, member selectors,
// source-selected aliases, or arbitrary ArtifactRefs.
type FallbackProvider interface {
	ResolveFallback(
		ctx context.Context,
		request FallbackRequest,
	) (FallbackTarget, bool, error)
}

// ResolvedRelationship is in-process resolver graph state. Consumer APIs
// expose CapabilityPlan rather than serializing resolver implementation data.
type ResolvedRelationship struct {
	Declared  declaration.Entry          `json:"-"`
	Form      declaration.MemberForm     `json:"-"`
	Status    ResolutionStatus           `json:"-"`
	Required  bool                       `json:"-"`
	Scope     declaration.LookupScope    `json:"-"`
	Overrides map[string]json.RawMessage `json:"-"`
	Use       map[string]json.RawMessage `json:"-"`

	Resolved *ResolvedEntry    `json:"-"`
	Selector *ResolvedSelector `json:"-"`
	Issue    *ResolutionIssue  `json:"-"`
}

func (r ResolvedRelationship) IsAvailable() bool {
	return r.Status == ResolutionAvailable &&
		(r.Resolved != nil || r.Selector != nil)
}

type ResolvedSelector struct {
	Type    declaration.Type        `json:"-"`
	Base    basespec.Locator        `json:"-"`
	Matches []ResolvedSelectorMatch `json:"-"`
}

type ResolvedSelectorMatch struct {
	Artifact artifact.ArtifactRef `json:"-"`
	Status   ResolutionStatus     `json:"-"`
	Resolved *ResolvedEntry       `json:"-"`
	Issue    *ResolutionIssue     `json:"-"`
}

// ResolvedEntry is an internal graph node. It can contain canonical
// declaration bodies and nested relationships. CapabilityPlan is its
// supported consumer-facing projection.
type ResolvedEntry struct {
	Type declaration.Type `json:"-"`

	scopeRootID       root.RootID
	DeclarationOrigin *artifact.Artifact `json:"-"`

	Artifact   *artifact.Artifact     `json:"-"`
	Definition *definition.Definition `json:"-"`
	Mapped     *MappedTarget          `json:"-"`

	Members            []*ResolvedEntry       `json:"-"`
	MemberResults      []ResolvedRelationship `json:"-"`
	AllowedTools       []*ResolvedEntry       `json:"-"`
	AllowedToolResults []ResolvedRelationship `json:"-"`

	DirectLoop           *ResolvedEntry        `json:"-"`
	DirectLoopResult     *ResolvedRelationship `json:"-"`
	DirectWorkflow       *ResolvedEntry        `json:"-"`
	DirectWorkflowResult *ResolvedRelationship `json:"-"`

	Loop      *ResolvedLoop      `json:"-"`
	Workflow  *ResolvedWorkflow  `json:"-"`
	Workspace *ResolvedWorkspace `json:"-"`
	MCP       *ResolvedMCP       `json:"-"`
}

type ResolvedMCP struct {
	Policy         *ResolvedEntry        `json:"-"`
	PolicyResult   *ResolvedRelationship `json:"-"`
	PolicyRequired bool                  `json:"-"`
}

type ResolvedLoop struct {
	BodyResult    *ResolvedRelationship    `json:"-"`
	Body          *ResolvedEntry           `json:"-"`
	MaxIterations int                      `json:"-"`
	Until         *declaration.OutputMatch `json:"-"`
}

type ResolvedWorkflow struct {
	Start []string               `json:"-"`
	Nodes []ResolvedWorkflowNode `json:"-"`
	Edges []ResolvedWorkflowEdge `json:"-"`
}

type ResolvedWorkflowNode struct {
	ID           string                `json:"-"`
	Join         string                `json:"-"`
	Member       *ResolvedEntry        `json:"-"`
	MemberResult *ResolvedRelationship `json:"-"`
}

type ResolvedWorkflowEdge struct {
	From  string                   `json:"-"`
	To    string                   `json:"-"`
	Match *declaration.OutputMatch `json:"-"`
}

type ResolvedWorkspace struct {
	Members       []*ResolvedEntry       `json:"-"`
	MemberResults []ResolvedRelationship `json:"-"`
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
