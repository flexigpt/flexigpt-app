package sessionv3

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrRuntimeClosed     = errors.New("session runtime is closed")
	ErrSessionNotFound   = errors.New("session not found")
	ErrExecutionNotFound = errors.New("execution not found")
	ErrCommandConflict   = errors.New("command ID was reused with another payload")
	ErrCommandInProgress = errors.New("command ID is already being processed")
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

	if err := validateTriggerRegistrations(s, executionIDs); err != nil {
		return err
	}

	for _, execution := range s.Executions {
		if execution.RootExecutionID != "" {
			if _, found := executionIDs[execution.RootExecutionID]; !found {
				return fmt.Errorf("execution %q has an unavailable root execution", execution.ID)
			}
		}
		if execution.ParentExecutionID != "" {
			if _, found := executionIDs[execution.ParentExecutionID]; !found {
				return fmt.Errorf("execution %q has an unavailable parent execution", execution.ID)
			}
		}
		for _, childID := range execution.ChildExecutionIDs {
			if _, found := executionIDs[childID]; !found {
				return fmt.Errorf("execution %q has an unavailable child execution", execution.ID)
			}
		}
	}

	for _, ownership := range s.ConversationOwnership {
		if err := validateID("conversation ID", ownership.ConversationID); err != nil {
			return err
		}
		if ownership.Owner.ExecutionID != "" {
			if _, found := executionIDs[ownership.Owner.ExecutionID]; !found {
				return fmt.Errorf(
					"conversation %q has an unavailable owner execution",
					ownership.ConversationID,
				)
			}
		}
		if ownership.Owner.Agent != nil {
			if err := validatePin(*ownership.Owner.Agent); err != nil {
				return err
			}
		}
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

func validateArtifactSelection(value ArtifactSelection) error {
	if err := validateID("Artifact Root ID", value.RootID); err != nil {
		return err
	}
	if err := validateID("Artifact ID", value.ArtifactID); err != nil {
		return err
	}
	if value.ExpectedDigest != "" &&
		(strings.TrimSpace(value.ExpectedDigest) != value.ExpectedDigest ||
			!utf8.ValidString(value.ExpectedDigest) ||
			strings.ContainsRune(value.ExpectedDigest, 0)) {
		return errors.New("artifact expected digest is invalid")
	}
	return nil
}

func validateOptionalArtifactSelection(value *ArtifactSelection) error {
	if value == nil {
		return nil
	}
	return validateArtifactSelection(*value)
}

func validateDefaults(value SessionDefaults) error {
	for _, selection := range []*ArtifactSelection{
		value.Model,
		value.Agent,
		value.Workspace,
	} {
		if err := validateOptionalArtifactSelection(selection); err != nil {
			return err
		}
	}

	if value.History.IncludePreviousUserTurns < 0 {
		return errors.New("history previous-user-turn count cannot be negative")
	}
	if value.Temperature != nil &&
		(math.IsNaN(*value.Temperature) || math.IsInf(*value.Temperature, 0)) {
		return errors.New("session temperature must be finite")
	}

	instructions := make(map[string]struct{}, len(value.SystemInstructions))
	for index, instruction := range value.SystemInstructions {
		if strings.TrimSpace(instruction.Content) == "" {
			return fmt.Errorf("system instruction %d has no content", index)
		}
		if instruction.ID == "" {
			continue
		}
		if err := validateID("instruction ID", instruction.ID); err != nil {
			return err
		}
		if _, duplicate := instructions[instruction.ID]; duplicate {
			return fmt.Errorf("duplicate system instruction %q", instruction.ID)
		}
		instructions[instruction.ID] = struct{}{}
	}

	validateSelections := func(
		label string,
		selections []ArtifactSelection,
	) error {
		seen := make(map[string]struct{}, len(selections))
		for index := range selections {
			selection := selections[index]
			if err := validateSelection(selection); err != nil {
				return fmt.Errorf("%s %d: %w", label, index, err)
			}
			key := selection.RootID + "\x00" + selection.ArtifactID
			if _, duplicate := seen[key]; duplicate {
				return fmt.Errorf("duplicate %s selection %q", label, key)
			}
			seen[key] = struct{}{}
		}
		return nil
	}

	if err := validateSelections("Tool", value.Tools); err != nil {
		return err
	}
	if err := validateSelections("Skill", value.Skills); err != nil {
		return err
	}
	if err := validateSelections("active Skill", value.ActiveSkills); err != nil {
		return err
	}

	enabledSkills := make(
		map[string]struct{},
		len(value.Skills),
	)
	for _, selection := range value.Skills {
		enabledSkills[selection.RootID+"\x00"+selection.ArtifactID] = struct{}{}
	}
	for _, selection := range value.ActiveSkills {
		key := selection.RootID + "\x00" + selection.ArtifactID
		if _, found := enabledSkills[key]; !found {
			return fmt.Errorf(
				"active Skill %q is not part of enabled Skills",
				key,
			)
		}
	}

	servers := make(map[string]struct{}, len(value.MCP))
	for index, selection := range value.MCP {
		if err := validateSelection(selection.Server); err != nil {
			return fmt.Errorf("MCP selection %d: %w", index, err)
		}
		serverKey := selection.Server.RootID + "\x00" +
			selection.Server.ArtifactID
		if _, duplicate := servers[serverKey]; duplicate {
			return fmt.Errorf(
				"duplicate MCP server selection %q",
				serverKey,
			)
		}
		servers[serverKey] = struct{}{}

		switch selection.ToolExposure {
		case MCPToolExposureNone:
			if len(selection.SelectedTools) != 0 {
				return fmt.Errorf(
					"MCP selection %d disables tools but contains selected tools",
					index,
				)
			}

		case "", MCPToolExposureAll:
			if len(selection.SelectedTools) != 0 {
				return fmt.Errorf(
					"MCP selection %d exposes all tools but contains selected tools",
					index,
				)
			}

		case MCPToolExposureSelected:
			if len(selection.SelectedTools) == 0 {
				return fmt.Errorf(
					"MCP selection %d selects tools but contains none",
					index,
				)
			}
			tools := make(
				map[string]struct{},
				len(selection.SelectedTools),
			)
			for toolIndex, tool := range selection.SelectedTools {
				if err := validateMCPToolSelection(tool); err != nil {
					return fmt.Errorf(
						"MCP selection %d tool %d: %w",
						index,
						toolIndex,
						err,
					)
				}
				if _, duplicate := tools[tool.ToolName]; duplicate {
					return fmt.Errorf(
						"duplicate selected MCP Tool %q",
						tool.ToolName,
					)
				}
				tools[tool.ToolName] = struct{}{}
			}

		default:
			return fmt.Errorf(
				"unsupported MCP tool exposure %q",
				selection.ToolExposure,
			)
		}
	}
	return nil
}

func validateMCPToolSelection(value MCPToolSelection) error {
	if err := validateID("MCP Tool name", value.ToolName); err != nil {
		return err
	}
	if value.ExpectedDigest != "" {
		if err := validateID("MCP Tool expected digest", value.ExpectedDigest); err != nil {
			return err
		}
	}

	if value.ApprovalRule != nil {
		switch *value.ApprovalRule {
		case "allow", "ask", "deny":
		default:
			return errors.New("invalid MCP Tool approval override")
		}
	}
	if value.ExecutionMode != nil {
		switch *value.ExecutionMode {
		case "auto", "manual":
		default:
			return errors.New("invalid MCP Tool execution override")
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
	if value.ExpectedDigest != "" &&
		(strings.TrimSpace(value.ExpectedDigest) != value.ExpectedDigest ||
			!utf8.ValidString(value.ExpectedDigest) ||
			strings.ContainsRune(value.ExpectedDigest, 0)) {
		return errors.New("artifact expected digest is invalid")
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

	switch e.Lifetime {
	case "", ExecutionChildLifetimeRoot, ExecutionChildLifetimeIndependent:
	default:
		return fmt.Errorf("unsupported execution lifetime %q", e.Lifetime)
	}
	if e.RootExecutionID != "" {
		if err := validateID("execution root ID", string(e.RootExecutionID)); err != nil {
			return err
		}
	}
	if e.ParentExecutionID != "" {
		if err := validateID("execution parent ID", string(e.ParentExecutionID)); err != nil {
			return err
		}
	}
	if e.ParentFrameID != "" {
		if err := validateID("execution parent frame ID", string(e.ParentFrameID)); err != nil {
			return err
		}
	}
	if err := e.ChildBudget.Validate(); err != nil {
		return err
	}
	if e.Trigger != nil {
		if err := e.Trigger.Validate(); err != nil {
			return err
		}
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

	signals := make(map[string]struct{}, len(e.Signals))
	for _, signal := range e.Signals {
		if err := signal.Validate(); err != nil {
			return err
		}
		if _, duplicate := signals[signal.ID]; duplicate {
			return fmt.Errorf("duplicate execution signal %q", signal.ID)
		}
		signals[signal.ID] = struct{}{}
	}

	channels := make(map[string]struct{}, len(e.Channels))
	for _, channel := range e.Channels {
		if err := channel.Validate(); err != nil {
			return err
		}
		if _, duplicate := channels[channel.Name]; duplicate {
			return fmt.Errorf("duplicate channel %q", channel.Name)
		}
		channels[channel.Name] = struct{}{}
	}
	for _, receipt := range e.EmitReceipts {
		if err := receipt.Validate(); err != nil {
			return err
		}
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
	for _, executionID := range p.ChildExecutionIDs {
		if err := validateID("ProgramFrame child Execution ID", string(executionID)); err != nil {
			return err
		}
	}
	if p.Fork != nil {
		if err := p.Fork.Validate(); err != nil {
			return err
		}
	}
	seenJoins := make(map[WorkItemID]struct{}, len(p.Joins))
	for _, join := range p.Joins {
		if err := join.Validate(); err != nil {
			return err
		}
		if _, duplicate := seenJoins[join.WorkItemID]; duplicate {
			return errors.New("ProgramFrame join WorkItem is duplicated")
		}
		seenJoins[join.WorkItemID] = struct{}{}
	}
	return nil
}

func (b ForkBoundary) Validate() error {
	if err := validateID("fork WorkItem ID", string(b.WorkItemID)); err != nil {
		return err
	}
	if len(b.Frames) == 0 {
		return errors.New("fork boundary has no child frames")
	}
	seenFrames := make(map[ProgramFrameID]struct{}, len(b.Frames))
	for _, frame := range b.Frames {
		if err := validateDeferredFrameIntent(frame); err != nil {
			return err
		}
		if _, duplicate := seenFrames[frame.ID]; duplicate {
			return errors.New("fork boundary child frame is duplicated")
		}
		seenFrames[frame.ID] = struct{}{}
	}
	return nil
}

func (b JoinBoundary) Validate() error {
	if err := validateID("join WorkItem ID", string(b.WorkItemID)); err != nil {
		return err
	}
	return validateID("join child ProgramFrame ID", string(b.ChildFrameID))
}

func validateDeferredFrameIntent(value ProgramFrameIntent) error {
	if err := validateID("deferred ProgramFrame ID", string(value.ID)); err != nil {
		return err
	}
	if value.ExecutionID != "" {
		if err := validateID("deferred Execution ID", string(value.ExecutionID)); err != nil {
			return err
		}
	}
	if value.Program.InterpreterID == "" ||
		value.Program.InterpreterVersion == 0 ||
		!value.Program.Payload.Valid() {
		return errors.New("deferred ProgramFrame program is invalid")
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
	if err := w.Effect.Validate(); err != nil {
		return err
	}
	if w.Effect.Kind == EffectKindKeyed &&
		strings.TrimSpace(w.Effect.IdempotencyKey) == "" {
		return errors.New("keyed work item has no idempotency key")
	}
	if err := validateEffectStateForWork(w); err != nil {
		return err
	}
	switch w.State {
	case WorkItemStateReady,
		WorkItemStateBinding,
		WorkItemStateDispatched,
		WorkItemStateRetryWaiting,
		WorkItemStateWaiting,
		WorkItemStateCompleted,
		WorkItemStateFailed,
		WorkItemStateUncertain,
		WorkItemStateCancelled,
		WorkItemStateBlocked:
	default:
		return fmt.Errorf("unsupported WorkItem state %q", w.State)
	}
	if w.Retry != nil {
		if err := w.Retry.Validate(); err != nil {
			return err
		}
	}
	if w.RetryState != nil && w.State != WorkItemStateRetryWaiting {
		return errors.New("non-retrying WorkItem has retry state")
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

func (e EffectContract) Validate() error {
	switch e.Kind {
	case EffectKindRead,
		EffectKindIdempotent,
		EffectKindKeyed,
		EffectKindNonIdempotent:
	default:
		return fmt.Errorf("unsupported effect kind %q", e.Kind)
	}
	if e.Kind == EffectKindKeyed &&
		strings.TrimSpace(e.IdempotencyKey) == "" {
		return errors.New("keyed work item has no idempotency key")
	}
	if e.Kind != EffectKindKeyed && e.IdempotencyKey != "" {
		return errors.New("non-keyed work item has an idempotency key")
	}
	return nil
}

func (e WorkflowStepEffect) Validate() error {
	switch e.Kind {
	case EffectKindRead, EffectKindIdempotent, EffectKindNonIdempotent:
		if e.Key != nil {
			return errors.New("non-keyed workflow effect contains a key")
		}
	case EffectKindKeyed:
		if e.Key == nil {
			return errors.New("keyed workflow effect has no key")
		}
		if err := e.Key.Validate(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported workflow effect kind %q", e.Kind)
	}
	return nil
}

func (b ExecutionBudget) Validate() error {
	if b.Duration != "" {
		if err := validateWorkflowTimeout(b.Duration); err != nil {
			return fmt.Errorf("execution budget duration: %w", err)
		}
	}
	if b.CleanupDuration != "" {
		if err := validateWorkflowTimeout(b.CleanupDuration); err != nil {
			return fmt.Errorf("execution cleanup duration: %w", err)
		}
	}
	for _, value := range []int{
		b.StepExecutions,
		b.CallAttempts,
		b.WorkflowDepth,
		b.Concurrency,
		b.ModelTokens,
		b.MessageCount,
		b.MessageBytes,
		b.EventCount,
		b.EventBytes,
	} {
		if value < 0 {
			return errors.New("execution budget cannot be negative")
		}
	}
	if b.Cost != nil && (b.Cost.Amount < 0 || strings.TrimSpace(b.Cost.Currency) == "") {
		return errors.New("execution budget cost is invalid")
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
		{WorkItemTypeTargetResolver, p.TargetResolver != nil},
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
		if p.LLM.InputActivityID == "" && p.LLM.Invocation == nil {
			return errors.New(
				"llm WorkItem requires an input activity or workflow invocation plan",
			)
		}
		if p.LLM.Invocation != nil {
			if err := p.LLM.Invocation.Validate(); err != nil {
				return err
			}
		}

	case WorkItemTypeTool:
		if err := p.Tool.Call.Validate(); err != nil {
			return err
		}
		if p.Tool.Target != nil && p.Tool.MCP != nil {
			return errors.New(
				"tool workItem cannot contain both Artifact and MCP targets",
			)
		}
		if p.Tool.Target != nil {
			if err := validatePin(*p.Tool.Target); err != nil {
				return err
			}
		}
		if p.Tool.MCP != nil {
			if err := validatePin(p.Tool.MCP.Server); err != nil {
				return err
			}
			if err := validateID(
				"MCP Tool target name",
				p.Tool.MCP.ToolName,
			); err != nil {
				return err
			}
		}

	case WorkItemTypeService:
		if p.Service.ServiceID == "" || p.Service.BindingVersion == "" || !p.Service.Input.Valid() {
			return errors.New("invalid service WorkItem")
		}
	case WorkItemTypeActivity:
		if p.Activity.ActivityID == "" || p.Activity.BindingVersion == "" || !p.Activity.Input.Valid() {
			return errors.New("invalid activity WorkItem")
		}
	case WorkItemTypeTransform:
		if !p.Transform.Input.Valid() ||
			!p.Transform.OutputSchema.Valid() {
			return errors.New("invalid transform WorkItem")
		}
	case WorkItemTypeCondition:
		if !p.Condition.Input.Valid() {
			return errors.New("invalid condition WorkItem")
		}
	case WorkItemTypeStateUpdate:
		if !p.StateUpdate.Base.Valid() ||
			!p.StateUpdate.Schema.Valid() {
			return errors.New("invalid StateUpdate WorkItem state boundary")
		}
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
		for _, childID := range p.Join.ChildFrames {
			if err := validateID(
				"join WorkItem child ProgramFrame ID",
				string(childID),
			); err != nil {
				return err
			}
		}
	case WorkItemTypeTargetResolver:
		if err := p.TargetResolver.Validate(); err != nil {
			return err
		}
	case WorkItemTypeWait:
		return p.Wait.Request.Validate()
	case WorkItemTypeEmit:
		if !p.Emit.Data.Valid() ||
			!p.Emit.Correlation.Valid() ||
			!p.Emit.DataSchema.Valid() ||
			!p.Emit.CorrelationSchema.Valid() {
			return errors.New("invalid emit WorkItem")
		}
		if p.Emit.MaxMessages < 0 || p.Emit.MaxBytes < 0 {
			return errors.New("emit WorkItem limits cannot be negative")
		}
		switch {
		case p.Emit.Destination.Channel == "" &&
			p.Emit.Destination.Event == "":
			return errors.New("emit WorkItem has no destination")
		case p.Emit.Destination.Channel != "" &&
			p.Emit.Destination.Event != "":
			return errors.New("emit WorkItem has more than one destination")
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
		if b.Go == nil ||
			strings.TrimSpace(b.Go.Function) == "" ||
			b.Go.TimeoutMS < 0 {
			return errors.New("go Tool binding is missing")
		}
		return validatePin(b.Go.Target)

	case ToolBindingKindMCP:
		if b.MCP == nil ||
			strings.TrimSpace(b.MCP.ServerID) == "" ||
			strings.TrimSpace(b.MCP.ProviderToolName) == "" ||
			strings.TrimSpace(b.MCP.ChoiceID) == "" ||
			strings.TrimSpace(b.MCP.ToolName) == "" ||
			strings.TrimSpace(b.MCP.ToolDigest) == "" {
			return errors.New("invalid MCP Tool binding")
		}
		if b.MCP.ServerArtifact != nil {
			if err := validatePin(*b.MCP.ServerArtifact); err != nil {
				return err
			}
		}
		switch b.MCP.ApprovalRule {
		case "allow", "ask", "deny":
		default:
			return errors.New("invalid MCP Tool approval rule")
		}
		switch b.MCP.ExecutionMode {
		case "auto", "manual":
		default:
			return errors.New("invalid MCP Tool execution mode")
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
	if kind == WorkItemTypeEmit && o.Emit == nil {
		return errors.New("emit WorkItem output is missing")
	}
	if kind == WorkItemTypeTargetResolver && o.TargetResolver == nil {
		return errors.New("target resolver WorkItem output is missing")
	}
	if o.TargetResolver != nil {
		if err := o.TargetResolver.Validate(); err != nil {
			return err
		}
	}
	if o.Emit != nil {
		if o.Emit.ID == "" || o.Emit.AcceptedAt.IsZero() {
			return errors.New("emit receipt is incomplete")
		}
		if o.Emit.Destination.Channel == "" &&
			o.Emit.Destination.Event == "" {
			return errors.New("emit receipt has no destination")
		}
	}
	if o.Receipt != nil {
		if err := o.Receipt.EffectState.Validate(); err != nil {
			return err
		}
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
	if a.Deadline != nil && a.Deadline.Before(a.StartedAt) {
		return errors.New("attempt deadline is before it started")
	}
	if err := a.EffectState.Validate(); err != nil {
		return err
	}
	return nil
}

func (w TargetResolverWork) Validate() error {
	if err := validateID("target resolver target set", w.TargetSet); err != nil {
		return err
	}
	if len(w.Members) == 0 {
		return errors.New("target resolver WorkItem has no members")
	}
	seen := make(map[string]struct{}, len(w.Members))
	for _, member := range w.Members {
		if err := validateID("target resolver member", member); err != nil {
			return err
		}
		if _, duplicate := seen[member]; duplicate {
			return errors.New("target resolver member is duplicated")
		}
		seen[member] = struct{}{}
	}
	return nil
}

func (o TargetResolverOutput) Validate() error {
	if err := validateID("target resolver target set", o.TargetSet); err != nil {
		return err
	}
	if len(o.Members) == 0 {
		return errors.New("target resolver output has no members")
	}
	seen := make(map[string]struct{}, len(o.Members))
	for _, member := range o.Members {
		if err := validateID("target resolver output member", member); err != nil {
			return err
		}
		if _, duplicate := seen[member]; duplicate {
			return errors.New("target resolver output member is duplicated")
		}
		seen[member] = struct{}{}
	}
	return nil
}

func (w WaitRequest) Validate() error {
	if !w.Summary.Valid() ||
		!w.InputSchema.Valid() ||
		!w.Correlation.Valid() ||
		!w.CorrelationSchema.Valid() ||
		!w.DataSchema.Valid() ||
		!w.Request.Valid() {
		return errors.New("wait request json is not canonical")
	}
	if w.EventSource != "" {
		if err := validateID("wait event source", w.EventSource); err != nil {
			return err
		}
	}
	if w.Filter != nil {
		if err := w.Filter.Validate(); err != nil {
			return err
		}
	}
	if w.Assignees != nil {
		if err := w.Assignees.Validate(); err != nil {
			return err
		}
	}
	switch w.Kind {
	case WaitKindTimer:
		if w.Timer == nil ||
			len(w.AllowedResolutions) != 0 ||
			w.RuntimeBound ||
			w.Filter != nil ||
			w.Assignees != nil ||
			len(w.Request) != 0 {
			return errors.New("invalid timer wait request")
		}
		if err := w.Timer.Validate(); err != nil {
			return err
		}
	case WaitKindApproval, WaitKindManual, WaitKindHuman,
		WaitKindEvent, WaitKindChannel, WaitKindChild, WaitKindRecovery:
		if w.Timer != nil {
			return errors.New("non-timer wait contains timer state")
		}
	default:
		return errors.New("unsupported wait kind")
	}

	if w.Deadline != nil && w.Deadline.IsZero() {
		return errors.New("invalid wait deadline")
	}
	switch w.Kind {
	case WaitKindTimer:
		if w.Timer == nil {
			return errors.New("timer wait has no timer state")
		}
		if err := w.Timer.Validate(); err != nil {
			return err
		}
		if len(w.AllowedResolutions) != 0 ||
			w.Completion != nil ||
			w.Signal != "" ||
			w.Channel != "" {
			return errors.New("timer wait has incompatible fields")
		}

	case WaitKindEvent:
		if w.Timer != nil ||
			w.Completion != nil ||
			w.Signal == "" ||
			len(w.AllowedResolutions) != 0 {
			return errors.New("event wait is incomplete")
		}
		if w.Assignees != nil || len(w.Request) != 0 {
			return errors.New("event wait contains interactive fields")
		}

	case WaitKindChannel:
		if w.Timer != nil ||
			w.Completion != nil ||
			w.Channel == "" ||
			len(w.AllowedResolutions) != 0 {
			return errors.New("channel wait is incomplete")
		}
		if w.Assignees != nil || len(w.Request) != 0 {
			return errors.New("channel wait contains interactive fields")
		}

	case WaitKindChild:
		if w.Timer != nil ||
			w.Completion == nil ||
			len(w.AllowedResolutions) != 0 {
			return errors.New("completion wait is incomplete")
		}
		if err := w.Completion.Validate(); err != nil {
			return err
		}
		if w.Filter != nil ||
			w.Assignees != nil ||
			len(w.Request) != 0 {
			return errors.New("completion wait contains incompatible fields")
		}

	case WaitKindApproval,
		WaitKindManual,
		WaitKindHuman,
		WaitKindRecovery:
		if w.Timer != nil ||
			w.Completion != nil ||
			len(w.AllowedResolutions) == 0 {
			return errors.New("interactive wait is incomplete")
		}
		if w.Kind == WaitKindHuman && w.Assignees == nil {
			return errors.New("human wait has no assignees")
		}
		if w.Kind != WaitKindHuman &&
			(w.Assignees != nil || len(w.Request) != 0) {
			return errors.New("non-human interactive wait contains human fields")
		}

	default:
		return fmt.Errorf("unsupported wait kind %q", w.Kind)
	}

	return nil
}

func (h CompletionHandle) Validate() error {
	count := 0
	for _, present := range []bool{
		h.WorkItemID != "",
		h.FrameID != "",
		h.ExecutionID != "",
	} {
		if present {
			count++
		}
	}
	if count != 1 {
		if count == 0 {
			return errors.New("completion handle has no target")
		}
		return errors.New("completion handle has more than one target")
	}
	return nil
}

func (w Wait) Validate() error {
	if w.ID == "" || w.WorkItemID == "" || w.CreatedAt.IsZero() || w.UpdatedAt.IsZero() {
		return errors.New("invalid wait")
	}
	return w.Request.Validate()
}

func (p RetryPolicy) Validate() error {
	if p.MaxAttempts < 1 {
		return errors.New("retry maxAttempts must be positive")
	}
	if p.MaxAttempts == 1 {
		return nil
	}
	if err := validateWorkflowTimeout(p.Backoff.Initial); err != nil {
		return fmt.Errorf("retry initial backoff: %w", err)
	}
	if p.Backoff.Maximum != "" {
		if err := validateWorkflowTimeout(p.Backoff.Maximum); err != nil {
			return fmt.Errorf("retry maximum backoff: %w", err)
		}
	}
	if p.Backoff.Multiplier != 0 && p.Backoff.Multiplier < 1 {
		return errors.New("retry multiplier must be at least one")
	}
	if p.Backoff.Jitter < 0 || p.Backoff.Jitter > 1 {
		return errors.New("retry jitter must be between zero and one")
	}
	if p.PerAttemptTimeout != "" {
		if err := validateWorkflowTimeout(p.PerAttemptTimeout); err != nil {
			return fmt.Errorf("retry per-attempt timeout: %w", err)
		}
	}
	for _, category := range p.Match.Categories {
		switch category {
		case ErrorCategoryValidation,
			ErrorCategoryRouting,
			ErrorCategoryExecution,
			ErrorCategoryTransient,
			ErrorCategoryTimeout,
			ErrorCategoryAuthorization,
			ErrorCategoryLimit,
			ErrorCategoryCancellation,
			ErrorCategoryUncertainty:
		default:
			return fmt.Errorf("unsupported retry error category %q", category)
		}
	}
	for _, code := range p.Match.Codes {
		if err := validateID("retry error code", code); err != nil {
			return err
		}
	}
	return nil
}

func (s ExecutionSignal) Validate() error {
	if err := validateID("execution signal ID", s.ID); err != nil {
		return err
	}
	if err := validateID("execution signal name", s.Name); err != nil {
		return err
	}
	if !s.Data.Valid() || !s.Correlation.Valid() || s.CreatedAt.IsZero() {
		return errors.New("execution signal is invalid")
	}
	for _, waitID := range s.ConsumedBy {
		if err := validateID("signal consumed wait ID", string(waitID)); err != nil {
			return err
		}
	}
	for _, subscription := range s.ConsumedSubscriptions {
		if err := validateID(
			"signal consumed subscription",
			subscription,
		); err != nil {
			return err
		}
	}
	for _, subscription := range s.ConsumedSubscriptions {
		if err := validateID(
			"signal consumed subscription",
			subscription,
		); err != nil {
			return err
		}
	}
	return nil
}

func (m ChannelMessage) Validate() error {
	if err := validateID("channel message ID", m.ID); err != nil {
		return err
	}
	if err := validateID("channel message channel", m.Channel); err != nil {
		return err
	}
	if m.Sequence == 0 || !m.Data.Valid() || !m.Correlation.Valid() || m.CreatedAt.IsZero() {
		return errors.New("channel message is invalid")
	}
	for _, waitID := range m.ConsumedBy {
		if err := validateID("channel consumed wait ID", string(waitID)); err != nil {
			return err
		}
	}
	return nil
}

func (l ChannelLog) Validate() error {
	if err := validateID("channel name", l.Name); err != nil {
		return err
	}
	if l.NextSequence < uint64(len(l.Messages)) {
		return errors.New("channel sequence is invalid")
	}
	seen := make(map[string]struct{}, len(l.Messages))
	var previous uint64
	for _, message := range l.Messages {
		if err := message.Validate(); err != nil {
			return err
		}
		if message.Channel != l.Name {
			return errors.New("channel message belongs to another channel")
		}
		if message.Sequence <= previous {
			return errors.New("channel messages are not ordered")
		}
		if _, duplicate := seen[message.ID]; duplicate {
			return errors.New("channel message identity is duplicated")
		}
		seen[message.ID] = struct{}{}
		previous = message.Sequence
	}
	return nil
}

func (r EmitReceipt) Validate() error {
	if err := validateID("emit receipt ID", r.ID); err != nil {
		return err
	}
	if r.AcceptedAt.IsZero() {
		return errors.New("emit receipt timestamp is required")
	}
	switch {
	case r.Destination.Channel == "" && r.Destination.Event == "":
		return errors.New("emit receipt has no destination")
	case r.Destination.Channel != "" && r.Destination.Event != "":
		return errors.New("emit receipt has more than one destination")
	default:
		return nil
	}
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}
