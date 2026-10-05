package composition

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	definitionModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition/model"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
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
// A locator resolver consumes committed catalog state only. It must not create
// Sources, prepare discovery, refresh content, or publish packages.
type LocatorResolver interface {
	ResolveArtifactLocator(
		ctx context.Context,
		request LocatorRequest,
	) (artifactModel.ArtifactRef, error)
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
	value := l.Normalized()
	if value.MaxDepth <= 0 || value.MaxDepth > spec.MaxDiscoveryDepth {
		return fmt.Errorf(
			"%w: Artifact composition depth limit is invalid",
			spec.ErrInvalid,
		)
	}
	if value.MaxNodes <= 0 || value.MaxNodes > spec.MaxDiscoveryEntries {
		return fmt.Errorf(
			"%w: Artifact composition node limit is invalid",
			spec.ErrInvalid,
		)
	}
	return nil
}

type ResolverOptions struct {
	Artifacts ArtifactReader
	Catalog   ArtifactCatalogReader

	// SourceEntries is required only for selector relationships. It remains
	// separate from catalog reads because selector bases require verified
	// physical Source entry metadata.
	SourceEntries SourceEntryInspector

	Locators        LocatorResolver
	Interpretations *interpretation.Registry
	Scope           ScopeBinding

	// DirectCapabilities are application-supplied non-Artifact targets. They
	// are consulted only after applicable Artifact lookup has no result.
	DirectCapabilities []DirectCapabilityProvider

	// ArtifactCapabilityProjectors let application composition attach a
	// runtime-neutral target to a source-backed Artifact. A projector may keep
	// the Artifact target or produce an explicitly direct target with durable
	// provider evidence. It cannot fabricate Artifact identity.
	ArtifactCapabilityProjectors map[declaration.Type]ArtifactCapabilityProjector

	// Refresh is absent from ordinary resolution. It is required only by the
	// explicit RefreshPlugin, RefreshAgent, RefreshTeam, and RefreshWorkspace
	// operations.
	Refresh RefreshCoordinator
	Limits  Limits
}

type Resolver struct {
	artifacts       ArtifactReader
	catalog         ArtifactCatalogReader
	sourceEntries   SourceEntryInspector
	locators        LocatorResolver
	interpretations *interpretation.Registry
	scope           ScopeBinding
	refresh         RefreshCoordinator
	limits          Limits

	directCapabilities []DirectCapabilityProvider
	projectors         map[declaration.Type]ArtifactCapabilityProjector
}

func New(options ResolverOptions) (*Resolver, error) {
	if options.Artifacts == nil {
		return nil, fmt.Errorf(
			"%w: Artifact composition ArtifactReader is nil",
			spec.ErrInvalid,
		)
	}
	if options.Catalog == nil {
		return nil, fmt.Errorf(
			"%w: Artifact composition ArtifactCatalogReader is nil",
			spec.ErrInvalid,
		)
	}
	if options.Interpretations == nil {
		return nil, fmt.Errorf(
			"%w: Artifact composition interpretation registry is nil",
			spec.ErrInvalid,
		)
	}
	if err := options.Scope.Validate(); err != nil {
		return nil, err
	}

	limits := options.Limits.Normalized()
	if err := limits.Validate(); err != nil {
		return nil, err
	}

	directCapabilities := append(
		[]DirectCapabilityProvider(nil),
		options.DirectCapabilities...,
	)
	seenProviders := make(
		map[string]struct{},
		len(directCapabilities),
	)
	for index, provider := range directCapabilities {
		if provider == nil {
			return nil, fmt.Errorf(
				"%w: direct capability provider %d is nil",
				spec.ErrInvalid,
				index,
			)
		}
		identity := provider.ProviderIdentity()
		if err := spec.ValidateIdentifier(
			"direct capability provider identity",
			identity,
			spec.MaxKindBytes,
		); err != nil {
			return nil, err
		}
		if _, duplicate := seenProviders[identity]; duplicate {
			return nil, fmt.Errorf(
				"%w: duplicate direct capability provider %q",
				spec.ErrConflict,
				identity,
			)
		}
		seenProviders[identity] = struct{}{}
	}

	projectors := make(
		map[declaration.Type]ArtifactCapabilityProjector,
		len(options.ArtifactCapabilityProjectors),
	)
	for declarationType, projector := range options.ArtifactCapabilityProjectors {
		if err := declarationType.Validate(); err != nil {
			return nil, err
		}
		if projector == nil {
			return nil, fmt.Errorf(
				"%w: Artifact capability projector for %q is nil",
				spec.ErrInvalid,
				declarationType,
			)
		}
		projectors[declarationType] = projector
	}

	return &Resolver{
		artifacts:          options.Artifacts,
		catalog:            options.Catalog,
		sourceEntries:      options.SourceEntries,
		locators:           options.Locators,
		interpretations:    options.Interpretations,
		scope:              options.Scope,
		refresh:            options.Refresh,
		limits:             limits,
		directCapabilities: directCapabilities,
		projectors:         projectors,
	}, nil
}

// WithRefreshCoordinator derives an explicitly refresh-capable resolver from
// the same immutable composition configuration. It does not construct another
// graph resolver, another locator registry, or another interpretation
// registry. Workspace owns its refresh coordinator because Workspace owns its
// source-preparation rules.
func (r *Resolver) WithRefreshCoordinator(
	coordinator RefreshCoordinator,
) (*Resolver, error) {
	if err := r.ready(); err != nil {
		return nil, err
	}
	if coordinator == nil {
		return nil, fmt.Errorf(
			"%w: Artifact composition refresh coordinator is nil",
			spec.ErrInvalid,
		)
	}
	output := *r
	output.refresh = coordinator
	return &output, nil
}

func (r *Resolver) ready() error {
	if r == nil ||
		r.artifacts == nil ||
		r.catalog == nil ||
		r.interpretations == nil {
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

// ResolvedRelationship is private composition graph state. Family APIs expose
// capability plans, direct Plugin membership views, or family-specific
// projections instead of serializing this implementation graph.
type ResolvedRelationship struct {
	Declared  declaration.Entry          `json:"-"`
	Form      declaration.MemberForm     `json:"-"`
	Status    ResolutionStatus           `json:"-"`
	Path      []string                   `json:"-"`
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

// ResolvedEntry joins a source-backed declaration Artifact, its current
// immutable Definition, and the family-owned direct relationship facts.
//
// A direct capability target has no Artifact or Definition of its own. It can
// still occur as a relationship result, but never as the root of an
// ArtifactRef-based resolution operation.
type ResolvedEntry struct {
	Type declaration.Type `json:"-"`

	scopeRootID       rootModel.RootID
	DeclarationOrigin *artifactModel.Artifact `json:"-"`

	Artifact      *artifactModel.Artifact     `json:"-"`
	Definition    *definitionModel.Definition `json:"-"`
	Target        *CapabilityTarget           `json:"-"`
	Relationships []ResolvedRelationship      `json:"-"`
}

func (r *ResolvedEntry) ArtifactRef() (
	artifactModel.ArtifactRef,
	bool,
) {
	if r == nil || r.Artifact == nil {
		return artifactModel.ArtifactRef{}, false
	}
	return r.Artifact.Ref(), true
}

func (r *ResolvedEntry) RootID() (
	rootModel.RootID,
	bool,
) {
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

func (r *ResolvedEntry) TargetValue() (
	CapabilityTarget,
	bool,
) {
	if r == nil || r.Target == nil {
		return CapabilityTarget{}, false
	}
	return r.Target.Clone(), true
}

func (r *ResolvedEntry) Relationship(
	path ...string,
) (*ResolvedRelationship, bool) {
	if r == nil {
		return nil, false
	}
	for _, value := range r.Relationships {
		if !slices.Equal(value.Path, path) {
			continue
		}
		copyValue := cloneResolvedRelationship(value)
		return &copyValue, true
	}
	return nil, false
}

func cloneResolvedRelationship(
	value ResolvedRelationship,
) ResolvedRelationship {
	output := value
	output.Declared = value.Declared.Clone()
	output.Path = append([]string(nil), value.Path...)
	output.Overrides = declaration.CloneRawMessageMap(value.Overrides)
	output.Use = declaration.CloneRawMessageMap(value.Use)
	if value.Issue != nil {
		issue := *value.Issue
		output.Issue = &issue
	}
	if value.Selector == nil {
		return output
	}

	selector := *value.Selector
	selector.Matches = make(
		[]ResolvedSelectorMatch,
		len(value.Selector.Matches),
	)
	copy(selector.Matches, value.Selector.Matches)
	for index := range selector.Matches {
		if selector.Matches[index].Issue == nil {
			continue
		}
		issue := *selector.Matches[index].Issue
		selector.Matches[index].Issue = &issue
	}
	output.Selector = &selector
	return output
}

func validateResolutionContext(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf(
			"%w: Artifact composition context is nil",
			spec.ErrInvalid,
		)
	}
	return ctx.Err()
}
