package sessionv3

import (
	"context"
	"errors"
)

type WorkflowPortDirection string

const (
	WorkflowPortDirectionInput  WorkflowPortDirection = "input"
	WorkflowPortDirectionOutput WorkflowPortDirection = "output"
)

type WorkflowPortAccess string

const (
	WorkflowPortAccessRead      WorkflowPortAccess = "read"
	WorkflowPortAccessWrite     WorkflowPortAccess = "write"
	WorkflowPortAccessReadWrite WorkflowPortAccess = "read-write"
)

type WorkflowPort struct {
	Name    string             `json:"name"`
	Channel string             `json:"channel"`
	Access  WorkflowPortAccess `json:"access"`
}

type WorkflowPortBinding struct {
	Name    string             `json:"name"`
	Channel string             `json:"channel"`
	Access  WorkflowPortAccess `json:"access"`
}

type PortPlan struct {
	Name          string                `json:"name"`
	Direction     WorkflowPortDirection `json:"direction"`
	Stream        bool                  `json:"stream,omitempty"`
	Schema        CanonicalValue        `json:"schema"`
	ChannelSchema CanonicalValue        `json:"channelSchema"`
	Channel       string                `json:"channel"`
	Access        WorkflowPortAccess    `json:"access"`
	MaxMessages   int                   `json:"maxMessages,omitempty"`
	MaxBytes      int                   `json:"maxBytes,omitempty"`
	OwnerFrameID  ProgramFrameID        `json:"ownerFrameID,omitempty"`
	ParentFrameID ProgramFrameID        `json:"parentFrameID,omitempty"`
}

type PortInput struct {
	Name     string           `json:"name"`
	Channel  string           `json:"channel"`
	Cursor   uint64           `json:"cursor,omitempty"`
	Values   []CanonicalValue `json:"values,omitempty"`
	Streamed bool             `json:"streamed,omitempty"`
}

type PortStream struct {
	Name         string `json:"name"`
	NextSequence uint64 `json:"nextSequence"`
}

type PortPlanValidator interface {
	ValidatePortPlans(
		ctx context.Context,
		request WorkDispatchRequest,
	) error
}

func (p WorkflowPort) Validate() error {
	if err := validateID("workflow port name", p.Name); err != nil {
		return err
	}
	if err := validateID("workflow port channel", p.Channel); err != nil {
		return err
	}
	switch p.Access {
	case WorkflowPortAccessRead,
		WorkflowPortAccessWrite,
		WorkflowPortAccessReadWrite:
	default:
		return errors.New("workflow port access is invalid")
	}
	return nil
}

func (p WorkflowPortBinding) Validate() error {
	if err := validateID("workflow call port name", p.Name); err != nil {
		return err
	}
	if err := validateID("workflow call port channel", p.Channel); err != nil {
		return err
	}
	switch p.Access {
	case WorkflowPortAccessRead,
		WorkflowPortAccessWrite,
		WorkflowPortAccessReadWrite:
	default:
		return errors.New("workflow call port access is invalid")
	}
	return nil
}

func (p PortPlan) Validate() error {
	if err := validateID("port plan name", p.Name); err != nil {
		return err
	}
	if err := validateID("port plan channel", p.Channel); err != nil {
		return err
	}
	switch p.Direction {
	case WorkflowPortDirectionInput:
		if p.Access != WorkflowPortAccessRead {
			return errors.New("input port plan requires read access")
		}
	case WorkflowPortDirectionOutput:
		if p.Access != WorkflowPortAccessWrite {
			return errors.New("output port plan requires write access")
		}
	default:
		return errors.New("port plan direction is invalid")
	}
	if len(p.Schema) == 0 || !p.Schema.Valid() {
		return errors.New("port plan schema is invalid")
	}
	if len(p.ChannelSchema) == 0 || !p.ChannelSchema.Valid() {
		return errors.New("port plan channel schema is invalid")
	}
	if p.MaxMessages < 0 || p.MaxBytes < 0 {
		return errors.New("port plan retention limit is invalid")
	}
	if p.OwnerFrameID != "" {
		if err := validateID(
			"port plan owner frame ID",
			string(p.OwnerFrameID),
		); err != nil {
			return err
		}
	}
	if p.ParentFrameID != "" {
		if err := validateID(
			"port plan parent frame ID",
			string(p.ParentFrameID),
		); err != nil {
			return err
		}
	}
	return nil
}

func (p PortInput) Validate() error {
	if err := validateID("port input name", p.Name); err != nil {
		return err
	}
	if err := validateID("port input channel", p.Channel); err != nil {
		return err
	}
	for _, value := range p.Values {
		if !value.Valid() {
			return errors.New("port input value is invalid")
		}
	}
	return nil
}

func (p PortStream) Validate() error {
	if err := validateID("port stream name", p.Name); err != nil {
		return err
	}
	if p.NextSequence == 0 {
		return errors.New("port stream sequence is invalid")
	}
	return nil
}

func portAccessForDirection(
	direction WorkflowPortDirection,
) (WorkflowPortAccess, error) {
	switch direction {
	case WorkflowPortDirectionInput:
		return WorkflowPortAccessRead, nil
	case WorkflowPortDirectionOutput:
		return WorkflowPortAccessWrite, nil
	default:
		return "", errors.New("port direction is invalid")
	}
}

func portStreamIndex(
	values []PortStream,
	name string,
) int {
	for index := range values {
		if values[index].Name == name {
			return index
		}
	}
	return -1
}

func clonePortPlans(values []PortPlan) []PortPlan {
	if len(values) == 0 {
		return nil
	}
	output := make([]PortPlan, len(values))
	for index := range values {
		output[index] = values[index]
		output[index].Schema = values[index].Schema.Clone()
		output[index].ChannelSchema = values[index].ChannelSchema.Clone()
	}
	return output
}

func clonePortInputs(values []PortInput) []PortInput {
	if len(values) == 0 {
		return nil
	}
	output := make([]PortInput, len(values))
	for index := range values {
		output[index] = values[index]
		output[index].Values = make(
			[]CanonicalValue,
			len(values[index].Values),
		)
		for valueIndex := range values[index].Values {
			output[index].Values[valueIndex] = values[index].Values[valueIndex].Clone()
		}
	}
	return output
}

func validateUniqueWorkflowPorts(values []WorkflowPort) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if err := value.Validate(); err != nil {
			return err
		}
		if _, found := seen[value.Name]; found {
			return errors.New("workflow port is duplicated")
		}
		seen[value.Name] = struct{}{}
	}
	return nil
}

func validateUniqueWorkflowPortBindings(
	values []WorkflowPortBinding,
) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if err := value.Validate(); err != nil {
			return err
		}
		if _, found := seen[value.Name]; found {
			return errors.New("workflow call port binding is duplicated")
		}
		seen[value.Name] = struct{}{}
	}
	return nil
}

func validateUniquePortPlans(values []PortPlan) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if err := value.Validate(); err != nil {
			return err
		}
		if _, found := seen[value.Name]; found {
			return errors.New("port plan is duplicated")
		}
		seen[value.Name] = struct{}{}
	}
	return nil
}

func validateUniquePortInputs(values []PortInput) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if err := value.Validate(); err != nil {
			return err
		}
		if _, found := seen[value.Name]; found {
			return errors.New("port input is duplicated")
		}
		seen[value.Name] = struct{}{}
	}
	return nil
}

func portSchemaCompatible(
	portSchema CanonicalValue,
	channelSchema CanonicalValue,
) error {
	if len(portSchema) == 0 || len(channelSchema) == 0 {
		return errors.New("port or channel schema is unavailable")
	}
	if err := validateWorkflowSchema(portSchema, MustCanonicalValue(nil)); err == nil {
		return nil
	}
	if err := validateWorkflowSchema(channelSchema, MustCanonicalValue(nil)); err == nil {
		return nil
	}
	return nil
}

func portPlanError(
	code string,
	message string,
) *SanitizedError {
	return &SanitizedError{
		Code:     code,
		Category: ErrorCategoryValidation,
		Message:  message,
	}
}
