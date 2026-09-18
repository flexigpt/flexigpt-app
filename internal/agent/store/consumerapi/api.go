package consumerapi

import (
	"context"
	"fmt"
	"maps"

	agentDomain "github.com/flexigpt/flexigpt-app/internal/agent/store/domain"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/resolve"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/compositionapi"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	"github.com/flexigpt/flexigpt-app/internal/collection"
)

type API struct {
	sources          compositionapi.SourceAPI
	discovery        compositionapi.DiscoveryAPI
	artifacts        compositionapi.ArtifactAPI
	resources        compositionapi.ResourceAPI
	managedArtifacts compositionapi.ManagedArtifactAPI
	protection       compositionapi.ProtectionAPI

	collections         *collection.API
	declarationResolver *resolve.Resolver
}

type apiOptions struct {
	locatorResolvers  []providerapi.LocatorResolverFactory
	fallbackProviders map[declaration.Type]resolve.FallbackProvider
}

type Option func(*apiOptions)

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

	output := &API{
		sources:          sources,
		discovery:        discovery,
		artifacts:        artifacts,
		resources:        resources,
		managedArtifacts: managedArtifacts,
		protection:       protection,
	}

	locators, err := resolve.NewProviderLocatorResolver(
		config.locatorResolvers,
		agentLocatorRuntime{api: output},
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
	api *API
}

func (r agentLocatorRuntime) ListArtifactsBySource(
	ctx context.Context,
	rootID root.RootID,
	sourceID source.SourceID,
) ([]artifact.Artifact, error) {
	if r.api == nil || r.api.artifacts == nil {
		return nil, basespec.ErrClosed
	}
	return r.api.artifacts.ListBySource(ctx, rootID, sourceID)
}
