package workflow

import (
	"encoding/json"
	"fmt"
	"sort"

	storeSpec "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
	workflowv1 "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/workflow/contract/v1"
)

// Inventory is the declaration-owned semantic inventory required by setup
// lowering. It is built from the already-admitted typed Workflow document.
//
// Setup consumes this inventory to select current bindings. It must not
// reparse source JSON or rediscover Workflow block semantics.
type Inventory struct {
	Targets     []TargetPlan        `json:"targets"`
	Calls       []CallSite          `json:"calls"`
	ContextUses []ContextUse        `json:"contextUses"`
	PortUses    []PortUse           `json:"portUses"`
	Retries     []RetryDeclaration  `json:"retries"`
	Effects     []EffectDeclaration `json:"effects"`
	Blocks      []BlockPath         `json:"blocks"`
}

// TargetPlan describes one named target or requirement declaration and, for
// Artifact-backed target kinds, its already-resolved composition result.
type TargetPlan struct {
	Path        string            `json:"path"`
	Name        string            `json:"name"`
	Requirement bool              `json:"requirement"`
	Target      workflowv1.Target `json:"target"`

	UsesComposition bool              `json:"usesComposition"`
	Resolution      *TargetResolution `json:"resolution,omitempty"`
}

// TargetResolution is the planning-owned relationship result for one
// Artifact-backed Workflow target declaration.
type TargetResolution struct {
	Status composition.ResolutionStatus `json:"status"`

	Target *composition.CapabilityTarget `json:"target,omitempty"`

	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// TargetUseKind is a source-planning fact. It lets setup select a current
// adapter without importing Workflow source grammar.
type TargetUseKind string

const (
	TargetUseKindAgent    TargetUseKind = "agent"
	TargetUseKindTool     TargetUseKind = "tool"
	TargetUseKindModel    TargetUseKind = "model"
	TargetUseKindService  TargetUseKind = "service"
	TargetUseKindActivity TargetUseKind = "activity"
	TargetUseKindWorkflow TargetUseKind = "workflow"
)

// TargetUse is one statically known possible target of a call site.
type TargetUse struct {
	Name string        `json:"name"`
	Kind TargetUseKind `json:"kind"`
}

// CallSite identifies one declared call operation. Mode is normalized to
// `invoke` when the source declaration omits it.
type CallSite struct {
	Path       string          `json:"path"`
	Mode       string          `json:"mode"`
	TargetSet  string          `json:"targetSet,omitempty"`
	TargetUses []TargetUse     `json:"targetUses,omitempty"`
	Call       workflowv1.Call `json:"call"`
}

// ContextUse identifies one explicit Workflow call-context value.
type ContextUse struct {
	Path     string           `json:"path"`
	CallPath string           `json:"callPath"`
	Name     string           `json:"name"`
	Value    workflowv1.Value `json:"value"`
}

type PortUseKind string

const (
	PortUseKindCall      PortUseKind = "call"
	PortUseKindPublished PortUseKind = "published"
)

// PortUse identifies a published Workflow port or a call-site port binding.
type PortUse struct {
	Path     string      `json:"path"`
	CallPath string      `json:"callPath,omitempty"`
	Kind     PortUseKind `json:"kind"`

	Name    string                 `json:"name"`
	Channel string                 `json:"channel"`
	Access  string                 `json:"access"`
	Binding workflowv1.PortBinding `json:"binding"`

	TargetName    string        `json:"targetName,omitempty"`
	TargetKind    TargetUseKind `json:"targetKind,omitempty"`
	Direction     string        `json:"direction,omitempty"`
	Stream        bool          `json:"stream,omitempty"`
	ContractKnown bool          `json:"contractKnown,omitempty"`
}

// RetryDeclaration identifies one source-declared retry policy at a stable
// nested step path.
type RetryDeclaration struct {
	Path     string           `json:"path"`
	StepPath string           `json:"stepPath"`
	Retry    workflowv1.Retry `json:"retry"`
}

// EffectDeclaration identifies one source-declared step effect at a stable
// nested step path.
type EffectDeclaration struct {
	Path     string            `json:"path"`
	StepPath string            `json:"stepPath"`
	Effect   workflowv1.Effect `json:"effect"`
}

type BlockPathRole string

const (
	BlockPathRoleRoot              BlockPathRole = "root"
	BlockPathRoleScope             BlockPathRole = "scope"
	BlockPathRoleChooseCase        BlockPathRole = "chooseCase"
	BlockPathRoleChooseDefault     BlockPathRole = "chooseDefault"
	BlockPathRoleParallelBranch    BlockPathRole = "parallelBranch"
	BlockPathRoleForeach           BlockPathRole = "foreach"
	BlockPathRoleRepeat            BlockPathRole = "repeat"
	BlockPathRoleMachineEntry      BlockPathRole = "machineEntry"
	BlockPathRoleMachineExit       BlockPathRole = "machineExit"
	BlockPathRoleMachineTransition BlockPathRole = "machineTransition"
	BlockPathRoleSaga              BlockPathRole = "saga"
	BlockPathRoleEffectReconcile   BlockPathRole = "effectReconcile"
	BlockPathRoleEffectCompensate  BlockPathRole = "effectCompensate"
	BlockPathRoleCatch             BlockPathRole = "catch"
	BlockPathRoleFinally           BlockPathRole = "finally"
)

// BlockPath identifies one nested source block. Paths are source-stable and
// independent of runtime ProgramFrame identities.
type BlockPath struct {
	Path       string        `json:"path"`
	ParentPath string        `json:"parentPath,omitempty"`
	Role       BlockPathRole `json:"role"`
}

func buildWorkflowInventory(
	document workflowv1.WorkflowDocument,
	capabilities composition.CapabilityPlan,
) (Inventory, error) {
	targets, err := targetPlansForDocument(document, capabilities)
	if err != nil {
		return Inventory{}, err
	}

	builder := inventoryBuilder{
		document: document,
		inventory: Inventory{
			Targets:     targets,
			Calls:       []CallSite{},
			ContextUses: []ContextUse{},
			PortUses:    []PortUse{},
			Retries:     []RetryDeclaration{},
			Effects:     []EffectDeclaration{},
			Blocks:      []BlockPath{},
		},
		blocks: make(map[string]struct{}),
	}
	if err := builder.addPublishedPorts(document); err != nil {
		return Inventory{}, err
	}
	if err := builder.addBlock(
		"root",
		"",
		BlockPathRoleRoot,
		document.Block,
	); err != nil {
		return Inventory{}, err
	}
	return builder.inventory, nil
}

func targetPlansForDocument(
	document workflowv1.WorkflowDocument,
	capabilities composition.CapabilityPlan,
) ([]TargetPlan, error) {
	occurrences := make(
		map[string]composition.CapabilityOccurrence,
		len(capabilities.Occurrences),
	)
	for _, occurrence := range capabilities.Occurrences {
		if _, duplicate := occurrences[occurrence.Path]; duplicate {
			return nil, fmt.Errorf(
				"%w: Workflow capability path %q is repeated",
				storeSpec.ErrIdentityConflict,
				occurrence.Path,
			)
		}
		occurrences[occurrence.Path] = occurrence
	}

	output := make(
		[]TargetPlan,
		0,
		len(document.Targets)+len(document.Requirements),
	)
	for _, group := range []struct {
		path        string
		requirement bool
		values      map[string]workflowv1.Target
	}{
		{
			path:   "targets",
			values: document.Targets,
		},
		{
			path:        "requirements",
			requirement: true,
			values:      document.Requirements,
		},
	} {
		names := sortedTargetNames(group.values)
		for _, name := range names {
			target, err := cloneWorkflowTarget(group.values[name])
			if err != nil {
				return nil, fmt.Errorf(
					"clone Workflow %s/%s: %w",
					group.path,
					name,
					err,
				)
			}

			_, usesComposition, err := target.DeclarationMember()
			if err != nil {
				return nil, fmt.Errorf(
					"workflow %s/%s declaration target: %w",
					group.path,
					name,
					err,
				)
			}

			path := group.path + "/" + name
			plan := TargetPlan{
				Path:            path,
				Name:            name,
				Requirement:     group.requirement,
				Target:          target,
				UsesComposition: usesComposition,
			}
			if usesComposition {
				occurrence, found := occurrences[path]
				if !found {
					return nil, fmt.Errorf(
						"%w: Workflow target %q has no composition result",
						storeSpec.ErrInvalid,
						path,
					)
				}
				resolution := targetResolutionForOccurrence(occurrence)
				plan.Resolution = &resolution
			}
			output = append(output, plan)
		}
	}
	return output, nil
}

func targetResolutionForOccurrence(
	occurrence composition.CapabilityOccurrence,
) TargetResolution {
	output := TargetResolution{
		Status:  occurrence.Status,
		Code:    occurrence.Code,
		Message: occurrence.Message,
	}
	if occurrence.Target != nil {
		target := occurrence.Target.Clone()
		output.Target = &target
	}
	return output
}

func sortedTargetNames(
	values map[string]workflowv1.Target,
) []string {
	output := make([]string, 0, len(values))
	for name := range values {
		output = append(output, name)
	}
	sort.Strings(output)
	return output
}

func cloneWorkflowTarget(
	value workflowv1.Target,
) (workflowv1.Target, error) {
	return jsonutil.CloneJSON(value)
}

type inventoryBuilder struct {
	document  workflowv1.WorkflowDocument
	inventory Inventory
	blocks    map[string]struct{}
}

func (b *inventoryBuilder) addPublishedPorts(
	document workflowv1.WorkflowDocument,
) error {
	names := make([]string, 0, len(document.Ports))
	for name := range document.Ports {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		binding := document.Ports[name]
		b.inventory.PortUses = append(
			b.inventory.PortUses,
			PortUse{
				Path:    "ports/" + name,
				Kind:    PortUseKindPublished,
				Name:    name,
				Channel: binding.Channel,
				Access:  binding.Access,
				Binding: binding,
			},
		)
	}
	return nil
}

func (b *inventoryBuilder) addBlock(
	path string,
	parentPath string,
	role BlockPathRole,
	block workflowv1.Block,
) error {
	if _, duplicate := b.blocks[path]; duplicate {
		return fmt.Errorf(
			"%w: Workflow block path %q is repeated",
			storeSpec.ErrIdentityConflict,
			path,
		)
	}
	b.blocks[path] = struct{}{}
	b.inventory.Blocks = append(b.inventory.Blocks, BlockPath{
		Path:       path,
		ParentPath: parentPath,
		Role:       role,
	})

	for index := range block.Steps {
		if err := b.addStep(
			path,
			block.Steps[index],
		); err != nil {
			return err
		}
	}
	for index := range block.Catch {
		pathValue := fmt.Sprintf("%s/catch/%d", path, index)
		if err := b.addBlock(
			pathValue,
			path,
			BlockPathRoleCatch,
			block.Catch[index].Do,
		); err != nil {
			return err
		}
	}
	if block.Finally != nil {
		if err := b.addBlock(
			path+"/finally",
			path,
			BlockPathRoleFinally,
			*block.Finally,
		); err != nil {
			return err
		}
	}
	return nil
}

func (b *inventoryBuilder) addStep(
	blockPath string,
	step workflowv1.Step,
) error {
	stepPath := blockPath + "/steps/" + step.ID
	if step.Retry != nil {
		retry, err := jsonutil.CloneJSON(*step.Retry)
		if err != nil {
			return fmt.Errorf(
				"clone Workflow retry %q: %w",
				stepPath,
				err,
			)
		}
		b.inventory.Retries = append(
			b.inventory.Retries,
			RetryDeclaration{
				Path:     stepPath + "/retry",
				StepPath: stepPath,
				Retry:    retry,
			},
		)
	}
	if step.Effect != nil {
		effect, err := jsonutil.CloneJSON(*step.Effect)
		if err != nil {
			return fmt.Errorf(
				"clone Workflow effect %q: %w",
				stepPath,
				err,
			)
		}
		b.inventory.Effects = append(
			b.inventory.Effects,
			EffectDeclaration{
				Path:     stepPath + "/effect",
				StepPath: stepPath,
				Effect:   effect,
			},
		)
		if step.Effect.Reconcile != nil {
			if err := b.addBlock(
				stepPath+"/effect/reconcile",
				stepPath,
				BlockPathRoleEffectReconcile,
				*step.Effect.Reconcile,
			); err != nil {
				return err
			}
		}
		if step.Effect.Compensate != nil {
			if err := b.addBlock(
				stepPath+"/effect/compensate",
				stepPath,
				BlockPathRoleEffectCompensate,
				*step.Effect.Compensate,
			); err != nil {
				return err
			}
		}
	}
	if step.Call != nil {
		if err := b.addCall(
			stepPath,
			*step.Call,
		); err != nil {
			return err
		}
	}

	switch {
	case step.Scope != nil:
		return b.addBlock(
			stepPath+"/scope",
			stepPath,
			BlockPathRoleScope,
			*step.Scope,
		)

	case step.Choose != nil:
		for index := range step.Choose.Cases {
			if err := b.addBlock(
				fmt.Sprintf(
					"%s/choose/cases/%d/do",
					stepPath,
					index,
				),
				stepPath,
				BlockPathRoleChooseCase,
				step.Choose.Cases[index].Do,
			); err != nil {
				return err
			}
		}
		if step.Choose.Default != nil {
			return b.addBlock(
				stepPath+"/choose/default",
				stepPath,
				BlockPathRoleChooseDefault,
				*step.Choose.Default,
			)
		}

	case step.Parallel != nil:
		names := make([]string, 0, len(step.Parallel.Branches))
		for name := range step.Parallel.Branches {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if err := b.addBlock(
				stepPath+"/parallel/branches/"+name+"/do",
				stepPath,
				BlockPathRoleParallelBranch,
				step.Parallel.Branches[name].Do,
			); err != nil {
				return err
			}
		}

	case step.Foreach != nil:
		return b.addBlock(
			stepPath+"/foreach/do",
			stepPath,
			BlockPathRoleForeach,
			step.Foreach.Do,
		)

	case step.Repeat != nil:
		return b.addBlock(
			stepPath+"/repeat/do",
			stepPath,
			BlockPathRoleRepeat,
			step.Repeat.Do,
		)

	case step.Machine != nil:
		names := make([]string, 0, len(step.Machine.States))
		for name := range step.Machine.States {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			state := step.Machine.States[name]
			statePath := stepPath + "/machine/states/" + name
			if state.Entry != nil {
				if err := b.addBlock(
					statePath+"/entry",
					statePath,
					BlockPathRoleMachineEntry,
					*state.Entry,
				); err != nil {
					return err
				}
			}
			if state.Exit != nil {
				if err := b.addBlock(
					statePath+"/exit",
					statePath,
					BlockPathRoleMachineExit,
					*state.Exit,
				); err != nil {
					return err
				}
			}
			for index := range state.Transitions {
				if state.Transitions[index].Do == nil {
					continue
				}
				if err := b.addBlock(
					fmt.Sprintf(
						"%s/transitions/%d/do",
						statePath,
						index,
					),
					statePath,
					BlockPathRoleMachineTransition,
					*state.Transitions[index].Do,
				); err != nil {
					return err
				}
			}
		}

	case step.Saga != nil:
		return b.addBlock(
			stepPath+"/saga/do",
			stepPath,
			BlockPathRoleSaga,
			step.Saga.Do,
		)
	}
	return nil
}

func (b *inventoryBuilder) addCall(
	stepPath string,
	call workflowv1.Call,
) error {
	cloned, err := jsonutil.CloneJSON(call)
	if err != nil {
		return fmt.Errorf(
			"clone Workflow call %q: %w",
			stepPath,
			err,
		)
	}

	targetUses, err := b.callTargetUses(cloned)
	if err != nil {
		return err
	}

	mode := cloned.Mode
	if mode == "" {
		mode = "invoke"
	}
	callPath := stepPath + "/call"
	b.inventory.Calls = append(
		b.inventory.Calls,
		CallSite{
			Path:       callPath,
			Mode:       mode,
			TargetSet:  cloned.Allowed,
			TargetUses: targetUses,
			Call:       cloned,
		},
	)

	contextNames := make([]string, 0, len(cloned.Context))
	for name := range cloned.Context {
		contextNames = append(contextNames, name)
	}
	sort.Strings(contextNames)
	for _, name := range contextNames {
		b.inventory.ContextUses = append(
			b.inventory.ContextUses,
			ContextUse{
				Path:     callPath + "/context/" + name,
				CallPath: callPath,
				Name:     name,
				Value:    cloneWorkflowValue(cloned.Context[name]),
			},
		)
	}

	portNames := make([]string, 0, len(cloned.Ports))
	for name := range cloned.Ports {
		portNames = append(portNames, name)
	}
	sort.Strings(portNames)
	for _, name := range portNames {
		binding := cloned.Ports[name]
		if len(targetUses) == 0 {
			b.inventory.PortUses = append(
				b.inventory.PortUses,
				PortUse{
					Path:     callPath + "/ports/" + name,
					CallPath: callPath,
					Kind:     PortUseKindCall,
					Name:     name,
					Channel:  binding.Channel,
					Access:   binding.Access,
					Binding:  binding,
				},
			)
			continue
		}

		for _, target := range targetUses {
			direction, stream, known := b.callPortFacts(
				cloned,
				target.Name,
				name,
			)
			b.inventory.PortUses = append(
				b.inventory.PortUses,
				PortUse{
					Path:          callPath + "/ports/" + name,
					CallPath:      callPath,
					Kind:          PortUseKindCall,
					Name:          name,
					Channel:       binding.Channel,
					Access:        binding.Access,
					Binding:       binding,
					TargetName:    target.Name,
					TargetKind:    target.Kind,
					Direction:     direction,
					Stream:        stream,
					ContractKnown: known,
				},
			)
		}
	}
	return nil
}

func (b *inventoryBuilder) callTargetUses(
	call workflowv1.Call,
) ([]TargetUse, error) {
	names := make(map[string]struct{})

	if name, literal, err := workflowLiteralTargetName(call.Target); err != nil {
		return nil, err
	} else if literal {
		if _, found := b.document.Targets[name]; !found {
			return nil, fmt.Errorf(
				"%w: Workflow call target %q is unavailable",
				storeSpec.ErrInvalid,
				name,
			)
		}
		names[name] = struct{}{}
	}

	if call.Allowed != "" {
		set, found := b.document.TargetSets[call.Allowed]
		if !found {
			return nil, fmt.Errorf(
				"%w: Workflow target set %q is unavailable",
				storeSpec.ErrInvalid,
				call.Allowed,
			)
		}
		for _, name := range set.Members {
			names[name] = struct{}{}
		}
		if set.Resolved != nil {
			for name := range b.document.Targets {
				names[name] = struct{}{}
			}
		}
	}

	ordered := make([]string, 0, len(names))
	for name := range names {
		ordered = append(ordered, name)
	}
	sort.Strings(ordered)

	output := make([]TargetUse, 0, len(ordered))
	for _, name := range ordered {
		target, found := b.document.Targets[name]
		if !found {
			return nil, fmt.Errorf(
				"%w: Workflow target %q is unavailable",
				storeSpec.ErrInvalid,
				name,
			)
		}
		output = append(output, TargetUse{
			Name: name,
			Kind: TargetUseKind(target.Kind),
		})
	}
	return output, nil
}

func (b *inventoryBuilder) callPortFacts(
	call workflowv1.Call,
	targetName string,
	portName string,
) (direction string, stream, known bool) {
	target := b.document.Targets[targetName]
	contracts := []*workflowv1.CallableInterface{
		target.Interface,
		call.Interface,
	}
	if call.Allowed != "" {
		set := b.document.TargetSets[call.Allowed]
		if set.Resolved != nil {
			contracts = append(contracts, &set.Resolved.Interface)
		} else {
			contracts = append(contracts, set.Interface)
		}
	}

	for _, contract := range contracts {
		if contract == nil {
			continue
		}
		port, found := contract.Ports[portName]
		if found {
			return port.Direction, port.Stream, true
		}
	}
	return "", false, false
}

func workflowLiteralTargetName(
	value workflowv1.Value,
) (name string, lit bool, err error) {
	var decoded any
	v := json.RawMessage(value)
	decoded, err = jsonutil.DecodeJSONRawNumbers[json.RawMessage](v)
	if err != nil {
		return "", false, err
	}

	switch current := decoded.(type) {
	case string:
		return current, true, nil
	case map[string]any:
		if len(current) != 1 {
			return "", false, nil
		}
		literal, found := current["$literal"]
		if !found {
			return "", false, nil
		}
		name, ok := literal.(string)
		return name, ok, nil
	default:
		return "", false, nil
	}
}

func cloneWorkflowValue(
	value workflowv1.Value,
) workflowv1.Value {
	return workflowv1.Value(
		append([]byte(nil), []byte(value)...),
	)
}
