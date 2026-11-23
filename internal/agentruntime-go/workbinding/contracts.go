package workbinding

import (
	"context"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

// Request is the neutral domain-execution request for one registered Service
// or Activity. It contains durable orchestration facts only. It does not
// expose Artifact Store handles, native clients, credentials, callbacks, or
// Workflow source declarations.
type Request struct {
	ExecutionID    sessionv3Runtime.ExecutionID
	WorkItemID     sessionv3Runtime.WorkItemID
	AttemptID      sessionv3Runtime.AttemptID
	Effect         sessionv3Runtime.EffectContract
	IdempotencyKey string

	Input   jsonutil.CanonicalValue
	Context jsonutil.CanonicalValue
	Ports   []sessionv3Runtime.PortInput
	Stream  sessionv3Runtime.WorkPortStream
}

// WaitRequest is the neutral domain request used to resolve one previously
// requested registered-binding wait.
type WaitRequest struct {
	Wait       sessionv3Runtime.Wait
	Resolution sessionv3Runtime.WaitResolution
	Response   jsonutil.CanonicalValue

	ExecutionID    sessionv3Runtime.ExecutionID
	WorkItemID     sessionv3Runtime.WorkItemID
	AttemptID      sessionv3Runtime.AttemptID
	Effect         sessionv3Runtime.EffectContract
	IdempotencyKey string

	Input   jsonutil.CanonicalValue
	Context jsonutil.CanonicalValue
	Ports   []sessionv3Runtime.PortInput
}

// BindResult allows a domain to prepare immediate work, a durable Runtime
// wait, or executable private state through one typed binding result.
type BindResult struct {
	SemanticBinding      sessionv3Runtime.SemanticBinding
	Wait                 *sessionv3Runtime.WaitRequest
	ImmediateOutput      *sessionv3Runtime.WorkItemOutput
	ImmediateEffectState *sessionv3Runtime.EffectState
	Private              any
}

// Binding is implemented by owning Service and Activity domains. Runtime owns
// WorkItem lifecycle and persistence; a domain owns native request creation,
// external execution, native result conversion, and domain-private state.
type Binding interface {
	ID() string
	Version() string
	Effect() sessionv3Runtime.EffectContract
	Bind(
		ctx context.Context,
		request Request,
	) (BindResult, error)
	Execute(
		ctx context.Context,
		request Request,
		state any,
	) (sessionv3Runtime.WorkExecutionResult, error)
	ResolveWait(
		ctx context.Context,
		request WaitRequest,
		state any,
	) (sessionv3Runtime.WaitResolutionResult, error)
}

// PortCapabilities is an explicit opt-in transport declaration. Final output
// delivery remains Runtime-owned.
type PortCapabilities struct {
	Context          bool
	InputPorts       bool
	FinalOutputPorts bool
	OutputStream     bool
}

type PortCapability interface {
	WorkflowPortCapabilities() PortCapabilities
}

// KeyedEffectCapability is an explicit target-domain declaration. It means
// that this binding has a native request mechanism for the stable key supplied
// in Request.IdempotencyKey. Runtime never adds that key to another domain's
// generic request grammar.
type KeyedEffectCapability interface {
	WorkflowNativeIdempotencyKeying() bool
}

func SupportsNativeIdempotencyKeying(value Binding) bool {
	capability, found := value.(KeyedEffectCapability)
	return found && capability.WorkflowNativeIdempotencyKeying()
}
