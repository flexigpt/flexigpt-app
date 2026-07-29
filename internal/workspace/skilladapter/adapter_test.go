package skilladapter

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/skillartifact"
)

func TestVerifySkillMDContentHappyErrorsAndBoundary(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	location := filepath.Join(directory, skillartifact.DefinitionFileName)
	content := []byte("---\nname: weather\ndescription: Forecast\n---\nUse the forecast tool.\n")
	if err := os.WriteFile(location, content, 0o600); err != nil {
		t.Fatalf("write skill: %v", err)
	}
	if err := verifySkillMDContent(location, cryptoutil.DigestBytes(content)); err != nil {
		t.Fatalf("verify valid content: %v", err)
	}
	if err := verifySkillMDContent(
		location,
		cryptoutil.DigestBytes([]byte("other")),
	); !errors.Is(
		err,
		basespec.ErrCatalogStale,
	) {
		t.Fatalf("mismatched digest error=%v, want ErrCatalogStale", err)
	}
	if err := verifySkillMDContent(
		directory,
		cryptoutil.DigestBytes(content),
	); !errors.Is(
		err,
		basespec.ErrInvalid,
	) {
		t.Fatalf("directory error=%v, want ErrInvalid", err)
	}

	large := filepath.Join(directory, "large.md")
	if err := os.WriteFile(large, bytes.Repeat([]byte("x"), basespec.MaxCandidateBytes+1), 0o600); err != nil {
		t.Fatalf("write oversized skill: %v", err)
	}
	if err := verifySkillMDContent(
		large,
		cryptoutil.DigestBytes([]byte("x")),
	); !errors.Is(
		err,
		basespec.ErrInvalid,
	) {
		t.Fatalf("oversized skill error=%v, want ErrInvalid", err)
	}
}

func TestVerifySourceGenerationClosesSnapshotsAndDetectsChanges(t *testing.T) {
	t.Parallel()

	value := source.Source{
		ID:          "019d3150-6d01-7a6b-a34e-d9032342bc31",
		RootID:      "019d3150-6d02-7a6b-a34e-d9032342bc31",
		Kind:        "test.source",
		DisplayName: "Source",
		Enabled:     true,
		Config:      []byte(`{}`),
		Revision:    1,
		CreatedAt:   time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC),
		ModifiedAt:  time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC),
	}

	snapshot := &skillTestSnapshot{generation: "generation-1"}
	runtime := skillTestRuntime{snapshot: snapshot}
	if err := verifySourceGeneration(t.Context(), runtime, value, "generation-1"); err != nil {
		t.Fatalf("verify unchanged generation: %v", err)
	}
	if snapshot.confirmed != 1 || snapshot.closed != 1 {
		t.Fatalf("snapshot confirmations=%d closes=%d", snapshot.confirmed, snapshot.closed)
	}

	snapshot = &skillTestSnapshot{generation: "generation-2"}
	runtime.snapshot = snapshot
	if err := verifySourceGeneration(
		t.Context(),
		runtime,
		value,
		"generation-1",
	); !errors.Is(
		err,
		basespec.ErrCatalogStale,
	) {
		t.Fatalf("changed generation error=%v, want ErrCatalogStale", err)
	}
	if snapshot.confirmed != 0 || snapshot.closed != 1 {
		t.Fatalf("changed snapshot confirmations=%d closes=%d", snapshot.confirmed, snapshot.closed)
	}

	snapshot = &skillTestSnapshot{generation: "generation-1", confirmErr: errors.New("confirm")}
	runtime.snapshot = snapshot
	if err := verifySourceGeneration(
		t.Context(),
		runtime,
		value,
		"generation-1",
	); !errors.Is(
		err,
		snapshot.confirmErr,
	) {
		t.Fatalf("confirm error=%v", err)
	}
}

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
			Artifact: basespec.ArtifactRef{ArtifactID: "019d3150-6d05-7a6b-a34e-d9032342bc31"},
			Skill:    SkillSummary{Name: "z"},
		},
		{
			Artifact: basespec.ArtifactRef{ArtifactID: "019d3150-6d04-7a6b-a34e-d9032342bc31"},
			Skill:    SkillSummary{Name: "a"},
		},
		{
			Artifact: basespec.ArtifactRef{ArtifactID: "019d3150-6d03-7a6b-a34e-d9032342bc31"},
			Skill:    SkillSummary{Name: "z"},
		},
	}
	sortWorkspaceSkills(values)
	if values[0].Skill.Name != "a" || values[1].Artifact.ArtifactID != "019d3150-6d03-7a6b-a34e-d9032342bc31" {
		t.Fatalf("sorted skills=%#v", values)
	}

	diagnostic := runtimeLocationDiagnostic(
		artifact.Artifact{Binding: basespec.SourceBinding{Locator: "skills/weather/SKILL.md"}},
		errors.New("unavailable"),
	)
	if diagnostic.Code == "" || diagnostic.Location == nil || diagnostic.Location.Locator != "skills/weather/SKILL.md" {
		t.Fatalf("runtime diagnostic=%#v", diagnostic)
	}
}

type skillTestRuntime struct {
	snapshot source.Snapshot
}

func (r skillTestRuntime) Get(context.Context, basespec.RootID, basespec.SourceID) (source.Source, error) {
	return source.Source{}, errors.New("not implemented")
}

func (r skillTestRuntime) Open(context.Context, source.Source) (source.Snapshot, error) {
	return r.snapshot, nil
}

type skillTestSnapshot struct {
	generation string
	confirmErr error
	closeErr   error
	confirmed  int
	closed     int
}

func (s *skillTestSnapshot) Generation() string { return s.generation }

func (*skillTestSnapshot) Stat(context.Context, basespec.Locator) (source.Entry, error) {
	return source.Entry{}, basespec.ErrNotFound
}

func (*skillTestSnapshot) ReadDir(context.Context, basespec.Locator) ([]source.Entry, error) {
	return nil, basespec.ErrNotFound
}

func (*skillTestSnapshot) Open(context.Context, basespec.Locator) (io.ReadCloser, error) {
	return nil, basespec.ErrNotFound
}

func (s *skillTestSnapshot) Confirm(context.Context) error {
	s.confirmed++
	return s.confirmErr
}

func (s *skillTestSnapshot) Close() error {
	s.closed++
	return s.closeErr
}
