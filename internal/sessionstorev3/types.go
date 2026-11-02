package sessionstorev3

import (
	"time"

	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
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
	ID                 sessionv3.SessionID     `json:"id"`
	Revision           sessionv3.Revision      `json:"revision"`
	Title              string                  `json:"title,omitempty"`
	CreatedAt          time.Time               `json:"createdAt"`
	UpdatedAt          time.Time               `json:"updatedAt"`
	ExecutionCount     int                     `json:"executionCount"`
	ActiveExecutionIDs []sessionv3.ExecutionID `json:"activeExecutionIDs,omitempty"`
}

type SessionPage struct {
	Items         []SessionListItem `json:"items"`
	NextPageToken string            `json:"nextPageToken,omitempty"`
}

type ListExecutionsRequest struct {
	SessionID  sessionv3.SessionID `json:"sessionID"`
	ActiveOnly bool                `json:"activeOnly,omitempty"`
	Limit      int                 `json:"limit,omitempty"`
}

type ExecutionListItem struct {
	ID        sessionv3.ExecutionID       `json:"id"`
	Revision  sessionv3.Revision          `json:"revision"`
	State     sessionv3.ExecutionState    `json:"state"`
	CreatedAt time.Time                   `json:"createdAt"`
	UpdatedAt time.Time                   `json:"updatedAt"`
	Outcome   *sessionv3.ExecutionOutcome `json:"outcome,omitempty"`
	WaitCount int                         `json:"waitCount"`
}

type sessionDocumentKind string

const (
	sessionDocumentKindSession sessionDocumentKind = "session"
	sessionDocumentKindDeleted sessionDocumentKind = "deleted"
)

type sessionDocument struct {
	SchemaVersion string              `json:"schemaVersion"`
	Kind          sessionDocumentKind `json:"kind"`
	SessionID     sessionv3.SessionID `json:"sessionID"`
	Session       *sessionv3.Session  `json:"session,omitempty"`
	DeletedAt     *time.Time          `json:"deletedAt,omitempty"`
}

type sessionIndexRecord struct {
	SchemaVersion string          `json:"schemaVersion"`
	Item          SessionListItem `json:"item"`
}

type sessionFTSRecord struct {
	SchemaVersion string              `json:"schemaVersion"`
	SessionID     sessionv3.SessionID `json:"sessionID"`
	Revision      sessionv3.Revision  `json:"revision"`
	UpdatedAt     time.Time           `json:"updatedAt"`
	Title         string              `json:"title"`
	Text          string              `json:"text"`
	Terms         []string            `json:"terms"`
}
