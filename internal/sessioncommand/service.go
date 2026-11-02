// Package sessioncommand is the transport-neutral backend Session API.
package sessioncommand

import (
	"context"
	"errors"

	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
)

type Runtime interface {
	CreateSession(ctx context.Context, cmd sessionv3.CreateSessionCommand) (*sessionv3.CommandReceipt, error)
	RenameSession(ctx context.Context, cmd sessionv3.RenameSessionCommand) (*sessionv3.CommandReceipt, error)
	UpdateSessionDefaults(
		ctx context.Context,
		cmd sessionv3.UpdateSessionDefaultsCommand,
	) (*sessionv3.CommandReceipt, error)
	DeleteSession(ctx context.Context, cmd sessionv3.DeleteSessionCommand) (*sessionv3.CommandReceipt, error)
	SubmitMessage(ctx context.Context, cmd sessionv3.SubmitMessageCommand) (*sessionv3.CommandReceipt, error)
	StartWorkflow(ctx context.Context, cmd sessionv3.StartWorkflowCommand) (*sessionv3.CommandReceipt, error)
	SignalExecution(ctx context.Context, cmd sessionv3.SignalExecutionCommand) (*sessionv3.CommandReceipt, error)
	ExecuteWorkItems(ctx context.Context, cmd sessionv3.ExecuteWorkItemsCommand) (*sessionv3.CommandReceipt, error)
	ContinueExecution(ctx context.Context, cmd sessionv3.ContinueExecutionCommand) (*sessionv3.CommandReceipt, error)
	ResolveWait(ctx context.Context, cmd sessionv3.ResolveWaitCommand) (*sessionv3.CommandReceipt, error)
	PauseExecution(ctx context.Context, cmd sessionv3.PauseExecutionCommand) (*sessionv3.CommandReceipt, error)
	ResumeExecution(ctx context.Context, cmd sessionv3.ResumeExecutionCommand) (*sessionv3.CommandReceipt, error)
	CancelExecution(ctx context.Context, cmd sessionv3.CancelExecutionCommand) (*sessionv3.CommandReceipt, error)
	GetSession(ctx context.Context, sessionID sessionv3.SessionID) (sessionv3.Session, error)
	Conversation(ctx context.Context, sessionID sessionv3.SessionID) (sessionv3.ConversationProjection, error)
	Activities(ctx context.Context, sessionID sessionv3.SessionID) (sessionv3.ActivityProjection, error)
	Workflow(ctx context.Context, sessionID sessionv3.SessionID) (sessionv3.WorkflowProjection, error)
	Close()
}

type Service struct{ runtime Runtime }

func New(runtime Runtime) (*Service, error) {
	if runtime == nil {
		return nil, errors.New("session command runtime is required")
	}
	return &Service{runtime: runtime}, nil
}

func (s *Service) CreateSession(
	ctx context.Context,
	c sessionv3.CreateSessionCommand,
) (*sessionv3.CommandReceipt, error) {
	return s.runtime.CreateSession(ctx, c)
}

func (s *Service) RenameSession(
	ctx context.Context,
	c sessionv3.RenameSessionCommand,
) (*sessionv3.CommandReceipt, error) {
	return s.runtime.RenameSession(ctx, c)
}

func (s *Service) UpdateSessionDefaults(
	ctx context.Context,
	c sessionv3.UpdateSessionDefaultsCommand,
) (*sessionv3.CommandReceipt, error) {
	return s.runtime.UpdateSessionDefaults(ctx, c)
}

func (s *Service) DeleteSession(
	ctx context.Context,
	c sessionv3.DeleteSessionCommand,
) (*sessionv3.CommandReceipt, error) {
	return s.runtime.DeleteSession(ctx, c)
}

func (s *Service) SubmitMessage(
	ctx context.Context,
	c sessionv3.SubmitMessageCommand,
) (*sessionv3.CommandReceipt, error) {
	return s.runtime.SubmitMessage(ctx, c)
}

func (s *Service) StartWorkflow(
	ctx context.Context,
	c sessionv3.StartWorkflowCommand,
) (*sessionv3.CommandReceipt, error) {
	return s.runtime.StartWorkflow(ctx, c)
}

func (s *Service) SignalExecution(
	ctx context.Context,
	c sessionv3.SignalExecutionCommand,
) (*sessionv3.CommandReceipt, error) {
	return s.runtime.SignalExecution(ctx, c)
}

func (s *Service) ExecuteWorkItems(
	ctx context.Context,
	c sessionv3.ExecuteWorkItemsCommand,
) (*sessionv3.CommandReceipt, error) {
	return s.runtime.ExecuteWorkItems(ctx, c)
}

func (s *Service) ContinueExecution(
	ctx context.Context,
	c sessionv3.ContinueExecutionCommand,
) (*sessionv3.CommandReceipt, error) {
	return s.runtime.ContinueExecution(ctx, c)
}

func (s *Service) ResolveWait(ctx context.Context, c sessionv3.ResolveWaitCommand) (*sessionv3.CommandReceipt, error) {
	return s.runtime.ResolveWait(ctx, c)
}

func (s *Service) PauseExecution(
	ctx context.Context,
	c sessionv3.PauseExecutionCommand,
) (*sessionv3.CommandReceipt, error) {
	return s.runtime.PauseExecution(ctx, c)
}

func (s *Service) ResumeExecution(
	ctx context.Context,
	c sessionv3.ResumeExecutionCommand,
) (*sessionv3.CommandReceipt, error) {
	return s.runtime.ResumeExecution(ctx, c)
}

func (s *Service) CancelExecution(
	ctx context.Context,
	c sessionv3.CancelExecutionCommand,
) (*sessionv3.CommandReceipt, error) {
	return s.runtime.CancelExecution(ctx, c)
}

func (s *Service) GetSession(ctx context.Context, id sessionv3.SessionID) (sessionv3.Session, error) {
	return s.runtime.GetSession(ctx, id)
}

func (s *Service) Conversation(ctx context.Context, id sessionv3.SessionID) (sessionv3.ConversationProjection, error) {
	return s.runtime.Conversation(ctx, id)
}

func (s *Service) Activities(ctx context.Context, id sessionv3.SessionID) (sessionv3.ActivityProjection, error) {
	return s.runtime.Activities(ctx, id)
}

func (s *Service) Workflow(ctx context.Context, id sessionv3.SessionID) (sessionv3.WorkflowProjection, error) {
	return s.runtime.Workflow(ctx, id)
}

func (s *Service) Close() {
	if s != nil && s.runtime != nil {
		s.runtime.Close()
		s.runtime = nil
	}
}
