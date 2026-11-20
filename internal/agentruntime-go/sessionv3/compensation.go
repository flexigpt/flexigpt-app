package sessionv3

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

type CompensationState string

const (
	CompensationStatePending     CompensationState = "pending"
	CompensationStateDispatching CompensationState = "dispatching"
	CompensationStateCompleted   CompensationState = "completed"
	CompensationStateFailed      CompensationState = "failed"
	CompensationStateUncertain   CompensationState = "uncertain"
	CompensationStateSkipped     CompensationState = "skipped"
)

type (
	EffectRecordID          string
	CompensationErrorPolicy string
)

const (
	CompensationErrorPolicyContinue CompensationErrorPolicy = "continue"
	CompensationErrorPolicyStop     CompensationErrorPolicy = "stop"
)

type WorkflowCompletedOperationFacts struct {
	OriginalWorkItemID     WorkItemID                   `json:"originalWorkItemID"`
	OriginalAttemptID      AttemptID                    `json:"originalAttemptID"`
	OriginalFrameID        ProgramFrameID               `json:"originalFrameID"`
	EffectID               EffectRecordID               `json:"effectID"`
	Target                 WorkflowOperationTargetFacts `json:"target"`
	Effect                 EffectContract               `json:"effect"`
	EffectState            EffectState                  `json:"effectState"`
	Input                  jsonutil.CanonicalValue      `json:"input"`
	Output                 jsonutil.CanonicalValue      `json:"output"`
	IdempotencyKey         string                       `json:"idempotencyKey"`
	CompensationWorkItemID WorkItemID                   `json:"compensationWorkItemID"`
	CompensationFrameID    ProgramFrameID               `json:"compensationFrameID"`

	Receipt *ExternalReceipt `json:"receipt,omitempty"`
}

type EffectRecord struct {
	ID                     EffectRecordID                  `json:"id"`
	OriginalWorkItemID     WorkItemID                      `json:"originalWorkItemID"`
	OriginalFrameID        ProgramFrameID                  `json:"originalFrameID"`
	SagaID                 string                          `json:"sagaID"`
	ConfirmationOrder      uint64                          `json:"confirmationOrder"`
	Dependencies           []WorkItemID                    `json:"dependencies,omitempty"`
	Completed              WorkflowCompletedOperationFacts `json:"completed"`
	Compensation           *WorkflowBlock                  `json:"compensation,omitempty"`
	CompensationPolicy     CompensationErrorPolicy         `json:"compensationPolicy"`
	State                  CompensationState               `json:"state"`
	CompensationWorkItemID WorkItemID                      `json:"compensationWorkItemID,omitempty"`
	CompensationFrameID    ProgramFrameID                  `json:"compensationFrameID,omitempty"`
	Receipt                *ExternalReceipt                `json:"receipt,omitempty"`
	Error                  *SanitizedError                 `json:"error,omitempty"`
}

type CompensationSettlement struct {
	EffectID   EffectRecordID    `json:"effectID"`
	WorkItemID WorkItemID        `json:"workItemID,omitempty"`
	FrameID    ProgramFrameID    `json:"frameID,omitempty"`
	State      CompensationState `json:"state"`
	Receipt    *ExternalReceipt  `json:"receipt,omitempty"`
	Error      *SanitizedError   `json:"error,omitempty"`
}

func (s CompensationErrorPolicy) Validate() error {
	switch s {
	case CompensationErrorPolicyContinue, CompensationErrorPolicyStop:
		return nil
	default:
		return errors.New("compensation error policy is invalid")
	}
}

func (s CompensationState) Validate() error {
	switch s {
	case CompensationStatePending,
		CompensationStateDispatching,
		CompensationStateCompleted,
		CompensationStateFailed,
		CompensationStateUncertain,
		CompensationStateSkipped:
		return nil
	default:
		return errors.New("compensation state is invalid")
	}
}

func (f WorkflowCompletedOperationFacts) Validate() error {
	if err := validateID(
		"completed operation work item ID",
		string(f.OriginalWorkItemID),
	); err != nil {
		return err
	}
	if err := validateID("completed operation effect ID", string(f.EffectID)); err != nil {
		return err
	}
	if !f.Input.Valid() || !f.Output.Valid() {
		return errors.New("completed operation facts are invalid")
	}
	if f.OriginalAttemptID != "" {
		if err := validateID(
			"completed operation attempt ID",
			string(f.OriginalAttemptID),
		); err != nil {
			return err
		}
	}
	if f.OriginalFrameID != "" {
		if err := validateID(
			"completed operation frame ID",
			string(f.OriginalFrameID),
		); err != nil {
			return err
		}
	}
	if f.Target.Name != "" {
		if err := f.Target.Validate(); err != nil {
			return err
		}
	}
	if f.Effect.Kind != "" {
		if err := f.Effect.Validate(); err != nil {
			return err
		}
	}
	if err := f.EffectState.Validate(); err != nil {
		return err
	}
	if f.Receipt != nil {
		return f.Receipt.Validate()
	}
	return nil
}

func (e EffectRecord) Validate() error {
	if err := validateID("effect record ID", string(e.ID)); err != nil {
		return err
	}
	if err := validateID(
		"effect record work item ID",
		string(e.OriginalWorkItemID),
	); err != nil {
		return err
	}
	if err := validateID(
		"effect record frame ID",
		string(e.OriginalFrameID),
	); err != nil {
		return err
	}
	if err := validateID("effect record saga ID", e.SagaID); err != nil {
		return err
	}
	if e.ConfirmationOrder == 0 {
		return errors.New("effect record confirmation order is invalid")
	}
	if err := e.Completed.Validate(); err != nil {
		return err
	}
	if e.Completed.OriginalWorkItemID != e.OriginalWorkItemID ||
		e.Completed.EffectID != e.ID {
		return errors.New("effect record completed facts differ from identity")
	}
	if e.Completed.OriginalFrameID != "" &&
		e.Completed.OriginalFrameID != e.OriginalFrameID {
		return errors.New("effect record completed frame differs from identity")
	}
	if e.Completed.CompensationWorkItemID != "" &&
		e.Completed.CompensationWorkItemID != e.CompensationWorkItemID {
		return errors.New("effect record completed compensation work differs from identity")
	}
	if e.Completed.CompensationFrameID != e.CompensationFrameID {
		return errors.New("effect record completed compensation frame differs from identity")
	}
	if e.Compensation == nil {
		return errors.New("effect record has no compensation block")
	}
	if err := e.Compensation.Validate(); err != nil {
		return err
	}
	if err := e.CompensationPolicy.Validate(); err != nil {
		return err
	}
	if err := e.State.Validate(); err != nil {
		return err
	}
	if e.CompensationWorkItemID != "" {
		if err := validateID(
			"compensation work item ID",
			string(e.CompensationWorkItemID),
		); err != nil {
			return err
		}
	}
	if e.CompensationFrameID != "" {
		if err := validateID(
			"compensation frame ID",
			string(e.CompensationFrameID),
		); err != nil {
			return err
		}
	}
	if e.Receipt != nil {
		if err := e.Receipt.Validate(); err != nil {
			return err
		}
	}
	if e.Error != nil {
		if strings.TrimSpace(e.Error.Code) == "" ||
			strings.TrimSpace(e.Error.Message) == "" ||
			!e.Error.Details.Valid() {
			return errors.New("effect record compensation error is invalid")
		}
	}
	return nil
}

func (s CompensationSettlement) Validate() error {
	if err := validateID("compensation settlement effect ID", string(s.EffectID)); err != nil {
		return err
	}
	switch s.State {
	case CompensationStateCompleted:
		if s.Receipt == nil || s.Error != nil {
			return errors.New("completed compensation settlement is invalid")
		}
	case CompensationStateFailed, CompensationStateUncertain:
		if s.Error == nil {
			return errors.New("failed compensation settlement has no error")
		}
	default:
		return errors.New("compensation settlement state is invalid")
	}
	if s.Receipt != nil {
		if err := s.Receipt.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runtime) recordConfirmedEffectLocked(
	execution *Execution,
	item *WorkItem,
) error {
	if execution == nil || item == nil ||
		item.SagaID == "" ||
		item.EffectPlan == nil ||
		item.EffectPlan.Compensate == nil ||
		item.EffectState != EffectStateConfirmedApplied {
		return nil
	}

	for _, record := range execution.Effects {
		if record.OriginalWorkItemID == item.ID {
			return nil
		}
	}

	input, err := compensationInput(*item)
	if err != nil {
		return err
	}
	output, err := workflowWorkValue(*item)
	if err != nil {
		return err
	}
	target, err := completedOperationTarget(*item)
	if err != nil {
		return err
	}

	effectID := EffectRecordID("effect:" + string(item.ID))
	record := EffectRecord{
		ID:                 effectID,
		OriginalWorkItemID: item.ID,
		OriginalFrameID:    item.FrameID,
		SagaID:             item.SagaID,
		ConfirmationOrder:  uint64(len(execution.Effects) + 1),
		Dependencies:       append([]WorkItemID(nil), item.Dependencies...),
		Completed: WorkflowCompletedOperationFacts{
			OriginalWorkItemID: item.ID,
			OriginalFrameID:    item.FrameID,
			EffectID:           effectID,
			Target:             target,
			Effect:             item.Effect,
			EffectState:        item.EffectState,
			Input:              input,
			Output:             output,
			Receipt:            cloneReconciliationReceipt(item.Output.Receipt),
			IdempotencyKey:     item.Effect.IdempotencyKey,
		},
		Compensation:       cloneWorkflowBlock(item.EffectPlan.Compensate),
		CompensationPolicy: item.CompensationPolicy,
		State:              CompensationStatePending,
	}
	if attempt := latestAttemptForWorkItem(*execution, item.ID); attempt != nil {
		record.Completed.OriginalAttemptID = attempt.ID
	}
	if err := record.Validate(); err != nil {
		return err
	}
	execution.Effects = append(execution.Effects, record)
	return nil
}

func completedOperationTarget(
	item WorkItem,
) (WorkflowOperationTargetFacts, error) {
	target := WorkflowOperationTargetFacts{
		WorkItemType: item.Type,
	}
	if item.Binding != nil {
		value := cloneReconciliationBinding(*item.Binding)
		target.Binding = &value
	}

	switch item.Type {
	case WorkItemTypeTool:
		if item.Input.Tool == nil {
			return WorkflowOperationTargetFacts{}, errors.New(
				"completed tool target is unavailable",
			)
		}
		target.Name = item.Input.Tool.Call.Name
	case WorkItemTypeService:
		if item.Input.Service == nil {
			return WorkflowOperationTargetFacts{}, errors.New(
				"completed service target is unavailable",
			)
		}
		target.Name = item.Input.Service.ServiceID
	case WorkItemTypeActivity:
		if item.Input.Activity == nil {
			return WorkflowOperationTargetFacts{}, errors.New(
				"completed activity target is unavailable",
			)
		}
		target.Name = item.Input.Activity.ActivityID
	case WorkItemTypeChildCall:
		if item.Input.ChildCall == nil {
			return WorkflowOperationTargetFacts{}, errors.New(
				"completed child-call target is unavailable",
			)
		}
		target.Name = item.Input.ChildCall.Target
	default:
		return WorkflowOperationTargetFacts{}, errors.New(
			"completed work item cannot have a compensation target",
		)
	}
	return target, target.Validate()
}

func compensationInput(item WorkItem) (jsonutil.CanonicalValue, error) {
	switch item.Type {
	case WorkItemTypeTool:
		if item.Input.Tool == nil {
			return nil, errors.New("compensation tool input is unavailable")
		}
		return item.Input.Tool.Call.Arguments.Clone(), nil
	case WorkItemTypeService:
		if item.Input.Service == nil {
			return nil, errors.New("compensation service input is unavailable")
		}
		return item.Input.Service.Input.Clone(), nil
	case WorkItemTypeActivity:
		if item.Input.Activity == nil {
			return nil, errors.New("compensation activity input is unavailable")
		}
		return item.Input.Activity.Input.Clone(), nil
	case WorkItemTypeChildCall:
		if item.Input.ChildCall == nil {
			return nil, errors.New(
				"compensation child call input is unavailable",
			)
		}
		return item.Input.ChildCall.Input.Clone(), nil
	default:
		return nil, errors.New("compensation is unavailable for this work item type")
	}
}

func cloneWorkflowBlock(value *WorkflowBlock) *WorkflowBlock {
	if value == nil {
		return nil
	}
	output := *value
	output.Steps = append([]WorkflowStep(nil), value.Steps...)
	output.Catch = append([]WorkflowCatchHandler(nil), value.Catch...)
	return &output
}

func effectRecordByID(
	execution *Execution,
	id EffectRecordID,
) *EffectRecord {
	if execution == nil {
		return nil
	}
	for index := range execution.Effects {
		if execution.Effects[index].ID == id {
			return &execution.Effects[index]
		}
	}
	return nil
}

func compensationRecordsInReverseDependencyOrder(
	execution Execution,
	sagaID string,
) ([]EffectRecordID, error) {
	records := make(map[EffectRecordID]EffectRecord)
	for _, record := range execution.Effects {
		if record.SagaID != sagaID ||
			record.State != CompensationStatePending ||
			record.Compensation == nil {
			continue
		}
		records[record.ID] = record
	}
	if len(records) == 0 {
		return nil, nil
	}

	remaining := make(map[EffectRecordID]EffectRecord, len(records))
	maps.Copy(remaining, records)
	output := make([]EffectRecordID, 0, len(remaining))

	for len(remaining) != 0 {
		candidates := make([]EffectRecord, 0, len(remaining))
		for _, candidate := range remaining {
			dependedOn := false
			for _, other := range remaining {
				if slices.Contains(other.Dependencies, candidate.OriginalWorkItemID) {
					dependedOn = true
				}
				if dependedOn {
					break
				}
			}
			if !dependedOn {
				candidates = append(candidates, candidate)
			}
		}
		if len(candidates) == 0 {
			return nil, errors.New("compensation dependency graph contains a cycle")
		}
		sort.Slice(candidates, func(left, right int) bool {
			if candidates[left].ConfirmationOrder !=
				candidates[right].ConfirmationOrder {
				return candidates[left].ConfirmationOrder >
					candidates[right].ConfirmationOrder
			}
			return candidates[left].ID > candidates[right].ID
		})
		selected := candidates[0]
		output = append(output, selected.ID)
		delete(remaining, selected.ID)
	}
	return output, nil
}

func (r *Runtime) markCompensationIntentLocked(
	execution *Execution,
	item *WorkItem,
) error {
	if execution == nil || item == nil ||
		item.Type != WorkItemTypeCompensation {
		return nil
	}
	if item.Input.Compensation == nil {
		return errors.New("compensation WorkItem has no compensation payload")
	}

	intent := item.Input.Compensation
	record := effectRecordByID(execution, intent.EffectID)
	if record == nil {
		return fmt.Errorf(
			"compensation effect record %q is unavailable",
			intent.EffectID,
		)
	}
	original := mutableWorkItem(execution, record.OriginalWorkItemID)
	if original == nil {
		return errors.New("compensation original WorkItem is unavailable")
	}

	record.State = CompensationStateDispatching
	record.CompensationWorkItemID = item.ID
	record.Completed.CompensationWorkItemID = item.ID
	record.Completed.CompensationFrameID = record.CompensationFrameID
	original.EffectState = EffectStateCompensating
	original.UpdatedAt = r.now()
	return nil
}

func (r *Runtime) applyCompensationSettlementsLocked(
	execution *Execution,
	values []CompensationSettlement,
) error {
	if len(values) == 0 {
		return nil
	}
	if execution == nil {
		return errors.New("compensation settlement execution is unavailable")
	}

	for _, value := range values {
		record := effectRecordByID(execution, value.EffectID)
		if record == nil {
			return fmt.Errorf(
				"compensation effect record %q is unavailable",
				value.EffectID,
			)
		}
		if record.State != CompensationStateDispatching {
			return fmt.Errorf(
				"compensation effect record %q is not dispatching",
				value.EffectID,
			)
		}

		record.State = value.State
		if value.WorkItemID != "" {
			record.CompensationWorkItemID = value.WorkItemID
		}
		if value.FrameID != "" {
			record.CompensationFrameID = value.FrameID
		}
		record.Receipt = cloneReconciliationReceipt(value.Receipt)
		record.Error = cloneWorkflowSanitizedError(value.Error)
		record.Completed.CompensationWorkItemID = record.CompensationWorkItemID
		record.Completed.CompensationFrameID = record.CompensationFrameID

		item := mutableWorkItem(execution, record.OriginalWorkItemID)
		if item == nil {
			return errors.New("compensation original work item is unavailable")
		}
		now := r.now()
		switch value.State {
		case CompensationStateCompleted:
			item.EffectState = EffectStateCompensated
		case CompensationStateFailed, CompensationStateUncertain:
			item.EffectState = EffectStateCompensationFailed
		default:
			return errors.New("compensation settlement state is unavailable")
		}
		item.UpdatedAt = now
	}
	return nil
}

func compensationPrimaryError(
	primary *SanitizedError,
	causes []SanitizedError,
) *SanitizedError {
	if primary == nil {
		primary = &SanitizedError{
			Code:     "saga_failed",
			Category: ErrorCategoryExecution,
			Message:  "saga failed without a primary error",
		}
	}
	output := cloneWorkflowSanitizedError(primary)
	for _, cause := range causes {
		output.Causes = append(
			output.Causes,
			cloneWorkflowErrorCause(cause),
		)
	}
	return output
}

func compensationFailureCause(
	record EffectRecord,
	errorValue *SanitizedError,
) SanitizedError {
	if errorValue != nil {
		return cloneWorkflowErrorCause(*errorValue)
	}
	return SanitizedError{
		Code:     "compensation_failed",
		Category: ErrorCategoryExecution,
		Message: "compensation did not produce a confirmed result for effect " +
			string(record.OriginalWorkItemID),
	}
}

func compensationReceipt(
	effectID EffectRecordID,
	frameID ProgramFrameID,
	now time.Time,
) *ExternalReceipt {
	return &ExternalReceipt{
		System:      "workflow-compensation",
		Reference:   string(effectID) + ":" + string(frameID),
		ConfirmedAt: now,
		EffectState: EffectStateCompensated,
	}
}
