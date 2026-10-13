package sessionstore

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	sessionRuntime "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/session"
	"github.com/flexigpt/flexigpt-app/internal/backgroundutil"
	"github.com/flexigpt/mapstore-go"
	"github.com/flexigpt/mapstore-go/ftsengine"
)

type sessionFTSColumn string

const (
	sessionFTSDatabaseFile = "sessions.fts.sqlite"
	sessionFTSTable        = "sessions"

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

func (v sessionFTSValues) Map() map[string]string {
	return map[string]string{
		string(sessionFTSTitleColumn):     v.Title,
		string(sessionFTSSystemColumn):    v.System,
		string(sessionFTSUserColumn):      v.User,
		string(sessionFTSAssistantColumn): v.Assistant,
		string(sessionFTSModifiedColumn):  v.Modified,
	}
}

func emptySessionFTSValues() map[string]string {
	return map[string]string{}
}

func newFTSListener(
	engine *ftsengine.Engine,
) mapstore.FileListener {
	return func(event mapstore.FileEvent) {
		backgroundutil.RunSafely(
			"session store FTS listener",
			func() {
				handleFTSEvent(engine, event)
			},
			nil,
		)
	}
}

func handleFTSEvent(engine *ftsengine.Engine, event mapstore.FileEvent) {
	if !isSessionDocumentFile(event.File) {
		return
	}

	switch event.Op {
	case mapstore.OpSetFile, mapstore.OpResetFile:
		document, err := decodeStoredSessionData(event.Data)
		if err != nil {
			return
		}

		values := sessionFTSValuesForDocument(
			event.File,
			document,
		)
		if err := engine.Upsert(
			context.Background(),
			event.File,
			values.Map(),
		); err != nil {
			slog.Error(
				"session store FTS upsert failed",
				"file",
				event.File,
				"error",
				err,
			)
		}

	case mapstore.OpDeleteFile:
		if err := engine.Delete(
			context.Background(),
			event.File,
		); err != nil {
			slog.Error(
				"session store FTS delete failed",
				"file",
				event.File,
				"error",
				err,
			)
		}

	case mapstore.OpSetKey, mapstore.OpDeleteKey:
	}
}

func syncSessionDocumentToFTS(
	_ context.Context,
	_ string,
	fullPath string,
	getPreviousComparison ftsengine.GetPrevCmp,
) (ftsengine.SyncDecision, error) {
	skip := ftsengine.SyncDecision{
		ID:        fullPath,
		CmpOut:    "",
		Vals:      emptySessionFTSValues(),
		Unchanged: false,
		Skip:      true,
	}

	if !isSessionDocumentFile(fullPath) {
		return skip, nil
	}

	comparison := sessionFileMTime(fullPath)
	if comparison == getPreviousComparison(fullPath) {
		return ftsengine.SyncDecision{
			ID:        fullPath,
			CmpOut:    "",
			Vals:      emptySessionFTSValues(),
			Unchanged: true,
			Skip:      false,
		}, nil
	}

	raw, err := os.ReadFile(fullPath)
	if err != nil {
		slog.Error(
			"session store FTS sync read failed",
			"file",
			fullPath,
			"error",
			err,
		)
		return skip, nil
	}

	document, err := decodeStoredSessionJSON(raw)
	if err != nil {
		//nolint:nilerr // Explicit.
		return skip, nil
	}

	values := sessionFTSValuesForDocument(fullPath, document)

	return ftsengine.SyncDecision{
		ID:        fullPath,
		CmpOut:    comparison,
		Vals:      values.Map(),
		Unchanged: false,
		Skip:      false,
	}, nil
}

func sessionFTSValuesForDocument(
	fullPath string,
	document storedSession,
) sessionFTSValues {
	var system bytes.Buffer
	var user bytes.Buffer
	var assistant bytes.Buffer

	for _, message := range document.Session.Transcript {
		if strings.TrimSpace(message.Text) == "" {
			continue
		}

		switch message.Role {
		case sessionRuntime.MessageRoleSystem,
			sessionRuntime.MessageRoleDeveloper:
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
		Modified:  sessionFileMTime(fullPath),
	}
}

func isSessionDocumentFile(
	path string,
) bool {
	return strings.HasSuffix(
		path,
		"."+SessionFileExtension.String(),
	)
}

func sessionFileMTime(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return ""
	}

	return info.ModTime().UTC().Format(time.RFC3339Nano)
}
