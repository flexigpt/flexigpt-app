package toolcatalog

import (
	_ "embed"
	"fmt"
	"maps"
	"sync"

	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	toolDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool/domain"
)

//go:embed catalog_generated.json
var generatedCatalogJSON []byte

var (
	generatedCatalogOnce sync.Once
	generatedCatalog     *installFlow.PreloadedGeneratedPackageSet
	errGeneratedCatalog  error

	generatedPluginIndexOnce sync.Once
	generatedPluginIndex     map[spec.LogicalName]spec.LogicalName
	errGeneratedPluginIndex  error
)

func generatedCatalogPreload() (
	*installFlow.PreloadedGeneratedPackageSet,
	error,
) {
	generatedCatalogOnce.Do(func() {
		generatedCatalog, errGeneratedCatalog = installFlow.PreloadGeneratedPackageSet(
			generatedCatalogJSON,
		)
	})
	if errGeneratedCatalog != nil {
		return nil, errGeneratedCatalog
	}
	return generatedCatalog, nil
}

func generatedCatalogValue() (
	installModel.CompiledPackageSet,
	error,
) {
	preloaded, err := generatedCatalogPreload()
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}
	return preloaded.PackageSet()
}

func GeneratedCatalogSet() (installModel.CompiledPackageSet, error) {
	return generatedCatalogValue()
}

func GeneratedCatalogFingerprint() cryptoutil.Digest {
	preloaded, err := generatedCatalogPreload()
	if err != nil {
		return ""
	}
	return preloaded.Fingerprint()
}

// GeneratedToolPluginIndex maps a generated Tool name to its generated
// Tool Plugin name. It is derived once from the compile-time catalog and
// avoids rediscovering membership by listing and decoding every Plugin at
// runtime.
func GeneratedToolPluginIndex() (
	map[spec.LogicalName]spec.LogicalName,
	error,
) {
	generatedPluginIndexOnce.Do(func() {
		set, err := generatedCatalogValue()
		if err != nil {
			errGeneratedPluginIndex = err
			return
		}

		plugins := make(
			map[spec.Locator]spec.LogicalName,
		)
		for _, packageValue := range set.Packages {
			if packageValue.Address.Kind !=
				toolDomain.ToolPluginPackageKind {
				continue
			}
			plugins[packageValue.EmbeddedRoot] = packageValue.Address.Name
		}

		index := make(
			map[spec.LogicalName]spec.LogicalName,
		)
		for _, packageValue := range set.Packages {
			if packageValue.Address.Kind != toolDomain.ToolPackageKind {
				continue
			}
			pluginName, found := plugins[packageValue.EmbeddedRoot]
			if !found {
				errGeneratedPluginIndex = fmt.Errorf(
					"generated Tool %q has no generated Tool Plugin",
					packageValue.Address.Name,
				)
				return
			}
			if previous, duplicate := index[packageValue.Address.Name]; duplicate {
				errGeneratedPluginIndex = fmt.Errorf(
					"generated Tool %q belongs to both %q and %q",
					packageValue.Address.Name,
					previous,
					pluginName,
				)
				return
			}
			index[packageValue.Address.Name] = pluginName
		}
		generatedPluginIndex = index
	})
	if errGeneratedPluginIndex != nil {
		return nil, errGeneratedPluginIndex
	}
	return maps.Clone(generatedPluginIndex), nil
}
