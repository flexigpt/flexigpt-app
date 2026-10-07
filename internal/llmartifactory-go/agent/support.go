package agent

import (
	"fmt"
	"maps"
	"strings"

	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/support"
	pluginAPI "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin"
)

type Support struct {
	BuiltinRoot    rootModel.RootID
	ManagedPackage support.PackageLayout
	Documents      support.Documents
	PluginProfile  pluginAPI.Profile
	ImportFormats  map[string]ImportFormat
}

// Validate establishes the application-supplied Agent support contract once at
// construction. Agent operations subsequently consume this immutable support
// without rechecking configuration dependencies.
func (s Support) Validate() error {
	if err := s.BuiltinRoot.Validate(); err != nil {
		return err
	}
	if err := s.ManagedPackage.Validate(); err != nil {
		return err
	}
	if err := s.Documents.Validate(); err != nil {
		return err
	}
	if s.Documents.Default != s.ManagedPackage.Document {
		return fmt.Errorf(
			"%w: Agent managed package document differs from Agent document support",
			spec.ErrInvalid,
		)
	}
	if err := s.PluginProfile.Validate(); err != nil {
		return err
	}
	if s.PluginProfile.BuiltinRoot != s.BuiltinRoot {
		return fmt.Errorf(
			"%w: Agent Plugin profile built-in Root differs from Agent support",
			spec.ErrInvalid,
		)
	}
	if s.PluginProfile.Source == nil {
		return fmt.Errorf(
			"%w: Agent Plugin profile requires a managed Source",
			spec.ErrInvalid,
		)
	}
	if len(s.ImportFormats) == 0 {
		return fmt.Errorf(
			"%w: Agent import format support is empty",
			spec.ErrInvalid,
		)
	}

	for extension, format := range s.ImportFormats {
		if extension == "" ||
			!strings.HasPrefix(extension, ".") ||
			strings.ToLower(extension) != extension {
			return fmt.Errorf(
				"%w: Agent import extension %q is invalid",
				spec.ErrInvalid,
				extension,
			)
		}
		switch format {
		case ImportFormatJSON, ImportFormatYAML:
		default:
			return fmt.Errorf(
				"%w: Agent import extension %q has unsupported format %q",
				spec.ErrInvalid,
				extension,
				format,
			)
		}
	}
	return nil
}

func (s Support) Clone() Support {
	output := s
	output.PluginProfile = s.PluginProfile.Clone()
	output.ImportFormats = maps.Clone(s.ImportFormats)
	output.Documents = s.Documents.Clone()
	return output
}
