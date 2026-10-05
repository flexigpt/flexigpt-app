package composition

import (
	"context"
	"encoding/json"
	"fmt"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
)

type CapabilityOccurrence struct {
	Path     string                  `json:"path"`
	Kind     string                  `json:"kind"`
	Type     declaration.Type        `json:"type"`
	Name     spec.LogicalName        `json:"name,omitempty"`
	Status   ResolutionStatus        `json:"status"`
	Required bool                    `json:"required"`
	Scope    declaration.LookupScope `json:"scope,omitempty"`

	Target *CapabilityTarget `json:"target,omitempty"`

	Overrides map[string]json.RawMessage `json:"overrides,omitempty"`
	Use       map[string]json.RawMessage `json:"use,omitempty"`

	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type CapabilityPlan struct {
	RootArtifact *artifactModel.ArtifactRef `json:"rootArtifact,omitempty"`
	RootTarget   *CapabilityTarget          `json:"rootTarget,omitempty"`
	RootType     declaration.Type           `json:"rootType"`
	RootName     spec.LogicalName           `json:"rootName"`
	Occurrences  []CapabilityOccurrence     `json:"occurrences"`
	Complete     bool                       `json:"complete"`
}

// ResolveCapabilities resolves any supported declaration Artifact and projects
// its recursively reachable family relationship graph as one capability plan.
func (r *Resolver) ResolveCapabilities(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (CapabilityPlan, error) {
	resolved, err := r.resolveTyped(ctx, ref, "")
	if err != nil {
		return CapabilityPlan{}, err
	}
	return capabilityPlanFor(resolved)
}

func (r *Resolver) ResolveSkillCapabilities(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (CapabilityPlan, error) {
	value, err := r.ResolveSkill(ctx, ref)
	if err != nil {
		return CapabilityPlan{}, err
	}
	return capabilityPlanFor(value)
}

func (r *Resolver) ResolvePluginCapabilities(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (CapabilityPlan, error) {
	value, err := r.ResolvePlugin(ctx, ref)
	if err != nil {
		return CapabilityPlan{}, err
	}
	return capabilityPlanFor(value)
}

func (r *Resolver) ResolveAgentCapabilities(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (CapabilityPlan, error) {
	value, err := r.ResolveAgent(ctx, ref)
	if err != nil {
		return CapabilityPlan{}, err
	}
	return capabilityPlanFor(value)
}

func (r *Resolver) ResolveTeamCapabilities(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (CapabilityPlan, error) {
	value, err := r.ResolveTeam(ctx, ref)
	if err != nil {
		return CapabilityPlan{}, err
	}
	return capabilityPlanFor(value)
}

func (r *Resolver) ResolveLoopCapabilities(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (CapabilityPlan, error) {
	value, err := r.ResolveLoop(ctx, ref)
	if err != nil {
		return CapabilityPlan{}, err
	}
	return capabilityPlanFor(value)
}

func (r *Resolver) ResolveWorkflowCapabilities(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (CapabilityPlan, error) {
	value, err := r.ResolveWorkflow(ctx, ref)
	if err != nil {
		return CapabilityPlan{}, err
	}
	return capabilityPlanFor(value)
}

func (r *Resolver) ResolveWorkspaceCapabilities(
	ctx context.Context,
	ref artifactModel.ArtifactRef,
) (CapabilityPlan, error) {
	value, err := r.ResolveWorkspace(ctx, ref)
	if err != nil {
		return CapabilityPlan{}, err
	}
	return capabilityPlanFor(value)
}

// CapabilityPlanForResolvedEntry projects a previously resolved declaration
// graph without resolving catalog state a second time.
func CapabilityPlanForResolvedEntry(
	root *ResolvedEntry,
) (CapabilityPlan, error) {
	return capabilityPlanFor(root)
}

func capabilityPlanFor(
	root *ResolvedEntry,
) (CapabilityPlan, error) {
	if root == nil {
		return CapabilityPlan{}, fmt.Errorf(
			"%w: capability plan has no root",
			spec.ErrReferenceUnresolved,
		)
	}

	plan := CapabilityPlan{
		RootType:    root.Type,
		Occurrences: make([]CapabilityOccurrence, 0),
	}
	if ref, found := root.ArtifactRef(); found {
		copyRef := ref
		plan.RootArtifact = &copyRef
	}
	if root.Target != nil {
		plan.RootTarget = pointerTarget(*root.Target)
		plan.RootName = root.Target.Name
	}
	if plan.RootName == "" && root.Artifact != nil {
		plan.RootName = root.Artifact.LogicalName
	}
	if plan.RootName == "" {
		return CapabilityPlan{}, fmt.Errorf(
			"%w: capability root has no identity",
			spec.ErrReferenceUnresolved,
		)
	}

	collector := capabilityCollector{
		plan:   &plan,
		active: make(map[string]struct{}),
	}
	collector.collectEntry("", root)
	plan.Complete = RequireComplete(plan.Occurrences) == nil
	return plan, nil
}

func RequireComplete(
	occurrences []CapabilityOccurrence,
) error {
	for _, occurrence := range occurrences {
		if !occurrence.Required ||
			occurrence.Status == ResolutionAvailable {
			continue
		}
		return fmt.Errorf(
			"%w: capability %q (%s/%s) is %s: %s",
			spec.ErrReferenceUnresolved,
			occurrence.Path,
			occurrence.Type,
			occurrence.Name,
			occurrence.Status,
			occurrence.Message,
		)
	}
	return nil
}

type capabilityCollector struct {
	plan   *CapabilityPlan
	active map[string]struct{}
}

func (c *capabilityCollector) collectEntry(
	path string,
	entry *ResolvedEntry,
) {
	if entry == nil {
		return
	}
	key := resolvedEntryKey(entry)
	if _, active := c.active[key]; active {
		return
	}
	c.active[key] = struct{}{}
	defer delete(c.active, key)

	for _, relationship := range entry.Relationships {
		c.collectRelationship(
			joinCapabilityPath(path, relationship.Path...),
			relationship,
		)
	}
}

func (c *capabilityCollector) collectRelationship(
	path string,
	relationship ResolvedRelationship,
) {
	header := relationship.Declared.Header()
	occurrence := CapabilityOccurrence{
		Path:      path,
		Kind:      "member",
		Type:      header.Type,
		Name:      spec.LogicalName(header.Name),
		Status:    relationship.Status,
		Required:  relationship.Required,
		Scope:     relationship.Scope,
		Overrides: declaration.CloneRawMessageMap(relationship.Overrides),
		Use:       declaration.CloneRawMessageMap(relationship.Use),
	}
	if relationship.Issue != nil {
		occurrence.Code = relationship.Issue.Code
		occurrence.Message = relationship.Issue.Message
	}
	if relationship.Resolved != nil &&
		relationship.Resolved.Target != nil {
		occurrence.Target = pointerTarget(*relationship.Resolved.Target)
	}

	if relationship.Selector == nil {
		c.plan.Occurrences = append(c.plan.Occurrences, occurrence)
		if relationship.IsAvailable() {
			c.collectEntry(path, relationship.Resolved)
		}
		return
	}

	occurrence.Kind = "selector"
	occurrence.Name = ""
	c.plan.Occurrences = append(c.plan.Occurrences, occurrence)

	for _, match := range relationship.Selector.Matches {
		matchOccurrence := CapabilityOccurrence{
			Path:     path + "/matches/" + string(match.Artifact.ArtifactID),
			Kind:     "selector-match",
			Type:     relationship.Selector.Type,
			Status:   match.Status,
			Required: relationship.Required,
			Scope:    relationship.Scope,
			Overrides: declaration.CloneRawMessageMap(
				relationship.Overrides,
			),
			Use: declaration.CloneRawMessageMap(relationship.Use),
		}
		if match.Resolved != nil {
			matchOccurrence.Name = resolvedEntryName(match.Resolved)
			if match.Resolved.Target != nil {
				matchOccurrence.Target = pointerTarget(
					*match.Resolved.Target,
				)
			}
		}
		if match.Issue != nil {
			matchOccurrence.Code = match.Issue.Code
			matchOccurrence.Message = match.Issue.Message
		}
		c.plan.Occurrences = append(
			c.plan.Occurrences,
			matchOccurrence,
		)
		if match.Status == ResolutionAvailable {
			c.collectEntry(matchOccurrence.Path, match.Resolved)
		}
	}
}

func joinCapabilityPath(
	parent string,
	segments ...string,
) string {
	output := parent
	for _, segment := range segments {
		if output != "" {
			output += "/"
		}
		output += segment
	}
	return output
}

func resolvedEntryKey(
	entry *ResolvedEntry,
) string {
	if ref, found := entry.ArtifactRef(); found {
		return string(ref.RootID) + "\x00" + string(ref.ArtifactID)
	}
	if entry != nil && entry.Target != nil {
		return string(entry.Target.Form) + "\x00" +
			entry.Target.ProviderIdentity + "\x00" +
			entry.Target.ProviderLocalID
	}
	return "unknown"
}

func resolvedEntryName(
	entry *ResolvedEntry,
) spec.LogicalName {
	if entry == nil {
		return ""
	}
	if entry.Target != nil {
		return entry.Target.Name
	}
	if entry.Artifact != nil {
		return entry.Artifact.LogicalName
	}
	return ""
}
