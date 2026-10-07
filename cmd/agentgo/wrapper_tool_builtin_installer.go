package main

import (
	"errors"
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin/toolcatalog"
	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/llmsupport"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	toolDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool/domain"
)

func NewToolBuiltInInstaller(
	hydrator installModel.CompiledHydrationCoordinator,
) (installFlow.HydrationInstaller, error) {
	if hydrator == nil {
		return nil, errors.New("tool generated catalog installer hydrator is required")
	}
	return toolcatalog.NewInstaller(toolcatalog.InstallerDependencies{
		Hydrator: hydrator,
	})
}

// ToolBuiltinCatalog binds FlexiGPT-generated Tool content to the explicit
// Tool-family configuration consumed by LLM Artifactory. The Tool family does
// not import artifactbuiltin or application topology directly.
func ToolBuiltinCatalog() (toolDomain.BuiltinCatalog, error) {
	source, err := topology.BuiltinSource(
		topology.BuiltinSourceRolePackages,
	)
	if err != nil {
		return toolDomain.BuiltinCatalog{}, err
	}

	support, err := llmsupport.Tool()
	if err != nil {
		return toolDomain.BuiltinCatalog{}, err
	}

	index, err := toolcatalog.GeneratedToolPluginIndex()
	if err != nil {
		return toolDomain.BuiltinCatalog{}, fmt.Errorf(
			"load generated Tool Plugin index: %w",
			err,
		)
	}

	value := toolDomain.BuiltinCatalog{
		RootID:        topology.BuiltinRootID(),
		SourceID:      source.ID,
		ToolPackage:   support.ToolPackage,
		PluginProfile: support.PluginProfile,
		PluginByTool:  index,
	}
	if err := value.Validate(); err != nil {
		return toolDomain.BuiltinCatalog{}, err
	}
	return value, nil
}
