package mcp

import (
	"fmt"

	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/support"
	pluginAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
)

type Support struct {
	BuiltinRoot           rootModel.RootID
	BuiltinPackageSource  sourceModel.SourceID
	BuiltinPluginDocument support.Document
	ServerPackage         support.PackageLayout
	PolicyPackage         support.PackageLayout
	PluginProfile         pluginAPI.Profile
}

func (s Support) Validate() error {
	if err := s.BuiltinRoot.Validate(); err != nil {
		return err
	}
	if err := s.BuiltinPackageSource.Validate(); err != nil {
		return err
	}
	if err := s.BuiltinPluginDocument.Validate(); err != nil {
		return err
	}
	if err := s.ServerPackage.Validate(); err != nil {
		return err
	}
	if err := s.PolicyPackage.Validate(); err != nil {
		return err
	}
	if err := s.PluginProfile.Validate(); err != nil {
		return err
	}
	if s.PluginProfile.BuiltinRoot != s.BuiltinRoot {
		return fmt.Errorf(
			"%w: MCP Plugin profile built-in Root differs from MCP support",
			spec.ErrInvalid,
		)
	}
	if s.PluginProfile.Source == nil {
		return fmt.Errorf(
			"%w: MCP Plugin profile requires a managed Source",
			spec.ErrInvalid,
		)
	}
	return nil
}

func (s Support) Clone() Support {
	output := s
	output.PluginProfile = s.PluginProfile.Clone()
	return output
}
