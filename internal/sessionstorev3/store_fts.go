package sessionstorev3

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	sessionv3Runtime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/sessionv3"
	"github.com/flexigpt/flexigpt-app/internal/backgroundutil"
	"github.com/flexigpt/flexigpt-app/internal/llmtoolsutil"
	"github.com/flexigpt/llmtools-go/fstool"
	"github.com/flexigpt/mapstore-go"
	"github.com/flexigpt/mapstore-go/ftsengine"
)

func sessionFTSValues(session sessionv3Runtime.Session, mtime string) map[string]string {
	var user, assistant, action strings.Builder
	for _, a := range session.Timeline {
		switch a.Kind {
		case sessionv3Runtime.ActivityKindMessage:
			if a.Message == nil {
				continue
			}
			if a.Message.Role == sessionv3Runtime.MessageRoleUser {
				user.WriteString(a.Message.Text)
				user.WriteString("\n")
			} else {
				assistant.WriteString(a.Message.Text)
				assistant.WriteString("\n")
			}
		case sessionv3Runtime.ActivityKindAction:
			if a.Action != nil {
				action.WriteString(a.Action.Title)
				action.WriteString("\n")
				action.WriteString(a.Action.Summary)
				action.WriteString("\n")
			}
		case sessionv3Runtime.ActivityKindWorkflow:
			if a.Workflow != nil {
				action.WriteString(a.Workflow.Summary)
				action.WriteString("\n")
			}
		case sessionv3Runtime.ActivityKindWait:
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
			s.ftsMu.Lock()
			defer s.ftsMu.Unlock()
			if s.fts == nil {
				return
			}
			switch event.Op {
			case mapstore.OpSetFile, mapstore.OpResetFile:
				doc, err := decodeStoreDocument(event.Data)
				if err != nil {
					slog.Error("sessionstorev3 FTS adaptation failed", "error", err)
					return
				}
				if doc.Kind == sessionDocumentKindDeleted {
					if err := s.fts.Delete(context.Background(), event.File); err != nil {
						slog.Error("sessionstorev3 FTS deletion failed", "error", err)
					}
					return
				}
				if doc.Session == nil {
					return
				}
				mtime, err := storeDocumentMTime(context.Background(), event.File)
				if err != nil {
					slog.Error("sessionstorev3 FTS file observation failed", "error", err)
					return
				}
				if err := s.fts.Upsert(
					context.Background(),
					event.File,
					sessionFTSValues(*doc.Session, mtime),
				); err != nil {
					slog.Error("sessionstorev3 FTS upsert failed", "error", err)
				}
			case mapstore.OpDeleteFile:
				if err := s.fts.Delete(context.Background(), event.File); err != nil {
					slog.Error("sessionstorev3 FTS file removal failed", "error", err)
				}
			case mapstore.OpSetKey, mapstore.OpDeleteKey:
			}
		}, nil)
	}
}

func (s *Store) syncDocumentToFTS(
	ctx context.Context,
	_, fullPath string,
	getPreviousComparison ftsengine.GetPrevCmp,
) (ftsengine.SyncDecision, error) {
	skip := ftsengine.SyncDecision{ID: fullPath, Vals: map[string]string{}, Skip: true}
	if !isSessionDocumentFile(fullPath) {
		return skip, nil
	}
	comparison, err := storeDocumentMTime(ctx, fullPath)
	if err != nil {
		return skip, err
	}
	if comparison == getPreviousComparison(fullPath) {
		return ftsengine.SyncDecision{
			ID:        fullPath,
			Vals:      map[string]string{},
			Unchanged: true,
		}, nil
	}

	raw, err := llmtoolsutil.ReadPortableTextFile(
		ctx,
		fullPath,
		maximumDocumentBytes,
	)
	if err != nil {
		return skip, err
	}
	doc, err := decodeStoreDocumentJSON(raw)
	if err != nil {
		return skip, err
	}
	if doc.Kind == sessionDocumentKindDeleted {
		if err := s.fts.Delete(ctx, fullPath); err != nil {
			return skip, err
		}
		return skip, nil
	}
	if doc.Session == nil {
		return skip, fmt.Errorf("%w: live FTS document has no Session", ErrInvalidDocument)
	}
	return ftsengine.SyncDecision{
		ID:     fullPath,
		CmpOut: comparison,
		Vals:   sessionFTSValues(*doc.Session, comparison),
	}, nil
}

func storeDocumentMTime(ctx context.Context, path string) (string, error) {
	value, err := llmtoolsutil.StatPath(ctx, fstool.StatPathArgs{Path: path})
	if err != nil {
		return "", err
	}
	if value == nil || !value.Exists || value.IsDir || value.ModTime == nil {
		return "", fmt.Errorf("%w: Session document is unavailable", ErrInvalidDocument)
	}
	return value.ModTime.UTC().Format(time.RFC3339Nano), nil
}
