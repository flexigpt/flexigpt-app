package builtin

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	skillDomain "github.com/flexigpt/flexigpt-app/internal/skill/store/domain"
	skillProviderAPI "github.com/flexigpt/flexigpt-app/internal/skill/store/providerapi"
)

func Compile(
	ctx context.Context,
	temporaryDirectory string,
) (installModel.CompiledPackageSet, error) {
	packages, err := builtin.EmbeddedSkillPackages()
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}

	prepared, err := PreparePackages(ctx, packages)
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}

	sourceProvider, err := skillProviderAPI.NewProvider()
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}

	return builtin.Compile(ctx, temporaryDirectory, builtin.Config{
		SetName:       documentTopology.BuiltinEmbeddedPackageSkills,
		SchemaVersion: skillDomain.HydrationSchemaVersion,
		InstallerName: skillDomain.BuiltInInstallerName,
		AdditionalProviders: []provider.Provider{
			sourceProvider,
		},
		Packages: packageInputs(prepared),
	})
}

func packageInputs(
	values []PreparedPackage,
) []builtin.PackageInput {
	output := make([]builtin.PackageInput, 0, len(values))

	for _, value := range values {
		expectations := make(
			[]builtin.Expectation,
			0,
			len(value.Expectations),
		)
		for _, expected := range value.Expectations {
			expectations = append(expectations, builtin.Expectation{
				Locator:          expected.Locator,
				Subresource:      expected.Subresource,
				Kind:             expected.Kind,
				LogicalName:      expected.LogicalName,
				DefinitionDigest: expected.DefinitionDigest,
			})
		}

		output = append(output, builtin.PackageInput{
			EmbeddedRoot: value.EmbeddedPackageRoot,
			Address:      value.PackageAddress,
			DocumentFile: value.DocumentFile,
			Files:        value.PackageFiles,
			Expectations: expectations,
		})
	}

	return output
}
