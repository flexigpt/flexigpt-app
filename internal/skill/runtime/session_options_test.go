package runtime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/flexigpt/agentskills-go/provider"
	"github.com/flexigpt/agentskills-go/provider/fs"
	agentskillsRuntimeSpec "github.com/flexigpt/agentskills-go/runtime/spec"
)

type fixedCatalogSource map[CatalogID][]SkillRegistration

func (s fixedCatalogSource) Skills(
	ctx context.Context,
	catalogID CatalogID,
) ([]SkillRegistration, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return append([]SkillRegistration(nil), s[catalogID]...), nil
}

func TestCreateSkillSessionAllowedSkills(t *testing.T) {
	skillDirectory := filepath.Join(t.TempDir(), "weather")
	if err := os.MkdirAll(skillDirectory, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(skillDirectory, "SKILL.md"),
		[]byte("---\nname: weather\ndescription: Weather helpers.\n---\n# Weather\n"),
		0o600,
	); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	definition := provider.SkillDef{
		Type:     fs.Type,
		Name:     "weather",
		Location: skillDirectory,
	}
	catalogID := CatalogID("test-session-options")

	service, err := New(WithCatalogSource(fixedCatalogSource{
		catalogID: {{
			Definition: definition,
			Revision:   "test-v1",
		}},
	}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		if err := service.Close(context.WithoutCancel(t.Context())); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	if err := service.SyncCatalog(t.Context(), catalogID); err != nil {
		t.Fatalf("SyncCatalog: %v", err)
	}

	tests := []struct {
		name       string
		allowed    []provider.SkillDef
		wantErrIs  error
		wantActive int
	}{
		{
			name:       "omitted allowlist is unrestricted",
			allowed:    nil,
			wantActive: 1,
		},
		{
			name:       "explicit empty allowlist denies every skill",
			allowed:    []provider.SkillDef{},
			wantErrIs:  agentskillsRuntimeSpec.ErrSkillNotAllowed,
			wantActive: 0,
		},
		{
			name:       "explicit matching allowlist permits skill",
			allowed:    []provider.SkillDef{definition},
			wantActive: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, err := service.CreateSkillSession(
				t.Context(),
				&CreateSkillSessionRequest{
					Body: &CreateSkillSessionRequestBody{
						AllowedSkills: test.allowed,
						ActiveSkills:  []provider.SkillDef{definition},
					},
				},
			)

			if test.wantErrIs != nil {
				if !errors.Is(err, test.wantErrIs) {
					t.Fatalf("CreateSkillSession error=%v, want %v", err, test.wantErrIs)
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateSkillSession: %v", err)
			}
			if response == nil || response.Body == nil {
				t.Fatal("CreateSkillSession returned an empty response")
			}
			if len(response.Body.ActiveSkills) != test.wantActive {
				t.Fatalf(
					"active skills=%d, want %d",
					len(response.Body.ActiveSkills),
					test.wantActive,
				)
			}
			if err := service.CloseSession(
				t.Context(),
				response.Body.SessionID,
			); err != nil {
				t.Fatalf("CloseSession: %v", err)
			}
		})
	}
}
