package builtin

import (
	"context"

	agentDomain "github.com/flexigpt/flexigpt-app/internal/agent/store/domain"
	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/builtin"
	documentTopology "github.com/flexigpt/flexigpt-app/internal/artifactcontract/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/install/topology"
)

func Compile(
	ctx context.Context,
	temporaryDirectory string,
) (topology.CompiledPackageSet, error) {
	packages, err := builtin.EmbeddedAgentPackages()
	if err != nil {
		return topology.CompiledPackageSet{}, err
	}

	prepared, err := PreparePackages(ctx, packages)
	if err != nil {
		return topology.CompiledPackageSet{}, err
	}

	return builtin.Compile(ctx, temporaryDirectory, builtin.Config{
		SetName:       documentTopology.BuiltinEmbeddedPackageAgents,
		SchemaVersion: agentDomain.HydrationSchemaVersion,
		InstallerName: agentDomain.BuiltInInstallerName,
		Packages:      packageInputs(prepared),
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
				LogicalVersion:   expected.LogicalVersion,
				DefinitionDigest: expected.DefinitionDigest,
			})
		}

		output = append(output, builtin.PackageInput{
			EmbeddedRoot: value.EmbeddedPackageRoot,
			Address:      value.PackageAddress,
			DocumentFile: value.PluginDocumentFile,
			Files:        value.PackageFiles,
			Expectations: expectations,
		})
	}

	return output
}
