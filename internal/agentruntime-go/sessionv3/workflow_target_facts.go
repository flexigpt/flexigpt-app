package sessionv3

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

// DirectCapabilityTarget is a runtime-neutral identity for a capability that
// has no source-backed Artifact. It deliberately contains no Artifact,
// Definition, revision, source, client, credential, or native handle.
type DirectCapabilityTarget struct {
	Kind             WorkflowTargetKind `json:"kind"`
	Name             string             `json:"name"`
	ProviderIdentity string             `json:"providerIdentity"`
	ProviderLocalID  string             `json:"providerLocalID"`
	Evidence         string             `json:"evidence,omitempty"`
}

// WorkflowTargetTransport is the target-domain transport contract prepared
// before Workflow Runtime creates WorkItems or child ProgramFrames.
type WorkflowTargetTransport struct {
	Modes             []WorkflowCallMode `json:"modes"`
	ContextForms      []string           `json:"contextForms,omitempty"`
	InputPorts        bool               `json:"inputPorts"`
	FinalOutputPorts  bool               `json:"finalOutputPorts"`
	StreamOutputPorts bool               `json:"streamOutputPorts"`
}

// WorkflowCapabilityFact is a current domain fact prepared by the target
// owner. It contains no private binding state.
type WorkflowCapabilityFact struct {
	Name        string         `json:"name"`
	Owner       string         `json:"owner"`
	Fingerprint string         `json:"fingerprint"`
	Details     CanonicalValue `json:"details,omitempty"`
}

// DirectCapabilityPlan is the durable result of target-domain preparation for
// one direct capability. AdapterID is an opaque setup-domain identifier. It
// is not an Artifact identity or a Workflow fallback envelope.
type DirectCapabilityPlan struct {
	AdapterID    string                   `json:"adapterID"`
	Target       DirectCapabilityTarget   `json:"target"`
	Facts        CanonicalValue           `json:"facts,omitempty"`
	Transport    WorkflowTargetTransport  `json:"transport"`
	Capabilities []WorkflowCapabilityFact `json:"capabilities,omitempty"`
}

// ProviderManagedToolChoice is one SDK Tool choice owned by a typed Model
// request plan. It cannot be lowered to a local Tool WorkItem.
type ProviderManagedToolChoice struct {
	Artifact    ArtifactPin    `json:"artifact"`
	Type        string         `json:"type"`
	InputSchema CanonicalValue `json:"inputSchema"`
	Call        ToolCall       `json:"call"`
}

// ProviderManagedToolTarget identifies the typed Model child plan that owns a
// provider SDK Tool Artifact.
type ProviderManagedToolTarget struct {
	OwnerModel ArtifactPin               `json:"ownerModel"`
	Choice     ProviderManagedToolChoice `json:"choice"`
}

// WorkflowRequirementFact records a root Workflow requirement separately from
// callable targets. It is never inserted into WorkflowProgram.Targets.
type WorkflowRequirementFact struct {
	Name           string                   `json:"name"`
	Kind           WorkflowTargetKind       `json:"kind"`
	Artifact       *ArtifactPin             `json:"artifact,omitempty"`
	Direct         *DirectCapabilityPlan    `json:"direct,omitempty"`
	BindingID      string                   `json:"bindingID,omitempty"`
	BindingVersion string                   `json:"bindingVersion,omitempty"`
	Capabilities   []WorkflowCapabilityFact `json:"capabilities,omitempty"`
}

// LLMToolOrigin preserves immutable invocation facts for Tool WorkItems that
// result from an LLM output. Tool binders must use this value rather than
// current Session defaults.
type LLMToolOrigin struct {
	OriginWorkItemID  WorkItemID               `json:"originWorkItemID"`
	TargetFingerprint string                   `json:"targetFingerprint"`
	Plan              *ModelDispatchPlan       `json:"plan,omitempty"`
	Direct            *DirectCapabilityPlan    `json:"direct,omitempty"`
	Capabilities      []WorkflowCapabilityFact `json:"capabilities,omitempty"`
}

func (v DirectCapabilityTarget) Validate() error {
	switch v.Kind {
	case WorkflowTargetModel,
		WorkflowTargetAgent,
		WorkflowTargetTool,
		WorkflowTargetService,
		WorkflowTargetActivity:
	default:
		return errors.New("direct capability target kind is invalid")
	}
	for _, field := range []struct {
		name  string
		value string
	}{
		{"direct capability name", v.Name},
		{"direct capability provider identity", v.ProviderIdentity},
		{"direct capability provider-local ID", v.ProviderLocalID},
	} {
		if err := validateID(field.name, field.value); err != nil {
			return err
		}
	}
	return nil
}

func (v WorkflowTargetTransport) Validate() error {
	if len(v.Modes) == 0 {
		return errors.New("workflow target transport has no supported mode")
	}

	seenModes := make(map[WorkflowCallMode]struct{}, len(v.Modes))
	for _, mode := range v.Modes {
		switch mode {
		case WorkflowCallModeInvoke,
			WorkflowCallModeStart,
			WorkflowCallModeHandoff:
		default:
			return errors.New("workflow target transport mode is invalid")
		}
		if _, duplicate := seenModes[mode]; duplicate {
			return errors.New("workflow target transport mode is duplicated")
		}
		seenModes[mode] = struct{}{}
	}

	seenForms := make(map[string]struct{}, len(v.ContextForms))
	for _, form := range v.ContextForms {
		if err := validateID("workflow target transport context form", form); err != nil {
			return err
		}
		if _, duplicate := seenForms[form]; duplicate {
			return errors.New("workflow target transport context form is duplicated")
		}
		seenForms[form] = struct{}{}
	}
	return nil
}

func (v WorkflowTargetTransport) SupportsMode(mode WorkflowCallMode) bool {
	return slices.Contains(v.Modes, mode)
}

func (v WorkflowTargetTransport) SupportsContext(name string) bool {
	return slices.Contains(v.ContextForms, name)
}

func (v WorkflowCapabilityFact) Validate() error {
	if err := validateID("workflow capability fact name", v.Name); err != nil {
		return err
	}
	if err := validateID("workflow capability fact owner", v.Owner); err != nil {
		return err
	}
	if err := validateID(
		"workflow capability fact fingerprint",
		v.Fingerprint,
	); err != nil {
		return err
	}
	if len(v.Details) != 0 && !v.Details.Valid() {
		return errors.New("workflow capability fact details are invalid")
	}
	return nil
}

// ValidateWorkflowCapabilityFacts checks exact declared requirement coverage.
// A target-domain adapter cannot silently omit, invent, or duplicate facts.
func ValidateWorkflowCapabilityFacts(
	requirements []WorkflowCapabilityRequirement,
	facts []WorkflowCapabilityFact,
) error {
	declared := make(map[string]struct{}, len(requirements))
	for _, requirement := range requirements {
		if err := requirement.Validate(); err != nil {
			return err
		}
		if _, duplicate := declared[requirement.Name]; duplicate {
			return errors.New("workflow capability requirement is duplicated")
		}
		declared[requirement.Name] = struct{}{}
	}

	seen := make(map[string]struct{}, len(facts))
	for _, fact := range facts {
		if err := fact.Validate(); err != nil {
			return err
		}
		if _, duplicate := seen[fact.Name]; duplicate {
			return errors.New("workflow capability fact is duplicated")
		}
		seen[fact.Name] = struct{}{}
		if _, found := declared[fact.Name]; !found {
			return fmt.Errorf(
				"workflow capability fact %q was not declared",
				fact.Name,
			)
		}
	}
	for name := range declared {
		if _, found := seen[name]; !found {
			return fmt.Errorf(
				"workflow capability requirement %q has no prepared fact",
				name,
			)
		}
	}
	return nil
}

func (v DirectCapabilityPlan) Validate() error {
	if err := validateID("direct capability adapter ID", v.AdapterID); err != nil {
		return err
	}
	if err := v.Target.Validate(); err != nil {
		return err
	}
	if len(v.Facts) != 0 && !v.Facts.Valid() {
		return errors.New("direct capability facts are invalid")
	}
	if err := v.Transport.Validate(); err != nil {
		return err
	}
	return ValidateWorkflowCapabilityFacts(nil, v.Capabilities)
}

func (v ProviderManagedToolChoice) Validate() error {
	if err := validatePin(v.Artifact); err != nil {
		return err
	}
	if err := validateID("provider-managed tool type", v.Type); err != nil {
		return err
	}
	if len(v.InputSchema) == 0 || !v.InputSchema.Valid() {
		return errors.New("provider-managed tool input schema is invalid")
	}
	if err := v.Call.Validate(); err != nil {
		return err
	}
	if v.Call.Binding.Kind != ToolBindingKindProvider {
		return errors.New("provider-managed tool choice has a local binding")
	}
	return nil
}

func (v ProviderManagedToolTarget) Validate() error {
	if err := validatePin(v.OwnerModel); err != nil {
		return err
	}
	return v.Choice.Validate()
}

func (v WorkflowRequirementFact) Validate() error {
	if err := validateID("workflow requirement name", v.Name); err != nil {
		return err
	}
	if v.Artifact != nil {
		if err := validatePin(*v.Artifact); err != nil {
			return err
		}
	}
	if v.Direct != nil {
		if err := v.Direct.Validate(); err != nil {
			return err
		}
	}
	if v.BindingID != "" {
		if err := validateID("workflow requirement binding ID", v.BindingID); err != nil {
			return err
		}
		if err := validateID(
			"workflow requirement binding version",
			v.BindingVersion,
		); err != nil {
			return err
		}
	}
	if v.Artifact == nil && v.Direct == nil && v.BindingID == "" {
		return errors.New("workflow requirement has no prepared target fact")
	}
	return ValidateWorkflowCapabilityFacts(nil, v.Capabilities)
}

func (v LLMToolOrigin) Validate() error {
	if err := validateID(
		"llm tool origin work item ID",
		string(v.OriginWorkItemID),
	); err != nil {
		return err
	}
	if err := validateID(
		"llm tool origin target fingerprint",
		v.TargetFingerprint,
	); err != nil {
		return err
	}
	if v.Plan == nil && v.Direct == nil {
		return errors.New("llm tool origin has no invocation facts")
	}
	if v.Plan != nil {
		if err := v.Plan.Validate(); err != nil {
			return err
		}
	}
	if v.Direct != nil {
		if err := v.Direct.Validate(); err != nil {
			return err
		}
	}
	return ValidateWorkflowCapabilityFacts(nil, v.Capabilities)
}

func cloneWorkflowCapabilityFacts(
	values []WorkflowCapabilityFact,
) []WorkflowCapabilityFact {
	if len(values) == 0 {
		return nil
	}
	output := make([]WorkflowCapabilityFact, len(values))
	for index := range values {
		output[index] = values[index]
		output[index].Details = values[index].Details.Clone()
	}
	return output
}

func cloneDirectCapabilityPlan(
	value *DirectCapabilityPlan,
) *DirectCapabilityPlan {
	if value == nil {
		return nil
	}
	output := *value
	output.Facts = value.Facts.Clone()
	output.Capabilities = cloneWorkflowCapabilityFacts(value.Capabilities)
	output.Transport.Modes = append(
		[]WorkflowCallMode(nil),
		value.Transport.Modes...,
	)
	output.Transport.ContextForms = append(
		[]string(nil),
		value.Transport.ContextForms...,
	)
	return &output
}

func cloneModelDispatchPlan(
	value *ModelDispatchPlan,
) *ModelDispatchPlan {
	if value == nil {
		return nil
	}
	output := jsonutil.CloneJSONInValue(*value)
	return &output
}

// LLMToolOriginForWorkItem creates the immutable origin facts that are copied
// into every local Tool WorkItem emitted by this LLM WorkItem.
func LLMToolOriginForWorkItem(
	item WorkItem,
) (*LLMToolOrigin, error) {
	if item.Type != WorkItemTypeLLM || item.Input.LLM == nil {
		return nil, errors.New("tool origin work item is not an llm work item")
	}
	if item.Binding == nil {
		return nil, errors.New("llm work item has no durable semantic binding")
	}

	origin := &LLMToolOrigin{
		OriginWorkItemID:  item.ID,
		TargetFingerprint: item.Binding.TargetFingerprint,
		Plan:              cloneModelDispatchPlan(item.Binding.Plan),
		Direct:            cloneDirectCapabilityPlan(item.Input.LLM.DirectTarget),
		Capabilities: cloneWorkflowCapabilityFacts(
			item.Input.LLM.CapabilityFacts,
		),
	}
	if err := origin.Validate(); err != nil {
		return nil, err
	}
	return origin, nil
}

// DirectCapabilityPlanForWorkItem returns the direct target plan carried by a
// runtime-neutral WorkItem payload. It never reads Session defaults.
func DirectCapabilityPlanForWorkItem(
	item WorkItem,
) *DirectCapabilityPlan {
	switch item.Type {
	case WorkItemTypeLLM:
		if item.Input.LLM != nil {
			return cloneDirectCapabilityPlan(item.Input.LLM.DirectTarget)
		}
	case WorkItemTypeTool:
		if item.Input.Tool != nil {
			return cloneDirectCapabilityPlan(item.Input.Tool.Direct)
		}
	case WorkItemTypeService:
		if item.Input.Service != nil {
			return cloneDirectCapabilityPlan(item.Input.Service.Direct)
		}
	case WorkItemTypeActivity:
		if item.Input.Activity != nil {
			return cloneDirectCapabilityPlan(item.Input.Activity.Direct)
		}
	default:
	}
	return nil
}

func directCapabilityTargetKindMatches(
	target DirectCapabilityTarget,
	kind WorkflowTargetKind,
) bool {
	return target.Kind == kind &&
		strings.TrimSpace(target.ProviderIdentity) != "" &&
		strings.TrimSpace(target.ProviderLocalID) != ""
}

func cloneLLMToolOrigin(value *LLMToolOrigin) *LLMToolOrigin {
	if value == nil {
		return nil
	}
	output := *value
	output.Plan = cloneModelDispatchPlan(value.Plan)
	output.Direct = cloneDirectCapabilityPlan(value.Direct)
	output.Capabilities = cloneWorkflowCapabilityFacts(value.Capabilities)
	return &output
}
