package store

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/flexigpt/flexigpt-app/internal/conversation/spec"
	"github.com/ppipada/mapstore-go"
	"github.com/ppipada/mapstore-go/ftsengine"
)

func NewFTSListner(e *ftsengine.Engine) mapstore.FileListener {
	return func(ev mapstore.FileEvent) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("fts listener panic",
					"op", ev.Op, "file", ev.File, "recover", r,
					"stack", string(debug.Stack()))
			}
		}()
		// Reject files that obviously do not belong to us.
		if ev.File == "" || !strings.HasSuffix(ev.File, ".json") {
			return
		}

		ctx := context.Background()
		switch ev.Op {
		case mapstore.OpSetFile, mapstore.OpResetFile:
			vals := extractFTS(ev.File, ev.Data)
			if len(vals) == 0 {
				slog.Warn("fts listener: nothing to index", "file", ev.File)
				return
			}
			if err := e.Upsert(ctx, ev.File, vals); err != nil {
				slog.Error("fts upsert failed", "file", ev.File, "err", err)
			}
		case mapstore.OpDeleteFile:
			if err := e.Delete(ctx, ev.File); err != nil {
				slog.Error("fts delete failed", "file", ev.File, "err", err)
			}
		case mapstore.OpSetKey, mapstore.OpDeleteKey:
			// Do nothing as we dont do key operations.
		}
	}
}

func processFTSDataForFile(
	ctx context.Context,
	baseDir, fullPath string,
	getPrevCmpVal ftsengine.GetPrevCmp,
) (
	ftsengine.SyncDecision, error,
) {
	skipSyncDecision := ftsengine.SyncDecision{
		ID:        fullPath,
		CmpOut:    "",
		Vals:      map[string]string{},
		Unchanged: false,
		Skip:      true,
	}
	if !strings.HasSuffix(fullPath, ".json") {
		return skipSyncDecision, nil
	}
	cmp := fileMTime(fullPath)
	prevCmp := getPrevCmpVal(fullPath)
	if cmp == prevCmp {
		return ftsengine.SyncDecision{
			ID:        fullPath,
			CmpOut:    "",
			Vals:      map[string]string{},
			Unchanged: true,
			Skip:      false,
		}, nil
	}

	raw, err := os.ReadFile(fullPath)
	if err != nil {
		slog.Error("conversation sync fts", "file", fullPath, "read error", err)
		return skipSyncDecision, nil
	}

	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		slog.Error("conversation sync fts", "file", fullPath, "json error", err)
		return skipSyncDecision, nil
	}

	if schemaVersion, _ := stringField(m, "schemaVersion"); schemaVersion == "" {
		return skipSyncDecision, nil
	}

	pt := spec.Conversation{}
	if err := json.Unmarshal(raw, &pt); err != nil {
		slog.Error("conversation sync fts", "file", fullPath, "non conversation file error", err)
		return skipSyncDecision, nil
	}

	vals := extractFTS(fullPath, m)
	if len(vals) == 0 {
		// Legacy/dead or non-indexable; don't touch the index for this file.
		return skipSyncDecision, nil
	}

	return ftsengine.SyncDecision{
		ID:        fullPath,
		CmpOut:    cmp,
		Vals:      vals,
		Unchanged: false,
		Skip:      false,
	}, nil
}

func extractFTS(fullPath string, m map[string]any) map[string]string {
	schemaVersion, _ := stringField(m, "schemaVersion")
	if schemaVersion == "" {
		return map[string]string{}
	}
	var (
		title, _ = stringField(m, "title")
		system   bytes.Buffer
		user     bytes.Buffer
		assist   bytes.Buffer
	)

	appendByRole := func(role, txt string) {
		if txt == "" {
			return
		}
		switch role {
		case "system":
			system.WriteString(txt + "\n")
		case "user":
			user.WriteString(txt + "\n")
		case "assistant":
			assist.WriteString(txt + "\n")
		default:
		}
	}

	msgs, _ := m["messages"].([]any)
	for _, raw := range msgs {
		msg, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		role, _ := msg["role"].(string)

		// 2) New shape: nested "messages" -> "contents".
		var ioList []any
		if inList, ok := msg["inputs"].([]any); ok {
			ioList = inList
		} else if outList, ok := msg["outputs"].([]any); ok {
			ioList = outList
		}
		if len(ioList) == 0 {
			continue
		}
		for _, ioRaw := range ioList {
			ioMap, ok := ioRaw.(map[string]any)
			if !ok {
				continue
			}
			var msgMap map[string]any
			if inMsgMap, ok := ioMap["inputMessage"].(map[string]any); ok {
				msgMap = inMsgMap
			} else if outMsgMap, ok := ioMap["outputMessage"].(map[string]any); ok {
				msgMap = outMsgMap
			}
			if len(msgMap) == 0 {
				continue
			}
			contents, _ := msgMap["contents"].([]any)
			for _, cRaw := range contents {
				item, ok := cRaw.(map[string]any)
				if !ok {
					continue
				}
				kind, _ := item["kind"].(string)
				switch kind {
				case "text":
					if txtItem, ok := item["textItem"].(map[string]any); ok {
						if txt, ok := txtItem["text"].(string); ok {
							appendByRole(role, txt)
						}
					}
				default:
				}
			}
		}
	}

	return map[string]string{
		"title":     title,
		"system":    system.String(),
		"user":      user.String(),
		"assistant": assist.String(),
		"mtime":     fileMTime(fullPath),
	}
}

func stringField(m map[string]any, key string) (string, bool) {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s, true
		}
	}
	// Also accept "Title" from StructToMap.
	if v, ok := m[strings.ToUpper(key[:1])+key[1:]]; ok {
		if s, ok := v.(string); ok {
			return s, true
		}
	}
	return "", false
}

func fileMTime(path string) string {
	st, err := os.Stat(path)
	if err != nil {
		return ""
	}
	return st.ModTime().UTC().Format(time.RFC3339Nano)
}
