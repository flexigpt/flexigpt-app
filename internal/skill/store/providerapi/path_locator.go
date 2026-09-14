package providerapi

import (
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/locatorpath"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	skillDomain "github.com/flexigpt/flexigpt-app/internal/skill/store/domain"
)

func NewPathLocatorResolver() providerapi.LocatorResolverFactory {
	return locatorpath.NewFactory(
		[]artifact.ArtifactKind{
			skillDomain.SkillArtifactKind,
		},
		skillPathPlanner{},
	)
}

type skillPathPlanner struct{}

func (skillPathPlanner) PlanPath(
	target basespec.Locator,
	discovery source.DiscoverySpec,
) (source.DiscoverySpec, []basespec.Locator, error) {
	output := discovery.Clone()
	base := path.Base(string(target))
	extension := strings.ToLower(path.Ext(base))

	if base == string(skillDomain.SkillDefinitionFileName) ||
		extension == ".json" ||
		extension == ".yaml" ||
		extension == ".yml" {
		if !containsLocator(output.ExplicitLocators, target) {
			output.ExplicitLocators = append(
				output.ExplicitLocators,
				target,
			)
		}
		if base == string(skillDomain.SkillDefinitionFileName) {
			if len(output.AllowedDecoderIDs) != 0 &&
				!containsDecoder(
					output.AllowedDecoderIDs,
					skillDomain.MarkdownDecoderID,
				) {
				return source.DiscoverySpec{}, nil, fmt.Errorf(
					"%w: Skill path requires decoder %q",
					basespec.ErrDecoderUnavailable,
					skillDomain.MarkdownDecoderID,
				)
			}
			output.DecoderHints = appendSkillDecoderHint(
				output.DecoderHints,
				target,
				false,
			)
		}
		return output, []basespec.Locator{target}, nil
	}

	if len(output.AllowedDecoderIDs) != 0 &&
		!containsDecoder(
			output.AllowedDecoderIDs,
			skillDomain.MarkdownDecoderID,
		) {
		return source.DiscoverySpec{}, nil, fmt.Errorf(
			"%w: Skill path requires decoder %q",
			basespec.ErrDecoderUnavailable,
			skillDomain.MarkdownDecoderID,
		)
	}

	root := source.DirectoryRoot{
		Root:      target,
		Recursive: false,
		IncludePatterns: []string{
			string(skillDomain.SkillDefinitionFileName),
		},
	}
	if !containsSkillRoot(output.DirectoryRoots, root) {
		output.DirectoryRoots = append(output.DirectoryRoots, root)
	}
	output.DecoderHints = appendSkillDecoderHint(
		output.DecoderHints,
		target,
		false,
	)

	skillLocator := basespec.Locator(
		path.Join(string(target), string(skillDomain.SkillDefinitionFileName)),
	)
	if target == "." {
		skillLocator = skillDomain.SkillDefinitionFileName
	}
	return output, []basespec.Locator{skillLocator}, nil
}

func containsLocator(
	values []basespec.Locator,
	target basespec.Locator,
) bool {
	return slices.Contains(values, target)
}

func containsDecoder(
	values []basespec.DecoderID,
	target basespec.DecoderID,
) bool {
	return slices.Contains(values, target)
}

func containsSkillRoot(
	values []source.DirectoryRoot,
	target source.DirectoryRoot,
) bool {
	for _, value := range values {
		if value.Root == target.Root &&
			value.Recursive == target.Recursive &&
			len(value.IncludePatterns) == 1 &&
			value.IncludePatterns[0] ==
				string(skillDomain.SkillDefinitionFileName) &&
			len(value.ExcludePatterns) == 0 {
			return true
		}
	}
	return false
}

func appendSkillDecoderHint(
	values []source.DecoderHint,
	locator basespec.Locator,
	recursive bool,
) []source.DecoderHint {
	for index := range values {
		if values[index].Locator != locator ||
			values[index].Recursive != recursive {
			continue
		}
		if !containsDecoder(
			values[index].DecoderIDs,
			skillDomain.MarkdownDecoderID,
		) {
			values[index].DecoderIDs = append(
				values[index].DecoderIDs,
				skillDomain.MarkdownDecoderID,
			)
		}
		return values
	}
	return append(values, source.DecoderHint{
		Locator:   locator,
		Recursive: recursive,
		DecoderIDs: []basespec.DecoderID{
			skillDomain.MarkdownDecoderID,
		},
	})
}
