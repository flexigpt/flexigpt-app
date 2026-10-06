package skillcatalog

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	installFlow "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	coreinterpretation "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
	skillSource "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/source"
	skillmarkdown "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/sourceformat/markdown"
)

func Compile(
	ctx context.Context,
	temporaryDirectory string,
	registry *coreinterpretation.Registry,
) (installModel.CompiledPackageSet, error) {
	packages, err := artifactbuiltin.EmbeddedSkillPackages()
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}

	prepared, err := PreparePackages(ctx, packages, registry)
	if err != nil {
		return installModel.CompiledPackageSet{}, err
	}

	// Skill package admission needs the SKILL.md source-format adapter in
	// addition to canonical declaration decoding.
	return artifactsetup.CompileBuiltInPackageSet(ctx, temporaryDirectory, artifactsetup.CompileConfig{
		SetName:            topology.BuiltinEmbeddedPackageSkills,
		SchemaVersion:      skillSource.HydrationSchemaVersion,
		InstallerName:      skillSource.BuiltInInstallerName,
		Interpretations:    registry,
		AdditionalDecoders: []ingest.Decoder{skillmarkdown.NewDecoder()},
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
