package sessionv3

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type readyExecutionWork struct {
	ExecutionID ExecutionID
	WorkItemIDs []WorkItemID
}

func mutableSessionExecution(
	session *Session,
	id ExecutionID,
) *Execution {
	if session == nil || id == "" {
		return nil
	}
	for index := range session.Executions {
		if session.Executions[index].ID == id {
			return &session.Executions[index]
		}
	}
	return nil
}

func cloneExecutionBudget(value ExecutionBudget) ExecutionBudget {
	output := value
	if value.Cost != nil {
		cost := *value.Cost
		output.Cost = &cost
	}
	return output
}

func programFrameDepth(value ProgramFrame) int {
	if value.Depth > 0 {
		return value.Depth
	}
	return 1
}

func effectiveProgramFrameBudget(
	execution Execution,
	frame ProgramFrame,
) ExecutionBudget {
	if frame.Budget != nil {
		return cloneExecutionBudget(*frame.Budget)
	}
	return cloneExecutionBudget(execution.Budget)
}

func workflowProgramFrameBudget(
	snapshot ProgramSnapshot,
) (*ExecutionBudget, error) {
	if snapshot.InterpreterID != WorkflowInterpreterID {
		return nil, nil
	}
	program, err := workflowProgramFromSnapshot(snapshot)
	if err != nil {
		return nil, err
	}
	value := cloneExecutionBudget(program.Limits)
	return &value, nil
}

func narrowProgramFrameBudget(
	parent ExecutionBudget,
	intent ProgramFrameIntent,
) (ExecutionBudget, error) {
	output := cloneExecutionBudget(parent)

	programBudget, err := workflowProgramFrameBudget(intent.Program)
	if err != nil {
		return ExecutionBudget{}, err
	}
	if programBudget != nil {
		output, err = narrowExecutionBudget(output, *programBudget)
		if err != nil {
			return ExecutionBudget{}, err
		}
	}
	if intent.Budget != nil {
		output, err = narrowExecutionBudget(output, *intent.Budget)
		if err != nil {
			return ExecutionBudget{}, err
		}
	}
	return output, nil
}

func childProgramFrameDepth(
	parent ProgramFrame,
	budget ExecutionBudget,
) (int, error) {
	depth := programFrameDepth(parent) + 1
	if budget.WorkflowDepth > 0 &&
		depth > budget.WorkflowDepth {
		return 0, errors.New("workflow depth budget is exceeded")
	}
	return depth, nil
}

func programFrameDeadline(
	now time.Time,
	parentDeadline *time.Time,
	intentDeadline *time.Time,
	budget ExecutionBudget,
) (*time.Time, error) {
	deadline := earlierDeadline(parentDeadline, intentDeadline)
	if budget.Duration == "" {
		return deadline, nil
	}
	budgetDeadline, err := budget.Duration.Deadline(now)
	if err != nil {
		return nil, err
	}
	return earlierDeadline(deadline, &budgetDeadline), nil
}

func workItemBudgetForFrame(
	execution Execution,
	frame ProgramFrame,
	requested *ExecutionBudget,
) (*ExecutionBudget, error) {
	if frame.Budget == nil && requested == nil {
		return nil, nil
	}

	inherited := effectiveProgramFrameBudget(execution, frame)
	if requested == nil {
		return &inherited, nil
	}

	value, err := narrowExecutionBudget(inherited, *requested)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func cloneConversationParticipant(
	value ConversationParticipant,
) ConversationParticipant {
	output := value
	if value.Agent != nil {
		agent := *value.Agent
		output.Agent = &agent
	}
	return output
}

func (r *Runtime) createIndependentChildExecutionLocked(
	session *Session,
	parent *Execution,
	parentFrame *ProgramFrame,
	intent ProgramFrameIntent,
) (*Execution, error) {
	if session == nil || parent == nil || parentFrame == nil {
		return nil, errors.New("independent child execution owner is unavailable")
	}
	if intent.Lifetime != ChildFrameLifetimeIndependent || intent.ExecutionID == "" {
		return nil, errors.New("independent child execution intent is incomplete")
	}
	if mutableSessionExecution(session, intent.ExecutionID) != nil {
		return nil, fmt.Errorf("child execution %q already exists", intent.ExecutionID)
	}

	interpreter := r.interpreterLocked(intent.Program)
	if interpreter == nil {
		return nil, fmt.Errorf("child interpreter %q is unavailable", intent.Program.InterpreterID)
	}
	if err := interpreter.ValidateSnapshot(intent.Program); err != nil {
		return nil, err
	}
	state, err := interpreter.InitialState(intent.Program)
	if err != nil {
		return nil, err
	}

	resources := []WorkflowResourceDefinition(nil)
	if intent.Program.InterpreterID == WorkflowInterpreterID {
		program, err := workflowProgramFromSnapshot(intent.Program)
		if err != nil {
			return nil, err
		}
		resources, err = initialExecutionResources(program.Resources)
		if err != nil {
			return nil, err
		}
	}

	ownership := ChildExecutionOwnership{}
	if intent.Ownership != nil {
		ownership = *intent.Ownership
		ownership.Budget = cloneExecutionBudget(intent.Ownership.Budget)
	}
	if err := ownership.Budget.Validate(); err != nil {
		return nil, err
	}
	parentBudget := effectiveProgramFrameBudget(*parent, *parentFrame)
	effectiveBudget, err := narrowExecutionBudget(
		parentBudget,
		ownership.Budget,
	)
	if err != nil {
		return nil, err
	}
	effectiveBudget, err = narrowProgramFrameBudget(
		effectiveBudget,
		intent,
	)
	if err != nil {
		return nil, err
	}

	now := r.now()
	depth, err := childProgramFrameDepth(*parentFrame, effectiveBudget)
	if err != nil {
		return nil, err
	}
	deadline, err := programFrameDeadline(
		now,
		parentFrame.Deadline,
		intent.Deadline,
		effectiveBudget,
	)
	if err != nil {
		return nil, err
	}

	rootID := parent.RootExecutionID
	if rootID == "" {
		rootID = parent.ID
	}
	frameID := intent.ID
	if frameID == "" {
		frameID = ProgramFrameID(r.newID())
	}
	handoffID := ""
	if intent.Handoff != nil {
		handoffID = intent.Handoff.ID
	}

	child := Execution{
		ID:                intent.ExecutionID,
		SessionID:         session.ID,
		ParentExecutionID: parent.ID,
		RootExecutionID:   rootID,
		ParentFrameID:     parentFrame.ID,
		Lifetime:          ExecutionChildLifetimeIndependent,
		Owner:             ownership.Owner,
		CancelWithParent:  ownership.CancelWithParent,
		ChildBudget:       cloneExecutionBudget(effectiveBudget),
		HandoffID:         handoffID,
		Revision:          1,
		State:             ExecutionStateRunning,
		CreatedAt:         now,
		UpdatedAt:         now,
		Deadline:          deadline,
		Budget:            effectiveBudget,
		Usage:             ExecutionBudgetUsage{},
		Cleanup:           CleanupState{},
		ProgramFrames: []ProgramFrame{{
			ID:        frameID,
			Depth:     depth,
			Program:   intent.Program,
			State:     state,
			Status:    ProgramFrameStateRunning,
			Path:      intent.Path,
			CreatedAt: now,
			UpdatedAt: now,
			Deadline:  deadline,
			Budget:    cloneBudget(&effectiveBudget),
			HandoffID: handoffID,
			Ports:     bindFramePortPlans(intent.Ports, parentFrame.ID, frameID),
		}},
		WorkItems:      []WorkItem{},
		Attempts:       []Attempt{},
		Waits:          []Wait{},
		Resources:      resources,
		ResourceLeases: []ResourceLease{},
		ResourceRates:  []ResourceRateWindow{},
	}

	parent.ChildExecutionIDs = append(parent.ChildExecutionIDs, child.ID)
	parentFrame.ChildExecutionIDs = append(parentFrame.ChildExecutionIDs, child.ID)
	session.Executions = append(session.Executions, child)
	created := &session.Executions[len(session.Executions)-1]
	if err := r.stageFramePortInputsLocked(created, &created.ProgramFrames[0]); err != nil {
		return nil, err
	}
	if inputs := created.ProgramFrames[0].PortInputs; len(inputs) != 0 &&
		created.ProgramFrames[0].Program.InterpreterID ==
			WorkflowInterpreterID {
		next, inputErr := workflowProgramWithPortInputs(
			created.ProgramFrames[0].Program,
			inputs,
		)
		if inputErr != nil {
			return nil, inputErr
		}
		created.ProgramFrames[0].Program = next
	}

	if intent.Handoff != nil {
		if err := r.applyConversationHandoffLocked(
			session,
			parent,
			parentFrame,
			created,
			created.ProgramFrames[0].ID,
			*intent.Handoff,
		); err != nil {
			return nil, err
		}
	}
	return created, nil
}

func (r *Runtime) applyConversationHandoffLocked(
	session *Session,
	parent *Execution,
	parentFrame *ProgramFrame,
	child *Execution,
	childFrameID ProgramFrameID,
	intent ConversationHandoffIntent,
) error {
	if session == nil || parent == nil || parentFrame == nil || child == nil {
		return errors.New("conversation handoff owner is unavailable")
	}
	if intent.ID == "" || intent.ConversationID == "" {
		return errors.New("conversation handoff intent is incomplete")
	}
	if err := validatePin(intent.TargetAgent); err != nil {
		return err
	}

	now := r.now()
	source := ConversationParticipant{
		Owner:       intent.Owner,
		ExecutionID: parent.ID,
		FrameID:     parentFrame.ID,
	}
	targetAgent := intent.TargetAgent
	target := ConversationParticipant{
		Owner:       intent.Owner,
		ExecutionID: child.ID,
		FrameID:     childFrameID,
		Agent:       &targetAgent,
	}

	for index := range session.ConversationOwnership {
		value := &session.ConversationOwnership[index]
		if value.ConversationID != intent.ConversationID {
			continue
		}
		if value.ActiveHandoffID != "" && value.ActiveHandoffID != intent.ID {
			return errors.New("conversation already has an active handoff")
		}
		if value.Owner.ExecutionID != "" {
			source = cloneConversationParticipant(value.Owner)
		}
		value.PreviousOwner = &source
		value.Owner = target
		value.ActiveHandoffID = intent.ID
		value.RestoreOnCompletion = intent.RestoreOnCompletion
		value.UpdatedAt = now
		value.CompletedAt = nil
		return nil
	}

	session.ConversationOwnership = append(
		session.ConversationOwnership,
		ConversationOwnershipState{
			ConversationID:      intent.ConversationID,
			Owner:               target,
			PreviousOwner:       &source,
			ActiveHandoffID:     intent.ID,
			RestoreOnCompletion: intent.RestoreOnCompletion,
			UpdatedAt:           now,
		},
	)
	return nil
}

func (r *Runtime) settleConversationHandoffLocked(
	session *Session,
	handoffID string,
) {
	if session == nil || handoffID == "" {
		return
	}
	now := r.now()
	for index := range session.ConversationOwnership {
		value := &session.ConversationOwnership[index]
		if value.ActiveHandoffID != handoffID {
			continue
		}
		if value.RestoreOnCompletion && value.PreviousOwner != nil {
			value.Owner = cloneConversationParticipant(*value.PreviousOwner)
		}
		value.ActiveHandoffID = ""
		value.CompletedAt = &now
		value.UpdatedAt = now
		return
	}
}

func (r *Runtime) settleTerminalExecutionLocked(
	session *Session,
	execution *Execution,
) {
	if session == nil || execution == nil || !execution.State.Terminal() {
		return
	}
	r.settleConversationHandoffLocked(session, execution.HandoffID)
	r.cancelExecutionDeadlineLocked(execution.ID)
	if execution.Outcome != nil {
		switch execution.Outcome.Kind {
		case ExecutionOutcomeFailed, ExecutionOutcomeCancelled:
			r.cancelOwnedChildExecutionsLocked(
				context.Background(),
				session,
				execution.ID,
				execution.Outcome.Message,
			)
		default:
		}
	}
	r.wakeCompletionExecutionWaitsLocked(session, *execution)
	r.drainTriggerQueuesLocked(context.Background(), session, execution.ID)
}

func completionExecutionResult(
	execution Execution,
) (CanonicalValue, *SanitizedError, bool) {
	var output CanonicalValue
	for _, frame := range execution.ProgramFrames {
		if frame.ParentFrameID == "" && len(frame.Output) != 0 {
			output = frame.Output.Clone()
			break
		}
	}
	value := MustCanonicalValue(completionWaitOutput{
		ExecutionID: execution.ID,
		Output:      output,
	})

	if execution.Outcome == nil {
		return value, nil, execution.State == ExecutionStateCompleted
	}
	switch execution.Outcome.Kind {
	case ExecutionOutcomeCompleted:
		return value, nil, true
	case ExecutionOutcomeCancelled:
		return value, &SanitizedError{
			Code:     completionCancelled,
			Category: ErrorCategoryCancellation,
			Message:  firstText(execution.Outcome.Message, "completed execution was cancelled"),
		}, true
	default:
		return value, &SanitizedError{
			Code:     firstText(execution.Outcome.Code, "completion_failed"),
			Category: ErrorCategoryExecution,
			Message:  firstText(execution.Outcome.Message, "completed execution failed"),
		}, true
	}
}

func (r *Runtime) wakeCompletionExecutionWaitsLocked(
	session *Session,
	completed Execution,
) {
	if session == nil || !completed.State.Terminal() {
		return
	}
	value, errorValue, confirmed := completionExecutionResult(completed)

	waiterIDs := make([]ExecutionID, 0, len(session.Executions))
	for _, execution := range session.Executions {
		if execution.ID != completed.ID {
			waiterIDs = append(waiterIDs, execution.ID)
		}
	}
	for _, waiterID := range waiterIDs {
		waiter := mutableSessionExecution(session, waiterID)
		if waiter == nil {
			continue
		}
		for index := range waiter.Waits {
			wait := &waiter.Waits[index]
			if wait.Status != WaitStatusPending ||
				wait.Request.Kind != WaitKindChild ||
				wait.Request.Completion == nil ||
				wait.Request.Completion.ExecutionID != completed.ID {
				continue
			}
			if err := validateWorkflowSchema(wait.Request.InputSchema, value); err != nil {
				continue
			}
			_ = r.resolveDurableWaitLocked(
				session,
				waiter,
				wait,
				value,
				&ExternalReceipt{
					System:      sessionV3Completion,
					Reference:   string(completed.ID),
					ConfirmedAt: r.now(),
				},
				errorValue,
				confirmed,
			)
		}
	}
}

func (r *Runtime) readyWorkForSessionLocked(
	session *Session,
) []readyExecutionWork {
	if session == nil {
		return nil
	}
	output := make([]readyExecutionWork, 0, len(session.Executions))
	for index := range session.Executions {
		execution := &session.Executions[index]
		ready := r.readyWorkLocked(*execution)
		if len(ready) == 0 {
			continue
		}
		output = append(output, readyExecutionWork{
			ExecutionID: execution.ID,
			WorkItemIDs: append([]WorkItemID(nil), ready...),
		})
	}
	return output
}

func (r *Runtime) armSessionExecutionTimersLocked(
	ctx context.Context,
	session *Session,
) {
	if r == nil || session == nil {
		return
	}
	for index := range session.Executions {
		execution := &session.Executions[index]
		r.armExecutionDeadlineLocked(ctx, session.ID, execution.ID, *execution)
		r.armCleanupDeadlineLocked(ctx, session.ID, execution.ID, *execution)
	}
}

func (r *Runtime) launchReadyWork(
	ctx context.Context,
	sessionID SessionID,
	values []readyExecutionWork,
) {
	for _, value := range values {
		r.launchWork(ctx, sessionID, value.ExecutionID, value.WorkItemIDs)
	}
}
