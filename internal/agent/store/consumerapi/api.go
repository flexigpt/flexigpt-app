package consumerapi

import (
	"context"
	"fmt"
	"maps"

	agentDomain "github.com/flexigpt/flexigpt-app/internal/agent/store/domain"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration/agentv1"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/materializetext"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/signer"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/collection"
)

type API struct {
	roots            compositionapi.RootAPI
	sources          compositionapi.SourceAPI
	discovery        compositionapi.DiscoveryAPI
	artifacts        compositionapi.ArtifactAPI
	resources        compositionapi.ResourceAPI
	protection       compositionapi.ProtectionAPI
	managedArtifacts compositionapi.ManagedArtifactAPI
	texts            *materializetext.Adapter

	collections         *collection.API
	declarationResolver *resolve.Resolver
	fallbackProviders   map[declaration.Type]resolve.FallbackProvider

	managedAgentProfile *declaration.ManagedProfilePolicy
	importSigner        *signer.Signer
}

type apiOptions struct {
	roots             compositionapi.RootAPI
	locatorResolvers  []providerapi.LocatorResolverFactory
	fallbackProviders map[declaration.Type]resolve.FallbackProvider
	targetMappers     map[declaration.Type]resolve.ArtifactTargetMapper
	importSigner      *signer.Signer
}

type Option func(*apiOptions)

// WithRoots enables consumer-facing cross-Root management operations and
// default user-Root Collection creation. Explicit Root-scoped operations do
// not require this option.
func WithRoots(
	value compositionapi.RootAPI,
) Option {
	return func(options *apiOptions) {
		options.roots = value
	}
}

func WithLocatorResolvers(
	values []providerapi.LocatorResolverFactory,
) Option {
	return func(options *apiOptions) {
		options.locatorResolvers = append(
			[]providerapi.LocatorResolverFactory(nil),
			values...,
		)
	}
}

func WithFallbackProviders(
	values map[declaration.Type]resolve.FallbackProvider,
) Option {
	return func(options *apiOptions) {
		if values == nil {
			options.fallbackProviders = nil
			return
		}

		options.fallbackProviders = make(
			map[declaration.Type]resolve.FallbackProvider,
			len(values),
		)
		maps.Copy(options.fallbackProviders, values)
	}
}

func WithTargetMappers(
	values map[declaration.Type]resolve.ArtifactTargetMapper,
) Option {
	return func(options *apiOptions) {
		if values == nil {
			options.targetMappers = nil
			return
		}

		options.targetMappers = make(
			map[declaration.Type]resolve.ArtifactTargetMapper,
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
	sources compositionapi.SourceAPI,
	discovery compositionapi.DiscoveryAPI,
	artifacts compositionapi.ArtifactAPI,
	resources compositionapi.ResourceAPI,
	managedArtifacts compositionapi.ManagedArtifactAPI,
	protection compositionapi.ProtectionAPI,
	options ...Option,
) (*API, error) {
	if sources == nil ||
		discovery == nil ||
		artifacts == nil ||
		resources == nil ||
		managedArtifacts == nil ||
		protection == nil {
		return nil, fmt.Errorf(
			"%w: Agent Store dependencies are incomplete",
			basespec.ErrInvalid,
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

	texts, err := materializetext.NewAdapter(resources)
	if err != nil {
		return nil, err
	}

	output := &API{
		roots:             config.roots,
		sources:           sources,
		discovery:         discovery,
		artifacts:         artifacts,
		resources:         resources,
		managedArtifacts:  managedArtifacts,
		protection:        protection,
		fallbackProviders: maps.Clone(config.fallbackProviders),

		texts:               texts,
		managedAgentProfile: managedAgentProfile,
		importSigner:        importSigner,
	}

	locators, err := resolve.NewProviderLocatorResolver(
		config.locatorResolvers,
		agentLocatorRuntime{artifacts: artifacts},
	)
	if err != nil {
		return nil, err
	}

	graphResolver, err := resolve.NewWithOptions(
		resolve.ResolverOptions{
			Artifacts:            artifacts,
			SourceArtifacts:      artifacts,
			SourceEntries:        resources,
			Locators:             locators,
			FallbackProviders:    config.fallbackProviders,
			TargetMappers:        config.targetMappers,
			ProtectedBuiltinRoot: agentBuiltinRootID(),
			Limits:               resolve.DefaultLimits(),
		},
	)
	if err != nil {
		return nil, err
	}

	collections, err := collection.NewWithResolver(
		sources,
		discovery,
		artifacts,
		managedArtifacts,
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
	rootID root.RootID,
	allowProtected bool,
) error {
	if a == nil || a.protection == nil {
		return basespec.ErrClosed
	}
	if err := rootID.Validate(); err != nil {
		return err
	}
	if !a.protection.IsProtectedRoot(rootID) {
		return nil
	}
	if !allowProtected {
		return fmt.Errorf(
			"%w: protected Root %q requires trusted installer access",
			basespec.ErrProtected,
			rootID,
		)
	}
	return a.protection.RequirePrivilegedInstaller(ctx)
}

type agentLocatorRuntime struct {
	artifacts compositionapi.ArtifactAPI
}

func (r agentLocatorRuntime) ListArtifactsBySource(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
) ([]artifact.Artifact, error) {
	if r.artifacts == nil {
		return nil, basespec.ErrClosed
	}
	return r.artifacts.ListBySource(ctx, rootID, sourceID)
}
