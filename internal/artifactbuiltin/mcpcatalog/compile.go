package mcpcatalog

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
	mcpDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/domain"
	mcpconfig "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/sourceformat/config"
)

func Compile(
	ctx context.Context,
	temporaryDirectory string,
	registry *coreinterpretation.Registry,
) (installModel.CompiledPackageSet, error) {
	packages, err := artifactbuiltin.EmbeddedMCPPackages()
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}

	prepared, err := PreparePackages(ctx, packages, registry)
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}

	// MCP package admission needs the standard MCP configuration adapter in
	// addition to canonical declaration decoding.
	return artifactsetup.CompileBuiltInPackageSet(ctx, temporaryDirectory, artifactsetup.CompileConfig{
		SetName:            topology.BuiltinEmbeddedPackageMCPs,
		SchemaVersion:      mcpDomain.HydrationSchemaVersion,
		InstallerName:      mcpDomain.BuiltInInstallerName,
		Interpretations:    registry,
		AdditionalDecoders: []ingest.Decoder{mcpconfig.NewDecoder()},
		Packages:           packageInputs(prepared),
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
			DocumentFile: value.DocumentFile,
			Files:        value.PackageFiles,
			Expectations: expectations,
		})
	}

	return output
}
