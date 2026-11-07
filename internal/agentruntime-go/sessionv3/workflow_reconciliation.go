package sessionv3

import (
	"errors"
	"slices"
	"strings"
)

type WorkflowReconciliationPhase string

const (
	WorkflowReconciliationPhaseIntent WorkflowReconciliationPhase = "intent"
	WorkflowReconciliationPhaseChild  WorkflowReconciliationPhase = "child"
	WorkflowReconciliationPhaseReview WorkflowReconciliationPhase = "review"
)

type WorkflowReconciliationProgress struct {
	OriginalWorkItemID WorkItemID                  `json:"originalWorkItemID"`
	IntentWorkItemID   WorkItemID                  `json:"intentWorkItemID"`
	ChildPath          string                      `json:"childPath,omitempty"`
	ReviewWorkItemID   WorkItemID                  `json:"reviewWorkItemID,omitempty"`
	Phase              WorkflowReconciliationPhase `json:"phase"`
	Operation          WorkflowOperationFacts      `json:"operation"`
	ReviewError        *SanitizedError             `json:"reviewError,omitempty"`
}

type WorkflowReconciliationResult struct {
	Decision ReconciliationDecision `json:"decision"`
	Output   CanonicalValue         `json:"output,omitempty"`
	Receipt  *ExternalReceipt       `json:"receipt,omitempty"`
	Error    *SanitizedError        `json:"error,omitempty"`
}

type WorkflowReconciliationReview struct {
	OriginalWorkItemID WorkItemID      `json:"originalWorkItemID"`
	Message            string          `json:"message"`
	Error              *SanitizedError `json:"error,omitempty"`
}

func (p WorkflowReconciliationProgress) Validate() error {
	if err := validateID(
		"workflow reconciliation original work item ID",
		string(p.OriginalWorkItemID),
	); err != nil {
		return err
	}
	if err := validateID(
		"workflow reconciliation intent work item ID",
		string(p.IntentWorkItemID),
	); err != nil {
		return err
	}
	if err := p.Operation.Validate(); err != nil {
		return err
	}
	if p.Operation.WorkItemID != p.OriginalWorkItemID {
		return errors.New(
			"workflow reconciliation operation identity differs from progress",
		)
	}
	switch p.Phase {
	case WorkflowReconciliationPhaseIntent:
		if p.ChildPath != "" || p.ReviewWorkItemID != "" {
			return errors.New(
				"workflow reconciliation intent phase has child or review state",
			)
		}
	case WorkflowReconciliationPhaseChild:
		if strings.TrimSpace(p.ChildPath) == "" ||
			p.ReviewWorkItemID != "" {
			return errors.New(
				"workflow reconciliation child phase is incomplete",
			)
		}
	case WorkflowReconciliationPhaseReview:
		if strings.TrimSpace(p.ChildPath) == "" ||
			p.ReviewError == nil {
			return errors.New(
				"workflow reconciliation review phase is incomplete",
			)
		}
	default:
		return errors.New("workflow reconciliation phase is invalid")
	}
	if p.ReviewWorkItemID != "" {
		if err := validateID(
			"workflow reconciliation review work item ID",
			string(p.ReviewWorkItemID),
		); err != nil {
			return err
		}
	}
	return validateReconciliationError(p.ReviewError)
}

func (r WorkflowReconciliationResult) Validate() error {
	settlement := ReconciliationSettlement{
		OriginalWorkItemID: "reconciliation-result",
		Decision:           r.Decision,
		Output:             r.Output,
		Receipt:            r.Receipt,
		Error:              r.Error,
	}
	return settlement.Validate()
}

func (i *WorkflowInterpreter) beginReconciliation(
	frame ProgramFrame,
	execution Execution,
	program WorkflowProgram,
	state WorkflowState,
	item WorkItem,
) (ProgramTransition, bool, error) {
	if state.Pending != nil ||
		item.State != WorkItemStateReconciling ||
		item.EffectPlan == nil ||
		item.EffectPlan.Reconcile == nil ||
		!item.Effect.Reconcile {
		return ProgramTransition{}, false, nil
	}
	if item.FrameID != frame.ID || item.ProgramPath != state.PendingPath {
		return ProgramTransition{}, false, nil
	}

	step, err := workflowStepForPath(program.Root, item.ProgramPath)
	if err != nil {
		return ProgramTransition{}, true, err
	}
	if step.Effect == nil || step.Effect.Reconcile == nil {
		return ProgramTransition{}, true, errors.New(
			"reconciling workflow work item has no reconciliation block",
		)
	}

	operation, err := workflowOperationFacts(
		execution,
		item,
		state.PendingTarget,
	)
	if err != nil {
		return ProgramTransition{}, true, err
	}
	intentID := WorkItemID(
		string(frame.ID) + ":" + string(item.ID) + ":reconciliation",
	)
	if err := validateID(
		"workflow reconciliation work item ID",
		string(intentID),
	); err != nil {
		return ProgramTransition{}, true, err
	}

	state.Pending = &WorkflowPending{
		Kind:   WorkflowPendingReconciliation,
		StepID: step.ID,
		Path:   item.ProgramPath,
		Reconciliation: &WorkflowReconciliationProgress{
			OriginalWorkItemID: item.ID,
			IntentWorkItemID:   intentID,
			Phase:              WorkflowReconciliationPhaseIntent,
			Operation:          operation,
		},
	}
	transition, err := workflowTransition(
		state,
		nil,
		nil,
		[]WorkItemIntent{{
			ID:          intentID,
			ProgramPath: item.ProgramPath + "/reconciliation",
			Type:        WorkItemTypeReconciliation,
			Cleanup:     true,
			Automatic:   true,
			Effect:      EffectContract{Kind: EffectKindRead},
			Input: WorkItemPayload{
				Reconciliation: &ReconciliationWork{
					OriginalWorkItemID: item.ID,
					Operation:          operation,
				},
			},
		}},
	)
	return transition, true, err
}

func (i *WorkflowInterpreter) settleReconciliationIntent(
	frame ProgramFrame,
	execution Execution,
	program WorkflowProgram,
	state WorkflowState,
	workItemID WorkItemID,
) (ProgramTransition, bool, error) {
	pending := state.Pending
	if pending == nil ||
		pending.Kind != WorkflowPendingReconciliation ||
		pending.Reconciliation == nil ||
		pending.Reconciliation.IntentWorkItemID != workItemID {
		return ProgramTransition{}, false, nil
	}

	item := executionWorkItem(execution, workItemID)
	if item == nil || item.FrameID != frame.ID {
		return ProgramTransition{}, true, errors.New(
			"workflow reconciliation intent work item is unavailable",
		)
	}
	if !item.State.Terminal() {
		t, err := workflowTransition(state, nil, nil, nil)
		if err != nil {
			return ProgramTransition{}, false, err
		}
		return t, true, nil
	}
	if item.State != WorkItemStateCompleted {
		return i.beginReconciliationReview(
			frame,
			execution,
			program,
			state,
			reconciliationFailure(
				"reconciliation_intent_failed",
				"reconciliation intent did not complete",
				item.Error,
			),
		)
	}

	step, err := workflowStepForPath(program.Root, pending.Path)
	if err != nil || step.Effect == nil || step.Effect.Reconcile == nil {
		return ProgramTransition{}, true, errors.New(
			"workflow reconciliation step is unavailable",
		)
	}

	operationValue := MustCanonicalValue(
		pending.Reconciliation.Operation,
	)
	childScope := WorkflowScopeContext{
		Cleanup:   true,
		Operation: &operationValue,
	}
	snapshot, err := i.childBlockProgramWithScope(
		frame,
		program,
		state,
		*step.Effect.Reconcile,
		false,
		childScope,
	)
	if err != nil {
		return ProgramTransition{}, true, err
	}

	childPath := pending.Path + "/reconciliation/child"
	pending.Reconciliation.Phase = WorkflowReconciliationPhaseChild
	pending.Reconciliation.ChildPath = childPath
	pending.Children = []WorkflowChild{{
		Name:  "reconciliation",
		Path:  childPath,
		State: WorkflowChildStatePending,
	}}
	t, err := workflowStructuredTransition(
		state,
		[]ProgramFrameIntent{{
			Path:     childPath,
			Program:  snapshot,
			Deadline: cloneTime(state.Deadline),
		}},
	)
	if err != nil {
		return ProgramTransition{}, false, err
	}
	return t, true, nil
}

func (i *WorkflowInterpreter) settleReconciliationChild(
	frame ProgramFrame,
	execution Execution,
	program WorkflowProgram,
	state WorkflowState,
	childID ProgramFrameID,
) (ProgramTransition, bool, error) {
	pending := state.Pending
	if pending == nil ||
		pending.Kind != WorkflowPendingReconciliation ||
		pending.Reconciliation == nil ||
		pending.Reconciliation.Phase != WorkflowReconciliationPhaseChild {
		return ProgramTransition{}, false, nil
	}

	var child *ProgramFrame
	for index := range execution.ProgramFrames {
		candidate := execution.ProgramFrames[index]
		if candidate.ID == childID &&
			candidate.ParentFrameID == frame.ID &&
			candidate.Path == pending.Reconciliation.ChildPath {
			child = &candidate
			break
		}
	}
	if child == nil || !child.Status.Terminal() {
		return ProgramTransition{}, false, nil
	}

	result, childErr := workflowFrameResult(*child)
	if childErr != nil {
		return i.beginReconciliationReview(
			frame,
			execution,
			program,
			state,
			childErr,
		)
	}
	switch result.Exit {
	case WorkflowExitNormal, WorkflowExitReturnBlock:
	default:
		return i.beginReconciliationReview(
			frame,
			execution,
			program,
			state,
			&SanitizedError{
				Code:     "reconciliation_control_invalid",
				Category: ErrorCategoryValidation,
				Message:  "reconciliation block returned unsupported control flow",
			},
		)
	}

	reconciled, err := DecodeCanonicalValue[WorkflowReconciliationResult](result.Value)
	if err != nil {
		return i.beginReconciliationReview(
			frame,
			execution,
			program,
			state,
			&SanitizedError{
				Code:     "reconciliation_result_invalid",
				Category: ErrorCategoryValidation,
				Message:  err.Error(),
			},
		)
	}
	if err := reconciled.Validate(); err != nil {
		return i.beginReconciliationReview(
			frame,
			execution,
			program,
			state,
			&SanitizedError{
				Code:     "reconciliation_result_invalid",
				Category: ErrorCategoryValidation,
				Message:  err.Error(),
			},
		)
	}

	original := executionWorkItem(
		execution,
		pending.Reconciliation.OriginalWorkItemID,
	)
	if original == nil ||
		original.State != WorkItemStateReconciling {
		return ProgramTransition{}, true, errors.New(
			"reconciliation original work item is unavailable",
		)
	}

	switch reconciled.Decision {
	case ReconciliationDecisionApplied:
		step, stepErr := workflowStepForPath(program.Root, pending.Path)
		if stepErr != nil {
			return ProgramTransition{}, true, stepErr
		}
		reconciledOutput := &WorkItemOutput{
			Value:   reconciled.Output.Clone(),
			Receipt: cloneReconciliationReceipt(reconciled.Receipt),
		}
		if err := i.validateStepOutput(
			program,
			state,
			step,
			reconciled.Output,
		); err != nil {
			return ProgramTransition{}, true, err
		}
		if err := validateWorkflowCallReceipt(
			program,
			state,
			step,
			reconciledOutput,
		); err != nil {
			return ProgramTransition{}, true, err
		}
		if err := state.recordOutcome(WorkflowStepOutcome{
			ID:     step.ID,
			Status: WorkflowStepOutcomeSucceeded,
			Output: reconciled.Output.Clone(),
			Target: state.PendingTarget,
		}); err != nil {
			return ProgramTransition{}, true, err
		}

		state.Pending = nil
		state.PendingPath = ""
		state.PendingTarget = ""
		state.Deadline = nil
		state.Index++

		transition, err := i.drive(frame, execution, program, state)
		if err != nil {
			return ProgramTransition{}, true, err
		}
		transition.Reconciliations = append(
			transition.Reconciliations,
			ReconciliationSettlement{
				OriginalWorkItemID: original.ID,
				Decision:           ReconciliationDecisionApplied,
				Output:             reconciled.Output.Clone(),
				Receipt:            cloneReconciliationReceipt(reconciled.Receipt),
			},
		)
		return transition, true, nil

	case ReconciliationDecisionNotApplied:
		state.Pending = nil
		transition, err := workflowTransition(state, nil, nil, nil)
		if err != nil {
			return ProgramTransition{}, true, err
		}
		transition.Reconciliations = []ReconciliationSettlement{{
			OriginalWorkItemID: original.ID,
			Decision:           ReconciliationDecisionNotApplied,
		}}
		return transition, true, nil

	case ReconciliationDecisionUnknown:
		return i.beginReconciliationReview(
			frame,
			execution,
			program,
			state,
			reconciled.Error,
		)

	default:
		return ProgramTransition{}, true, errors.New(
			"reconciliation decision is unavailable",
		)
	}
}

func (i *WorkflowInterpreter) beginReconciliationReview(
	frame ProgramFrame,
	execution Execution,
	program WorkflowProgram,
	state WorkflowState,
	failure *SanitizedError,
) (ProgramTransition, bool, error) {
	if failure == nil {
		failure = &SanitizedError{
			Code:     reconciliationUnknownCode,
			Category: ErrorCategoryUncertainty,
			Message:  reconciliationUnknownMessage,
		}
	}
	pending := state.Pending
	if pending == nil ||
		pending.Kind != WorkflowPendingReconciliation ||
		pending.Reconciliation == nil {
		return ProgramTransition{}, true, errors.New(
			"workflow reconciliation progress is unavailable",
		)
	}

	pending.Reconciliation.Phase = WorkflowReconciliationPhaseReview
	pending.Reconciliation.ReviewError = cloneReconciliationError(failure)
	if pending.Reconciliation.ChildPath == "" {
		pending.Reconciliation.ChildPath =
			pending.Path + "/reconciliation/child"
	}
	transition, err := i.driveReconciliation(
		frame,
		execution,
		program,
		state,
	)
	if err != nil {
		return ProgramTransition{}, true, err
	}
	transition.Reconciliations = append(
		transition.Reconciliations,
		ReconciliationSettlement{
			OriginalWorkItemID: pending.Reconciliation.OriginalWorkItemID,
			Decision:           ReconciliationDecisionUnknown,
			Error:              cloneReconciliationError(failure),
		},
	)
	return transition, true, nil
}

func (i *WorkflowInterpreter) driveReconciliation(
	frame ProgramFrame,
	execution Execution,
	_ WorkflowProgram,
	state WorkflowState,
) (ProgramTransition, error) {
	pending := state.Pending
	if pending == nil ||
		pending.Kind != WorkflowPendingReconciliation ||
		pending.Reconciliation == nil {
		return ProgramTransition{}, errors.New(
			"workflow reconciliation progress is unavailable",
		)
	}
	progress := pending.Reconciliation

	switch progress.Phase {
	case WorkflowReconciliationPhaseIntent:
		if executionWorkItem(execution, progress.IntentWorkItemID) != nil {
			return workflowTransition(state, nil, nil, nil)
		}
		return workflowTransition(
			state,
			nil,
			nil,
			[]WorkItemIntent{{
				ID:          progress.IntentWorkItemID,
				ProgramPath: pending.Path + "/reconciliation",
				Cleanup:     true,
				Type:        WorkItemTypeReconciliation,
				Automatic:   true,
				Effect:      EffectContract{Kind: EffectKindRead},
				Input: WorkItemPayload{
					Reconciliation: &ReconciliationWork{
						OriginalWorkItemID: progress.OriginalWorkItemID,
						Operation:          progress.Operation,
					},
				},
			}},
		)

	case WorkflowReconciliationPhaseChild:
		return workflowTransition(state, nil, nil, nil)

	case WorkflowReconciliationPhaseReview:
		if progress.ReviewWorkItemID != "" &&
			executionWorkItem(execution, progress.ReviewWorkItemID) != nil {
			return workflowTransition(state, nil, nil, nil)
		}

		reviewID := WorkItemID(
			string(frame.ID) + ":" +
				string(progress.OriginalWorkItemID) +
				":reconciliation-review",
		)
		progress.ReviewWorkItemID = reviewID
		summary := MustCanonicalValue(WorkflowReconciliationReview{
			OriginalWorkItemID: progress.OriginalWorkItemID,
			Message:            "reconciliation requires recovery review",
			Error:              cloneReconciliationError(progress.ReviewError),
		})
		return workflowTransition(
			state,
			nil,
			nil,
			[]WorkItemIntent{{
				ID:          reviewID,
				ProgramPath: pending.Path + "/reconciliation/review",
				Cleanup:     true,
				Type:        WorkItemTypeWait,
				Automatic:   true,
				Effect:      EffectContract{Kind: EffectKindRead},
				Input: WorkItemPayload{
					Wait: &WaitWork{
						Request: WaitRequest{
							Kind:    WaitKindRecovery,
							Summary: summary,
							AllowedResolutions: []WaitResolution{
								WaitResolutionContinue,
								WaitResolutionCancel,
							},
						},
					},
				},
			}},
		)

	default:
		return ProgramTransition{}, errors.New(
			"workflow reconciliation phase is unavailable",
		)
	}
}

func workflowOperationFacts(
	execution Execution,
	item WorkItem,
	targetName string,
) (WorkflowOperationFacts, error) {
	if strings.TrimSpace(targetName) == "" {
		return WorkflowOperationFacts{}, errors.New(
			"reconciliation operation has no selected target",
		)
	}
	input, err := reconciliationInput(item)
	if err != nil {
		return WorkflowOperationFacts{}, err
	}
	if item.Error == nil {
		return WorkflowOperationFacts{}, errors.New(
			"reconciliation operation has no uncertainty error",
		)
	}

	facts := WorkflowOperationFacts{
		WorkItemID:     item.ID,
		Input:          input,
		IdempotencyKey: item.Effect.IdempotencyKey,
		Target: WorkflowOperationTargetFacts{
			Name:         targetName,
			WorkItemType: item.Type,
		},
		Uncertainty: cloneReconciliationError(item.Error),
	}
	if attempt := latestAttemptForWorkItem(execution, item.ID); attempt != nil {
		facts.AttemptID = attempt.ID
	}
	if item.Binding != nil {
		binding := cloneReconciliationBinding(*item.Binding)
		facts.Target.Binding = &binding
	}
	if item.Output != nil && item.Output.Receipt != nil {
		facts.Receipt = cloneReconciliationReceipt(item.Output.Receipt)
	}
	if err := facts.Validate(); err != nil {
		return WorkflowOperationFacts{}, err
	}
	return facts, nil
}

func reconciliationInput(item WorkItem) (CanonicalValue, error) {
	switch item.Type {
	case WorkItemTypeTool:
		if item.Input.Tool == nil {
			return nil, errors.New("reconciliation tool input is unavailable")
		}
		return item.Input.Tool.Call.Arguments.Clone(), nil

	case WorkItemTypeService:
		if item.Input.Service == nil {
			return nil, errors.New("reconciliation service input is unavailable")
		}
		return item.Input.Service.Input.Clone(), nil

	case WorkItemTypeActivity:
		if item.Input.Activity == nil {
			return nil, errors.New("reconciliation activity input is unavailable")
		}
		return item.Input.Activity.Input.Clone(), nil

	default:
		return nil, errors.New(
			"reconciliation is unavailable for this work item type",
		)
	}
}

func latestAttemptForWorkItem(
	execution Execution,
	workItemID WorkItemID,
) *Attempt {
	for _, v := range slices.Backward(execution.Attempts) {
		if v.WorkItemID != workItemID {
			continue
		}
		value := v
		return &value
	}
	return nil
}

func reconciliationFailure(
	code string,
	message string,
	cause *SanitizedError,
) *SanitizedError {
	value := &SanitizedError{
		Code:     code,
		Category: ErrorCategoryUncertainty,
		Message:  message,
	}
	if cause != nil {
		value.Causes = []SanitizedError{*cloneReconciliationError(cause)}
	}
	return value
}

func cloneReconciliationError(
	value *SanitizedError,
) *SanitizedError {
	if value == nil {
		return nil
	}
	output := *value
	output.Details = value.Details.Clone()
	output.Causes = make([]SanitizedError, len(value.Causes))
	for index := range value.Causes {
		copyValue := cloneReconciliationError(&value.Causes[index])
		if copyValue != nil {
			output.Causes[index] = *copyValue
		}
	}
	return &output
}

func cloneReconciliationReceipt(
	value *ExternalReceipt,
) *ExternalReceipt {
	if value == nil {
		return nil
	}
	output := *value
	return &output
}

func cloneReconciliationBinding(
	value SemanticBinding,
) SemanticBinding {
	output := value
	output.Artifacts = append([]ArtifactPin(nil), value.Artifacts...)
	return output
}
