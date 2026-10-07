package markdown

import (
	"context"
	"fmt"
	"path"

	ingestModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/ingest/model"
	managedpackageModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/source/managedpackage/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec/diagnostic"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/support"
	skillSource "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/skill/source"
)

// Decoder adapts application-supported Skill package documents to generic
// Skill Artifact Store Definitions.
type Decoder struct {
	candidates support.Candidates
	documents  support.Documents
	layout     support.PackageLayout
}

func NewDecoder(
	candidates support.Candidates,
	documents support.Documents,
	layout support.PackageLayout,
) (*Decoder, error) {
	if err := candidates.Validate(); err != nil {
		return nil, err
	}
	if err := documents.Validate(); err != nil {
		return nil, err
	}
	if err := layout.Validate(); err != nil {
		return nil, err
	}
	if layout.Document != documents.Default {
		return nil, fmt.Errorf(
			"%w: Skill package layout document differs from Skill documents",
			spec.ErrInvalid,
		)
	}
	return &Decoder{
		candidates: candidates.Clone(),
		documents:  documents.Clone(),
		layout:     layout,
	}, nil
}

func (*Decoder) ID() spec.DecoderID {
	return skillSource.MarkdownDecoderID
}

func (*Decoder) Revision() string {
	return skillSource.SkillSchemaVersion
}

func (d *Decoder) Recognize(
	_ context.Context,
	candidate ingestModel.Candidate,
) ingestModel.Recognition {
	if !d.documents.Matches(candidate.Locator) {
		return ingestModel.RecognitionNone
	}
	if !candidate.RequestsDecoder(skillSource.MarkdownDecoderID) &&
		!d.candidates.Matches(candidate.Locator) {
		return ingestModel.RecognitionNone
	}
	if candidate.RequestsDecoder(skillSource.MarkdownDecoderID) {
		return ingestModel.RecognitionPreferred
	}
	return ingestModel.RecognitionPossible
}

func (d *Decoder) Decode(
	_ context.Context,
	candidate ingestModel.Candidate,
) ([]ingestModel.Decoded, []diagnostic.Diagnostic) {
	if !d.documents.Matches(candidate.Locator) {
		return nil, nil
	}

	expectedName := expectedSkillName(d.layout, candidate.Locator)
	value, warnings, err := skillSource.DecodeSkillDocument(
		d.documents,
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

func expectedSkillName(
	layout support.PackageLayout,
	locator spec.Locator,
) string {
	skillDirectory := path.Dir(string(locator))
	if skillDirectory == "." {
		return ""
	}
	packageDirectory := spec.Locator(path.Dir(skillDirectory))
	if address, err := managedpackageModel.ParseManagedPackageAddressDirectory(
		packageDirectory,
	); err == nil &&
		address.Kind == layout.Kind &&
		path.Base(skillDirectory) == string(address.Name) {
		return string(address.Name)
	}
	return path.Base(skillDirectory)
}
