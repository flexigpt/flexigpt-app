package workflowv1

import "github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/core/declaration"

type Block struct {
	State   *State                  `json:"state,omitempty"`
	Listen  map[string]Subscription `json:"listen,omitempty"`
	Steps   []Step                  `json:"steps"`
	Output  Value                   `json:"output,omitempty"`
	Catch   []Handler               `json:"catch,omitempty"`
	Finally *Block                  `json:"finally,omitempty"`
}

type State struct {
	Schema  Schema `json:"schema"`
	Initial Value  `json:"initial"`
}

type Step struct {
	ID           string     `json:"id"`
	When         Value      `json:"when,omitempty"`
	Timeout      string     `json:"timeout,omitempty"`
	Retry        *Retry     `json:"retry,omitempty"`
	Effect       *Effect    `json:"effect,omitempty"`
	Claims       []Claim    `json:"claims,omitempty"`
	Limits       *Limits    `json:"limits,omitempty"`
	OutputSchema Schema     `json:"outputSchema,omitempty"`
	Call         *Call      `json:"call,omitempty"`
	Transform    *Transform `json:"transform,omitempty"`
	Set          *Set       `json:"set,omitempty"`
	Scope        *Block     `json:"scope,omitempty"`
	Choose       *Choose    `json:"choose,omitempty"`
	Parallel     *Parallel  `json:"parallel,omitempty"`
	Foreach      *Foreach   `json:"foreach,omitempty"`
	Repeat       *Repeat    `json:"repeat,omitempty"`
	Machine      *Machine   `json:"machine,omitempty"`
	Wait         *Wait      `json:"wait,omitempty"`
	Emit         *Emit      `json:"emit,omitempty"`
	Saga         *Saga      `json:"saga,omitempty"`
	Cancel       *Cancel    `json:"cancel,omitempty"`
	Return       *Return    `json:"return,omitempty"`
	Fail         *Fail      `json:"fail,omitempty"`
	Break        *Empty     `json:"break,omitempty"`
	Continue     *Empty     `json:"continue,omitempty"`
}

type Empty struct{}

type TargetKind string

const (
	TargetAgent    TargetKind = "agent"
	TargetTool     TargetKind = "tool"
	TargetModel    TargetKind = "model"
	TargetService  TargetKind = "service"
	TargetActivity TargetKind = "activity"
	TargetWorkflow TargetKind = "workflow"
)

// Reference selects a portable source-backed declaration.
// Server is the shared MCP configuration selector, not an MCP tool name.
type Reference struct {
	Name    string                  `json:"name"`
	Scope   declaration.LookupScope `json:"scope,omitempty"`
	Locator *declaration.Locator    `json:"locator,omitempty"`
	Server  string                  `json:"server,omitempty"`
}

// Target.Ref is the callable logical reference.
//
// For an MCP Tool, Kind is tool, Server identifies the MCP declaration,
// and Ref identifies the operation exposed by that server.
//
// Service and activity targets are binding requirements. They do not create
// synthetic Artifacts or acquire transport or credential authority.
type Target struct {
	Kind      TargetKind              `json:"kind"`
	Ref       string                  `json:"ref"`
	Scope     declaration.LookupScope `json:"scope,omitempty"`
	Locator   *declaration.Locator    `json:"locator,omitempty"`
	Server    *Reference              `json:"server,omitempty"`
	Interface *CallableInterface      `json:"interface,omitempty"`
}

type CallableInterface struct {
	InputSchema   Schema                  `json:"inputSchema"`
	OutputSchema  Schema                  `json:"outputSchema"`
	ErrorSchema   Schema                  `json:"errorSchema,omitempty"`
	ReceiptSchema Schema                  `json:"receiptSchema,omitempty"`
	Modes         []string                `json:"modes,omitempty"`
	Context       map[string]Schema       `json:"context,omitempty"`
	Ports         map[string]CallablePort `json:"ports,omitempty"`
	Effect        *EffectContract         `json:"effect,omitempty"`
	Limits        *Limits                 `json:"limits,omitempty"`
	Capabilities  []string                `json:"capabilities,omitempty"`
}

type CallablePort struct {
	Direction string `json:"direction"`
	Schema    Schema `json:"schema"`
	Stream    bool   `json:"stream,omitempty"`
}

type EffectContract struct {
	Kind       string `json:"kind"`
	Keyed      bool   `json:"keyed,omitempty"`
	Reconcile  bool   `json:"reconcile,omitempty"`
	Compensate bool   `json:"compensate,omitempty"`
}

// TargetSet is either an explicit member set or a set returned by explicit
// resolver work. Resolved.Value must refer to work declared in the flow;
// validation never invokes a resolver.
type TargetSet struct {
	Members   []string           `json:"members,omitempty"`
	Resolved  *ResolvedTargetSet `json:"resolved,omitempty"`
	Interface *CallableInterface `json:"interface,omitempty"`
}

type ResolvedTargetSet struct {
	Value       Value             `json:"value"`
	Interface   CallableInterface `json:"interface"`
	Constraints Value             `json:"constraints,omitempty"`
}

type Call struct {
	Target       Value                  `json:"target"`
	Allowed      string                 `json:"allowed,omitempty"`
	Interface    *CallableInterface     `json:"interface,omitempty"`
	Input        Value                  `json:"input"`
	Context      map[string]Value       `json:"context,omitempty"`
	Ports        map[string]PortBinding `json:"ports,omitempty"`
	Mode         string                 `json:"mode,omitempty"`
	Lifetime     string                 `json:"lifetime,omitempty"`
	Ownership    *Ownership             `json:"ownership,omitempty"`
	Conversation Value                  `json:"conversation,omitempty"`
	OnCompletion string                 `json:"onCompletion,omitempty"`
}

type Ownership struct {
	Owner            Value  `json:"owner"`
	CancelWithParent bool   `json:"cancelWithParent"`
	Budgets          Limits `json:"budgets"`
}

type PortBinding struct {
	Channel string `json:"channel"`
	Access  string `json:"access"`
}

type Transform struct {
	Value        Value  `json:"value"`
	OutputSchema Schema `json:"outputSchema,omitempty"`
}

type Set struct {
	Values map[string]Value `json:"values,omitempty"`
	Remove []string         `json:"remove,omitempty"`
}

type Choose struct {
	Cases   []Case `json:"cases"`
	Default *Block `json:"default,omitempty"`
}

type Case struct {
	When Value `json:"when"`
	Do   Block `json:"do"`
}

type Completion struct {
	Mode   string `json:"mode"`
	Quorum *int   `json:"quorum,omitempty"`
}

type Branch struct {
	When Value `json:"when,omitempty"`
	Do   Block `json:"do"`
}

type Parallel struct {
	Completion *Completion       `json:"completion,omitempty"`
	Errors     string            `json:"errors,omitempty"`
	Remaining  string            `json:"remaining,omitempty"`
	Branches   map[string]Branch `json:"branches"`
}

type Foreach struct {
	Items       Value       `json:"items"`
	ItemSchema  Schema      `json:"itemSchema"`
	Concurrency *int        `json:"concurrency,omitempty"`
	MaxItems    *int        `json:"maxItems,omitempty"`
	Completion  *Completion `json:"completion,omitempty"`
	Errors      string      `json:"errors,omitempty"`
	Remaining   string      `json:"remaining,omitempty"`
	Do          Block       `json:"do"`
}

type Repeat struct {
	Count         Value  `json:"count,omitempty"`
	While         Value  `json:"while,omitempty"`
	Until         Value  `json:"until,omitempty"`
	Lifetime      string `json:"lifetime,omitempty"`
	State         *State `json:"state,omitempty"`
	MaxIterations *int   `json:"maxIterations,omitempty"`
	Timeout       string `json:"timeout,omitempty"`
	OnExhausted   string `json:"onExhausted,omitempty"`
	Do            Block  `json:"do"`
}

type Machine struct {
	Initial string                  `json:"initial"`
	State   *State                  `json:"state,omitempty"`
	Listen  map[string]Subscription `json:"listen,omitempty"`
	States  map[string]MachineState `json:"states"`
}

type MachineState struct {
	Entry       *Block       `json:"entry,omitempty"`
	Exit        *Block       `json:"exit,omitempty"`
	Transitions []Transition `json:"transitions,omitempty"`
	Final       bool         `json:"final,omitempty"`
	Output      Value        `json:"output,omitempty"`
}

type Transition struct {
	Event string `json:"event,omitempty"`
	After string `json:"after,omitempty"`
	When  Value  `json:"when,omitempty"`
	Do    *Block `json:"do,omitempty"`
	To    string `json:"to"`
}

type Wait struct {
	For            string     `json:"for"`
	After          Value      `json:"after,omitempty"`
	At             Value      `json:"at,omitempty"`
	Subscription   string     `json:"subscription,omitempty"`
	Channel        string     `json:"channel,omitempty"`
	Cursor         Value      `json:"cursor,omitempty"`
	Match          Value      `json:"match,omitempty"`
	Consume        *bool      `json:"consume,omitempty"`
	Handle         Value      `json:"handle,omitempty"`
	Assignees      *Assignees `json:"assignees,omitempty"`
	Request        Value      `json:"request,omitempty"`
	ResponseSchema Schema     `json:"responseSchema,omitempty"`
}

type Assignees struct {
	Users Value `json:"users,omitempty"`
	Roles Value `json:"roles,omitempty"`
}

type Emit struct {
	To               Destination `json:"to"`
	Kind             string      `json:"kind,omitempty"`
	Data             Value       `json:"data"`
	Correlation      Value       `json:"correlation,omitempty"`
	DeduplicationKey Value       `json:"deduplicationKey,omitempty"`
}

type Destination struct {
	Channel string `json:"channel,omitempty"`
	Event   string `json:"event,omitempty"`
}

type Saga struct {
	CompensateOn       []string `json:"compensateOn,omitempty"`
	CompensationErrors string   `json:"compensationErrors,omitempty"`
	Concurrency        *int     `json:"concurrency,omitempty"`
	Do                 Block    `json:"do"`
}

type Cancel struct {
	Handle Value `json:"handle"`
	Reason Value `json:"reason,omitempty"`
}

type Return struct {
	To    string `json:"to,omitempty"`
	Value Value  `json:"value"`
}

type Fail struct {
	Code      string `json:"code"`
	Category  string `json:"category,omitempty"`
	Message   Value  `json:"message"`
	Retryable *bool  `json:"retryable,omitempty"`
	Details   Value  `json:"details,omitempty"`
}

type ErrorMatch struct {
	Categories   []string `json:"categories,omitempty"`
	Codes        []string `json:"codes,omitempty"`
	Retryable    *bool    `json:"retryable,omitempty"`
	EffectStates []string `json:"effectStates,omitempty"`
	Schema       Schema   `json:"schema,omitempty"`
}

type Handler struct {
	Match ErrorMatch `json:"match"`
	Do    Block      `json:"do"`
}

type Retry struct {
	MaxAttempts       int         `json:"maxAttempts"`
	Match             *ErrorMatch `json:"match,omitempty"`
	Backoff           *Backoff    `json:"backoff,omitempty"`
	PerAttemptTimeout string      `json:"perAttemptTimeout,omitempty"`
}

type Backoff struct {
	Initial    string   `json:"initial"`
	Maximum    string   `json:"maximum,omitempty"`
	Multiplier *float64 `json:"multiplier,omitempty"`
	Jitter     *float64 `json:"jitter,omitempty"`
}

type Effect struct {
	Kind       string `json:"kind"`
	Key        Value  `json:"key,omitempty"`
	Reconcile  *Block `json:"reconcile,omitempty"`
	Compensate *Block `json:"compensate,omitempty"`
}

type EventContract struct {
	Schema            Schema `json:"schema"`
	Source            string `json:"source"`
	CorrelationSchema Schema `json:"correlationSchema,omitempty"`
}

type Subscription struct {
	Event       string `json:"event"`
	Correlation Value  `json:"correlation,omitempty"`
	Filter      Value  `json:"filter,omitempty"`
	Retention   string `json:"retention,omitempty"`
	Capacity    *int   `json:"capacity,omitempty"`
	MaxBytes    *int   `json:"maxBytes,omitempty"`
}

type Channel struct {
	Schema      Schema   `json:"schema"`
	MaxMessages *int     `json:"maxMessages,omitempty"`
	MaxBytes    *int     `json:"maxBytes,omitempty"`
	Readers     []string `json:"readers,omitempty"`
	Writers     []string `json:"writers,omitempty"`
}

type Trigger struct {
	Event    *EventTrigger `json:"event,omitempty"`
	Schedule *Schedule     `json:"schedule,omitempty"`
}

type EventTrigger struct {
	Event       string `json:"event"`
	Filter      Value  `json:"filter,omitempty"`
	Input       Value  `json:"input"`
	InstanceKey Value  `json:"instanceKey,omitempty"`
	Behavior    string `json:"behavior,omitempty"`
}

type Schedule struct {
	Calendar string            `json:"calendar,omitempty"`
	Interval string            `json:"interval,omitempty"`
	TimeZone string            `json:"timeZone,omitempty"`
	Input    Value             `json:"input"`
	Overlap  string            `json:"overlap,omitempty"`
	Missed   *MissedOccurrence `json:"missed,omitempty"`
}

type MissedOccurrence struct {
	Mode           string `json:"mode"`
	MaxOccurrences *int   `json:"maxOccurrences,omitempty"`
}

type Resource struct {
	Concurrency *int  `json:"concurrency,omitempty"`
	Rate        *Rate `json:"rate,omitempty"`
	Keyed       bool  `json:"keyed,omitempty"`
}

type Rate struct {
	Units int    `json:"units"`
	Per   string `json:"per"`
	Burst *int   `json:"burst,omitempty"`
}

type Claim struct {
	Resource string `json:"resource"`
	Units    *int   `json:"units,omitempty"`
	Key      Value  `json:"key,omitempty"`
}

type Cost struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type Limits struct {
	Duration        string `json:"duration,omitempty"`
	StepExecutions  *int   `json:"stepExecutions,omitempty"`
	CallAttempts    *int   `json:"callAttempts,omitempty"`
	WorkflowDepth   *int   `json:"workflowDepth,omitempty"`
	Concurrency     *int   `json:"concurrency,omitempty"`
	ModelTokens     *int   `json:"modelTokens,omitempty"`
	Cost            *Cost  `json:"cost,omitempty"`
	MessageCount    *int   `json:"messageCount,omitempty"`
	MessageBytes    *int   `json:"messageBytes,omitempty"`
	EventCount      *int   `json:"eventCount,omitempty"`
	EventBytes      *int   `json:"eventBytes,omitempty"`
	CleanupDuration string `json:"cleanupDuration,omitempty"`
}
