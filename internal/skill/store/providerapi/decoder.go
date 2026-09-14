package providerapi

import (
	"context"
	"path"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/providerapi"
	skillDomain "github.com/flexigpt/flexigpt-app/internal/skill/store/domain"
)

// Decoder adapts Agent Skills SKILL.md source packages to generic Skill
// Artifact Store Definitions.
type Decoder struct{}

func NewDecoder() *Decoder {
	return &Decoder{}
}

func (*Decoder) ID() basespec.DecoderID {
	return skillDomain.MarkdownDecoderID
}

func (*Decoder) Revision() string {
	return skillDomain.SkillSchemaVersion
}

func (*Decoder) Recognize(
	_ context.Context,
	candidate providerapi.Candidate,
) providerapi.Recognition {
	if basespec.Locator(path.Base(string(candidate.Locator))) !=
		skillDomain.SkillDefinitionFileName {
		return providerapi.RecognitionNone
	}
	if candidate.RequestsDecoder(skillDomain.MarkdownDecoderID) {
		return providerapi.RecognitionPreferred
	}
	return providerapi.RecognitionPossible
}

func (*Decoder) Decode(
	_ context.Context,
	candidate providerapi.Candidate,
) ([]providerapi.Decoded, []diagnostic.Diagnostic) {
	if basespec.Locator(path.Base(string(candidate.Locator))) !=
		skillDomain.SkillDefinitionFileName {
		return nil, nil
	}

	expectedName := expectedSkillName(candidate.Locator)
	value, warnings, err := skillDomain.DecodeSkillDocument(
		candidate.Content,
		expectedName,
	)
	if err != nil {
		return nil, []diagnostic.Diagnostic{{
			Severity: diagnostic.SeverityError,
			Code:     "agent.skill.invalid",
			Message:  diagnostic.BoundedMessage(err.Error()),
			Location: &diagnostic.Location{
				Locator: candidate.Locator,
			},
		}}
	}

	for index := range warnings {
		warnings[index].Location = &diagnostic.Location{
			Locator: candidate.Locator,
		}
	}

	return []providerapi.Decoded{{
		Definition: value,
	}}, warnings
}
