package markdown

import (
	"context"
	"path"

	ingestModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest/model"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
	skillSource "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/source"
)

// Decoder adapts Agent Skills SKILL.md source packages to generic Skill
// Artifact Store Definitions.
type Decoder struct{}

func NewDecoder() *Decoder {
	return &Decoder{}
}

func (*Decoder) ID() spec.DecoderID {
	return skillSource.MarkdownDecoderID
}

func (*Decoder) Revision() string {
	return skillSource.SkillSchemaVersion
}

func (*Decoder) Recognize(
	_ context.Context,
	candidate ingestModel.Candidate,
) ingestModel.Recognition {
	if !skillSource.IsSkillDefinitionFile(candidate.Locator) {
		return ingestModel.RecognitionNone
	}
	if candidate.RequestsDecoder(skillSource.MarkdownDecoderID) {
		return ingestModel.RecognitionPreferred
	}
	return ingestModel.RecognitionPossible
}

func (*Decoder) Decode(
	_ context.Context,
	candidate ingestModel.Candidate,
) ([]ingestModel.Decoded, []diagnostic.Diagnostic) {
	if !skillSource.IsSkillDefinitionFile(candidate.Locator) {
		return nil, nil
	}

	expectedName := expectedSkillName(candidate.Locator)
	value, warnings, err := skillSource.DecodeSkillDocument(
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

	return []ingestModel.Decoded{{
		Definition: value,
	}}, warnings
}

func expectedSkillName(locator spec.Locator) string {
	parent := path.Dir(string(locator))
	if parent == "." {
		return ""
	}
	if address, err := managedpackageModel.ParseManagedPackageAddressDirectory(
		spec.Locator(parent),
	); err == nil &&
		address.Kind == skillSource.ManagedSkillPackageKind {
		return string(address.Name)
	}
	return path.Base(parent)
}
