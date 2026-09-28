package builtin

import (
	_ "embed"
	"fmt"
	"maps"
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi/topology"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	toolDomain "github.com/flexigpt/flexigpt-app/internal/tool/store/domain"
)

//go:embed catalog_generated.json
var generatedCatalogJSON []byte

var (
	generatedCatalogOnce sync.Once
	generatedCatalog     topology.CompiledPackageSet
	errGeneratedCatalog  error

	generatedCatalogFingerprintOnce sync.Once
	generatedCatalogFingerprint     cryptoutil.Digest

	generatedCollectionIndexOnce sync.Once
	generatedCollectionIndex     map[basespec.LogicalName]basespec.LogicalName
	errGeneratedCollectionIndex  error
)

func generatedCatalogValue() (
	topology.CompiledPackageSet,
	error,
) {
	generatedCatalogOnce.Do(func() {
		generatedCatalog, errGeneratedCatalog = builtin.DecodeGeneratedPackageSet(generatedCatalogJSON)
	})
	if errGeneratedCatalog != nil {
		return topology.CompiledPackageSet{}, errGeneratedCatalog
	}
	return generatedCatalog, nil
}

func GeneratedCatalogSet() (topology.CompiledPackageSet, error) {
	value, err := generatedCatalogValue()
	if err != nil {
		return topology.CompiledPackageSet{}, err
	}
	return value.Clone(), nil
}

func GeneratedCatalogFingerprint() cryptoutil.Digest {
	generatedCatalogFingerprintOnce.Do(func() {
		value, err := generatedCatalogValue()
		if err != nil {
			return
		}
		_, generatedCatalogFingerprint, _ = builtin.CanonicalGeneratedPackageSet(value)
	})
	return generatedCatalogFingerprint
}

// GeneratedToolCollectionIndex maps a generated Tool name to its generated
// Tool Collection name. It is derived once from the compile-time catalog and
// avoids rediscovering membership by listing and decoding every Collection at
// runtime.
func GeneratedToolCollectionIndex() (
	map[basespec.LogicalName]basespec.LogicalName,
	error,
) {
	generatedCollectionIndexOnce.Do(func() {
		set, err := generatedCatalogValue()
		if err != nil {
			errGeneratedCollectionIndex = err
			return
		}

		collections := make(
			map[basespec.Locator]basespec.LogicalName,
		)
		for _, packageValue := range set.Packages {
			if packageValue.Address.Kind !=
				toolDomain.ToolCollectionPackageKind {
				continue
			}
			collections[packageValue.EmbeddedRoot] = packageValue.Address.Name
		}

		index := make(
			map[basespec.LogicalName]basespec.LogicalName,
		)
		for _, packageValue := range set.Packages {
			if packageValue.Address.Kind != toolDomain.ToolPackageKind {
				continue
			}
			collectionName, found := collections[packageValue.EmbeddedRoot]
			if !found {
				errGeneratedCollectionIndex = fmt.Errorf(
					"generated Tool %q has no generated Tool Collection",
					packageValue.Address.Name,
				)
				return
			}
			if previous, duplicate := index[packageValue.Address.Name]; duplicate {
				errGeneratedCollectionIndex = fmt.Errorf(
					"generated Tool %q belongs to both %q and %q",
					packageValue.Address.Name,
					previous,
					collectionName,
				)
				return
			}
			index[packageValue.Address.Name] = collectionName
		}
		generatedCollectionIndex = index
	})
	if errGeneratedCollectionIndex != nil {
		return nil, errGeneratedCollectionIndex
	}
	return maps.Clone(generatedCollectionIndex), nil
}
