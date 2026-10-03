package builtin

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	modelDomain "github.com/flexigpt/flexigpt-app/internal/model/store/domain"
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

	return builtin.Compile(
		ctx,
		temporaryDirectory,
		builtin.Config{
			SetName:       generatedCatalogName,
			SchemaVersion: modelDomain.HydrationSchemaVersion,
			InstallerName: modelDomain.BuiltInInstallerName,
			Packages:      packageInputs(values),
		},
	)
}

func packageInputs(
	values []PreparedPackage,
) []builtin.PackageInput {
	output := make([]builtin.PackageInput, 0, len(values))
	for _, value := range values {
		output = append(output, builtin.PackageInput{
			EmbeddedRoot: value.EmbeddedPackageRoot,
			Address:      value.Address,
			DocumentFile: value.DocumentFile,
			Files:        value.PackageFiles,
			Expectations: []builtin.Expectation{{
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
