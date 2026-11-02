package sessionstorev3

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/backgroundutil"
	"github.com/flexigpt/mapstore-go"
	"github.com/flexigpt/mapstore-go/ftsengine"
)

func sessionFTSValues(session sessionv3.Session, mtime string) map[string]string {
	var user, assistant, action strings.Builder
	for _, a := range session.Timeline {
		switch a.Kind {
		case sessionv3.ActivityKindMessage:
			if a.Message == nil {
				continue
			}
			if a.Message.Role == sessionv3.MessageRoleUser {
				user.WriteString(a.Message.Text)
				user.WriteString("\n")
			} else {
				assistant.WriteString(a.Message.Text)
				assistant.WriteString("\n")
			}
		case sessionv3.ActivityKindAction:
			if a.Action != nil {
				action.WriteString(a.Action.Title)
				action.WriteString("\n")
				action.WriteString(a.Action.Summary)
				action.WriteString("\n")
			}
		case sessionv3.ActivityKindWorkflow:
			if a.Workflow != nil {
				action.WriteString(a.Workflow.Summary)
				action.WriteString("\n")
			}
		case sessionv3.ActivityKindWait:
			if a.Wait != nil {
				action.WriteString(a.Wait.Summary)
				action.WriteString("\n")
			}
		}
	}
	return map[string]string{
		"title":     session.Title,
		"user":      user.String(),
		"assistant": assistant.String(),
		"action":    action.String(),
		"mtime":     mtime,
	}
}

func isSessionDocumentFile(path string) bool {
	return strings.HasSuffix(path, documentExtension)
}

func (s *Store) ftsListener() mapstore.FileListener {
	return func(event mapstore.FileEvent) {
		backgroundutil.RunSafely("SessionStoreV3 FTS document notification", func() {
			if !isSessionDocumentFile(event.File) {
				return
			}
			s.ftsMu.RLock()
			defer s.ftsMu.RUnlock()
			switch event.Op {
			case mapstore.OpSetFile, mapstore.OpResetFile:
				doc, err := decodeStoreDocument(event.Data)
				if err != nil {
					slog.Error("sessionstorev3 FTS adaptation failed", "error", err)
					return
				}
				if doc.Kind == sessionDocumentKindDeleted {
					_ = s.fts.Delete(context.Background(), event.File)
					return
				}
				if doc.Session == nil {
					return
				}
				mtime := doc.Session.UpdatedAt.UTC().Format(time.RFC3339Nano)
				_ = s.fts.Upsert(context.Background(), event.File, sessionFTSValues(*doc.Session, mtime))
			case mapstore.OpDeleteFile:
				_ = s.fts.Delete(context.Background(), event.File)
			case mapstore.OpSetKey, mapstore.OpDeleteKey:
			}
		}, nil)
	}
}

func (s *Store) syncDocumentToFTS(
	_ context.Context,
	_, fullPath string,
	getPreviousComparison ftsengine.GetPrevCmp,
) (ftsengine.SyncDecision, error) {
	skip := ftsengine.SyncDecision{ID: fullPath, Vals: map[string]string{}, Skip: true}
	if !isSessionDocumentFile(fullPath) {
		return skip, nil
	}
	stat, err := s.documents.GetFileData(mapstore.FileKey{FileName: fullPath}, false)
	if err != nil {
		return skip, err
	}
	_ = stat
	return skip, nil
}
