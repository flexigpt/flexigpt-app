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
	"github.com/flexigpt/mapstore-go"
	"github.com/flexigpt/mapstore-go/dirpartition"
	"github.com/flexigpt/mapstore-go/ftsengine"
	"github.com/flexigpt/mapstore-go/jsonencdec"
	"github.com/flexigpt/mapstore-go/uuidv7filename"
)

const defaultSessionFilenameTitle = "Session"

type storedSession struct {
	SchemaVersion SchemaVersion          `json:"schemaVersion"`
	Session       sessionRuntime.Session `json:"session"`
	Runs          []sessionRuntime.Run   `json:"runs,omitempty"`
}

type Store struct {
	baseDir   string
	enableFTS bool
	store     *mapstore.MapDirectoryStore
	fts       *ftsengine.Engine
	pp        mapstore.PartitionProvider

	ftsRebuildCtx    context.Context
	ftsRebuildCancel context.CancelFunc
	ftsRebuildWG     sync.WaitGroup
}

type Option func(*Store) error

func WithPartitionProvider(
	provider mapstore.PartitionProvider,
) Option {
	return func(store *Store) error {
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
	defaultPartitionProvider := dirpartition.MonthPartitionProvider{
		TimeFn: func(fileKey mapstore.FileKey) (time.Time, error) {
			info, err := uuidv7filename.Parse(fileKey.FileName)
			if err != nil {
				return time.Time{}, err
			}
			return info.Time, nil
		},
	}

	value := &Store{
		baseDir: filepath.Clean(baseDir),
		pp:      &defaultPartitionProvider,
	}

	for _, option := range options {
		if err := option(value); err != nil {
			return nil, err
		}
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
		value.ftsRebuildCtx, value.ftsRebuildCancel = context.WithCancel(
			context.Background(),
		)
		value.ftsRebuildWG.Go(func() {
			stat, syncErr := ftsengine.SyncDirToFTS(
				value.ftsRebuildCtx,
				value.fts,
				value.baseDir,
				string(sessionFTSModifiedColumn),
				1000,
				syncSessionDocumentToFTS,
			)
			if syncErr != nil {
				slog.Error(
					"session store FTS rebuild failed",
					"error",
					syncErr,
				)
				return
			}
			if stat != nil {
				slog.Info("session store FTS rebuild", "stat", stat)
			}
		})
	}

	directoryOptions := []mapstore.DirOption{
		mapstore.WithDirLogger(slog.Default()),
	}
	if value.fts != nil {
		directoryOptions = append(
			directoryOptions,
			mapstore.WithDirFileListeners(newFTSListener(value.fts)),
		)
	}

	directoryStore, err := mapstore.NewMapDirectoryStore(
		value.baseDir,
		true,
		value.pp,
		jsonencdec.JSONEncoderDecoder{},
		directoryOptions...,
	)
	if err != nil {
		return nil, err
	}

	value.store = directoryStore
	return value, nil
}

func (s *Store) Close() (err error) {
	if s.ftsRebuildCancel != nil {
		s.ftsRebuildCancel()
		s.ftsRebuildWG.Wait()
		s.ftsRebuildCancel = nil
	}

	if s.store != nil {
		err = errors.Join(err, s.store.CloseAll())
		s.store = nil
	}

	if s.fts != nil {
		err = errors.Join(err, s.fts.Close())
		s.fts = nil
	}

	return err
}

func (s *Store) HandleCheckpoint(
	_ context.Context,
	checkpoint sessionRuntime.RuntimeCheckpointView,
) error {
	document := storedSessionFromCheckpoint(checkpoint)

	existing, filename, found, err := s.readStoredSession(
		document.Session.ID,
		false,
	)
	if err != nil {
		return err
	}

	if found && existing.Session.Revision > document.Session.Revision {
		return nil
	}

	var checkpointRun *sessionRuntime.Run
	if checkpoint.Run != nil {
		run := projectRun(*checkpoint.Run)
		checkpointRun = &run
	}

	if found {
		document.Runs = slices.Clone(existing.Runs)
	}

	if checkpointRun != nil {
		mergeRun(&document.Runs, *checkpointRun)
	}

	for _, run := range document.Runs {
		appendRunID(&document.Session.RunIDs, run.ID)
	}

	return s.writeStoredSession(document, filename)
}

func (s *Store) Restore(
	ctx context.Context,
	restorer sessionRuntime.RuntimeRestorer,
) error {
	documents, err := s.allStoredSessions()
	if err != nil {
		return err
	}

	var restoreErr error

	for _, document := range documents {
		restoreErr = errors.Join(
			restoreErr,
			restorer.RestoreSession(
				ctx,
				projectSession(document.Session),
			),
		)
	}

	for _, document := range documents {
		for _, run := range orderedRuns(document) {
			restoreErr = errors.Join(
				restoreErr,
				restorer.RestoreRun(ctx, projectRun(run)),
			)
		}
	}

	return restoreErr
}

func (s *Store) ReadSession(
	_ context.Context,
	id sessionRuntime.SessionID,
	forceFetch bool,
) (sessionRuntime.Session, error) {
	document, _, found, err := s.readStoredSession(id, forceFetch)
	if err != nil {
		return sessionRuntime.Session{}, err
	}
	if !found {
		return sessionRuntime.Session{}, ErrSessionNotFound
	}

	return projectSession(document.Session), nil
}

func (s *Store) ReadConversation(
	_ context.Context,
	id sessionRuntime.SessionID,
	forceFetch bool,
) (ConversationProjection, error) {
	document, _, found, err := s.readStoredSession(id, forceFetch)
	if err != nil {
		return ConversationProjection{}, err
	}
	if !found {
		return ConversationProjection{}, ErrSessionNotFound
	}

	return ConversationProjection{
		SchemaVersion:    SessionSchemaVersion,
		SessionID:        document.Session.ID,
		SessionRevision:  document.Session.Revision,
		Title:            document.Session.Title,
		CreatedAt:        document.Session.CreatedAt,
		ModifiedAt:       document.Session.UpdatedAt,
		Messages:         projectTranscript(document.Session.Transcript),
		AssociatedRunIDs: slices.Clone(document.Session.RunIDs),
	}, nil
}

func (s *Store) ReadRun(
	_ context.Context,
	sessionID sessionRuntime.SessionID,
	runID sessionRuntime.RunID,
	forceFetch bool,
) (sessionRuntime.RunSnapshot, error) {
	document, _, found, err := s.readStoredSession(
		sessionID,
		forceFetch,
	)
	if err != nil {
		return sessionRuntime.RunSnapshot{}, err
	}
	if !found {
		return sessionRuntime.RunSnapshot{}, ErrSessionNotFound
	}

	for _, run := range document.Runs {
		if run.ID != runID {
			continue
		}

		return sessionRuntime.RunSnapshot{
			SessionID:       sessionID,
			SessionRevision: document.Session.Revision,
			Run:             projectRun(run),
		}, nil
	}

	return sessionRuntime.RunSnapshot{}, ErrRunNotFound
}

func (s *Store) ListSessionRuns(
	_ context.Context,
	request ListSessionRunsRequest,
) ([]RunListItem, error) {
	document, _, found, err := s.readStoredSession(
		request.SessionID,
		false,
	)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrSessionNotFound
	}

	runs := orderedRuns(document)
	items := make([]RunListItem, 0, len(runs))

	for _, run := range runs {
		if request.ActiveOnly && run.State.Terminal() {
			continue
		}

		items = append(items, RunListItem{
			ID:               run.ID,
			Revision:         run.Revision,
			State:            run.State,
			CreatedAt:        run.CreatedAt,
			UpdatedAt:        run.UpdatedAt,
			PendingWaitCount: pendingWaitCount(run),
			Outcome:          copyRunOutcome(run.Outcome),
		})
	}

	sort.Slice(items, func(left, right int) bool {
		if !items[left].UpdatedAt.Equal(items[right].UpdatedAt) {
			return items[left].UpdatedAt.After(items[right].UpdatedAt)
		}
		return items[left].ID > items[right].ID
	})

	if request.Limit > 0 && len(items) > request.Limit {
		items = items[:request.Limit]
	}

	return items, nil
}

func (s *Store) ListSessions(
	_ context.Context,
	request ListSessionsRequest,
) (SessionPage, error) {
	entries, next, err := s.store.ListFiles(
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
		if _, err := uuidv7filename.Parse(filename); err != nil {
			continue
		}

		document, err := s.readStoredSessionFile(filename, false)
		if err != nil {
			continue
		}

		items = append(items, SessionListItem{
			ID:           document.Session.ID,
			Revision:     document.Session.Revision,
			Title:        document.Session.Title,
			CreatedAt:    document.Session.CreatedAt,
			ModifiedAt:   entry.FileInfo.ModTime().UTC(),
			RunCount:     len(document.Runs),
			ActiveRunIDs: activeRunIDs(document),
		})
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
	for _, hit := range hits {
		filename := filepath.Base(hit.ID)
		if _, err := uuidv7filename.Parse(filename); err != nil {
			continue
		}

		document, err := s.readStoredSessionFile(filename, false)
		if err != nil {
			continue
		}

		items = append(items, SessionListItem{
			ID:           document.Session.ID,
			Revision:     document.Session.Revision,
			Title:        document.Session.Title,
			CreatedAt:    document.Session.CreatedAt,
			ModifiedAt:   document.Session.UpdatedAt,
			RunCount:     len(document.Runs),
			ActiveRunIDs: activeRunIDs(document),
		})
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
	document, _, found, err := s.readStoredSession(id, true)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrSessionNotFound
	}

	return json.MarshalIndent(document, "", "  ")
}

func (s *Store) allStoredSessions() ([]storedSession, error) {
	documents := make([]storedSession, 0)
	seenTokens := make([]string, 0)
	pageToken := ""

	for {
		entries, next, err := s.store.ListFiles(
			mapstore.ListingConfig{
				SortOrder: mapstore.SortOrderDescending,
				PageSize:  MaxPageSize,
			},
			pageToken,
		)
		if err != nil {
			return nil, err
		}

		for _, entry := range entries {
			filename := filepath.Base(entry.BaseRelativePath)
			if _, err := uuidv7filename.Parse(filename); err != nil {
				continue
			}

			document, err := s.readStoredSessionFile(filename, false)
			if err != nil {
				continue
			}

			documents = append(documents, cloneStoredSession(document))
		}

		if next == "" {
			break
		}
		if slices.Contains(seenTokens, next) {
			return nil, fmt.Errorf(
				"session store listing repeated page token %q",
				next,
			)
		}

		seenTokens = append(seenTokens, next)
		pageToken = next
	}

	return documents, nil
}

func (s *Store) readStoredSession(
	id sessionRuntime.SessionID,
	forceFetch bool,
) (doc storedSession, fname string, found bool, err error) {
	entries, _, err := s.store.ListFiles(
		mapstore.ListingConfig{
			FilenamePrefix: string(id),
			PageSize:       MaxPageSize,
		},
		"",
	)
	if err != nil {
		return storedSession{}, "", false, err
	}

	var selected storedSession
	selectedFilename := ""

	for _, entry := range entries {
		filename := filepath.Base(entry.BaseRelativePath)
		info, err := uuidv7filename.Parse(filename)
		if err != nil || info.ID != string(id) {
			continue
		}

		document, err := s.readStoredSessionFile(filename, forceFetch)
		if err != nil {
			return storedSession{}, "", false, err
		}
		if document.Session.ID != id {
			continue
		}

		if !found ||
			document.Session.Revision > selected.Session.Revision ||
			(document.Session.Revision == selected.Session.Revision &&
				document.Session.UpdatedAt.After(selected.Session.UpdatedAt)) {
			selected = document
			selectedFilename = filename
			found = true
		}
	}

	return selected, selectedFilename, found, nil
}

func (s *Store) readStoredSessionFile(
	filename string,
	forceFetch bool,
) (storedSession, error) {
	value, err := s.store.GetFileData(
		mapstore.FileKey{FileName: filename},
		forceFetch,
	)
	if err != nil {
		return storedSession{}, err
	}

	return decodeStoredSessionData(value)
}

func (s *Store) writeStoredSession(
	document storedSession,
	previousFilename string,
) error {
	filename, err := sessionFilename(document.Session)
	if err != nil {
		return err
	}

	value, err := jsonencdec.StructWithJSONTagsToMap(&document)
	if err != nil {
		return err
	}

	if err := s.store.SetFileData(
		mapstore.FileKey{FileName: filename},
		value,
	); err != nil {
		return err
	}

	if previousFilename != "" && previousFilename != filename {
		if err := s.store.DeleteFile(
			mapstore.FileKey{FileName: previousFilename},
		); err != nil {
			slog.Warn(
				"session store remove renamed session file",
				"file",
				previousFilename,
				"error",
				err,
			)
		}
	}

	return nil
}

func decodeStoredSessionData(
	value map[string]any,
) (storedSession, error) {
	var document storedSession
	if err := jsonencdec.MapToStructWithJSONTags(value, &document); err != nil {
		return storedSession{}, err
	}
	if document.SchemaVersion != SessionSchemaVersion {
		return storedSession{}, fmt.Errorf(
			"%w: %q",
			ErrUnsupportedStore,
			document.SchemaVersion,
		)
	}

	return document, nil
}

func decodeStoredSessionJSON(
	value []byte,
) (storedSession, error) {
	var document storedSession
	if err := json.Unmarshal(value, &document); err != nil {
		return storedSession{}, err
	}
	if document.SchemaVersion != SessionSchemaVersion {
		return storedSession{}, fmt.Errorf(
			"%w: %q",
			ErrUnsupportedStore,
			document.SchemaVersion,
		)
	}

	return document, nil
}

func sessionFilename(
	value sessionRuntime.Session,
) (string, error) {
	title := strings.TrimSpace(value.Title)
	if title == "" {
		title = defaultSessionFilenameTitle
	}

	info, err := uuidv7filename.Build(
		string(value.ID),
		title,
		SessionFileExtension.String(),
	)
	if err != nil {
		return "", err
	}

	return info.FileName, nil
}

func mergeRun(
	runs *[]sessionRuntime.Run,
	incoming sessionRuntime.Run,
) {
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

func appendRunID(
	runIDs *[]sessionRuntime.RunID,
	id sessionRuntime.RunID,
) {
	if id == "" || slices.Contains(*runIDs, id) {
		return
	}
	*runIDs = append(*runIDs, id)
}

func orderedRuns(
	document storedSession,
) []sessionRuntime.Run {
	output := make([]sessionRuntime.Run, 0, len(document.Runs))

	for _, id := range document.Session.RunIDs {
		run, found := findRun(document.Runs, id)
		if !found {
			continue
		}
		output = append(output, run)
	}

	for _, run := range document.Runs {
		if containsRun(output, run.ID) {
			continue
		}
		output = append(output, run)
	}

	return output
}

func activeRunIDs(
	document storedSession,
) []sessionRuntime.RunID {
	runs := orderedRuns(document)
	output := make([]sessionRuntime.RunID, 0, len(runs))

	for _, run := range runs {
		if run.State.Terminal() {
			continue
		}
		output = append(output, run.ID)
	}

	return output
}

func findRun(
	runs []sessionRuntime.Run,
	id sessionRuntime.RunID,
) (sessionRuntime.Run, bool) {
	for _, run := range runs {
		if run.ID == id {
			return run, true
		}
	}

	return sessionRuntime.Run{}, false
}

func containsRun(
	runs []sessionRuntime.Run,
	id sessionRuntime.RunID,
) bool {
	for _, run := range runs {
		if run.ID == id {
			return true
		}
	}
	return false
}

func pendingWaitCount(
	run sessionRuntime.Run,
) int {
	count := 0
	for _, wait := range run.WaitingRequests {
		if wait.Status == sessionRuntime.WaitStatusPending ||
			wait.Status == sessionRuntime.WaitStatusResolving {
			count++
		}
	}

	return count
}

func copyRunOutcome(
	value *sessionRuntime.RunOutcome,
) *sessionRuntime.RunOutcome {
	if value == nil {
		return nil
	}

	output := *value
	return &output
}
