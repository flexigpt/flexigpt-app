package plugin

import (
	"context"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	pluginv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/plugin/contract/v1"
)

// DirectMembership is the resolved immediate membership of one Plugin.
//
// It intentionally does not recursively expand a selected Agent, Skill, MCP,
// Tool, Team, Loop, Workflow, or nested Plugin capability graph. Recursive
// expansion remains ResolveCapabilities.
type DirectMembership struct {
	Plugin   PluginView     `json:"plugin"`
	Members  []DirectMember `json:"members"`
	Complete bool           `json:"complete"`
}

// DirectMember is one Plugin declaration member in the exact normalized order
// exposed by PluginView. Selector members retain their direct selector matches
// without recursively expanding those matched Artifacts.
type DirectMember struct {
	Index int              `json:"index"`
	Entry PluginMemberView `json:"entry"`

	Status composition.ResolutionStatus  `json:"status"`
	Target *composition.CapabilityTarget `json:"target,omitempty"`

	SelectorMatches []DirectSelectorMatch `json:"selectorMatches,omitempty"`

	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// DirectSelectorMatch is one immediate selector result. It represents the
// selected Artifact and its direct resolution status only.
type DirectSelectorMatch struct {
	Artifact artifactModel.ArtifactRef     `json:"artifact"`
	Status   composition.ResolutionStatus  `json:"status"`
	Target   *composition.CapabilityTarget `json:"target,omitempty"`

	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// ResolveDirectMembers resolves only immediate Plugin declaration membership.
// It does not call ResolveCapabilities and therefore cannot recursively expand
// a member's own graph.
func (a *API) ResolveDirectMembers(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (DirectMembership, error) {
	if a == nil || a.resolver == nil || a.artifacts == nil {
		return DirectMembership{}, spec.ErrClosed
	}
	if err := ref.Validate(); err != nil {
		return DirectMembership{}, err
	}

	resolved, err := a.resolver.ResolvePluginMembers(ctx, ref)
	if err != nil {
		return DirectMembership{}, err
	}
	if resolved == nil || resolved.Type != declaration.TypePlugin {
		return DirectMembership{}, fmt.Errorf(
			"%w: direct membership root is not a Plugin",
			spec.ErrReferenceUnresolved,
		)
	}

	pluginRef, found := resolved.ArtifactRef()
	if !found {
		return DirectMembership{}, fmt.Errorf(
			"%w: Plugin direct membership has no source-backed root",
			spec.ErrReferenceUnresolved,
		)
	}

	view, err := a.Read(ctx, pluginRef)
	if err != nil {
		return DirectMembership{}, err
	}
	definitionValue, err := a.artifacts.GetDefinition(ctx, pluginRef)
	if err != nil {
		return DirectMembership{}, err
	}
	if view.Artifact.ResolvedDefinition == nil ||
		*view.Artifact.ResolvedDefinition != definitionValue.Digest {
		return DirectMembership{}, fmt.Errorf(
			"%w: Plugin Definition changed during direct membership resolution",
			spec.ErrRefreshRequired,
		)
	}

	document, err := pluginv1.FromDefinition(definitionValue)
	if err != nil {
		return DirectMembership{}, err
	}
	indexes, err := declaration.SortedMemberIndexes(
		"Plugin members",
		document.Members,
	)
	if err != nil {
		return DirectMembership{}, err
	}
	if len(indexes) != len(view.Entries) {
		return DirectMembership{}, fmt.Errorf(
			"%w: Plugin view member ordering is incomplete",
			spec.ErrInvalid,
		)
	}

	relationships := make(
		map[string]composition.ResolvedRelationship,
		len(resolved.Relationships),
	)
	for _, relationship := range resolved.Relationships {
		identity, err := declaration.MemberIdentityJSON(
			relationship.Declared,
		)
		if err != nil {
			return DirectMembership{}, err
		}
		if _, duplicate := relationships[string(identity)]; duplicate {
			return DirectMembership{}, fmt.Errorf(
				"%w: Plugin direct membership repeats a normalized member",
				spec.ErrIdentityConflict,
			)
		}
		relationships[string(identity)] = relationship
	}

	output := DirectMembership{
		Plugin:   view,
		Members:  make([]DirectMember, 0, len(indexes)),
		Complete: true,
	}
	for displayIndex, sourceIndex := range indexes {
		member := document.Members[sourceIndex]
		identity, err := declaration.MemberIdentityJSON(member)
		if err != nil {
			return DirectMembership{}, err
		}

		relationship, found := relationships[string(identity)]
		if !found {
			return DirectMembership{}, fmt.Errorf(
				"%w: Plugin direct membership has no relationship result",
				spec.ErrInvalid,
			)
		}

		value := DirectMember{
			Index:  displayIndex,
			Entry:  clonePluginMemberView(view.Entries[displayIndex]),
			Status: relationship.Status,
		}
		if relationship.Issue != nil {
			value.Code = relationship.Issue.Code
			value.Message = relationship.Issue.Message
		}
		if relationship.Resolved != nil &&
			relationship.Resolved.Target != nil {
			target := relationship.Resolved.Target.Clone()
			value.Target = &target
		}
		if relationship.Required &&
			relationship.Status != composition.ResolutionAvailable {
			output.Complete = false
		}

		if relationship.Selector != nil {
			value.SelectorMatches = make(
				[]DirectSelectorMatch,
				0,
				len(relationship.Selector.Matches),
			)
			for _, match := range relationship.Selector.Matches {
				selectorMatch := DirectSelectorMatch{
					Artifact: match.Artifact,
					Status:   match.Status,
				}
				if match.Issue != nil {
					selectorMatch.Code = match.Issue.Code
					selectorMatch.Message = match.Issue.Message
				}
				if match.Resolved != nil &&
					match.Resolved.Target != nil {
					target := match.Resolved.Target.Clone()
					selectorMatch.Target = &target
				}
				if relationship.Required &&
					match.Status != composition.ResolutionAvailable {
					output.Complete = false
				}
				value.SelectorMatches = append(
					value.SelectorMatches,
					selectorMatch,
				)
			}
		}

		output.Members = append(output.Members, value)
	}
	return output, nil
}

func clonePluginMemberView(
	value PluginMemberView,
) PluginMemberView {
	output := value
	if value.Locator != nil {
		locator := value.Locator.Clone()
		output.Locator = &locator
	}
	return output
}
