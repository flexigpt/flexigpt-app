package modelcatalog

import (
	_ "embed"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
)

//go:embed catalog_generated.json
var generatedCatalogJSON []byte

var generatedCatalog = artifactbuiltin.NewGeneratedPackageCatalog(
	generatedCatalogJSON,
)

func generatedCatalogValue() (
	installModel.CompiledPackageSet,
	error,
) {
	return generatedCatalog.PackageSet()
}
