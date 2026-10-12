package sessionstore

import (
	"errors"
	"time"

	sessionRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/session"
)

type SchemaVersion string

type FileExtension string

func (e FileExtension) String() string {
	return string(e)
}

const (
	SessionSchemaVersion SchemaVersion = "v1"
	SessionFileExtension FileExtension = "json"

	DefaultPageSize = 12
	MaxPageSize     = 256
)

var (
	ErrSessionNotFound  = errors.New("session store session not found")
	ErrRunNotFound      = errors.New("session store run not found")
	ErrFTSDisabled      = errors.New("session store full-text search is disabled")
	ErrUnsupportedStore = errors.New("unsupported session store document")
)

type ListSessionsRequest struct {
	PageToken string `json:"pageToken,omitempty"`
	PageSize  int    `json:"pageSize,omitempty"`
}

type SearchSessionsRequest struct {
	Query     string `json:"query"`
	PageToken string `json:"pageToken,omitempty"`
	PageSize  int    `json:"pageSize,omitempty"`
}

type SessionListItem struct {
	ID           sessionRuntime.SessionID `json:"id"`
	Revision     sessionRuntime.Revision  `json:"revision"`
	Title        string                   `json:"title,omitempty"`
	CreatedAt    time.Time                `json:"createdAt"`
	ModifiedAt   time.Time                `json:"modifiedAt"`
	RunCount     int                      `json:"runCount"`
	ActiveRunIDs []sessionRuntime.RunID   `json:"activeRunIDs,omitempty"`
}

type SessionPage struct {
	Items         []SessionListItem `json:"items"`
	NextPageToken string            `json:"nextPageToken,omitempty"`
}

type ConversationProjection struct {
	SchemaVersion    SchemaVersion                      `json:"schemaVersion"`
	SessionID        sessionRuntime.SessionID           `json:"sessionID"`
	SessionRevision  sessionRuntime.Revision            `json:"sessionRevision"`
	Title            string                             `json:"title,omitempty"`
	CreatedAt        time.Time                          `json:"createdAt"`
	ModifiedAt       time.Time                          `json:"modifiedAt"`
	Messages         []sessionRuntime.TranscriptMessage `json:"messages"`
	AssociatedRunIDs []sessionRuntime.RunID             `json:"associatedRunIDs,omitempty"`
}

type ListSessionRunsRequest struct {
	SessionID  sessionRuntime.SessionID `json:"sessionID"`
	ActiveOnly bool                     `json:"activeOnly,omitempty"`
	Limit      int                      `json:"limit,omitempty"`
}

type RunListItem struct {
	ID               sessionRuntime.RunID       `json:"id"`
	Revision         sessionRuntime.Revision    `json:"revision"`
	State            sessionRuntime.RunState    `json:"state"`
	CreatedAt        time.Time                  `json:"createdAt"`
	UpdatedAt        time.Time                  `json:"updatedAt"`
	PendingWaitCount int                        `json:"pendingWaitCount"`
	Outcome          *sessionRuntime.RunOutcome `json:"outcome,omitempty"`
}

func normalizedPageSize(value int) int {
	if value <= 0 {
		return DefaultPageSize
	}
	if value > MaxPageSize {
		return MaxPageSize
	}
	return value
}
