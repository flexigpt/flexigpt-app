package builtin

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	toolDomain "github.com/flexigpt/flexigpt-app/internal/tool/store/domain"
)

func Compile(
	ctx context.Context,
	temporaryDirectory string,
	goTools toolDomain.GoToolLocator,
) (installModel.CompiledPackageSet, error) {
	packages, err := builtin.EmbeddedToolPackages()
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}

	prepared, err := PreparePackages(ctx, packages, goTools)
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}

	return builtin.Compile(ctx, temporaryDirectory, builtin.Config{
		SetName:       documentTopology.BuiltinEmbeddedPackageTools,
		SchemaVersion: toolDomain.HydrationSchemaVersion,
		InstallerName: toolDomain.BuiltInInstallerName,
		Packages:      packageInputs(prepared),
	})
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
