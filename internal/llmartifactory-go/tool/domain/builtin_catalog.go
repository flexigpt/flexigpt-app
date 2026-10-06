package domain

import (
	"maps"

	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	sourceModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

// BuiltinCatalog is application-supplied generated Tool inventory.
//
// It identifies the protected Tool Artifact namespace and maps every
// generated Tool logical name to its generated Tool Plugin logical name.
//
// The catalog intentionally contains both Go Tools and provider-native SDK
// Tools. The distinction belongs to the Tool declaration implementation and
// runtime hydration boundary, not to Plugin membership or Artifact identity.
// It contains no embedded filesystem, generated JSON payload, Tool invocation
// capability, or application topology lookup behavior.
type BuiltinCatalog struct {
	RootID   rootModel.RootID
	SourceID sourceModel.SourceID

	PluginByTool map[spec.LogicalName]spec.LogicalName
}

func (c BuiltinCatalog) Validate() error {
	if err := c.RootID.Validate(); err != nil {
		return err
	}
	if err := c.SourceID.Validate(); err != nil {
		return err
	}
	if c.PluginByTool == nil {
		return spec.ErrInvalid
	}
	for toolName, pluginName := range c.PluginByTool {
		if err := toolName.Validate(); err != nil {
			return err
		}
		if err := pluginName.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (c BuiltinCatalog) Clone() BuiltinCatalog {
	output := c
	output.PluginByTool = maps.Clone(c.PluginByTool)
	return output
}
