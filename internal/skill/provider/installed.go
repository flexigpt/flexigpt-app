package provider

import (
	"context"
	"errors"
	"maps"
	"strings"

	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	skillSpec "github.com/flexigpt/flexigpt-app/internal/skill/spec"
	"github.com/flexigpt/flexigpt-app/internal/skill/store"
)

const installedIdentityPrefix = "installed/"

type Installed struct {
	store *store.SkillStore
}

func NewInstalled(value *store.SkillStore) (*Installed, error) {
	if value == nil {
		return nil, errors.New("installed Skill Store is nil")
	}
	return &Installed{store: value}, nil
}

func (*Installed) Owns(identity string) bool {
	return strings.HasPrefix(identity, installedIdentityPrefix)
}

func (p *Installed) List(
	ctx context.Context,
	_ Scope,
) ([]Skill, error) {
	var output []Skill
	pageToken := ""
	for {
		response, err := p.store.ListSkills(ctx, &skillSpec.ListSkillsRequest{
			IncludeDisabled:     true,
			IncludeMissing:      true,
			RecommendedPageSize: 256,
			PageToken:           pageToken,
		})
		if err != nil {
			return nil, err
		}
		if response == nil || response.Body == nil {
			return nil, errors.New("installed Skill Store returned an empty list response")
		}
		for _, item := range response.Body.SkillListItems {
			value := item.SkillDefinition
			if err := store.ValidateSkill(&value); err != nil {
				return nil, err
			}
			ref := skillSpec.SkillRef{
				BundleID:  item.BundleID,
				SkillSlug: item.SkillSlug,
				SkillID:   value.ID,
			}
			insert := value.Insert
			if insert == "" {
				insert = agentskillsSpec.SkillInsertInstructions
			}
			available := value.Presence == nil ||
				value.Presence.Status == skillSpec.SkillPresenceUnknown ||
				value.Presence.Status == skillSpec.SkillPresencePresent
			diagnostics := installedDiagnostics(value)
			projected := Skill{
				Identity:         installedIdentity(ref),
				Origin:           OriginInstalled,
				InstalledRef:     &ref,
				Name:             value.Name,
				DisplayName:      value.DisplayName,
				Description:      value.Description,
				Insert:           insert,
				Arguments:        append([]agentskillsSpec.SkillArgument(nil), value.Arguments...),
				Tags:             append([]string(nil), value.Tags...),
				Enabled:          value.IsEnabled,
				Available:        available,
				RuntimeAllowed:   value.IsEnabled,
				BuiltIn:          value.IsBuiltIn,
				Priority:         0,
				CatalogCurrent:   available,
				State:            installedState(value),
				DefinitionDigest: value.Digest,
				Diagnostics:      diagnostics,
				CreatedAt:        value.CreatedAt,
				ModifiedAt:       value.ModifiedAt,
			}
			if err := projected.Validate(); err != nil {
				return nil, err
			}
			output = append(output, projected)
		}
		if response.Body.NextPageToken == nil ||
			*response.Body.NextPageToken == "" {
			break
		}
		pageToken = *response.Body.NextPageToken
	}
	return output, nil
}

func (p *Installed) Render(
	ctx context.Context,
	request RenderRequest,
) (RenderedSkill, error) {
	ref, err := parseInstalledIdentity(request.Identity)
	if err != nil {
		return RenderedSkill{}, err
	}
	response, err := p.store.GetSkill(ctx, &skillSpec.GetSkillRequest{
		BundleID:        ref.BundleID,
		SkillSlug:       ref.SkillSlug,
		IncludeDisabled: true,
	})
	if err != nil || response == nil || response.Body == nil {
		//nolint:nilerr // Explicit rendered skill return.
		return RenderedSkill{
			Available: false,
			Diagnostics: []artifactstore.Diagnostic{
				unavailableDiagnostic(
					"skill.provider.unavailable",
					"the installed Skill is unavailable",
				),
			},
		}, nil
	}
	value := *response.Body
	if value.ID != ref.SkillID {
		return RenderedSkill{
			Available: false,
			Diagnostics: []artifactstore.Diagnostic{
				unavailableDiagnostic(
					"skill.provider.stale-identity",
					"the installed Skill identity is stale",
				),
			},
		}, nil
	}
	if err := store.ValidateSkill(&value); err != nil {
		return RenderedSkill{}, err
	}
	list, err := p.List(ctx, Scope{})
	if err != nil {
		return RenderedSkill{}, err
	}
	var projected Skill
	for _, item := range list {
		if item.Identity == request.Identity {
			projected = item
			break
		}
	}
	if !projected.Enabled || !projected.Available {
		return RenderedSkill{
			Skill:       projected,
			Available:   false,
			Diagnostics: append([]artifactstore.Diagnostic(nil), projected.Diagnostics...),
		}, nil
	}
	rendered, err := p.store.RenderSkill(ctx, &skillSpec.RenderSkillRequest{
		Body: &skillSpec.RenderSkillRequestBody{
			SkillRef:  ref,
			Arguments: request.Arguments,
		},
	})
	if err != nil {
		//nolint:nilerr // Explicit rendered skill return.
		return RenderedSkill{
			Skill:     projected,
			Available: false,
			Diagnostics: []artifactstore.Diagnostic{
				unavailableDiagnostic(
					"skill.provider.render-unavailable",
					"the installed Skill could not be rendered",
				),
			},
		}, nil
	}
	return RenderedSkill{
		Skill:            projected,
		Available:        true,
		Text:             rendered.Body.Text,
		Insert:           rendered.Body.Insert,
		Arguments:        append([]agentskillsSpec.SkillArgument(nil), rendered.Body.Arguments...),
		AppliedArguments: cloneStrings(rendered.Body.AppliedArguments),
		Diagnostics:      append([]artifactstore.Diagnostic(nil), projected.Diagnostics...),
	}, nil
}

func installedIdentity(ref skillSpec.SkillRef) string {
	return installedIdentityPrefix +
		string(ref.BundleID) + "/" +
		string(ref.SkillSlug) + "/" +
		string(ref.SkillID)
}

func parseInstalledIdentity(value string) (skillSpec.SkillRef, error) {
	relative, found := strings.CutPrefix(value, installedIdentityPrefix)
	if !found {
		return skillSpec.SkillRef{}, errors.New("identity is not an installed Skill")
	}
	parts := strings.Split(relative, "/")
	if len(parts) != 3 ||
		parts[0] == "" ||
		parts[1] == "" ||
		parts[2] == "" {
		return skillSpec.SkillRef{}, errors.New("installed Skill identity is invalid")
	}
	return skillSpec.SkillRef{
		BundleID:  skillSpec.SkillBundleID(parts[0]),
		SkillSlug: skillSpec.SkillSlug(parts[1]),
		SkillID:   skillSpec.SkillID(parts[2]),
	}, nil
}

func installedState(value skillSpec.Skill) string {
	if value.Presence == nil {
		return string(skillSpec.SkillPresenceUnknown)
	}
	return string(value.Presence.Status)
}

func installedDiagnostics(value skillSpec.Skill) []artifactstore.Diagnostic {
	var output []artifactstore.Diagnostic
	for _, warning := range value.RuntimeWarnings {
		if strings.TrimSpace(warning) == "" {
			continue
		}
		output = artifactstore.AppendDiagnostics(output, artifactstore.Diagnostic{
			Severity: artifactstore.DiagnosticWarning,
			Code:     "skill.provider.runtime-warning",
			Message:  warning,
		})
	}
	if value.Presence != nil &&
		value.Presence.Status != skillSpec.SkillPresencePresent &&
		value.Presence.Status != skillSpec.SkillPresenceUnknown {
		output = artifactstore.AppendDiagnostics(output, artifactstore.Diagnostic{
			Severity: artifactstore.DiagnosticWarning,
			Code:     "skill.provider.source-unavailable",
			Message:  "the installed Skill source is not currently available",
		})
	}
	return output
}

func cloneStrings(value map[string]string) map[string]string {
	if value == nil {
		return nil
	}
	output := make(map[string]string, len(value))
	maps.Copy(output, value)
	return output
}

var _ Provider = (*Installed)(nil)
