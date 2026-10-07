package tool

import (
	"fmt"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/support"
	pluginAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
)

type Support struct {
	ToolPackage     support.PackageLayout
	Documents       support.Documents
	PluginDocuments support.Documents
	PluginProfile   pluginAPI.Profile
}

func (s Support) Validate() error {
	if err := s.ToolPackage.Validate(); err != nil {
		return err
	}
	if err := s.Documents.Validate(); err != nil {
		return err
	}
	if err := s.PluginDocuments.Validate(); err != nil {
		return err
	}
	if s.Documents.Default != s.ToolPackage.Document {
		return fmt.Errorf(
			"%w: Tool package document differs from Tool document support",
			spec.ErrInvalid,
		)
	}
	if err := s.PluginProfile.Validate(); err != nil {
		return err
	}
	if s.PluginDocuments.Default != s.PluginProfile.Package.Document {
		return fmt.Errorf(
			"%w: Tool Plugin package document differs from Tool Plugin document support",
			spec.ErrInvalid,
		)
	}
	return nil
}

func (s Support) Clone() Support {
	output := s
	output.PluginProfile = s.PluginProfile.Clone()
	output.Documents = s.Documents.Clone()
	output.PluginDocuments = s.PluginDocuments.Clone()
	return output
}
