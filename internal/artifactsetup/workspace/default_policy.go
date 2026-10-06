package workspace

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin/workspacecatalog/defaultpolicy"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	coredecoder "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/decoder"
	workspaceAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workspace"
)

// DefaultWorkspaceConfig supplies the complete application-selected Workspace
// consumer configuration. Runtime-specific composition and MCP ports remain
// unset because wrappers bind those runtime capabilities explicitly.
func DefaultWorkspaceConfig() (
	workspaceAPI.Config,
	error,
) {
	source, err := DefaultPolicySource()
	if err != nil {
		return workspaceAPI.Config{}, err
	}

	config := workspaceAPI.DefaultConfig()
	config.DefaultPolicySource = source
	return config, nil
}

// DefaultPolicySource binds application-shipped Workspace policy content to
// the explicit embedded Source configuration consumed by Workspace local
// deployment behavior.
func DefaultPolicySource() (
	workspaceAPI.DefaultPolicySource,
	error,
) {
	policy, err := defaultpolicy.Load()
	if err != nil {
		return workspaceAPI.DefaultPolicySource{}, err
	}

	value := workspaceAPI.DefaultPolicySource{
		ProviderKey: defaultpolicy.ProviderKey,
		Root:        spec.Locator(defaultpolicy.PolicyRoot),
		Locator:     spec.Locator(defaultpolicy.PolicyLocator),
		DecoderID:   coredecoder.YAMLDecoderID,
		Policy:      policy,
	}
	if err := value.Validate(); err != nil {
		return workspaceAPI.DefaultPolicySource{}, err
	}
	return value, nil
}
