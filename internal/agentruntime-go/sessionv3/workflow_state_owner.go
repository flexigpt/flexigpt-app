package sessionv3

import "errors"

// WorkflowStateOwner identifies the nearest durable state-owning boundary.
// It is propagated into child Workflow programs so inherited set operations
// retain the schema and owner identity that were active at frame creation.
type WorkflowStateOwner struct {
	ID     string         `json:"id"`
	Schema CanonicalValue `json:"schema"`
}

func (o WorkflowStateOwner) Validate() error {
	if err := validateID("workflow state owner ID", o.ID); err != nil {
		return err
	}
	if len(o.Schema) == 0 || !o.Schema.Valid() {
		return errors.New("workflow state owner schema is invalid")
	}
	return nil
}

func cloneWorkflowStateOwner(
	value *WorkflowStateOwner,
) *WorkflowStateOwner {
	if value == nil {
		return nil
	}
	output := *value
	output.Schema = value.Schema.Clone()
	return &output
}

func workflowStateOwnerForFrame(
	frame ProgramFrame,
	program WorkflowProgram,
) (*WorkflowStateOwner, error) {
	if frame.ID == "" {
		return nil, errors.New("workflow state owner frame is unavailable")
	}
	if program.Root.State != nil {
		owner := WorkflowStateOwner{
			ID:     "state-owner:" + string(frame.ID) + ":block",
			Schema: program.Root.State.Schema.Clone(),
		}
		if err := owner.Validate(); err != nil {
			return nil, err
		}
		return &owner, nil
	}
	if program.Scope.StateOwner == nil {
		return nil, nil
	}
	owner := cloneWorkflowStateOwner(program.Scope.StateOwner)
	if err := owner.Validate(); err != nil {
		return nil, err
	}
	return owner, nil
}

func workflowSyntheticStateOwner(
	frameID ProgramFrameID,
	path string,
	kind string,
	schema CanonicalValue,
) *WorkflowStateOwner {
	return &WorkflowStateOwner{
		ID:     "state-owner:" + string(frameID) + ":" + path + ":" + kind,
		Schema: schema.Clone(),
	}
}
