package sessionstore

import (
	"errors"
	"fmt"
	"time"

	sessionRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/session"
)

type (
	SchemaVersion string
	FileExtension string
)

func (extension FileExtension) String() string {
	return string(extension)
}

const (
	SessionSchemaVersion SchemaVersion = "session-runtime/v2"
	SessionFileExtension FileExtension = "session"

	DefaultPageSize = 12
	MaxPageSize     = 256

	maximumDocumentBytes      = 32 << 20
	maximumJSONValueBytes     = 2 << 20
	maximumTranscriptMessages = 100_000
	maximumAssociatedRuns     = 10_000
	maximumAttachmentDepth    = 8

	indexFileExtension = "sessionindex"
	documentDirectory  = "documents"
	indexDirectory     = "index"
)

var (
	ErrSessionNotFound  = errors.New("session store session not found")
	ErrRunNotFound      = errors.New("session store run not found")
	ErrFTSDisabled      = errors.New("session store full-text search is disabled")
	ErrUnsupportedStore = errors.New("unsupported session store document")
	ErrInvalidDocument  = errors.New("invalid session store document")
	ErrStoreClosed      = errors.New("session store is closed")
	ErrSessionRetired   = errors.New("session ID was durably deleted")
)

type PersistenceError struct {
	SessionID sessionRuntime.SessionID
	Operation string
	Cause     error
}

func (failure PersistenceError) Error() string {
	return fmt.Sprintf(
		"session persistence %s failed for %q: %v",
		failure.Operation,
		failure.SessionID,
		failure.Cause,
	)
}

func (failure PersistenceError) Unwrap() error {
	return failure.Cause
}

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

type documentKind string

const (
	documentKindSession documentKind = "session"
	documentKindDeleted documentKind = "deleted"
)

type deletionRecord struct {
	SessionID sessionRuntime.SessionID `json:"sessionID"`
	DeletedAt time.Time                `json:"deletedAt"`
}

// storedSession is a new persistence envelope. A deletion record is a durable
// write barrier, not an executable Session or an additional runtime state root.
type storedSession struct {
	SchemaVersion SchemaVersion           `json:"schemaVersion"`
	Kind          documentKind            `json:"kind"`
	Session       *sessionRuntime.Session `json:"session,omitempty"`
	Runs          []sessionRuntime.Run    `json:"runs,omitempty"`
	Deletion      *deletionRecord         `json:"deletion,omitempty"`
}

type sessionIndexRecord struct {
	SchemaVersion SchemaVersion   `json:"schemaVersion"`
	Item          SessionListItem `json:"item"`
}

func normalizedPageSize(value int) int {
	if value <= 0 {
		return DefaultPageSize
	}
	return min(value, MaxPageSize)
}
