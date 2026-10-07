package skill

import (
	"fmt"

	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/support"
	pluginAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
)

type Support struct {
	BuiltinRoot   rootModel.RootID
	Documents     support.Documents
	Package       support.PackageLayout
	Discovery     sourceModel.DiscoverySpec
	PluginProfile pluginAPI.Profile
}

func (s Support) Validate() error {
	if err := s.BuiltinRoot.Validate(); err != nil {
		return err
	}
	if err := s.Documents.Validate(); err != nil {
		return err
	}
	if err := s.Package.Validate(); err != nil {
		return err
	}
	if s.Documents.Default != s.Package.Document {
		return fmt.Errorf(
			"%w: Skill package document differs from Skill document support",
			spec.ErrInvalid,
		)
	}
	if err := s.Discovery.Validate(); err != nil {
		return err
	}
	if err := s.PluginProfile.Validate(); err != nil {
		return err
	}
	if s.PluginProfile.BuiltinRoot != s.BuiltinRoot {
		return fmt.Errorf(
			"%w: Skill Plugin profile built-in Root differs from Skill support",
			spec.ErrInvalid,
		)
	}
	if s.PluginProfile.Source == nil {
		return fmt.Errorf(
			"%w: Skill Plugin profile requires a managed Source",
			spec.ErrInvalid,
		)
	}
	return nil
}

func (s Support) Clone() Support {
	output := s
	output.Documents = s.Documents.Clone()
	output.Discovery = s.Discovery.Clone()
	output.PluginProfile = s.PluginProfile.Clone()
	return output
}
