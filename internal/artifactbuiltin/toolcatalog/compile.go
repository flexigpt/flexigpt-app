package toolcatalog

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
	toolDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/tool/domain"
)

func Compile(
	ctx context.Context,
	temporaryDirectory string,
	goTools toolDomain.GoToolLocator,
	registry *interpretation.Registry,
) (installModel.CompiledPackageSet, error) {
	packages, err := artifactbuiltin.EmbeddedToolPackages()
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}

	prepared, err := PreparePackages(ctx, packages, goTools, registry)
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}

	return artifactsetup.CompileBuiltInPackageSet(ctx, temporaryDirectory, artifactsetup.CompileConfig{
		SetName:         topology.BuiltinEmbeddedPackageTools,
		SchemaVersion:   toolDomain.HydrationSchemaVersion,
		InstallerName:   toolDomain.BuiltInInstallerName,
		Interpretations: registry,
		Packages:        packageInputs(prepared),
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
