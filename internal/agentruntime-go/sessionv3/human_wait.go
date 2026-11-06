package sessionv3

import (
	"errors"
	"slices"
)

type HumanActor struct {
	ID    string   `json:"id"`
	Roles []string `json:"roles,omitempty"`
}

func (a HumanActor) Validate() error {
	if err := validateID("human actor ID", a.ID); err != nil {
		return err
	}

	seen := make(map[string]struct{}, len(a.Roles))
	for _, role := range a.Roles {
		if err := validateID("human actor role", role); err != nil {
			return err
		}
		if _, found := seen[role]; found {
			return errors.New("human actor role is duplicated")
		}
		seen[role] = struct{}{}
	}
	return nil
}

func cloneHumanActor(value *HumanActor) *HumanActor {
	if value == nil {
		return nil
	}

	output := *value
	output.Roles = append([]string(nil), value.Roles...)
	return &output
}

func humanWaitActorAuthorized(
	wait Wait,
	actor *HumanActor,
) error {
	if wait.Request.Kind != WaitKindHuman {
		return nil
	}
	if actor == nil {
		return errors.New("human wait resolution requires an actor")
	}
	if err := actor.Validate(); err != nil {
		return err
	}
	if wait.Request.Assignees == nil {
		return errors.New("human wait has no persisted assignees")
	}

	if slices.Contains(wait.Request.Assignees.Users, actor.ID) {
		return nil
	}

	for _, role := range actor.Roles {
		if slices.Contains(wait.Request.Assignees.Roles, role) {
			return nil
		}
	}

	return errors.New("human wait actor is not authorized")
}
