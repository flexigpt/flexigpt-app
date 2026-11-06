package sessionv3

import (
	"errors"
	"fmt"
	"strings"
)

func (r WorkflowTargetSetResolution) Validate() error {
	if err := validateID("workflow target set resolution name", r.Name); err != nil {
		return err
	}
	if err := validateID(
		"workflow target set resolution WorkItem ID",
		string(r.WorkItemID),
	); err != nil {
		return err
	}
	if len(r.Members) == 0 {
		return errors.New("workflow target set resolution has no members")
	}
	seen := make(map[string]struct{}, len(r.Members))
	for _, member := range r.Members {
		if err := validateID(
			"workflow target set resolution member",
			member,
		); err != nil {
			return err
		}
		if _, duplicate := seen[member]; duplicate {
			return errors.New("workflow target set resolution member is duplicated")
		}
		seen[member] = struct{}{}
	}
	return nil
}

func (i *WorkflowInterpreter) beginTargetSetResolution(
	frame ProgramFrame,
	program WorkflowProgram,
	state *WorkflowState,
	path string,
	call WorkflowCall,
) (ProgramTransition, bool, error) {
	if state == nil || call.AllowedTargetSet == "" {
		return ProgramTransition{}, false, nil
	}
	targetSet, found := program.TargetSets[call.AllowedTargetSet]
	if !found {
		return ProgramTransition{}, false, fmt.Errorf(
			"workflow target set %q is unavailable",
			call.AllowedTargetSet,
		)
	}
	if targetSet.Resolved == nil {
		return ProgramTransition{}, false, nil
	}
	if _, found := workflowTargetSetState(*state, call.AllowedTargetSet); found {
		return ProgramTransition{}, false, nil
	}
	if state.TargetSetResolution != nil {
		if state.TargetSetResolution.Name != call.AllowedTargetSet {
			return ProgramTransition{}, false, errors.New(
				"workflow target set resolver overlaps another resolver",
			)
		}
		t, err := workflowTransition(*state, nil, nil, nil)
		if err != nil {
			return ProgramTransition{}, false, err
		}
		return t, true, nil
	}

	members, err := i.evaluateResolverMembers(
		program,
		*state,
		call.AllowedTargetSet,
		targetSet,
	)
	if err != nil {
		return ProgramTransition{}, false, err
	}
	workID := WorkItemID(
		string(frame.ID) + ":target-set:" +
			call.AllowedTargetSet + ":" + strings.ReplaceAll(path, "/", ":"),
	)
	if err := validateID(
		"workflow target resolver WorkItem ID",
		string(workID),
	); err != nil {
		return ProgramTransition{}, false, err
	}

	state.TargetSetResolution = &WorkflowTargetSetResolution{
		Name:       call.AllowedTargetSet,
		WorkItemID: workID,
		Members:    append([]string(nil), members...),
	}
	transition, err := workflowTransition(
		*state,
		nil,
		nil,
		[]WorkItemIntent{{
			ID:          workID,
			ProgramPath: path + "/target-set/" + call.AllowedTargetSet,
			Type:        WorkItemTypeTargetResolver,
			Automatic:   true,
			Effect:      EffectContract{Kind: EffectKindRead},
			Input: WorkItemPayload{
				TargetResolver: &TargetResolverWork{
					TargetSet: call.AllowedTargetSet,
					Members:   append([]string(nil), members...),
				},
			},
		}},
	)
	if err != nil {
		return ProgramTransition{}, false, err
	}
	return transition, true, nil
}

func (i *WorkflowInterpreter) evaluateResolverMembers(
	program WorkflowProgram,
	state WorkflowState,
	name string,
	targetSet WorkflowTargetSet,
) ([]string, error) {
	if targetSet.Resolved == nil {
		return nil, errors.New("workflow target resolver is unavailable")
	}
	value, err := i.evaluate(
		targetSet.Resolved.Value,
		program,
		state,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"evaluate workflow target set %q: %w",
			name,
			err,
		)
	}
	resolved, err := DecodeCanonicalValue[[]string](value)
	if err != nil {
		return nil, fmt.Errorf(
			"workflow target set %q must resolve to a string array",
			name,
		)
	}
	if len(resolved) == 0 {
		return nil, fmt.Errorf(
			"workflow target set %q resolved no targets",
			name,
		)
	}

	output := make([]string, 0, len(resolved))
	seen := make(map[string]struct{}, len(resolved))
	for _, member := range resolved {
		if err := validateID(
			"workflow target set resolved target",
			member,
		); err != nil {
			return nil, err
		}
		if _, found := program.Targets[member]; !found {
			return nil, fmt.Errorf(
				"workflow target set %q resolved unknown target %q",
				name,
				member,
			)
		}
		if _, duplicate := seen[member]; duplicate {
			continue
		}
		if targetSet.Resolved.Constraints != nil {
			candidate := MustCanonicalValue(member)
			constrained := program
			constrained.Scope.Item = &candidate
			matched, matchErr := i.condition(
				*targetSet.Resolved.Constraints,
				constrained,
				state,
			)
			if matchErr != nil {
				return nil, fmt.Errorf(
					"evaluate workflow target set constraint: %w",
					matchErr,
				)
			}
			if !matched {
				continue
			}
		}
		seen[member] = struct{}{}
		output = append(output, member)
	}
	if len(output) == 0 {
		return nil, fmt.Errorf(
			"workflow target set %q has no permitted targets",
			name,
		)
	}
	return output, nil
}

func (i *WorkflowInterpreter) settleTargetSetResolver(
	state *WorkflowState,
	program WorkflowProgram,
	execution Execution,
	frameID ProgramFrameID,
	workID WorkItemID,
) (bool, error) {
	if state == nil || state.TargetSetResolution == nil ||
		state.TargetSetResolution.WorkItemID != workID {
		return false, nil
	}
	item := executionWorkItem(execution, workID)
	if item == nil || item.FrameID != frameID {
		return true, errors.New("workflow target resolver WorkItem is unavailable")
	}
	if item.Type != WorkItemTypeTargetResolver {
		return true, errors.New("workflow target resolver WorkItem has another type")
	}
	if item.State != WorkItemStateCompleted ||
		item.Output == nil ||
		item.Output.TargetResolver == nil {
		return true, errors.New("workflow target resolver WorkItem did not complete")
	}
	output := item.Output.TargetResolver
	if output.TargetSet != state.TargetSetResolution.Name {
		return true, errors.New("workflow target resolver output target set differs")
	}
	if err := output.Validate(); err != nil {
		return true, err
	}
	targetSet, found := program.TargetSets[output.TargetSet]
	if !found || targetSet.Resolved == nil {
		return true, errors.New("workflow target resolver declaration is unavailable")
	}
	if !sameWorkflowMembers(
		state.TargetSetResolution.Members,
		output.Members,
	) {
		return true, errors.New("workflow target resolver output changed its persisted membership")
	}
	for _, member := range output.Members {
		if _, found := program.Targets[member]; !found {
			return true, errors.New("workflow target resolver output contains an unknown target")
		}
	}
	if previous, found := workflowTargetSetState(
		*state,
		output.TargetSet,
	); found {
		if !sameWorkflowMembers(previous.Members, output.Members) {
			return true, errors.New("workflow target set membership changed after resolution")
		}
		state.TargetSetResolution = nil
		return true, nil
	}
	state.TargetSets = append(state.TargetSets, WorkflowTargetSetState{
		Name:       output.TargetSet,
		Members:    append([]string(nil), output.Members...),
		ResolvedAt: state.LogicalTime.UTC(),
	})
	state.TargetSetResolution = nil
	return true, nil
}

func sameWorkflowMembers(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
