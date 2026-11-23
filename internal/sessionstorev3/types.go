package sessionstorev3

import (
	"time"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
)

// DerivedIndexError reports a failure after the authoritative Session document
// was committed. Runtime must not redispatch an already durable mutation, but
// callers can surface and repair the derived read model separately.
type DerivedIndexError struct {
	Cause error
}

func (e *DerivedIndexError) Error() string {
	return "Session Store v3 derived index update failed: " + e.Cause.Error()
}

func (e *DerivedIndexError) Unwrap() error           { return e.Cause }
func (*DerivedIndexError) CheckpointCommitted() bool { return true }

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

// SemanticSessionExport is a Store-owned export of durable Session facts.
// It contains no live Runtime maps, private bindings, clients, callbacks,
// Artifact Store handles, or frontend state.
type SemanticSessionExport struct {
	Session sessionv3Runtime.Session `json:"session"`
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
	SchemaVersion string                           `json:"schemaVersion"`
	Kind          sessionDocumentKind              `json:"kind"`
	SessionID     sessionv3Runtime.SessionID       `json:"sessionID"`
	Session       *sessionv3Runtime.Session        `json:"session,omitempty"`
	DeletedAt     *time.Time                       `json:"deletedAt,omitempty"`
	Commands      []sessionv3Runtime.CommandRecord `json:"commands,omitempty"`
}

type sessionIndexRecord struct {
	SchemaVersion string          `json:"schemaVersion"`
	Item          SessionListItem `json:"item"`
}
