package skillcatalog

import (
	"context"

	"github.com/flexigpt/flexigpt-app/internal/artifactbuiltin"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install"
	installModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/flow/install/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup"
	"github.com/flexigpt/flexigpt-app/internal/artifactsetup/topology"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration/interpretation"
	skillDomain "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/domain"
	skillmarkdown "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/sourceformat/markdown"
)

func Compile(
	ctx context.Context,
	temporaryDirectory string,
	registry *interpretation.Registry,
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
		SchemaVersion:      skillDomain.HydrationSchemaVersion,
		InstallerName:      skillDomain.BuiltInInstallerName,
		AdditionalDecoders: []ingest.Decoder{skillmarkdown.NewDecoder()},
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
