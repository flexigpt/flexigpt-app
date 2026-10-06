package agent

import (
	"context"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/definition"
	managepackageFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/managepackage"
	refreshFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/refresh"
	resourceFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/resource"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	agentv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/contract/v1"
	agentDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/domain"
	agentsigner "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/import/signer"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
	pluginAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
)

type Service struct {
	roots            root.API
	cat              catalog.API
	sources          source.API
	discovery        refreshFlow.API
	artifacts        artifact.API
	resources        resourceFlow.API
	protection       root.ProtectionAPI
	managedArtifacts managepackageFlow.API
	definitions      definition.API
	plugins          *pluginAPI.API

	declarationResolver *composition.Resolver
	managedAgentProfile *declaration.ManagedProfilePolicy
	importSigner        *agentsigner.Signer
	interpretations     *coreinterpretation.Registry
}

type apiOptions struct {
	roots           root.API
	resolver        *composition.Resolver
	importSigner    *agentsigner.Signer
	interpretations *coreinterpretation.Registry
}

type Option func(*apiOptions)

// WithDeclarationInterpretations supplies the immutable declaration registry
// required by managed Agent import admission and contained-definition planning.
func WithDeclarationInterpretations(
	value *coreinterpretation.Registry,
) Option {
	return func(options *apiOptions) {
		options.interpretations = value
	}
}

// WithRoots enables consumer-facing cross-Root management operations and
// default user-Root Plugin creation. Explicit Root-scoped operations do
// not require this option.
func WithRoots(
	value root.API,
) Option {
	return func(options *apiOptions) {
		options.roots = value
	}
}

// WithCompositionResolver supplies the one LLM Artifactory composition owner.
// Agent does not construct a private locator registry or graph resolver.
func WithCompositionResolver(
	value *composition.Resolver,
) Option {
	return func(options *apiOptions) {
		options.resolver = value
	}
}

func WithManagedAgentImportSigner(
	value *agentsigner.Signer,
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
) (*Service, error) {
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
	if config.resolver == nil {
		return nil, fmt.Errorf(
			"%w: Agent composition resolver is required",
			spec.ErrInvalid,
		)
	}
	if config.interpretations == nil {
		return nil, fmt.Errorf(
			"%w: Agent declaration interpretation registry is required",
			spec.ErrInvalid,
		)
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
		importSigner, err = agentsigner.NewSigner(nil)
		if err != nil {
			return nil, err
		}
	}

	output := &Service{
		roots:               config.roots,
		sources:             sources,
		cat:                 cat,
		discovery:           discovery,
		artifacts:           artifacts,
		resources:           resources,
		managedArtifacts:    managedArtifacts,
		protection:          protection,
		definitions:         definitions,
		managedAgentProfile: managedAgentProfile,
		importSigner:        importSigner,
		declarationResolver: config.resolver,
		interpretations:     config.interpretations,
	}

	plugins, err := pluginAPI.New(
		artifacts,
		cat,
		sources,
		discovery,
		managedArtifacts,
		definitions,
		config.resolver,
		agentDomain.AgentPluginProfile(),
	)
	if err != nil {
		return nil, err
	}

	output.plugins = plugins
	return output, nil
}

func (a *Service) requireMutable(
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
