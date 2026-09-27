package builtin

import (
	_ "embed"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi/topology"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

//go:embed catalog_generated.json
var generatedCatalogJSON []byte

var generatedCatalog, generatedCatalogFingerprint, generatedCatalogErr = builtin.DecodeGeneratedPackageSet(
	generatedCatalogJSON,
)

func GeneratedCatalogSet() (topology.CompiledPackageSet, error) {
	if generatedCatalogErr != nil {
		return topology.CompiledPackageSet{}, generatedCatalogErr
	}
	return generatedCatalog.Clone(), nil
}

func GeneratedCatalogFingerprint() cryptoutil.Digest {
	return generatedCatalogFingerprint
}
