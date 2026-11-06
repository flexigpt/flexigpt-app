package sessionstorev3

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/uuidutil"
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

func (s *Store) readDocument(id sessionv3Runtime.SessionID, _ bool) (sessionDocument, bool, error) {
	if err := validateStoreSessionID(id); err != nil {
		return sessionDocument{}, false, err
	}

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
	if doc.SessionID != id {
		return sessionDocument{}, false, fmt.Errorf(
			"%w: document identity differs from requested Session ID",
			ErrInvalidDocument,
		)
	}
	return doc, true, nil
}

func (s *Store) writeDocument(session sessionv3Runtime.Session) error {
	doc := sessionDocument{
		SchemaVersion: sessionv3Runtime.SchemaVersion,
		Kind:          sessionDocumentKindSession,
		SessionID:     session.ID,
		Session:       &session,
	}
	return s.writeRawDocument(session.ID, doc)
}

func (s *Store) writeRawDocument(id sessionv3Runtime.SessionID, doc sessionDocument) error {
	filename, err := stableSessionFilename(id, documentExtension)
	if err != nil {
		return err
	}
	if doc.SessionID != id {
		return fmt.Errorf("%w: document identity differs from filename identity", ErrInvalidDocument)
	}
	if err := validateStoreDocument(doc); err != nil {
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

func (s *Store) writeIndex(session sessionv3Runtime.Session) error {
	active := []sessionv3Runtime.ExecutionID{}
	for _, e := range session.Executions {
		if !e.State.Terminal() {
			active = append(active, e.ID)
		}
	}
	record := sessionIndexRecord{
		SchemaVersion: sessionv3Runtime.SchemaVersion,
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

func (s *Store) deleteIndex(id sessionv3Runtime.SessionID) error {
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

func (s *Store) readIndex(id sessionv3Runtime.SessionID) (sessionIndexRecord, bool, error) {
	filename, err := stableSessionFilename(id, indexExtension)
	if err != nil {
		return sessionIndexRecord{}, false, err
	}
	if err := validateStoreSessionID(id); err != nil {
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

	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		maximumDocumentBytes,
	)
	if err != nil {
		return sessionIndexRecord{}, err
	}
	var record sessionIndexRecord
	if err := jsonutil.DecodeCanonicalObjectExactInto(
		canonical,
		&record,
		maximumDocumentBytes,
	); err != nil {
		return sessionIndexRecord{}, err
	}
	if record.SchemaVersion != sessionv3Runtime.SchemaVersion {
		return sessionIndexRecord{}, fmt.Errorf("%w: unsupported index schema", ErrInvalidDocument)
	}
	if err := validateStoreSessionID(record.Item.ID); err != nil {
		return sessionIndexRecord{}, err
	}
	expected, err := stableSessionFilename(record.Item.ID, indexExtension)
	if err != nil {
		return sessionIndexRecord{}, err
	}
	if expected != filename {
		return sessionIndexRecord{}, fmt.Errorf("%w: index identity mismatch", ErrInvalidDocument)
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
		expected, err := stableSessionFilename(doc.SessionID, documentExtension)
		if err != nil {
			return nil, err
		}
		if expected != filename {
			return nil, fmt.Errorf(
				"%w: document filename identity mismatch",
				ErrInvalidDocument,
			)
		}
		out = append(out, doc)
	}
	return out, nil
}

func cloneSession(v sessionv3Runtime.Session) sessionv3Runtime.Session {
	return jsonutil.CloneJSONInValue(v)
}

func cloneExecution(v sessionv3Runtime.Execution) sessionv3Runtime.Execution {
	return jsonutil.CloneJSONInValue(v)
}

func cloneOutcome(v *sessionv3Runtime.ExecutionOutcome) *sessionv3Runtime.ExecutionOutcome {
	if v == nil {
		return nil
	}
	output := *v
	if v.Error != nil {
		output.Error = &sessionv3Runtime.SanitizedError{
			Code:      v.Error.Code,
			Category:  v.Error.Category,
			Message:   v.Error.Message,
			Retryable: v.Error.Retryable,
			StepPath:  v.Error.StepPath,
			Effect:    v.Error.Effect,
			Details:   v.Error.Details.Clone(),
			Causes:    append([]sessionv3Runtime.SanitizedError(nil), v.Error.Causes...),
		}
	}
	return &output
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

func stableSessionFilename(id sessionv3Runtime.SessionID, ext string) (string, error) {
	if err := validateStoreSessionID(id); err != nil {
		return "", err
	}
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
	seenTokens := map[string]struct{}{}
	seenFiles := map[string]struct{}{}
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
				if _, duplicate := seenFiles[filename]; duplicate {
					continue
				}
				seenFiles[filename] = struct{}{}
				out = append(out, filename)
			}
		}
		if next == "" {
			return out, nil
		}
		if _, duplicate := seenTokens[next]; duplicate {
			return nil, fmt.Errorf("%w: repeated listing cursor", ErrInvalidDocument)
		}
		seenTokens[next] = struct{}{}
		token = next
	}
}

func decodeStoreDocument(data map[string]any) (sessionDocument, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return sessionDocument{}, err
	}
	return decodeStoreDocumentJSON(raw)
}

func decodeStoreDocumentJSON(raw []byte) (sessionDocument, error) {
	if len(raw) > maximumDocumentBytes {
		return sessionDocument{}, fmt.Errorf("%w: document exceeds size limit", ErrInvalidDocument)
	}

	canonical, err := jsonutil.CanonicalizeObject(
		raw,
		maximumDocumentBytes,
	)
	if err != nil {
		return sessionDocument{}, fmt.Errorf("%w: %w", ErrInvalidDocument, err)
	}
	var doc sessionDocument
	if err := jsonutil.DecodeCanonicalObjectExactInto(
		canonical,
		&doc,
		maximumDocumentBytes,
	); err != nil {
		return sessionDocument{}, fmt.Errorf("%w: %w", ErrInvalidDocument, err)
	}
	if err := validateStoreDocument(doc); err != nil {
		return sessionDocument{}, err
	}
	return doc, nil
}

func validateStoreSessionID(id sessionv3Runtime.SessionID) error {
	if err := uuidutil.ValidateUUIDv7(string(id)); err != nil {
		return fmt.Errorf("%w: invalid Session ID: %w", ErrInvalidDocument, err)
	}
	return nil
}

func validateStoreDocument(doc sessionDocument) error {
	if doc.SchemaVersion != sessionv3Runtime.SchemaVersion {
		return fmt.Errorf("%w: unsupported schema version %q", ErrInvalidDocument, doc.SchemaVersion)
	}
	if err := validateStoreSessionID(doc.SessionID); err != nil {
		return err
	}

	switch doc.Kind {
	case sessionDocumentKindSession:
		if doc.Session == nil || doc.DeletedAt != nil {
			return fmt.Errorf("%w: malformed live Session document", ErrInvalidDocument)
		}
		if doc.Session.ID != doc.SessionID {
			return fmt.Errorf("%w: Session document identity mismatch", ErrInvalidDocument)
		}
		if err := doc.Session.Validate(); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidDocument, err)
		}
	case sessionDocumentKindDeleted:
		if doc.Session != nil || doc.DeletedAt == nil || doc.DeletedAt.IsZero() {
			return fmt.Errorf("%w: malformed Session tombstone", ErrInvalidDocument)
		}
	default:
		return fmt.Errorf("%w: unsupported document kind %q", ErrInvalidDocument, doc.Kind)
	}
	return nil
}
