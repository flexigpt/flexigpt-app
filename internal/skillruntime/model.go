package skillruntime

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	skillstoreSpec "github.com/flexigpt/flexigpt-app/internal/skillstore/spec"
)

type Origin string

const (
	OriginInstalled Origin = "installed"
	OriginWorkspace Origin = "workspace"
)

type Scope struct {
	WorkspaceRootID artifactstore.RootID `json:"workspaceRootID,omitempty"`
}

type Skill struct {
	Identity string `json:"identity"`
	Origin   Origin `json:"origin"`

	InstalledRef      *skillstoreSpec.SkillRef `json:"installedRef,omitempty"`
	WorkspaceRootID   artifactstore.RootID     `json:"workspaceRootID,omitempty"`
	WorkspaceRecordID artifactstore.RecordID   `json:"workspaceRecordID,omitempty"`
	RecordRevision    uint64                   `json:"recordRevision,omitempty"`

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
	Shadowed       bool   `json:"shadowed"`
	ShadowedBy     string `json:"shadowedBy,omitempty"`

	DefinitionDigest string                     `json:"definitionDigest,omitempty"`
	SourceID         artifactstore.SourceID     `json:"sourceID,omitempty"`
	Locator          artifactstore.Locator      `json:"locator,omitempty"`
	Diagnostics      []artifactstore.Diagnostic `json:"diagnostics,omitempty"`

	CreatedAt  time.Time `json:"createdAt"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

func (s Skill) Validate() error {
	if strings.TrimSpace(s.Identity) == "" {
		return errors.New("Skill provider identity is empty")
	}
	if strings.TrimSpace(s.Name) == "" {
		return errors.New("Skill provider name is empty")
	}
	switch s.Origin {
	case OriginInstalled:
		if s.InstalledRef == nil {
			return errors.New("installed Skill has no installed reference")
		}
	case OriginWorkspace:
		if s.WorkspaceRootID == "" || s.WorkspaceRecordID == "" {
			return errors.New("workspace Skill has no Workspace identity")
		}
	default:
		return fmt.Errorf("unsupported Skill origin %q", s.Origin)
	}
	switch s.Insert {
	case agentskillsSpec.SkillInsertInstructions, agentskillsSpec.SkillInsertUserMessage:
	default:
		return fmt.Errorf("unsupported Skill insert behavior %q", s.Insert)
	}
	return artifactstore.ValidateDiagnostics(s.Diagnostics)
}

type RenderRequest struct {
	Scope     Scope             `json:"scope"`
	Identity  string            `json:"identity"`
	Arguments map[string]string `json:"arguments,omitempty"`
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
	Owns(identity string) bool
	List(ctx context.Context, scope Scope) ([]Skill, error)
	Render(ctx context.Context, request RenderRequest) (RenderedSkill, error)
}

type ListProvidedSkillsRequest struct {
	WorkspaceRootID artifactstore.RootID `json:"workspaceRootID,omitempty"`
}

type ListProvidedSkillsResponseBody struct {
	Skills []Skill `json:"skills"`
}

type ListProvidedSkillsResponse struct {
	Body *ListProvidedSkillsResponseBody
}

type RenderProvidedSkillRequestBody struct {
	WorkspaceRootID artifactstore.RootID `json:"workspaceRootID,omitempty"`
	Identity        string               `json:"identity"                  required:"true"`
	Arguments       map[string]string    `json:"arguments,omitempty"`
}

type RenderProvidedSkillRequest struct {
	Body *RenderProvidedSkillRequestBody
}

type RenderProvidedSkillResponse struct {
	Body *RenderedSkill
}

func unavailableDiagnostic(code, message string) artifactstore.Diagnostic {
	return artifactstore.Diagnostic{Severity: artifactstore.DiagnosticWarning, Code: code, Message: message}
}
