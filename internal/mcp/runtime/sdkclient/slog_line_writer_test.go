package sdkclient

import (
	"log/slog"
	"strings"
	"testing"

	mcpAuth "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/auth"
	mcpConnection "github.com/flexigpt/flexigpt-app/internal/mcp/runtime/connection"
)

func TestSlogLineWriterRedactsStdioSecretAcrossWrites(t *testing.T) {
	handler := &captureSlogHandler{}
	logger := slog.New(handler)

	redactor := mcpAuth.NewSecretRedactor(mcpConnection.PreparedConnection{
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
