package mcpcatalog

import (
	_ "embed"
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

//go:embed catalog_generated.json
var generatedCatalogJSON []byte

var (
	generatedCatalogOnce sync.Once
	generatedCatalog     installModel.CompiledPackageSet
	errGeneratedCatalog  error

	generatedCatalogFingerprintOnce sync.Once
	generatedCatalogFingerprint     cryptoutil.Digest
)

func generatedCatalogValue() (
	installModel.CompiledPackageSet,
	error,
) {
	generatedCatalogOnce.Do(func() {
		generatedCatalog, errGeneratedCatalog = install.DecodeGeneratedPackageSet(generatedCatalogJSON)
	})
	if errGeneratedCatalog != nil {
		return installModel.CompiledPackageSet{}, errGeneratedCatalog
	}
	return generatedCatalog, nil
}

func GeneratedCatalogSet() (installModel.CompiledPackageSet, error) {
	value, err := generatedCatalogValue()
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}
	return value.Clone(), nil
}

func GeneratedCatalogFingerprint() cryptoutil.Digest {
	generatedCatalogFingerprintOnce.Do(func() {
		value, err := generatedCatalogValue()
		if err != nil {
			return
		}
		_, generatedCatalogFingerprint, _ = install.CanonicalGeneratedPackageSet(value)
	})
	return generatedCatalogFingerprint
}
