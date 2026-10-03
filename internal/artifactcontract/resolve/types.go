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
	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

const (
	membersStr  = "members"
	loopStr     = "loop"
	workflowStr = "workflow"
)

type LocatorRequest struct {
	RootID              rootModel.RootID
	From                *artifactModel.Artifact
	Entry               declaration.Entry
	Locator             declaration.Locator
	ExpectedType        declaration.Type
	ExpectedLogicalName spec.LogicalName
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
	) (artifactModel.ArtifactRef, error)
}

// ArtifactTargetRequest contains a source-backed terminal Artifact selected
// by normal resolver lookup. A registered type mapper may replace it with a
// mapped target before it reaches a consumer capability plan.
type ArtifactTargetRequest struct {
	Artifact   artifactModel.Artifact
	Definition definitionModel.Definition
	Type       declaration.Type
}

// ArtifactTargetMapper projects one source-backed Artifact into a mapped
// target. The mapper is registered by application composition for types that
// intentionally expose a runtime capability rather than a raw Artifact.
//
// Returning handled=false leaves the ordinary Artifact target unchanged.
// Returning handled=true with an error makes the relationship unavailable.
type ArtifactTargetMapper interface {
	MapArtifactTarget(
		ctx context.Context,
		request ArtifactTargetRequest,
	) (MappedTarget, bool, error)
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
	if l.MaxDepth <= 0 || l.MaxDepth > spec.MaxDiscoveryDepth {
		return fmt.Errorf(
			"%w: Artifact resolver depth limit is invalid",
			spec.ErrInvalid,
		)
	}
	if l.MaxNodes <= 0 || l.MaxNodes > spec.MaxDiscoveryEntries {
		return fmt.Errorf(
			"%w: Artifact resolver node limit is invalid",
			spec.ErrInvalid,
		)
	}
	return nil
}

type ResolverOptions struct {
	Artifacts ArtifactReader

	// Catalog is the committed Artifact read projection used for named,
	// contained, and selector relationship resolution.
	//
	// Production composition should provide store/artifact/catalog.API.
	Catalog ArtifactCatalogReader

	// SourceEntries verifies that a selector base is a directory in the
	// declaring Source. It is used only for member-selector resolution.
	SourceEntries SourceEntryInspector

	Locators LocatorResolver
	Registry *Registry

	// ProtectedBuiltinRoot enables current Root followed by protected built-in
	// Root lookup for named external members.
	ProtectedBuiltinRoot rootModel.RootID

	// Tool and Model are the initial mapped-fallback-capable types. A future
	// type may register a provider when its resolver and consumer support it.
	FallbackProviders map[declaration.Type]FallbackProvider

	// TargetMappers project an Artifact-backed terminal target into a mapped
	// target for selected declaration types.
	TargetMappers map[declaration.Type]ArtifactTargetMapper

	// Refresh is never used by normal resolution. It is used only by
	// RefreshPlugin, RefreshAgent, RefreshTeam, and RefreshWorkspace.
	Refresh RefreshCoordinator
	Limits  Limits
}

type Resolver struct {
	artifacts     ArtifactReader
	catalog       ArtifactCatalogReader
	locators      LocatorResolver
	sourceEntries SourceEntryInspector
	registry      *Registry
	targetMappers map[declaration.Type]ArtifactTargetMapper
	builtinRoot   rootModel.RootID
	refresh       RefreshCoordinator
	limits        Limits
}

func NewWithOptions(options ResolverOptions) (*Resolver, error) {
	if options.Artifacts == nil {
		return nil, fmt.Errorf(
			"%w: Artifact resolver ArtifactReader is nil",
			spec.ErrInvalid,
		)
	}
	if options.Catalog == nil {
		return nil, fmt.Errorf(
			"%w: Artifact resolver ArtifactCatalogReader is nil",
			spec.ErrInvalid,
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

	targetMappers := make(
		map[declaration.Type]ArtifactTargetMapper,
		len(options.TargetMappers),
	)
	for declarationType, mapper := range options.TargetMappers {
		if err := declarationType.Validate(); err != nil {
			return nil, err
		}
		if mapper == nil {
			return nil, fmt.Errorf(
				"%w: Artifact target mapper for %q is nil",
				spec.ErrInvalid,
				declarationType,
			)
		}
		targetMappers[declarationType] = mapper
	}

	return &Resolver{
		artifacts:     options.Artifacts,
		catalog:       options.Catalog,
		locators:      options.Locators,
		sourceEntries: options.SourceEntries,
		registry:      registry,
		targetMappers: targetMappers,
		builtinRoot:   options.ProtectedBuiltinRoot,
		refresh:       options.Refresh,
		limits:        limits,
	}, nil
}

func (r *Resolver) ready() error {
	if r == nil || r.artifacts == nil || r.catalog == nil {
		return spec.ErrClosed
	}
	return nil
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
	RootID rootModel.RootID
	Type   declaration.Type
	Name   spec.LogicalName
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
	Base    spec.Locator            `json:"-"`
	Matches []ResolvedSelectorMatch `json:"-"`
}

type ResolvedSelectorMatch struct {
	Artifact artifactModel.ArtifactRef `json:"-"`
	Status   ResolutionStatus          `json:"-"`
	Resolved *ResolvedEntry            `json:"-"`
	Issue    *ResolutionIssue          `json:"-"`
}

// ResolvedEntry is an internal graph node. It can contain canonical
// declaration bodies and nested relationships. CapabilityPlan is its
// supported consumer-facing projection.
type ResolvedEntry struct {
	Type declaration.Type `json:"-"`

	scopeRootID       rootModel.RootID
	DeclarationOrigin *artifactModel.Artifact `json:"-"`

	Artifact   *artifactModel.Artifact     `json:"-"`
	Definition *definitionModel.Definition `json:"-"`
	Mapped     *MappedTarget               `json:"-"`

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

func (r *ResolvedEntry) ArtifactRef() (artifactModel.ArtifactRef, bool) {
	if r == nil || r.Artifact == nil {
		return artifactModel.ArtifactRef{}, false
	}
	return r.Artifact.Ref(), true
}

func (r *ResolvedEntry) RootID() (rootModel.RootID, bool) {
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
			spec.ErrInvalid,
		)
	}
	return ctx.Err()
}
