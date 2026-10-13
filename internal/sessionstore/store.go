package sessionstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	sessionRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/session"
	"github.com/flexigpt/flexigpt-app/internal/backgroundutil"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/mapstore-go"
	"github.com/flexigpt/mapstore-go/dirpartition"
	"github.com/flexigpt/mapstore-go/ftsengine"
	"github.com/flexigpt/mapstore-go/jsonencdec"
	"github.com/flexigpt/mapstore-go/uuidv7filename"
)

type Store struct {
	mu     sync.Mutex
	closed bool

	baseDir   string
	enableFTS bool
	pp        mapstore.PartitionProvider

	documents *mapstore.MapDirectoryStore
	index     *mapstore.MapDirectoryStore

	ftsMu             sync.RWMutex
	fts               *ftsengine.Engine
	maintenanceCancel context.CancelFunc
	maintenanceWG     sync.WaitGroup
}

type Option func(*Store) error

func WithPartitionProvider(provider mapstore.PartitionProvider) Option {
	return func(store *Store) error {
		if provider == nil {
			return errors.New("SessionStore partition provider is required")
		}
		store.pp = provider
		return nil
	}
}

func WithFTS(enabled bool) Option {
	return func(store *Store) error {
		store.enableFTS = enabled
		return nil
	}
}

func New(baseDir string, options ...Option) (*Store, error) {
	if strings.TrimSpace(baseDir) == "" {
		return nil, errors.New("SessionStore directory is required")
	}

	partition := &dirpartition.MonthPartitionProvider{
		TimeFn: func(key mapstore.FileKey) (time.Time, error) {
			info, err := uuidv7filename.Parse(key.FileName)
			if err != nil {
				return time.Time{}, err
			}
			return info.Time, nil
		},
	}
	value := &Store{
		baseDir: filepath.Clean(baseDir),
		pp:      partition,
	}
	for _, option := range options {
		if option != nil {
			if err := option(value); err != nil {
				return nil, err
			}
		}
	}

	fail := func(cause error) (*Store, error) {
		return nil, errors.Join(cause, value.Close())
	}

	if value.enableFTS {
		engine, err := ftsengine.NewEngine(
			ftsengine.Config{
				BaseDir:    value.baseDir,
				DBFileName: sessionFTSDatabaseFile,
				Table:      sessionFTSTable,
				Columns: []ftsengine.Column{
					{Name: string(sessionFTSTitleColumn), Weight: 1},
					{Name: string(sessionFTSSystemColumn), Weight: 2},
					{Name: string(sessionFTSUserColumn), Weight: 3},
					{Name: string(sessionFTSAssistantColumn), Weight: 4},
					{Name: string(sessionFTSModifiedColumn), Unindexed: true},
				},
			},
			ftsengine.WithLogger(slog.Default()),
		)
		if err != nil {
			return nil, err
		}
		value.fts = engine
	}

	documentOptions := []mapstore.DirOption{
		mapstore.WithDirLogger(slog.Default()),
	}
	if value.fts != nil {
		documentOptions = append(
			documentOptions,
			mapstore.WithDirFileListeners(value.ftsListener()),
		)
	}

	documents, err := mapstore.NewMapDirectoryStore(
		filepath.Join(value.baseDir, documentDirectory),
		true,
		value.pp,
		jsonencdec.JSONEncoderDecoder{},
		documentOptions...,
	)
	if err != nil {
		return fail(err)
	}
	value.documents = documents

	index, err := mapstore.NewMapDirectoryStore(
		filepath.Join(value.baseDir, indexDirectory),
		true,
		value.pp,
		jsonencdec.JSONEncoderDecoder{},
		mapstore.WithDirLogger(slog.Default()),
	)
	if err != nil {
		return fail(err)
	}
	value.index = index

	// The index is derived. Reconcile it before publication so a crash between
	// document publication and index publication cannot hide a durable Session.
	if err := value.rebuildIndex(); err != nil {
		return fail(err)
	}

	if value.fts != nil {
		maintenanceContext, cancel := context.WithCancel(context.Background())
		value.maintenanceCancel = cancel
		backgroundutil.Launch(
			&value.maintenanceWG,
			"SessionStore FTS reconciliation",
			func() {
				value.ftsMu.RLock()
				defer value.ftsMu.RUnlock()

				stat, err := ftsengine.SyncDirToFTS(
					maintenanceContext,
					value.fts,
					filepath.Join(value.baseDir, documentDirectory),
					string(sessionFTSModifiedColumn),
					1000,
					value.syncDocumentToFTS,
				)
				if err != nil {
					slog.Error("sessionStore FTS reconciliation failed", "error", err)
					return
				}
				if stat != nil {
					slog.Info("sessionStore FTS reconciliation completed", "stat", stat)
				}
			},
			nil,
		)
	}
	return value, nil
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

func (s *Store) HandleCheckpoint(
	_ context.Context,
	checkpoint sessionRuntime.RuntimeCheckpointView,
) error {
	if err := s.lock(); err != nil {
		return err
	}
	defer s.mu.Unlock()

	incoming, err := storedSessionFromCheckpoint(checkpoint)
	if err != nil {
		return PersistenceError{
			SessionID: checkpoint.Session.ID,
			Operation: "projection",
			Cause:     err,
		}
	}
	if err := validateSessionID(incoming.Session.ID); err != nil {
		return err
	}

	current, found, err := s.readDocument(incoming.Session.ID, false)
	if err != nil {
		return err
	}
	if found && current.Kind == documentKindDeleted {
		// A committed deletion dominates every previously queued checkpoint.
		return nil
	}

	if found {
		if current.Session.Revision > incoming.Session.Revision ||
			(current.Session.Revision == incoming.Session.Revision &&
				current.Session.UpdatedAt.After(incoming.Session.UpdatedAt)) {
			incoming.Session = current.Session
		}

		runs := slices.Clone(current.Runs)
		for _, run := range incoming.Runs {
			mergeRun(&runs, run)
		}
		incoming.Runs = runs
	}

	reconcileRunAssociations(incoming.Session, incoming.Runs)
	if err := validateStoredSession(incoming); err != nil {
		return PersistenceError{
			SessionID: incoming.Session.ID,
			Operation: "validation",
			Cause:     err,
		}
	}

	if err := s.writeDocument(incoming); err != nil {
		return PersistenceError{
			SessionID: incoming.Session.ID,
			Operation: "document publication",
			Cause:     err,
		}
	}
	if err := s.writeIndex(incoming); err != nil {
		return PersistenceError{
			SessionID: incoming.Session.ID,
			Operation: "index publication; semantic document was committed",
			Cause:     err,
		}
	}
	return nil
}

func (s *Store) HandleSessionDeleted(
	_ context.Context,
	id sessionRuntime.SessionID,
) error {
	if err := s.lock(); err != nil {
		return err
	}
	defer s.mu.Unlock()

	if err := validateSessionID(id); err != nil {
		return err
	}

	tombstone := storedSession{
		SchemaVersion: SessionSchemaVersion,
		Kind:          documentKindDeleted,
		Deletion: &deletionRecord{
			SessionID: id,
			DeletedAt: time.Now().UTC(),
		},
	}
	if err := s.writeDocument(tombstone); err != nil {
		return PersistenceError{
			SessionID: id,
			Operation: "deletion barrier publication",
			Cause:     err,
		}
	}

	if err := s.deleteIndex(id); err != nil {
		return PersistenceError{
			SessionID: id,
			Operation: "index deletion; durable deletion barrier was committed",
			Cause:     err,
		}
	}
	return nil
}

func (s *Store) Restore(
	ctx context.Context,
	restorer sessionRuntime.RuntimeRestorer,
) error {
	if restorer == nil {
		return errors.New("SessionStore runtime restorer is required")
	}
	if err := s.lock(); err != nil {
		return err
	}
	documents, err := s.allDocuments()
	s.mu.Unlock()
	if err != nil {
		return err
	}

	// No persistence lock or I/O is held while the Session coordinator accepts
	// restored semantic records.
	for _, document := range documents {
		if document.Kind != documentKindSession {
			continue
		}
		if err := restorer.RestoreSession(ctx, *document.Session); err != nil {
			return fmt.Errorf("restore Session %q: %w", document.Session.ID, err)
		}
	}
	for _, document := range documents {
		if document.Kind != documentKindSession {
			continue
		}
		for _, run := range orderedRuns(document) {
			if err := restorer.RestoreRun(ctx, run); err != nil {
				return fmt.Errorf("restore Run %q: %w", run.ID, err)
			}
		}
	}
	return nil
}

func (s *Store) ReadSession(
	_ context.Context,
	id sessionRuntime.SessionID,
	forceFetch bool,
) (sessionRuntime.Session, error) {
	if err := s.lock(); err != nil {
		return sessionRuntime.Session{}, err
	}
	defer s.mu.Unlock()

	document, err := s.requireSession(id, forceFetch)
	if err != nil {
		return sessionRuntime.Session{}, err
	}
	return *document.Session, nil
}

func (s *Store) ReadConversation(
	_ context.Context,
	id sessionRuntime.SessionID,
	forceFetch bool,
) (ConversationProjection, error) {
	if err := s.lock(); err != nil {
		return ConversationProjection{}, err
	}
	defer s.mu.Unlock()

	document, err := s.requireSession(id, forceFetch)
	if err != nil {
		return ConversationProjection{}, err
	}
	value := document.Session
	return ConversationProjection{
		SchemaVersion:    SessionSchemaVersion,
		SessionID:        value.ID,
		SessionRevision:  value.Revision,
		Title:            value.Title,
		CreatedAt:        value.CreatedAt,
		ModifiedAt:       value.UpdatedAt,
		Messages:         value.Transcript,
		AssociatedRunIDs: slices.Clone(value.RunIDs),
	}, nil
}

func (s *Store) ReadRun(
	_ context.Context,
	sessionID sessionRuntime.SessionID,
	runID sessionRuntime.RunID,
	forceFetch bool,
) (sessionRuntime.RunSnapshot, error) {
	if err := s.lock(); err != nil {
		return sessionRuntime.RunSnapshot{}, err
	}
	defer s.mu.Unlock()

	document, err := s.requireSession(sessionID, forceFetch)
	if err != nil {
		return sessionRuntime.RunSnapshot{}, err
	}
	for _, run := range document.Runs {
		if run.ID == runID {
			return sessionRuntime.RunSnapshot{
				SessionID:       sessionID,
				SessionRevision: document.Session.Revision,
				Run:             run,
			}, nil
		}
	}
	return sessionRuntime.RunSnapshot{}, ErrRunNotFound
}

func (s *Store) ListSessionRuns(
	_ context.Context,
	request ListSessionRunsRequest,
) ([]RunListItem, error) {
	if err := s.lock(); err != nil {
		return nil, err
	}
	defer s.mu.Unlock()

	document, err := s.requireSession(request.SessionID, false)
	if err != nil {
		return nil, err
	}

	items := make([]RunListItem, 0, len(document.Runs))
	for _, run := range orderedRuns(document) {
		if request.ActiveOnly && run.State.Terminal() {
			continue
		}
		count := 0
		for _, wait := range run.WaitingRequests {
			if wait.Status == sessionRuntime.WaitStatusPending ||
				wait.Status == sessionRuntime.WaitStatusResolving {
				count++
			}
		}
		items = append(items, RunListItem{
			ID:               run.ID,
			Revision:         run.Revision,
			State:            run.State,
			CreatedAt:        run.CreatedAt,
			UpdatedAt:        run.UpdatedAt,
			PendingWaitCount: count,
			Outcome:          run.Outcome,
		})
	}
	sort.Slice(items, func(left, right int) bool {
		if !items[left].UpdatedAt.Equal(items[right].UpdatedAt) {
			return items[left].UpdatedAt.After(items[right].UpdatedAt)
		}
		return items[left].ID > items[right].ID
	})
	if request.Limit > 0 {
		items = items[:min(request.Limit, len(items))]
	}
	return items, nil
}

func (s *Store) ListSessions(
	_ context.Context,
	request ListSessionsRequest,
) (SessionPage, error) {
	if err := s.lock(); err != nil {
		return SessionPage{}, err
	}
	defer s.mu.Unlock()

	entries, next, err := s.index.ListFiles(
		mapstore.ListingConfig{
			SortOrder: mapstore.SortOrderDescending,
			PageSize:  normalizedPageSize(request.PageSize),
		},
		request.PageToken,
	)
	if err != nil {
		return SessionPage{}, err
	}

	items := make([]SessionListItem, 0, len(entries))
	for _, entry := range entries {
		filename := filepath.Base(entry.BaseRelativePath)
		if !strings.HasSuffix(filename, "."+indexFileExtension) {
			continue
		}
		record, err := s.readIndexFile(filename)
		if err != nil {
			return SessionPage{}, err
		}
		items = append(items, record.Item)
	}
	return SessionPage{
		Items:         items,
		NextPageToken: next,
	}, nil
}

func (s *Store) SearchSessions(
	ctx context.Context,
	request SearchSessionsRequest,
) (SessionPage, error) {
	if err := s.lock(); err != nil {
		return SessionPage{}, err
	}
	defer s.mu.Unlock()

	if s.fts == nil {
		return SessionPage{}, ErrFTSDisabled
	}
	hits, next, err := s.fts.Search(
		ctx,
		request.Query,
		request.PageToken,
		normalizedPageSize(request.PageSize),
	)
	if err != nil {
		return SessionPage{}, err
	}

	items := make([]SessionListItem, 0, len(hits))
	seen := make(map[sessionRuntime.SessionID]struct{}, len(hits))
	for _, hit := range hits {
		info, err := uuidv7filename.Parse(filepath.Base(hit.ID))
		if err != nil {
			return SessionPage{}, fmt.Errorf("%w: malformed FTS identity", ErrInvalidDocument)
		}
		id := sessionRuntime.SessionID(info.ID)
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
	return SessionPage{
		Items:         items,
		NextPageToken: next,
	}, nil
}

func (s *Store) ExportSession(
	_ context.Context,
	id sessionRuntime.SessionID,
) ([]byte, error) {
	if err := s.lock(); err != nil {
		return nil, err
	}
	defer s.mu.Unlock()

	document, err := s.requireSession(id, true)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(document, "", "  ")
}

func (s *Store) requireSession(
	id sessionRuntime.SessionID,
	forceFetch bool,
) (storedSession, error) {
	document, found, err := s.readDocument(id, forceFetch)
	if err != nil {
		return storedSession{}, err
	}
	if !found || document.Kind == documentKindDeleted {
		return storedSession{}, ErrSessionNotFound
	}
	return document, nil
}

func (s *Store) readDocument(
	id sessionRuntime.SessionID,
	forceFetch bool,
) (storedSession, bool, error) {
	if err := validateSessionID(id); err != nil {
		return storedSession{}, false, err
	}

	filename, err := stableFilename(id, SessionFileExtension.String())
	if err != nil {
		return storedSession{}, false, err
	}
	found, err := hasExactFile(s.documents, filename, string(id))
	if err != nil || !found {
		return storedSession{}, false, err
	}

	data, err := s.documents.GetFileData(
		mapstore.FileKey{FileName: filename},
		forceFetch,
	)
	if err != nil {
		return storedSession{}, false, err
	}
	document, err := decodeStoredSessionData(data)
	if err != nil {
		return storedSession{}, false, err
	}

	if documentID(document) != id {
		return storedSession{}, false, fmt.Errorf(
			"%w: document identity differs from filename",
			ErrInvalidDocument,
		)
	}
	return document, true, nil
}

func (s *Store) writeDocument(document storedSession) error {
	id := documentID(document)
	filename, err := stableFilename(id, SessionFileExtension.String())
	if err != nil {
		return err
	}

	raw, err := json.Marshal(document)
	if err != nil {
		return err
	}
	if len(raw) > maximumDocumentBytes {
		return fmt.Errorf("%w: document size limit exceeded", ErrInvalidDocument)
	}

	data, err := jsonencdec.StructWithJSONTagsToMap(&document)
	if err != nil {
		return err
	}
	return s.documents.SetFileData(
		mapstore.FileKey{FileName: filename},
		data,
	)
}

func (s *Store) writeIndex(document storedSession) error {
	record := indexForDocument(document)
	filename, err := stableFilename(record.Item.ID, indexFileExtension)
	if err != nil {
		return err
	}
	data, err := jsonencdec.StructWithJSONTagsToMap(&record)
	if err != nil {
		return err
	}
	return s.index.SetFileData(mapstore.FileKey{FileName: filename}, data)
}

func (s *Store) deleteIndex(id sessionRuntime.SessionID) error {
	filename, err := stableFilename(id, indexFileExtension)
	if err != nil {
		return err
	}
	found, err := hasExactFile(s.index, filename, string(id))
	if err != nil || !found {
		return err
	}
	return s.index.DeleteFile(mapstore.FileKey{FileName: filename})
}

func (s *Store) readIndex(
	id sessionRuntime.SessionID,
) (sessionIndexRecord, bool, error) {
	filename, err := stableFilename(id, indexFileExtension)
	if err != nil {
		return sessionIndexRecord{}, false, err
	}
	found, err := hasExactFile(s.index, filename, string(id))
	if err != nil || !found {
		return sessionIndexRecord{}, false, err
	}
	record, err := s.readIndexFile(filename)
	return record, true, err
}

func (s *Store) readIndexFile(filename string) (sessionIndexRecord, error) {
	data, err := s.index.GetFileData(mapstore.FileKey{FileName: filename}, false)
	if err != nil {
		return sessionIndexRecord{}, err
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return sessionIndexRecord{}, err
	}
	record, err := jsonutil.DecodeJSONRaw[sessionIndexRecord](raw)
	if err != nil {
		return sessionIndexRecord{}, err
	}
	if record.SchemaVersion != SessionSchemaVersion {
		return sessionIndexRecord{}, ErrUnsupportedStore
	}
	if err := validateSessionID(record.Item.ID); err != nil {
		return sessionIndexRecord{}, err
	}
	expected, err := stableFilename(record.Item.ID, indexFileExtension)
	if err != nil || expected != filename {
		return sessionIndexRecord{}, fmt.Errorf("%w: index identity mismatch", ErrInvalidDocument)
	}
	return record, nil
}

func (s *Store) allDocuments() ([]storedSession, error) {
	filenames, err := listAllFilenames(s.documents, SessionFileExtension.String())
	if err != nil {
		return nil, err
	}
	output := make([]storedSession, 0, len(filenames))
	for _, filename := range filenames {
		data, err := s.documents.GetFileData(mapstore.FileKey{FileName: filename}, false)
		if err != nil {
			return nil, err
		}
		document, err := decodeStoredSessionData(data)
		if err != nil {
			return nil, fmt.Errorf("read %q: %w", filename, err)
		}
		expected, err := stableFilename(documentID(document), SessionFileExtension.String())
		if err != nil || expected != filename {
			return nil, fmt.Errorf("%w: document filename mismatch", ErrInvalidDocument)
		}
		output = append(output, document)
	}
	return output, nil
}

func (s *Store) rebuildIndex() error {
	documents, err := s.allDocuments()
	if err != nil {
		return err
	}

	live := make(map[sessionRuntime.SessionID]struct{}, len(documents))
	for _, document := range documents {
		if document.Kind == documentKindDeleted {
			if err := s.deleteIndex(document.Deletion.SessionID); err != nil {
				return err
			}
			continue
		}
		live[document.Session.ID] = struct{}{}
		if err := s.writeIndex(document); err != nil {
			return err
		}
	}

	filenames, err := listAllFilenames(s.index, indexFileExtension)
	if err != nil {
		return err
	}
	for _, filename := range filenames {
		record, err := s.readIndexFile(filename)
		if err != nil {
			return err
		}
		if _, found := live[record.Item.ID]; !found {
			if err := s.index.DeleteFile(mapstore.FileKey{FileName: filename}); err != nil {
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

func decodeStoredSessionData(data map[string]any) (storedSession, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return storedSession{}, err
	}
	return decodeStoredSessionJSON(raw)
}

func decodeStoredSessionJSON(raw []byte) (storedSession, error) {
	if len(raw) > maximumDocumentBytes {
		return storedSession{}, fmt.Errorf("%w: document size limit exceeded", ErrInvalidDocument)
	}
	document, err := jsonutil.DecodeJSONRaw[storedSession](raw)
	if err != nil {
		return storedSession{}, fmt.Errorf("%w: %w", ErrInvalidDocument, err)
	}
	if err := validateStoredSession(document); err != nil {
		return storedSession{}, err
	}
	return document, nil
}

func stableFilename(id sessionRuntime.SessionID, extension string) (string, error) {
	info, err := uuidv7filename.Build(string(id), "Session", extension)
	if err != nil {
		return "", err
	}
	return info.FileName, nil
}

func documentID(document storedSession) sessionRuntime.SessionID {
	if document.Kind == documentKindDeleted {
		return document.Deletion.SessionID
	}
	return document.Session.ID
}

func hasExactFile(
	store *mapstore.MapDirectoryStore,
	filename string,
	prefix string,
) (bool, error) {
	entries, _, err := store.ListFiles(
		mapstore.ListingConfig{
			FilenamePrefix: prefix,
			PageSize:       MaxPageSize,
		},
		"",
	)
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		if filepath.Base(entry.BaseRelativePath) == filename {
			return true, nil
		}
	}
	return false, nil
}

func listAllFilenames(
	store *mapstore.MapDirectoryStore,
	extension string,
) ([]string, error) {
	output := make([]string, 0)
	token := ""
	seen := make(map[string]struct{})

	for {
		entries, next, err := store.ListFiles(
			mapstore.ListingConfig{
				SortOrder: mapstore.SortOrderDescending,
				PageSize:  MaxPageSize,
			},
			token,
		)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			filename := filepath.Base(entry.BaseRelativePath)
			if strings.HasSuffix(filename, "."+extension) {
				output = append(output, filename)
			}
		}
		if next == "" {
			return output, nil
		}
		if _, duplicate := seen[next]; duplicate {
			return nil, fmt.Errorf("%w: repeated listing cursor", ErrInvalidDocument)
		}
		seen[next] = struct{}{}
		token = next
	}
}

func indexForDocument(document storedSession) sessionIndexRecord {
	sessionValue := document.Session
	active := make([]sessionRuntime.RunID, 0)
	for _, run := range orderedRuns(document) {
		if !run.State.Terminal() {
			active = append(active, run.ID)
		}
	}
	return sessionIndexRecord{
		SchemaVersion: SessionSchemaVersion,
		Item: SessionListItem{
			ID:           sessionValue.ID,
			Revision:     sessionValue.Revision,
			Title:        sessionValue.Title,
			CreatedAt:    sessionValue.CreatedAt,
			ModifiedAt:   sessionValue.UpdatedAt,
			RunCount:     len(document.Runs),
			ActiveRunIDs: active,
		},
	}
}

func mergeRun(runs *[]sessionRuntime.Run, incoming sessionRuntime.Run) {
	for index := range *runs {
		current := (*runs)[index]
		if current.ID != incoming.ID {
			continue
		}
		if incoming.Revision > current.Revision ||
			(incoming.Revision == current.Revision &&
				incoming.UpdatedAt.After(current.UpdatedAt)) {
			(*runs)[index] = incoming
		}
		return
	}
	*runs = append(*runs, incoming)
}

func reconcileRunAssociations(
	sessionValue *sessionRuntime.Session,
	runs []sessionRuntime.Run,
) {
	available := make(map[sessionRuntime.RunID]struct{}, len(runs))
	for _, run := range runs {
		available[run.ID] = struct{}{}
	}

	ids := make([]sessionRuntime.RunID, 0, len(runs))
	for _, id := range sessionValue.RunIDs {
		if _, found := available[id]; found {
			ids = append(ids, id)
			delete(available, id)
		}
	}
	for _, run := range runs {
		if _, found := available[run.ID]; found {
			ids = append(ids, run.ID)
			delete(available, run.ID)
		}
	}
	sessionValue.RunIDs = ids
}

func orderedRuns(document storedSession) []sessionRuntime.Run {
	byID := make(map[sessionRuntime.RunID]sessionRuntime.Run, len(document.Runs))
	for _, run := range document.Runs {
		byID[run.ID] = run
	}
	output := make([]sessionRuntime.Run, 0, len(document.Runs))
	for _, id := range document.Session.RunIDs {
		if run, found := byID[id]; found {
			output = append(output, run)
		}
	}
	return output
}
