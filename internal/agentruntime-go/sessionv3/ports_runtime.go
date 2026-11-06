package sessionv3

import (
	"context"
	"errors"
	"fmt"
	"strconv"
)

const (
	portMessageKind       = "port"
	portStreamMessageKind = "port-stream"
	portMessageSeparator  = ":"
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

func streamPortPlansForFrame(
	frame ProgramFrame,
) []PortPlan {
	output := make([]PortPlan, 0, len(frame.Ports))
	for _, plan := range frame.Ports {
		if plan.Direction == WorkflowPortDirectionOutput && plan.Stream {
			output = append(output, plan)
		}
	}
	return output
}

func (r *Runtime) stageWorkPortInputsLocked(
	execution *Execution,
	item *WorkItem,
) error {
	if execution == nil || item == nil {
		return errors.New("work port owner is unavailable")
	}
	if len(item.Ports) == 0 {
		return nil
	}

	inputs, err := r.portInputsLocked(
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

func (r *Runtime) stageFramePortInputsLocked(
	execution *Execution,
	frame *ProgramFrame,
) error {
	if execution == nil || frame == nil {
		return errors.New("frame port owner is unavailable")
	}
	if len(frame.Ports) == 0 {
		return nil
	}
	inputs, err := r.portInputsLocked(
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
	execution *Execution,
	plans []PortPlan,
	existing []PortInput,
) ([]PortInput, error) {
	output := clonePortInputs(existing)
	for _, plan := range plans {
		if plan.Direction != WorkflowPortDirectionInput {
			continue
		}
		if err := plan.Validate(); err != nil {
			return nil, err
		}
		channel := executionChannel(execution, plan.Channel)
		if channel == nil {
			return nil, fmt.Errorf(
				"port input channel %q is unavailable",
				plan.Channel,
			)
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

func (r *Runtime) deliverWorkPortsLocked(
	session *Session,
	execution *Execution,
	item *WorkItem,
) error {
	if session == nil || execution == nil || item == nil ||
		len(item.Ports) == 0 || item.Output == nil {
		return nil
	}

	value, err := portOutputValue(*item)
	if err != nil {
		return err
	}
	for _, plan := range item.Ports {
		if plan.Direction != WorkflowPortDirectionOutput {
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

	for _, plan := range frame.Ports {
		if plan.Direction != WorkflowPortDirectionOutput {
			continue
		}
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

func (r *Runtime) deliverLLMStreamPortsLocked(
	session *Session,
	execution *Execution,
	item *WorkItem,
	value string,
) error {
	if session == nil || execution == nil || item == nil || value == "" {
		return nil
	}
	streamValue := MustCanonicalValue(value)
	for _, plan := range item.Ports {
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
	value CanonicalValue,
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

	maxMessages, maxBytes := portRetention(plan)
	channel, err := mutableChannelLog(
		execution,
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
	if err := r.wakeChannelWaitsLocked(session, execution, message); err != nil {
		return err
	}
	return nil
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

func portOutputValue(item WorkItem) (CanonicalValue, error) {
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
