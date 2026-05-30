package sdkclient

import (
	"context"
	"log/slog"
	"maps"
	"strings"
	"sync"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/mcp/auth"
)

func TestSlogLineWriterRedactsStdioSecretAcrossWrites(t *testing.T) {
	handler := &captureSlogHandler{}
	logger := slog.New(handler)

	redactor := auth.NewSecretRedactor(auth.ResolvedTransportAuth{
		SensitiveValues: []string{"top-secret"},
	})

	writer := newSlogLineWriter(logger, "server", "mcp stdio stderr", redactor)

	if _, err := writer.Write([]byte("prefix top-")); err != nil {
		t.Fatalf("write #1: %v", err)
	}
	if _, err := writer.Write([]byte("secret suffix\n")); err != nil {
		t.Fatalf("write #2: %v", err)
	}

	records := handler.getRecords()
	if len(records) != 1 {
		t.Fatalf("records = %d, want 1", len(records))
	}

	line := records[0]["line"]
	if strings.Contains(line, "top-secret") {
		t.Fatalf("log line leaked secret: %q", line)
	}
	if line != "prefix [REDACTED] suffix" {
		t.Fatalf("line = %q", line)
	}
}

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
