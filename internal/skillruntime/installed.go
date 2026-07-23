package skillruntime

import (
	"context"
	"errors"
	"strings"

	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore"
	"github.com/flexigpt/flexigpt-app/internal/skillruntime/spec"
	"github.com/flexigpt/flexigpt-app/internal/skillstore"
	skillstoreSpec "github.com/flexigpt/flexigpt-app/internal/skillstore/spec"
)

const installedIdentityPrefix = "installed/"

type Installed struct {
	store   *skillstore.SkillStore
	runtime *SkillRuntime
}

func NewInstalled(value *skillstore.SkillStore, runtime *SkillRuntime) (*Installed, error) {
	if value == nil {
		return nil, errors.New("installed Skill Store is nil")
	}
	if runtime == nil {
		return nil, errors.New("Skill runtime is nil")
	}
	return &Installed{store: value, runtime: runtime}, nil
}

func (*Installed) Owns(identity string) bool {
	return strings.HasPrefix(identity, installedIdentityPrefix)
}

func (p *Installed) List(ctx context.Context, _ Scope) ([]Skill, error) {
	bundleEnabled := map[skillstoreSpec.SkillBundleID]bool{}
	bundleToken := ""
	for {
		response, err := p.store.ListSkillBundles(
			ctx,
			&skillstoreSpec.ListSkillBundlesRequest{
				IncludeDisabled: true,
				PageSize:        256,
				PageToken:       bundleToken,
			},
		)
		if err != nil {
			return nil, err
		}
		if response == nil || response.Body == nil {
			return nil, errors.New(
				"installed Skill Store returned an empty bundle list response",
			)
		}
		for _, bundle := range response.Body.SkillBundles {
			bundleEnabled[bundle.ID] = bundle.IsEnabled
		}
		if response.Body.NextPageToken == nil ||
			*response.Body.NextPageToken == "" {
			break
		}
		bundleToken = *response.Body.NextPageToken
	}

	var output []Skill
	pageToken := ""
	for {
		response, err := p.store.ListSkills(ctx, &skillstoreSpec.ListSkillsRequest{
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
			if err := skillstore.ValidateSkill(&value); err != nil {
				return nil, err
			}
			enabled, found := bundleEnabled[item.BundleID]
			if !found {
				return nil, errors.New(
					"installed Skill references an unavailable bundle",
				)
			}
			enabled = enabled && value.IsEnabled
			ref := skillstoreSpec.SkillRef{BundleID: item.BundleID, SkillSlug: item.SkillSlug, SkillID: value.ID}
			insert := value.Insert
			if insert == "" {
				insert = agentskillsSpec.SkillInsertInstructions
			}
			available := value.Presence == nil || value.Presence.Status == skillstoreSpec.SkillPresenceUnknown ||
				value.Presence.Status == skillstoreSpec.SkillPresencePresent
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
				Enabled:          enabled,
				Available:        available,
				RuntimeAllowed:   enabled,
				BuiltIn:          value.IsBuiltIn,
				Priority:         0,
				CatalogCurrent:   available,
				State:            installedState(value),
				DefinitionDigest: value.Digest,
				Diagnostics:      installedDiagnostics(value),
				CreatedAt:        value.CreatedAt,
				ModifiedAt:       value.ModifiedAt,
			}
			if err := projected.Validate(); err != nil {
				return nil, err
			}
			output = append(output, projected)
		}
		if response.Body.NextPageToken == nil || *response.Body.NextPageToken == "" {
			break
		}
		pageToken = *response.Body.NextPageToken
	}
	return output, nil
}

func (p *Installed) Render(ctx context.Context, request RenderRequest) (RenderedSkill, error) {
	ref, err := parseInstalledIdentity(request.Identity)
	if err != nil {
		return RenderedSkill{}, err
	}
	response, err := p.store.GetSkill(ctx, &skillstoreSpec.GetSkillRequest{
		BundleID:        ref.BundleID,
		SkillSlug:       ref.SkillSlug,
		IncludeDisabled: true,
	})
	if err != nil || response == nil || response.Body == nil {
		//nolint:nilerr // Explicit rendered skill return.
		return RenderedSkill{
			Available: false,
			Diagnostics: []artifactstore.Diagnostic{
				unavailableDiagnostic("skill.provider.unavailable", "the installed Skill is unavailable"),
			},
		}, nil
	}
	if response.Body.ID != ref.SkillID {
		return RenderedSkill{
			Available: false,
			Diagnostics: []artifactstore.Diagnostic{
				unavailableDiagnostic("skill.provider.stale-identity", "the installed Skill identity is stale"),
			},
		}, nil
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
	rendered, err := p.runtime.RenderSkill(
		ctx,
		&spec.RenderSkillRequest{Body: &spec.RenderSkillRequestBody{
			SkillRef: spec.SkillRef{
				BundleID:  ref.BundleID,
				SkillSlug: ref.SkillSlug,
				SkillID:   ref.SkillID,
			},
			Arguments: request.Arguments,
		}},
	)
	if err != nil || rendered == nil || rendered.Body == nil {
		//nolint:nilerr // Explicit rendered skill return.
		return RenderedSkill{
			Skill:     projected,
			Available: false,
			Diagnostics: []artifactstore.Diagnostic{
				unavailableDiagnostic("skill.provider.render-unavailable", "the installed Skill could not be rendered"),
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

func installedIdentity(ref skillstoreSpec.SkillRef) string {
	return installedIdentityPrefix + string(ref.BundleID) + "/" + string(ref.SkillSlug) + "/" + string(ref.SkillID)
}

func parseInstalledIdentity(value string) (skillstoreSpec.SkillRef, error) {
	relative, found := strings.CutPrefix(value, installedIdentityPrefix)
	if !found {
		return skillstoreSpec.SkillRef{}, errors.New("identity is not an installed Skill")
	}
	parts := strings.Split(relative, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return skillstoreSpec.SkillRef{}, errors.New("installed Skill identity is invalid")
	}
	return skillstoreSpec.SkillRef{
		BundleID:  skillstoreSpec.SkillBundleID(parts[0]),
		SkillSlug: skillstoreSpec.SkillSlug(parts[1]),
		SkillID:   skillstoreSpec.SkillID(parts[2]),
	}, nil
}

func installedState(value skillstoreSpec.Skill) string {
	if value.Presence == nil {
		return string(skillstoreSpec.SkillPresenceUnknown)
	}
	return string(value.Presence.Status)
}

func installedDiagnostics(value skillstoreSpec.Skill) []artifactstore.Diagnostic {
	var output []artifactstore.Diagnostic
	for _, warning := range value.RuntimeWarnings {
		if strings.TrimSpace(warning) != "" {
			output = artifactstore.AppendDiagnostics(
				output,
				artifactstore.Diagnostic{
					Severity: artifactstore.DiagnosticWarning,
					Code:     "skill.provider.runtime-warning",
					Message:  warning,
				},
			)
		}
	}
	if value.Presence != nil && value.Presence.Status != skillstoreSpec.SkillPresencePresent &&
		value.Presence.Status != skillstoreSpec.SkillPresenceUnknown {
		output = artifactstore.AppendDiagnostics(
			output,
			artifactstore.Diagnostic{
				Severity: artifactstore.DiagnosticWarning,
				Code:     "skill.provider.source-unavailable",
				Message:  "the installed Skill source is not currently available",
			},
		)
	}
	return output
}
