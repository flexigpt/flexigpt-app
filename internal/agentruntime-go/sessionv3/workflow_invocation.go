package sessionv3

import "errors"

// LLMInvocationPlan is the typed boundary between a Workflow call and a
// built-in single-model or Agent-loop ProgramFrame.
//
// Input and Context are arbitrary Workflow JSON only at this boundary. The
// setup-owned model binder decodes each value once into its concrete request
// type before it prepares provider input.
type LLMInvocationPlan struct {
	Input   CanonicalValue `json:"input"`
	Context CanonicalValue `json:"context,omitempty"`
}

func (p LLMInvocationPlan) Validate() error {
	if len(p.Input) == 0 || !p.Input.Valid() || !p.Context.Valid() {
		return errors.New("workflow llm invocation plan is invalid")
	}
	return nil
}

func cloneLLMInvocationPlan(
	value *LLMInvocationPlan,
) *LLMInvocationPlan {
	if value == nil {
		return nil
	}

	output := *value
	output.Input = value.Input.Clone()
	output.Context = value.Context.Clone()
	return &output
}

func workflowSelectionForPin(
	pin ArtifactPin,
) *ArtifactSelection {
	return &ArtifactSelection{
		RootID:           pin.RootID,
		ArtifactID:       pin.ArtifactID,
		ExpectedRevision: pin.Revision,
		ExpectedDigest:   pin.Digest,
	}
}
