package sdkclient

import (
	"log/slog"
	"strings"
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/mcp/auth"
)

type slogLineWriter struct {
	logger   *slog.Logger
	serverID string
	message  string
	redactor *auth.SecretRedactor

	mu  sync.Mutex
	buf []byte
}

func newSlogLineWriter(
	logger *slog.Logger,
	serverID string,
	message string,
	redactor *auth.SecretRedactor,
) *slogLineWriter {
	if logger == nil {
		logger = slog.Default()
	}
	if message == "" {
		message = "mcp process log"
	}
	return &slogLineWriter{
		logger:   logger,
		serverID: serverID,
		message:  message,
		redactor: redactor,
	}
}

func (w *slogLineWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	w.buf = append(w.buf, p...)

	for {
		i := -1
		for idx, b := range w.buf {
			if b == '\n' {
				i = idx
				break
			}
		}
		if i < 0 {
			break
		}

		line := string(w.buf[:i])
		line = strings.TrimRight(line, "\r")
		w.buf = w.buf[i+1:]

		if strings.TrimSpace(line) == "" {
			continue
		}
		if w.redactor != nil {
			line = w.redactor.Redact(line)
		}
		w.logger.Info(w.message, "serverID", w.serverID, "line", line)
	}

	return len(p), nil
}
