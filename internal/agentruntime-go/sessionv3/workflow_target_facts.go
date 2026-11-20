package sessionv3

import (
	"errors"
	"slices"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

// WorkflowTargetTransport is the target-domain transport contract prepared
// before Workflow Runtime creates WorkItems or child ProgramFrames.
type WorkflowTargetTransport struct {
	Modes             []WorkflowCallMode `json:"modes"`
	ContextForms      []string           `json:"contextForms,omitempty"`
	InputPorts        bool               `json:"inputPorts"`
	FinalOutputPorts  bool               `json:"finalOutputPorts"`
	StreamOutputPorts bool               `json:"streamOutputPorts"`
}

// ProviderManagedToolChoice is one SDK Tool choice owned by a typed Model
// request plan. It cannot be lowered to a local Tool WorkItem.
type ProviderManagedToolChoice struct {
	Artifact           ArtifactPin             `json:"artifact"`
	Type               string                  `json:"type"`
	InputSchema        jsonutil.CanonicalValue `json:"inputSchema"`
	UserArgumentSchema jsonutil.CanonicalValue `json:"userArgumentSchema,omitempty"`
	Configuration      jsonutil.CanonicalValue `json:"configuration,omitempty"`
	Call               ToolCall                `json:"call"`
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
	Name           string             `json:"name"`
	Kind           WorkflowTargetKind `json:"kind"`
	Artifact       *ArtifactPin       `json:"artifact,omitempty"`
	BindingID      string             `json:"bindingID,omitempty"`
	BindingVersion string             `json:"bindingVersion,omitempty"`
}

// WorkflowPreparedText records a source-backed Text contribution selected
// during Workflow preparation. Text materialization occurs later through the
// Text domain, after exact Artifact-pin verification.
type WorkflowPreparedText struct {
	Artifact ArtifactPin `json:"artifact"`
}

type WorkflowPreparedTool struct {
	Artifact    ArtifactPin `json:"artifact"`
	AutoExecute *bool       `json:"autoExecute,omitempty"`
}

type WorkflowPreparedSkill struct {
	Artifact     ArtifactPin `json:"artifact"`
	Active       bool        `json:"active,omitempty"`
	Instructions bool        `json:"instructions,omitempty"`
}

type WorkflowPreparedMCP struct {
	Server                    ArtifactPin     `json:"server"`
	IncludeServerInstructions bool            `json:"includeServerInstructions,omitempty"`
	ToolExposure              MCPToolExposure `json:"toolExposure"`
}

// WorkflowPreparedInvocation is a setup-owned immutable contribution
// snapshot. It contains source-backed pins and declaration contribution facts,
// but no client, credential, Artifact Store handle, callback, or provider
// request.
//
// Runtime uses this snapshot for Workflow Model and Agent child calls instead
// of resolving current Agent or Workspace contribution graphs at dispatch.
type WorkflowPreparedInvocation struct {
	Model                    *ArtifactPin            `json:"model,omitempty"`
	Agent                    *ArtifactPin            `json:"agent,omitempty"`
	Workspace                *ArtifactPin            `json:"workspace,omitempty"`
	IncludeModelSystemPrompt *bool                   `json:"includeModelSystemPrompt,omitempty"`
	Texts                    []WorkflowPreparedText  `json:"texts,omitempty"`
	Tools                    []WorkflowPreparedTool  `json:"tools,omitempty"`
	Skills                   []WorkflowPreparedSkill `json:"skills,omitempty"`
	MCP                      []WorkflowPreparedMCP   `json:"mcp,omitempty"`
}

// LLMToolOrigin preserves immutable invocation facts for Tool WorkItems that
// result from an LLM output. Tool binders must use this value rather than
// current Session defaults.
type LLMToolOrigin struct {
	OriginWorkItemID  WorkItemID `json:"originWorkItemID"`
	TargetFingerprint string     `json:"targetFingerprint"`
	PlanReference     string     `json:"planReference,omitempty"`

	// Plan is retained only for documents created before canonical origin-plan
	// references. New Tool WorkItems contain PlanReference and resolve the
	// canonical immutable plan from their origin LLM WorkItem.
	Plan *ModelDispatchPlan `json:"plan,omitempty"`
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
	if len(v.UserArgumentSchema) != 0 &&
		!v.UserArgumentSchema.Valid() {
		return errors.New(
			"provider-managed tool user argument schema is invalid",
		)
	}
	if len(v.Configuration) != 0 && !v.Configuration.Valid() {
		return errors.New("provider-managed tool configuration is invalid")
	}
	if len(v.Configuration) != 0 && len(v.UserArgumentSchema) == 0 {
		return errors.New(
			"provider-managed tool configuration has no user argument schema",
		)
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
	switch v.Kind {
	case WorkflowTargetModel,
		WorkflowTargetAgent,
		WorkflowTargetTool,
		WorkflowTargetMCPTool,
		WorkflowTargetWorkflow,
		WorkflowTargetProviderTool:
		if v.Artifact == nil ||
			v.BindingID != "" ||
			v.BindingVersion != "" {
			return errors.New(
				"artifact-backed workflow requirement is invalid",
			)
		}
		return validatePin(*v.Artifact)

	case WorkflowTargetService, WorkflowTargetActivity:
		if v.Artifact != nil {
			return errors.New(
				"registered-binding workflow requirement contains an Artifact",
			)
		}
		if err := validateID("workflow requirement binding ID", v.BindingID); err != nil {
			return err
		}
		return validateID(
			"workflow requirement binding version",
			v.BindingVersion,
		)

	default:
		return errors.New("workflow requirement has an unsupported target kind")
	}
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
	if v.Plan == nil && v.PlanReference == "" {
		return errors.New("llm tool origin has no immutable invocation plan")
	}
	if v.Plan != nil {
		return v.Plan.Validate()
	}
	return validateID("llm tool origin plan reference", v.PlanReference)
}

func (v WorkflowPreparedInvocation) Validate() error {
	for _, pin := range []*ArtifactPin{v.Model, v.Agent, v.Workspace} {
		if pin != nil {
			if err := validatePin(*pin); err != nil {
				return err
			}
		}
	}
	for _, value := range v.Texts {
		if err := validatePin(value.Artifact); err != nil {
			return err
		}
	}
	for _, value := range v.Tools {
		if err := validatePin(value.Artifact); err != nil {
			return err
		}
	}
	for _, value := range v.Skills {
		if err := validatePin(value.Artifact); err != nil {
			return err
		}
	}
	for _, value := range v.MCP {
		if err := validatePin(value.Server); err != nil {
			return err
		}
		switch value.ToolExposure {
		case MCPToolExposureNone, MCPToolExposureAll, MCPToolExposureSelected:
		default:
			return errors.New("prepared workflow MCP exposure is invalid")
		}
	}
	for _, duplicate := range [][]ArtifactPin{
		preparedInvocationTextPins(v.Texts),
		preparedInvocationToolPins(v.Tools),
		preparedInvocationSkillPins(v.Skills),
		preparedInvocationMCPPins(v.MCP),
	} {
		seen := make(map[string]struct{}, len(duplicate))
		for _, pin := range duplicate {
			key := pin.RootID + "\x00" + pin.ArtifactID
			if _, found := seen[key]; found {
				return errors.New("prepared workflow invocation contribution is duplicated")
			}
			seen[key] = struct{}{}
		}
	}
	return nil
}

func preparedInvocationTextPins(values []WorkflowPreparedText) []ArtifactPin {
	output := make([]ArtifactPin, 0, len(values))
	for _, value := range values {
		output = append(output, value.Artifact)
	}
	return output
}

func preparedInvocationToolPins(values []WorkflowPreparedTool) []ArtifactPin {
	output := make([]ArtifactPin, 0, len(values))
	for _, value := range values {
		output = append(output, value.Artifact)
	}
	return output
}

func preparedInvocationSkillPins(values []WorkflowPreparedSkill) []ArtifactPin {
	output := make([]ArtifactPin, 0, len(values))
	for _, value := range values {
		output = append(output, value.Artifact)
	}
	return output
}

func preparedInvocationMCPPins(values []WorkflowPreparedMCP) []ArtifactPin {
	output := make([]ArtifactPin, 0, len(values))
	for _, value := range values {
		output = append(output, value.Server)
	}
	return output
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
		PlanReference:     "workitem:" + string(item.ID),
	}
	if err := origin.Validate(); err != nil {
		return nil, err
	}
	return origin, nil
}

// LLMOriginPlan resolves the canonical immutable ModelDispatchPlan retained
// by the originating LLM WorkItem. It never uses current Session defaults.
func LLMOriginPlan(
	session Session,
	origin LLMToolOrigin,
) (*ModelDispatchPlan, error) {
	if origin.Plan != nil {
		return cloneModelDispatchPlan(origin.Plan), nil
	}
	for _, execution := range session.Executions {
		for _, item := range execution.WorkItems {
			if item.ID != origin.OriginWorkItemID ||
				item.Type != WorkItemTypeLLM ||
				item.Binding == nil ||
				item.Binding.Plan == nil {
				continue
			}
			if origin.PlanReference != "" &&
				origin.PlanReference != "workitem:"+string(item.ID) {
				return nil, errors.New(
					"llm tool origin plan reference differs from its origin work item",
				)
			}
			if item.Binding.TargetFingerprint != origin.TargetFingerprint {
				return nil, errors.New(
					"llm tool origin target fingerprint differs from its origin work item",
				)
			}
			return cloneModelDispatchPlan(item.Binding.Plan), nil
		}
	}
	return nil, errors.New("llm tool origin plan is unavailable")
}

func (v WorkflowTargetTransport) Validate() error {
	if len(v.Modes) == 0 {
		return errors.New("workflow target transport has no supported mode")
	}
	seenModes := make(map[WorkflowCallMode]struct{}, len(v.Modes))
	for _, mode := range v.Modes {
		switch mode {
		case WorkflowCallModeInvoke, WorkflowCallModeStart, WorkflowCallModeHandoff:
		default:
			return errors.New("workflow target transport mode is invalid")
		}
		if _, duplicate := seenModes[mode]; duplicate {
			return errors.New("workflow target transport mode is duplicated")
		}
		seenModes[mode] = struct{}{}
	}
	return nil
}

func (v WorkflowTargetTransport) SupportsMode(mode WorkflowCallMode) bool {
	return slices.Contains(v.Modes, mode)
}

func cloneLLMToolOrigin(value *LLMToolOrigin) *LLMToolOrigin {
	if value == nil {
		return nil
	}
	output := *value
	output.Plan = cloneModelDispatchPlan(value.Plan)
	output.PlanReference = value.PlanReference
	return &output
}
