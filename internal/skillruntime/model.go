package skillruntime

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	skillruntimeSpec "github.com/flexigpt/flexigpt-app/internal/skillruntime/spec"
	skillstoreSpec "github.com/flexigpt/flexigpt-app/internal/skillstore/spec"
)

type Origin string

const (
	OriginInstalled Origin = "installed"
	OriginWorkspace Origin = "workspace"
)

type Scope struct {
	Workspace *artifactstore.CollectionRef `json:"workspace,omitempty"`
}

type Skill struct {
	Ref    skillruntimeSpec.SkillRef `json:"ref"`
	Origin Origin                    `json:"origin"`

	InstalledRef     *skillstoreSpec.SkillRef     `json:"installedRef,omitempty"`
	Workspace        *artifactstore.CollectionRef `json:"workspace,omitempty"`
	ArtifactRevision uint64                       `json:"artifactRevision,omitempty"`

	Name        string                          `json:"name"`
	DisplayName string                          `json:"displayName"`
	Description string                          `json:"description"`
	Insert      agentskillsSpec.SkillInsert     `json:"insert"`
	Arguments   []agentskillsSpec.SkillArgument `json:"arguments,omitempty"`
	Tags        []string                        `json:"tags,omitempty"`

	Enabled        bool `json:"enabled"`
	Available      bool `json:"available"`
	RuntimeAllowed bool `json:"runtimeAllowed"`
	BuiltIn        bool `json:"builtIn"`

	CatalogCurrent bool   `json:"catalogCurrent"`
	State          string `json:"state,omitempty"`

	DefinitionDigest string                     `json:"definitionDigest,omitempty"`
	SourceID         artifactstore.SourceID     `json:"sourceID,omitempty"`
	Locator          artifactstore.Locator      `json:"locator,omitempty"`
	Diagnostics      []artifactstore.Diagnostic `json:"diagnostics,omitempty"`

	CreatedAt  time.Time `json:"createdAt"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

func (s Skill) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return errors.New("Skill provider name is empty")
	}
	switch s.Origin {
	case OriginInstalled:
		if s.InstalledRef == nil || s.Ref.Artifact != nil {
			return errors.New("installed Skill has no installed reference")
		}
		if s.Ref.BundleID == "" ||
			s.Ref.SkillSlug == "" ||
			s.Ref.SkillID == "" {
			return errors.New("installed Skill has an incomplete runtime reference")
		}
		if s.InstalledRef.BundleID != s.Ref.BundleID ||
			s.InstalledRef.SkillSlug != s.Ref.SkillSlug ||
			s.InstalledRef.SkillID != s.Ref.SkillID {
			return errors.New("installed Skill references do not match")
		}

	case OriginWorkspace:
		if s.Workspace == nil || s.Ref.Artifact == nil {
			return errors.New("Workspace Skill has no typed ArtifactRef")
		}
		if err := s.Workspace.Validate(); err != nil {
			return err
		}
		if err := s.Ref.Artifact.Validate(); err != nil {
			return err
		}
		if s.Workspace.RootID != s.Ref.Artifact.RootID {
			return errors.New("Workspace Skill ArtifactRef belongs to another Root")
		}

	default:
		return fmt.Errorf("unsupported Skill origin %q", s.Origin)
	}
	switch s.Insert {
	case agentskillsSpec.SkillInsertInstructions, agentskillsSpec.SkillInsertUserMessage:
	default:
		return fmt.Errorf("unsupported Skill insert behavior %q", s.Insert)
	}
	if s.CreatedAt.IsZero() || s.ModifiedAt.IsZero() {
		return errors.New("Skill timestamps are required")
	}
	if s.ModifiedAt.Before(s.CreatedAt) {
		return errors.New("Skill modified time precedes creation")
	}
	return artifactstore.ValidateDiagnostics(s.Diagnostics)
}

type RenderRequest struct {
	Scope     Scope                     `json:"scope"`
	Ref       skillruntimeSpec.SkillRef `json:"ref"`
	Arguments map[string]string         `json:"arguments,omitempty"`
}

type RenderedSkill struct {
	Skill            Skill                           `json:"skill"`
	Available        bool                            `json:"available"`
	Text             string                          `json:"text,omitempty"`
	Insert           agentskillsSpec.SkillInsert     `json:"insert,omitempty"`
	Arguments        []agentskillsSpec.SkillArgument `json:"arguments,omitempty"`
	AppliedArguments map[string]string               `json:"appliedArguments,omitempty"`
	Diagnostics      []artifactstore.Diagnostic      `json:"diagnostics,omitempty"`
}

type Provider interface {
	Owns(ref skillruntimeSpec.SkillRef) bool
	List(ctx context.Context, scope Scope) ([]Skill, error)
	Render(ctx context.Context, request RenderRequest) (RenderedSkill, error)
}

type ListProvidedSkillsRequest struct {
	Workspace *artifactstore.CollectionRef `json:"workspace,omitempty"`
}

type ListProvidedSkillsResponseBody struct {
	Skills []Skill `json:"skills"`
}

type ListProvidedSkillsResponse struct {
	Body *ListProvidedSkillsResponseBody
}

type RenderProvidedSkillRequestBody struct {
	Workspace *artifactstore.CollectionRef `json:"workspace,omitempty"`
	Ref       skillruntimeSpec.SkillRef    `json:"ref"                 required:"true"`
	Arguments map[string]string            `json:"arguments,omitempty"`
}

type RenderProvidedSkillRequest struct {
	Body *RenderProvidedSkillRequestBody
}

type RenderProvidedSkillResponse struct {
	Body *RenderedSkill
}

func unavailableDiagnostic(code, message string) artifactstore.Diagnostic {
	return artifactstore.Diagnostic{
		Severity: artifactstore.DiagnosticWarning,
		Code:     code,
		Message:  message,
	}
}
