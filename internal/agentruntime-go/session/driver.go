package session

import "encoding/json"

type DriverEventKind string

const (
	DriverEventRunStarted          DriverEventKind = "runStarted"
	DriverEventModelFinished       DriverEventKind = "modelFinished"
	DriverEventToolSettled         DriverEventKind = "toolSettled"
	DriverEventToolWaitResolved    DriverEventKind = "toolWaitResolved"
	DriverEventExplicitExecution   DriverEventKind = "explicitExecution"
	DriverEventContinuationRequest DriverEventKind = "continuationRequest"
	DriverEventControlsChanged     DriverEventKind = "controlsChanged"
	DriverEventResumed             DriverEventKind = "resumed"
)

type DriverToolCallView struct {
	ID                ToolCallID      `json:"id"`
	OriginOperationID OperationID     `json:"originOperationID"`
	Required          bool            `json:"required"`
	ProviderManaged   bool            `json:"providerManaged"`
	Automatic         bool            `json:"automatic"`
	State             ToolCallState   `json:"state"`
	SelectedResult    *ToolResultKind `json:"selectedResult,omitempty"`
}

type DriverView struct {
	RunState               RunState             `json:"runState"`
	ActiveModelOperationID OperationID          `json:"activeModelOperationID,omitempty"`
	Controls               AutomationControls   `json:"controls"`
	RequestedCallIDs       []ToolCallID         `json:"requestedCallIDs,omitempty"`
	ContinuationRequested  bool                 `json:"continuationRequested"`
	ToolCalls              []DriverToolCallView `json:"toolCalls,omitempty"`
}

type DriverEvent struct {
	Kind              DriverEventKind `json:"kind"`
	OperationID       OperationID     `json:"operationID,omitempty"`
	ToolCallID        ToolCallID      `json:"toolCallID,omitempty"`
	ModelUsable       bool            `json:"modelUsable"`
	ModelHasToolCalls bool            `json:"modelHasToolCalls"`
	View              DriverView      `json:"view"`
}

type OperationIntentKind string

const (
	OperationIntentStartModel  OperationIntentKind = "startModel"
	OperationIntentExecuteTool OperationIntentKind = "executeTool"
	OperationIntentCompleteRun OperationIntentKind = "completeRun"
)

type OperationIntent struct {
	Kind       OperationIntentKind `json:"kind"`
	ToolCallID ToolCallID          `json:"toolCallID,omitempty"`
	Automatic  bool                `json:"automatic,omitempty"`
}

type DriverTransition struct {
	NextState    DriverState       `json:"nextState"`
	Operations   []OperationIntent `json:"operations,omitempty"`
	SetLifecycle *RunState         `json:"setLifecycle,omitempty"`
	Outcome      *RunOutcome       `json:"outcome,omitempty"`
}

type ExecutionDriver interface {
	Kind() string
	Version() uint32
	InitialState() DriverState
	Advance(s DriverState, e DriverEvent) (DriverTransition, error)
}

func encodeDriverState(kind string, version uint32, value any) DriverState {
	raw, _ := json.Marshal(value)
	return DriverState{
		Kind:    kind,
		Version: version,
		Data:    raw,
	}
}
