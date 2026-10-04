package builtin

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/mcp/store/domain"
	mcpProviderAPI "github.com/flexigpt/flexigpt-app/internal/mcp/store/providerapi"
)

func Compile(
	ctx context.Context,
	temporaryDirectory string,
) (installModel.CompiledPackageSet, error) {
	packages, err := builtin.EmbeddedMCPPackages()
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}

	prepared, err := PreparePackages(ctx, packages)
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}

	sourceProvider, err := mcpProviderAPI.NewRegistration()
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}

	return builtin.Compile(ctx, temporaryDirectory, builtin.Config{
		SetName:            documentTopology.BuiltinEmbeddedPackageMCPs,
		SchemaVersion:      mcpDomain.HydrationSchemaVersion,
		InstallerName:      mcpDomain.BuiltInInstallerName,
		AdditionalDecoders: sourceProvider.Decoders(),
		Packages:           packageInputs(prepared),
	})
}

func packageInputs(
	values []PreparedPackage,
) []install.PackageInput {
	output := make([]install.PackageInput, 0, len(values))

	for _, value := range values {
		expectations := make(
			[]install.Expectation,
			0,
			len(value.Expectations),
		)
		for _, expected := range value.Expectations {
			expectations = append(expectations, install.Expectation{
				Locator:          expected.Locator,
				Subresource:      expected.Subresource,
				Kind:             expected.Kind,
				LogicalName:      expected.LogicalName,
				LogicalVersion:   expected.LogicalVersion,
				DefinitionDigest: expected.DefinitionDigest,
			})
		}

		output = append(output, install.PackageInput{
			EmbeddedRoot: value.EmbeddedPackageRoot,
			Address:      value.PackageAddress,
			DocumentFile: value.DocumentFile,
			Files:        value.PackageFiles,
			Expectations: expectations,
		})
	}

	return output
}
