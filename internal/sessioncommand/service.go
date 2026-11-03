// Package sessioncommand is the transport-neutral backend Session API.
package sessioncommand

import (
	"context"
	"errors"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
)

type Runtime interface {
	CreateSession(
		ctx context.Context,
		cmd sessionv3Runtime.CreateSessionCommand,
	) (*sessionv3Runtime.CommandReceipt, error)
	RenameSession(
		ctx context.Context,
		cmd sessionv3Runtime.RenameSessionCommand,
	) (*sessionv3Runtime.CommandReceipt, error)
	UpdateSessionDefaults(
		ctx context.Context,
		cmd sessionv3Runtime.UpdateSessionDefaultsCommand,
	) (*sessionv3Runtime.CommandReceipt, error)
	DeleteSession(
		ctx context.Context,
		cmd sessionv3Runtime.DeleteSessionCommand,
	) (*sessionv3Runtime.CommandReceipt, error)
	SubmitMessage(
		ctx context.Context,
		cmd sessionv3Runtime.SubmitMessageCommand,
	) (*sessionv3Runtime.CommandReceipt, error)
	StartWorkflow(
		ctx context.Context,
		cmd sessionv3Runtime.StartWorkflowCommand,
	) (*sessionv3Runtime.CommandReceipt, error)
	SignalExecution(
		ctx context.Context,
		cmd sessionv3Runtime.SignalExecutionCommand,
	) (*sessionv3Runtime.CommandReceipt, error)
	ExecuteWorkItems(
		ctx context.Context,
		cmd sessionv3Runtime.ExecuteWorkItemsCommand,
	) (*sessionv3Runtime.CommandReceipt, error)
	ContinueExecution(
		ctx context.Context,
		cmd sessionv3Runtime.ContinueExecutionCommand,
	) (*sessionv3Runtime.CommandReceipt, error)
	ResolveWait(ctx context.Context, cmd sessionv3Runtime.ResolveWaitCommand) (*sessionv3Runtime.CommandReceipt, error)
	PauseExecution(
		ctx context.Context,
		cmd sessionv3Runtime.PauseExecutionCommand,
	) (*sessionv3Runtime.CommandReceipt, error)
	ResumeExecution(
		ctx context.Context,
		cmd sessionv3Runtime.ResumeExecutionCommand,
	) (*sessionv3Runtime.CommandReceipt, error)
	CancelExecution(
		ctx context.Context,
		cmd sessionv3Runtime.CancelExecutionCommand,
	) (*sessionv3Runtime.CommandReceipt, error)
	GetSession(ctx context.Context, sessionID sessionv3Runtime.SessionID) (sessionv3Runtime.Session, error)
	Conversation(
		ctx context.Context,
		sessionID sessionv3Runtime.SessionID,
	) (sessionv3Runtime.ConversationProjection, error)
	Activities(ctx context.Context, sessionID sessionv3Runtime.SessionID) (sessionv3Runtime.ActivityProjection, error)
	Workflow(ctx context.Context, sessionID sessionv3Runtime.SessionID) (sessionv3Runtime.WorkflowProjection, error)
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
	c sessionv3Runtime.CreateSessionCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return s.runtime.CreateSession(ctx, c)
}

func (s *Service) RenameSession(
	ctx context.Context,
	c sessionv3Runtime.RenameSessionCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return s.runtime.RenameSession(ctx, c)
}

func (s *Service) UpdateSessionDefaults(
	ctx context.Context,
	c sessionv3Runtime.UpdateSessionDefaultsCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return s.runtime.UpdateSessionDefaults(ctx, c)
}

func (s *Service) DeleteSession(
	ctx context.Context,
	c sessionv3Runtime.DeleteSessionCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return s.runtime.DeleteSession(ctx, c)
}

func (s *Service) SubmitMessage(
	ctx context.Context,
	c sessionv3Runtime.SubmitMessageCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return s.runtime.SubmitMessage(ctx, c)
}

func (s *Service) StartWorkflow(
	ctx context.Context,
	c sessionv3Runtime.StartWorkflowCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return s.runtime.StartWorkflow(ctx, c)
}

func (s *Service) SignalExecution(
	ctx context.Context,
	c sessionv3Runtime.SignalExecutionCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return s.runtime.SignalExecution(ctx, c)
}

func (s *Service) ExecuteWorkItems(
	ctx context.Context,
	c sessionv3Runtime.ExecuteWorkItemsCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return s.runtime.ExecuteWorkItems(ctx, c)
}

func (s *Service) ContinueExecution(
	ctx context.Context,
	c sessionv3Runtime.ContinueExecutionCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return s.runtime.ContinueExecution(ctx, c)
}

func (s *Service) ResolveWait(
	ctx context.Context,
	c sessionv3Runtime.ResolveWaitCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return s.runtime.ResolveWait(ctx, c)
}

func (s *Service) PauseExecution(
	ctx context.Context,
	c sessionv3Runtime.PauseExecutionCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return s.runtime.PauseExecution(ctx, c)
}

func (s *Service) ResumeExecution(
	ctx context.Context,
	c sessionv3Runtime.ResumeExecutionCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return s.runtime.ResumeExecution(ctx, c)
}

func (s *Service) CancelExecution(
	ctx context.Context,
	c sessionv3Runtime.CancelExecutionCommand,
) (*sessionv3Runtime.CommandReceipt, error) {
	return s.runtime.CancelExecution(ctx, c)
}

func (s *Service) GetSession(ctx context.Context, id sessionv3Runtime.SessionID) (sessionv3Runtime.Session, error) {
	return s.runtime.GetSession(ctx, id)
}

func (s *Service) Conversation(
	ctx context.Context,
	id sessionv3Runtime.SessionID,
) (sessionv3Runtime.ConversationProjection, error) {
	return s.runtime.Conversation(ctx, id)
}

func (s *Service) Activities(
	ctx context.Context,
	id sessionv3Runtime.SessionID,
) (sessionv3Runtime.ActivityProjection, error) {
	return s.runtime.Activities(ctx, id)
}

func (s *Service) Workflow(
	ctx context.Context,
	id sessionv3Runtime.SessionID,
) (sessionv3Runtime.WorkflowProjection, error) {
	return s.runtime.Workflow(ctx, id)
}

func (s *Service) Close() {
	if s != nil && s.runtime != nil {
		s.runtime.Close()
		s.runtime = nil
	}
}
