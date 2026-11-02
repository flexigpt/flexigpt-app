package sessionv3

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrRuntimeClosed     = errors.New("session runtime is closed")
	ErrSessionNotFound   = errors.New("session not found")
	ErrExecutionNotFound = errors.New("execution not found")
	ErrCommandConflict   = errors.New("command ID was reused with another payload")
)

func validateID(subject, value string) error {
	if strings.TrimSpace(value) == "" || strings.TrimSpace(value) != value ||
		!utf8.ValidString(value) || strings.ContainsRune(value, 0) || len(value) > 16<<10 {
		return fmt.Errorf("invalid %s", subject)
	}
	return nil
}

func (s Session) Validate() error {
	if err := validateID("session ID", string(s.ID)); err != nil {
		return err
	}
	if s.Revision == 0 || s.CreatedAt.IsZero() || s.UpdatedAt.IsZero() {
		return errors.New("session revision and timestamps are required")
	}
	if !s.Metadata.Valid() || !s.Defaults.Metadata.Valid() {
		return errors.New("session metadata is not canonical JSON")
	}
	if err := validateDefaults(s.Defaults); err != nil {
		return err
	}

	activityIDs := make(map[ActivityID]struct{}, len(s.Timeline))
	for index, activity := range s.Timeline {
		if err := activity.Validate(); err != nil {
			return fmt.Errorf("timeline activity %d: %w", index, err)
		}
		if _, duplicate := activityIDs[activity.ID]; duplicate {
			return fmt.Errorf("duplicate activity %q", activity.ID)
		}
		activityIDs[activity.ID] = struct{}{}
	}

	executionIDs := make(map[ExecutionID]struct{}, len(s.Executions))
	for index, execution := range s.Executions {
		if err := execution.Validate(s.ID); err != nil {
			return fmt.Errorf("execution %d: %w", index, err)
		}
		if _, duplicate := executionIDs[execution.ID]; duplicate {
			return fmt.Errorf("duplicate execution %q", execution.ID)
		}
		executionIDs[execution.ID] = struct{}{}
	}

	commands := make(map[CommandID]struct{}, len(s.Commands))
	for _, command := range s.Commands {
		if err := validateID("command ID", string(command.ID)); err != nil {
			return err
		}
		if command.Fingerprint == "" || command.RecordedAt.IsZero() {
			return errors.New("invalid durable command record")
		}
		if _, duplicate := commands[command.ID]; duplicate {
			return fmt.Errorf("duplicate command record %q", command.ID)
		}
		commands[command.ID] = struct{}{}
	}
	return nil
}

func validateDefaults(value SessionDefaults) error {
	for _, selection := range []*ArtifactSelection{
		value.Model, value.Agent, value.Workspace,
	} {
		if selection == nil {
			continue
		}
		if err := validateSelection(*selection); err != nil {
			return err
		}
	}
	for _, instruction := range value.SystemInstructions {
		if err := validateID("instruction ID", instruction.ID); err != nil {
			return err
		}
		if instruction.Content == "" {
			return errors.New("instruction content is required")
		}
	}
	return nil
}

func validateSelection(value ArtifactSelection) error {
	if err := validateID("Artifact Root ID", value.RootID); err != nil {
		return err
	}
	if err := validateID("Artifact ID", value.ArtifactID); err != nil {
		return err
	}
	return nil
}

func validatePin(value ArtifactPin) error {
	if err := validateID("Artifact Root ID", value.RootID); err != nil {
		return err
	}
	if err := validateID("Artifact ID", value.ArtifactID); err != nil {
		return err
	}
	if value.Revision == 0 || strings.TrimSpace(value.Digest) == "" {
		return errors.New("artifact pin revision and digest are required")
	}
	return nil
}

func (a Activity) Validate() error {
	if err := validateID("activity ID", string(a.ID)); err != nil {
		return err
	}
	if a.CreatedAt.IsZero() {
		return errors.New("activity timestamp is required")
	}
	switch a.Visibility {
	case ActivityVisibilityUser, ActivityVisibilityAudit, ActivityVisibilityBoth:
	default:
		return fmt.Errorf("unsupported activity visibility %q", a.Visibility)
	}

	present := 0
	if a.Message != nil {
		present++
	}
	if a.Action != nil {
		present++
	}
	if a.Workflow != nil {
		present++
	}
	if a.Wait != nil {
		present++
	}
	if present != 1 {
		return errors.New("activity must have exactly one typed payload")
	}

	switch a.Kind {
	case ActivityKindMessage:
		if a.Message == nil || !a.Message.Metadata.Valid() {
			return errors.New("invalid message activity")
		}
		for _, attachment := range a.Message.Attachments {
			if err := attachment.Validate(); err != nil {
				return err
			}
		}
	case ActivityKindAction:
		if a.Action == nil || !a.Action.Output.Valid() {
			return errors.New("invalid action activity")
		}
	case ActivityKindWorkflow:
		if a.Workflow == nil || !a.Workflow.Output.Valid() {
			return errors.New("invalid workflow activity")
		}
	case ActivityKindWait:
		if a.Wait == nil || a.Wait.WaitID == "" {
			return errors.New("invalid wait activity")
		}
	default:
		return fmt.Errorf("unsupported activity kind %q", a.Kind)
	}
	return nil
}

func (a Attachment) Validate() error {
	switch a.Kind {
	case AttachmentKindText:
		if a.Text == "" {
			return errors.New("text attachment has no content")
		}
	case AttachmentKindImage, AttachmentKindFile:
		if a.Base64Data == "" && a.URL == "" {
			return errors.New("binary attachment has no content")
		}
	case AttachmentKindURL:
		if a.URL == "" {
			return errors.New("URL attachment has no URL")
		}
	default:
		return fmt.Errorf("unsupported attachment kind %q", a.Kind)
	}
	if strings.ContainsRune(a.Label, 0) || strings.ContainsRune(a.URL, 0) {
		return errors.New("attachment contains NUL")
	}
	return nil
}

func (e Execution) Validate(sessionID SessionID) error {
	if err := validateID("execution ID", string(e.ID)); err != nil {
		return err
	}
	if e.SessionID != sessionID || e.Revision == 0 || e.CreatedAt.IsZero() || e.UpdatedAt.IsZero() {
		return errors.New("invalid execution identity, revision, or timestamps")
	}
	switch e.State {
	case ExecutionStateQueued, ExecutionStateRunning, ExecutionStateWaiting, ExecutionStatePaused,
		ExecutionStateCancelling, ExecutionStateCompleted, ExecutionStateFailed, ExecutionStateCancelled:
	default:
		return fmt.Errorf("unsupported execution state %q", e.State)
	}

	frames := make(map[ProgramFrameID]struct{}, len(e.ProgramFrames))
	for _, frame := range e.ProgramFrames {
		if err := frame.Validate(); err != nil {
			return err
		}
		if _, duplicate := frames[frame.ID]; duplicate {
			return fmt.Errorf("duplicate frame %q", frame.ID)
		}
		frames[frame.ID] = struct{}{}
	}
	items := make(map[WorkItemID]struct{}, len(e.WorkItems))
	for _, item := range e.WorkItems {
		if _, found := frames[item.FrameID]; !found {
			return fmt.Errorf("work item %q has no frame", item.ID)
		}
		if err := item.Validate(); err != nil {
			return err
		}
		if _, duplicate := items[item.ID]; duplicate {
			return fmt.Errorf("duplicate work item %q", item.ID)
		}
		items[item.ID] = struct{}{}
	}
	attempts := make(map[AttemptID]struct{}, len(e.Attempts))
	for _, attempt := range e.Attempts {
		if _, found := items[attempt.WorkItemID]; !found {
			return fmt.Errorf("attempt %q has no work item", attempt.ID)
		}
		if err := attempt.Validate(); err != nil {
			return err
		}
		if _, duplicate := attempts[attempt.ID]; duplicate {
			return fmt.Errorf("duplicate attempt %q", attempt.ID)
		}
		attempts[attempt.ID] = struct{}{}
	}
	waits := make(map[WaitID]struct{}, len(e.Waits))
	for _, wait := range e.Waits {
		if _, found := items[wait.WorkItemID]; !found && wait.WorkItemID != "recovery" {
			return fmt.Errorf("wait %q has no work item", wait.ID)
		}
		if err := wait.Validate(); err != nil {
			return err
		}
		if _, duplicate := waits[wait.ID]; duplicate {
			return fmt.Errorf("duplicate wait %q", wait.ID)
		}
		waits[wait.ID] = struct{}{}
	}
	return nil
}

func (p ProgramFrame) Validate() error {
	if err := validateID("ProgramFrame ID", string(p.ID)); err != nil {
		return err
	}
	if p.Program.InterpreterID == "" || p.Program.InterpreterVersion == 0 {
		return errors.New("ProgramFrame interpreter is required")
	}
	if !p.Program.Payload.Valid() || !p.State.Data.Valid() || p.State.Type == "" || p.State.Version == 0 {
		return errors.New("ProgramFrame payload or state is invalid")
	}
	if p.Program.Artifact != nil {
		if err := validatePin(*p.Program.Artifact); err != nil {
			return err
		}
	}
	if p.CreatedAt.IsZero() || p.UpdatedAt.IsZero() {
		return errors.New("ProgramFrame timestamps are required")
	}
	return nil
}

func (w WorkItem) Validate() error {
	if err := validateID("WorkItem ID", string(w.ID)); err != nil {
		return err
	}
	if err := validateID("WorkItem frame ID", string(w.FrameID)); err != nil {
		return err
	}
	if w.CreatedAt.IsZero() || w.UpdatedAt.IsZero() {
		return errors.New("WorkItem timestamps are required")
	}
	if w.Deadline != nil && w.Deadline.IsZero() {
		return errors.New("invalid WorkItem deadline")
	}
	switch w.Effect.Kind {
	case EffectKindRead, EffectKindIdempotent, EffectKindKeyed, EffectKindNonIdempotent:
	default:
		return fmt.Errorf("unsupported effect kind %q", w.Effect.Kind)
	}
	if w.Effect.Kind == EffectKindKeyed && w.Effect.IdempotencyKey == "" {
		return errors.New("keyed WorkItem has no idempotency key")
	}
	if err := w.Input.Validate(w.Type); err != nil {
		return err
	}
	if w.Binding != nil {
		if err := w.Binding.Validate(); err != nil {
			return err
		}
	}
	if w.Output != nil {
		if err := w.Output.Validate(w.Type); err != nil {
			return err
		}
	}
	return nil
}

func (p WorkItemPayload) Validate(kind WorkItemType) error {
	values := []struct {
		kind    WorkItemType
		present bool
	}{
		{WorkItemTypeLLM, p.LLM != nil},
		{WorkItemTypeTool, p.Tool != nil},
		{WorkItemTypeService, p.Service != nil},
		{WorkItemTypeActivity, p.Activity != nil},
		{WorkItemTypeTransform, p.Transform != nil},
		{WorkItemTypeCondition, p.Condition != nil},
		{WorkItemTypeStateUpdate, p.StateUpdate != nil},
		{WorkItemTypeFork, p.Fork != nil},
		{WorkItemTypeJoin, p.Join != nil},
		{WorkItemTypeWait, p.Wait != nil},
		{WorkItemTypeEmit, p.Emit != nil},
		{WorkItemTypeCompensation, p.Compensation != nil},
	}
	count := 0
	for _, value := range values {
		if value.present {
			count++
			if value.kind != kind {
				return fmt.Errorf("WorkItem type %q has payload %q", kind, value.kind)
			}
		}
	}
	if count != 1 {
		return errors.New("WorkItem must have exactly one typed payload")
	}
	switch kind {
	case WorkItemTypeLLM:
		if p.LLM.InputActivityID == "" {
			return errors.New("LLM WorkItem input activity is required")
		}
	case WorkItemTypeTool:
		return p.Tool.Call.Validate()
	case WorkItemTypeService:
		if p.Service.ServiceID == "" || p.Service.BindingVersion == "" || !p.Service.Input.Valid() {
			return errors.New("invalid service WorkItem")
		}
	case WorkItemTypeActivity:
		if p.Activity.ActivityID == "" || p.Activity.BindingVersion == "" || !p.Activity.Input.Valid() {
			return errors.New("invalid activity WorkItem")
		}
	case WorkItemTypeTransform:
		if !p.Transform.Input.Valid() {
			return errors.New("invalid transform WorkItem")
		}
	case WorkItemTypeCondition:
		if !p.Condition.Input.Valid() {
			return errors.New("invalid condition WorkItem")
		}
	case WorkItemTypeStateUpdate:
		for _, assignment := range p.StateUpdate.Assignments {
			if assignment.Path == "" || !assignment.Value.Valid() {
				return errors.New("invalid state assignment")
			}
		}
	case WorkItemTypeFork:
		if len(p.Fork.ChildFrames) == 0 {
			return errors.New("fork WorkItem has no child frames")
		}
	case WorkItemTypeJoin:
		if len(p.Join.ChildFrames) == 0 || p.Join.Policy == "" {
			return errors.New("invalid join WorkItem")
		}
	case WorkItemTypeWait:
		return p.Wait.Request.Validate()
	case WorkItemTypeEmit:
		if !p.Emit.Data.Valid() || (p.Emit.Destination.Channel == "" && p.Emit.Destination.Event == "") {
			return errors.New("invalid emit WorkItem")
		}
	case WorkItemTypeCompensation:
		if p.Compensation.OriginalWorkItemID == "" || !p.Compensation.Input.Valid() {
			return errors.New("invalid compensation WorkItem")
		}
	}
	return nil
}

func (c ToolCall) Validate() error {
	if c.ID == "" || c.Name == "" || !c.Arguments.Valid() {
		return errors.New("invalid Tool call")
	}
	return c.Binding.Validate()
}

func (b ToolBinding) Validate() error {
	count := 0
	if b.Go != nil {
		count++
	}
	if b.MCP != nil {
		count++
	}
	if b.Skill != nil {
		count++
	}
	if b.Provider != nil {
		count++
	}
	if b.Unresolved != nil {
		count++
	}
	if count != 1 {
		return errors.New("tool binding must have exactly one typed binding")
	}
	switch b.Kind {
	case ToolBindingKindGo:
		if b.Go == nil {
			return errors.New("go Tool binding is missing")
		}
		return validatePin(b.Go.Target)
	case ToolBindingKindMCP:
		if b.MCP == nil || b.MCP.ServerID == "" || b.MCP.ProviderToolName == "" || b.MCP.ToolName == "" ||
			b.MCP.ToolDigest == "" {
			return errors.New("invalid MCP Tool binding")
		}
		if b.MCP.ServerArtifact != nil {
			return validatePin(*b.MCP.ServerArtifact)
		}
	case ToolBindingKindSkill:
		if b.Skill == nil || b.Skill.Name == "" {
			return errors.New("invalid Skill Tool binding")
		}
	case ToolBindingKindProvider:
		if b.Provider == nil || b.Provider.Name == "" {
			return errors.New("invalid provider Tool binding")
		}
	case ToolBindingKindUnresolved:
		if b.Unresolved == nil || b.Unresolved.Reason == "" {
			return errors.New("invalid unresolved Tool binding")
		}
	default:
		return fmt.Errorf("unsupported Tool binding kind %q", b.Kind)
	}
	return nil
}

func (b SemanticBinding) Validate() error {
	if b.Version == "" || b.TargetFingerprint == "" || b.BoundAt.IsZero() {
		return errors.New("semantic binding is incomplete")
	}
	for _, artifact := range b.Artifacts {
		if err := validatePin(artifact); err != nil {
			return err
		}
	}
	return nil
}

func (o WorkItemOutput) Validate(kind WorkItemType) error {
	if !o.Value.Valid() {
		return errors.New("WorkItem output value is not canonical")
	}
	if kind == WorkItemTypeLLM && o.LLM == nil {
		return errors.New("LLM WorkItem output is missing")
	}
	if kind == WorkItemTypeTool && o.Tool == nil {
		return errors.New("tool WorkItem output is missing")
	}
	if o.LLM != nil {
		for _, call := range o.LLM.ToolCalls {
			if err := call.Validate(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (a Attempt) Validate() error {
	if a.ID == "" || a.WorkItemID == "" || a.Number < 1 || a.StartedAt.IsZero() {
		return errors.New("invalid attempt")
	}
	if a.FinishedAt != nil && a.FinishedAt.Before(a.StartedAt) {
		return errors.New("attempt finished before it started")
	}
	return nil
}

func (w WaitRequest) Validate() error {
	if !w.Summary.Valid() || !w.InputSchema.Valid() || !w.Correlation.Valid() {
		return errors.New("wait request JSON is not canonical")
	}
	if len(w.AllowedResolutions) == 0 {
		return errors.New("wait request has no allowed resolution")
	}
	if w.Deadline != nil && w.Deadline.IsZero() {
		return errors.New("invalid wait deadline")
	}
	return nil
}

func (w Wait) Validate() error {
	if w.ID == "" || w.WorkItemID == "" || w.CreatedAt.IsZero() || w.UpdatedAt.IsZero() {
		return errors.New("invalid wait")
	}
	return w.Request.Validate()
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}
