package agentcatalog

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/llmsupport"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	agentDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/agent/domain"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
)

func Compile(
	ctx context.Context,
	temporaryDirectory string,
	interpretations *coreinterpretation.Registry,
) (installModel.CompiledPackageSet, error) {
	agentSupport, err := llmsupport.Agent()
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}

	packages, err := artifactbuiltin.EmbeddedAgentPackages()
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}

	prepared, err := PreparePackages(
		ctx,
		packages,
		interpretations,
		agentSupport.Documents,
	)
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}

	return artifactsetup.CompileBuiltInPackageSet(ctx, temporaryDirectory, artifactsetup.CompileConfig{
		SetName:         topology.BuiltinEmbeddedPackageAgents,
		SchemaVersion:   agentDomain.HydrationSchemaVersion,
		InstallerName:   agentDomain.BuiltInInstallerName,
		Interpretations: interpretations,
		Packages:        packageInputs(prepared),
	})
}

func packageInputs(
	values []PreparedPackage,
) []installFlow.PackageInput {
	output := make([]installFlow.PackageInput, 0, len(values))

	for _, value := range values {
		expectations := make(
			[]installFlow.Expectation,
			0,
			len(value.Expectations),
		)
		for _, expected := range value.Expectations {
			expectations = append(expectations, installFlow.Expectation{
				Locator:          expected.Locator,
				Subresource:      expected.Subresource,
				Kind:             expected.Kind,
				LogicalName:      expected.LogicalName,
				LogicalVersion:   expected.LogicalVersion,
				DefinitionDigest: expected.DefinitionDigest,
			})
		}

		output = append(output, installFlow.PackageInput{
			EmbeddedRoot: value.EmbeddedPackageRoot,
			Address:      value.PackageAddress,
			DocumentFile: value.PluginDocumentFile,
			Files:        value.PackageFiles,
			Expectations: expectations,
		})
	}

	return output
}
