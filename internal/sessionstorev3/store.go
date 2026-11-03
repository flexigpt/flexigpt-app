package sessionstorev3

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/backgroundutil"
	"github.com/flexigpt/mapstore-go"
	"github.com/flexigpt/mapstore-go/dirpartition"
	"github.com/flexigpt/mapstore-go/ftsengine"
	"github.com/flexigpt/mapstore-go/jsonencdec"
	"github.com/flexigpt/mapstore-go/uuidv7filename"
)

const (
	documentDirectory      = "documents"
	indexDirectory         = "index"
	ftsDirectory           = "fts"
	documentExtension      = ".session.json"
	indexExtension         = ".session-index.json"
	ftsExtension           = ".session-fts.json"
	defaultPageSize        = 24
	maxPageSize            = 256
	maximumDocumentBytes   = 64 << 20
	sessionFTSDatabaseFile = "sessions-v3.fts.sqlite"
	sessionFTSTable        = "sessions_v3"
)

var (
	ErrStoreClosed       = errors.New("session Store v3 is closed")
	ErrSessionNotFound   = errors.New("session Store v3 Session not found")
	ErrExecutionNotFound = errors.New("session Store v3 Execution not found")
	ErrInvalidDocument   = errors.New("invalid Session Store v3 document")
)

type Store struct {
	mu                sync.Mutex
	baseDir           string
	closed            bool
	documents         *mapstore.MapDirectoryStore
	index             *mapstore.MapDirectoryStore
	fts               *ftsengine.Engine
	ftsMu             sync.RWMutex
	maintenanceCancel context.CancelFunc
	maintenanceWG     sync.WaitGroup
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
	engine, err := ftsengine.NewEngine(
		ftsengine.Config{
			BaseDir:    s.baseDir,
			DBFileName: sessionFTSDatabaseFile,
			Table:      sessionFTSTable,
			Columns: []ftsengine.Column{
				{Name: "title", Weight: 1},
				{Name: "user", Weight: 3},
				{Name: "assistant", Weight: 3},
				{Name: "action", Weight: 2},
				{Name: "mtime", Unindexed: true},
			},
		},
		ftsengine.WithLogger(slog.Default()),
	)
	if err != nil {
		return nil, err
	}
	s.fts = engine
	partition := &dirpartition.MonthPartitionProvider{TimeFn: func(key mapstore.FileKey) (time.Time, error) {
		info, err := uuidv7filename.Parse(key.FileName)
		if err != nil {
			return time.Time{}, err
		}
		return info.Time, nil
	}}
	documents, err := mapstore.NewMapDirectoryStore(
		filepath.Join(s.baseDir, documentDirectory),
		true,
		partition,
		jsonencdec.JSONEncoderDecoder{},
		mapstore.WithDirLogger(slog.Default()),
		mapstore.WithDirFileListeners(s.ftsListener()),
	)
	if err != nil {
		_ = engine.Close()
		return nil, err
	}
	s.documents = documents
	indexStore, err := mapstore.NewMapDirectoryStore(
		filepath.Join(s.baseDir, indexDirectory),
		true,
		partition,
		jsonencdec.JSONEncoderDecoder{},
		mapstore.WithDirLogger(slog.Default()),
	)
	if err != nil {
		_ = documents.CloseAll()
		_ = engine.Close()
		return nil, err
	}
	s.index = indexStore
	if err := s.RebuildDerived(context.Background()); err != nil {
		_ = s.Close()
		return nil, err
	}
	maintenanceCtx, cancel := context.WithCancel(context.Background())
	s.maintenanceCancel = cancel
	backgroundutil.Launch(&s.maintenanceWG, "SessionStoreV3 FTS reconciliation", func() {
		s.ftsMu.Lock()
		defer s.ftsMu.Unlock()

		if s.fts == nil {
			return
		}
		if _, err := ftsengine.SyncDirToFTS(
			maintenanceCtx,
			s.fts,
			filepath.Join(s.baseDir, documentDirectory),
			"mtime",
			1000,
			s.syncDocumentToFTS,
		); err != nil {
			slog.Error("sessionstorev3 FTS reconciliation failed", "error", err)
		}
	}, nil)
	return s, nil
}

func (s *Store) Close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	cancel := s.maintenanceCancel
	s.maintenanceCancel = nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	s.maintenanceWG.Wait()
	s.mu.Lock()
	defer s.mu.Unlock()
	var result error
	if s.documents != nil {
		result = errors.Join(result, s.documents.CloseAll())
		s.documents = nil
	}
	if s.index != nil {
		result = errors.Join(result, s.index.CloseAll())
		s.index = nil
	}
	s.ftsMu.Lock()
	if s.fts != nil {
		result = errors.Join(result, s.fts.Close())
		s.fts = nil
	}
	s.ftsMu.Unlock()
	return result
}

func (s *Store) HandleCheckpoint(_ context.Context, view sessionv3.CheckpointView) error {
	if err := view.Session.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidDocument, err)
	}
	if err := s.lock(); err != nil {
		return err
	}
	defer s.mu.Unlock()
	current, found, err := s.readDocument(view.Session.ID, false)
	if err != nil {
		return err
	}
	if found && current.Kind == sessionDocumentKindDeleted {
		return nil
	}

	incoming := cloneSession(view.Session)
	if found && current.Session != nil &&
		(current.Session.Revision > incoming.Revision ||
			(current.Session.Revision == incoming.Revision &&
				!current.Session.UpdatedAt.Before(incoming.UpdatedAt))) {
		incoming = cloneSession(*current.Session)
	}

	if err := s.writeDocument(incoming); err != nil {
		return err
	}
	return s.writeIndex(incoming)
}

func (s *Store) HandleSessionDeleted(_ context.Context, id sessionv3.SessionID) error {
	if err := s.lock(); err != nil {
		return err
	}
	defer s.mu.Unlock()
	if err := validateStoreSessionID(id); err != nil {
		return err
	}
	now := time.Now().UTC()
	doc := sessionDocument{
		SchemaVersion: sessionv3.SchemaVersion,
		Kind:          sessionDocumentKindDeleted,
		SessionID:     id,
		DeletedAt:     &now,
	}
	if err := s.writeRawDocument(id, doc); err != nil {
		return err
	}
	return s.deleteIndex(id)
}

func (s *Store) Restore(ctx context.Context, restorer sessionv3.RuntimeRestorer) error {
	if restorer == nil {
		return errors.New("session Runtime v3 restorer is required")
	}
	if err := s.lock(); err != nil {
		return err
	}
	documents, err := s.allDocuments()
	s.mu.Unlock()
	if err != nil {
		return err
	}
	if retired, ok := restorer.(sessionv3.RetiredSessionRestorer); ok {
		for _, doc := range documents {
			if doc.Kind == sessionDocumentKindDeleted {
				if err := retired.RestoreRetiredSession(ctx, doc.SessionID); err != nil {
					return err
				}
			}
		}
	}
	for _, doc := range documents {
		if doc.Kind == sessionDocumentKindSession && doc.Session != nil {
			if err := restorer.RestoreSession(ctx, *doc.Session); err != nil {
				return fmt.Errorf("restore Session %q: %w", doc.SessionID, err)
			}
		}
	}
	return nil
}

func (s *Store) ReadSession(id sessionv3.SessionID) (sessionv3.Session, error) {
	if err := s.lock(); err != nil {
		return sessionv3.Session{}, err
	}
	defer s.mu.Unlock()
	doc, found, err := s.readDocument(id, false)
	if err != nil {
		return sessionv3.Session{}, err
	}
	if !found || doc.Kind != sessionDocumentKindSession || doc.Session == nil {
		return sessionv3.Session{}, ErrSessionNotFound
	}
	return cloneSession(*doc.Session), nil
}

func (s *Store) ReadConversation(id sessionv3.SessionID) (sessionv3.ConversationProjection, error) {
	session, err := s.ReadSession(id)
	if err != nil {
		return sessionv3.ConversationProjection{}, err
	}
	messages := []sessionv3.Activity{}
	for _, a := range session.Timeline {
		if a.Kind == sessionv3.ActivityKindMessage && a.Visibility != sessionv3.ActivityVisibilityAudit {
			messages = append(messages, a)
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

func (s *Store) ReadActivities(id sessionv3.SessionID) (sessionv3.ActivityProjection, error) {
	session, err := s.ReadSession(id)
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

func (s *Store) ReadWorkflow(id sessionv3.SessionID) (sessionv3.WorkflowProjection, error) {
	session, err := s.ReadSession(id)
	if err != nil {
		return sessionv3.WorkflowProjection{}, err
	}
	activities := []sessionv3.Activity{}
	executions := []sessionv3.Execution{}
	for _, a := range session.Timeline {
		if a.Kind == sessionv3.ActivityKindWorkflow || a.Kind == sessionv3.ActivityKindAction ||
			a.Kind == sessionv3.ActivityKindWait {
			activities = append(activities, a)
		}
	}
	for _, e := range session.Executions {
		for _, f := range e.ProgramFrames {
			if f.Program.InterpreterID == "artifact.workflow.v1" {
				executions = append(executions, e)
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
	sessionID sessionv3.SessionID,
	executionID sessionv3.ExecutionID,
) (sessionv3.Execution, error) {
	session, err := s.ReadSession(sessionID)
	if err != nil {
		return sessionv3.Execution{}, err
	}
	for _, e := range session.Executions {
		if e.ID == executionID {
			return cloneExecution(e), nil
		}
	}
	return sessionv3.Execution{}, ErrExecutionNotFound
}

func (s *Store) ListExecutions(request ListExecutionsRequest) ([]ExecutionListItem, error) {
	session, err := s.ReadSession(request.SessionID)
	if err != nil {
		return nil, err
	}
	items := []ExecutionListItem{}
	for _, e := range session.Executions {
		if request.ActiveOnly && e.State.Terminal() {
			continue
		}
		waitCount := 0
		for _, w := range e.Waits {
			if w.Status == sessionv3.WaitStatusPending || w.Status == sessionv3.WaitStatusResolving {
				waitCount++
			}
		}
		items = append(items, ExecutionListItem{
			ID:        e.ID,
			Revision:  e.Revision,
			State:     e.State,
			CreatedAt: e.CreatedAt,
			UpdatedAt: e.UpdatedAt,
			Outcome:   cloneOutcome(e.Outcome),
			WaitCount: waitCount,
		})
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

func (s *Store) ListSessions(request ListSessionsRequest) (SessionPage, error) {
	if err := s.lock(); err != nil {
		return SessionPage{}, err
	}
	defer s.mu.Unlock()
	entries, next, err := s.index.ListFiles(
		mapstore.ListingConfig{SortOrder: mapstore.SortOrderDescending, PageSize: normalizePageSize(request.PageSize)},
		request.PageToken,
	)
	if err != nil {
		return SessionPage{}, err
	}
	items := []SessionListItem{}
	for _, entry := range entries {
		filename := filepathBase(entry.BaseRelativePath)
		if !strings.HasSuffix(filename, indexExtension) {
			continue
		}
		record, err := s.readIndexFile(filename)
		if err != nil {
			return SessionPage{}, err
		}
		items = append(items, record.Item)
	}
	return SessionPage{Items: items, NextPageToken: next}, nil
}

func (s *Store) SearchSessions(ctx context.Context, request SearchSessionsRequest) (SessionPage, error) {
	if err := s.lock(); err != nil {
		return SessionPage{}, err
	}
	defer s.mu.Unlock()
	if s.fts == nil {
		return SessionPage{}, errors.New("FTS is disabled")
	}

	s.ftsMu.RLock()
	engine := s.fts
	hits, next, err := engine.Search(
		ctx,
		request.Query,
		request.PageToken,
		normalizePageSize(request.PageSize),
	)
	s.ftsMu.RUnlock()
	if err != nil {
		return SessionPage{}, err
	}
	items := []SessionListItem{}
	seen := map[sessionv3.SessionID]struct{}{}
	for _, hit := range hits {
		info, err := uuidv7filename.Parse(filepathBase(hit.ID))
		if err != nil {
			return SessionPage{}, fmt.Errorf("%w: malformed FTS identity", ErrInvalidDocument)
		}
		id := sessionv3.SessionID(info.ID)
		if _, duplicate := seen[id]; duplicate {
			continue
		}
		seen[id] = struct{}{}
		record, found, err := s.readIndex(id)
		if err != nil {
			return SessionPage{}, err
		}
		if found {
			items = append(items, record.Item)
		}
	}
	return SessionPage{Items: items, NextPageToken: next}, nil
}

func (s *Store) ExportSession(id sessionv3.SessionID) ([]byte, error) {
	session, err := s.ReadSession(id)
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

	return s.rebuildDerivedLocked(ctx)
}

func (s *Store) rebuildDerivedLocked(ctx context.Context) error {
	documents, err := s.allDocuments()
	if err != nil {
		return err
	}

	live := make(map[sessionv3.SessionID]struct{}, len(documents))
	for _, doc := range documents {
		if err := ctx.Err(); err != nil {
			return err
		}
		if doc.Kind == sessionDocumentKindSession && doc.Session != nil {
			live[doc.Session.ID] = struct{}{}
			if err := s.writeIndex(*doc.Session); err != nil {
				return err
			}
			continue
		}
		if doc.Kind == sessionDocumentKindDeleted {
			if err := s.deleteIndex(doc.SessionID); err != nil {
				return err
			}
		}
	}

	indexFiles, err := listAllStoreFiles(s.index, indexExtension)
	if err != nil {
		return err
	}
	for _, filename := range indexFiles {
		record, err := s.readIndexFile(filename)
		if err != nil {
			return err
		}
		if _, found := live[record.Item.ID]; found {
			continue
		}
		if err := s.index.DeleteFile(mapstore.FileKey{FileName: filename}); err != nil {
			return err
		}
	}
	return nil
}
