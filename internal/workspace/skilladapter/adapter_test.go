package skilladapter

import (
	"errors"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/skillartifact"
)

func TestSkillSummarySortingAndDiagnostics(t *testing.T) {
	t.Parallel()

	created := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	artifactValue := artifact.Artifact{
		ID:         "019d3150-6d03-7a6b-a34e-d9032342bc31",
		Enabled:    true,
		CreatedAt:  created,
		ModifiedAt: created.Add(time.Second),
	}
	summary := skillSummary(artifactValue, skillartifact.Body{
		Name:        "weather",
		DisplayName: "Weather",
		Description: "Forecast",
		Insert:      "instructions",
		Tags:        []string{"one"},
		Arguments:   []skillartifact.Argument{{Name: "city", Description: "City", Default: "Paris"}},
	})
	if summary.ID != artifactValue.ID || summary.Slug != "weather" || summary.Name != "weather" ||
		summary.Arguments[0].Name != "city" || !summary.IsEnabled {
		t.Fatalf("summary=%#v", summary)
	}
	summary.Tags[0] = "changed"
	if s := skillSummary(
		artifactValue,
		skillartifact.Body{Tags: []string{"one"}},
	); len(s.Tags) != 1 ||
		s.Tags[0] != "one" {
		t.Fatalf("skillSummary did not own tags: %#v", s)
	}

	values := []WorkspaceSkill{
		{
			Artifact: artifact.ArtifactRef{ArtifactID: "019d3150-6d05-7a6b-a34e-d9032342bc31"},
			Skill:    SkillSummary{Name: "z"},
		},
		{
			Artifact: artifact.ArtifactRef{ArtifactID: "019d3150-6d04-7a6b-a34e-d9032342bc31"},
			Skill:    SkillSummary{Name: "a"},
		},
		{
			Artifact: artifact.ArtifactRef{ArtifactID: "019d3150-6d03-7a6b-a34e-d9032342bc31"},
			Skill:    SkillSummary{Name: "z"},
		},
	}
	sortWorkspaceSkills(values)
	if values[0].Skill.Name != "a" || values[1].Artifact.ArtifactID != "019d3150-6d03-7a6b-a34e-d9032342bc31" {
		t.Fatalf("sorted skills=%#v", values)
	}

	diagnostic := runtimeLocationDiagnostic(
		artifact.Artifact{Binding: artifact.SourceBinding{Locator: "skills/weather/SKILL.md"}},
		errors.New("unavailable"),
	)
	if diagnostic.Code == "" || diagnostic.Location == nil || diagnostic.Location.Locator != "skills/weather/SKILL.md" {
		t.Fatalf("runtime diagnostic=%#v", diagnostic)
	}
}
