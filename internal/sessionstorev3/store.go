package sessionstorev3

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
)

const (
	documentDirectory    = "documents"
	indexDirectory       = "index"
	ftsDirectory         = "fts"
	documentExtension    = ".session.json"
	indexExtension       = ".session-index.json"
	ftsExtension         = ".session-fts.json"
	defaultPageSize      = 24
	maxPageSize          = 256
	maximumDocumentBytes = 64 << 20
)

var (
	ErrStoreClosed       = errors.New("session Store v3 is closed")
	ErrSessionNotFound   = errors.New("session Store v3 Session not found")
	ErrExecutionNotFound = errors.New("session Store v3 Execution not found")
	ErrInvalidDocument   = errors.New("invalid Session Store v3 document")
)

type Store struct {
	mu      sync.Mutex
	baseDir string
	closed  bool
}

func New(baseDir string) (*Store, error) {
	if strings.TrimSpace(baseDir) == "" {
		return nil, errors.New("session Store v3 directory is required")
	}
	s := &Store{baseDir: filepath.Clean(baseDir)}
	for _, dir := range []string{documentDirectory, indexDirectory, ftsDirectory} {
		if err := os.MkdirAll(filepath.Join(s.baseDir, dir), 0o700); err != nil {
			return nil, err
		}
	}
	if err := s.RebuildDerived(context.Background()); err != nil {
		return nil, errors.Join(err, s.Close())
	}
	return s, nil
}

func (s *Store) Close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
	return nil
}

// HandleCheckpoint is synchronous. The Session Runtime calls it before an
// external dispatch. An error prevents the dispatch from starting.
func (s *Store) HandleCheckpoint(ctx context.Context, view sessionv3.CheckpointView) error {
	if err := view.Session.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidDocument, err)
	}
	if err := validateSessionID(view.Session.ID); err != nil {
		return err
	}
	if err := s.lock(); err != nil {
		return err
	}
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	current, found, err := s.readDocumentLocked(view.Session.ID)
	if err != nil {
		return err
	}
	if found && current.Kind == sessionDocumentKindDeleted {
		return nil
	}
	incoming := cloneSession(view.Session)
	if found && current.Session != nil &&
		(current.Session.Revision > incoming.Revision || (current.Session.Revision == incoming.Revision && current.Session.UpdatedAt.After(incoming.UpdatedAt))) {
		incoming = cloneSession(*current.Session)
	}
	if err := s.writeDocumentLocked(incoming); err != nil {
		return err
	}
	if err := s.writeIndexLocked(incoming); err != nil {
		return err
	}
	return s.writeFTSLocked(incoming)
}

func (s *Store) HandleSessionDeleted(ctx context.Context, id sessionv3.SessionID) error {
	if err := validateSessionID(id); err != nil {
		return err
	}
	if err := s.lock(); err != nil {
		return err
	}
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	now := time.Now().UTC()
	doc := sessionDocument{
		SchemaVersion: sessionv3.SchemaVersion,
		Kind:          sessionDocumentKindDeleted,
		SessionID:     id,
		DeletedAt:     &now,
	}
	if err := writeAtomicJSON(s.documentPath(id), doc); err != nil {
		return err
	}
	if err := removeIfExists(s.indexPath(id)); err != nil {
		return err
	}
	return removeIfExists(s.ftsPath(id))
}

func (s *Store) Restore(ctx context.Context, restorer sessionv3.RuntimeRestorer) error {
	if restorer == nil {
		return errors.New("session Runtime v3 restorer is required")
	}
	if err := s.lock(); err != nil {
		return err
	}
	paths, err := filepath.Glob(filepath.Join(s.baseDir, documentDirectory, "*"+documentExtension))
	s.mu.Unlock()
	if err != nil {
		return err
	}
	sort.Strings(paths)
	if retired, ok := restorer.(sessionv3.RetiredSessionRestorer); ok {
		for _, path := range paths {
			if err := ctx.Err(); err != nil {
				return err
			}
			doc, err := readDocument(path)
			if err != nil {
				return err
			}
			if doc.Kind == sessionDocumentKindDeleted {
				if err := retired.RestoreRetiredSession(ctx, doc.SessionID); err != nil {
					return err
				}
			}
		}
	}
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return err
		}
		doc, err := readDocument(path)
		if err != nil {
			return err
		}
		if doc.Kind == sessionDocumentKindSession && doc.Session != nil {
			if err := restorer.RestoreSession(ctx, *doc.Session); err != nil {
				return fmt.Errorf("restore Session %q: %w", doc.SessionID, err)
			}
		}
	}
	return nil
}

func (s *Store) ReadSession(ctx context.Context, id sessionv3.SessionID) (sessionv3.Session, error) {
	if err := validateSessionID(id); err != nil {
		return sessionv3.Session{}, err
	}
	if err := s.lock(); err != nil {
		return sessionv3.Session{}, err
	}
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return sessionv3.Session{}, err
	}
	doc, found, err := s.readDocumentLocked(id)
	if err != nil {
		return sessionv3.Session{}, err
	}
	if !found || doc.Kind != sessionDocumentKindSession || doc.Session == nil {
		return sessionv3.Session{}, ErrSessionNotFound
	}
	return cloneSession(*doc.Session), nil
}

func (s *Store) ReadConversation(
	ctx context.Context,
	id sessionv3.SessionID,
) (sessionv3.ConversationProjection, error) {
	session, err := s.ReadSession(ctx, id)
	if err != nil {
		return sessionv3.ConversationProjection{}, err
	}
	messages := []sessionv3.Activity{}
	for _, activity := range session.Timeline {
		if activity.Kind == sessionv3.ActivityKindMessage && activity.Visibility != sessionv3.ActivityVisibilityAudit {
			messages = append(messages, activity)
		}
	}
	return sessionv3.ConversationProjection{
		SchemaVersion:   sessionv3.SchemaVersion,
		SessionID:       session.ID,
		SessionRevision: session.Revision,
		Title:           session.Title,
		Messages:        messages,
	}, nil
}

func (s *Store) ReadActivities(ctx context.Context, id sessionv3.SessionID) (sessionv3.ActivityProjection, error) {
	session, err := s.ReadSession(ctx, id)
	if err != nil {
		return sessionv3.ActivityProjection{}, err
	}
	return sessionv3.ActivityProjection{
		SchemaVersion:   sessionv3.SchemaVersion,
		SessionID:       session.ID,
		SessionRevision: session.Revision,
		Activities:      append([]sessionv3.Activity(nil), session.Timeline...),
	}, nil
}

func (s *Store) ReadWorkflow(ctx context.Context, id sessionv3.SessionID) (sessionv3.WorkflowProjection, error) {
	session, err := s.ReadSession(ctx, id)
	if err != nil {
		return sessionv3.WorkflowProjection{}, err
	}
	activities := []sessionv3.Activity{}
	executions := []sessionv3.Execution{}
	for _, activity := range session.Timeline {
		if activity.Kind == sessionv3.ActivityKindWorkflow || activity.Kind == sessionv3.ActivityKindAction ||
			activity.Kind == sessionv3.ActivityKindWait {
			activities = append(activities, activity)
		}
	}
	for _, execution := range session.Executions {
		for _, frame := range execution.ProgramFrames {
			if frame.Program.InterpreterID == "artifact.workflow.v1" {
				executions = append(executions, execution)
				break
			}
		}
	}
	return sessionv3.WorkflowProjection{
		SchemaVersion:   sessionv3.SchemaVersion,
		SessionID:       session.ID,
		SessionRevision: session.Revision,
		Activities:      activities,
		Executions:      executions,
	}, nil
}

func (s *Store) ReadExecution(
	ctx context.Context,
	sessionID sessionv3.SessionID,
	executionID sessionv3.ExecutionID,
) (sessionv3.Execution, error) {
	session, err := s.ReadSession(ctx, sessionID)
	if err != nil {
		return sessionv3.Execution{}, err
	}
	for _, execution := range session.Executions {
		if execution.ID == executionID {
			return cloneExecution(execution), nil
		}
	}
	return sessionv3.Execution{}, ErrExecutionNotFound
}

func (s *Store) ListExecutions(ctx context.Context, request ListExecutionsRequest) ([]ExecutionListItem, error) {
	session, err := s.ReadSession(ctx, request.SessionID)
	if err != nil {
		return nil, err
	}
	items := []ExecutionListItem{}
	for _, execution := range session.Executions {
		if request.ActiveOnly && execution.State.Terminal() {
			continue
		}
		waitCount := 0
		for _, wait := range execution.Waits {
			if wait.Status == sessionv3.WaitStatusPending || wait.Status == sessionv3.WaitStatusResolving {
				waitCount++
			}
		}
		items = append(
			items,
			ExecutionListItem{
				ID:        execution.ID,
				Revision:  execution.Revision,
				State:     execution.State,
				CreatedAt: execution.CreatedAt,
				UpdatedAt: execution.UpdatedAt,
				Outcome:   cloneOutcome(execution.Outcome),
				WaitCount: waitCount,
			},
		)
	}
	sort.Slice(items, func(i, j int) bool {
		if !items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].UpdatedAt.After(items[j].UpdatedAt)
		}
		return items[i].ID > items[j].ID
	})
	if request.Limit > 0 && len(items) > request.Limit {
		items = items[:request.Limit]
	}
	return items, nil
}

func (s *Store) ListSessions(ctx context.Context, request ListSessionsRequest) (SessionPage, error) {
	if err := s.lock(); err != nil {
		return SessionPage{}, err
	}
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return SessionPage{}, err
	}
	paths, err := filepath.Glob(filepath.Join(s.baseDir, indexDirectory, "*"+indexExtension))
	if err != nil {
		return SessionPage{}, err
	}
	items := []SessionListItem{}
	for _, path := range paths {
		var record sessionIndexRecord
		if err := readJSON(path, &record); err != nil {
			return SessionPage{}, err
		}
		if record.SchemaVersion != sessionv3.SchemaVersion {
			return SessionPage{}, fmt.Errorf("%w: unsupported index schema", ErrInvalidDocument)
		}
		items = append(items, record.Item)
	}
	sort.Slice(items, func(i, j int) bool {
		if !items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].UpdatedAt.After(items[j].UpdatedAt)
		}
		return items[i].ID < items[j].ID
	})
	start, err := pageStart(items, request.PageToken)
	if err != nil {
		return SessionPage{}, err
	}
	end := min(start+normalizePageSize(request.PageSize), len(items))
	page := SessionPage{Items: append([]SessionListItem(nil), items[start:end]...)}
	if end < len(items) {
		page.NextPageToken = string(items[end-1].ID)
	}
	return page, nil
}

func (s *Store) SearchSessions(ctx context.Context, request SearchSessionsRequest) (SessionPage, error) {
	query := tokenize(request.Query)
	if len(query) == 0 {
		return SessionPage{Items: []SessionListItem{}}, nil
	}
	if err := s.lock(); err != nil {
		return SessionPage{}, err
	}
	defer s.mu.Unlock()
	paths, err := filepath.Glob(filepath.Join(s.baseDir, ftsDirectory, "*"+ftsExtension))
	if err != nil {
		return SessionPage{}, err
	}
	type hit struct {
		item  SessionListItem
		score int
	}
	hits := []hit{}
	for _, path := range paths {
		var fts sessionFTSRecord
		if err := readJSON(path, &fts); err != nil {
			return SessionPage{}, err
		}
		score := termScore(fts.Terms, query)
		if score == 0 {
			continue
		}
		var index sessionIndexRecord
		if err := readJSON(s.indexPath(fts.SessionID), &index); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return SessionPage{}, err
		}
		hits = append(hits, hit{item: index.Item, score: score})
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].score != hits[j].score {
			return hits[i].score > hits[j].score
		}
		return hits[i].item.UpdatedAt.After(hits[j].item.UpdatedAt)
	})
	items := make([]SessionListItem, 0, len(hits))
	for _, value := range hits {
		items = append(items, value.item)
	}
	start, err := pageStart(items, request.PageToken)
	if err != nil {
		return SessionPage{}, err
	}
	end := min(start+normalizePageSize(request.PageSize), len(items))
	page := SessionPage{Items: append([]SessionListItem(nil), items[start:end]...)}
	if end < len(items) {
		page.NextPageToken = string(items[end-1].ID)
	}
	return page, nil
}

func (s *Store) ExportSession(ctx context.Context, id sessionv3.SessionID) ([]byte, error) {
	session, err := s.ReadSession(ctx, id)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(session, "", "  ")
}

func (s *Store) RebuildDerived(ctx context.Context) error {
	if err := s.lock(); err != nil {
		return err
	}
	defer s.mu.Unlock()
	if err := clearDirectory(filepath.Join(s.baseDir, indexDirectory)); err != nil {
		return err
	}
	if err := clearDirectory(filepath.Join(s.baseDir, ftsDirectory)); err != nil {
		return err
	}
	paths, err := filepath.Glob(filepath.Join(s.baseDir, documentDirectory, "*"+documentExtension))
	if err != nil {
		return err
	}
	sort.Strings(paths)
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return err
		}
		doc, err := readDocument(path)
		if err != nil {
			return err
		}
		if doc.Kind == sessionDocumentKindSession && doc.Session != nil {
			if err := s.writeIndexLocked(*doc.Session); err != nil {
				return err
			}
			if err := s.writeFTSLocked(*doc.Session); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Store) lock() error {
	if s == nil {
		return ErrStoreClosed
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return ErrStoreClosed
	}
	return nil
}

func (s *Store) readDocumentLocked(id sessionv3.SessionID) (sessionDocument, bool, error) {
	doc, err := readDocument(s.documentPath(id))
	if errors.Is(err, os.ErrNotExist) {
		return sessionDocument{}, false, nil
	}
	return doc, err == nil, err
}

func (s *Store) writeDocumentLocked(session sessionv3.Session) error {
	copyValue := cloneSession(session)
	return writeAtomicJSON(
		s.documentPath(session.ID),
		sessionDocument{
			SchemaVersion: sessionv3.SchemaVersion,
			Kind:          sessionDocumentKindSession,
			SessionID:     session.ID,
			Session:       &copyValue,
		},
	)
}

func (s *Store) writeIndexLocked(session sessionv3.Session) error {
	active := []sessionv3.ExecutionID{}
	for _, execution := range session.Executions {
		if !execution.State.Terminal() {
			active = append(active, execution.ID)
		}
	}
	return writeAtomicJSON(
		s.indexPath(session.ID),
		sessionIndexRecord{
			SchemaVersion: sessionv3.SchemaVersion,
			Item: SessionListItem{
				ID:                 session.ID,
				Revision:           session.Revision,
				Title:              session.Title,
				CreatedAt:          session.CreatedAt,
				UpdatedAt:          session.UpdatedAt,
				ExecutionCount:     len(session.Executions),
				ActiveExecutionIDs: active,
			},
		},
	)
}

func (s *Store) writeFTSLocked(session sessionv3.Session) error {
	var text strings.Builder
	text.WriteString(session.Title)
	text.WriteByte('\n')
	for _, activity := range session.Timeline {
		switch activity.Kind {
		case sessionv3.ActivityKindMessage:
			if activity.Message != nil {
				text.WriteString(activity.Message.Text)
				text.WriteByte('\n')
			}
		case sessionv3.ActivityKindAction:
			if activity.Action != nil {
				text.WriteString(activity.Action.Title)
				text.WriteByte('\n')
				text.WriteString(activity.Action.Summary)
				text.WriteByte('\n')
			}
		case sessionv3.ActivityKindWorkflow:
			if activity.Workflow != nil {
				text.WriteString(activity.Workflow.Phase)
				text.WriteByte('\n')
				text.WriteString(activity.Workflow.Summary)
				text.WriteByte('\n')
			}
		case sessionv3.ActivityKindWait:
			if activity.Wait != nil {
				text.WriteString(activity.Wait.Summary)
				text.WriteByte('\n')
			}
		}
	}
	return writeAtomicJSON(
		s.ftsPath(session.ID),
		sessionFTSRecord{
			SchemaVersion: sessionv3.SchemaVersion,
			SessionID:     session.ID,
			Revision:      session.Revision,
			UpdatedAt:     session.UpdatedAt,
			Title:         session.Title,
			Text:          text.String(),
			Terms:         tokenize(text.String()),
		},
	)
}

func (s *Store) documentPath(id sessionv3.SessionID) string {
	return filepath.Join(s.baseDir, documentDirectory, string(id)+documentExtension)
}

func (s *Store) indexPath(id sessionv3.SessionID) string {
	return filepath.Join(s.baseDir, indexDirectory, string(id)+indexExtension)
}

func (s *Store) ftsPath(id sessionv3.SessionID) string {
	return filepath.Join(s.baseDir, ftsDirectory, string(id)+ftsExtension)
}

func readDocument(path string) (sessionDocument, error) {
	var doc sessionDocument
	if err := readJSON(path, &doc); err != nil {
		return sessionDocument{}, err
	}
	if doc.SchemaVersion != sessionv3.SchemaVersion {
		return sessionDocument{}, fmt.Errorf("%w: unsupported schema", ErrInvalidDocument)
	}
	if err := validateSessionID(doc.SessionID); err != nil {
		return sessionDocument{}, err
	}
	switch doc.Kind {
	case sessionDocumentKindDeleted:
		if doc.Session != nil || doc.DeletedAt == nil || doc.DeletedAt.IsZero() {
			return sessionDocument{}, fmt.Errorf("%w: malformed deletion", ErrInvalidDocument)
		}
	case sessionDocumentKindSession:
		if doc.Session == nil || doc.DeletedAt != nil || doc.Session.ID != doc.SessionID {
			return sessionDocument{}, fmt.Errorf("%w: malformed Session document", ErrInvalidDocument)
		}
		if err := doc.Session.Validate(); err != nil {
			return sessionDocument{}, fmt.Errorf("%w: %w", ErrInvalidDocument, err)
		}
	default:
		return sessionDocument{}, fmt.Errorf("%w: unsupported document kind", ErrInvalidDocument)
	}
	return doc, nil
}

func readJSON(path string, target any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(raw) > maximumDocumentBytes {
		return fmt.Errorf("%w: document exceeds size limit", ErrInvalidDocument)
	}
	return json.Unmarshal(raw, target)
}

func writeAtomicJSON(path string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if len(raw) > maximumDocumentBytes {
		return fmt.Errorf("%w: document exceeds size limit", ErrInvalidDocument)
	}
	temporary := path + ".tmp"
	file, err := os.OpenFile(temporary, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(raw); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporary, path); err != nil {
		return err
	}
	if dir, err := os.Open(filepath.Dir(path)); err == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}
	return nil
}

func removeIfExists(path string) error {
	err := os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func clearDirectory(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(path, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func validateSessionID(id sessionv3.SessionID) error {
	if err := uuidutil.ValidateUUIDv7(string(id)); err != nil {
		return fmt.Errorf("%w: Session ID: %w", ErrInvalidDocument, err)
	}
	return nil
}

func normalizePageSize(value int) int {
	if value <= 0 {
		return defaultPageSize
	}
	return min(value, maxPageSize)
}

func pageStart(items []SessionListItem, token string) (int, error) {
	if token == "" {
		return 0, nil
	}
	for index, item := range items {
		if string(item.ID) == token {
			return index + 1, nil
		}
	}
	return 0, fmt.Errorf("%w: invalid page token", ErrInvalidDocument)
}

func tokenize(value string) []string {
	seen := map[string]struct{}{}
	terms := []string{}
	var current strings.Builder
	flush := func() {
		if current.Len() == 0 {
			return
		}
		term := strings.ToLower(current.String())
		current.Reset()
		if len(term) < 2 {
			return
		}
		if _, found := seen[term]; found {
			return
		}
		seen[term] = struct{}{}
		terms = append(terms, term)
	}
	for _, character := range value {
		if unicode.IsLetter(character) || unicode.IsDigit(character) {
			current.WriteRune(unicode.ToLower(character))
		} else {
			flush()
		}
	}
	flush()
	sort.Strings(terms)
	return terms
}

func termScore(record, query []string) int {
	available := map[string]struct{}{}
	for _, term := range record {
		available[term] = struct{}{}
	}
	for _, term := range query {
		if _, found := available[term]; !found {
			return 0
		}
	}
	return len(query)
}

func cloneSession(value sessionv3.Session) sessionv3.Session {
	raw, _ := json.Marshal(value)
	var output sessionv3.Session
	_ = json.Unmarshal(raw, &output)
	return output
}

func cloneExecution(value sessionv3.Execution) sessionv3.Execution {
	raw, _ := json.Marshal(value)
	var output sessionv3.Execution
	_ = json.Unmarshal(raw, &output)
	return output
}

func cloneOutcome(value *sessionv3.ExecutionOutcome) *sessionv3.ExecutionOutcome {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}
