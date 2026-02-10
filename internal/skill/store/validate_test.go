package store

import (
	"strings"
	"testing"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/skill/spec"
)

func TestValidateSkillBundle_Table(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)
	ok := func() spec.SkillBundle {
		return spec.SkillBundle{
			SchemaVersion: spec.SkillSchemaVersion,
			ID:            "b1",
			Slug:          "ok-bundle",
			DisplayName:   "Display",
			Description:   "",
			IsEnabled:     true,
			IsBuiltIn:     false,
			CreatedAt:     now,
			ModifiedAt:    now,
		}
	}

	tests := []struct {
		name    string
		mut     func(*spec.SkillBundle)
		wantSub string
	}{
		{
			"nil",
			func(b *spec.SkillBundle) { *b = spec.SkillBundle{} /* handled by nil case separately */ },
			"bundle is nil",
		},
		{"bad-schema", func(b *spec.SkillBundle) { b.SchemaVersion = "x" }, "schemaVersion"},
		{"empty-id", func(b *spec.SkillBundle) { b.ID = "" }, "id is empty"},
		{"bad-slug", func(b *spec.SkillBundle) { b.Slug = badSlug }, "invalid slug"},
		{"zero-times", func(b *spec.SkillBundle) { b.CreatedAt = time.Time{} }, "createdAt/modifiedAt is zero"},
		{
			"modified-before-created",
			func(b *spec.SkillBundle) { b.CreatedAt = now; b.ModifiedAt = now.Add(-time.Second) },
			"modifiedAt is before createdAt",
		},
		{"empty-displayName", func(b *spec.SkillBundle) { b.DisplayName = "   " }, "displayName is empty"},
		{
			"displayName-too-long",
			func(b *spec.SkillBundle) { b.DisplayName = strings.Repeat("a", maxDisplayNameLen+1) },
			"displayName too long",
		},
		{
			"description-too-long",
			func(b *spec.SkillBundle) { b.Description = strings.Repeat("a", maxDescriptionLen+1) },
			"description too long",
		},
		{"softdeleted-cannot-be-enabled", func(b *spec.SkillBundle) {
			tm := now
			b.SoftDeletedAt = &tm
			b.IsEnabled = true
		}, "soft-deleted bundle cannot be enabled"},
	}

	// Explicit nil case.
	if err := validateSkillBundle(nil); err == nil || !strings.Contains(err.Error(), "bundle is nil") {
		t.Fatalf("nil: got %v", err)
	}

	for _, tc := range tests {

		if tc.name == "nil" {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			b := ok()
			tc.mut(&b)
			err := validateSkillBundle(&b)
			if err == nil {
				t.Fatalf("expected error")
			}
			if tc.wantSub != "" && !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("err=%q does not contain %q", err.Error(), tc.wantSub)
			}
		})
	}
}

func TestValidateSkill_Table(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC)

	ok := func() spec.Skill {
		return spec.Skill{
			SchemaVersion: spec.SkillSchemaVersion,
			ID:            "s1",
			Slug:          "ok-skill",
			Type:          spec.SkillTypeFS,
			Location:      "/tmp/x",
			Name:          "Name",
			Tags:          []string{"tag1"},
			Presence:      &spec.SkillPresence{Status: spec.SkillPresenceUnknown},
			IsEnabled:     true,
			IsBuiltIn:     false,
			CreatedAt:     now,
			ModifiedAt:    now,
		}
	}

	tests := []struct {
		name    string
		mut     func(*spec.Skill)
		wantSub string
	}{
		{"bad-schema", func(s *spec.Skill) { s.SchemaVersion = "x" }, "schemaVersion"},
		{"empty-id", func(s *spec.Skill) { s.ID = "" }, "id is empty"},
		{"bad-slug", func(s *spec.Skill) { s.Slug = badSlug }, "invalid slug"},
		{"empty-location", func(s *spec.Skill) { s.Location = "   " }, "location is empty"},
		{
			"location-too-long",
			func(s *spec.Skill) { s.Location = strings.Repeat("a", maxLocationLen+1) },
			"location too long",
		},
		{"empty-name", func(s *spec.Skill) { s.Name = "  " }, "name is empty"},
		{"name-too-long", func(s *spec.Skill) { s.Name = strings.Repeat("a", maxNameLen+1) }, "name too long"},
		{"zero-times", func(s *spec.Skill) { s.CreatedAt = time.Time{} }, "createdAt/modifiedAt is zero"},
		{
			"modified-before-created",
			func(s *spec.Skill) { s.ModifiedAt = now.Add(-time.Second) },
			"modifiedAt is before createdAt",
		},
		{"invalid-type", func(s *spec.Skill) { s.Type = "nope" }, "invalid type"},
		{
			"builtin-cannot-be-fs",
			func(s *spec.Skill) { s.IsBuiltIn = true; s.Type = spec.SkillTypeFS },
			"built-in skill cannot be type=fs",
		},
		{
			"nonbuiltin-cannot-be-embeddedfs",
			func(s *spec.Skill) { s.IsBuiltIn = false; s.Type = spec.SkillTypeEmbeddedFS },
			"non-built-in skill cannot be type=embeddedfs",
		},
		{
			"invalid-presence-status",
			func(s *spec.Skill) { s.Presence = &spec.SkillPresence{Status: "bad"} },
			"invalid presence.status",
		},
	}

	if err := validateSkill(nil); err == nil || !strings.Contains(err.Error(), "skill is nil") {
		t.Fatalf("nil: got %v", err)
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sk := ok()
			tc.mut(&sk)
			err := validateSkill(&sk)
			if err == nil {
				t.Fatalf("expected error")
			}
			if tc.wantSub != "" && !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("err=%q does not contain %q", err.Error(), tc.wantSub)
			}
		})
	}
}
