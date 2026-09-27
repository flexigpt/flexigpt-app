package builtin

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi/topology"
	toolDomain "github.com/flexigpt/flexigpt-app/internal/tool/store/domain"
)

func Compile(
	ctx context.Context,
	temporaryDirectory string,
	goTools toolDomain.GoToolLocator,
) (topology.CompiledPackageSet, error) {
	packages, err := builtin.EmbeddedToolPackages()
	if err != nil {
		return topology.CompiledPackageSet{}, err
	}

	prepared, err := PreparePackages(ctx, packages, goTools)
	if err != nil {
		return topology.CompiledPackageSet{}, err
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
