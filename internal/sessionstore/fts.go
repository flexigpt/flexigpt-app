package sessionstore

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	sessionRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/session"
	"github.com/flexigpt/flexigpt-app/internal/backgroundutil"
	"github.com/flexigpt/flexigpt-app/internal/llmtoolsutil"
	"github.com/flexigpt/llmtools-go/fstool"
	"github.com/flexigpt/mapstore-go"
	"github.com/flexigpt/mapstore-go/ftsengine"
)

type sessionFTSColumn string

const (
	sessionFTSDatabaseFile = "sessions-v2.fts.sqlite"
	sessionFTSTable        = "sessions_v2"

	sessionFTSTitleColumn     sessionFTSColumn = "title"
	sessionFTSSystemColumn    sessionFTSColumn = "system"
	sessionFTSUserColumn      sessionFTSColumn = "user"
	sessionFTSAssistantColumn sessionFTSColumn = "assistant"
	sessionFTSModifiedColumn  sessionFTSColumn = "mtime"
)

type sessionFTSValues struct {
	Title     string
	System    string
	User      string
	Assistant string
	Modified  string
}

func (value sessionFTSValues) Map() map[string]string {
	return map[string]string{
		string(sessionFTSTitleColumn):     value.Title,
		string(sessionFTSSystemColumn):    value.System,
		string(sessionFTSUserColumn):      value.User,
		string(sessionFTSAssistantColumn): value.Assistant,
		string(sessionFTSModifiedColumn):  value.Modified,
	}
}

func (s *Store) ftsListener() mapstore.FileListener {
	return func(event mapstore.FileEvent) {
		backgroundutil.RunSafely(
			"SessionStore FTS document notification",
			func() {
				if !isSessionDocumentFile(event.File) {
					return
				}

				s.ftsMu.RLock()
				defer s.ftsMu.RUnlock()

				switch event.Op {
				case mapstore.OpSetFile, mapstore.OpResetFile:
					document, err := decodeStoredSessionData(event.Data)
					if err != nil {
						slog.Error("sessionStore FTS document adaptation failed", "error", err)
						return
					}
					if document.Kind == documentKindDeleted {
						if err := s.fts.Delete(context.Background(), event.File); err != nil {
							slog.Error("sessionStore FTS deletion failed", "error", err)
						}
						return
					}
					comparison, err := sessionFileMTime(context.Background(), event.File)
					if err != nil {
						slog.Error("sessionStore FTS file observation failed", "error", err)
						return
					}
					values := sessionFTSValuesForDocument(document, comparison)
					if err := s.fts.Upsert(context.Background(), event.File, values.Map()); err != nil {
						slog.Error("sessionStore FTS upsert failed", "error", err)
					}

				case mapstore.OpDeleteFile:
					if err := s.fts.Delete(context.Background(), event.File); err != nil {
						slog.Error("sessionStore FTS file removal failed", "error", err)
					}

				case mapstore.OpSetKey, mapstore.OpDeleteKey:
				}
			},
			nil,
		)
	}
}

func (s *Store) syncDocumentToFTS(
	ctx context.Context,
	_ string,
	fullPath string,
	getPreviousComparison ftsengine.GetPrevCmp,
) (ftsengine.SyncDecision, error) {
	skip := ftsengine.SyncDecision{
		ID:   fullPath,
		Vals: map[string]string{},
		Skip: true,
	}
	if !isSessionDocumentFile(fullPath) {
		return skip, nil
	}

	comparison, err := sessionFileMTime(ctx, fullPath)
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
	document, err := decodeStoredSessionJSON(raw)
	if err != nil {
		return skip, err
	}
	if document.Kind == documentKindDeleted {
		if err := s.fts.Delete(ctx, fullPath); err != nil {
			return skip, err
		}
		return skip, nil
	}

	values := sessionFTSValuesForDocument(document, comparison)
	return ftsengine.SyncDecision{
		ID:     fullPath,
		CmpOut: comparison,
		Vals:   values.Map(),
	}, nil
}

func sessionFTSValuesForDocument(
	document storedSession,
	comparison string,
) sessionFTSValues {
	var system, user, assistant bytes.Buffer
	for _, message := range document.Session.Transcript {
		if strings.TrimSpace(message.Text) == "" {
			continue
		}
		switch message.Role {
		case sessionRuntime.MessageRoleSystem, sessionRuntime.MessageRoleDeveloper:
			system.WriteString(message.Text)
			system.WriteByte('\n')
		case sessionRuntime.MessageRoleUser:
			user.WriteString(message.Text)
			user.WriteByte('\n')
		case sessionRuntime.MessageRoleAssistant:
			assistant.WriteString(message.Text)
			assistant.WriteByte('\n')
		default:
		}
	}
	return sessionFTSValues{
		Title:     document.Session.Title,
		System:    system.String(),
		User:      user.String(),
		Assistant: assistant.String(),
		Modified:  comparison,
	}
}

func isSessionDocumentFile(path string) bool {
	return strings.HasSuffix(path, "."+SessionFileExtension.String())
}

func sessionFileMTime(ctx context.Context, path string) (string, error) {
	value, err := llmtoolsutil.StatPath(ctx, fstool.StatPathArgs{Path: path})
	if err != nil {
		return "", err
	}
	if value == nil || !value.Exists || value.IsDir || value.ModTime == nil {
		return "", fmt.Errorf("%w: Session document is unavailable", ErrInvalidDocument)
	}
	return value.ModTime.UTC().Format(time.RFC3339Nano), nil
}
