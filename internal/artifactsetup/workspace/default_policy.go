package workspace

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin/workspacecatalog/defaultpolicy"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/decoder"
	workspaceConsumerAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace/consumerapi"
)

// DefaultWorkspaceConfig supplies the complete application-selected Workspace
// consumer configuration. Runtime-specific composition and MCP ports remain
// unset because wrappers bind those runtime capabilities explicitly.
func DefaultWorkspaceConfig() (
	workspaceConsumerAPI.Config,
	error,
) {
	source, err := DefaultPolicySource()
	if err != nil {
		return workspaceConsumerAPI.Config{}, err
	}

	config := workspaceConsumerAPI.DefaultConfig()
	config.DefaultPolicySource = source
	return config, nil
}

// DefaultPolicySource binds application-shipped Workspace policy content to
// the explicit embedded Source configuration consumed by Workspace local
// deployment behavior.
func DefaultPolicySource() (
	workspaceConsumerAPI.DefaultPolicySource,
	error,
) {
	policy, err := defaultpolicy.Load()
	if err != nil {
		return workspaceConsumerAPI.DefaultPolicySource{}, err
	}

	value := workspaceConsumerAPI.DefaultPolicySource{
		ProviderKey: defaultpolicy.ProviderKey,
		Root:        spec.Locator(defaultpolicy.PolicyRoot),
		Locator:     spec.Locator(defaultpolicy.PolicyLocator),
		DecoderID:   decoder.YAMLDecoderID,
		Policy:      policy,
	}
	if err := value.Validate(); err != nil {
		return workspaceConsumerAPI.DefaultPolicySource{}, err
	}
	return value, nil
}
