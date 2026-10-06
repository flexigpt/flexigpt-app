package spec

import (
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/composition"
)

// ToolSelection selects one source-backed Tool capability for an inference
// request.
//
// UserArgSchemaInstance is SDK Tool configuration. It is interpreted only by
// SDK Tool hydration; Go Tool argument schemas remain declaration-owned and
// are supplied by the Tool Artifact itself.
type ToolSelection struct {
	ChoiceID              string                       `json:"choiceID"`
	Target                composition.CapabilityTarget `json:"target"`
	AutoExecute           bool                         `json:"autoExecute"`
	UserArgSchemaInstance jsonutil.JSONRawString       `json:"userArgSchemaInstance,omitempty"`
}

func (s ToolSelection) Validate() error {
	if err := spec.ValidateRequiredText(
		"Tool choice ID",
		s.ChoiceID,
		spec.MaxURIBytes,
	); err != nil {
		return err
	}
	return s.Target.Validate()
}
