package sessionstorev3

import (
	"time"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
)

type ListSessionsRequest struct {
	PageSize  int    `json:"pageSize,omitempty"`
	PageToken string `json:"pageToken,omitempty"`
}

type SearchSessionsRequest struct {
	Query     string `json:"query"`
	PageSize  int    `json:"pageSize,omitempty"`
	PageToken string `json:"pageToken,omitempty"`
}

type SessionListItem struct {
	ID                 sessionv3Runtime.SessionID     `json:"id"`
	Revision           sessionv3Runtime.Revision      `json:"revision"`
	Title              string                         `json:"title,omitempty"`
	CreatedAt          time.Time                      `json:"createdAt"`
	UpdatedAt          time.Time                      `json:"updatedAt"`
	ExecutionCount     int                            `json:"executionCount"`
	ActiveExecutionIDs []sessionv3Runtime.ExecutionID `json:"activeExecutionIDs,omitempty"`
}

type SessionPage struct {
	Items         []SessionListItem `json:"items"`
	NextPageToken string            `json:"nextPageToken,omitempty"`
}

type ListExecutionsRequest struct {
	SessionID  sessionv3Runtime.SessionID `json:"sessionID"`
	ActiveOnly bool                       `json:"activeOnly,omitempty"`
	Limit      int                        `json:"limit,omitempty"`
}

type ExecutionListItem struct {
	ID        sessionv3Runtime.ExecutionID       `json:"id"`
	Revision  sessionv3Runtime.Revision          `json:"revision"`
	State     sessionv3Runtime.ExecutionState    `json:"state"`
	CreatedAt time.Time                          `json:"createdAt"`
	UpdatedAt time.Time                          `json:"updatedAt"`
	Outcome   *sessionv3Runtime.ExecutionOutcome `json:"outcome,omitempty"`
	WaitCount int                                `json:"waitCount"`
}

type sessionDocumentKind string

const (
	sessionDocumentKindSession sessionDocumentKind = "session"
	sessionDocumentKindDeleted sessionDocumentKind = "deleted"
)

type sessionDocument struct {
	SchemaVersion string                     `json:"schemaVersion"`
	Kind          sessionDocumentKind        `json:"kind"`
	SessionID     sessionv3Runtime.SessionID `json:"sessionID"`
	Session       *sessionv3Runtime.Session  `json:"session,omitempty"`
	DeletedAt     *time.Time                 `json:"deletedAt,omitempty"`
}

type sessionIndexRecord struct {
	SchemaVersion string          `json:"schemaVersion"`
	Item          SessionListItem `json:"item"`
}
