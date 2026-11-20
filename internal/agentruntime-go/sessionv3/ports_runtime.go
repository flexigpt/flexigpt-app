package sessionv3

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"

	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

const (
	portInputOwnerUnavailableMessage             = "port input owner is unavailable"
	portInputNotDeclaredMessage                  = "port input is not declared"
	portInputFactsChangedMessage                 = "port input facts differ from its plan"
	portChannelParentExecutionUnavailableMessage = "parent port channel execution is unavailable"
	portChannelParentFrameUnavailableMessage     = "parent port channel frame is unavailable"
	portChannelForwardingCycleMessage            = "port channel forwarding contains a cycle"
)

const (
	portMessageKind       = "port"
	portStreamMessageKind = "port-stream"
	portMessageSeparator  = ":"

	publishedWorkflowPortNamespacePrefix = "workflow-port"
	publishedWorkflowPortInputPrefix     = "port-input"
	publishedWorkflowPortForwardPrefix   = "port-forward"
)

func bindWorkPortPlans(
	values []PortPlan,
	frameID ProgramFrameID,
) []PortPlan {
	output := clonePortPlans(values)
	for index := range output {
		output[index].OwnerFrameID = frameID
		output[index].ParentFrameID = ""
	}
	return output
}

func bindFramePortPlans(
	values []PortPlan,
	parentID ProgramFrameID,
	childID ProgramFrameID,
) []PortPlan {
	output := clonePortPlans(values)
	for index := range output {
		output[index].ParentFrameID = parentID
		output[index].OwnerFrameID = childID
	}
	return output
}

// preparePublishedWorkflowPortBoundaryLocked creates one private channel
// namespace for a nested Workflow invocation. Child input ports receive a
// snapshot of the caller channel. Child output records are forwarded to the
// caller channel through durable ChannelForward facts.
func (r *Runtime) preparePublishedWorkflowPortBoundaryLocked(
	session *Session,
	parentExecutionID ExecutionID,
	childExecution *Execution,
	childFrame *ProgramFrame,
) error {
	if session == nil || childExecution == nil || childFrame == nil {
		return errors.New("published workflow port owner is unavailable")
	}
	if parentExecutionID == "" ||
		mutableSessionExecution(session, parentExecutionID) == nil {
		return errors.New(portChannelParentExecutionUnavailableMessage)
	}
	if err := r.stageFramePortInputsLocked(
		session,
		childExecution,
		childFrame,
	); err != nil {
		return err
	}

	program, err := workflowProgramFromSnapshot(childFrame.Program)
	if err != nil {
		return err
	}
	published := make(map[string]WorkflowPort, len(program.Ports))
	for _, port := range program.Ports {
		published[port.Name] = port
	}

	namespace := portBoundaryMessageID(
		publishedWorkflowPortNamespacePrefix,
		string(childExecution.ID),
		string(childFrame.ID),
	)
	snapshot, aliases, err := namespaceWorkflowProgramChannels(
		childFrame.Program,
		namespace,
	)
	if err != nil {
		return err
	}

	inputsByName := make(map[string]PortInput, len(childFrame.PortInputs))
	for _, input := range childFrame.PortInputs {
		inputsByName[input.Name] = input
	}
	localInputs := make([]PortInput, 0, len(childFrame.PortInputs))
	hasOutputForward := false

	for _, plan := range childFrame.Ports {
		port, found := published[plan.Name]
		if !found {
			return errors.New(
				"published workflow port is unavailable at the child boundary",
			)
		}
		localChannel, found := aliases[port.Channel]
		if !found {
			return errors.New(
				"published workflow port has no invocation-local channel",
			)
		}

		switch plan.Direction {
		case WorkflowPortDirectionInput:
			channel, found := workflowChannelForName(
				program,
				port.Channel,
			)
			if !found {
				return errors.New(
					"published workflow input port has no declared channel",
				)
			}

			source := inputsByName[plan.Name]
			local := PortInput{
				Name:     plan.Name,
				Channel:  localChannel,
				Streamed: plan.Stream,
			}
			for index, value := range source.Values {
				sequence, inputErr := r.appendPublishedWorkflowPortInputLocked(
					childExecution,
					childFrame.ID,
					plan,
					channel,
					localChannel,
					index,
					value,
				)
				if inputErr != nil {
					return inputErr
				}
				local.Values = append(local.Values, value.Clone())
				if sequence > local.Cursor {
					local.Cursor = sequence
				}
			}
			if log := executionChannel(childExecution, localChannel); log != nil &&
				log.NextSequence > local.Cursor {
				local.Cursor = log.NextSequence
			}
			localInputs = append(localInputs, local)

		case WorkflowPortDirectionOutput:
			if err := appendChannelForward(
				childExecution,
				ChannelForward{
					SourceChannel:     localChannel,
					TargetExecutionID: parentExecutionID,
					TargetChannel:     plan.Channel,
					PortName:          plan.Name,
					Stream:            plan.Stream,
					Schema:            plan.Schema.Clone(),
					ChannelSchema:     plan.ChannelSchema.Clone(),
					MaxMessages:       plan.MaxMessages,
					MaxBytes:          plan.MaxBytes,
				},
			); err != nil {
				return err
			}
			hasOutputForward = true

		default:
			return errors.New("published workflow port direction is invalid")
		}
	}

	if len(localInputs) != 0 {
		snapshot, err = workflowProgramWithPortInputs(snapshot, localInputs)
		if err != nil {
			return err
		}
	}
	childFrame.Program = snapshot
	childFrame.UpdatedAt = r.now()

	// A published Workflow channel owns its final and streamed records. The
	// containing child frame must not append its internal block-result value
	// as a second final port record.
	if hasOutputForward {
		childFrame.PortDelivery = FramePortDeliveryModeForwardOnly
	}
	r.touchExecutionLocked(session, childExecution)
	return nil
}

func (r *Runtime) appendPublishedWorkflowPortInputLocked(
	execution *Execution,
	frameID ProgramFrameID,
	plan PortPlan,
	channelDeclaration WorkflowChannel,
	localChannel string,
	index int,
	value jsonutil.CanonicalValue,
) (uint64, error) {
	if execution == nil {
		return 0, errors.New("published workflow input execution is unavailable")
	}
	if err := validateWorkflowSchema(plan.Schema, value); err != nil {
		return 0, fmt.Errorf("validate published workflow input port: %w", err)
	}
	if err := validateWorkflowSchema(
		channelDeclaration.Schema,
		value,
	); err != nil {
		return 0, fmt.Errorf(
			"validate published workflow input channel: %w",
			err,
		)
	}

	maxMessages, maxBytes := publishedWorkflowChannelRetention(
		channelDeclaration,
	)
	channel, err := mutableChannelLog(
		execution,
		localChannel,
		maxMessages,
		maxBytes,
	)
	if err != nil {
		return 0, err
	}

	messageID := portBoundaryMessageID(
		publishedWorkflowPortInputPrefix,
		string(frameID),
		plan.Name,
		strconv.Itoa(index),
	)
	for _, existing := range channel.Messages {
		if existing.ID != messageID {
			continue
		}
		if !canonicalValuesEqual(existing.Data, value) {
			return 0, errors.New(
				"published workflow input port changed during transfer",
			)
		}
		return existing.Sequence, nil
	}
	if len(channel.Messages) >= channel.MaxMessages {
		return 0, errors.New("published workflow input channel retention limit exceeded")
	}

	correlation := jsonutil.MustCanonicalValue(nil)
	if channel.Bytes+len(value)+len(correlation) > channel.MaxBytes {
		return 0, errors.New("published workflow input channel byte limit exceeded")
	}
	if err := r.reserveChannelMessageBudgetLocked(
		execution,
		value,
		correlation,
	); err != nil {
		return 0, err
	}

	channel.NextSequence++
	kind := portMessageKind
	if plan.Stream {
		kind = portStreamMessageKind
	}
	message := ChannelMessage{
		ID:          messageID,
		Sequence:    channel.NextSequence,
		Channel:     channel.Name,
		Kind:        kind,
		Data:        value.Clone(),
		Correlation: correlation,
		CreatedAt:   r.now(),
	}
	channel.Messages = append(channel.Messages, message)
	channel.Bytes += len(message.Data) + len(message.Correlation)
	return message.Sequence, nil
}

func publishedWorkflowChannelRetention(
	channel WorkflowChannel,
) (maxMessages, maxBytes int) {
	maxMessages = maximumCoordinationRecords
	if channel.MaxMessages > 0 && channel.MaxMessages < maxMessages {
		maxMessages = channel.MaxMessages
	}
	maxBytes = maximumCoordinationBytes
	if channel.MaxBytes > 0 && channel.MaxBytes < maxBytes {
		maxBytes = channel.MaxBytes
	}
	return maxMessages, maxBytes
}

func appendChannelForward(
	execution *Execution,
	value ChannelForward,
) error {
	if execution == nil {
		return errors.New("channel forward execution is unavailable")
	}
	if err := value.Validate(); err != nil {
		return err
	}
	for _, existing := range execution.ChannelForwards {
		if existing.SourceChannel != value.SourceChannel ||
			existing.TargetExecutionID != value.TargetExecutionID ||
			existing.TargetChannel != value.TargetChannel ||
			existing.PortName != value.PortName {
			continue
		}
		if !sameChannelForward(existing, value) {
			return errors.New("published workflow channel forward facts changed")
		}
		return nil
	}
	execution.ChannelForwards = append(execution.ChannelForwards, value)
	return nil
}

func sameChannelForward(left, right ChannelForward) bool {
	return left.SourceChannel == right.SourceChannel &&
		left.TargetExecutionID == right.TargetExecutionID &&
		left.TargetChannel == right.TargetChannel &&
		left.PortName == right.PortName &&
		left.Stream == right.Stream &&
		left.MaxMessages == right.MaxMessages &&
		left.MaxBytes == right.MaxBytes &&
		canonicalValuesEqual(left.Schema, right.Schema) &&
		canonicalValuesEqual(left.ChannelSchema, right.ChannelSchema)
}

func namespaceWorkflowProgramChannels(
	snapshot ProgramSnapshot,
	namespace string,
) (ProgramSnapshot, map[string]string, error) {
	program, err := workflowProgramFromSnapshot(snapshot)
	if err != nil {
		return ProgramSnapshot{}, nil, err
	}

	aliases := make(map[string]string, len(program.Channels))
	for _, channel := range program.Channels {
		value := namespace + ":" + channel.Name
		if err := validateID("workflow invocation channel", value); err != nil {
			return ProgramSnapshot{}, nil, err
		}
		aliases[channel.Name] = value
	}
	for index := range program.Channels {
		program.Channels[index].Name = aliases[program.Channels[index].Name]
	}
	for index := range program.Ports {
		value, found := aliases[program.Ports[index].Channel]
		if !found {
			return ProgramSnapshot{}, nil, errors.New(
				"published workflow port has no declared local channel",
			)
		}
		program.Ports[index].Channel = value
	}
	if err := namespaceWorkflowBlockChannels(&program.Root, aliases); err != nil {
		return ProgramSnapshot{}, nil, err
	}

	payload, err := canonicalWorkflowValue(program)
	if err != nil {
		return ProgramSnapshot{}, nil, err
	}
	snapshot.Payload = payload
	return snapshot, aliases, nil
}

func namespaceWorkflowBlockChannels(
	block *WorkflowBlock,
	aliases map[string]string,
) error {
	if block == nil {
		return nil
	}
	for index := range block.Steps {
		step := &block.Steps[index]
		if step.Call != nil {
			for portIndex := range step.Call.Ports {
				channel, err := namespaceWorkflowChannel(
					aliases,
					step.Call.Ports[portIndex].Channel,
				)
				if err != nil {
					return err
				}
				step.Call.Ports[portIndex].Channel = channel
			}
		}
		if step.Wait != nil && step.Wait.Channel != nil {
			channel, err := namespaceWorkflowChannel(
				aliases,
				step.Wait.Channel.Channel,
			)
			if err != nil {
				return err
			}
			step.Wait.Channel.Channel = channel
		}
		if step.Emit != nil && step.Emit.Destination.Channel != "" {
			channel, err := namespaceWorkflowChannel(
				aliases,
				step.Emit.Destination.Channel,
			)
			if err != nil {
				return err
			}
			step.Emit.Destination.Channel = channel
		}
		if step.Effect != nil {
			if err := namespaceWorkflowBlockChannels(
				step.Effect.Reconcile,
				aliases,
			); err != nil {
				return err
			}
			if err := namespaceWorkflowBlockChannels(
				step.Effect.Compensate,
				aliases,
			); err != nil {
				return err
			}
		}

		switch {
		case step.Scope != nil:
			if err := namespaceWorkflowBlockChannels(step.Scope, aliases); err != nil {
				return err
			}
		case step.Choose != nil:
			for caseIndex := range step.Choose.Cases {
				if err := namespaceWorkflowBlockChannels(
					&step.Choose.Cases[caseIndex].Do,
					aliases,
				); err != nil {
					return err
				}
			}
			if err := namespaceWorkflowBlockChannels(
				step.Choose.Default,
				aliases,
			); err != nil {
				return err
			}
		case step.Parallel != nil:
			for branchIndex := range step.Parallel.Branches {
				if err := namespaceWorkflowBlockChannels(
					&step.Parallel.Branches[branchIndex].Do,
					aliases,
				); err != nil {
					return err
				}
			}
		case step.Foreach != nil:
			if err := namespaceWorkflowBlockChannels(&step.Foreach.Do, aliases); err != nil {
				return err
			}
		case step.Repeat != nil:
			if err := namespaceWorkflowBlockChannels(&step.Repeat.Do, aliases); err != nil {
				return err
			}
		case step.Machine != nil:
			for stateIndex := range step.Machine.States {
				machineState := &step.Machine.States[stateIndex]
				if err := namespaceWorkflowBlockChannels(
					machineState.Entry,
					aliases,
				); err != nil {
					return err
				}
				if err := namespaceWorkflowBlockChannels(
					machineState.Exit,
					aliases,
				); err != nil {
					return err
				}
				for transitionIndex := range machineState.Transitions {
					if err := namespaceWorkflowBlockChannels(
						machineState.Transitions[transitionIndex].Do,
						aliases,
					); err != nil {
						return err
					}
				}
			}
		case step.Saga != nil:
			if err := namespaceWorkflowBlockChannels(&step.Saga.Do, aliases); err != nil {
				return err
			}
		}
	}
	for index := range block.Catch {
		if err := namespaceWorkflowBlockChannels(
			&block.Catch[index].Do,
			aliases,
		); err != nil {
			return err
		}
	}
	return namespaceWorkflowBlockChannels(block.Finally, aliases)
}

func namespaceWorkflowChannel(
	aliases map[string]string,
	channel string,
) (string, error) {
	value, found := aliases[channel]
	if !found {
		return "", errors.New("workflow channel is unavailable in this invocation")
	}
	return value, nil
}

func llmPortPlansForFrame(
	frame ProgramFrame,
) []PortPlan {
	output := make([]PortPlan, 0, len(frame.Ports))
	for _, plan := range frame.Ports {
		if plan.Direction == WorkflowPortDirectionInput ||
			(plan.Direction == WorkflowPortDirectionOutput &&
				plan.Stream) {
			output = append(output, plan)
		}
	}
	return output
}

func (r *Runtime) stageWorkPortInputsLocked(
	session *Session,
	execution *Execution,
	item *WorkItem,
) error {
	if session == nil || execution == nil || item == nil {
		return errors.New("work port owner is unavailable")
	}
	if len(item.Ports) == 0 {
		return nil
	}
	if err := validateWorkPortOwnership(*execution, *item); err != nil {
		return err
	}

	inputs, err := r.portInputsLocked(
		session,
		execution,
		item.Ports,
		item.PortInputs,
	)
	if err != nil {
		return err
	}
	item.PortInputs = inputs
	return nil
}

func validateWorkPortOwnership(
	execution Execution,
	item WorkItem,
) error {
	frame := mutableFrame(&execution, item.FrameID)
	if frame == nil {
		return errors.New("work port owner frame is unavailable")
	}
	for _, plan := range item.Ports {
		if plan.OwnerFrameID == "" || plan.OwnerFrameID != item.FrameID {
			return errors.New("work port owner differs from its work item frame")
		}
		parent := frame.ParentFrameID
		if parent == "" && execution.ParentFrameID != "" {
			parent = execution.ParentFrameID
		}
		if plan.ParentFrameID != "" && parent != plan.ParentFrameID {
			return errors.New("work port parent ownership differs from its frame")
		}
	}
	return nil
}

func validateFramePortOwnership(
	execution Execution,
	frame ProgramFrame,
) error {
	parent := frame.ParentFrameID
	if parent == "" && execution.ParentFrameID != "" {
		parent = execution.ParentFrameID
	}
	for _, plan := range frame.Ports {
		if plan.OwnerFrameID == "" || plan.OwnerFrameID != frame.ID {
			return errors.New("frame port owner differs from its program frame")
		}
		if plan.ParentFrameID != "" && plan.ParentFrameID != parent {
			return errors.New("frame port parent ownership differs from its program frame")
		}
	}
	return nil
}

func mergeWorkPortPlans(
	work []PortPlan,
	frame []PortPlan,
) ([]PortPlan, error) {
	output := clonePortPlans(work)
	for _, plan := range frame {
		for _, current := range output {
			if current.Name == plan.Name {
				return nil, errors.New("work and frame port plans have a duplicate name")
			}
		}
		output = append(output, plan)
	}
	return output, validateUniquePortPlans(output)
}

func (r *Runtime) stageFramePortInputsLocked(
	session *Session,
	execution *Execution,
	frame *ProgramFrame,
) error {
	if session == nil || execution == nil || frame == nil {
		return errors.New("frame port owner is unavailable")
	}
	if len(frame.Ports) == 0 {
		return nil
	}
	if err := validateFramePortOwnership(*execution, *frame); err != nil {
		return err
	}
	inputs, err := r.portInputsLocked(
		session,
		execution,
		frame.Ports,
		frame.PortInputs,
	)
	if err != nil {
		return err
	}
	frame.PortInputs = inputs
	return nil
}

func (r *Runtime) portInputsLocked(
	session *Session,
	execution *Execution,
	plans []PortPlan,
	existing []PortInput,
) ([]PortInput, error) {
	if session == nil || execution == nil {
		return nil, errors.New(portInputOwnerUnavailableMessage)
	}

	output := clonePortInputs(existing)
	inputPlans := make(map[string]PortPlan, len(plans))
	for _, plan := range plans {
		if plan.Direction == WorkflowPortDirectionInput {
			inputPlans[plan.Name] = plan
		}
	}
	for _, input := range output {
		plan, found := inputPlans[input.Name]
		if !found {
			return nil, errors.New(portInputNotDeclaredMessage)
		}
		if input.Channel != plan.Channel || input.Streamed != plan.Stream {
			return nil, errors.New(portInputFactsChangedMessage)
		}
	}

	for _, plan := range plans {
		if plan.Direction != WorkflowPortDirectionInput {
			continue
		}
		if err := plan.Validate(); err != nil {
			return nil, err
		}
		channelExecution, err := portChannelExecution(
			session,
			execution,
			plan,
		)
		if err != nil {
			return nil, err
		}
		channel := executionChannel(channelExecution, plan.Channel)
		if channel == nil {
			maxMessages, maxBytes := portRetention(plan)
			channel, err = mutableChannelLog(
				channelExecution,
				plan.Channel,
				maxMessages,
				maxBytes,
			)
			if err != nil {
				return nil, err
			}
			r.touchExecutionLocked(session, channelExecution)
		}

		index := -1
		for current := range output {
			if output[current].Name == plan.Name {
				index = current
				break
			}
		}
		if index < 0 {
			output = append(output, PortInput{
				Name:     plan.Name,
				Channel:  plan.Channel,
				Streamed: plan.Stream,
			})
			index = len(output) - 1
		}
		input := &output[index]
		if input.Channel != plan.Channel {
			return nil, errors.New("port input channel changed")
		}

		for _, message := range channel.Messages {
			if message.Sequence <= input.Cursor {
				continue
			}
			if err := validateWorkflowSchema(plan.Schema, message.Data); err != nil {
				return nil, fmt.Errorf(
					"port input %q schema: %w",
					plan.Name,
					err,
				)
			}
			if err := validateWorkflowSchema(
				plan.ChannelSchema,
				message.Data,
			); err != nil {
				return nil, fmt.Errorf(
					"port input %q channel schema: %w",
					plan.Name,
					err,
				)
			}
			input.Values = append(input.Values, message.Data.Clone())
			input.Cursor = message.Sequence
		}
	}
	if err := validateUniquePortInputs(output); err != nil {
		return nil, err
	}
	return output, nil
}

func executionChannel(
	execution *Execution,
	name string,
) *ChannelLog {
	if execution == nil {
		return nil
	}
	for index := range execution.Channels {
		if execution.Channels[index].Name == name {
			return &execution.Channels[index]
		}
	}
	return nil
}

func portChannelExecution(
	session *Session,
	execution *Execution,
	plan PortPlan,
) (*Execution, error) {
	return resolvePortChannelExecution(
		session,
		execution,
		plan,
		map[portForwardingIdentity]struct{}{},
	)
}

type portForwardingIdentity struct {
	ExecutionID ExecutionID
	FrameID     ProgramFrameID
	Name        string
	Channel     string
	Direction   WorkflowPortDirection
	Stream      bool
}

func resolvePortChannelExecution(
	session *Session,
	execution *Execution,
	plan PortPlan,
	visited map[portForwardingIdentity]struct{},
) (*Execution, error) {
	if session == nil || execution == nil {
		return nil, errors.New(portInputOwnerUnavailableMessage)
	}

	frameID := plan.ParentFrameID
	if frameID == "" {
		frameID = plan.OwnerFrameID
	}
	if frameID == "" {
		return execution, nil
	}

	ownerExecution, ownerFrame, err := portFrameOwner(
		session,
		execution,
		frameID,
	)
	if err != nil {
		return nil, err
	}

	identity := portForwardingIdentity{
		ExecutionID: ownerExecution.ID,
		FrameID:     ownerFrame.ID,
		Name:        plan.Name,
		Channel:     plan.Channel,
		Direction:   plan.Direction,
		Stream:      plan.Stream,
	}
	if _, found := visited[identity]; found {
		return nil, errors.New(portChannelForwardingCycleMessage)
	}
	visited[identity] = struct{}{}

	if forwarding := forwardingPortPlan(*ownerFrame, plan); forwarding != nil &&
		forwarding.ParentFrameID != "" {
		return resolvePortChannelExecution(
			session,
			ownerExecution,
			*forwarding,
			visited,
		)
	}
	return ownerExecution, nil
}

func portFrameOwner(
	session *Session,
	execution *Execution,
	frameID ProgramFrameID,
) (*Execution, *ProgramFrame, error) {
	current := execution
	for current != nil {
		if frame := mutableFrame(current, frameID); frame != nil {
			return current, frame, nil
		}
		if current.ParentExecutionID == "" {
			break
		}
		current = mutableSessionExecution(
			session,
			current.ParentExecutionID,
		)
	}
	if execution.ParentExecutionID == "" {
		return nil, nil, errors.New(portChannelParentFrameUnavailableMessage)
	}
	return nil, nil, errors.New(portChannelParentExecutionUnavailableMessage)
}

func forwardingPortPlan(
	frame ProgramFrame,
	plan PortPlan,
) *PortPlan {
	for index := range frame.Ports {
		candidate := &frame.Ports[index]
		if candidate.Name != plan.Name ||
			candidate.Channel != plan.Channel ||
			candidate.Direction != plan.Direction ||
			candidate.Stream != plan.Stream {
			continue
		}
		return candidate
	}
	return nil
}

func (r *Runtime) deliverWorkPortsLocked(
	session *Session,
	execution *Execution,
	item *WorkItem,
) error {
	if session == nil || execution == nil || item == nil ||
		len(item.Ports) == 0 || item.Output == nil {
		return nil
	}
	if err := validateWorkPortOwnership(*execution, *item); err != nil {
		return err
	}

	value, err := portOutputValue(*item)
	if err != nil {
		return err
	}
	for _, plan := range item.Ports {
		if plan.Direction != WorkflowPortDirectionOutput {
			continue
		}

		// Streamed ports contain only ordered stream records emitted through
		// WorkPortStream or the LLM stream bridge. A final result must never be
		// converted into a synthetic stream record.
		if plan.Stream {
			continue
		}
		if err := r.appendPortMessageLocked(
			session,
			execution,
			plan,
			portMessageID(item.ID, plan.Name, 0),
			portMessageKind,
			value,
		); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runtime) deliverFramePortsLocked(
	session *Session,
	execution *Execution,
	frame *ProgramFrame,
) error {
	if session == nil || execution == nil || frame == nil ||
		len(frame.Ports) == 0 || len(frame.Output) == 0 {
		return nil
	}
	if !frame.PortDelivery.DeliversFinalOutput() {
		return nil
	}
	if err := validateFramePortOwnership(*execution, *frame); err != nil {
		return err
	}

	for _, plan := range frame.Ports {
		if plan.Direction != WorkflowPortDirectionOutput ||
			plan.Stream {
			continue
		}

		// A child ProgramFrame owns final-output delivery. Stream delivery
		// occurs on the child WorkItem before child settlement.
		if err := r.appendPortMessageLocked(
			session,
			execution,
			plan,
			portMessageID(WorkItemID(frame.ID), plan.Name, 0),
			portMessageKind,
			frame.Output,
		); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runtime) appendWorkPortStream(
	ctx context.Context,
	sessionID SessionID,
	executionID ExecutionID,
	workItemID WorkItemID,
	name string,
	value jsonutil.CanonicalValue,
) error {
	if len(value) == 0 || !value.Valid() {
		return errors.New("port stream value is not canonical json")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return ErrRuntimeClosed
	}
	session, execution, err := r.executionLocked(sessionID, executionID)
	if err != nil {
		return err
	}
	item := mutableWorkItem(execution, workItemID)
	if item == nil ||
		item.State != WorkItemStateDispatched ||
		execution.Controls.CancellationRequested ||
		execution.State.Terminal() {
		return context.Canceled
	}
	if err := validateWorkPortOwnership(*execution, *item); err != nil {
		return err
	}

	var plan *PortPlan
	for index := range item.Ports {
		current := &item.Ports[index]
		if current.Name != name ||
			current.Direction != WorkflowPortDirectionOutput ||
			!current.Stream {
			continue
		}
		plan = current
		break
	}
	if plan == nil {
		return errors.New("streamed port is not declared for the work item")
	}

	sequence := nextPortStreamSequence(item, name)
	if err := r.appendPortMessageLocked(
		session,
		execution,
		*plan,
		portMessageID(item.ID, name, sequence),
		portStreamMessageKind,
		value,
	); err != nil {
		return err
	}
	execution = mutableSessionExecution(session, executionID)
	if execution == nil {
		return ErrExecutionNotFound
	}
	item = mutableWorkItem(execution, workItemID)
	if item == nil {
		return ErrExecutionNotFound
	}
	item.UpdatedAt = r.now()
	r.touchExecutionLocked(session, execution)

	if err := r.persistLocked(ctx, "portStream", session); err != nil {
		return err
	}
	r.publishLocked(ctx, session, execution.ID, RuntimeEventWorkChanged)
	return nil
}

func (r *Runtime) deliverLLMStreamPortsLocked(
	session *Session,
	execution *Execution,
	item *WorkItem,
	value string,
) error {
	if session == nil || execution == nil || item == nil || value == "" {
		return nil
	}
	streamValue := jsonutil.MustCanonicalValue(value)
	for _, plan := range item.Ports {
		// A streamed port carries incremental records only.
		if plan.Direction != WorkflowPortDirectionOutput || !plan.Stream {
			continue
		}
		sequence := nextPortStreamSequence(item, plan.Name)
		if err := r.appendPortMessageLocked(
			session,
			execution,
			plan,
			portMessageID(item.ID, plan.Name, sequence),
			portStreamMessageKind,
			streamValue,
		); err != nil {
			return err
		}
	}
	return nil
}

func nextPortStreamSequence(
	item *WorkItem,
	name string,
) uint64 {
	index := portStreamIndex(item.PortStreams, name)
	if index < 0 {
		item.PortStreams = append(item.PortStreams, PortStream{
			Name:         name,
			NextSequence: 1,
		})
		return 1
	}
	item.PortStreams[index].NextSequence++
	return item.PortStreams[index].NextSequence
}

func portMessageID(
	workID WorkItemID,
	name string,
	sequence uint64,
) string {
	value := string(workID) + portMessageSeparator + name
	if sequence != 0 {
		value += portMessageSeparator + strconv.FormatUint(sequence, 10)
	}
	return value
}

func (r *Runtime) appendPortMessageLocked(
	session *Session,
	execution *Execution,
	plan PortPlan,
	messageID string,
	kind string,
	value jsonutil.CanonicalValue,
) error {
	if err := plan.Validate(); err != nil {
		return err
	}
	if !value.Valid() {
		return errors.New("port output is not canonical json")
	}

	if err := validateWorkflowSchema(plan.Schema, value); err != nil {
		return fmt.Errorf("validate port output: %w", err)
	}
	if err := validateWorkflowSchema(plan.ChannelSchema, value); err != nil {
		return fmt.Errorf(
			"validate port channel output: %w",
			err,
		)
	}

	maxMessages, maxBytes := portRetention(plan)
	channelExecution, err := portChannelExecution(
		session,
		execution,
		plan,
	)
	if err != nil {
		return err
	}
	channelExecutionID := channelExecution.ID
	channel, err := mutableChannelLog(
		channelExecution,
		plan.Channel,
		maxMessages,
		maxBytes,
	)
	if err != nil {
		return err
	}
	for _, existing := range channel.Messages {
		if existing.ID == messageID {
			return nil
		}
	}
	if len(channel.Messages) >= channel.MaxMessages {
		return errors.New("port channel retention limit exceeded")
	}
	if channel.Bytes+len(value) > channel.MaxBytes {
		return errors.New("port channel byte limit exceeded")
	}
	if err := r.reserveChannelMessageBudgetLocked(
		channelExecution,
		value,
		nil,
	); err != nil {
		return err
	}

	now := r.now()
	channel.NextSequence++
	message := ChannelMessage{
		ID:        messageID,
		Sequence:  channel.NextSequence,
		Channel:   channel.Name,
		Kind:      kind,
		Data:      value.Clone(),
		CreatedAt: now,
	}
	channel.Messages = append(channel.Messages, message)
	channel.Bytes += len(message.Data)
	if err := r.wakeChannelWaitsLocked(session, channelExecution, message); err != nil {
		return err
	}
	if err := r.forwardChannelMessageLocked(
		session,
		channelExecution,
		message,
	); err != nil {
		return err
	}
	channelExecution = mutableSessionExecution(session, channelExecutionID)
	if channelExecution == nil {
		return errors.New(portChannelParentExecutionUnavailableMessage)
	}
	r.touchExecutionLocked(session, channelExecution)

	// The receiver can belong to another independent Execution. Launching
	// ready work here keeps stream and final-port delivery reactive instead
	// of waiting for the sending child to settle.
	r.launchReadyWork(
		context.Background(),
		session.ID,
		r.readyWorkForSessionLocked(session),
	)
	return nil
}

type channelForwardRoute struct {
	SourceExecution ExecutionID
	SourceChannel   string
	TargetExecution ExecutionID
	TargetChannel   string
	PortName        string
}

func (r *Runtime) forwardChannelMessageLocked(
	session *Session,
	sourceExecution *Execution,
	message ChannelMessage,
) error {
	return r.forwardChannelMessageFromLocked(
		session,
		sourceExecution,
		message,
		map[channelForwardRoute]struct{}{},
	)
}

func (r *Runtime) forwardChannelMessageFromLocked(
	session *Session,
	sourceExecution *Execution,
	message ChannelMessage,
	visited map[channelForwardRoute]struct{},
) error {
	if session == nil || sourceExecution == nil {
		return errors.New("channel forward source is unavailable")
	}

	forwards := append(
		[]ChannelForward(nil),
		sourceExecution.ChannelForwards...,
	)
	for _, forward := range forwards {
		if forward.SourceChannel != message.Channel {
			continue
		}
		if err := forward.Validate(); err != nil {
			return err
		}
		route := channelForwardRoute{
			SourceExecution: sourceExecution.ID,
			SourceChannel:   forward.SourceChannel,
			TargetExecution: forward.TargetExecutionID,
			TargetChannel:   forward.TargetChannel,
			PortName:        forward.PortName,
		}
		if _, found := visited[route]; found {
			return errors.New(portChannelForwardingCycleMessage)
		}
		if route.SourceExecution == route.TargetExecution &&
			route.SourceChannel == route.TargetChannel {
			return errors.New(portChannelForwardingCycleMessage)
		}
		visited[route] = struct{}{}

		target := mutableSessionExecution(
			session,
			forward.TargetExecutionID,
		)
		if target == nil {
			delete(visited, route)
			return errors.New(portChannelParentExecutionUnavailableMessage)
		}
		if err := validateWorkflowSchema(forward.Schema, message.Data); err != nil {
			delete(visited, route)
			return fmt.Errorf("validate forwarded workflow port: %w", err)
		}
		if err := validateWorkflowSchema(
			forward.ChannelSchema,
			message.Data,
		); err != nil {
			delete(visited, route)
			return fmt.Errorf(
				"validate forwarded workflow port channel: %w",
				err,
			)
		}

		forwarded, appended, err := r.appendForwardedChannelMessageLocked(
			session,
			target,
			sourceExecution.ID,
			forward,
			message,
		)
		if err == nil && appended {
			target = mutableSessionExecution(
				session,
				forward.TargetExecutionID,
			)
			if target == nil {
				err = errors.New(portChannelParentExecutionUnavailableMessage)
			} else {
				err = r.forwardChannelMessageFromLocked(
					session,
					target,
					forwarded,
					visited,
				)
			}
		}
		delete(visited, route)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Runtime) appendForwardedChannelMessageLocked(
	session *Session,
	target *Execution,
	sourceExecutionID ExecutionID,
	forward ChannelForward,
	source ChannelMessage,
) (ChannelMessage, bool, error) {
	if session == nil || target == nil {
		return ChannelMessage{}, false, errors.New(
			portChannelParentExecutionUnavailableMessage,
		)
	}
	maxMessages, maxBytes := portRetention(PortPlan{
		MaxMessages: forward.MaxMessages,
		MaxBytes:    forward.MaxBytes,
	})
	channel, err := mutableChannelLog(
		target,
		forward.TargetChannel,
		maxMessages,
		maxBytes,
	)
	if err != nil {
		return ChannelMessage{}, false, err
	}
	messageID := portBoundaryMessageID(
		publishedWorkflowPortForwardPrefix,
		string(sourceExecutionID),
		source.Channel,
		source.ID,
		string(forward.TargetExecutionID),
		forward.TargetChannel,
		forward.PortName,
	)
	for _, existing := range channel.Messages {
		if existing.ID == messageID {
			return existing, false, nil
		}
	}
	if len(channel.Messages) >= channel.MaxMessages {
		return ChannelMessage{}, false, errors.New(
			"forwarded workflow port channel retention limit exceeded",
		)
	}
	if channel.Bytes+len(source.Data)+len(source.Correlation) > channel.MaxBytes {
		return ChannelMessage{}, false, errors.New(
			"forwarded workflow port channel byte limit exceeded",
		)
	}
	if err := r.reserveChannelMessageBudgetLocked(
		target,
		source.Data,
		source.Correlation,
	); err != nil {
		return ChannelMessage{}, false, err
	}

	now := r.now()
	channel.NextSequence++
	message := ChannelMessage{
		ID:          messageID,
		Sequence:    channel.NextSequence,
		Channel:     channel.Name,
		Kind:        source.Kind,
		Data:        source.Data.Clone(),
		Correlation: source.Correlation.Clone(),
		CreatedAt:   now,
	}
	channel.Messages = append(channel.Messages, message)
	channel.Bytes += len(message.Data) + len(message.Correlation)
	if err := r.wakeChannelWaitsLocked(session, target, message); err != nil {
		return ChannelMessage{}, false, err
	}

	target = mutableSessionExecution(session, target.ID)
	if target == nil {
		return ChannelMessage{}, false, errors.New(
			portChannelParentExecutionUnavailableMessage,
		)
	}
	r.touchExecutionLocked(session, target)
	return message, true, nil
}

func portBoundaryMessageID(prefix string, values ...string) string {
	sum := sha256.New()
	for _, value := range values {
		_, _ = sum.Write([]byte(value))
		_, _ = sum.Write([]byte{0})
	}
	return prefix + ":" + hex.EncodeToString(sum.Sum(nil))
}

func portRetention(plan PortPlan) (maxMessages, maxBytes int) {
	maxMessages = maximumCoordinationRecords
	if plan.MaxMessages > 0 && plan.MaxMessages < maxMessages {
		maxMessages = plan.MaxMessages
	}
	maxBytes = maximumCoordinationBytes
	if plan.MaxBytes > 0 && plan.MaxBytes < maxBytes {
		maxBytes = plan.MaxBytes
	}
	return maxMessages, maxBytes
}

func portOutputValue(item WorkItem) (jsonutil.CanonicalValue, error) {
	if item.Output == nil {
		return nil, errors.New("port owner has no output")
	}
	if len(item.Output.Value) != 0 {
		return item.Output.Value.Clone(), nil
	}
	if item.Output.Tool != nil &&
		len(item.Output.Tool.Structured) != 0 {
		return item.Output.Tool.Structured.Clone(), nil
	}
	if item.Output.LLM != nil {
		return canonicalWorkflowValue(item.Output.LLM)
	}
	return nil, errors.New("port owner output has no semantic value")
}

func (r *Runtime) validatePortPlansForDispatch(
	ctx context.Context,
	request WorkDispatchRequest,
) error {
	if len(request.WorkItem.Ports) == 0 {
		return nil
	}
	if err := validateUniquePortPlans(request.WorkItem.Ports); err != nil {
		return err
	}
	validator, found := r.deps.Binder.(PortPlanValidator)
	if !found {
		return errors.New("port adapter is unavailable for this work item")
	}
	return validator.ValidatePortPlans(ctx, request)
}
