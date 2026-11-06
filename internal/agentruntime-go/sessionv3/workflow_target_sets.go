package sessionv3

import (
	"errors"
	"fmt"
	"time"
)

// WorkflowTargetSet is the typed runtime representation of either a static
// target collection or a resolver-backed collection. Resolver expressions are
// evaluated once at the first call boundary, and the resulting member set is
// persisted in WorkflowState.TargetSets before target dispatch begins.
type WorkflowTargetSet struct {
	Members  []string                   `json:"members,omitempty"`
	Resolved *WorkflowResolvedTargetSet `json:"resolved,omitempty"`
	Contract *WorkflowCallableInterface `json:"contract,omitempty"`
}

type WorkflowResolvedTargetSet struct {
	Value       WorkflowValue             `json:"value"`
	Constraints *WorkflowValue            `json:"constraints,omitempty"`
	Contract    WorkflowCallableInterface `json:"contract"`
}

type WorkflowTargetSetState struct {
	Name       string    `json:"name"`
	Members    []string  `json:"members"`
	ResolvedAt time.Time `json:"resolvedAt"`
}

func (s WorkflowTargetSet) Validate() error {
	if len(s.Members) == 0 && s.Resolved == nil {
		return errors.New("workflow target set has no members or resolver")
	}
	if len(s.Members) != 0 && s.Resolved != nil {
		return errors.New(
			"workflow target set cannot contain static members and a resolver",
		)
	}
	seen := make(map[string]struct{}, len(s.Members))
	for _, member := range s.Members {
		if err := validateID("workflow target set member", member); err != nil {
			return err
		}
		if _, duplicate := seen[member]; duplicate {
			return errors.New("workflow target set member is duplicated")
		}
		seen[member] = struct{}{}
	}
	if s.Contract != nil {
		if err := s.Contract.Validate(); err != nil {
			return err
		}
	}
	if s.Resolved != nil {
		if err := s.Resolved.Value.Validate(); err != nil {
			return err
		}
		if s.Resolved.Constraints != nil {
			if err := s.Resolved.Constraints.Validate(); err != nil {
				return err
			}
		}
		if err := s.Resolved.Contract.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (s WorkflowResolvedTargetSet) Validate() error {
	if err := s.Value.Validate(); err != nil {
		return err
	}
	if s.Constraints != nil {
		if err := s.Constraints.Validate(); err != nil {
			return err
		}
	}
	return s.Contract.Validate()
}

func (s WorkflowTargetSetState) Validate() error {
	if err := validateID("workflow target set name", s.Name); err != nil {
		return err
	}
	if s.ResolvedAt.IsZero() {
		return errors.New("workflow target set resolution time is required")
	}
	if len(s.Members) == 0 {
		return errors.New("workflow target set resolved no members")
	}
	seen := make(map[string]struct{}, len(s.Members))
	for _, member := range s.Members {
		if err := validateID("workflow target set resolved member", member); err != nil {
			return err
		}
		if _, duplicate := seen[member]; duplicate {
			return errors.New("workflow target set resolved member is duplicated")
		}
		seen[member] = struct{}{}
	}
	return nil
}

func validateWorkflowTargetSetStates(
	values []WorkflowTargetSetState,
) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if err := value.Validate(); err != nil {
			return err
		}
		if _, duplicate := seen[value.Name]; duplicate {
			return errors.New("workflow target set state is duplicated")
		}
		seen[value.Name] = struct{}{}
	}
	return nil
}

func workflowTargetSetState(
	state WorkflowState,
	name string,
) (*WorkflowTargetSetState, bool) {
	for index := range state.TargetSets {
		if state.TargetSets[index].Name != name {
			continue
		}
		copyValue := state.TargetSets[index]
		copyValue.Members = append([]string(nil), copyValue.Members...)
		return &copyValue, true
	}
	return nil, false
}

func (i *WorkflowInterpreter) resolveWorkflowCallTargets(
	program WorkflowProgram,
	state *WorkflowState,
	call WorkflowCall,
) ([]string, error) {
	if state == nil {
		return nil, errors.New("workflow state is unavailable")
	}
	if call.AllowedTargetSet == "" {
		return append([]string(nil), call.AllowedTargets...), nil
	}

	targetSet, found := program.TargetSets[call.AllowedTargetSet]
	if !found {
		return nil, fmt.Errorf(
			"workflow target set %q is unavailable",
			call.AllowedTargetSet,
		)
	}
	if targetSet.Resolved == nil {
		return append([]string(nil), targetSet.Members...), nil
	}
	if previous, found := workflowTargetSetState(
		*state,
		call.AllowedTargetSet,
	); found {
		return append([]string(nil), previous.Members...), nil
	}
	return nil, fmt.Errorf(
		"workflow target set %q has no durable resolver result",
		call.AllowedTargetSet,
	)
}

func workflowTargetSetContract(
	program WorkflowProgram,
	call WorkflowCall,
) *WorkflowCallableInterface {
	if call.AllowedTargetSet == "" {
		return call.SetContract
	}
	targetSet, found := program.TargetSets[call.AllowedTargetSet]
	if !found {
		return call.SetContract
	}
	if targetSet.Resolved != nil {
		return &targetSet.Resolved.Contract
	}
	return targetSet.Contract
}
