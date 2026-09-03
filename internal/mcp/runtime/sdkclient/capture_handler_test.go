package sdkclient

import (
	"context"
	"log/slog"
	"maps"
	"sync"
)

type captureSlogHandler struct {
	mu      sync.Mutex
	records []map[string]string
}

func (h *captureSlogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (h *captureSlogHandler) Handle(ctx context.Context, record slog.Record) error {
	out := map[string]string{
		"message": record.Message,
	}
	record.Attrs(func(attr slog.Attr) bool {
		out[attr.Key] = attr.Value.String()
		return true
	})

	h.mu.Lock()
	h.records = append(h.records, out)
	h.mu.Unlock()

	return nil
}

func (h *captureSlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *captureSlogHandler) WithGroup(name string) slog.Handler {
	return h
}

func (h *captureSlogHandler) getRecords() []map[string]string {
	h.mu.Lock()
	defer h.mu.Unlock()

	out := make([]map[string]string, len(h.records))
	for i, record := range h.records {
		cp := make(map[string]string, len(record))
		maps.Copy(cp, record)
		out[i] = cp
	}
	return out
}
