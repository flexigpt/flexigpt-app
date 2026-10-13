package sdkclient

import (
	"bytes"
	"log/slog"
	"strings"
	"sync"

	mcpAuth "github.com/flexigpt/flexigpt-app/internal/agentruntime-go/mcp/auth"
)

const maxSlogLineBytes = 64 << 10

type slogLineWriter struct {
	logger   *slog.Logger
	serverID string
	message  string
	redactor *mcpAuth.SecretRedactor

	mu         sync.Mutex
	buf        []byte
	discarding bool
}

func newSlogLineWriter(
	logger *slog.Logger,
	serverID string,
	message string,
	redactor *mcpAuth.SecretRedactor,
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

	written := len(p)
	for len(p) != 0 {
		if w.discarding {
			index := bytes.IndexByte(p, '\n')
			if index < 0 {
				return written, nil
			}
			p = p[index+1:]
			w.discarding = false
			continue
		}

		index := bytes.IndexByte(p, '\n')
		if index < 0 {
			if len(w.buf)+len(p) > maxSlogLineBytes {
				w.buf = nil
				w.discarding = true
				w.logger.Warn(
					w.message,
					"serverID", w.serverID,
					"line", "[line omitted: exceeds log safety limit]",
				)
				return written, nil
			}
			w.buf = append(w.buf, p...)
			return written, nil
		}

		if len(w.buf)+index > maxSlogLineBytes {
			w.buf = nil
			w.logger.Warn(
				w.message,
				"serverID", w.serverID,
				"line", "[line omitted: exceeds log safety limit]",
			)
			p = p[index+1:]
			continue
		}

		w.buf = append(w.buf, p[:index]...)
		line := strings.TrimRight(string(w.buf), "\r")
		w.buf = nil
		p = p[index+1:]

		if strings.TrimSpace(line) != "" {
			if w.redactor != nil {
				line = w.redactor.Redact(line)
			}
			w.logger.Info(w.message, "serverID", w.serverID, "line", line)
		}
	}

	return written, nil
}
