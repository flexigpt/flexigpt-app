package builtin

import (
	_ "embed"
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/install/topology"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

//go:embed catalog_generated.json
var generatedCatalogJSON []byte

var (
	generatedCatalogOnce sync.Once
	generatedCatalog     topology.CompiledPackageSet
	errGeneratedCatalog  error

	generatedCatalogFingerprintOnce sync.Once
	generatedCatalogFingerprint     cryptoutil.Digest
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
