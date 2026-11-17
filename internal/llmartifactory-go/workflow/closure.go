package workflow

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	artifactModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model"
	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
)

// ArtifactClosureNodeID is a stable source-backed Artifact node identity.
// It is derived only from an ArtifactRef and does not contain runtime state.
type ArtifactClosureNodeID string

// ArtifactRelationshipRole identifies why a source-backed dependency is part
// of a Workflow Artifact closure.
type ArtifactRelationshipRole string

const (
	ArtifactRelationshipRoleCallableTarget  ArtifactRelationshipRole = "callableTarget"
	ArtifactRelationshipRoleRequirement     ArtifactRelationshipRole = "requirement"
	ArtifactRelationshipRoleNestedWorkflow  ArtifactRelationshipRole = "nestedWorkflow"
	ArtifactRelationshipRoleMCPServer       ArtifactRelationshipRole = "mcpServer"
	ArtifactRelationshipRoleTargetSetMember ArtifactRelationshipRole = "targetSetMember"
	ArtifactRelationshipRoleComposition     ArtifactRelationshipRole = "composition"
	ArtifactRelationshipRoleSelectorMember  ArtifactRelationshipRole = "selectorMember"
)

// WorkflowPlanNode contains typed admitted Workflow facts for one Workflow
// Artifact closure node. It has no setup, runtime, binding, client, secret,
// callback, or native-handle data.
type WorkflowPlanNode struct {
	Document workflowv1.WorkflowDocument `json:"document"`

	// Capabilities remains diagnostic composition information. Callers use
	// Inventory for callable-target and requirement semantics.
	Capabilities        composition.CapabilityPlan `json:"capabilities"`
	CompositionComplete bool                       `json:"compositionComplete"`
	Inventory           Inventory                  `json:"inventory"`
}

// ArtifactClosureNode is one source-backed Artifact in the static closure.
type ArtifactClosureNode struct {
	ID ArtifactClosureNodeID `json:"id"`

	Artifact         artifactModel.ArtifactRef  `json:"artifact"`
	Kind             artifactModel.ArtifactKind `json:"kind"`
	Revision         uint64                     `json:"revision"`
	DefinitionDigest cryptoutil.Digest          `json:"definitionDigest"`

	// Paths identifies every deterministic static occurrence of this node.
	// The root node always contains the path `root`.
	Paths []string `json:"paths"`

	// Workflow is set only for Workflow Artifacts. It gives downstream setup
	// enough admitted source facts to compile a nested Workflow without
	// calling ResolvePlan again.
	Workflow *WorkflowPlanNode `json:"workflow,omitempty"`
}

// ArtifactClosureEdge is one source-backed parent-to-child dependency edge.
type ArtifactClosureEdge struct {
	Parent   ArtifactClosureNodeID    `json:"parent"`
	Child    ArtifactClosureNodeID    `json:"child"`
	Path     string                   `json:"path"`
	Role     ArtifactRelationshipRole `json:"role"`
	Required bool                     `json:"required"`
}

// DirectCapabilityEvidence records an available direct capability at a static
// relationship path. It deliberately has no synthetic Artifact node.
type DirectCapabilityEvidence struct {
	Parent   ArtifactClosureNodeID        `json:"parent"`
	Path     string                       `json:"path"`
	Role     ArtifactRelationshipRole     `json:"role"`
	Required bool                         `json:"required"`
	Target   composition.CapabilityTarget `json:"target"`
}

// ArtifactClosure is the complete source-backed static dependency graph for a
// Workflow plan. Complete describes only static composition availability. It
// does not establish enablement, credential, provider, policy, binding, or
// execution readiness.
type ArtifactClosure struct {
	Root ArtifactClosureNodeID `json:"root"`

	Complete bool `json:"complete"`

	Nodes              []ArtifactClosureNode      `json:"nodes"`
	Edges              []ArtifactClosureEdge      `json:"edges"`
	DirectCapabilities []DirectCapabilityEvidence `json:"directCapabilities"`

	// Fingerprint is a canonical digest of Root, Complete, Nodes, Edges, and
	// DirectCapabilities. It excludes runtime state and local enablement.
	Fingerprint cryptoutil.Digest `json:"fingerprint"`
}

// PlanRefreshError reports a closure that could not be stabilized while
// planning. It unwraps to the generic refresh-required sentinel.
type PlanRefreshError struct {
	Artifact artifactModel.ArtifactRef
	Path     string
	Reason   string
}

func (e *PlanRefreshError) Error() string {
	if e == nil {
		return "workflow plan refresh required"
	}

	message := "workflow plan refresh required"
	if e.Path != "" {
		message += " at " + e.Path
	}
	if e.Artifact.ArtifactID != "" {
		message += fmt.Sprintf(" for Artifact %q", e.Artifact.ArtifactID)
	}
	if strings.TrimSpace(e.Reason) != "" {
		message += ": " + e.Reason
	}
	return message
}

func (*PlanRefreshError) Unwrap() error {
	return storeSpec.ErrRefreshRequired
}

// PlanConflictError reports internally inconsistent source-backed closure
// facts. It unwraps to the generic conflict sentinel.
type PlanConflictError struct {
	Artifact artifactModel.ArtifactRef
	Path     string
	Reason   string
}

func (e *PlanConflictError) Error() string {
	if e == nil {
		return "workflow plan conflict"
	}

	message := "workflow plan conflict"
	if e.Path != "" {
		message += " at " + e.Path
	}
	if e.Artifact.ArtifactID != "" {
		message += fmt.Sprintf(" for Artifact %q", e.Artifact.ArtifactID)
	}
	if strings.TrimSpace(e.Reason) != "" {
		message += ": " + e.Reason
	}
	return message
}

func (*PlanConflictError) Unwrap() error {
	return storeSpec.ErrConflict
}

// Clone returns a detached caller-owned snapshot.
func (p Plan) Clone() (Plan, error) {
	return jsonutil.CloneJSON(p)
}

// Clone returns a detached caller-owned closure snapshot.
func (c ArtifactClosure) Clone() (ArtifactClosure, error) {
	return jsonutil.CloneJSON(c)
}

// Clone returns a detached caller-owned typed Workflow closure node.
func (n WorkflowPlanNode) Clone() (WorkflowPlanNode, error) {
	return jsonutil.CloneJSON(n)
}

// Clone returns a detached caller-owned source inventory.
func (i Inventory) Clone() (Inventory, error) {
	return jsonutil.CloneJSON(i)
}

type artifactClosureBuilder struct {
	nodes     map[ArtifactClosureNodeID]*ArtifactClosureNode
	nodePaths map[ArtifactClosureNodeID]map[string]struct{}

	edges    []ArtifactClosureEdge
	edgeKeys map[string]struct{}

	directCapabilities []DirectCapabilityEvidence
	directKeys         map[string]struct{}

	active map[ArtifactClosureNodeID]struct{}
}

func buildArtifactClosure(
	root *composition.ResolvedEntry,
) (ArtifactClosure, error) {
	if root == nil {
		return ArtifactClosure{}, &PlanConflictError{
			Reason: "Workflow closure root is nil",
		}
	}

	builder := artifactClosureBuilder{
		nodes:      make(map[ArtifactClosureNodeID]*ArtifactClosureNode),
		nodePaths:  make(map[ArtifactClosureNodeID]map[string]struct{}),
		edgeKeys:   make(map[string]struct{}),
		directKeys: make(map[string]struct{}),
		active:     make(map[ArtifactClosureNodeID]struct{}),
	}
	rootID, err := builder.visit(root, "root")
	if err != nil {
		return ArtifactClosure{}, err
	}
	return builder.finish(rootID)
}

func (b *artifactClosureBuilder) visit(
	entry *composition.ResolvedEntry,
	path string,
) (ArtifactClosureNodeID, error) {
	nodeID, err := b.addNode(entry, path)
	if err != nil {
		return "", err
	}
	if _, active := b.active[nodeID]; active {
		return nodeID, nil
	}
	b.active[nodeID] = struct{}{}
	defer delete(b.active, nodeID)

	relationships := sortedClosureRelationships(entry.Relationships)
	for _, relationship := range relationships {
		if len(relationship.Path) == 0 {
			return "", &PlanConflictError{
				Artifact: b.nodes[nodeID].Artifact,
				Path:     path,
				Reason:   "resolved relationship has no static path",
			}
		}

		if relationship.Selector != nil {
			for _, match := range sortedClosureSelectorMatches(
				relationship.Selector.Matches,
			) {
				if match.Status != composition.ResolutionAvailable {
					continue
				}
				if match.Resolved == nil {
					return "", &PlanConflictError{
						Artifact: b.nodes[nodeID].Artifact,
						Path: appendClosurePath(
							path,
							relationship.Path...,
						),
						Reason: "available selector match has no resolved entry",
					}
				}

				segments := append([]string(nil), relationship.Path...)
				segments = append(
					segments,
					"matches",
					string(match.Artifact.ArtifactID),
				)
				if err := b.linkEntry(
					nodeID,
					appendClosurePath(path, segments...),
					ArtifactRelationshipRoleSelectorMember,
					relationship.Required,
					match.Resolved,
					true,
				); err != nil {
					return "", err
				}
			}
			continue
		}

		if relationship.Status != composition.ResolutionAvailable {
			continue
		}
		if relationship.Resolved == nil {
			return "", &PlanConflictError{
				Artifact: b.nodes[nodeID].Artifact,
				Path:     appendClosurePath(path, relationship.Path...),
				Reason:   "available relationship has no resolved entry",
			}
		}

		if err := b.linkEntry(
			nodeID,
			appendClosurePath(path, relationship.Path...),
			closureRelationshipRole(entry, relationship),
			relationship.Required,
			relationship.Resolved,
			true,
		); err != nil {
			return "", err
		}
	}

	if err := b.addStaticTargetSetEdges(entry, nodeID, path); err != nil {
		return "", err
	}
	return nodeID, nil
}

func (b *artifactClosureBuilder) addNode(
	entry *composition.ResolvedEntry,
	path string,
) (ArtifactClosureNodeID, error) {
	if entry == nil || entry.Artifact == nil || entry.Definition == nil {
		return "", &PlanConflictError{
			Path:   path,
			Reason: "source-backed closure entry is incomplete",
		}
	}

	artifact := entry.Artifact.Clone()
	if artifact.ResolvedDefinition == nil ||
		*artifact.ResolvedDefinition != entry.Definition.Digest {
		return "", &PlanRefreshError{
			Artifact: artifact.Ref(),
			Path:     path,
			Reason:   "Artifact Definition changed during closure construction",
		}
	}
	if entry.Type != declaration.Type(artifact.Kind) ||
		entry.Definition.Kind != artifact.Kind ||
		entry.Definition.LogicalName != artifact.LogicalName ||
		entry.Definition.LogicalVersion != artifact.LogicalVersion {
		return "", &PlanConflictError{
			Artifact: artifact.Ref(),
			Path:     path,
			Reason:   "resolved Artifact and Definition identity differ",
		}
	}
	if err := validateClosureSourceEntry(entry, artifact); err != nil {
		return "", err
	}

	nodeID := artifactClosureNodeID(artifact.Ref())
	if existing, found := b.nodes[nodeID]; found {
		if existing.Artifact != artifact.Ref() ||
			existing.Kind != artifact.Kind {
			return "", &PlanConflictError{
				Artifact: artifact.Ref(),
				Path:     path,
				Reason:   "Artifact identity has incompatible closure facts",
			}
		}
		if existing.Revision != artifact.Revision ||
			existing.DefinitionDigest != entry.Definition.Digest {
			return "", &PlanRefreshError{
				Artifact: artifact.Ref(),
				Path:     path,
				Reason:   "Artifact changed between closure occurrences",
			}
		}
		b.addNodePath(nodeID, path)
		return nodeID, nil
	}

	node := &ArtifactClosureNode{
		ID:               nodeID,
		Artifact:         artifact.Ref(),
		Kind:             artifact.Kind,
		Revision:         artifact.Revision,
		DefinitionDigest: entry.Definition.Digest,
	}
	if entry.Type == declaration.TypeWorkflow {
		workflow, err := workflowPlanNodeFor(entry)
		if err != nil {
			return "", err
		}
		node.Workflow = &workflow
	}

	b.nodes[nodeID] = node
	b.addNodePath(nodeID, path)
	return nodeID, nil
}

func validateClosureSourceEntry(
	entry *composition.ResolvedEntry,
	artifact artifactModel.Artifact,
) error {
	if entry.Target == nil {
		return &PlanConflictError{
			Artifact: artifact.Ref(),
			Reason:   "source-backed closure entry has no capability target",
		}
	}

	target := entry.Target.Clone()
	if err := target.Validate(); err != nil {
		return &PlanConflictError{
			Artifact: artifact.Ref(),
			Reason:   fmt.Sprintf("closure capability target is invalid: %v", err),
		}
	}
	if target.Type != entry.Type || target.Name != artifact.LogicalName {
		return &PlanConflictError{
			Artifact: artifact.Ref(),
			Reason:   "closure capability target identity differs from Artifact",
		}
	}
	if target.Form == composition.TargetFormArtifact &&
		(target.Artifact == nil || *target.Artifact != artifact.Ref()) {
		return &PlanConflictError{
			Artifact: artifact.Ref(),
			Reason:   "Artifact capability target differs from closure Artifact",
		}
	}
	return nil
}

func workflowPlanNodeFor(
	entry *composition.ResolvedEntry,
) (WorkflowPlanNode, error) {
	if entry == nil || entry.Definition == nil {
		return WorkflowPlanNode{}, &PlanConflictError{
			Reason: "Workflow closure node has no Definition",
		}
	}

	admitted, found := entry.AdmittedEntry()
	if !found {
		return WorkflowPlanNode{}, &PlanConflictError{
			Artifact: entry.Artifact.Ref(),
			Reason:   "Workflow closure node has no admitted declaration",
		}
	}
	document, err := workflowv1.DecodeResolvedWorkflowDefinition(
		*entry.Definition,
		admitted,
	)
	if err != nil {
		return WorkflowPlanNode{}, err
	}
	capabilities, err := composition.CapabilityPlanForResolvedEntry(entry)
	if err != nil {
		return WorkflowPlanNode{}, err
	}
	inventory, err := buildWorkflowInventory(document, capabilities)
	if err != nil {
		return WorkflowPlanNode{}, err
	}

	return WorkflowPlanNode{
		Document:            document,
		Capabilities:        capabilities,
		CompositionComplete: capabilities.Complete,
		Inventory:           inventory,
	}, nil
}

func (b *artifactClosureBuilder) linkEntry(
	parent ArtifactClosureNodeID,
	path string,
	role ArtifactRelationshipRole,
	required bool,
	child *composition.ResolvedEntry,
	descend bool,
) error {
	if child == nil {
		return &PlanConflictError{
			Artifact: b.nodes[parent].Artifact,
			Path:     path,
			Reason:   "closure relationship child is nil",
		}
	}
	if child.Artifact == nil {
		return b.addDirectCapability(parent, path, role, required, child.Target)
	}

	childID, err := b.addNode(child, path)
	if err != nil {
		return err
	}
	b.addEdge(ArtifactClosureEdge{
		Parent:   parent,
		Child:    childID,
		Path:     path,
		Role:     role,
		Required: required,
	})

	if child.Target != nil &&
		child.Target.Form == composition.TargetFormDirect {
		if err := b.addDirectCapability(
			parent,
			path,
			role,
			required,
			child.Target,
		); err != nil {
			return err
		}
	}

	if !descend {
		return nil
	}
	_, err = b.visit(child, path)
	return err
}

func (b *artifactClosureBuilder) addStaticTargetSetEdges(
	entry *composition.ResolvedEntry,
	parent ArtifactClosureNodeID,
	parentPath string,
) error {
	if entry == nil || entry.Type != declaration.TypeWorkflow {
		return nil
	}

	node := b.nodes[parent]
	if node == nil || node.Workflow == nil {
		return &PlanConflictError{
			Path:   parentPath,
			Reason: "Workflow closure node has no typed Workflow facts",
		}
	}

	for _, member := range node.Workflow.Document.StaticTargetSetMembers() {
		target, found := node.Workflow.Document.Targets[member.Target]
		if !found {
			return &PlanConflictError{
				Artifact: node.Artifact,
				Path:     appendClosurePath(parentPath, "targetSets", member.TargetSet),
				Reason:   "static target set references an unknown target",
			}
		}
		if !target.UsesArtifactComposition() {
			continue
		}

		relationship := workflowTargetRelationship(entry, member.Target)
		if relationship == nil {
			return &PlanConflictError{
				Artifact: node.Artifact,
				Path: appendClosurePath(
					parentPath,
					"targetSets",
					member.TargetSet,
					"members",
					member.Target,
				),
				Reason: "source-backed static target-set member has no relationship",
			}
		}
		if relationship.Status != composition.ResolutionAvailable {
			continue
		}
		if relationship.Resolved == nil {
			return &PlanConflictError{
				Artifact: node.Artifact,
				Path: appendClosurePath(
					parentPath,
					"targetSets",
					member.TargetSet,
					"members",
					member.Target,
				),
				Reason: "available static target-set member has no resolved entry",
			}
		}

		if err := b.linkEntry(
			parent,
			appendClosurePath(
				parentPath,
				"targetSets",
				member.TargetSet,
				"members",
				member.Target,
			),
			ArtifactRelationshipRoleTargetSetMember,
			relationship.Required,
			relationship.Resolved,
			false,
		); err != nil {
			return err
		}
	}
	return nil
}

func workflowTargetRelationship(
	entry *composition.ResolvedEntry,
	name string,
) *composition.ResolvedRelationship {
	if entry == nil {
		return nil
	}
	for index := range entry.Relationships {
		relationship := &entry.Relationships[index]
		if len(relationship.Path) != 2 ||
			relationship.Path[0] != "targets" ||
			relationship.Path[1] != name {
			continue
		}
		return relationship
	}
	return nil
}

func (b *artifactClosureBuilder) addDirectCapability(
	parent ArtifactClosureNodeID,
	path string,
	role ArtifactRelationshipRole,
	required bool,
	target *composition.CapabilityTarget,
) error {
	if target == nil {
		return &PlanConflictError{
			Artifact: b.nodes[parent].Artifact,
			Path:     path,
			Reason:   "direct capability relationship has no target evidence",
		}
	}

	value := target.Clone()
	if err := value.Validate(); err != nil {
		return &PlanConflictError{
			Artifact: b.nodes[parent].Artifact,
			Path:     path,
			Reason:   fmt.Sprintf("direct capability evidence is invalid: %v", err),
		}
	}
	if value.Form != composition.TargetFormDirect {
		return &PlanConflictError{
			Artifact: b.nodes[parent].Artifact,
			Path:     path,
			Reason:   "non-Artifact closure relationship has no direct capability target",
		}
	}

	evidence := DirectCapabilityEvidence{
		Parent:   parent,
		Path:     path,
		Role:     role,
		Required: required,
		Target:   value,
	}
	key := strings.Join([]string{
		string(evidence.Parent),
		evidence.Path,
		string(evidence.Role),
		strconv.FormatBool(evidence.Required),
		string(value.Type),
		string(value.Name),
		value.ProviderIdentity,
		value.ProviderLocalID,
		string(value.Evidence),
	}, "\x00")
	if _, found := b.directKeys[key]; found {
		return nil
	}
	b.directKeys[key] = struct{}{}
	b.directCapabilities = append(b.directCapabilities, evidence)
	return nil
}

func (b *artifactClosureBuilder) addEdge(
	value ArtifactClosureEdge,
) {
	key := strings.Join([]string{
		string(value.Parent),
		string(value.Child),
		value.Path,
		string(value.Role),
		strconv.FormatBool(value.Required),
	}, "\x00")
	if _, found := b.edgeKeys[key]; found {
		return
	}
	b.edgeKeys[key] = struct{}{}
	b.edges = append(b.edges, value)
}

func (b *artifactClosureBuilder) addNodePath(
	node ArtifactClosureNodeID,
	path string,
) {
	if path == "" {
		return
	}
	paths := b.nodePaths[node]
	if paths == nil {
		paths = make(map[string]struct{})
		b.nodePaths[node] = paths
	}
	paths[path] = struct{}{}
}

func (b *artifactClosureBuilder) finish(
	root ArtifactClosureNodeID,
) (ArtifactClosure, error) {
	rootNode, found := b.nodes[root]
	if !found || rootNode.Workflow == nil {
		return ArtifactClosure{}, &PlanConflictError{
			Reason: "Workflow closure root has no typed Workflow node",
		}
	}

	ids := make([]ArtifactClosureNodeID, 0, len(b.nodes))
	for id := range b.nodes {
		ids = append(ids, id)
	}
	slices.Sort(ids)

	nodes := make([]ArtifactClosureNode, 0, len(ids))
	for _, id := range ids {
		node := *b.nodes[id]
		paths := make([]string, 0, len(b.nodePaths[id]))
		for path := range b.nodePaths[id] {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		node.Paths = paths
		nodes = append(nodes, node)
	}

	edges := append([]ArtifactClosureEdge(nil), b.edges...)
	sort.Slice(edges, func(left, right int) bool {
		return closureEdgeLess(edges[left], edges[right])
	})

	direct := append(
		[]DirectCapabilityEvidence(nil),
		b.directCapabilities...,
	)
	sort.Slice(direct, func(left, right int) bool {
		return closureDirectCapabilityLess(direct[left], direct[right])
	})

	output := ArtifactClosure{
		Root:               root,
		Complete:           rootNode.Workflow.CompositionComplete,
		Nodes:              nodes,
		Edges:              edges,
		DirectCapabilities: direct,
	}
	fingerprint, err := artifactClosureFingerprint(output)
	if err != nil {
		return ArtifactClosure{}, err
	}
	output.Fingerprint = fingerprint
	return output, nil
}

func artifactClosureFingerprint(
	closure ArtifactClosure,
) (cryptoutil.Digest, error) {
	return cryptoutil.CanonicalDigest(struct {
		Root               ArtifactClosureNodeID      `json:"root"`
		Complete           bool                       `json:"complete"`
		Nodes              []ArtifactClosureNode      `json:"nodes"`
		Edges              []ArtifactClosureEdge      `json:"edges"`
		DirectCapabilities []DirectCapabilityEvidence `json:"directCapabilities"`
	}{
		Root:               closure.Root,
		Complete:           closure.Complete,
		Nodes:              closure.Nodes,
		Edges:              closure.Edges,
		DirectCapabilities: closure.DirectCapabilities,
	})
}

func (c ArtifactClosure) rootNode() (ArtifactClosureNode, bool) {
	for index := range c.Nodes {
		if c.Nodes[index].ID == c.Root {
			return c.Nodes[index], true
		}
	}
	return ArtifactClosureNode{}, false
}

func closureRelationshipRole(
	parent *composition.ResolvedEntry,
	relationship composition.ResolvedRelationship,
) ArtifactRelationshipRole {
	if parent == nil || parent.Type != declaration.TypeWorkflow {
		return ArtifactRelationshipRoleComposition
	}
	if len(relationship.Path) == 0 {
		return ArtifactRelationshipRoleComposition
	}

	header := relationship.Declared.Header()
	switch relationship.Path[0] {
	case "targets":
		switch header.Type {
		case declaration.TypeWorkflow:
			return ArtifactRelationshipRoleNestedWorkflow
		case declaration.TypeMCP:
			return ArtifactRelationshipRoleMCPServer
		default:
			return ArtifactRelationshipRoleCallableTarget
		}

	case "requirements":
		if header.Type == declaration.TypeMCP {
			return ArtifactRelationshipRoleMCPServer
		}
		return ArtifactRelationshipRoleRequirement

	default:
		return ArtifactRelationshipRoleComposition
	}
}

func sortedClosureRelationships(
	input []composition.ResolvedRelationship,
) []composition.ResolvedRelationship {
	output := append([]composition.ResolvedRelationship(nil), input...)
	sort.SliceStable(output, func(left, right int) bool {
		leftPath := strings.Join(output[left].Path, "\x00")
		rightPath := strings.Join(output[right].Path, "\x00")
		if leftPath != rightPath {
			return leftPath < rightPath
		}
		if output[left].Form != output[right].Form {
			return output[left].Form < output[right].Form
		}
		leftHeader := output[left].Declared.Header()
		rightHeader := output[right].Declared.Header()
		if leftHeader.Type != rightHeader.Type {
			return leftHeader.Type < rightHeader.Type
		}
		return leftHeader.Name < rightHeader.Name
	})
	return output
}

func sortedClosureSelectorMatches(
	input []composition.ResolvedSelectorMatch,
) []composition.ResolvedSelectorMatch {
	output := append([]composition.ResolvedSelectorMatch(nil), input...)
	sort.SliceStable(output, func(left, right int) bool {
		if output[left].Artifact.RootID != output[right].Artifact.RootID {
			return output[left].Artifact.RootID <
				output[right].Artifact.RootID
		}
		if output[left].Artifact.ArtifactID !=
			output[right].Artifact.ArtifactID {
			return output[left].Artifact.ArtifactID <
				output[right].Artifact.ArtifactID
		}
		return output[left].Status < output[right].Status
	})
	return output
}

func closureEdgeLess(
	left ArtifactClosureEdge,
	right ArtifactClosureEdge,
) bool {
	if left.Parent != right.Parent {
		return left.Parent < right.Parent
	}
	if left.Path != right.Path {
		return left.Path < right.Path
	}
	if left.Role != right.Role {
		return left.Role < right.Role
	}
	if left.Child != right.Child {
		return left.Child < right.Child
	}
	if left.Required != right.Required {
		return !left.Required && right.Required
	}
	return false
}

func closureDirectCapabilityLess(
	left DirectCapabilityEvidence,
	right DirectCapabilityEvidence,
) bool {
	if left.Parent != right.Parent {
		return left.Parent < right.Parent
	}
	if left.Path != right.Path {
		return left.Path < right.Path
	}
	if left.Role != right.Role {
		return left.Role < right.Role
	}
	if left.Target.ProviderIdentity != right.Target.ProviderIdentity {
		return left.Target.ProviderIdentity <
			right.Target.ProviderIdentity
	}
	if left.Target.ProviderLocalID != right.Target.ProviderLocalID {
		return left.Target.ProviderLocalID <
			right.Target.ProviderLocalID
	}
	if left.Target.Type != right.Target.Type {
		return left.Target.Type < right.Target.Type
	}
	if left.Target.Name != right.Target.Name {
		return left.Target.Name < right.Target.Name
	}
	if left.Target.Evidence != right.Target.Evidence {
		return left.Target.Evidence < right.Target.Evidence
	}
	if left.Required != right.Required {
		return !left.Required && right.Required
	}
	return false
}

func appendClosurePath(
	parent string,
	segments ...string,
) string {
	child := strings.Join(segments, "/")
	if parent == "" || parent == "root" {
		return child
	}
	if child == "" {
		return parent
	}
	return parent + "/" + child
}

func artifactClosureNodeID(
	ref artifactModel.ArtifactRef,
) ArtifactClosureNodeID {
	return ArtifactClosureNodeID(
		"artifact:" + string(ref.RootID) + ":" + string(ref.ArtifactID),
	)
}
