package modelcatalog

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup"
	modelDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/model/domain"
)

func Compile(
	ctx context.Context,
	temporaryDirectory string,
	prepared []PreparedPackage,
) (installModel.CompiledPackageSet, error) {
	values, err := NormalizePreparedPackages(prepared)
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}

	return artifactsetup.CompileBuiltInPackageSet(
		ctx,
		temporaryDirectory,
		artifactsetup.CompileConfig{
			SetName:       generatedCatalogName,
			SchemaVersion: modelDomain.HydrationSchemaVersion,
			InstallerName: modelDomain.BuiltInInstallerName,
			Packages:      packageInputs(values),
		},
	)
}

func packageInputs(
	values []PreparedPackage,
) []install.PackageInput {
	output := make([]install.PackageInput, 0, len(values))
	for _, value := range values {
		output = append(output, install.PackageInput{
			EmbeddedRoot: value.EmbeddedPackageRoot,
			Address:      value.Address,
			DocumentFile: value.DocumentFile,
			Files:        value.PackageFiles,
			Expectations: []install.Expectation{{
				Locator:          value.DocumentFile,
				Kind:             value.ExpectedKind,
				LogicalName:      value.ExpectedLogicalName,
				LogicalVersion:   value.ExpectedLogicalVersion,
				DefinitionDigest: value.ExpectedDefinition,
			}},
		})
	}
	return output
}
