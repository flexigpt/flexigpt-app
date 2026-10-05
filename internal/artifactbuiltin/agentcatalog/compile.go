package agentcatalog

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	agentDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/domain"
)

func Compile(
	ctx context.Context,
	temporaryDirectory string,
) (installModel.CompiledPackageSet, error) {
	packages, err := artifactbuiltin.EmbeddedAgentPackages()
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}

	prepared, err := PreparePackages(ctx, packages)
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}

	return artifactsetup.CompileBuiltInPackageSet(ctx, temporaryDirectory, artifactsetup.CompileConfig{
		SetName:       topology.BuiltinEmbeddedPackageAgents,
		SchemaVersion: agentDomain.HydrationSchemaVersion,
		InstallerName: agentDomain.BuiltInInstallerName,
		Packages:      packageInputs(prepared),
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
			DocumentFile: value.PluginDocumentFile,
			Files:        value.PackageFiles,
			Expectations: expectations,
		})
	}

	return output
}
