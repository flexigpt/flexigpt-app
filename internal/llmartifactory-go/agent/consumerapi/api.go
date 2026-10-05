package consumerapi

import (
	"context"
	"fmt"
	"maps"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog"
	catalogModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition"
	managepackageFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managepackage"
	refreshFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh"
	resourceFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	agentDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/domain"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/import/signer"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition/locator"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/agentv1"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/text/materialize"
)

type API struct {
	roots            root.API
	cat              catalog.API
	sources          source.API
	discovery        refreshFlow.API
	artifacts        artifact.API
	resources        resourceFlow.API
	protection       root.ProtectionAPI
	managedArtifacts managepackageFlow.API
	definitions      definition.API
	texts            *materialize.Adapter

	collections         *plugin.API
	declarationResolver *composition.Resolver
	fallbackProviders   map[declaration.Type]composition.FallbackProvider

	managedAgentProfile *declaration.ManagedProfilePolicy
	importSigner        *signer.Signer
}

type apiOptions struct {
	roots             root.API
	locatorResolvers  []locator.Factory
	fallbackProviders map[declaration.Type]composition.FallbackProvider
	targetMappers     map[declaration.Type]composition.ArtifactTargetMapper
	importSigner      *signer.Signer
}

type Option func(*apiOptions)

// WithRoots enables consumer-facing cross-Root management operations and
// default user-Root Collection creation. Explicit Root-scoped operations do
// not require this option.
func WithRoots(
	value root.API,
) Option {
	return func(options *apiOptions) {
		options.roots = value
	}
}

func WithLocatorResolvers(
	values []locator.Factory,
) Option {
	return func(options *apiOptions) {
		options.locatorResolvers = append(
			[]locator.Factory(nil),
			values...,
		)
	}
}

func WithFallbackProviders(
	values map[declaration.Type]composition.FallbackProvider,
) Option {
	return func(options *apiOptions) {
		if values == nil {
			options.fallbackProviders = nil
			return
		}

		options.fallbackProviders = make(
			map[declaration.Type]composition.FallbackProvider,
			len(values),
		)
		maps.Copy(options.fallbackProviders, values)
	}
}

func WithTargetMappers(
	values map[declaration.Type]composition.ArtifactTargetMapper,
) Option {
	return func(options *apiOptions) {
		if values == nil {
			options.targetMappers = nil
			return
		}

		options.targetMappers = make(
			map[declaration.Type]composition.ArtifactTargetMapper,
			len(values),
		)
		maps.Copy(options.targetMappers, values)
	}
}

func WithManagedAgentImportSigner(
	value *signer.Signer,
) Option {
	return func(options *apiOptions) {
		options.importSigner = value
	}
}

func New(
	sources source.API,
	discovery refreshFlow.API,
	artifacts artifact.API,
	cat catalog.API,
	resources resourceFlow.API,
	managedArtifacts managepackageFlow.API,
	protection root.ProtectionAPI,
	definitions definition.API,
	options ...Option,
) (*API, error) {
	if sources == nil ||
		discovery == nil ||
		artifacts == nil ||
		resources == nil ||
		managedArtifacts == nil ||
		protection == nil || cat == nil || definitions == nil {
		return nil, fmt.Errorf(
			"%w: Agent Store dependencies are incomplete",
			spec.ErrInvalid,
		)
	}

	config := apiOptions{}
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}

	profiles, err := declaration.NewManagedProfileRegistry(
		agentv1.ManagedAgentImportProfileDescriptor(),
	)
	if err != nil {
		return nil, err
	}
	managedAgentProfile, err := profiles.ManagedProfilePolicyFor(
		agentv1.AgentType,
	)
	if err != nil {
		return nil, err
	}

	importSigner := config.importSigner
	if importSigner == nil {
		importSigner, err = signer.NewSigner(nil)
		if err != nil {
			return nil, err
		}
	}

	texts, err := materialize.NewAdapter(resources)
	if err != nil {
		return nil, err
	}

	output := &API{
		roots:             config.roots,
		sources:           sources,
		cat:               cat,
		discovery:         discovery,
		artifacts:         artifacts,
		resources:         resources,
		managedArtifacts:  managedArtifacts,
		protection:        protection,
		definitions:       definitions,
		fallbackProviders: maps.Clone(config.fallbackProviders),

		texts:               texts,
		managedAgentProfile: managedAgentProfile,
		importSigner:        importSigner,
	}

	locators, err := composition.NewProviderLocatorResolver(
		config.locatorResolvers,
		agentLocatorRuntime{catalog: cat},
	)
	if err != nil {
		return nil, err
	}

	graphResolver, err := composition.NewWithOptions(
		composition.ResolverOptions{
			Artifacts:            artifacts,
			Catalog:              cat,
			SourceEntries:        resources,
			Locators:             locators,
			FallbackProviders:    config.fallbackProviders,
			TargetMappers:        config.targetMappers,
			ProtectedBuiltinRoot: agentBuiltinRootID(),
			Limits:               composition.DefaultLimits(),
		},
	)
	if err != nil {
		return nil, err
	}

	collections, err := plugin.NewWithResolver(
		artifacts,
		cat,
		sources,
		discovery,
		managedArtifacts,
		definitions,
		graphResolver,
		agentDomain.AgentCollectionDomainPolicy(),
	)
	if err != nil {
		return nil, err
	}

	output.collections = collections
	output.declarationResolver = graphResolver
	return output, nil
}

func (a *API) requireMutable(
	ctx context.Context,
	rootID rootModel.RootID,
) error {
	if a == nil || a.protection == nil {
		return spec.ErrClosed
	}
	if err := rootID.Validate(); err != nil {
		return err
	}
	if !a.protection.IsProtectedRoot(rootID) {
		return nil
	}
	return a.protection.RequireInstallerPrivilege(ctx)
}

type agentLocatorRuntime struct {
	catalog catalog.API
}

func (r agentLocatorRuntime) ListBySource(
	ctx context.Context,
	rootID rootModel.RootID,
	sourceID sourceModel.SourceID,
	options catalogModel.ListOptions,
) ([]catalogModel.Entry, error) {
	if r.catalog == nil {
		return nil, spec.ErrClosed
	}
	return r.catalog.ListBySource(ctx, rootID, sourceID, options)
}
