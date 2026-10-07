package agentcatalog

import (
	_ "embed"
	"sync"

	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

//go:embed catalog_generated.json
var generatedCatalogJSON []byte

var (
	generatedCatalogOnce sync.Once
	generatedCatalog     *installFlow.PreloadedGeneratedPackageSet
	errGeneratedCatalog  error
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
