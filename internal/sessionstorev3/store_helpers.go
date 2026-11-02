package sessionstorev3

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/mapstore-go"
	"github.com/flexigpt/mapstore-go/jsonencdec"
	"github.com/flexigpt/mapstore-go/uuidv7filename"
)

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

func (s *Store) readDocument(id sessionv3.SessionID, _ bool) (sessionDocument, bool, error) {
	filename, err := stableSessionFilename(id, documentExtension)
	if err != nil {
		return sessionDocument{}, false, err
	}
	found, err := hasExactStoreFile(s.documents, filename, string(id))
	if err != nil || !found {
		return sessionDocument{}, false, err
	}
	data, err := s.documents.GetFileData(mapstore.FileKey{FileName: filename}, false)
	if err != nil {
		return sessionDocument{}, false, err
	}
	doc, err := decodeStoreDocument(data)
	if err != nil {
		return sessionDocument{}, false, err
	}
	return doc, true, nil
}

func (s *Store) writeDocument(session sessionv3.Session) error {
	doc := sessionDocument{
		SchemaVersion: sessionv3.SchemaVersion,
		Kind:          sessionDocumentKindSession,
		SessionID:     session.ID,
		Session:       &session,
	}
	return s.writeRawDocument(session.ID, doc)
}

func (s *Store) writeRawDocument(id sessionv3.SessionID, doc sessionDocument) error {
	filename, err := stableSessionFilename(id, documentExtension)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	if len(raw) > maximumDocumentBytes {
		return fmt.Errorf("%w: document too large", ErrInvalidDocument)
	}
	data, err := jsonencdec.StructWithJSONTagsToMap(&doc)
	if err != nil {
		return err
	}
	return s.documents.SetFileData(mapstore.FileKey{FileName: filename}, data)
}

func (s *Store) writeIndex(session sessionv3.Session) error {
	active := []sessionv3.ExecutionID{}
	for _, e := range session.Executions {
		if !e.State.Terminal() {
			active = append(active, e.ID)
		}
	}
	record := sessionIndexRecord{
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
	}
	filename, err := stableSessionFilename(session.ID, indexExtension)
	if err != nil {
		return err
	}
	data, err := jsonencdec.StructWithJSONTagsToMap(&record)
	if err != nil {
		return err
	}
	return s.index.SetFileData(mapstore.FileKey{FileName: filename}, data)
}

func (s *Store) deleteIndex(id sessionv3.SessionID) error {
	filename, err := stableSessionFilename(id, indexExtension)
	if err != nil {
		return err
	}
	found, err := hasExactStoreFile(s.index, filename, string(id))
	if err != nil || !found {
		return err
	}
	return s.index.DeleteFile(mapstore.FileKey{FileName: filename})
}

func (s *Store) readIndex(id sessionv3.SessionID) (sessionIndexRecord, bool, error) {
	filename, err := stableSessionFilename(id, indexExtension)
	if err != nil {
		return sessionIndexRecord{}, false, err
	}
	found, err := hasExactStoreFile(s.index, filename, string(id))
	if err != nil || !found {
		return sessionIndexRecord{}, false, err
	}
	si, err := s.readIndexFile(filepath.Base(filename))
	return si, true, err
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
	var record sessionIndexRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return sessionIndexRecord{}, err
	}
	return record, nil
}

func (s *Store) allDocuments() ([]sessionDocument, error) {
	filenames, err := listAllStoreFiles(s.documents, documentExtension)
	if err != nil {
		return nil, err
	}
	out := make([]sessionDocument, 0, len(filenames))
	for _, filename := range filenames {
		data, err := s.documents.GetFileData(mapstore.FileKey{FileName: filename}, false)
		if err != nil {
			return nil, err
		}
		doc, err := decodeStoreDocument(data)
		if err != nil {
			return nil, err
		}
		out = append(out, doc)
	}
	return out, nil
}

func cloneSession(v sessionv3.Session) sessionv3.Session {
	raw, _ := json.Marshal(v)
	var out sessionv3.Session
	_ = json.Unmarshal(raw, &out)
	return out
}

func cloneExecution(v sessionv3.Execution) sessionv3.Execution {
	raw, _ := json.Marshal(v)
	var out sessionv3.Execution
	_ = json.Unmarshal(raw, &out)
	return out
}

func cloneOutcome(v *sessionv3.ExecutionOutcome) *sessionv3.ExecutionOutcome {
	if v == nil {
		return nil
	}
	c := *v
	return &c
}

func normalizePageSize(n int) int {
	if n <= 0 {
		return defaultPageSize
	}
	if n > maxPageSize {
		return maxPageSize
	}
	return n
}

func filepathBase(p string) string {
	return filepath.Base(p)
}

func stableSessionFilename(id sessionv3.SessionID, ext string) (string, error) {
	info, err := uuidv7filename.Build(string(id), "Session", strings.TrimPrefix(ext, "."))
	if err != nil {
		return "", err
	}
	return info.FileName, nil
}

func hasExactStoreFile(store *mapstore.MapDirectoryStore, filename, prefix string) (bool, error) {
	entries, _, err := store.ListFiles(mapstore.ListingConfig{FilenamePrefix: prefix, PageSize: maxPageSize}, "")
	if err != nil {
		return false, err
	}
	for _, e := range entries {
		if filepath.Base(e.BaseRelativePath) == filename {
			return true, nil
		}
	}
	return false, nil
}

func listAllStoreFiles(store *mapstore.MapDirectoryStore, ext string) ([]string, error) {
	out := []string{}
	token := ""
	for {
		entries, next, err := store.ListFiles(
			mapstore.ListingConfig{SortOrder: mapstore.SortOrderDescending, PageSize: maxPageSize},
			token,
		)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			filename := filepath.Base(e.BaseRelativePath)
			if strings.HasSuffix(filename, ext) {
				out = append(out, filename)
			}
		}
		if next == "" {
			return out, nil
		}
		token = next
	}
}

func decodeStoreDocument(data map[string]any) (sessionDocument, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return sessionDocument{}, err
	}
	var doc sessionDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		return sessionDocument{}, err
	}
	return doc, nil
}
