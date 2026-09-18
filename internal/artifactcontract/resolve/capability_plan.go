package resolve

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/artifact"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type CapabilityOccurrence struct {
	Path     string                  `json:"path"`
	Kind     string                  `json:"kind"`
	Type     declaration.Type        `json:"type"`
	Name     basespec.LogicalName    `json:"name,omitempty"`
	Status   ResolutionStatus        `json:"status"`
	Required bool                    `json:"required"`
	Scope    declaration.LookupScope `json:"scope,omitempty"`

	Artifact *artifact.ArtifactRef `json:"artifact,omitempty"`
	Mapped   *MappedTarget         `json:"mapped,omitempty"`

	Overrides map[string]json.RawMessage `json:"overrides,omitempty"`
	Use       map[string]json.RawMessage `json:"use,omitempty"`

	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type CapabilityPlan struct {
	RootArtifact *artifact.ArtifactRef  `json:"rootArtifact,omitempty"`
	RootMapped   *MappedTarget          `json:"rootMapped,omitempty"`
	RootType     declaration.Type       `json:"rootType"`
	RootName     basespec.LogicalName   `json:"rootName"`
	Occurrences  []CapabilityOccurrence `json:"occurrences"`
	Complete     bool                   `json:"complete"`
}

// ResolveCapabilities resolves any supported declaration Artifact and returns
// its complete capability plan. This is the generic entry point for consumer
// APIs that accept Plugin, Agent, Team, Loop, Workflow, Workspace, Skill, or
// other declaration roots.
func (r *Resolver) ResolveCapabilities(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (CapabilityPlan, error) {
	if r == nil || r.artifacts == nil {
		return CapabilityPlan{}, basespec.ErrClosed
	}
	if err := validateResolutionContext(ctx); err != nil {
		return CapabilityPlan{}, err
	}
	if err := ref.Validate(); err != nil {
		return CapabilityPlan{}, err
	}

	terminal, err := r.ResolveTerminalArtifact(ctx, ref)
	if err != nil {
		return CapabilityPlan{}, err
	}
	record, err := r.artifacts.Get(ctx, terminal)
	if err != nil {
		return CapabilityPlan{}, err
	}

	declarationType := declaration.Type(record.Kind)
	if err := declarationType.Validate(); err != nil {
		return CapabilityPlan{}, fmt.Errorf(
			"%w: Artifact %q has unsupported declaration type %q",
			basespec.ErrUnsupported,
			record.ID,
			record.Kind,
		)
	}

	resolved, err := r.resolveTyped(ctx, terminal, declarationType)
	if err != nil {
		return CapabilityPlan{}, err
	}
	return capabilityPlanFor(resolved)
}

// ResolveSkillCapabilities resolves a Skill and its allowed Tool
// relationships, including mapped Tool fallback targets.
func (r *Resolver) ResolveSkillCapabilities(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (CapabilityPlan, error) {
	value, err := r.ResolveSkill(ctx, ref)
	if err != nil {
		return CapabilityPlan{}, err
	}
	return capabilityPlanFor(value)
}

func (r *Resolver) ResolvePluginCapabilities(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (CapabilityPlan, error) {
	value, err := r.ResolvePlugin(ctx, ref)
	if err != nil {
		return CapabilityPlan{}, err
	}
	return capabilityPlanFor(value)
}

func (r *Resolver) ResolveAgentCapabilities(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (CapabilityPlan, error) {
	value, err := r.ResolveAgent(ctx, ref)
	if err != nil {
		return CapabilityPlan{}, err
	}
	return capabilityPlanFor(value)
}

func (r *Resolver) ResolveTeamCapabilities(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (CapabilityPlan, error) {
	value, err := r.ResolveTeam(ctx, ref)
	if err != nil {
		return CapabilityPlan{}, err
	}
	return capabilityPlanFor(value)
}

func (r *Resolver) ResolveLoopCapabilities(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (CapabilityPlan, error) {
	value, err := r.ResolveLoop(ctx, ref)
	if err != nil {
		return CapabilityPlan{}, err
	}
	return capabilityPlanFor(value)
}

func (r *Resolver) ResolveWorkflowCapabilities(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (CapabilityPlan, error) {
	value, err := r.ResolveWorkflow(ctx, ref)
	if err != nil {
		return CapabilityPlan{}, err
	}
	return capabilityPlanFor(value)
}

func (r *Resolver) ResolveWorkspaceCapabilities(
	ctx context.Context,
	ref artifact.ArtifactRef,
) (CapabilityPlan, error) {
	value, err := r.resolveTyped(ctx, ref, declaration.TypeWorkspace)
	if err != nil {
		return CapabilityPlan{}, err
	}
	return capabilityPlanFor(value)
}

// CapabilityPlanForResolvedEntry projects a previously resolved declaration
// tree into a capability plan without resolving the Artifact graph again.
func CapabilityPlanForResolvedEntry(
	root *ResolvedEntry,
) (CapabilityPlan, error) {
	return capabilityPlanFor(root)
}

func capabilityPlanFor(root *ResolvedEntry) (CapabilityPlan, error) {
	if root == nil {
		return CapabilityPlan{}, fmt.Errorf(
			"%w: capability plan has no root",
			basespec.ErrReferenceUnresolved,
		)
	}

	plan := CapabilityPlan{
		RootType:    root.Type,
		Occurrences: make([]CapabilityOccurrence, 0),
	}
	if ref, found := root.ArtifactRef(); found {
		value := ref
		plan.RootArtifact = &value
		plan.RootName = root.Artifact.LogicalName
	}
	if root.Mapped != nil {
		plan.RootMapped = cloneMappedTarget(root.Mapped)
		plan.RootName = root.Mapped.Name
	}
	if plan.RootName == "" {
		return CapabilityPlan{}, fmt.Errorf(
			"%w: capability root has no identity",
			basespec.ErrReferenceUnresolved,
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

func RequireComplete(occurrences []CapabilityOccurrence) error {
	for _, occurrence := range occurrences {
		if !occurrence.Required ||
			occurrence.Status == ResolutionAvailable {
			continue
		}
		return fmt.Errorf(
			"%w: capability %q (%s/%s) is %s: %s",
			basespec.ErrReferenceUnresolved,
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

	switch entry.Type {
	case declaration.TypePlugin,
		declaration.TypeAgent,
		declaration.TypeTeam:
		for _, relationship := range entry.MemberResults {
			c.collectRelationship(
				memberOccurrencePath(path, membersStr, relationship),
				relationship,
			)
		}
		if entry.DirectLoopResult != nil {
			c.collectRelationship(
				memberOccurrencePath(path, loopStr, *entry.DirectLoopResult),
				*entry.DirectLoopResult,
			)
		}
		if entry.DirectWorkflowResult != nil {
			c.collectRelationship(
				memberOccurrencePath(path, workflowStr, *entry.DirectWorkflowResult),
				*entry.DirectWorkflowResult,
			)
		}

	case declaration.TypeSkill:
		for _, relationship := range entry.AllowedToolResults {
			c.collectRelationship(
				memberOccurrencePath(path, "allowedTools", relationship),
				relationship,
			)
		}

	case declaration.TypeMCP:
		if entry.MCP != nil && entry.MCP.PolicyResult != nil {
			c.collectRelationship(
				memberOccurrencePath(path, "policy", *entry.MCP.PolicyResult),
				*entry.MCP.PolicyResult,
			)
		}

	case declaration.TypeLoop:
		if entry.Loop != nil && entry.Loop.BodyResult != nil {
			c.collectRelationship(
				memberOccurrencePath(path, "body", *entry.Loop.BodyResult),
				*entry.Loop.BodyResult,
			)
		}

	case declaration.TypeWorkflow:
		if entry.Workflow == nil {
			return
		}
		for _, node := range entry.Workflow.Nodes {
			if node.MemberResult == nil {
				continue
			}
			c.collectRelationship(
				memberOccurrencePath(
					path,
					"nodes/"+declaration.StableWorkflowNodeSegment(node.ID),
					*node.MemberResult,
				),
				*node.MemberResult,
			)
		}

	case declaration.TypeWorkspace:
		if entry.Workspace == nil {
			return
		}
		for _, relationship := range entry.Workspace.MemberResults {
			c.collectRelationship(
				memberOccurrencePath(path, membersStr, relationship),
				relationship,
			)
		}
	default:
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
		Name:      basespec.LogicalName(header.Name),
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
	if relationship.Resolved != nil {
		if ref, found := relationship.Resolved.ArtifactRef(); found {
			value := ref
			occurrence.Artifact = &value
		}
		occurrence.Mapped = cloneMappedTarget(relationship.Resolved.Mapped)
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
		value := match.Artifact
		matchOccurrence.Artifact = &value
		if match.Resolved != nil {
			if terminal, found := match.Resolved.ArtifactRef(); found {
				terminalValue := terminal
				matchOccurrence.Artifact = &terminalValue
			}
			matchOccurrence.Name = resolvedEntryName(match.Resolved)
			matchOccurrence.Mapped = cloneMappedTarget(match.Resolved.Mapped)
		}
		if match.Issue != nil {
			matchOccurrence.Code = match.Issue.Code
			matchOccurrence.Message = match.Issue.Message
		}
		c.plan.Occurrences = append(c.plan.Occurrences, matchOccurrence)
		if match.Status == ResolutionAvailable {
			c.collectEntry(matchOccurrence.Path, match.Resolved)
		}
	}
}

func memberOccurrencePath(
	parent string,
	field string,
	relationship ResolvedRelationship,
) string {
	header := relationship.Declared.Header()
	if relationship.Form == declaration.MemberSelector {
		return joinCapabilityPath(
			parent,
			field,
			"selector-"+relationshipDigest(relationship),
		)
	}

	segments := []string{field}
	if !isDirectProgramSlotMember(segments, header.Type) {
		segments = append(segments, string(header.Type))
	}
	if header.Type == declaration.TypeText {
		if insert, err := relationship.Declared.TextInsert(); err == nil {
			segments = append(segments, string(insert))
		}
	}
	segments = append(segments, header.Name)
	if relationship.Form == declaration.MemberNamed {
		segments = append(segments, relationshipDigest(relationship))
	}
	return joinCapabilityPath(parent, segments...)
}

func relationshipDigest(relationship ResolvedRelationship) string {
	raw, err := relationship.Declared.CanonicalJSON()
	if err != nil {
		return "invalid"
	}
	value := strings.TrimPrefix(
		string(cryptoutil.DigestBytes(raw)),
		cryptoutil.DigestSHA256Prefix,
	)
	if len(value) > 16 {
		return value[:16]
	}
	return value
}

func joinCapabilityPath(parent string, segments ...string) string {
	output := parent
	for _, segment := range segments {
		if output != "" {
			output += "/"
		}
		output += segment
	}
	return output
}

func resolvedEntryKey(entry *ResolvedEntry) string {
	if ref, found := entry.ArtifactRef(); found {
		return string(ref.RootID) + "\x00" + string(ref.ArtifactID)
	}
	if entry.Mapped != nil {
		return "mapped\x00" + entry.Mapped.Provider + "\x00" +
			entry.Mapped.Identifier
	}
	return "unknown"
}

func resolvedEntryName(entry *ResolvedEntry) basespec.LogicalName {
	if entry == nil {
		return ""
	}
	if entry.Artifact != nil {
		return entry.Artifact.LogicalName
	}
	if entry.Mapped != nil {
		return entry.Mapped.Name
	}
	return ""
}
