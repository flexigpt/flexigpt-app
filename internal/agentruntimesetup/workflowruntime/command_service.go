package workflowruntime

import (
	"context"
	"errors"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
)

var (
	errWorkflowCommandServiceDependencies = errors.New(
		"workflow command service dependencies are incomplete",
	)
	errWorkflowCommandServiceUnavailable = errors.New(
		"workflow command service is unavailable",
	)
)

// WorkflowCommandRuntime is the narrow Runtime command surface used by
// setup-owned Workflow command preparation. Runtime receives only prepared
// ProgramSnapshot values and remains independent from Artifact and setup
// packages.
type WorkflowCommandRuntime interface {
	FindCommandReceipt(
		ctx context.Context,
		sessionID sessionv3Runtime.SessionID,
		meta sessionv3Runtime.CommandMeta,
		payload any,
	) (*sessionv3Runtime.CommandReceipt, bool, error)
	GetSession(
		ctx context.Context,
		id sessionv3Runtime.SessionID,
	) (sessionv3Runtime.Session, error)
	StartWorkflow(
		ctx context.Context,
		command sessionv3Runtime.StartWorkflowCommand,
	) (*sessionv3Runtime.CommandReceipt, error)
	RegisterWorkflowTriggers(
		ctx context.Context,
		command sessionv3Runtime.RegisterWorkflowTriggersCommand,
	) (*sessionv3Runtime.CommandReceipt, error)
}

// WorkflowCommandService owns setup-time Workflow command preparation.
//
// It performs durable duplicate receipt lookup before it accesses current
// Artifact, Workflow, or setup resources. The Wails wrapper delegates to this
// service and does not own Artifact preparation.
type WorkflowCommandService struct {
	runtime   WorkflowCommandRuntime
	admission *Admission
}

type WorkflowStartRequest struct {
	Meta        sessionv3Runtime.CommandMeta       `json:"meta"`
	SessionID   sessionv3Runtime.SessionID         `json:"sessionID"`
	ExecutionID sessionv3Runtime.ExecutionID       `json:"executionID,omitempty"`
	Workflow    sessionv3Runtime.ArtifactSelection `json:"workflow"`
	Input       jsonutil.CanonicalValue            `json:"input"`
}

type WorkflowTriggerRegistrationRequest struct {
	Meta      sessionv3Runtime.CommandMeta       `json:"meta"`
	SessionID sessionv3Runtime.SessionID         `json:"sessionID"`
	Workflow  sessionv3Runtime.ArtifactSelection `json:"workflow"`
}

func NewWorkflowCommandService(
	runtime WorkflowCommandRuntime,
	admission *Admission,
) (*WorkflowCommandService, error) {
	if runtime == nil || admission == nil {
		return nil, errWorkflowCommandServiceDependencies
	}
	return &WorkflowCommandService{
		runtime:   runtime,
		admission: admission,
	}, nil
}

// StartWorkflow resolves the durable command identity before it prepares the
// Workflow. A duplicate therefore succeeds without requiring current source
// files, Artifact state, or setup-domain resources.
func (s *WorkflowCommandService) StartWorkflow(
	ctx context.Context,
	request WorkflowStartRequest,
) (*sessionv3Runtime.CommandReceipt, error) {
	runtime, admission, err := s.ready()
	if err != nil {
		return nil, err
	}

	identity := sessionv3Runtime.WorkflowStartCommandIdentity{
		Meta:        request.Meta,
		SessionID:   request.SessionID,
		ExecutionID: request.ExecutionID,
		Workflow:    request.Workflow,
		Input:       request.Input.Clone(),
	}
	receipt, found, err := runtime.FindCommandReceipt(
		ctx,
		request.SessionID,
		request.Meta,
		identity,
	)
	if err != nil || found {
		return receipt, err
	}

	session, err := runtime.GetSession(ctx, request.SessionID)
	if err != nil {
		return nil, err
	}
	defaults := session.Defaults
	program, err := admission.PrepareWorkflow(
		ctx,
		PreparationRequest{
			Workflow:           request.Workflow,
			Input:              request.Input.Clone(),
			InvocationDefaults: &defaults,
		},
	)
	if err != nil {
		return &sessionv3Runtime.CommandReceipt{
			CommandID:   request.Meta.ID,
			SessionID:   request.SessionID,
			ExecutionID: request.ExecutionID,
			Message:     err.Error(),
		}, nil
	}

	return runtime.StartWorkflow(
		ctx,
		sessionv3Runtime.StartWorkflowCommand{
			Meta:            request.Meta,
			SessionID:       request.SessionID,
			ExecutionID:     request.ExecutionID,
			Program:         program,
			CommandIdentity: &identity,
		},
	)
}

// RegisterWorkflowTriggers resolves the durable command identity before it
// prepares the input-unbound trigger ProgramSnapshot.
func (s *WorkflowCommandService) RegisterWorkflowTriggers(
	ctx context.Context,
	request WorkflowTriggerRegistrationRequest,
) (*sessionv3Runtime.CommandReceipt, error) {
	runtime, admission, err := s.ready()
	if err != nil {
		return nil, err
	}

	identity := sessionv3Runtime.WorkflowTriggerRegistrationCommandIdentity{
		Meta:      request.Meta,
		SessionID: request.SessionID,
		Workflow:  request.Workflow,
	}
	receipt, found, err := runtime.FindCommandReceipt(
		ctx,
		request.SessionID,
		request.Meta,
		identity,
	)
	if err != nil || found {
		return receipt, err
	}

	session, err := runtime.GetSession(ctx, request.SessionID)
	if err != nil {
		return nil, err
	}
	defaults := session.Defaults
	program, err := admission.PrepareWorkflowTriggers(
		ctx,
		TriggerPreparationRequest{
			Workflow:           request.Workflow,
			InvocationDefaults: &defaults,
		},
	)
	if err != nil {
		return &sessionv3Runtime.CommandReceipt{
			CommandID: request.Meta.ID,
			SessionID: request.SessionID,
			Message:   err.Error(),
		}, nil
	}

	return runtime.RegisterWorkflowTriggers(
		ctx,
		sessionv3Runtime.RegisterWorkflowTriggersCommand{
			Meta:            request.Meta,
			SessionID:       request.SessionID,
			Program:         program,
			CommandIdentity: &identity,
		},
	)
}

func (s *WorkflowCommandService) ready() (
	WorkflowCommandRuntime,
	*Admission,
	error,
) {
	if s == nil || s.runtime == nil || s.admission == nil {
		return nil, nil, errWorkflowCommandServiceUnavailable
	}
	return s.runtime, s.admission, nil
}
